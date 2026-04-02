# Timeline View

## Purpose

The timeline view shows a student's log events in chronological order with grading rule waypoints overlaid. It allows an admin to see exactly what happened during gameplay and understand why a progress point has a pencil (stuck active), empty cell (missing grade), or unexpected result.

## Event Stream Processing

### Loading Events

1. Query stratalog `logdata` for the student: `{game: "mhs", playerId: loginID}`, sorted by `_id` ascending (chronological)
2. If unit filter applied, filter by `sceneName` matching known scene names for that unit
3. Load grading rules JSON config (cached)
4. Build an `eventKey → []GradingRule` index from the config

### Scene-to-Unit Mapping

The log entries have `sceneName` values that map to units. Known patterns from the data:

```json
{
  "unit1": ["Unit 1 Dev", "Unit 1 Prod"],
  "unit2": ["Unit 2 Prod (Refactor)", "Unit 2 Dev"],
  "unit3": ["Unit 3 Dev", "Unit 3 Dungeon Dev"],
  "unit4": ["Unit 4 Dev", "Unit 4 Dev - Dungeon", "Unit 4 Dev - Anderson Base"],
  "unit5": ["Unit 5 Dev", "Unit 5 Dev - Dungeon"]
}
```

This mapping should be part of the embedded JSON config so it can be updated as scene names change between game versions. Add to `mhs_grading_rules.json`:

```json
{
  "scene_to_unit": {
    "Unit 1 Dev": "unit1",
    "Unit 1 Prod": "unit1",
    "Unit 2 Prod (Refactor)": "unit2",
    "Unit 2 Dev": "unit2",
    "Unit 3 Dev": "unit3",
    "Unit 3 Dungeon Dev": "unit3",
    "Unit 4 Dev": "unit4",
    "Unit 4 Dev - Dungeon": "unit4",
    "Unit 4 Dev - Anderson Base": "unit4",
    "Unit 5 Dev": "unit5",
    "Unit 5 Dev - Dungeon": "unit5"
  }
}
```

### Annotating Events

For each log entry, check if its `eventKey` appears in the grading rules index:

```go
type TimelineEntry struct {
    ID              string    // ObjectID hex
    EventType       string
    EventKey        string
    SceneName       string
    ServerTimestamp  time.Time
    Data            map[string]interface{}
    Unit            string    // derived from sceneName

    // Annotations (populated from grading rules)
    IsStartEvent    bool      // this event starts a progress point
    IsEndEvent      bool      // this event ends/triggers a progress point
    PointIDs        []string  // which progress point(s) this relates to
    Annotation      string   // human-readable label (e.g., "U3P2 Start: Pollution Solution Part I")

    // Classification
    Category        string    // "waypoint", "dialogue", "quest", "position", "gameplay", "system"
    IsAnomaly       bool
    AnomalyType     string    // "duplicate", "orphan", etc.
}
```

### Event Categories

Events are classified for display filtering:

| Category | Event Types | Default Visibility |
|----------|------------|-------------------|
| `waypoint` | Events matching start/end keys in grading rules | Always shown |
| `quest` | `questEvent` (questActiveEvent, questFinishEvent) | Shown |
| `dialogue` | `DialogueEvent` (DialogueNodeEvent, DialogueFinishEvent) | Shown |
| `argumentation` | `argumentationEvent`, `argumentationNodeEvent`, `argumentationToolEvent` | Shown |
| `gameplay` | `Topographic Map Event`, `TopographicMapEvent`, `WaterChamberEvent`, `soilMachine`, `TerasGardenBox`, `Soil Key Puzzle` | Shown |
| `position` | `PlayerPositionEvent` | Hidden by default |
| `system` | `EndOfUnit`, other | Shown |

Users can toggle category visibility with checkboxes in the timeline header.

## Timeline Rendering

### Visual Layout

