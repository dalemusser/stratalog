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
	defaultUserLimit = 20
	defaultLogLimit  = 25
)

// Handler handles log browser HTTP requests.
type Handler struct {
	db                   *mongo.Database
	store                *Store
	errLog               *errorsfeature.ErrorLogger
	logger               *zap.Logger
	defaultLimit         int
	apiKey               string
	logReadRoutesEnabled bool // whether the list/view/download routes are served (shown on the playground and docs pages)
	hub                  *Hub
}

// NewHandler creates a new log browser handler.
func NewHandler(db *mongo.Database, errLog *errorsfeature.ErrorLogger, defaultLimit int, apiKey string, logReadRoutesEnabled bool, logger *zap.Logger) *Handler {
	if defaultLimit <= 0 {
		defaultLimit = defaultLogLimit
	}
	return &Handler{
		db:                   db,
		store:                NewStore(db, logger),
		errLog:               errLog,
		logger:               logger,
		defaultLimit:         defaultLimit,
		apiKey:               apiKey,
		logReadRoutesEnabled: logReadRoutesEnabled,
		hub:                  NewHub(),
	}
}

// Hub returns the handler's event hub for broadcasting log events.
func (h *Handler) Hub() *Hub {
	return h.hub
}

// ServeList renders the main log browser page.
func (h *Handler) ServeList(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeouts.Long())
	defer cancel()

	games, err := h.store.ListGames(ctx)
	if err != nil {
		h.errLog.Log(r, "failed to list games", err)
		http.Error(w, "Failed to load games", http.StatusInternalServerError)
		return
	}

	totalAllLogs, _ := h.store.CountAllLogs(ctx)

	selectedGame := r.URL.Query().Get("game")
	selectedUser := r.URL.Query().Get("user_id")
	selectedEventType := r.URL.Query().Get("eventType")
	userSearch := r.URL.Query().Get("search")
	limitStr := r.URL.Query().Get("limit")
	afterID := r.URL.Query().Get("after")
	beforeID := r.URL.Query().Get("before")
	pageStr := r.URL.Query().Get("page")

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

	tzGroups, _ := timezones.Groups()

	data := ListVM{
		BaseVM:            viewdata.NewBaseVM(r, h.db, "Log Browser", "/dashboard"),
		TimezoneGroups:    tzGroups,
		Games:             games,
		SelectedGame:      selectedGame,
		SelectedUser:      selectedUser,
		SelectedEventType: selectedEventType,
		UserSearch:        userSearch,
		UserPage:          page,
		LogLimit:          limit,
		Limit:             limit,
		DefaultLimit:      h.defaultLimit,
		APIKey:            h.apiKey,
		TotalAllLogs:      totalAllLogs,
	}

	if selectedGame != "" && userSearch != "" {
		users, total, err := h.store.ListUsersWithCounts(ctx, selectedGame, userSearch, page, defaultUserLimit)
		if err != nil {
			h.logger.Warn("failed to list users with counts", zap.Error(err))
		} else {
			data.Users = make([]UserRowVM, len(users))
			for i, u := range users {
				data.Users[i] = UserRowVM{
					UserID:   u.UserID,
					LogCount: u.LogCount,
				}
			}
			data.UserTotal = total

			data.UserRangeStart = (page-1)*defaultUserLimit + 1
			data.UserRangeEnd = data.UserRangeStart + len(users) - 1
			if data.UserRangeEnd > int(total) {
				data.UserRangeEnd = int(total)
			}
			if total == 0 {
				data.UserRangeStart = 0
				data.UserRangeEnd = 0
			}

			data.UserHasPrev = page > 1
			data.UserHasNext = int64(page*defaultUserLimit) < total
			data.UserPrevPage = page - 1
			data.UserNextPage = page + 1
		}

		eventTypes, err := h.store.ListEventTypes(ctx, selectedGame)
		if err != nil {
			h.logger.Warn("failed to list event types", zap.Error(err))
		} else {
			data.EventTypes = make([]string, len(eventTypes))
			for i, et := range eventTypes {
				data.EventTypes[i] = et.Name
			}
		}

		logs, hasPrev, hasNext, err := h.store.ListLogs(ctx, selectedGame, selectedUser, selectedEventType, limit, afterID, beforeID)
		if err != nil {
			h.logger.Warn("failed to list logs", zap.Error(err))
		} else {
			data.Logs = make([]LogRowVM, len(logs))
			for i, l := range logs {
				fullEntry := buildFullLogEntry(l)
				jsonBytes, _ := json.MarshalIndent(fullEntry, "", "  ")
				data.Logs[i] = LogRowVM{
					ID:              l.ID.Hex(),
					Game:            l.Game,
					UserID:          l.UserID,
					EventType:       l.EventType,
					Timestamp:       l.Timestamp,
					ServerTimestamp: l.ServerTimestamp,
					Data:            string(jsonBytes),
				}
			}
			data.HasPrev = hasPrev
			data.HasNext = hasNext

			if len(logs) > 0 {
				data.PrevCursor = logs[0].ID.Hex()
				data.NextCursor = logs[len(logs)-1].ID.Hex()
			}

			total, err := h.store.CountLogs(ctx, selectedGame, selectedUser, selectedEventType)
			if err == nil {
				data.LogTotal = total
			}
		}
	}

	if r.Header.Get("HX-Request") == "true" {
		target := r.Header.Get("HX-Target")
		switch target {
		case "users-section":
			templates.RenderSnippet(w, "logbrowser/users_partial", UsersPartialVM{
				SelectedGame:   selectedGame,
				SelectedUser:   selectedUser,
				UserSearch:     userSearch,
				Users:          data.Users,
				UserTotal:      data.UserTotal,
				UserPage:       page,
				UserHasPrev:    data.UserHasPrev,
				UserHasNext:    data.UserHasNext,
				UserRangeStart: data.UserRangeStart,
				UserRangeEnd:   data.UserRangeEnd,
				UserPrevPage:   data.UserPrevPage,
				UserNextPage:   data.UserNextPage,
				Limit:          limit,
			})
			return
		case "logs-section":
			templates.RenderSnippet(w, "logbrowser/logs_partial", LogsPartialVM{
				BaseVM:            data.BaseVM,
				SelectedGame:      selectedGame,
				SelectedUser:      selectedUser,
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

// ServeUsers handles GET /users — HTMX partial for the users table.
func (h *Handler) ServeUsers(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeouts.Long())
	defer cancel()

	game := r.URL.Query().Get("game")
	search := r.URL.Query().Get("search")
	selectedUser := r.URL.Query().Get("user_id")
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

	data := UsersPartialVM{
		SelectedGame: game,
		SelectedUser: selectedUser,
		UserSearch:   search,
		UserPage:     page,
		Limit:        limit,
	}

	if game == "" || search == "" {
		templates.RenderSnippet(w, "logbrowser/users_partial", data)
		return
	}

	users, total, err := h.store.ListUsersWithCounts(ctx, game, search, page, defaultUserLimit)
	if err != nil {
		h.logger.Warn("failed to list users with counts", zap.Error(err))
		templates.RenderSnippet(w, "logbrowser/users_partial", data)
		return
	}

	data.Users = make([]UserRowVM, len(users))
	for i, u := range users {
		data.Users[i] = UserRowVM{
			UserID:   u.UserID,
			LogCount: u.LogCount,
		}
	}
	data.UserTotal = total

	data.UserRangeStart = (page-1)*defaultUserLimit + 1
	data.UserRangeEnd = data.UserRangeStart + len(users) - 1
	if data.UserRangeEnd > int(total) {
		data.UserRangeEnd = int(total)
	}
	if total == 0 {
		data.UserRangeStart = 0
		data.UserRangeEnd = 0
	}

	data.UserHasPrev = page > 1
	data.UserHasNext = int64(page*defaultUserLimit) < total
	data.UserPrevPage = page - 1
	data.UserNextPage = page + 1

	templates.RenderSnippet(w, "logbrowser/users_partial", data)
}

// ServeGamePicker handles GET /game-picker — game selector modal.
func (h *Handler) ServeGamePicker(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeouts.Medium())
	defer cancel()

	selectedGame := r.URL.Query().Get("selected")
	query := r.URL.Query().Get("q")

	games, err := h.store.ListGames(ctx)
	if err != nil {
		h.logger.Warn("failed to list games", zap.Error(err))
		games = []string{}
	}

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

	if r.Header.Get("HX-Target") == "game-list" {
		templates.RenderSnippet(w, "logbrowser/game_picker_list", data)
		return
	}

	templates.RenderSnippet(w, "logbrowser/game_picker", data)
}

// ServeLogs handles GET /data — HTMX partial for logs list.
func (h *Handler) ServeLogs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeouts.Long())
	defer cancel()

	game := r.URL.Query().Get("game")
	userID := r.URL.Query().Get("user_id")
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
		SelectedUser:      userID,
		SelectedEventType: eventType,
		Limit:             limit,
	}

	if game == "" {
		templates.RenderSnippet(w, "logbrowser/logs_partial", data)
		return
	}

	logs, hasPrev, hasNext, err := h.store.ListLogs(ctx, game, userID, eventType, limit, afterID, beforeID)
	if err != nil {
		h.logger.Warn("failed to list logs", zap.Error(err))
		templates.RenderSnippet(w, "logbrowser/logs_partial", data)
		return
	}

	data.Logs = make([]LogRowVM, len(logs))
	for i, l := range logs {
		fullEntry := buildFullLogEntry(l)
		jsonBytes, _ := json.MarshalIndent(fullEntry, "", "  ")
		data.Logs[i] = LogRowVM{
			ID:              l.ID.Hex(),
			Game:            l.Game,
			UserID:          l.UserID,
			EventType:       l.EventType,
			Timestamp:       l.Timestamp,
			ServerTimestamp: l.ServerTimestamp,
			Data:            string(jsonBytes),
		}
	}
	data.HasPrev = hasPrev
	data.HasNext = hasNext

	if len(logs) > 0 {
		data.PrevCursor = logs[0].ID.Hex()
		data.NextCursor = logs[len(logs)-1].ID.Hex()
	}

	total, err := h.store.CountLogs(ctx, game, userID, eventType)
	if err == nil {
		data.Total = total
		data.LogTotal = total
	}

	templates.RenderSnippet(w, "logbrowser/logs_partial", data)
}

