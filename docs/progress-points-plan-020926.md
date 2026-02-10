# MHS Progress Points Grading Plan (StrataLog → StrataHub Dashboard)

## Initial Statement of Need

We have the MHS Dashboard in StrataLog.  The MHS Dashboard displays the performance of gameplay on the dashboard for teachers to see. The Dashboard has a list of students and a list of units and progress points.  Each progress point in each unit displays a green or yellow square.  The determination of green or yellow comes from running database queries on the log data when provided eventKey values appear in the log data.  Once we receive a log entry with a given eventKey we know the student has progressed to a point where we can run the associated query to determine if they should receive a green or yellow for that progress point.

In stratalog we receive log entries for a given playerId. Those log entries may contain an "eventKey" property. If the "eventKey" has one of the values presented below we then run a query for mongodb to determine if the player (playerId) should receive a "green" or "yellow" for the progress point (point) in the unit specified. The progress point values, once determined, are stored in a collection in the database for the MHS Dashboard to display in StrataHub.

Originally I was thinking of checking log entries as they are submitted and if they match one of the eventKey values provided below the need to run the script for the given playerId and eventKey would be put in a queue where a grading service would pull items out of the queue, perform the query, and then put the result in the database to be used by the dashboard. One of the problems with the queue approach is that we have already had the logger collect log entries before there was an implementation of a queue.  And it is likely based on how development is going we won't have determined the eventKey values and database queries needed until we have collected some data for them which means for some players their records will exist in the database before there will be knowledge to add them to what the queueing mechanism knows those eventKey values should cause entries to be added to the queue.  So, this had me thinking the grader should query for the eventKey values and to know which ones have already been processed, keep the date and time of the last query and use that to determine which ones to retrieve in the next query.

We need to write a service that handles the determination of green or yellow for progress points in units in the game for players. We need to determine how the results of the queries (the green and yellow values for each progress point for each player) are stored so that the dashboard can efficiently load them for display on the dashboard. 

We need to determine how the following information is codified for the grader so we can add progress points for units as they are determined over time. 

## Overview

The **MHS Dashboard** in **StrataLog** displays student gameplay progress for teachers.

- The dashboard shows a **grid**: students (rows) × units/points (columns).
- Each unit/point cell is one of:
  - **green** (completed / OK)
  - **yellow** (needs review)
  - **not started** (no grade yet)

A unit/point becomes green or yellow by running a **MongoDB grading rule** against the **log data**.

## Log data facts

Log entries are stored in the `logdata` collection.

- Each log entry belongs to a game via: `game: "mhs"`
- Each log entry belongs to a student via: `playerId`
- Some log entries include: `eventKey`

There are two time values in each record:

- `timestamp` (string): client/game time, e.g. `2025-11-25T18:29:18.2877523Z`
- `serverTimestamp` (ISODate): when the server inserted the record

### Time choice

We **must use `timestamp`** for anything involving elapsed time between gameplay events.

Reason: log entries can be cached and sent later. Using `serverTimestamp` would measure *network/upload delay* rather than *gameplay time*.

## Terminology

### Trigger event vs grading window

These two ideas are different and should be treated separately in implementation:

- **Trigger event**: “we should evaluate (or re-evaluate) unit X / point Y for this player now.”
- **Grading window**: “what slice of the player’s log history counts for this evaluation.”

In practice:

- Trigger discovery should use **arrival order** (`_id` cursor) so we can backfill and catch new events.
- Duration/sequence evaluation should use **client time** (`timestamp`).

## When grades are computed

A grade is computed when a log record arrives whose `eventKey` matches a unit/point’s trigger.

Note: in an earlier version of the documents provided by Data Analytics there was a progress point with two possible eventKey triggers, therefore it is possible there could be more than one trigger for grading a progress point.

## Which unit/points use time duration

Only these currently rely on a start/end window (and therefore on duration/sequence correctness):

- **Unit 2, Point 2**
- **Unit 2, Point 3**

Everything else is based on presence/absence or counts and does not depend on elapsed time.

## Grader service approach

### Why we are not using a simple queue-first approach

A pure “push work to a queue at ingest time” approach breaks down because:

- log data already exists from before the queue mechanism existed
- grading rules (trigger keys + queries) are discovered over time, *after* data is collected

### Recommended approach: incremental scanner using `_id` cursor

Maintain a single persisted cursor:

- `graderState.lastSeenId` (ObjectId)

Then the grader loops:

1. Query for new log records with:
   - `game: "mhs"`
   - `_id: { $gt: lastSeenId }`
   - `eventKey` in the set of known trigger keys
