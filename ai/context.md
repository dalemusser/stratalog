# StrataLog — Context for Claude Code

## What This Repo Does

StrataLog is a production-ready game logging service that stores player event logs with flexible JSON schema support. It provides a REST API for secure log ingestion (via Bearer token authentication), a developer console for viewing and managing logs with filtering/search capabilities, and built-in API documentation and testing playground. The service supports per-game MongoDB collections for data isolation and includes public endpoints for viewing/downloading logs without authentication.

## Technology Stack

- **Language(s):** Go 1.24+
- **Framework(s):** Chi (HTTP router), Waffle (custom config/templates/utilities framework)
- **Database:** MongoDB 7.0+
- **Frontend:** HTMX, Tailwind CSS
- **Authentication:** Bearer token (API), Session-based (web UI)
- **Deployment:** Docker/Docker Compose support, Linux 386 builds available

## Folder Structure

```
stratalog/
├── cmd/stratalog/
│   └── main.go                 # Entry point; delegates to waffle.app.Run()
├── internal/
│   ├── app/
│   │   ├── bootstrap/          # App initialization: config, DB connection, routes, hooks
│   │   ├── features/           # Feature handlers (logapi, logbrowser, auth, stats, etc.)
│   │   ├── store/              # MongoDB data access layer (entities, CRUD)
│   │   ├── resources/          # Static assets (CSS, images, templates)
│   │   └── system/             # Shared utilities (auth, viewdata, ledger, apistats)
│   └── domain/models/          # Domain entities (User, File, Folder, etc.)
├── docs/                       # Configuration, deployment, API docs
├── Makefile                    # Build targets (build, dev, test, css, setup)
├── config.example.toml         # Configuration template
├── docker-compose.yml          # Local dev environment (MongoDB, Mailpit)
└── Dockerfile                  # Production container
```

The organizing principle is **feature-based architecture**: each user-facing feature (log API, log browser, authentication, stats, etc.) lives in `internal/app/features/<feature>` with handler, routes, and view models. Database access patterns are centralized in `internal/app/store/<entity>` following a consistent CRUD pattern.

## Code Patterns & Conventions

### Feature Structure

Each feature in `internal/app/features/<feature>/` contains:
- `handler.go` — Main HTTP handler struct with dependencies (db, logger, etc.)
- `routes.go` — Returns `http.Handler` (chi router or mux)
- `types.go` (optional) — Request/response structs
- Templates in `internal/app/resources/` organized by feature

**Handler pattern:**
```go
type Handler struct {
    db     *mongo.Database
    logger *zap.Logger
    // other dependencies
}

func NewHandler(db *mongo.Database, logger *zap.Logger, ...) *Handler { ... }
func (h *Handler) Routes() http.Handler { ... }
```

### Store Pattern

Data access in `internal/app/store/<entity>/`:
- `New(db *mongo.Database) *Store`
- CRUD methods with `context.Context` as first parameter
- Input/Update structs for complex operations
- Returns `(T, error)` or `([]T, error)`

**Store pattern example:**
```go
type Store struct { db *mongo.Database }
func New(db *mongo.Database) *Store { return &Store{db} }
func (s *Store) Create(ctx context.Context, input *CreateInput) (*Entity, error) { ... }
func (s *Store) GetByID(ctx context.Context, id string) (*Entity, error) { ... }
func (s *Store) List(ctx context.Context, filter bson.M) ([]Entity, error) { ... }
```

### Naming Conventions

- **Go code:** `camelCase` for vars/funcs, `PascalCase` for types/exports
- **API URLs:** `/api/v1/logs`, `/console/`, `/logs/view` (hyphens for multi-word paths)
- **JSON fields:** `snake_case` (e.g., `user_id`, `event_type`, `player_id`)
- **Database fields:** `snake_case` in MongoDB; `_id` for primary keys
- **Game names:** Alphanumeric, underscores, hyphens only (validated by regex `^[A-Za-z0-9_-]+$`)
- **User IDs:** 24-character lowercase hex strings (MongoDB ObjectID hex form)

