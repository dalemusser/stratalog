package logapi

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/dalemusser/stratalog/internal/app/system/ledger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// gameRegex validates game names (alphanumeric, underscores, hyphens only)
var gameRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// logdataCollection is the unified collection name for all log data
const logdataCollection = "logdata"

// LogBroadcaster is a function that broadcasts log events to SSE subscribers.
type LogBroadcaster func(game, playerID, eventType string, serverTimestamp time.Time, data map[string]interface{})

// Handler handles log API requests.
type Handler struct {
	db           *mongo.Database
	logger       *zap.Logger
	maxBatchSize int
	broadcaster  LogBroadcaster
}

// NewHandler creates a new logapi handler.
func NewHandler(db *mongo.Database, logger *zap.Logger, maxBatchSize int) *Handler {
	if maxBatchSize <= 0 {
		maxBatchSize = 100
	}
	return &Handler{
		db:           db,
		logger:       logger,
		maxBatchSize: maxBatchSize,
	}
}

// SetBroadcaster sets the function to broadcast log events to SSE subscribers.
func (h *Handler) SetBroadcaster(b LogBroadcaster) {
	h.broadcaster = b
}


// SubmitHandler handles POST /api/log/submit and POST /logs (legacy) requests.
// It accepts both single log entries and batch submissions.
//
// Single entry format:
//
//	{
//	    "game": "mhs",
//	    "playerId": "player001",
//	    "eventType": "level_complete",
//	    "level": 5,
//	    "score": 1000
//	}
//
// Batch entry format:
//
//	{
//	    "game": "mhs",
//	    "entries": [
//	        {"playerId": "player001", "eventType": "level_start", "level": 5},
//	        {"playerId": "player001", "eventType": "level_complete", "level": 5, "score": 1000}
//	    ]
//	}
func (h *Handler) SubmitHandler(w http.ResponseWriter, r *http.Request) {
	// Limit body size to 1MB for backward compatibility with strata_log
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	// Parse the raw JSON to detect format
	var raw map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		// Check if it's a body too large error
		if err.Error() == "http: request body too large" {
			writeJSONError(w, r, "request body too large", "BODY_TOO_LARGE", http.StatusRequestEntityTooLarge)
			return
		}
		writeJSONError(w, r, "Invalid JSON payload", "INVALID_JSON", http.StatusBadRequest)
		return
	}

	// Check if this is a batch request (has "entries" array)
	if entries, ok := raw["entries"].([]interface{}); ok {
		h.handleBatchSubmit(w, r, raw, entries)
		return
	}

	// Single entry submission
	h.handleSingleSubmit(w, r, raw)
}

// handleSingleSubmit processes a single log entry submission.
// Stores documents flat in the unified logdata collection for backward compatibility
// with the original strata_log API.
func (h *Handler) handleSingleSubmit(w http.ResponseWriter, r *http.Request, raw map[string]interface{}) {
	// Extract and validate required game field
	game, ok := raw["game"].(string)
	if !ok || game == "" {
		writeJSONError(w, r, "missing or invalid 'game' field", "MISSING_FIELD", http.StatusBadRequest)
		return
	}
	if !gameRegex.MatchString(game) {
		writeJSONError(w, r, "invalid 'game' value", "INVALID_GAME", http.StatusBadRequest)
		return
	}

	// Add server timestamp - use "serverTimestamp" for backward compatibility with strata_log
	now := time.Now().UTC()
	raw["serverTimestamp"] = now

	// Insert into unified logdata collection
	coll := h.db.Collection(logdataCollection)
	_, err := coll.InsertOne(r.Context(), raw)
	if err != nil {
		playerID, _ := raw["playerId"].(string)
		h.logger.Error("failed to insert log entry",
			zap.String("game", game),
			zap.String("playerId", playerID),
			zap.Error(err),
		)
		writeJSONError(w, r, "Failed to save log entry", "INSERT_FAILED", http.StatusInternalServerError)
		return
	}

	playerID, _ := raw["playerId"].(string)
	eventType, _ := raw["eventType"].(string)
	h.logger.Debug("log entry saved",
		zap.String("game", game),
		zap.String("playerId", playerID),
		zap.String("eventType", eventType),
	)

	// Broadcast to SSE subscribers
	if h.broadcaster != nil {
		// Extract data fields (everything except known fields)
		data := make(map[string]interface{})
		for k, v := range raw {
			if k != "game" && k != "playerId" && k != "eventType" && k != "timestamp" && k != "serverTimestamp" && k != "_id" {
				data[k] = v
			}
		}
		h.broadcaster(game, playerID, eventType, now, data)
	}

	// Ensure indexes exist (async)
	go h.ensureIndexes()

	// Return backward-compatible response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(LogResponse{
		Status:     "success",
		ReceivedAt: now.Format(time.RFC3339),
	})
}

