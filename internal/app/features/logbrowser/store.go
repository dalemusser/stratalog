package logbrowser

import (
	"context"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// LogEntry represents a log entry in the database.
type LogEntry struct {
	ID          primitive.ObjectID     `bson:"_id,omitempty"`
	Game        string                 `bson:"game"`
	PlayerID    string                 `bson:"playerId,omitempty"`
	EventType   string                 `bson:"eventType,omitempty"`
	Timestamp   *time.Time             `bson:"timestamp,omitempty"`
	ServerTimestamp time.Time              `bson:"serverTimestamp"`
	Data        map[string]interface{} `bson:"data,omitempty"`
}

// UserWithCount represents a player with their log count.
type UserWithCount struct {
	PlayerID string
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
	// Get distinct game values from the unified logdata collection
	coll := s.db.Collection(logdataCollection)
	values, err := coll.Distinct(ctx, "game", bson.M{})
	if err != nil {
		return nil, err
	}

	// Convert to string slice and sort
	games := make([]string, 0, len(values))
	for _, v := range values {
		if game, ok := v.(string); ok && game != "" {
			games = append(games, game)
		}
	}

	// Sort alphabetically
	sort.Strings(games)
	return games, nil
}

// ListPlayersWithCounts returns players with their log counts for a game.
func (s *Store) ListPlayersWithCounts(ctx context.Context, game, search string, page, limit int) ([]UserWithCount, int64, error) {
	coll := s.db.Collection(logdataCollection)

	// Build match stage for game and optional search
	match := bson.M{"game": game}
	if search != "" {
		match["playerId"] = bson.M{"$regex": primitive.Regex{Pattern: search, Options: "i"}}
	}

	// Aggregation pipeline to get unique players with counts
	// Use $ifNull to handle null/missing, giving us "" or the actual value
	// This naturally groups null, missing, and "" together since they all become ""
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"$ifNull": bson.A{"$playerId", ""}},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}, {Key: "_id", Value: 1}}}},
	}

	// Get total count first
	countPipeline := append(pipeline, bson.D{{Key: "$count", Value: "total"}})
	countCur, err := coll.Aggregate(ctx, countPipeline)
	if err != nil {
		return nil, 0, err
	}
	defer countCur.Close(ctx)

	var total int64
	if countCur.Next(ctx) {
		var result struct {
			Total int64 `bson:"total"`
		}
		if err := countCur.Decode(&result); err == nil {
			total = result.Total
		}
	}

	// Add pagination
	skip := (page - 1) * limit
	paginatedPipeline := append(pipeline,
		bson.D{{Key: "$skip", Value: skip}},
		bson.D{{Key: "$limit", Value: limit}},
	)

	cur, err := coll.Aggregate(ctx, paginatedPipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	var results []UserWithCount
	for cur.Next(ctx) {
		var doc struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cur.Decode(&doc); err != nil {
			continue
		}
		results = append(results, UserWithCount{
			PlayerID: doc.ID,
			LogCount: doc.Count,
		})
	}

	return results, total, nil
}

// ListEventTypes returns all event types for a game with counts.
func (s *Store) ListEventTypes(ctx context.Context, game string) ([]EventTypeItem, error) {
	coll := s.db.Collection(logdataCollection)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"game": game}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$eventType",
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}, {Key: "_id", Value: 1}}}},
	}

	cur, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var results []EventTypeItem
	for cur.Next(ctx) {
		var doc struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cur.Decode(&doc); err != nil {
			continue
		}
		if doc.ID != "" {
			results = append(results, EventTypeItem{
				Name:  doc.ID,
				Count: doc.Count,
			})
		}
	}

	return results, nil
}