// HandleDeleteLog handles POST /{game}/{id}/delete — delete a single log.
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

	w.Header().Set("HX-Trigger", "log-deleted")
	w.WriteHeader(http.StatusOK)
}

// ServePlayground renders the API playground page.
func (h *Handler) ServePlayground(w http.ResponseWriter, r *http.Request) {
	data := PlaygroundVM{
		BaseVM:      viewdata.NewBaseVM(r, h.db, "Log API Playground", "/console/api/logs"),
		APIKey:      h.apiKey,
		ListEnabled: h.logReadRoutesEnabled,
	}
	templates.Render(w, r, "logbrowser/playground", data)
}

// ServeDocs renders the API documentation page.
func (h *Handler) ServeDocs(w http.ResponseWriter, r *http.Request) {
	data := DocsVM{
		BaseVM:            viewdata.NewBaseVM(r, h.db, "Log API Documentation", "/console/api/logs"),
		MaxBatchSize:      100,
		ReadRoutesEnabled: h.logReadRoutesEnabled,
	}
	templates.Render(w, r, "logbrowser/docs", data)
}

// ServeRecentLogs renders the recent logs page showing entries across all games.
func (h *Handler) ServeRecentLogs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeouts.Medium())
	defer cancel()

	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	logs, err := h.store.ListRecentLogs(ctx, limit)
	if err != nil {
		h.errLog.Log(r, "failed to list recent logs", err)
		http.Error(w, "Failed to load recent logs", http.StatusInternalServerError)
		return
	}

	total, _ := h.store.CountAllLogs(ctx)
	tzGroups, _ := timezones.Groups()

	logRows := make([]LogRowVM, len(logs))
	for i, l := range logs {
		fullEntry := buildFullLogEntry(l)
		jsonBytes, _ := json.MarshalIndent(fullEntry, "", "  ")
		logRows[i] = LogRowVM{
			ID:              l.ID.Hex(),
			Game:            l.Game,
			UserID:          l.UserID,
			EventType:       l.EventType,
			Timestamp:       l.Timestamp,
			ServerTimestamp: l.ServerTimestamp,
			Data:            string(jsonBytes),
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

// ServeRecentLogsStream handles GET /recent/stream — SSE endpoint for real-time log updates.
func (h *Handler) ServeRecentLogsStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	ch := h.hub.Subscribe()
	defer h.hub.Unsubscribe(ch)

	h.logger.Debug("SSE client connected",
		zap.Int("subscribers", h.hub.SubscriberCount()),
	)

	_, _ = w.Write([]byte("event: connected\ndata: {\"status\":\"connected\"}\n\n"))
	flusher.Flush()

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
			jsonData, err := json.Marshal(event)
			if err != nil {
				h.logger.Warn("failed to marshal SSE event", zap.Error(err))
				continue
			}
			_, _ = w.Write([]byte("event: log\ndata: "))
			_, _ = w.Write(jsonData)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}

// HandleDeleteUserLogs handles POST /{game}/user/{userID}/delete — delete all logs for a user.
func (h *Handler) HandleDeleteUserLogs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeouts.Long())
	defer cancel()

	game := chi.URLParam(r, "game")
	userID := chi.URLParam(r, "userID")

	count, err := h.store.DeleteUserLogs(ctx, game, userID)
	if err != nil {
		h.errLog.Log(r, "failed to delete user logs", err)
		http.Error(w, "Failed to delete logs", http.StatusInternalServerError)
		return
	}

	h.logger.Info("user logs deleted",
		zap.String("game", game),
		zap.String("user_id", userID),
		zap.Int64("count", count),
	)

	w.Header().Set("HX-Trigger", "logs-deleted")
	w.WriteHeader(http.StatusOK)
}

