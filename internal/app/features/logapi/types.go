// Package logapi provides the log submission API endpoints for game event logging.
package logapi

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// LogEntry represents a single log entry in the database.
// user_id is the 24-char hex string of stratahub.users._id.
type LogEntry struct {
	ID              primitive.ObjectID     `bson:"_id,omitempty" json:"id,omitempty"`
	Game            string                 `bson:"game" json:"game"`
	UserID          string                 `bson:"user_id,omitempty" json:"user_id,omitempty"`
	EventType       string                 `bson:"eventType,omitempty" json:"eventType,omitempty"`
	Timestamp       *time.Time             `bson:"timestamp,omitempty" json:"timestamp,omitempty"`
	ServerTimestamp time.Time              `bson:"serverTimestamp" json:"serverTimestamp"`
	Data            map[string]interface{} `bson:"data,omitempty" json:"data,omitempty"`
}

// SingleLogRequest represents a single log entry submission.
type SingleLogRequest struct {
	Game      string                 `json:"game"`
	UserID    string                 `json:"user_id,omitempty"`
	EventType string                 `json:"eventType,omitempty"`
	Timestamp *time.Time             `json:"timestamp,omitempty"`
	Data      map[string]interface{} `json:"-"`
}

// BatchLogRequest represents a batch log entry submission.
type BatchLogRequest struct {
	Game    string          `json:"game"`
	Entries []BatchLogEntry `json:"entries"`
}

// BatchLogEntry represents a single entry within a batch submission.
type BatchLogEntry struct {
	UserID    string                 `json:"user_id,omitempty"`
	EventType string                 `json:"eventType,omitempty"`
	Timestamp *time.Time             `json:"timestamp,omitempty"`
	Data      map[string]interface{} `json:"-"`
}

// LogResponse represents the response for a successful log submission.
type LogResponse struct {
	Status     string `json:"status"`
	ReceivedAt string `json:"received_at"`
}

// LogQueryParams represents query parameters for listing logs.
type LogQueryParams struct {
	Game      string     `json:"game"`
	UserID    string     `json:"user_id,omitempty"`
	EventType string     `json:"eventType,omitempty"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
}

// LogListResponse represents the response for listing logs.
type LogListResponse struct {
	Entries []LogEntry `json:"entries"`
	Total   int64      `json:"total"`
	Limit   int        `json:"limit"`
	Offset  int        `json:"offset"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}
