package logapi

import (
	"context"
	"encoding/json"
	"html"
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

// gameRegex validates game names (alphanumeric, underscores, hyphens only).
var gameRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// userIDRegex matches a 24-character lowercase hex string (Mongo ObjectID hex).
// All player identity values must match this — playerId and other variants
// are not accepted.
var userIDRegex = regexp.MustCompile(`^[0-9a-f]{24}$`)

// logdataCollection is the unified collection name for all log data.
const logdataCollection = "logdata"

// The most entries the staff view and download pages return in one request.
// Both read the whole result into memory, so neither is unbounded; a larger
// export goes through the log browser's per-user download or the database.
const (
	viewDefaultLimit     = 100
	viewMaxLimit         = 1000
	downloadDefaultLimit = 1000
	downloadMaxLimit     = 10000
)

// pageLimit reads the "limit" query parameter: def when it is absent or not
// a positive number, never more than max.
func pageLimit(r *http.Request, def, max int) int {
	limit := def
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > max {
		limit = max
	}
	return limit
}

// LogBroadcaster is a function that broadcasts log events to SSE subscribers.
type LogBroadcaster func(game, userID, eventType string, serverTimestamp time.Time, data map[string]interface{})

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

// SubmitHandler handles POST /api/log/submit (and the legacy /logs path).
// It accepts both single log entries and batch submissions.
//
// Identity contract:
//   - Each entry MUST include "user_id" as a 24-character lowercase hex string
//     (the hex form of stratahub.users._id).
//   - "playerId" and other identity field names are rejected.
//
// Single entry format:
//
//	{
//	    "game": "mhs",
//	    "user_id": "69b4449ec6006ac370dad9df",
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
//	        {"user_id": "69b4449ec6006ac370dad9df", "eventType": "level_start", "level": 5},
//	        {"user_id": "69b4449ec6006ac370dad9df", "eventType": "level_complete", "score": 1000}
//	    ]
//	}
func (h *Handler) SubmitHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var raw map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		if err.Error() == "http: request body too large" {
			writeJSONError(w, r, "request body too large", "BODY_TOO_LARGE", http.StatusRequestEntityTooLarge)
			return
		}
		writeJSONError(w, r, "Invalid JSON payload", "INVALID_JSON", http.StatusBadRequest)
		return
	}

	if entries, ok := raw["entries"].([]interface{}); ok {
		h.handleBatchSubmit(w, r, raw, entries)
		return
	}

	h.handleSingleSubmit(w, r, raw)
}

// identityKeysSeen returns the names of identity-shaped fields present in
// the given entry. Used for diagnostic logging on rejections — captures the
// shape of what the client sent without including the values themselves
// (which may be PII or large payloads).
func identityKeysSeen(entry map[string]interface{}) []string {
	candidates := []string{"user_id", "playerId", "PlayerID", "player_id", "login_id", "loginID", "LoginID", "email", "Email"}
	seen := make([]string, 0, 2)
	for _, k := range candidates {
		if _, ok := entry[k]; ok {
			seen = append(seen, k)
		}
	}
	return seen
}

// logRejection emits a Warn-level zap entry whenever a submission is
// rejected client-side validation. This surfaces "client is misbehaving"
// (e.g., a stale game build still sending playerId) on the server side
// without having to wait for a bug report. Level Warn keeps healthy
// traffic quiet but surfaces validation failures by default.
func (h *Handler) logRejection(r *http.Request, batchIndex int, msg, code string, entry, envelope map[string]interface{}) {
	game := ""
	if envelope != nil {
		game, _ = envelope["game"].(string)
	}
	if game == "" && entry != nil {
		game, _ = entry["game"].(string)
	}
	fields := []zap.Field{
		zap.String("code", code),
		zap.String("error_msg", msg),
		zap.String("game", game),
		zap.String("remote_addr", r.RemoteAddr),
		zap.String("user_agent", r.UserAgent()),
	}
	if batchIndex >= 0 {
		fields = append(fields, zap.Int("batch_index", batchIndex))
	}
	if entry != nil {
		fields = append(fields, zap.Strings("identity_keys_present", identityKeysSeen(entry)))
	}
	h.logger.Warn("rejected log submission", fields...)
}

// validateEntry checks the required fields on an entry: a present, valid
// game name (single-entry only — batch entries inherit game from the
// envelope), and a 24-char hex user_id. It also rejects the legacy
// playerId field.
func validateEntry(entry map[string]interface{}, requireGame bool) (errMsg, errCode string) {
	if requireGame {
		game, ok := entry["game"].(string)
		if !ok || game == "" {
			return "missing or invalid 'game' field", "MISSING_FIELD"
		}
		if !gameRegex.MatchString(game) {
			return "invalid 'game' value", "INVALID_GAME"
		}
	}
	if _, hasPlayerID := entry["playerId"]; hasPlayerID {
		return "'playerId' is not accepted; submit 'user_id' instead", "DEPRECATED_FIELD"
	}
	uid, ok := entry["user_id"].(string)
	if !ok || uid == "" {
		return "missing or invalid 'user_id' field", "MISSING_FIELD"
	}
	if !userIDRegex.MatchString(uid) {
		return "'user_id' must be a 24-character lowercase hex string", "INVALID_USER_ID"
	}
	return "", ""
}

