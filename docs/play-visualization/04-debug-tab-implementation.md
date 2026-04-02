# Debug Tab Implementation

## Tab Addition

Add a fourth tab to the MHS Dashboard tab bar, visible only when `IsAdmin` is true (admin, coordinator, superadmin roles — not leader/teacher).

### Tab Button

In `mhsdashboard_view.gohtml`, after the Analytics tab button:

```html
{{ if .IsAdmin }}
<button id="mhs-tab-btn-debug" type="button"
        class="px-4 py-2 text-sm font-medium mhs-tab-inactive"
        onclick="switchTab('debug')">
  Debug
</button>
{{ end }}
```

### Tab Panel

In `mhsdashboard_grid.gohtml`, after the analytics panel:

```html
{{ if .IsAdmin }}
<div id="mhs-tab-debug" class="hidden h-full overflow-auto">
  <!-- Debug content loaded here -->
</div>
{{ end }}
```

### Legend

```html
{{ if .IsAdmin }}
<div id="mhs-legend-debug" class="hidden items-center gap-4 text-xs">
  <span class="flex items-center gap-1">
    <span class="w-3 h-3 rounded-sm bg-green-500"></span> Start Event
  </span>
  <span class="flex items-center gap-1">
    <span class="w-3 h-3 rounded-sm bg-blue-500"></span> End Event
  </span>
  <span class="flex items-center gap-1">
    <span class="w-3 h-3 rounded-sm bg-amber-500"></span> Anomaly
  </span>
  <span class="flex items-center gap-1">
    <span class="w-3 h-3 rounded-sm bg-red-500"></span> Missing
  </span>
</div>
{{ end }}
```

### JavaScript Update

Update `switchTab()` to include the debug panel, button, and legend in its panel/button/legend maps. The `{{ if .IsAdmin }}` guard on the HTML means the DOM elements won't exist for non-admin users, so the JS gracefully skips null entries (existing pattern handles this).

## User Interaction Flow

1. Admin/coordinator opens MHS Dashboard, selects org → group (existing flow)
2. Clicks "Debug" tab — sees the student list with anomaly indicators
3. Clicks a student name — loads their debug detail view via HTMX
4. Can filter by unit, expand/collapse event categories
5. Can switch to position view (Phase 2)

## Debug Tab Content Structure

The debug tab has two views:

### Student List View (default)

Shows all students in the group with anomaly counts:

```
┌─────────────────────────────────────────────────────────┐
│ Student               Pencils  Empty  Duplicates  Total │
│ ─────────────────────────────────────────────────────── │
│ ▸ Cannon, Aaron         2       1        0          3   │
│ ▸ Carroll, Cheyenne     0       3        1          4   │
│ ▸ Coulter, Gavyn        1       0        0          1   │
│ ▸ Dowers, Abcah         0       0        0          0   │
│   ...                                                   │
└─────────────────────────────────────────────────────────┘
```

- Pencils = progress points with "active" status (started, never finished)
- Empty = progress points with no grade at all (expected start never seen)
- Duplicates = events that fired multiple times within a short window
- Students with 0 anomalies shown in muted style
- Click a student row to load their detail view

### Student Detail View (loaded via HTMX)

Shows timeline, anomaly summary, and unit filter for a single student.

