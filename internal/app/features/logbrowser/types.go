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

	// Player filter
	Players        []PlayerRowVM
	SelectedPlayer string
	PlayerSearch   string
	PlayerPage     int
	PlayerTotal    int64
	PlayerHasPrev  bool
	PlayerHasNext  bool
	PlayerPrevPage int
	PlayerNextPage int
	PlayerRangeStart int
	PlayerRangeEnd   int

	// Event type filter
	EventTypes        []string
	SelectedEventType string

	// Logs display
	Logs         []LogRowVM
	LogTotal     int64
	LogLimit     int
	Limit        int // Alias for LogLimit, used by players_content template
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
	ID          string
	Game        string
	PlayerID    string
	EventType   string
	Timestamp   *time.Time
	ServerTimestamp time.Time
	Data        string // JSON-formatted data
}

// PlayerRowVM represents a player with log count.
type PlayerRowVM struct {
	PlayerID string
	LogCount int64
}

// PlayersPartialVM is the view model for the players partial.
type PlayersPartialVM struct {
	SelectedGame     string
	SelectedPlayer   string
	PlayerSearch     string
	Players          []PlayerRowVM
	PlayerTotal      int64
	PlayerPage       int
	PlayerHasPrev    bool
	PlayerHasNext    bool
	PlayerRangeStart int
	PlayerRangeEnd   int
	PlayerPrevPage   int
	PlayerNextPage   int
	Limit            int
}

// LogsPartialVM is the view model for the logs partial.
type LogsPartialVM struct {
	viewdata.BaseVM
	SelectedGame      string
	SelectedPlayer    string
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
