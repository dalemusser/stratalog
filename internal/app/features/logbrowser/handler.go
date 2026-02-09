package logbrowser

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	errorsfeature "github.com/dalemusser/stratalog/internal/app/features/errors"
	"github.com/dalemusser/stratalog/internal/app/system/timeouts"
	"github.com/dalemusser/stratalog/internal/app/system/timezones"
	"github.com/dalemusser/stratalog/internal/app/system/viewdata"
	"github.com/dalemusser/waffle/pantry/templates"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

const (
	defaultPlayerLimit = 20
	defaultLogLimit    = 25
)

// Handler handles log browser HTTP requests.
type Handler struct {
	db           *mongo.Database
	store        *Store
	errLog       *errorsfeature.ErrorLogger
	logger       *zap.Logger
	defaultLimit int
	apiKey       string
	hub          *Hub
}

// NewHandler creates a new log browser handler.
func NewHandler(db *mongo.Database, errLog *errorsfeature.ErrorLogger, defaultLimit int, apiKey string, logger *zap.Logger) *Handler {
	if defaultLimit <= 0 {
		defaultLimit = defaultLogLimit
	}
	return &Handler{
		db:           db,
		store:        NewStore(db, logger),
		errLog:       errLog,
		logger:       logger,
		defaultLimit: defaultLimit,
		apiKey:       apiKey,
		hub:          NewHub(),
	}
}

// Hub returns the handler's event hub for broadcasting log events.
func (h *Handler) Hub() *Hub {
	return h.hub
}