```
┌──────────────────────────────────────────────────────────────────────┐
│ Filter: [✓ Waypoints] [✓ Quests] [✓ Dialogue] [✓ Gameplay]        │
│         [ ] Position  [✓ System]                                     │
│                                                                      │
│ ═══ UNIT 1 ══════════════════════════════════════════════════════    │
│                                                                      │
│ ▌START▐ u1p1: Getting Your Space Legs                               │
│ 16:24:17  questActiveEvent:28        Unit 1 Dev                     │
│ 16:24:28  DialogueNodeEvent          conversationId:31 nodeId:5     │
│ 16:25:02  DialogueNodeEvent          conversationId:31 nodeId:8     │
│ ...                                                                  │
│ 16:26:28  DialogueNodeEvent:31:29    Unit 1 Dev                     │
│ ▌END▐ u1p1 → Grade: passed                                          │
│                                                                      │
│ ▌START▐ u1p2: Info and Intros                                       │
│ 16:26:28  DialogueNodeEvent:31:29    Unit 1 Dev                     │
│ ...                                                                  │
│ 16:39:56  DialogueNodeEvent:30:98    Unit 1 Dev                     │
│ ▌END▐ u1p2 → Grade: passed                                          │
│                                                                      │
│ ▌START▐ u1p3: Defend the Expedition                                 │
│ 16:39:56  DialogueNodeEvent:30:98    Unit 1 Dev                     │
│ 16:40:12  DialogueNodeEvent:70:25    ⚠ Yellow key (wrong answer)   │
│ ...                                                                  │
│ 16:44:06  questActiveEvent:34        Unit 1 Dev                     │
│ ▌END▐ u1p3 → Grade: flagged (1 wrong answer)                       │
│                                                                      │
│ ═══ UNIT 2 ══════════════════════════════════════════════════════    │
│ ...                                                                  │
└──────────────────────────────────────────────────────────────────────┘
```

### Waypoint Markers

- **Start markers**: Green left border bar with "START u1p1: Activity Name"
- **End markers**: Blue left border bar with "END u1p1 → Grade: passed/flagged/active"
- **Grade display**: Pulled from mhsgrader's `progress_point_grades` for the student
- **Missing end**: If start exists but no end event found, show a red "MISSING END" marker at the point where the end was expected

### Event Row Rendering

Each event row shows:
```
[timestamp]  [eventKey or eventType]  [scene]  [data summary]  [annotations]
```

- Timestamp: Formatted in the organization's timezone (existing pattern)
- EventKey: If present, shown prominently. If absent, show eventType
- Data summary: Compact representation of the `data` map (e.g., "conversationId:109 nodeId:38")
- Annotations: Icons/badges for yellow keys, success keys, bonus events

### Color Coding

- Start events: green background highlight
- End events: blue background highlight
- Yellow/negative keys: amber text or left border
- Positive/success keys: green text
- Duplicate events: red left border with "DUPLICATE" badge
- Position events: muted gray (when shown)

## Event Count and Pagination

A student may have thousands of events. Strategies:

1. **Unit filtering** is the primary tool — most debugging is unit-specific
2. **Position events hidden by default** removes ~20% of entries
3. **Lazy loading**: Initially render first 200 events, "Load more" button or scroll-triggered HTMX for the rest
4. **Waypoint jumping**: Quick links to jump to each progress point start/end in the timeline

## Data Formatting Helpers

### Data Map Summary

Convert the `data` map to a compact string:
- `{"conversationId": 109, "dialogueEventType": "DialogueNodeEvent", "nodeId": 38}` → `"conversation:109 node:38"`
- `{"Unit": "5"}` → `"Unit 5"`
- `{"questEventType": "questFinishEvent", "questName": "Power Play", "questSuccessOrFailure": "Succeeded"}` → `"Quest: Power Play → Succeeded"`
- `{"position": {"x": 12.5, "y": 0, "z": -34.2}}` → `"pos(12.5, -34.2)"`

### Duration Between Events

Show duration gaps between consecutive events when > 30 seconds:
```
16:39:56  DialogueNodeEvent:30:98    Unit 1 Dev
              ── 4m 10s ──
16:44:06  questActiveEvent:34        Unit 1 Dev
```

Large gaps (> 5 minutes) may indicate the student left and returned, which is relevant for understanding "active" grades.
