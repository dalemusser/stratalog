# StrataLog API Documentation

This document provides a complete reference for the StrataLog REST API.

## Authentication

All authenticated endpoints require a Bearer token in the Authorization header:

```
Authorization: Bearer YOUR_API_KEY
```

The API key is configured via the `STRATALOG_API_KEY` environment variable or `api_key` in config.toml.

---

## Endpoints

### Submit Log Entry

Submit a single log entry or batch of entries.

**Endpoint:** `POST /api/v1/logs` or `POST /logs`

**Authentication:** Required (Bearer token)

#### Single Entry Request

```json
{
  "game": "mygame",
  "player_id": "player001",
  "event_type": "level_complete",
  "timestamp": "2024-01-15T10:30:00Z",
  "level": 5,
  "score": 1000
}
```

#### Batch Request

```json
{
  "game": "mygame",
  "entries": [
    {"player_id": "player001", "event_type": "level_start", "level": 5},
    {"player_id": "player001", "event_type": "enemy_defeated", "enemy": "dragon"},
    {"player_id": "player001", "event_type": "level_complete", "level": 5, "score": 1000}
  ]
}
```

#### Request Fields

| Field | Required | Description |
|-------|----------|-------------|
| `game` | Yes | Game identifier (determines storage collection) |
| `player_id` | No | Player identifier |
| `event_type` | No | Type of event (e.g., "level_complete", "login") |
| `timestamp` | No | Client timestamp (RFC3339 format). Server adds `serverTimestamp` automatically |
| `entries` | No | Array of entries for batch submission (max 100 by default) |
| `*` | No | Any additional fields are stored in the `data` object |

#### Success Response (201 Created)

Single entry:
```json
{
  "success": true,
  "id": "507f1f77bcf86cd799439011",
  "message": "Log entry saved"
}
```

Batch entry:
```json
{
  "success": true,
  "count": 3,
  "message": "Batch log entries saved"
}
```

#### Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 400 | `INVALID_JSON` | Request body is not valid JSON |
| 400 | `MISSING_FIELD` | Required field `game` is missing |
| 400 | `EMPTY_ENTRIES` | Batch entries array is empty |
| 400 | `BATCH_TOO_LARGE` | Batch exceeds maximum size |
| 400 | `INVALID_ENTRY` | Invalid entry in batch array |
| 401 | - | Missing or invalid Authorization header |
| 500 | `INSERT_FAILED` | Database insert operation failed |

---

### List Log Entries

Query log entries with filters.

**Endpoint:** `GET /api/v1/logs` or `GET /logs`

**Authentication:** Required (Bearer token)

#### Query Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `game` | Yes | Filter by game name |
| `player_id` | No | Filter by player ID |
| `event_type` | No | Filter by event type |
| `start_time` | No | Filter entries after this time (RFC3339) |
| `end_time` | No | Filter entries before this time (RFC3339) |
| `limit` | No | Max entries to return (default: 100, max: 1000) |
| `offset` | No | Skip this many entries for pagination |

#### Example Request

```
GET /api/v1/logs?game=mygame&player_id=player001&limit=50
```

#### Success Response (200 OK)

```json
{
  "entries": [
    {
      "id": "507f1f77bcf86cd799439011",
      "game": "mygame",
      "player_id": "player001",
      "event_type": "level_complete",
      "timestamp": "2024-01-15T10:30:00Z",
      "serverTimestamp": "2024-01-15T10:30:05.123Z",
      "data": {
        "level": 5,
        "score": 1000
      }
    }
  ],
  "total": 150,
  "limit": 100,
  "offset": 0
}
```

#### Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 400 | `MISSING_PARAM` | Required parameter `game` is missing |
| 401 | - | Missing or invalid Authorization header |
| 500 | `QUERY_FAILED` | Database query operation failed |
| 500 | `DECODE_FAILED` | Failed to decode log entries |

---

### View Logs (Public)

View recent log entries as an HTML page.

**Endpoint:** `GET /logs/view`

**Authentication:** Not required

#### Query Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `game` | Yes | Game name to view |
| `limit` | No | Max entries (default: 100, max: 1000) |

#### Example

```
GET /logs/view?game=mygame&limit=50
```

