# StrataLog Features & Capabilities

StrataLog is a game logging service built on the Waffle framework with MongoDB, Chi router, HTMX, and Tailwind CSS. It provides a production API for game log ingestion and a developer console for viewing and managing logs.

---

## Table of Contents

1. [Log API](#log-api)
2. [Log Browser](#log-browser)
3. [API Playground & Documentation](#api-playground--documentation)
4. [Authentication](#authentication)
5. [User Management](#user-management)
6. [Site Administration](#site-administration)
7. [Audit & Monitoring](#audit--monitoring)
8. [Configuration](#configuration)

---

## Log API

The core feature of StrataLog is a REST API for submitting and querying game logs.

### Endpoints

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/api/v1/logs` | POST | Bearer | Submit log entries |
| `/api/v1/logs` | GET | Bearer | Query log entries |
| `/logs` | POST | Bearer | Legacy submit endpoint |
| `/logs` | GET | Bearer | Legacy query endpoint |
| `/logs/view` | GET | None | Public HTML view |
| `/logs/download` | GET | None | Public JSON download |

### Single Entry Submission

Submit individual log entries with flexible schema:

```json
{
  "game": "mygame",
  "player_id": "player001",
  "event_type": "level_complete",
  "level": 5,
  "score": 1000
}
```

### Batch Submission

Submit multiple entries in a single request (up to 100 by default):

```json
{
  "game": "mygame",
  "entries": [
    {"player_id": "p1", "event_type": "login"},
    {"player_id": "p1", "event_type": "level_start", "level": 1},
    {"player_id": "p1", "event_type": "level_complete", "level": 1}
  ]
}
```

### Query Capabilities

- Filter by game (required)
- Filter by player_id
- Filter by event_type
- Filter by time range (start_time, end_time)
- Pagination with limit/offset
- Results sorted by timestamp (newest first)

### Data Model

| Field | Description |
|-------|-------------|
| `game` | Game identifier (required) |
| `player_id` | Player identifier (optional) |
| `event_type` | Event type string (optional) |
| `timestamp` | Client timestamp (optional) |
| `serverTimestamp` | Server timestamp (auto) |
| `data` | Additional fields (flexible schema) |

### Per-Game Collections

Each game's logs are stored in a separate MongoDB collection (`logs_<game>`) for:
- Better query performance
- Easier data isolation
- Independent scaling

---

## Log Browser

A developer console for viewing and managing logs, available at `/console/api/logs`.

### Features

| Feature | Description |
|---------|-------------|
| **Game Selector** | Switch between games |
| **Player Filter** | Filter by player with search |
| **Event Type Filter** | Filter by event type |
| **Pagination** | Navigate through log entries |
| **Expandable Rows** | View full JSON data |
| **Delete Operations** | Delete individual logs or all logs for a player |
| **Real-time Updates** | HTMX-powered dynamic loading |

### Access Control

- Requires authentication
- Admin and Developer roles only
- Delete operations admin-only in practice

---

## API Playground & Documentation

### Playground (`/console/api/logs/playground`)

Interactive API testing interface:

- Select operation (Submit or List)
- Configure request parameters
- Execute requests with live response
- View cURL equivalent commands
- API key auto-filled from config

### Documentation (`/console/api/logs/docs`)

Built-in API reference:

- Authentication instructions
- Request/response formats
- Field descriptions
- Error codes
- Example payloads

---

## Authentication

StrataLog provides authentication for console access.

### Supported Auth Methods

| Method | Description |
|--------|-------------|
| **Password** | Email/password with bcrypt hashing |
| **Email** | Passwordless via one-time codes or magic links |
| **Google OAuth** | OAuth2 integration |
| **Trust** | Development-only quick login |

### API Authentication

External API access uses Bearer token authentication:

```
Authorization: Bearer YOUR_API_KEY
```

The API key is configured via `STRATALOG_API_KEY` environment variable.

### Security Features

- Session-based authentication for console
- Configurable session duration
- CSRF protection on console routes
- Rate limiting on login attempts
- Audit logging of auth events

---

## User Management

### User Roles

| Role | Capabilities |
|------|--------------|
| **Admin** | Full access: user management, settings, all features |
| **Developer** | Log browser, playground, documentation, statistics |

### Admin Capabilities

- Create/edit/delete users
- Assign roles
- Enable/disable accounts
- Reset passwords
- Send invitations

---

## Site Administration

### Site Settings

| Setting | Description |
|---------|-------------|
| Site Name | Displayed in header |
| Logo | Upload custom logo |
| Landing Page | Customize homepage |
| Footer | Custom footer HTML |

### Announcements

System-wide announcements with:
- Multiple types (info, warning, success, error)
- Optional scheduling (start/end dates)
- Dismissible option
- Admin management interface

### Invitations

- Generate invitation links
- Email delivery
- Configurable expiry
- Single-use tokens

---

## Audit & Monitoring

### API Statistics

Track API usage at `/console/api/stats`:

| Metric | Description |
|--------|-------------|
| Request Count | Total API requests |
| Success/Error Rate | Request outcomes |
| Response Times | Average latency |
| By Endpoint | Breakdown by operation |

Statistics are aggregated into configurable time buckets.

### Audit Logging

Security event tracking:

#### Auth Events
- Login success/failure
- Logout
- Password changes
- Email verification

#### Admin Events
- User management actions
- Settings changes
- Configuration updates

### Request Ledger

API error logging for debugging:
- Failed requests (status >= 400)
- Request body preview
- Error messages
- Headers captured

### Health Endpoints

| Endpoint | Purpose |
|----------|---------|
| `/health` | Load balancer health check |
| `/healthz` | Kubernetes liveness probe |
| `/readyz` | Kubernetes readiness probe |

---

## Configuration

Configuration via environment variables (`STRATALOG_` prefix) or config file.

### Log API Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `api_key` | (none) | Bearer token for API auth |
| `max_batch_size` | 100 | Max entries per batch |
| `max_body_size` | 1MB | Max request body size |
| `api_stats_bucket` | 1h | Stats aggregation interval |

### Database

| Setting | Default | Description |
|---------|---------|-------------|
| `mongo_uri` | localhost:27017 | MongoDB connection |
| `mongo_database` | stratalog | Database name |
| `mongo_max_pool_size` | 100 | Max connections |

### Sessions

| Setting | Default | Description |
|---------|---------|-------------|
| `session_key` | (dev key) | Signing key (32+ chars) |
| `session_name` | stratalog-session | Cookie name |
| `session_max_age` | 24h | Session duration |

### Rate Limiting

| Setting | Default | Description |
|---------|---------|-------------|
| `rate_limit_enabled` | true | Enable/disable |
| `rate_limit_login_attempts` | 5 | Max failed attempts |
| `rate_limit_login_window` | 15m | Time window |
| `rate_limit_login_lockout` | 15m | Lockout duration |

### Storage

| Setting | Default | Description |
|---------|---------|-------------|
| `storage_type` | local | `local` or `s3` |
| `storage_local_path` | ./uploads | Upload directory |
| `storage_local_url` | /files | URL prefix |

### Email

| Setting | Default | Description |
|---------|---------|-------------|
| `mail_smtp_host` | localhost | SMTP server |
| `mail_smtp_port` | 1025 | SMTP port |
| `mail_from` | noreply@example.com | From address |
| `mail_from_name` | StrataLog | From name |

---

## Architecture

### Technology Stack

- **Go** - Backend language
- **Chi** - HTTP router
- **MongoDB** - Database
- **HTMX** - Dynamic UI updates
- **Tailwind CSS** - Styling
- **Waffle** - Framework (config, templates, utilities)

### Package Structure

```
internal/app/
├── features/
│   ├── logapi/         # REST API handlers
│   ├── logbrowser/     # Console UI handlers
│   ├── apistats/       # Statistics feature
│   └── ...             # Other features
├── store/
│   └── apistats/       # Statistics storage
├── system/
│   ├── auth/           # Authentication
│   ├── apistats/       # Stats middleware
│   └── ...             # Other utilities
└── bootstrap/
    ├── config.go       # Configuration
    └── routes.go       # Route mounting
```

### Request Flow

1. Request received by Chi router
2. Global middleware applied (CORS, security headers, session)
3. Route-specific middleware (auth, CSRF, stats recording)
4. Handler processes request
5. Response returned (JSON for API, HTML for console)