// ListLogs returns logs with cursor-based pagination.
func (s *Store) ListLogs(ctx context.Context, game, playerID, eventType string, limit int, afterID, beforeID string) ([]LogEntry, bool, bool, error) {
	coll := s.db.Collection(logdataCollection)

	filter := bson.M{"game": game}
	if playerID == "__empty__" {
		// Filter for logs with no playerId (null, empty string, or missing)
		filter["$or"] = []bson.M{
			{"playerId": nil},
			{"playerId": ""},
			{"playerId": bson.M{"$exists": false}},
		}
	} else if playerID != "" {
		filter["playerId"] = playerID
	}
	if eventType != "" {
		filter["eventType"] = eventType
	}

	// Handle cursor-based pagination
	sortDir := -1 // Descending by default (newest first)
	if beforeID != "" {
		if oid, err := primitive.ObjectIDFromHex(beforeID); err == nil {
			filter["_id"] = bson.M{"$gt": oid}
			sortDir = 1 // Ascending to get items before cursor
		}
	} else if afterID != "" {
		if oid, err := primitive.ObjectIDFromHex(afterID); err == nil {
			filter["_id"] = bson.M{"$lt": oid}
		}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "serverTimestamp", Value: sortDir}, {Key: "_id", Value: sortDir}}).
		SetLimit(int64(limit + 1)) // Fetch one extra to detect if there are more

	cur, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, false, false, err
	}
	defer cur.Close(ctx)

	var entries []LogEntry
	// Known fields that should not be included in Data
	knownFields := map[string]bool{
		"_id": true, "game": true, "playerId": true, "eventType": true,
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
		if pid, ok := raw["playerId"].(string); ok {
			entry.PlayerID = pid
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
		// Collect remaining fields into Data
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

	// Determine pagination state
	hasMore := len(entries) > limit
	if hasMore {
		entries = entries[:limit]
	}

	// If we were paginating backwards, reverse the results
	if beforeID != "" && len(entries) > 0 {
		for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
			entries[i], entries[j] = entries[j], entries[i]
		}
	}

	// Check if there are previous items
	hasPrev := afterID != "" || beforeID != ""
	hasNext := hasMore || beforeID != ""

	if beforeID != "" {
		hasPrev = hasMore
		hasNext = true
	}

	return entries, hasPrev, hasNext, nil
}

// CountLogs returns the total count of logs matching the filter.
func (s *Store) CountLogs(ctx context.Context, game, playerID, eventType string) (int64, error) {
	coll := s.db.Collection(logdataCollection)

	filter := bson.M{"game": game}
	if playerID == "__empty__" {
		// Filter for logs with no playerId (null, empty string, or missing)
		filter["$or"] = []bson.M{
			{"playerId": nil},
			{"playerId": ""},
			{"playerId": bson.M{"$exists": false}},
		}
	} else if playerID != "" {
		filter["playerId"] = playerID
	}
	if eventType != "" {
		filter["eventType"] = eventType
	}

	return coll.CountDocuments(ctx, filter)
}

// DeleteLog deletes a single log entry.
func (s *Store) DeleteLog(ctx context.Context, game string, id primitive.ObjectID) error {
	coll := s.db.Collection(logdataCollection)
	// Filter by both _id and game for safety
	_, err := coll.DeleteOne(ctx, bson.M{"_id": id, "game": game})
	return err
}

// DeletePlayerLogs deletes all logs for a player in a game.
func (s *Store) DeletePlayerLogs(ctx context.Context, game, playerID string) (int64, error) {
	coll := s.db.Collection(logdataCollection)

	filter := bson.M{"game": game}
	if playerID == "__empty__" {
		// Delete logs with no playerId (null, empty string, or missing)
		filter["$or"] = []bson.M{
			{"playerId": nil},
			{"playerId": ""},
			{"playerId": bson.M{"$exists": false}},
		}
	} else {
		filter["playerId"] = playerID
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

	opts := options.Find().
		SetSort(bson.D{{Key: "serverTimestamp", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(int64(limit))

	cur, err := coll.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var entries []LogEntry
	// Known fields that should not be included in Data
	knownFields := map[string]bool{
		"_id": true, "game": true, "playerId": true, "eventType": true,
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
		if pid, ok := raw["playerId"].(string); ok {
			entry.PlayerID = pid
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
		// Collect remaining fields into Data
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
