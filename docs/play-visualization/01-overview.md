# Play Visualization & Debug Tools — Overview

## Problem Statement

The MHS Dashboard shows progress point grades as a color-coded grid. Two recurring issues need investigation tools:

1. **Pencil icons (active/in-progress)**: The grader received a start event for a progress point but never received the corresponding end event. The grade is stuck at "active" status. Causes may include: student quit mid-activity, game bug that didn't fire the end event, eventKey mismatch between game version and grading rule, or network/timing issues.

2. **Empty cells (no grade)**: The grader never received a start event for the progress point. Causes may include: student skipped the activity, game didn't fire the start event, eventKey format changed between versions, or student played through without triggering the specific dialogue node.

Both require correlating raw log events (stored in stratalog's `logdata` collection) with grading rules (defined in mhsgrading, implemented in mhsgrader) and grading results (stored in mhsgrader's `progress_point_grades` collection).

## Decision: Build in MHS Dashboard

The debug tools will be built as a new tab in the MHS Dashboard within stratahub, rather than in stratalog, for these reasons:

- Dashboard already has organizational context (org, group, student) — no need to rebuild navigation
- Teachers/admins see issues (pencils, empty cells) in the dashboard and want to drill down from there
- Dashboard already has access to mhsgrader's grades database
- Adding stratalog DB access to stratahub follows the same pattern as the existing mhsgrader DB connection

The new tab will be **visible only to admin and coordinator roles** (not teachers/leaders). Teacher-facing tools may be considered later based on what we learn.

## Scope

### Phase 1: Event Timeline with Progress Point Waypoints
- New "Debug" tab in MHS Dashboard (admin/coordinator only)
- Connect stratahub to stratalog's database (read-only)
- Embed grading rule eventKeys as JSON config
- Per-student timeline view showing all events with progress point start/end markers
- Anomaly detection and summary (stuck actives, missing starts, duplicate events)
- Filter by unit, collapse position events

### Phase 2: Position Visualization
- 2D canvas plot of PlayerPositionEvent data per unit/scene
- Derive scene bounds from actual position data
- Show movement path with timestamps
- Useful for understanding where students get stuck or wander

### Phase 3: Teacher-Facing Tools (Future)
- If Phase 1 & 2 reveal patterns useful to teachers, create simplified versions
- Would appear in the existing Progress tab or as a separate teacher-visible feature

## Architecture

```
┌─────────────────────────────────────────────────┐
│  MHS Dashboard (stratahub)                      │
│                                                 │
│  Tabs: Progress | Devices | Analytics | Debug   │
│                                                 │
│  Debug tab (admin/coordinator only):            │
│  ┌─────────────────────────────────────────┐    │
│  │ Anomaly Summary                         │    │
│  │ Timeline View (events + waypoints)      │    │
│  │ Position Plot (Phase 2)                 │    │
│  └─────────────────────────────────────────┘    │
│                                                 │
│  Data sources:                                  │
│  ├── stratahub DB (users, groups, memberships)  │
│  ├── mhsgrader DB (progress_point_grades)       │
│  └── stratalog DB (logdata) ← NEW              │
│                                                 │
│  Embedded config:                               │
│  └── mhs_grading_rules.json (eventKeys)         │
└─────────────────────────────────────────────────┘
```

## Related Documents

- [02-database-connection.md](02-database-connection.md) — Adding stratalog DB to stratahub
- [03-grading-rules-json.md](03-grading-rules-json.md) — Embedded grading rules config
- [04-debug-tab-implementation.md](04-debug-tab-implementation.md) — Tab UI, handler, templates
- [05-timeline-view.md](05-timeline-view.md) — Event timeline with waypoint overlay
- [06-anomaly-detection.md](06-anomaly-detection.md) — Detecting and reporting issues
- [07-position-visualization.md](07-position-visualization.md) — Phase 2 movement plotting
- [08-files-to-modify.md](08-files-to-modify.md) — Complete file change list