// ServeList renders the main log browser page.
func (h *Handler) ServeList(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeouts.Medium())
	defer cancel()

	// Load games
	games, err := h.store.ListGames(ctx)
	if err != nil {
		h.errLog.Log(r, "failed to list games", err)
		http.Error(w, "Failed to load games", http.StatusInternalServerError)
		return
	}

	// Get total log count across all games
	totalAllLogs, _ := h.store.CountAllLogs(ctx)

	// Parse query params
	selectedGame := r.URL.Query().Get("game")
	selectedPlayer := r.URL.Query().Get("player")
	selectedEventType := r.URL.Query().Get("eventType")
	playerSearch := r.URL.Query().Get("search")
	limitStr := r.URL.Query().Get("limit")
	afterID := r.URL.Query().Get("after")
	beforeID := r.URL.Query().Get("before")
	pageStr := r.URL.Query().Get("page")

	// Default to first game if none selected
	if selectedGame == "" && len(games) > 0 {
		selectedGame = games[0]
	}

	limit := h.defaultLimit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	// Load timezone groups
	tzGroups, _ := timezones.Groups()

	data := ListVM{
		BaseVM:            viewdata.NewBaseVM(r, h.db, "Log Browser", "/dashboard"),
		TimezoneGroups:    tzGroups,
		Games:             games,
		SelectedGame:      selectedGame,
		SelectedPlayer:    selectedPlayer,
		SelectedEventType: selectedEventType,
		PlayerSearch:      playerSearch,
		PlayerPage:        page,
		LogLimit:          limit,
		Limit:             limit,
		DefaultLimit:      h.defaultLimit,
		APIKey:            h.apiKey,
		TotalAllLogs:      totalAllLogs,
	}

	// If game selected, load players with counts
	if selectedGame != "" {
		players, total, err := h.store.ListPlayersWithCounts(ctx, selectedGame, playerSearch, page, defaultPlayerLimit)
		if err != nil {
			h.logger.Warn("failed to list players with counts", zap.Error(err))
		} else {
			data.Players = make([]PlayerRowVM, len(players))
			for i, p := range players {
				data.Players[i] = PlayerRowVM{
					PlayerID: p.PlayerID,
					LogCount: p.LogCount,
				}
			}
			data.PlayerTotal = total

			// Calculate pagination
			data.PlayerRangeStart = (page-1)*defaultPlayerLimit + 1
			data.PlayerRangeEnd = data.PlayerRangeStart + len(players) - 1
			if data.PlayerRangeEnd > int(total) {
				data.PlayerRangeEnd = int(total)
			}
			if total == 0 {
				data.PlayerRangeStart = 0
				data.PlayerRangeEnd = 0
			}

			data.PlayerHasPrev = page > 1
			data.PlayerHasNext = int64(page*defaultPlayerLimit) < total
			data.PlayerPrevPage = page - 1
			data.PlayerNextPage = page + 1
		}

		// Load event types
		eventTypes, err := h.store.ListEventTypes(ctx, selectedGame)
		if err != nil {
			h.logger.Warn("failed to list event types", zap.Error(err))
		} else {
			data.EventTypes = make([]string, len(eventTypes))
			for i, et := range eventTypes {
				data.EventTypes[i] = et.Name
			}
		}

		// Load logs
		logs, hasPrev, hasNext, err := h.store.ListLogs(ctx, selectedGame, selectedPlayer, selectedEventType, limit, afterID, beforeID)
		if err != nil {
			h.logger.Warn("failed to list logs", zap.Error(err))
		} else {
			data.Logs = make([]LogRowVM, len(logs))
			for i, l := range logs {
				// Build full log entry for display/download
				fullEntry := buildFullLogEntry(l)
				jsonBytes, _ := json.MarshalIndent(fullEntry, "", "  ")
				data.Logs[i] = LogRowVM{
					ID:          l.ID.Hex(),
					Game:        l.Game,
					PlayerID:    l.PlayerID,
					EventType:   l.EventType,
					Timestamp:   l.Timestamp,
					ServerTimestamp: l.ServerTimestamp,
					Data:        string(jsonBytes),
				}
			}
			data.HasPrev = hasPrev
			data.HasNext = hasNext

			// Set cursors for pagination
			if len(logs) > 0 {
				data.PrevCursor = logs[0].ID.Hex()
				data.NextCursor = logs[len(logs)-1].ID.Hex()
			}

			// Get total count
			total, err := h.store.CountLogs(ctx, selectedGame, selectedPlayer, selectedEventType)
			if err == nil {
				data.LogTotal = total
			}
		}
	}

	// Check if HTMX request targeting specific elements
	if r.Header.Get("HX-Request") == "true" {
		target := r.Header.Get("HX-Target")
		switch target {
		case "players-section":
			templates.RenderSnippet(w, "logbrowser/players_partial", PlayersPartialVM{
				SelectedGame:     selectedGame,
				SelectedPlayer:   selectedPlayer,
				PlayerSearch:     playerSearch,
				Players:          data.Players,
				PlayerTotal:      data.PlayerTotal,
				PlayerPage:       page,
				PlayerHasPrev:    data.PlayerHasPrev,
				PlayerHasNext:    data.PlayerHasNext,
				PlayerRangeStart: data.PlayerRangeStart,
				PlayerRangeEnd:   data.PlayerRangeEnd,
				PlayerPrevPage:   data.PlayerPrevPage,
				PlayerNextPage:   data.PlayerNextPage,
				Limit:            limit,
			})
			return
		case "logs-section":
			templates.RenderSnippet(w, "logbrowser/logs_partial", LogsPartialVM{
				BaseVM:            data.BaseVM,
				SelectedGame:      selectedGame,
				SelectedPlayer:    selectedPlayer,
				SelectedEventType: selectedEventType,
				Logs:              data.Logs,
				Total:             data.LogTotal,
				Limit:             limit,
				HasPrev:           data.HasPrev,
				HasNext:           data.HasNext,
				PrevCursor:        data.PrevCursor,
				NextCursor:        data.NextCursor,
			})
			return
		}
	}

	templates.Render(w, r, "logbrowser/list", data)
}

