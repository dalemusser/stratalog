

# Progress Point Database Records (StrataLog)

This document defines the database records used to store **progress point evaluation results** for the MHS Dashboard.

These records materialize gameplay evaluation results so the dashboard can load quickly, while still preserving enough structured data to explain *why* a progress point is yellow.

---

## Design goals

- Fast dashboard loading (no heavy log queries at render time)
- Deterministic green/yellow values per (player, unit, point)
- Support backfill and near‑real‑time updates
- Preserve grading evidence for audit and explanation
- Allow future rule evolution without breaking historical data

---

## 1. `progress_point_grades`

**Purpose:**

Stores the *current* grade (green/yellow) for a single progress point for a single player.

There is **at most one record** per:

- game
- playerId
- unit
- point

This is the primary collection queried by the MHS Dashboard.

### Document shape

```js
{
  _id: ObjectId(),

  // Identity / partitioning
  game: "mhs",
  playerId: "wenyi10@mhs.mhs",
  unit: 2,
  point: 3,

  // Dashboard value
  color: "green", // or "yellow"

  // Which grading rule produced this value
  ruleId: "u2p3_v1",

  // Operational metadata
  computedAt: ISODate("2026-01-31T21:01:23.456Z"),

  // Trigger that caused this evaluation
  trigger: {
    eventKey: "DialogueNodeEvent:22:18",
    logId: ObjectId("..."),
    logTimestamp: "2025-11-25T18:29:18.2877523Z" // client timestamp (optional)
  },

  // Machine‑readable explanation scaffold
  reasonCode: "TOO_MANY_TARGETS",

  // Metrics captured during grading (rule‑specific)
  metrics: {
    // Example values (vary per rule)
    countTargets: 9,
    threshold: 6,
    score: 3.33,
    posCount: 7,
    negCount: 11,

    // Windowed rules only
    window: {
      startKey: "DialogueNodeEvent:20:33",
      endKey: "DialogueNodeEvent:22:1",
      startTimestamp: "2025-11-25T18:10:00.000Z",
      endTimestamp: "2025-11-25T18:35:00.000Z",
      durationSeconds: 1500,
      maxDurationSeconds: 7200
    }
  },

  // Optional attempt / replay discriminator (future‑proofing)
  attemptId: null
}
```

### Indexes

```js
// Enforce one grade per progress point
{ game: 1, playerId: 1, unit: 1, point: 1 }  // unique

// Fast dashboard loads
{ game: 1, playerId: 1 }
```

---

## 2. `progress_point_grade_events` (optional but recommended)

**Purpose:**

Stores an **append‑only audit trail** of every grading run.

This is useful for:

- debugging grading behavior
- explaining changes to teachers/admins
- validating rule updates

### Document shape

```js
{
  _id: ObjectId(),

  game: "mhs",
  playerId: "wenyi10@mhs.mhs",
  unit: 2,
  point: 3,

  ruleId: "u2p3_v1",
  computedAt: ISODate("2026-01-31T21:01:23.456Z"),

  // What triggered this grading run
  trigger: {
    eventKey: "DialogueNodeEvent:22:18",
    logId: ObjectId("..."),
    cursorId: ObjectId("...") // grader cursor position (optional)
  },

  // Result of the grading rule
  result: {
    color: "yellow",
    reasonCode: "TOO_MANY_TARGETS",
    metrics: { /* same structure as progress_point_grades.metrics */ }
  }
}
```

### Indexes

```js
{ game: 1, playerId: 1, computedAt: -1 }
{ game: 1, unit: 1, point: 1, computedAt: -1 }
```

---

## 3. `grader_state`

**Purpose:**

Tracks incremental scanning state for the grading service.

This allows the grader to:

- backfill historical log data
- process new log entries in near‑real‑time
- resume safely after restarts

### Document shape

```js
{
  _id: "mhs-grader", // stable identifier
  game: "mhs",

  // Cursor for scanning logdata
  lastSeenId: ObjectId("..."),

  // Operational bookkeeping
  updatedAt: ISODate("2026-01-31T21:02:00.000Z"),
  mode: "realtime", // "backfill" | "realtime"

  lag: {
    lastProcessedLogId: ObjectId("..."),
    lastProcessedLogServerTimestamp: ISODate("2026-01-31T21:01:59.000Z")
  },

  worker: {
    hostname: "ip-10-0-0-12",
    pid: 1234
  }
}
```

### Indexes

No additional indexes required (lookup by `_id`).

---

## 4. Relationship to dashboard loading

The MHS Dashboard loads progress data by:

1. Resolving the set of `playerId` values for the selected group/class (via StrataHub membership data).
2. Querying:

```js
db.progress_point_grades.find({
  game: "mhs",
  playerId: { $in: playerIds }
});
```

3. Rendering the grid:

- Existing records → green/yellow
- Missing records → not started

---

## Notes on replay / save rewind

If players can load earlier saves and replay content, log data may contain *multiple attempts* for the same unit/point.

Two strategies are supported:

1. **Preferred:** add an `attemptId` / `runId` to log records and include it in grading queries.
2. **Fallback:** store window anchors (`startLogId`, `endLogId`) during grading so evaluations remain bounded to a single attempt.

The schema above supports either approach without breaking compatibility.

---