### Error Handling

- Errors returned from handlers are logged via `zap.Logger`
- HTTP responses use `writeJSONError(w, r, message, code, statusCode)` helper
- Error codes: `INVALID_JSON`, `BODY_TOO_LARGE`, `INVALID_GAME`, etc.
- No panics; all errors surface as HTTP responses

### Authentication

- **API endpoints:** Require Bearer token (set via `STRATALOG_API_KEY` config)
- **Web UI:** Session-based authentication using `auth.CurrentUser(r)` helper
- **Middleware:** `sessionMgr.RequireAuth`, `sessionMgr.RequireRole("admin")`
- **Staff pages:** `/logs/view`, `/logs/download` require a console sign-in (admin or developer); nothing that returns log entries is public
- **Restricted key:** `api_key_restricted` is accepted only for the user ids in `api_key_restricted_user_ids` (default: the game's built-in developer user); `api_key_restricted_enforce = false` logs other use instead of refusing it (`auth.APIKeyAuthRestricted`)
- **Key replacement:** `api_keys_extra` (comma-separated) lists further accepted Bearer tokens, so a key can be replaced without downtime

### View Data

Templates receive a `BaseVM` (view model) struct with:
- `SiteName`, `Title`, `BackURL`, `CurrentPath`
- `IsLoggedIn`, `Role`, `UserName`
- `LogoURL`, `FooterHTML`

Initialize via `viewdata.New(r)` (minimal) or `viewdata.NewBaseVM(r, db, title, backURL)` (full).

## Key Features & Endpoints

### Log API (Authenticated via Bearer Token)

- **POST** `/api/v1/logs` — Submit single or batch log entries
  - Batch payload: `{ "game": "...", "entries": [ {...}, {...} ] }`
  - Single payload: `{ "game": "...", "user_id": "...", "eventType": "...", ... }`
  - All entries must include `user_id` (24-char hex) or `player_id` (legacy compatibility)
  
- **GET** `/api/v1/logs?game=<name>&limit=10&offset=0` — Query logs with filters

- **Legacy** POST/GET `/logs` — Backward-compatible endpoints

### Log Browser (Authenticated, Admin/Dev Role)

- **GET** `/console/api/logs` — View, search, filter logs by game/player/event type
- **GET** `/console/api/logs/playground` — Interactive API testing
- **GET** `/console/api/logs/docs` — API reference documentation

### Staff View and Download (console sign-in: admin or developer)

Off by default together with the list endpoint (`log_read_routes_enabled`, default false): while off, `GET /api/log/list`, `GET /logs`, `/logs/view` and `/logs/download` answer 410 through `logapi.Handler.DisabledHandler`, which logs who asked. The handlers remain; the Log Browser is the way to read entries.

- **GET** `/logs/view?game=<name>` — HTML view of recent logs
- **GET** `/logs/download?game=<name>` — Download logs as JSON file

### Stats & Monitoring

- **GET** `/api/stats` — API request counts, response times, error rates
- **GET** `/health` — Health check endpoint

## Key Dependencies & Gotchas

### MongoDB Collection Strategy

StrataLog uses per-game collections: all logs for a single game are stored in a collection named after the game (e.g., `mhs`, `test`). The unified collection name for metadata/analytics is `logdata`. This design trades flexibility for performance isolation—each game's logs are completely separate. Be aware that a misbehaving game client can flood a single collection; rate limiting is done at the API handler level.

### Waffle Framework Lifecycle

The app uses the Waffle framework's lifecycle hooks (defined in `bootstrap/hooks.go`):
1. `LoadConfig` — Loads TOML + env vars (STRATALOG_ prefix)
2. `ValidateConfig` — Checks MongoDB URI, API key
3. `ConnectDB` — Connects to MongoDB, returns DBDeps
4. `EnsureSchema` — Creates indexes
5. `Startup` — Seeds admin user, loads shared templates
6. `BuildHandler` — Constructs router; mounts all features
7. `Shutdown` — Disconnects MongoDB gracefully

Modifying the bootstrap flow requires changes in `bootstrap/hooks.go` and `bootstrap/routes.go`.

### Ledger & API Stats

Two key systems in `internal/app/system/`:
- **Ledger** (`ledger/`) — Tracks resource allocations (for quota/billing)
- **API Stats** (`apistats/`) — Tracks request counts, latencies, error rates per API key

Both write to MongoDB collections and are meant to be queried by monitoring/analytics systems.

### User ID Validation

All log entries must use `user_id` (not `playerId` or other variants). The regex check in `logapi/handler.go` ensures it matches `^[0-9a-f]{24}$` (24-char lowercase hex). Legacy entries with `playerId` are rejected or require a migration adapter.

## How to Run Locally

### Prerequisites
- Go 1.24+
- MongoDB 7.0+
- (Optional) Mailpit for email testing

### Quick Start

```bash
# 1. Copy config template
cp config.example.toml config.toml

# 2. Edit config.toml with your MongoDB URI and API key
vi config.toml

# 3. Build Tailwind CSS
make setup-tailwind
make css

# 4. Run in development mode (hot reload if 'air' is installed)
make dev

# 5. Access the app
http://localhost:8080/dashboard
```

### With Docker Compose

```bash
# Start MongoDB + Mailpit + app
docker compose up -d

# Or just start dependencies for local dev
docker compose up -d mongodb mailpit
make dev
```

### Testing

```bash
# Run all tests
make test

# With coverage
make test-cover

# Lint and format
make lint
make fmt
```

## Related Repos

- **waffle** — Shared Go framework for configuration, templates, utilities (used by stratalog, stratahub)
- **stratahub** — Main web app for Mission HydroSci; uses similar architecture
- **mhscurriculum** — Curriculum docs for Mission HydroSci (data source for stratahub)

## Notes for Claude

### When Working on Log API

- Always validate game names (alphanumeric, -, _ only)
- Always validate `user_id` as 24-char lowercase hex; do not accept `playerId`
- Batch submissions support up to `max_batch_size` entries (default 100, configurable)
- Request body limited to 1MB by default (configurable)
- All timestamps should be RFC3339 format; server adds a `serverTimestamp` if client doesn't provide one

### When Adding Features

1. Create `internal/app/features/<feature>/` with `handler.go`, `routes.go`
2. Create store(s) in `internal/app/store/<entity>/` if persisting data
3. Mount routes in `bootstrap/routes.go` (BuildHandler function)
4. Add configuration in `bootstrap/appconfig.go` if needed
5. Follow existing naming/error handling patterns

### Configuration

All configuration uses environment variables with `STRATALOG_` prefix or `config.toml`. Key settings:
- `STRATALOG_API_KEY` — Bearer token for API endpoints
- `STRATALOG_MONGO_URI` — MongoDB connection string
- `STRATALOG_MONGO_DATABASE` — Database name (default: `stratalog`)
- `STRATALOG_MAX_BATCH_SIZE` — Max entries per batch request (default: 100)
- `STRATALOG_MAX_BODY_SIZE` — Max request body size in bytes (default: 1MB)

See `bootstrap/appconfig.go` for all available options.

### Common Gotchas

- **HTMX form submissions:** Use `<form hx-post="...">` with CSRF token from template `{{ .CSRFToken }}`
- **Context cancellation:** Always check `ctx.Done()` in long-running operations
- **MongoDB ObjectID:** Use `primitive.ObjectID` for internal IDs; convert to hex string for APIs
- **Template paths:** Templates are embedded; paths relative to `internal/app/resources/`
- **Middleware order matters:** CSRF must come before auth checks in the router stack