// handleBatchSubmit processes a batch log entry submission.
// Stores documents flat in the unified logdata collection for backward compatibility.
func (h *Handler) handleBatchSubmit(w http.ResponseWriter, r *http.Request, raw map[string]interface{}, entries []interface{}) {
	// Extract and validate required game field
	game, ok := raw["game"].(string)
	if !ok || game == "" {
		writeJSONError(w, r, "missing or invalid 'game' field", "MISSING_FIELD", http.StatusBadRequest)
		return
	}
	if !gameRegex.MatchString(game) {
		writeJSONError(w, r, "invalid 'game' value", "INVALID_GAME", http.StatusBadRequest)
		return
	}

	if len(entries) == 0 {
		writeJSONError(w, r, "Entries array is empty", "EMPTY_ENTRIES", http.StatusBadRequest)
		return
	}

	if len(entries) > h.maxBatchSize {
		writeJSONError(w, r, "Batch size exceeds maximum of "+strconv.Itoa(h.maxBatchSize), "BATCH_TOO_LARGE", http.StatusBadRequest)
		return
	}

	// Convert entries to flat documents with game and serverTimestamp added
	now := time.Now().UTC()
	docs := make([]interface{}, 0, len(entries))

	for i, e := range entries {
		entryMap, ok := e.(map[string]interface{})
		if !ok {
			writeJSONError(w, r, "Invalid entry at index "+strconv.Itoa(i), "INVALID_ENTRY", http.StatusBadRequest)
			return
		}

		// Add game and serverTimestamp to each entry (stored flat)
		entryMap["game"] = game
		entryMap["serverTimestamp"] = now
		docs = append(docs, entryMap)
	}

	// Insert all entries into unified logdata collection
	coll := h.db.Collection(logdataCollection)
	_, err := coll.InsertMany(r.Context(), docs)
	if err != nil {
		h.logger.Error("failed to insert batch log entries",
			zap.String("game", game),
			zap.Int("count", len(docs)),
			zap.Error(err),
		)
		writeJSONError(w, r, "Failed to save log entries", "INSERT_FAILED", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("batch log entries saved",
		zap.String("game", game),
		zap.Int("count", len(docs)),
	)

	// Broadcast each entry to SSE subscribers
	if h.broadcaster != nil {
		for _, doc := range docs {
			if entryMap, ok := doc.(map[string]interface{}); ok {
				playerID, _ := entryMap["playerId"].(string)
				eventType, _ := entryMap["eventType"].(string)
				// Extract data fields (everything except known fields)
				data := make(map[string]interface{})
				for k, v := range entryMap {
					if k != "game" && k != "playerId" && k != "eventType" && k != "timestamp" && k != "serverTimestamp" && k != "_id" {
						data[k] = v
					}
				}
				h.broadcaster(game, playerID, eventType, now, data)
			}
		}
	}

	// Ensure indexes exist (async)
	go h.ensureIndexes()

	// Return backward-compatible response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(LogResponse{
		Status:     "success",
		ReceivedAt: now.Format(time.RFC3339),
	})
}