// ServePlayers handles GET /players - HTMX partial for players table.
func (h *Handler) ServePlayers(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeouts.Medium())
	defer cancel()

	game := r.URL.Query().Get("game")
	search := r.URL.Query().Get("search")
	selectedPlayer := r.URL.Query().Get("player")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := h.defaultLimit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	data := PlayersPartialVM{
		SelectedGame:   game,
		SelectedPlayer: selectedPlayer,
		PlayerSearch:   search,
		PlayerPage:     page,
		Limit:          limit,
	}

	if game == "" {
		templates.RenderSnippet(w, "logbrowser/players_partial", data)
		return
	}

	players, total, err := h.store.ListPlayersWithCounts(ctx, game, search, page, defaultPlayerLimit)
	if err != nil {
		h.logger.Warn("failed to list players with counts", zap.Error(err))
		templates.RenderSnippet(w, "logbrowser/players_partial", data)
		return
	}

	data.Players = make([]PlayerRowVM, len(players))
	for i, p := range players {
		data.Players[i] = PlayerRowVM{
			PlayerID: p.PlayerID,
			LogCount: p.LogCount,
		}
	}
	data.PlayerTotal = total

	// Calculate pagination
	data.PlayerRangeStart = (page-1)*defaultPlayerLimit + 1
	data.PlayerRangeEnd = data.PlayerRangeStart + len(players) - 1
	if data.PlayerRangeEnd > int(total) {
		data.PlayerRangeEnd = int(total)
	}
	if total == 0 {
		data.PlayerRangeStart = 0
		data.PlayerRangeEnd = 0
	}

	data.PlayerHasPrev = page > 1
	data.PlayerHasNext = int64(page*defaultPlayerLimit) < total
	data.PlayerPrevPage = page - 1
	data.PlayerNextPage = page + 1

	templates.RenderSnippet(w, "logbrowser/players_partial", data)
}

// ServeGamePicker handles GET /game-picker - game selector modal.
func (h *Handler) ServeGamePicker(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeouts.Medium())
	defer cancel()

	selectedGame := r.URL.Query().Get("selected")
	query := r.URL.Query().Get("q")

	// Load games
	games, err := h.store.ListGames(ctx)
	if err != nil {
		h.logger.Warn("failed to list games", zap.Error(err))
		games = []string{}
	}

	// Filter games by query if provided
	var filteredGames []GamePickerItem
	queryLower := strings.ToLower(query)
	for _, g := range games {
		if query == "" || strings.Contains(strings.ToLower(g), queryLower) {
			filteredGames = append(filteredGames, GamePickerItem{
				Name:     g,
				Selected: g == selectedGame,
			})
		}
	}

	data := GamePickerVM{
		Games:      filteredGames,
		SelectedID: selectedGame,
		Query:      query,
	}

	// If HTMX request targeting just the list, render only the list portion
	if r.Header.Get("HX-Target") == "game-list" {
		templates.RenderSnippet(w, "logbrowser/game_picker_list", data)
		return
	}

	templates.RenderSnippet(w, "logbrowser/game_picker", data)
}

// ServeLogs handles GET /data - HTMX partial for logs list.
func (h *Handler) ServeLogs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeouts.Medium())
	defer cancel()

	game := r.URL.Query().Get("game")
	player := r.URL.Query().Get("player")
	eventType := r.URL.Query().Get("eventType")
	limitStr := r.URL.Query().Get("limit")
	afterID := r.URL.Query().Get("after")
	beforeID := r.URL.Query().Get("before")

	limit := h.defaultLimit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	data := LogsPartialVM{
		BaseVM:            viewdata.NewBaseVM(r, h.db, "", ""),
		SelectedGame:      game,
		SelectedPlayer:    player,
		SelectedEventType: eventType,
		Limit:             limit,
	}

	if game == "" {
		templates.RenderSnippet(w, "logbrowser/logs_partial", data)
		return
	}

	logs, hasPrev, hasNext, err := h.store.ListLogs(ctx, game, player, eventType, limit, afterID, beforeID)
	if err != nil {
		h.logger.Warn("failed to list logs", zap.Error(err))
		templates.RenderSnippet(w, "logbrowser/logs_partial", data)
		return
	}

	data.Logs = make([]LogRowVM, len(logs))
	for i, l := range logs {
		// Build full log entry for display/download
		fullEntry := buildFullLogEntry(l)
		jsonBytes, _ := json.MarshalIndent(fullEntry, "", "  ")
		data.Logs[i] = LogRowVM{
			ID:          l.ID.Hex(),
			Game:        l.Game,
			PlayerID:    l.PlayerID,
			EventType:   l.EventType,
			Timestamp:   l.Timestamp,
			ServerTimestamp: l.ServerTimestamp,
			Data:        string(jsonBytes),
		}
	}
	data.HasPrev = hasPrev
	data.HasNext = hasNext

	if len(logs) > 0 {
		data.PrevCursor = logs[0].ID.Hex()
		data.NextCursor = logs[len(logs)-1].ID.Hex()
	}

	total, err := h.store.CountLogs(ctx, game, player, eventType)
	if err == nil {
		data.Total = total
		data.LogTotal = total
	}

	templates.RenderSnippet(w, "logbrowser/logs_partial", data)
}