// handleSingleSubmit processes a single log entry submission.
func (h *Handler) handleSingleSubmit(w http.ResponseWriter, r *http.Request, raw map[string]interface{}) {
	if msg, code := validateEntry(raw, true); msg != "" {
		h.logRejection(r, -1, msg, code, raw, nil)
		writeJSONError(w, r, msg, code, http.StatusBadRequest)
		return
	}

	game, _ := raw["game"].(string)
	userID, _ := raw["user_id"].(string)

	now := time.Now().UTC()
	raw["serverTimestamp"] = now

	coll := h.db.Collection(logdataCollection)
	if _, err := coll.InsertOne(r.Context(), raw); err != nil {
		h.logger.Error("failed to insert log entry",
			zap.String("game", game),
			zap.String("user_id", userID),
			zap.Error(err),
		)
		writeJSONError(w, r, "Failed to save log entry", "INSERT_FAILED", http.StatusInternalServerError)
		return
	}

	eventType, _ := raw["eventType"].(string)
	h.logger.Debug("log entry saved",
		zap.String("game", game),
		zap.String("user_id", userID),
		zap.String("eventType", eventType),
	)

	if h.broadcaster != nil {
		data := extractData(raw)
		h.broadcaster(game, userID, eventType, now, data)
	}

	go h.ensureIndexes()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(LogResponse{
		Status:     "success",
		ReceivedAt: now.Format(time.RFC3339),
	})
}

