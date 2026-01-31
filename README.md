# StrataLog

StrataLog is a game logging service that stores player event logs with flexible schema support. It provides a production-ready REST API for log ingestion, a developer console for viewing and managing logs, and built-in API documentation.

## Features

- **Log API**: Submit single or batch log entries via REST API with Bearer token authentication
- **Flexible Schema**: Store any JSON data alongside standard fields (game, player_id, event_type, timestamp)
- **Per-Game Collections**: Each game gets its own MongoDB collection for isolation and performance
- **Log Browser**: View, search, and filter logs by game, player, or event type
- **API Playground**: Interactive API testing interface with live request/response
- **API Documentation**: Built-in API reference documentation
- **Public Endpoints**: View and download logs without authentication
- **API Statistics**: Track request counts, response times, and error rates

## API Endpoints

### Authenticated (Bearer token required)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/logs` | Submit log entries (single or batch) |
| `GET` | `/api/v1/logs?game=<name>` | Query logs with filters |
| `POST` | `/logs` | Legacy endpoint for log submission |
| `GET` | `/logs` | Legacy endpoint for log queries |

### Public (no authentication)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/logs/view?game=<name>` | HTML view of recent logs |
| `GET` | `/logs/download?game=<name>` | Download logs as JSON file |

## Log Entry Format

### Single Entry

```json
{
  "game": "mygame",
  "player_id": "player001",
  "event_type": "level_complete",
  "timestamp": "2024-01-15T10:30:00Z",
  "level": 5,
  "score": 1000,
  "time_taken": 120
}
```

### Batch Submission

```json
{
  "game": "mygame",
  "entries": [
    {"player_id": "player001", "event_type": "level_start", "level": 5},
    {"player_id": "player001", "event_type": "item_collected", "item": "key"},
    {"player_id": "player001", "event_type": "level_complete", "level": 5, "score": 1000}
  ]
}
```

### Required Fields

- `game` - Game identifier (determines which collection to store in)

### Optional Standard Fields

- `player_id` - Player identifier
- `event_type` - Type of event (e.g., "level_complete", "login", "purchase")
- `timestamp` - Client-side timestamp (RFC3339 format)

### Additional Data

Any additional fields in the JSON payload are stored in a `data` object.

## Console Features

Access the developer console at `/console/api/logs` (requires admin or developer role).

- **Log Browser** (`/console/api/logs`) - View and filter logs
- **API Playground** (`/console/api/logs/playground`) - Test API requests interactively
- **API Documentation** (`/console/api/logs/docs`) - API reference

## Quick Start

### Prerequisites

- Go 1.24+
- MongoDB 7.0+
- (Optional) Mailpit for email testing

### Setup

1. Clone the repository
2. Copy configuration:
   ```bash
   cp config.example.toml config.toml
   ```
3. Configure MongoDB connection and API key in `config.toml`
4. Build and run:
   ```bash
   make dev
   ```
5. Access the dashboard at `http://localhost:8080/dashboard`

### Docker

```bash
# Start all services (MongoDB, Mailpit, App)
docker compose up -d

# Or just start dependencies for local development
docker compose up -d mongodb mailpit
make dev
```

### Testing the API

```bash
# Submit a log entry
curl -X POST http://localhost:8080/api/v1/logs \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"game":"test","player_id":"player1","event_type":"login"}'

# Query logs
curl -X GET "http://localhost:8080/api/v1/logs?game=test&limit=10" \
  -H "Authorization: Bearer your-api-key"

# View logs (public, no auth)
curl http://localhost:8080/logs/view?game=test
```

## Configuration

Configuration via environment variables with `STRATALOG_` prefix or `config.toml` file.

### Key Settings

| Setting | Description | Default |
|---------|-------------|---------|
| `api_key` | Bearer token for API authentication | (none) |
| `max_batch_size` | Maximum entries per batch submission | 100 |
| `max_body_size` | Maximum request body size | 1MB |
| `mongo_uri` | MongoDB connection string | `mongodb://localhost:27017` |
| `mongo_database` | Database name | `stratalog` |

See `config.example.toml` for all configuration options.

## Development

```bash
make build       # Build the application
make run         # Build and run
make dev         # Run with hot reload (requires air)
make test        # Run tests
make css         # Build Tailwind CSS
make css-watch   # Watch and rebuild CSS
```

## Documentation

- [Configuration Guide](docs/configuration.md)
- [Deployment Guide](docs/deployment.md)
- [API Documentation](docs/api-documentation.md)
- [Database Schema](docs/database-schema.md)

## Architecture

StrataLog is built on:

- **Go** with Chi router
- **MongoDB** for data storage
- **HTMX** for interactive UI
- **Tailwind CSS** for styling
- **Waffle** framework for configuration, templates, and utilities

## License

MIT License