// ListHandler handles GET /logs and GET /api/v1/logs requests.
// Query parameters:
//   - game (required): Filter by game name
//   - playerId: Filter by player ID
//   - eventType: Filter by event type
//   - start_time: Filter entries after this time (RFC3339)
//   - end_time: Filter entries before this time (RFC3339)
//   - limit: Max entries to return (default 100, use 0 for all)
//   - offset: Skip this many entries (for pagination)
func (h *Handler) ListHandler(w http.ResponseWriter, r *http.Request) {
	game := r.URL.Query().Get("game")
	if game == "" {
		writeJSONError(w, r, "Missing required parameter: game", "MISSING_PARAM", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	params := LogQueryParams{
		Game:      game,
		PlayerID:  r.URL.Query().Get("playerId"),
		EventType: r.URL.Query().Get("eventType"),
	}

	// Parse time parameters
	if st := r.URL.Query().Get("start_time"); st != "" {
		if t, err := time.Parse(time.RFC3339, st); err == nil {
			params.StartTime = &t
		}
	}
	if et := r.URL.Query().Get("end_time"); et != "" {
		if t, err := time.Parse(time.RFC3339, et); err == nil {
			params.EndTime = &t
		}
	}

	// Parse pagination (limit=0 means all records)
	params.Limit = 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n >= 0 {
			params.Limit = n
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			params.Offset = n
		}
	}

	// Build filter
	filter := bson.M{"game": game}
	if params.PlayerID != "" {
		filter["playerId"] = params.PlayerID
	}
	if params.EventType != "" {
		filter["eventType"] = params.EventType
	}
	if params.StartTime != nil || params.EndTime != nil {
		timeFilter := bson.M{}
		if params.StartTime != nil {
			timeFilter["$gte"] = *params.StartTime
		}
		if params.EndTime != nil {
			timeFilter["$lte"] = *params.EndTime
		}
		filter["serverTimestamp"] = timeFilter
	}

	// Query the unified logdata collection
	coll := h.db.Collection(logdataCollection)

	// Get total count
	total, err := coll.CountDocuments(r.Context(), filter)
	if err != nil {
		h.logger.Error("failed to count log entries",
			zap.String("game", game),
			zap.Error(err),
		)
		writeJSONError(w, r, "Failed to query logs", "QUERY_FAILED", http.StatusInternalServerError)
		return
	}

	// Query entries
	opts := options.Find().
		SetSort(bson.D{{Key: "serverTimestamp", Value: -1}}).
		SetSkip(int64(params.Offset))
	if params.Limit > 0 {
		opts.SetLimit(int64(params.Limit))
	}

	cur, err := coll.Find(r.Context(), filter, opts)
	if err != nil {
		h.logger.Error("failed to query log entries",
			zap.String("game", game),
			zap.Error(err),
		)
		writeJSONError(w, r, "Failed to query logs", "QUERY_FAILED", http.StatusInternalServerError)
		return
	}
	defer cur.Close(r.Context())

	var entries []LogEntry
	if err := cur.All(r.Context(), &entries); err != nil {
		h.logger.Error("failed to decode log entries",
			zap.String("game", game),
			zap.Error(err),
		)
		writeJSONError(w, r, "Failed to decode logs", "DECODE_FAILED", http.StatusInternalServerError)
		return
	}

	// Return empty array instead of null
	if entries == nil {
		entries = []LogEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(LogListResponse{
		Entries: entries,
		Total:   total,
		Limit:   params.Limit,
		Offset:  params.Offset,
	})
}

// ensureIndexes creates indexes for the unified logdata collection.
func (h *Handler) ensureIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	coll := h.db.Collection(logdataCollection)
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "game", Value: 1},
				{Key: "serverTimestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "game", Value: 1},
				{Key: "playerId", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "game", Value: 1},
				{Key: "eventType", Value: 1},
			},
		},
	}

	if _, err := coll.Indexes().CreateMany(ctx, indexes); err != nil {
		h.logger.Warn("failed to create indexes for logdata collection",
			zap.Error(err),
		)
	}
}