// HandleDownloadUserLogs handles GET /download?game=X&user_id=Y — download all logs for a user as JSON.
func (h *Handler) HandleDownloadUserLogs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), timeouts.Long())
	defer cancel()

	game := r.URL.Query().Get("game")
	userID := r.URL.Query().Get("user_id")

	logs, _, _, err := h.store.ListLogs(ctx, game, userID, "", 10000, "", "")
	if err != nil {
		h.errLog.Log(r, "failed to list logs for download", err)
		http.Error(w, "Failed to load logs", http.StatusInternalServerError)
		return
	}

	entries := make([]map[string]interface{}, len(logs))
	for i, l := range logs {
		entries[i] = buildFullLogEntry(l)
	}

	now := time.Now()
	filename := "logs-" + game
	if userID != "" && userID != "__empty__" {
		filename += "-" + userID
	}
	filename += "-" + now.Format("2006-01-02-150405") + ".json"

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(entries); err != nil {
		h.logger.Warn("failed to encode logs for download", zap.Error(err))
	}
}

// buildFullLogEntry constructs a complete log entry map for JSON serialization.
func buildFullLogEntry(l LogEntry) map[string]interface{} {
	entry := make(map[string]interface{})
	entry["_id"] = l.ID.Hex()
	entry["game"] = l.Game
	if l.UserID != "" {
		entry["user_id"] = l.UserID
	}
	if l.EventType != "" {
		entry["eventType"] = l.EventType
	}
	if l.Timestamp != nil {
		entry["timestamp"] = l.Timestamp
	}
	entry["serverTimestamp"] = l.ServerTimestamp

	for k, v := range l.Data {
		entry[k] = v
	}

	return entry
}
