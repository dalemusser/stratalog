package logbrowser

import (
	"time"

	"github.com/dalemusser/stratalog/internal/app/system/timezones"
	"github.com/dalemusser/stratalog/internal/app/system/viewdata"
)

// ListVM is the view model for the main log browser page.
type ListVM struct {
	viewdata.BaseVM

	// Timezone data
	TimezoneGroups []timezones.ZoneGroup

	// Game selection
	Games        []string
	SelectedGame string

	// Total logs across all games
	TotalAllLogs int64

	// User filter
	Users          []UserRowVM
	SelectedUser   string
	UserSearch     string
	UserPage       int
	UserTotal      int64
	UserHasPrev    bool
	UserHasNext    bool
	UserPrevPage   int
	UserNextPage   int
	UserRangeStart int
	UserRangeEnd   int

	// Event type filter
	EventTypes        []string
	SelectedEventType string

	// Logs display
	Logs         []LogRowVM
	LogTotal     int64
	LogLimit     int
	Limit        int // Alias for LogLimit, used by users_content template
	DefaultLimit int
	HasPrev      bool
	HasNext    bool
	PrevCursor string
	NextCursor string

	// API configuration
	APIKey string
}

// LogRowVM represents a single log entry in the browser.
type LogRowVM struct {
	ID              string
	Game            string
	UserID          string
	EventType       string
	Timestamp       *time.Time
	ServerTimestamp time.Time
	Data            string // JSON-formatted data
}

// UserRowVM represents a user with log count.
type UserRowVM struct {
	UserID   string
	LogCount int64
}

// UsersPartialVM is the view model for the users partial.
type UsersPartialVM struct {
	SelectedGame   string
	SelectedUser   string
	UserSearch     string
	Users          []UserRowVM
	UserTotal      int64
	UserPage       int
	UserHasPrev    bool
	UserHasNext    bool
	UserRangeStart int
	UserRangeEnd   int
	UserPrevPage   int
	UserNextPage   int
	Limit          int
}

// LogsPartialVM is the view model for the logs partial.
type LogsPartialVM struct {
	viewdata.BaseVM
	SelectedGame      string
	SelectedUser      string
	SelectedEventType string
	Logs              []LogRowVM
	Total             int64
	LogTotal          int64 // Alias for Total, used by logs_content template
	Limit             int
	HasPrev           bool
	HasNext           bool
	PrevCursor        string
	NextCursor        string
}

// GamePickerVM is the view model for the game picker modal.
type GamePickerVM struct {
	Games      []GamePickerItem
	SelectedID string
	Query      string
}

// GamePickerItem represents a game in the picker.
type GamePickerItem struct {
	Name     string
	Selected bool
}

// EventTypePickerVM is the view model for event type picker.
type EventTypePickerVM struct {
	EventTypes []EventTypeItem
	SelectedID string
	Query      string
}

// EventTypeItem represents an event type in the picker.
type EventTypeItem struct {
	Name     string
	Count    int64
	Selected bool
}

// PlaygroundVM is the view model for the API playground page.
type PlaygroundVM struct {
	viewdata.BaseVM
	APIKey string
}

// DocsVM is the view model for the API documentation page.
type DocsVM struct {
	viewdata.BaseVM
	MaxBatchSize int
}

// RecentLogsVM is the view model for the recent logs page.
type RecentLogsVM struct {
	viewdata.BaseVM
	TimezoneGroups []timezones.ZoneGroup
	Logs           []LogRowVM
	Total          int64
	Limit          int
	LimitOptions   []int
}