2. For each returned record:
   - map `eventKey` → (unit, point, rule)
   - run the rule for that `playerId`
   - write/update the resulting progress-point grade record
   - advance `lastSeenId`

Notes:

- This supports **backfill** by starting `lastSeenId` very small / null.
- This supports **near-real-time updates** by polling every few seconds.
- Trigger discovery uses `_id` (arrival order), while grading windows use `timestamp` (gameplay time).

### Indexes to keep this fast

On `logdata`:

- `{ game: 1, eventKey: 1, _id: 1 }` — fast “new triggers since cursor” scanning
- `{ game: 1, playerId: 1, eventKey: 1, timestamp: 1 }` — fast grading queries with sort/range on `timestamp`

## Grade storage for dashboard efficiency

Grades should be stored in a dedicated collection (example name: `progress_point_grades`).

A grade record should at minimum include:

- `game: "mhs"`
- `playerId`
- `unit`
- `point`
- `color: "green" | "yellow"`
- `computedAt: ISODate(...)` (server time)

### Strongly recommended additions (for teacher explanation modal + debugging)

Store structured evidence so we can explain “why yellow” later:

- `ruleId` (e.g. `u2p3_v1`) — internal versioning of the grading rule
- `triggerLogId` — the `_id` of the log record that caused the grade run
- `metrics` — values used to decide (counts, score, duration seconds, etc.)
- `reasonCode` — internal code for the decision (e.g. `TOO_MANY_TARGETS`, `MISSING_PASS_NODE`)

This allows teacher-friendly messages to be generated later without re-querying the entire log history.

## Replay / “load earlier save” handling

Replays can create both old and new events for the “same” unit/point.

Two approaches:

1. **Best**: add an attempt boundary to logs (`runId` / `attemptId` / `saveSessionId`) and include it in every grading query.
2. **If not available**: store window anchors when grading (e.g. `startLogId` and `endLogId`) so future work can avoid mixing attempts.

---

### Example Progress Point Grades Database Record

```js
rs0 [primary] mhsgrader> db.progress_point_grades.find({"playerId":"wenyi12@mhs.mhs"})
[
  {
    _id: ObjectId("6985675c4a9aa941aa3447b1"),
    game: 'mhs',
    playerId: 'wenyi12@mhs.mhs',
    grades: {
      u1p1: {
        color: 'green',
        computedAt: ISODate("2026-02-06T04:00:28.626Z"),
        ruleId: 'u1p1_v1'
      },
      u1p2: {
        color: 'green',
        computedAt: ISODate("2026-02-06T04:00:28.628Z"),
        ruleId: 'u1p2_v1'
      },
      u1p3: {
        color: 'green',
        computedAt: ISODate("2026-02-06T04:00:28.636Z"),
        ruleId: 'u1p3_v1'
      },
      u1p4: {
        color: 'green',
        computedAt: ISODate("2026-02-06T04:00:28.640Z"),
        ruleId: 'u1p4_v1'
      },
      u2p1: {
        color: 'yellow',
        computedAt: ISODate("2026-02-06T04:00:28.646Z"),
        ruleId: 'u2p1_v1',
        reasonCode: 'INCORRECT_PATH',
        metrics: { reason: 'Player took incorrect path in quest 21' }
      },
      u2p2: {
        color: 'green',
        computedAt: ISODate("2026-02-06T04:00:28.758Z"),
        ruleId: 'u2p2_v1'
      },
      u2p3: {
        color: 'green',
        computedAt: ISODate("2026-02-06T04:00:28.769Z"),
        ruleId: 'u2p3_v1'
      },
      u2p7: {
        color: 'green',
        computedAt: ISODate("2026-02-06T04:00:28.786Z"),
        ruleId: 'u2p7_v1'
      },
      u2p4: {
        color: 'yellow',
        computedAt: ISODate("2026-02-06T04:00:28.793Z"),
        ruleId: 'u2p4_v1',
        reasonCode: 'BAD_FEEDBACK',
        metrics: { reason: 'Player received negative feedback' }
      },
      u2p5: {
        color: 'yellow',
        computedAt: ISODate("2026-02-06T04:00:28.810Z"),
        ruleId: 'u2p5_v1',
        reasonCode: 'LOW_SCORE',
        metrics: { score: Long("1"), threshold: 4 }
      },
      u2p6: {
        color: 'yellow',
        computedAt: ISODate("2026-02-06T04:00:28.814Z"),
        ruleId: 'u2p6_v1',
        reasonCode: 'YELLOW_PATH',
        metrics: { reason: 'Player encountered yellow path nodes' }
      }
    },
    lastUpdated: ISODate("2026-02-06T04:00:28.814Z")
  }
]
rs0 [primary] mhsgrader> 
```