// ViewHandler handles GET /logs/view?game=<name> requests.
// This is a public endpoint (no authentication required) that returns an HTML view of logs.
func (h *Handler) ViewHandler(w http.ResponseWriter, r *http.Request) {
	game := r.URL.Query().Get("game")
	if game == "" {
		http.Error(w, "Missing required parameter: game", http.StatusBadRequest)
		return
	}

	// Parse limit (default 100, use 0 for all)
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n >= 0 {
			limit = n
		}
	}

	// Query logs from unified logdata collection
	coll := h.db.Collection(logdataCollection)
	filter := bson.M{"game": game}

	opts := options.Find().
		SetSort(bson.D{{Key: "serverTimestamp", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cur, err := coll.Find(r.Context(), filter, opts)
	if err != nil {
		h.logger.Error("failed to query log entries for view",
			zap.String("game", game),
			zap.Error(err),
		)
		http.Error(w, "Failed to query logs", http.StatusInternalServerError)
		return
	}
	defer cur.Close(r.Context())

	var entries []LogEntry
	if err := cur.All(r.Context(), &entries); err != nil {
		h.logger.Error("failed to decode log entries for view",
			zap.String("game", game),
			zap.Error(err),
		)
		http.Error(w, "Failed to decode logs", http.StatusInternalServerError)
		return
	}

	// Build simple HTML response
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// Write HTML header
	_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
<title>Logs for ` + game + `</title>
<style>
body { font-family: monospace; padding: 20px; }
h1 { margin-bottom: 20px; }
.entry { background: #f5f5f5; padding: 10px; margin-bottom: 10px; border-radius: 4px; }
.timestamp { color: #666; font-size: 0.9em; }
.event-type { font-weight: bold; color: #0066cc; }
.player-id { color: #006600; }
.data { white-space: pre-wrap; background: #fff; padding: 5px; margin-top: 5px; border: 1px solid #ddd; }
</style>
</head>
<body>
<h1>Logs for "` + game + `"</h1>
<p>Showing ` + strconv.Itoa(len(entries)) + ` entries</p>
`))

	// Write entries
	for _, entry := range entries {
		_, _ = w.Write([]byte(`<div class="entry">`))
		_, _ = w.Write([]byte(`<span class="timestamp">` + entry.ServerTimestamp.Format(time.RFC3339) + `</span>`))
		if entry.EventType != "" {
			_, _ = w.Write([]byte(` <span class="event-type">[` + entry.EventType + `]</span>`))
		}
		if entry.PlayerID != "" {
			_, _ = w.Write([]byte(` <span class="player-id">Player: ` + entry.PlayerID + `</span>`))
		}
		if len(entry.Data) > 0 {
			dataJSON, _ := json.MarshalIndent(entry.Data, "", "  ")
			_, _ = w.Write([]byte(`<div class="data">` + string(dataJSON) + `</div>`))
		}
		_, _ = w.Write([]byte(`</div>`))
	}

	// Write footer
	_, _ = w.Write([]byte(`</body></html>`))
}

// DownloadHandler handles GET /logs/download?game=<name> requests.
// This is a public endpoint (no authentication required) that returns logs as a JSON download.
func (h *Handler) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	game := r.URL.Query().Get("game")
	if game == "" {
		writeJSONError(w, r, "Missing required parameter: game", "MISSING_PARAM", http.StatusBadRequest)
		return
	}

	// Parse limit (default 1000, use 0 for all)
	limit := 1000
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n >= 0 {
			limit = n
		}
	}

	// Query logs from unified logdata collection
	coll := h.db.Collection(logdataCollection)
	filter := bson.M{"game": game}

	opts := options.Find().
		SetSort(bson.D{{Key: "serverTimestamp", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cur, err := coll.Find(r.Context(), filter, opts)
	if err != nil {
		h.logger.Error("failed to query log entries for download",
			zap.String("game", game),
			zap.Error(err),
		)
		writeJSONError(w, r, "Failed to query logs", "QUERY_FAILED", http.StatusInternalServerError)
		return
	}
	defer cur.Close(r.Context())

	var entries []LogEntry
	if err := cur.All(r.Context(), &entries); err != nil {
		h.logger.Error("failed to decode log entries for download",
			zap.String("game", game),
			zap.Error(err),
		)
		writeJSONError(w, r, "Failed to decode logs", "DECODE_FAILED", http.StatusInternalServerError)
		return
	}

	// Return empty array instead of null
	if entries == nil {
		entries = []LogEntry{}
	}

	// Set headers for download
	filename := game + "_logs_" + time.Now().Format("20060102_150405") + ".json"
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	_ = json.NewEncoder(w).Encode(entries)
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, r *http.Request, msg, code string, status int) {
	// Set error message in ledger context for debugging
	ledger.SetErrorMessage(r.Context(), msg)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: msg,
		Code:  code,
	})
}
