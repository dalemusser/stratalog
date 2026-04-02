# Files to Modify

Complete list of files to create or modify for Phase 1 (Debug Tab + Timeline + Anomaly Detection).

## New Files

### Stratahub

| File | Purpose |
|------|---------|
| `internal/app/store/logdata/store.go` | Read-only store for stratalog's `logdata` collection |
| `internal/app/resources/mhs_grading_rules.json` | Embedded grading rule eventKeys and scene mappings |
| `internal/app/features/mhsdashboard/grading_rules.go` | Loader for grading rules JSON (sync.Once pattern) |
| `internal/app/features/mhsdashboard/debug.go` | Debug tab handlers (ServeDebugStudents, ServeDebugDetail, ServeDebugTimeline) |
| `internal/app/features/mhsdashboard/anomaly.go` | Anomaly detection logic |
| `internal/app/features/mhsdashboard/timeline.go` | Timeline building and event annotation |
| `internal/app/features/mhsdashboard/templates/mhsdashboard_debug_students.gohtml` | Student anomaly list template |
| `internal/app/features/mhsdashboard/templates/mhsdashboard_debug_detail.gohtml` | Student detail view template |
| `internal/app/features/mhsdashboard/templates/mhsdashboard_debug_timeline.gohtml` | Timeline partial template |

### Phase 2 additions

| File | Purpose |
|------|---------|
| `internal/app/features/mhsdashboard/positions.go` | Position data handler (JSON endpoint) |
| `internal/app/features/mhsdashboard/templates/mhsdashboard_debug_positions.gohtml` | Position canvas template partial |

## Modified Files

### Bootstrap (database connection)

| File | Change |
|------|--------|
| `internal/app/bootstrap/appconfig.go` | Add `StratalogDatabase string` field |
| `internal/app/bootstrap/config.go` | Add `stratalog_database` config key, load in `LoadConfig()` |
| `internal/app/bootstrap/db.go` | Add `client.Database(appCfg.StratalogDatabase)` in `ConnectDB()` |
| `internal/app/bootstrap/dbdeps.go` | Add `StratalogDatabase *mongo.Database` field |
| `internal/app/bootstrap/routes.go` | Pass `deps.StratalogDatabase` to mhsdashboard handler |

### Resources

| File | Change |
|------|--------|
| `internal/app/resources/resources.go` | Add `mhs_grading_rules.json` to embed directive |

### MHS Dashboard feature

| File | Change |
|------|--------|
| `internal/app/features/mhsdashboard/handler.go` | Add `LogDB`, `LogStore` fields; update `NewHandler` signature |
| `internal/app/features/mhsdashboard/routes.go` | Add debug endpoints |
| `internal/app/features/mhsdashboard/types.go` | Add debug view model types (`DebugStudentRow`, `DebugDetailData`, `DebugAnomaly`, `TimelineEntry`, `GradingRulesConfig`, `GradingRule`) |
| `internal/app/features/mhsdashboard/templates/mhsdashboard_view.gohtml` | Add Debug tab button (admin-gated), legend, switchTab() update, debug CSS |
| `internal/app/features/mhsdashboard/templates/mhsdashboard_grid.gohtml` | Add Debug panel div (admin-gated) |
| `internal/app/features/mhsdashboard/templates.go` | Register new template files |

### Configuration

| File | Change |
|------|--------|
| `stratahub_update/config.toml` | Add `stratalog_database = "stratalog"` |

## Dependency Graph

```
config.toml
  → appconfig.go (StratalogDatabase)
    → config.go (stratalog_database key)
      → db.go (client.Database)
        → dbdeps.go (StratalogDatabase field)
          → routes.go (pass to handler)
            → handler.go (LogDB, LogStore)
              → debug.go (uses LogStore + GradesDB)
                → anomaly.go (analyzes grades + logs)
                → timeline.go (builds annotated timeline)
                  → templates (render views)

mhs_grading_rules.json
  → resources.go (embed)
    → grading_rules.go (load + cache)
      → debug.go (annotate events)
      → anomaly.go (detect issues)
```

## Implementation Order

### Step 1: Database plumbing
1. `appconfig.go` — add field
2. `config.go` — add key and loader
3. `db.go` — add database reference
4. `dbdeps.go` — add to struct
5. `config.toml` — add production config
6. Build and verify no errors

### Step 2: Log data store
1. Create `store/logdata/store.go` with basic query methods
2. Wire into handler

### Step 3: Grading rules JSON
1. Create `mhs_grading_rules.json` with all 24 rules
2. Update `resources.go` embed
3. Create `grading_rules.go` loader
4. Build and verify loading

### Step 4: Types and models
1. Add all debug-related types to `types.go`

### Step 5: Tab UI shell
1. Add tab button, panel, legend to templates
2. Update `switchTab()` JavaScript
3. Add debug routes (returning placeholder content)
4. Build and verify tab appears for admin

### Step 6: Student list with anomaly counts
1. Implement `ServeDebugStudents` with grade-only anomaly counts
2. Create student list template
3. Test with real data

### Step 7: Anomaly detection
1. Implement `anomaly.go` detection logic
2. Integrate with student detail view

### Step 8: Timeline view
1. Implement `timeline.go` event annotation
2. Create timeline template
3. Add unit filtering
4. Add event category toggles

### Step 9: Position visualization (Phase 2)
1. Implement `positions.go` JSON endpoint
2. Create canvas template and JavaScript
3. Add scene selector and waypoint overlay

## Build Verification

After each step, run:
```bash
cd stratahub && go build ./...
```

After Step 5, deploy to staging/dev to verify the tab appears correctly and doesn't affect existing functionality.
