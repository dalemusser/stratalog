package logbrowser

import (
	"context"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// LogEntry represents a log entry in the database.
// UserID is the 24-char hex string of stratahub.users._id.
type LogEntry struct {
	ID              primitive.ObjectID     `bson:"_id,omitempty"`
	Game            string                 `bson:"game"`
	UserID          string                 `bson:"user_id,omitempty"`
	EventType       string                 `bson:"eventType,omitempty"`
	Timestamp       *time.Time             `bson:"timestamp,omitempty"`
	ServerTimestamp time.Time              `bson:"serverTimestamp"`
	Data            map[string]interface{} `bson:"data,omitempty"`
}

// UserWithCount represents a user with their log count.
type UserWithCount struct {
	UserID   string
	LogCount int64
}

// Store handles log browser database operations.
type Store struct {
	db     *mongo.Database
	logger *zap.Logger
}

// NewStore creates a new log browser store.
func NewStore(db *mongo.Database, logger *zap.Logger) *Store {
	return &Store{db: db, logger: logger}
}

// logdataCollection is the unified collection name for all log data.
const logdataCollection = "logdata"

// ListGames returns all games that have logs.
func (s *Store) ListGames(ctx context.Context) ([]string, error) {
	coll := s.db.Collection(logdataCollection)
	values, err := coll.Distinct(ctx, "game", bson.M{})
	if err != nil {
		return nil, err
	}

	games := make([]string, 0, len(values))
	for _, v := range values {
		if game, ok := v.(string); ok && game != "" {
			games = append(games, game)
		}
	}

	sort.Strings(games)
	return games, nil
}

// ListUsersWithCounts returns users with their log counts for a game.
// Optimized to avoid scanning the full collection twice:
//   - Without search: uses distinct() for the user list (fast index scan via
//     idx_logdata_game_user_id), then counts per user only for the current page.
//   - With search: uses distinct() + client-side prefix filter, then counts
//     only for the current page.
func (s *Store) ListUsersWithCounts(ctx context.Context, game, search string, page, limit int) ([]UserWithCount, int64, error) {
	coll := s.db.Collection(logdataCollection)

	if search == "" {
		return s.listUsersDistinct(ctx, coll, game, page, limit)
	}
	return s.listUsersSearch(ctx, coll, game, search, page, limit)
}

func (s *Store) listUsersDistinct(ctx context.Context, coll *mongo.Collection, game string, page, limit int) ([]UserWithCount, int64, error) {
	values, err := coll.Distinct(ctx, "user_id", bson.M{"game": game})
	if err != nil {
		return nil, 0, err
	}

	var userIDs []string
	for _, v := range values {
		id, _ := v.(string)
		userIDs = append(userIDs, id)
	}
	sort.Strings(userIDs)

	total := int64(len(userIDs))

	skip := (page - 1) * limit
	if skip >= len(userIDs) {
		return nil, total, nil
	}
	end := skip + limit
	if end > len(userIDs) {
		end = len(userIDs)
	}
	pageIDs := userIDs[skip:end]

	results := make([]UserWithCount, len(pageIDs))
	for i, uid := range pageIDs {
		filter := bson.M{"game": game}
		if uid == "" {
			filter["$or"] = []bson.M{
				{"user_id": nil},
				{"user_id": ""},
				{"user_id": bson.M{"$exists": false}},
			}
		} else {
			filter["user_id"] = uid
		}
		count, err := coll.CountDocuments(ctx, filter)
		if err != nil {
			s.logger.Warn("failed to count logs for user", zap.String("user_id", uid), zap.Error(err))
			count = 0
		}
		results[i] = UserWithCount{UserID: uid, LogCount: count}
	}

	return results, total, nil
}

func (s *Store) listUsersSearch(ctx context.Context, coll *mongo.Collection, game, search string, page, limit int) ([]UserWithCount, int64, error) {
	values, err := coll.Distinct(ctx, "user_id", bson.M{"game": game})
	if err != nil {
		return nil, 0, err
	}

	searchLower := strings.ToLower(search)
	var matched []string
	for _, v := range values {
		id, _ := v.(string)
		if strings.HasPrefix(strings.ToLower(id), searchLower) {
			matched = append(matched, id)
		}
	}
	sort.Strings(matched)

	total := int64(len(matched))

	skip := (page - 1) * limit
	if skip >= len(matched) {
		return nil, total, nil
	}
	end := skip + limit
	if end > len(matched) {
		end = len(matched)
	}
	pageIDs := matched[skip:end]

	results := make([]UserWithCount, len(pageIDs))
	for i, uid := range pageIDs {
		filter := bson.M{"game": game, "user_id": uid}
		count, err := coll.CountDocuments(ctx, filter)
		if err != nil {
			count = 0
		}
		results[i] = UserWithCount{UserID: uid, LogCount: count}
	}

	return results, total, nil
}

// ListEventTypes returns all event types for a game.
func (s *Store) ListEventTypes(ctx context.Context, game string) ([]EventTypeItem, error) {
	coll := s.db.Collection(logdataCollection)

	values, err := coll.Distinct(ctx, "eventType", bson.M{"game": game})
	if err != nil {
		return nil, err
	}

	var results []EventTypeItem
	for _, v := range values {
		name, _ := v.(string)
		if name != "" {
			results = append(results, EventTypeItem{Name: name})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	return results, nil
}

// ListLogs returns logs with cursor-based pagination.
// userID is the 24-char hex of stratahub.users._id; pass "__empty__" to filter
// for logs missing user_id (should never occur after the de-identification cutover).
func (s *Store) ListLogs(ctx context.Context, game, userID, eventType string, limit int, afterID, beforeID string) ([]LogEntry, bool, bool, error) {
	coll := s.db.Collection(logdataCollection)

	filter := bson.M{"game": game}
	if userID == "__empty__" {
		filter["$or"] = []bson.M{
			{"user_id": nil},
			{"user_id": ""},
			{"user_id": bson.M{"$exists": false}},
		}
	} else if userID != "" {
		filter["user_id"] = userID
	}
	if eventType != "" {
		filter["eventType"] = eventType
	}

	sortDir := -1
	if beforeID != "" {
		if oid, err := primitive.ObjectIDFromHex(beforeID); err == nil {
			filter["_id"] = bson.M{"$gt": oid}
			sortDir = 1
		}
	} else if afterID != "" {
		if oid, err := primitive.ObjectIDFromHex(afterID); err == nil {
			filter["_id"] = bson.M{"$lt": oid}
		}
	}

	// Sort by serverTimestamp only — DocumentDB's planner can't combine a
	// compound sort with the idx_game_serverTimestamp index, so adding _id
	// as a tiebreaker forces a full in-memory SORT step (12+ seconds on
	// 1.8M+ rows) that exceeds the request timeout.
	opts := options.Find().
		SetSort(bson.D{{Key: "serverTimestamp", Value: sortDir}}).
		SetLimit(int64(limit + 1))

	cur, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, false, false, err
	}
	defer cur.Close(ctx)

	var entries []LogEntry
	knownFields := map[string]bool{
		"_id": true, "game": true, "user_id": true, "eventType": true,
		"timestamp": true, "serverTimestamp": true,
	}
	for cur.Next(ctx) {
		var raw bson.M
		if err := cur.Decode(&raw); err != nil {
			continue
		}
		entry := LogEntry{}
		if game, ok := raw["game"].(string); ok {
			entry.Game = game
		}
		if id, ok := raw["_id"].(primitive.ObjectID); ok {
			entry.ID = id
		}
		if uid, ok := raw["user_id"].(string); ok {
			entry.UserID = uid
		}
		if et, ok := raw["eventType"].(string); ok {
			entry.EventType = et
		}
		if ts, ok := raw["timestamp"].(primitive.DateTime); ok {
			t := ts.Time()
			entry.Timestamp = &t
		}
		if st, ok := raw["serverTimestamp"].(primitive.DateTime); ok {
			entry.ServerTimestamp = st.Time()
		}
		data := make(map[string]interface{})
		for k, v := range raw {
			if !knownFields[k] {
				data[k] = v
			}
		}
		if len(data) > 0 {
			entry.Data = data
		}
		entries = append(entries, entry)
	}

	hasMore := len(entries) > limit
	if hasMore {
		entries = entries[:limit]
	}

	if beforeID != "" && len(entries) > 0 {
		for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
			entries[i], entries[j] = entries[j], entries[i]
		}
	}

	hasPrev := afterID != "" || beforeID != ""
	hasNext := hasMore || beforeID != ""

	if beforeID != "" {
		hasPrev = hasMore
		hasNext = true
	}

	return entries, hasPrev, hasNext, nil
}

// CountLogs returns the total count of logs matching the filter.
func (s *Store) CountLogs(ctx context.Context, game, userID, eventType string) (int64, error) {
	coll := s.db.Collection(logdataCollection)

	filter := bson.M{"game": game}
	if userID == "__empty__" {
		filter["$or"] = []bson.M{
			{"user_id": nil},
			{"user_id": ""},
			{"user_id": bson.M{"$exists": false}},
		}
	} else if userID != "" {
		filter["user_id"] = userID
	}
	if eventType != "" {
		filter["eventType"] = eventType
	}

	return coll.CountDocuments(ctx, filter)
}

// DeleteLog deletes a single log entry.
func (s *Store) DeleteLog(ctx context.Context, game string, id primitive.ObjectID) error {
	coll := s.db.Collection(logdataCollection)
	_, err := coll.DeleteOne(ctx, bson.M{"_id": id, "game": game})
	return err
}

// DeleteUserLogs deletes all logs for a user in a game.
func (s *Store) DeleteUserLogs(ctx context.Context, game, userID string) (int64, error) {
	coll := s.db.Collection(logdataCollection)

	filter := bson.M{"game": game}
	if userID == "__empty__" {
		filter["$or"] = []bson.M{
			{"user_id": nil},
			{"user_id": ""},
			{"user_id": bson.M{"$exists": false}},
		}
	} else {
		filter["user_id"] = userID
	}

	result, err := coll.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

// DeleteGameLogs deletes all logs for a game.
func (s *Store) DeleteGameLogs(ctx context.Context, game string) (int64, error) {
	coll := s.db.Collection(logdataCollection)
	result, err := coll.DeleteMany(ctx, bson.M{"game": game})
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

// ListRecentLogs returns the most recent log entries across all games.
func (s *Store) ListRecentLogs(ctx context.Context, limit int) ([]LogEntry, error) {
	coll := s.db.Collection(logdataCollection)

	// Sort by serverTimestamp only — see note above ListLogs. Same DocumentDB
	// planner limitation; adding _id as a tiebreaker forces a COLLSCAN.
	opts := options.Find().
		SetSort(bson.D{{Key: "serverTimestamp", Value: -1}}).
		SetLimit(int64(limit))

	cur, err := coll.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var entries []LogEntry
	knownFields := map[string]bool{
		"_id": true, "game": true, "user_id": true, "eventType": true,
		"timestamp": true, "serverTimestamp": true,
	}
	for cur.Next(ctx) {
		var raw bson.M
		if err := cur.Decode(&raw); err != nil {
			continue
		}
		entry := LogEntry{}
		if game, ok := raw["game"].(string); ok {
			entry.Game = game
		}
		if id, ok := raw["_id"].(primitive.ObjectID); ok {
			entry.ID = id
		}
		if uid, ok := raw["user_id"].(string); ok {
			entry.UserID = uid
		}
		if et, ok := raw["eventType"].(string); ok {
			entry.EventType = et
		}
		if ts, ok := raw["timestamp"].(primitive.DateTime); ok {
			t := ts.Time()
			entry.Timestamp = &t
		}
		if st, ok := raw["serverTimestamp"].(primitive.DateTime); ok {
			entry.ServerTimestamp = st.Time()
		}
		data := make(map[string]interface{})
		for k, v := range raw {
			if !knownFields[k] {
				data[k] = v
			}
		}
		if len(data) > 0 {
			entry.Data = data
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// CountAllLogs returns the total count of all logs.
func (s *Store) CountAllLogs(ctx context.Context) (int64, error) {
	coll := s.db.Collection(logdataCollection)
	return coll.CountDocuments(ctx, bson.M{})
}
