# Adding Stratalog Database to Stratahub

## Current State

Stratahub connects to two MongoDB databases on the same DocumentDB cluster:
- **stratahub** (primary) — users, groups, organizations, settings, etc.
- **mhsgrader** — `progress_point_grades` collection (read-only from dashboard)

Both databases are accessed through a single MongoDB client. The mhsgrader database was added by:
1. Adding a config key (`mhsgrader_database`) with default `"mhsgrader"`
2. Adding a field to `AppConfig` (`MHSGraderDatabase`)
3. Calling `client.Database(appCfg.MHSGraderDatabase)` in `ConnectDB()`
4. Adding `MHSGraderDatabase *mongo.Database` to `DBDeps`
5. Passing it to handlers that need it

## Changes Required

Follow the identical pattern to add stratalog's database.

### appconfig.go

Add field:
```go
StratalogDatabase string
```

### config.go

Add to `appConfigKeys`:
```go
{Name: "stratalog_database", Default: "stratalog", Desc: "Stratalog database name for log data access"},
```

Add to `LoadConfig()` return:
```go
StratalogDatabase: appValues.String("stratalog_database"),
```

### db.go — ConnectDB()

Add after mhsgrader database line:
```go
stratalogDB := client.Database(appCfg.StratalogDatabase)
```

Return it in DBDeps.

### dbdeps.go

Add field:
```go
StratalogDatabase *mongo.Database
```

### routes.go — BuildHandler()

Pass to the mhsdashboard handler:
```go
mhsDashboardHandler := mhsdashboardfeature.NewHandler(
    deps.StrataHubMongoDatabase,
    deps.MHSGraderDatabase,
    deps.StratalogDatabase,  // NEW
    errLog, logger,
)
```

### mhsdashboard handler

Update `Handler` struct and `NewHandler()` to accept and store `LogDB *mongo.Database`.

### Production config (config.toml)

Add:
```toml
stratalog_database = "stratalog"
```

Since all databases are on the same DocumentDB cluster and share the same `mongo_uri`, no additional connection string is needed. The existing client simply references a third database name.

## Store for Log Data

Create a new store package at:
```
stratahub/internal/app/store/logdata/store.go
```

This store wraps read-only queries against stratalog's `logdata` collection:

```go
package logdata

type Store struct {
    c *mongo.Collection
}

func New(db *mongo.Database) *Store {
    return &Store{c: db.Collection("logdata")}
}

// ListForPlayer returns all log entries for a player in a game, ordered by _id.
func (s *Store) ListForPlayer(ctx context.Context, game, playerID string) ([]LogEntry, error)

// ListForPlayerUnit returns log entries filtered by scene names for a unit.
func (s *Store) ListForPlayerUnit(ctx context.Context, game, playerID string, sceneNames []string) ([]LogEntry, error)

// ListForPlayerWindow returns log entries between two ObjectIDs.
func (s *Store) ListForPlayerWindow(ctx context.Context, game, playerID string, afterID, beforeID primitive.ObjectID) ([]LogEntry, error)
```

The `LogEntry` struct mirrors stratalog's document shape:
```go
type LogEntry struct {
    ID              primitive.ObjectID     `bson:"_id"`
    Game            string                 `bson:"game"`
    PlayerID        string                 `bson:"playerId"`
    EventType       string                 `bson:"eventType"`
    EventKey        string                 `bson:"eventKey,omitempty"`
    ServerTimestamp time.Time              `bson:"serverTimestamp"`
    SceneName       string                 `bson:"sceneName,omitempty"`
    Version         string                 `bson:"version,omitempty"`
    Data            map[string]interface{} `bson:"data,omitempty"`
    Device          map[string]interface{} `bson:"device,omitempty"`
}
```

## Indexes

The stratalog database already has these indexes on `logdata`:
- `idx_logdata_game_playerId` — covers our primary query pattern (game + playerId)
- `idx_logdata_game_serverTimestamp` — covers time-range queries

No additional indexes are needed. Stratahub will only read, never write.