```
┌─────────────────────────────────────────────────────────┐
│ ← Back to list          Dowers, Abcah                   │
│                                                         │
│ Unit: [All ▾] [1] [2] [3] [4] [5]                     │
│                                                         │
│ ┌─── Anomaly Summary ──────────────────────────────┐   │
│ │ U3P2: active — start seen 16:24:45, no end event │   │
│ │ U4P6: empty — start event never received          │   │
│ │ U4: 3× EndOfUnit in 18 seconds (game bug)        │   │
│ └──────────────────────────────────────────────────┘   │
│                                                         │
│ ┌─── Timeline ─────────────────────────────────────┐   │
│ │ [Event list — see 05-timeline-view.md]           │   │
│ └──────────────────────────────────────────────────┘   │
│                                                         │
│ ┌─── Position (Phase 2) ──────────────────────────┐   │
│ │ [Canvas plot — see 07-position-visualization.md] │   │
│ └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

## New Endpoints

Add to `mhsdashboard/routes.go`:

```go
// Debug tab endpoints (admin/coordinator only)
r.Get("/debug/students", h.ServeDebugStudents)      // Student anomaly list
r.Get("/debug/student/{userID}", h.ServeDebugDetail) // Single student detail
r.Get("/debug/timeline/{userID}", h.ServeDebugTimeline) // Timeline events (HTMX partial)
r.Get("/debug/positions/{userID}", h.ServeDebugPositions) // Position data as JSON (Phase 2)
```

All debug endpoints check `IsAdmin` and return 403 for non-admin roles.

### ServeDebugStudents

1. Load group members (existing pattern)
2. Load grades from mhsgrader for all members
3. Load grading rules config
4. For each student, compute anomaly counts:
   - Count "active" grades (pencils)
   - Count missing grades (points where no grade exists but prior points have grades)
   - Optionally sample log data to check for duplicate end-of-unit events
5. Render student list with anomaly columns

### ServeDebugDetail

1. Load student info and grades
2. Load log events from stratalog for the student (full set or filtered by unit)
3. Annotate events with grading rule waypoints
4. Run anomaly detection
5. Render detail view with anomaly summary + timeline

### ServeDebugTimeline

HTMX endpoint for timeline content, supports unit filtering:
- `?unit=3` — filter to unit 3 scenes only
- `?show_position=true` — include position events (default: hidden)

### ServeDebugPositions

Returns JSON array of position events for a unit, used by the canvas renderer (Phase 2):
```json
{
  "positions": [
    {"x": 12.5, "y": 0, "z": -34.2, "time": "2026-03-24T16:26:32Z", "scene": "Unit 1 Dev"},
    ...
  ],
  "bounds": {"minX": -50, "maxX": 80, "minZ": -60, "maxZ": 40}
}
```

## Handler Changes

### Handler struct

Add fields:
```go
type Handler struct {
    DB              *mongo.Database  // stratahub
    GradesDB        *mongo.Database  // mhsgrader
    LogDB           *mongo.Database  // stratalog (NEW)
    LogStore        *logdata.Store   // wraps LogDB.Collection("logdata")
    // ... existing fields
}
```

### NewHandler

Accept the new DB parameter:
```go
func NewHandler(db, gradesDB, logDB *mongo.Database, ...) *Handler {
    h := &Handler{
        DB:       db,
        GradesDB: gradesDB,
        LogDB:    logDB,
        // ... existing stores
    }
    if logDB != nil {
        h.LogStore = logdata.New(logDB)
    }
    return h
}
```

The `logDB != nil` check allows graceful degradation if the stratalog database isn't configured (e.g., in dev environments without it). The debug tab simply won't show data.

## View Models

```go
// DebugStudentRow represents a student in the debug list
type DebugStudentRow struct {
    ID            string
    Name          string
    LoginID       string
    PencilCount   int  // active grades (started, never finished)
    EmptyCount    int  // missing grades (expected but not present)
    DuplicateCount int // duplicate end-of-unit or trigger events
    TotalAnomalies int
}

// DebugDetailData is the view model for the student detail view
type DebugDetailData struct {
    StudentID   string
    StudentName string
    LoginID     string
    SelectedUnit string // "" for all, "unit1"-"unit5" for filtered
    Anomalies   []DebugAnomaly
    Timeline    []TimelineEntry
    Grades      map[string][]ProgressGradeItem // from mhsgrader
}

// DebugAnomaly represents a detected issue
type DebugAnomaly struct {
    Type        string // "pencil", "empty", "duplicate", "gap"
    Severity    string // "error", "warning", "info"
    PointID     string // e.g., "u3p2" (empty for non-point anomalies)
    Unit        string
    Description string
    Timestamp   string // when the issue occurred
}
```

## Template Files

New template files in `mhsdashboard/templates/`:

- `mhsdashboard_debug_students.gohtml` — Student anomaly list (rendered into debug tab panel)
- `mhsdashboard_debug_detail.gohtml` — Single student detail with anomaly summary + timeline
- `mhsdashboard_debug_timeline.gohtml` — Timeline partial (HTMX refreshable for unit filtering)

These are registered in the existing `templates.go` `init()` function alongside the current templates.
