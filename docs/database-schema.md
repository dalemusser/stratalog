# StrataLog Database Schema

This document describes the MongoDB database schema used by StrataLog.

## Overview

StrataLog uses a dynamic collection strategy where each game gets its own collection for log entries. This provides:

- **Isolation**: Each game's logs are stored separately
- **Performance**: Queries are scoped to a single collection
- **Scalability**: Collections can be sharded independently

## Collections

### Log Collections (`logs_<game>`)

Each game has its own collection named `logs_<gamename>`. For example, a game called "mhs" would have logs stored in the `logs_mhs` collection.

#### Schema

```javascript
{
  _id: ObjectId,                    // Auto-generated MongoDB ID
  game: String,                     // Game identifier (redundant but useful)
  player_id: String,                // Player identifier (optional)
  event_type: String,               // Event type (optional)
  timestamp: ISODate,               // Client-provided timestamp (optional)
  serverTimestamp: ISODate,             // Server timestamp (auto-generated)
  data: Object                      // Additional fields from the payload
}
```

#### Example Document

```javascript
{
  "_id": ObjectId("507f1f77bcf86cd799439011"),
  "game": "mhs",
  "player_id": "player001",
  "event_type": "level_complete",
  "timestamp": ISODate("2024-01-15T10:30:00Z"),
  "serverTimestamp": ISODate("2024-01-15T10:30:05.123Z"),
  "data": {
    "level": 5,
    "score": 1000,
    "time_taken": 120,
    "enemies_defeated": 15
  }
}
```

#### Indexes

Created automatically when logs are submitted to a new game:

| Index | Fields | Purpose |
|-------|--------|---------|
| game_serverTimestamp | `{game: 1, serverTimestamp: -1}` | Query by game, sorted by time |
| player_serverTimestamp | `{player_id: 1, serverTimestamp: -1}` | Query by player |
| event_serverTimestamp | `{event_type: 1, serverTimestamp: -1}` | Query by event type |

---

### System Collections

StrataLog also uses several system collections for authentication, audit logging, and administration.

#### users

User accounts for console access.

```javascript
{
  _id: ObjectId,
  full_name: String,
  full_name_ci: String,           // Case-insensitive for sorting
  login_id: String,               // Email or username
  login_id_ci: String,            // Folded for matching
  email: String,
  auth_method: String,            // password, email, google, trust
  password_hash: String,          // Bcrypt hash
  role: String,                   // admin, developer
  status: String,                 // active, disabled
  theme_preference: String,       // light, dark, system
  created_at: ISODate,
  updated_at: ISODate
}
```

#### sessions

Active user sessions for the console.

```javascript
{
  _id: ObjectId,
  token: String,                  // Unique session token
  user_id: ObjectId,
  ip: String,
  user_agent: String,
  login_at: ISODate,
  logout_at: ISODate,             // null if active
  last_activity: ISODate,
  expires_at: ISODate             // TTL index
}
```

#### audit_events

Security and admin action audit log.

```javascript
{
  _id: ObjectId,
  timestamp: ISODate,
  category: String,               // auth, admin
  event_type: String,
  user_id: ObjectId,
  actor_id: ObjectId,
  ip: String,
  user_agent: String,
  success: Boolean,
  failure_reason: String,
  details: Object
}
```

#### api_stats

API request statistics (aggregated).

```javascript
{
  _id: ObjectId,
  stat_type: String,              // log_submit, log_list
  bucket: ISODate,                // Time bucket for aggregation
  count: Number,
  total_duration_ms: Number,
  error_count: Number
}
```

#### site_settings

Site-wide configuration.

```javascript
{
  _id: ObjectId,
  site_name: String,
  logo_path: String,
  footer_html: String,
  landing_title: String,
  landing_content: String,
  updated_at: ISODate,
  updated_by_id: ObjectId,
  updated_by_name: String
}
```

#### pages

Editable content pages (about, contact, terms, privacy).

```javascript
{
  _id: ObjectId,
  slug: String,                   // Unique: about, contact, terms, privacy
  title: String,
  content: String,                // HTML content
  updated_at: ISODate,
  updated_by_id: ObjectId,
  updated_by_name: String
}
```

#### announcements

System-wide announcements.

```javascript
{
  _id: ObjectId,
  title: String,
  content: String,
  type: String,                   // info, warning, success, error
  dismissible: Boolean,
  active: Boolean,
  starts_at: ISODate,
  ends_at: ISODate,
  created_at: ISODate,
  updated_at: ISODate
}
```

#### ledger

API error log for debugging.

```javascript
{
  _id: ObjectId,
  timestamp: ISODate,
  method: String,
  path: String,
  status: Number,
  duration_ms: Number,
  request_body: String,           // Truncated preview
  error_message: String,
  headers: Object
}
```

---

## Data Flow

### Log Submission

1. Client sends POST request with JSON payload
2. Server extracts `game` field (required)
3. Server extracts standard fields: `player_id`, `event_type`, `timestamp`
4. Remaining fields go into `data` object
5. Server adds `serverTimestamp` (current UTC time)
6. Document inserted into `logs_<game>` collection
7. Indexes created if new collection

### Batch Submission

1. Client sends POST with `game` and `entries` array
2. Server validates batch size (max configurable, default 100)
3. Each entry processed as above
4. All entries get same `serverTimestamp`
5. `insertMany` for atomic batch insert

---

## Query Patterns

### By Game (most recent)

```javascript
db.logs_mhs.find({game: "mhs"})
  .sort({serverTimestamp: -1})
  .limit(100)
```

### By Player

```javascript
db.logs_mhs.find({
  game: "mhs",
  player_id: "player001"
}).sort({serverTimestamp: -1})
```

### By Event Type

```javascript
db.logs_mhs.find({
  game: "mhs",
  event_type: "level_complete"
}).sort({serverTimestamp: -1})
```

### By Time Range

```javascript
db.logs_mhs.find({
  game: "mhs",
  serverTimestamp: {
    $gte: ISODate("2024-01-01"),
    $lte: ISODate("2024-01-31")
  }
}).sort({serverTimestamp: -1})
```

### Combined Filters

```javascript
db.logs_mhs.find({
  game: "mhs",
  player_id: "player001",
  event_type: "level_complete",
  serverTimestamp: {$gte: ISODate("2024-01-01")}
}).sort({serverTimestamp: -1})
```

---

## Maintenance

### Listing Game Collections

```javascript
db.getCollectionNames().filter(n => n.startsWith("logs_"))
```

### Collection Statistics

```javascript
db.logs_mhs.stats()
```

### Delete Old Logs

```javascript
// Delete logs older than 90 days
db.logs_mhs.deleteMany({
  serverTimestamp: {$lt: new Date(Date.now() - 90*24*60*60*1000)}
})
```

### Drop Game Collection

```javascript
db.logs_mhs.drop()
```