// HandleDeleteLog handles POST /{game}/{id}/delete - delete a single log.
func (h *Handler) HandleDeleteLog(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeouts.Short())
	defer cancel()

	game := chi.URLParam(r, "game")
	idStr := chi.URLParam(r, "id")

	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid log ID", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteLog(ctx, game, id); err != nil {
		h.errLog.Log(r, "failed to delete log", err)
		http.Error(w, "Failed to delete log", http.StatusInternalServerError)
		return
	}

	h.logger.Info("log deleted",
		zap.String("game", game),
		zap.String("id", idStr),
	)

	// Return success - the client will refresh the list
	w.Header().Set("HX-Trigger", "log-deleted")
	w.WriteHeader(http.StatusOK)
}

// ServePlayground renders the API playground page.
func (h *Handler) ServePlayground(w http.ResponseWriter, r *http.Request) {
	data := PlaygroundVM{
		BaseVM: viewdata.NewBaseVM(r, h.db, "Log API Playground", "/console/api/logs"),
		APIKey: h.apiKey,
	}
	templates.Render(w, r, "logbrowser/playground", data)
}

// ServeDocs renders the API documentation page.
func (h *Handler) ServeDocs(w http.ResponseWriter, r *http.Request) {
	data := DocsVM{
		BaseVM:       viewdata.NewBaseVM(r, h.db, "Log API Documentation", "/console/api/logs"),
		MaxBatchSize: 100,
	}
	templates.Render(w, r, "logbrowser/docs", data)
}

// ServeRecentLogs renders the recent logs page showing entries across all games.
func (h *Handler) ServeRecentLogs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeouts.Medium())
	defer cancel()

	// Parse limit from query params
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	// Load recent logs
	logs, err := h.store.ListRecentLogs(ctx, limit)
	if err != nil {
		h.errLog.Log(r, "failed to list recent logs", err)
		http.Error(w, "Failed to load recent logs", http.StatusInternalServerError)
		return
	}

	// Get total count
	total, _ := h.store.CountAllLogs(ctx)

	// Load timezone groups
	tzGroups, _ := timezones.Groups()

	// Build log rows
	logRows := make([]LogRowVM, len(logs))
	for i, l := range logs {
		// Build full log entry for display/download
		fullEntry := buildFullLogEntry(l)
		jsonBytes, _ := json.MarshalIndent(fullEntry, "", "  ")
		logRows[i] = LogRowVM{
			ID:          l.ID.Hex(),
			Game:        l.Game,
			PlayerID:    l.PlayerID,
			EventType:   l.EventType,
			Timestamp:   l.Timestamp,
			ServerTimestamp: l.ServerTimestamp,
			Data:        string(jsonBytes),
		}
	}

	data := RecentLogsVM{
		BaseVM:         viewdata.NewBaseVM(r, h.db, "Recent Logs", "/console/api/logs"),
		TimezoneGroups: tzGroups,
		Logs:           logRows,
		Total:          total,
		Limit:          limit,
		LimitOptions:   []int{25, 50, 100, 250, 500, 1000},
	}

	templates.Render(w, r, "logbrowser/recent", data)
}