// handleBatchSubmit processes a batch log entry submission.
func (h *Handler) handleBatchSubmit(w http.ResponseWriter, r *http.Request, raw map[string]interface{}, entries []interface{}) {
	game, ok := raw["game"].(string)
	if !ok || game == "" {
		h.logRejection(r, -1, "missing or invalid 'game' field", "MISSING_FIELD", nil, raw)
		writeJSONError(w, r, "missing or invalid 'game' field", "MISSING_FIELD", http.StatusBadRequest)
		return
	}
	if !gameRegex.MatchString(game) {
		h.logRejection(r, -1, "invalid 'game' value", "INVALID_GAME", nil, raw)
		writeJSONError(w, r, "invalid 'game' value", "INVALID_GAME", http.StatusBadRequest)
		return
	}

	if len(entries) == 0 {
		h.logRejection(r, -1, "Entries array is empty", "EMPTY_ENTRIES", nil, raw)
		writeJSONError(w, r, "Entries array is empty", "EMPTY_ENTRIES", http.StatusBadRequest)
		return
	}
	if len(entries) > h.maxBatchSize {
		h.logRejection(r, -1, "Batch size exceeds maximum of "+strconv.Itoa(h.maxBatchSize), "BATCH_TOO_LARGE", nil, raw)
		writeJSONError(w, r, "Batch size exceeds maximum of "+strconv.Itoa(h.maxBatchSize), "BATCH_TOO_LARGE", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	docs := make([]interface{}, 0, len(entries))

	for i, e := range entries {
		entryMap, ok := e.(map[string]interface{})
		if !ok {
			h.logRejection(r, i, "Invalid entry at index "+strconv.Itoa(i), "INVALID_ENTRY", nil, raw)
			writeJSONError(w, r, "Invalid entry at index "+strconv.Itoa(i), "INVALID_ENTRY", http.StatusBadRequest)
			return
		}
		if msg, code := validateEntry(entryMap, false); msg != "" {
			h.logRejection(r, i, msg, code, entryMap, raw)
			writeJSONError(w, r, "entry "+strconv.Itoa(i)+": "+msg, code, http.StatusBadRequest)
			return
		}
		entryMap["game"] = game
		entryMap["serverTimestamp"] = now
		docs = append(docs, entryMap)
	}

	coll := h.db.Collection(logdataCollection)
	if _, err := coll.InsertMany(r.Context(), docs); err != nil {
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

	if h.broadcaster != nil {
		for _, doc := range docs {
			if entryMap, ok := doc.(map[string]interface{}); ok {
				userID, _ := entryMap["user_id"].(string)
				eventType, _ := entryMap["eventType"].(string)
				data := extractData(entryMap)
				h.broadcaster(game, userID, eventType, now, data)
			}
		}
	}

	go h.ensureIndexes()

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
//   - user_id: Filter by user ID (24-char hex)
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

	userID := r.URL.Query().Get("user_id")
	if userID != "" && !userIDRegex.MatchString(userID) {
		h.logger.Warn("rejected log list query",
			zap.String("code", "INVALID_USER_ID"),
			zap.String("error_msg", "'user_id' must be a 24-character lowercase hex string"),
			zap.String("game", game),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
		)
		writeJSONError(w, r, "'user_id' must be a 24-character lowercase hex string", "INVALID_USER_ID", http.StatusBadRequest)
		return
	}

	params := LogQueryParams{
		Game:      game,
		UserID:    userID,
		EventType: r.URL.Query().Get("eventType"),
	}

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

	filter := bson.M{"game": game}
	if params.UserID != "" {
		filter["user_id"] = params.UserID
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

	coll := h.db.Collection(logdataCollection)

	total, err := coll.CountDocuments(r.Context(), filter)
	if err != nil {
		h.logger.Error("failed to count log entries",
			zap.String("game", game),
			zap.Error(err),
		)
		writeJSONError(w, r, "Failed to query logs", "QUERY_FAILED", http.StatusInternalServerError)
		return
	}

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
				{Key: "user_id", Value: 1},
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

// ViewHandler handles GET /logs/view?game=<name> requests: an HTML view of
// the newest entries (limit: 100 by default, 1000 at most). The route
// requires a console sign-in (admin or developer).
func (h *Handler) ViewHandler(w http.ResponseWriter, r *http.Request) {
	game := r.URL.Query().Get("game")
	if game == "" {
		http.Error(w, "Missing required parameter: game", http.StatusBadRequest)
		return
	}
	if !gameRegex.MatchString(game) {
		http.Error(w, "Invalid parameter: game", http.StatusBadRequest)
		return
	}

	limit := pageLimit(r, viewDefaultLimit, viewMaxLimit)

	coll := h.db.Collection(logdataCollection)
	filter := bson.M{"game": game}

	opts := options.Find().
		SetSort(bson.D{{Key: "serverTimestamp", Value: -1}}).
		SetLimit(int64(limit))

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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
<title>Logs for ` + html.EscapeString(game) + `</title>
<style>
body { font-family: monospace; padding: 20px; }
h1 { margin-bottom: 20px; }
.entry { background: #f5f5f5; padding: 10px; margin-bottom: 10px; border-radius: 4px; }
.timestamp { color: #666; font-size: 0.9em; }
.event-type { font-weight: bold; color: #0066cc; }
.user-id { color: #006600; }
.data { white-space: pre-wrap; background: #fff; padding: 5px; margin-top: 5px; border: 1px solid #ddd; }
</style>
</head>
<body>
<h1>Logs for "` + html.EscapeString(game) + `"</h1>
<p>Showing ` + strconv.Itoa(len(entries)) + ` entries</p>
`))

	for _, entry := range entries {
		_, _ = w.Write([]byte(`<div class="entry">`))
		_, _ = w.Write([]byte(`<span class="timestamp">` + entry.ServerTimestamp.Format(time.RFC3339) + `</span>`))
		if entry.EventType != "" {
			_, _ = w.Write([]byte(` <span class="event-type">[` + html.EscapeString(entry.EventType) + `]</span>`))
		}
		if entry.UserID != "" {
			_, _ = w.Write([]byte(` <span class="user-id">User: ` + html.EscapeString(entry.UserID) + `</span>`))
		}
		if len(entry.Data) > 0 {
			dataJSON, _ := json.MarshalIndent(entry.Data, "", "  ")
			_, _ = w.Write([]byte(`<div class="data">` + html.EscapeString(string(dataJSON)) + `</div>`))
		}
		_, _ = w.Write([]byte(`</div>`))
	}

	_, _ = w.Write([]byte(`</body></html>`))
}

// DownloadHandler handles GET /logs/download?game=<name> requests: the
// newest entries as a JSON download (limit: 1000 by default, 10000 at most).
// The route requires a console sign-in (admin or developer).
func (h *Handler) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	game := r.URL.Query().Get("game")
	if game == "" {
		writeJSONError(w, r, "Missing required parameter: game", "MISSING_PARAM", http.StatusBadRequest)
		return
	}
	if !gameRegex.MatchString(game) {
		writeJSONError(w, r, "invalid 'game' value", "INVALID_GAME", http.StatusBadRequest)
		return
	}

	limit := pageLimit(r, downloadDefaultLimit, downloadMaxLimit)

	coll := h.db.Collection(logdataCollection)
	filter := bson.M{"game": game}

	opts := options.Find().
		SetSort(bson.D{{Key: "serverTimestamp", Value: -1}}).
		SetLimit(int64(limit))

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

	if entries == nil {
		entries = []LogEntry{}
	}

	filename := game + "_logs_" + time.Now().Format("20060102_150405") + ".json"
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	_ = json.NewEncoder(w).Encode(entries)
}

// extractData returns the entry map minus the structural fields stratalog
// owns (game, user_id, eventType, timestamps, _id). The remainder is treated
// as the per-entry payload broadcast to SSE subscribers.
func extractData(entry map[string]interface{}) map[string]interface{} {
	data := make(map[string]interface{})
	for k, v := range entry {
		switch k {
		case "game", "user_id", "eventType", "timestamp", "serverTimestamp", "_id":
			continue
		}
		data[k] = v
	}
	return data
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, r *http.Request, msg, code string, status int) {
	ledger.SetErrorMessage(r.Context(), msg)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: msg,
		Code:  code,
	})
}