Returns an HTML page displaying the log entries.

---

### Download Logs (Public)

Download log entries as a JSON file.

**Endpoint:** `GET /logs/download`

**Authentication:** Not required

#### Query Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `game` | Yes | Game name to download |
| `limit` | No | Max entries (default: 1000, max: 10000) |

#### Example

```
GET /logs/download?game=mygame
```

Returns a JSON file named `<game>_logs_<timestamp>.json`.

---

## Error Response Format

All error responses follow this format:

```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE"
}
```

---

## Rate Limiting

The API does not currently implement rate limiting. For high-volume applications, consider implementing client-side rate limiting or contacting the administrator for guidance.

---

## Examples

### cURL Examples

#### Submit single log entry

```bash
curl -X POST https://example.com/api/v1/logs \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "game": "mygame",
    "player_id": "player001",
    "event_type": "login"
  }'
```

#### Submit batch entries

```bash
curl -X POST https://example.com/api/v1/logs \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "game": "mygame",
    "entries": [
      {"player_id": "player001", "event_type": "level_start", "level": 1},
      {"player_id": "player001", "event_type": "level_complete", "level": 1, "score": 500}
    ]
  }'
```

#### Query logs

```bash
curl -X GET "https://example.com/api/v1/logs?game=mygame&player_id=player001&limit=10" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

#### Query with time range

```bash
curl -X GET "https://example.com/api/v1/logs?game=mygame&start_time=2024-01-01T00:00:00Z&end_time=2024-01-31T23:59:59Z" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### JavaScript Example

```javascript
const API_KEY = 'your-api-key';
const BASE_URL = 'https://example.com';

// Submit a log entry
async function submitLog(game, playerId, eventType, data = {}) {
  const response = await fetch(`${BASE_URL}/api/v1/logs`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${API_KEY}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      game,
      player_id: playerId,
      event_type: eventType,
      ...data
    })
  });
  return response.json();
}

// Query logs
async function queryLogs(game, options = {}) {
  const params = new URLSearchParams({ game, ...options });
  const response = await fetch(`${BASE_URL}/api/v1/logs?${params}`, {
    headers: {
      'Authorization': `Bearer ${API_KEY}`
    }
  });
  return response.json();
}

// Usage
await submitLog('mygame', 'player001', 'level_complete', { level: 5, score: 1000 });
const logs = await queryLogs('mygame', { player_id: 'player001', limit: 50 });
```

### Unity C# Example

```csharp
using UnityEngine;
using UnityEngine.Networking;
using System.Collections;
using System.Text;

public class StrataLogClient : MonoBehaviour
{
    private const string API_URL = "https://example.com/api/v1/logs";
    private const string API_KEY = "your-api-key";

    public IEnumerator SubmitLog(string game, string playerId, string eventType, object data)
    {
        var payload = new {
            game = game,
            player_id = playerId,
            event_type = eventType,
            // Add additional fields from data
        };

        string json = JsonUtility.ToJson(payload);
        byte[] bodyRaw = Encoding.UTF8.GetBytes(json);

        using (UnityWebRequest request = new UnityWebRequest(API_URL, "POST"))
        {
            request.uploadHandler = new UploadHandlerRaw(bodyRaw);
            request.downloadHandler = new DownloadHandlerBuffer();
            request.SetRequestHeader("Content-Type", "application/json");
            request.SetRequestHeader("Authorization", "Bearer " + API_KEY);

            yield return request.SendWebRequest();

            if (request.result == UnityWebRequest.Result.Success)
            {
                Debug.Log("Log submitted: " + request.downloadHandler.text);
            }
            else
            {
                Debug.LogError("Error: " + request.error);
            }
        }
    }
}
```

---

## Best Practices

1. **Batch when possible**: For high-volume logging, use batch submissions to reduce network overhead.

2. **Include timestamps**: Provide client-side `timestamp` for accurate event timing; the server adds `serverTimestamp` for when it was received.

3. **Use meaningful event types**: Consistent event type naming makes querying and analysis easier.

4. **Handle errors gracefully**: Implement retry logic for transient failures (5xx errors).

5. **Secure your API key**: Never expose the API key in client-side code for web applications. Use a backend proxy if needed.