// ServeRecentLogsStream handles GET /recent/stream - SSE endpoint for real-time log updates.
func (h *Handler) ServeRecentLogsStream(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Check if we can flush
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Subscribe to the hub
	ch := h.hub.Subscribe()
	defer h.hub.Unsubscribe(ch)

	h.logger.Debug("SSE client connected",
		zap.Int("subscribers", h.hub.SubscriberCount()),
	)

	// Send initial connection event
	_, _ = w.Write([]byte("event: connected\ndata: {\"status\":\"connected\"}\n\n"))
	flusher.Flush()

	// Stream events until client disconnects
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			h.logger.Debug("SSE client disconnected",
				zap.Int("subscribers", h.hub.SubscriberCount()-1),
			)
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			// Marshal event to JSON
			jsonData, err := json.Marshal(event)
			if err != nil {
				h.logger.Warn("failed to marshal SSE event", zap.Error(err))
				continue
			}
			// Write SSE event
			_, _ = w.Write([]byte("event: log\ndata: "))
			_, _ = w.Write(jsonData)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}

// HandleDeletePlayerLogs handles POST /{game}/player/{playerID}/delete - delete all logs for a player.
func (h *Handler) HandleDeletePlayerLogs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeouts.Medium())
	defer cancel()

	game := chi.URLParam(r, "game")
	playerID := chi.URLParam(r, "playerID")

	count, err := h.store.DeletePlayerLogs(ctx, game, playerID)
	if err != nil {
		h.errLog.Log(r, "failed to delete player logs", err)
		http.Error(w, "Failed to delete logs", http.StatusInternalServerError)
		return
	}

	h.logger.Info("player logs deleted",
		zap.String("game", game),
		zap.String("player_id", playerID),
		zap.Int64("count", count),
	)

	// Return success - the client will refresh
	w.Header().Set("HX-Trigger", "logs-deleted")
	w.WriteHeader(http.StatusOK)
}

// HandleDownloadPlayerLogs handles GET /download?game=X&player=Y - download all logs for a player as JSON.
func (h *Handler) HandleDownloadPlayerLogs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeouts.Long())
	defer cancel()

	game := r.URL.Query().Get("game")
	playerID := r.URL.Query().Get("player")

	// Get all logs for this player (no pagination limit)
	logs, _, _, err := h.store.ListLogs(ctx, game, playerID, "", 10000, "", "")
	if err != nil {
		h.errLog.Log(r, "failed to list logs for download", err)
		http.Error(w, "Failed to load logs", http.StatusInternalServerError)
		return
	}

	// Build full log entries
	entries := make([]map[string]interface{}, len(logs))
	for i, l := range logs {
		entries[i] = buildFullLogEntry(l)
	}

	// Set download headers with timestamp
	now := time.Now()
	filename := "logs-" + game
	if playerID != "" && playerID != "__empty__" {
		filename += "-" + playerID
	}
	filename += "-" + now.Format("2006-01-02-150405") + ".json"

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

	// Write JSON
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(entries); err != nil {
		h.logger.Warn("failed to encode logs for download", zap.Error(err))
	}
}

// buildFullLogEntry constructs a complete log entry map for JSON serialization.
// It includes all standard fields plus any extra data fields.
func buildFullLogEntry(l LogEntry) map[string]interface{} {
	entry := make(map[string]interface{})
	entry["_id"] = l.ID.Hex()
	entry["game"] = l.Game
	if l.PlayerID != "" {
		entry["playerId"] = l.PlayerID
	}
	if l.EventType != "" {
		entry["eventType"] = l.EventType
	}
	if l.Timestamp != nil {
		entry["timestamp"] = l.Timestamp
	}
	entry["serverTimestamp"] = l.ServerTimestamp

	// Add all extra data fields
	for k, v := range l.Data {
		entry[k] = v
	}

	return entry
}
