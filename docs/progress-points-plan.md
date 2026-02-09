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

Most unit/points have **one trigger eventKey**.

One exception currently uses **two triggers**:

- **Unit 2, Point 7** triggers on `DialogueNodeEvent:20:74` **or** `DialogueNodeEvent:20:75`.

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

# Grading Rules (mongosh scripts)

**All scripts below are intended to be run individually** (not as one combined script).

Each script:

- sets a `color` variable to `"green"` or `"yellow"`
- evaluates data in `db.logdata`
- always filters to `game: "mhs"`

> **Important:** All event key strings below have been verified to contain **no leading/trailing spaces**.

---

## Unit 1, Point 1

**Trigger:** `DialogueNodeEvent:31:29`

Always green.

```js
// unit 1, point 1
// Trigger eventKey: "DialogueNodeEvent:31:29"

let color = "green";
color;
```

---

## Unit 1, Point 2

**Trigger:** `DialogueNodeEvent:30:98`

Always green.

```js
// unit 1, point 2
// Trigger eventKey: "DialogueNodeEvent:30:98"

let color = "green";
color;
```

---

## Unit 1, Point 3

**Trigger:** `QuestActiveEvent:34`

Rule: if the player has *either* of the specified keys, it’s yellow; otherwise green.

```js
// unit 1, point 3
// Trigger eventKey: "QuestActiveEvent:34"

// Player identifier to evaluate
const playerId = "wenyi10@mhs.mhs";

const color =
  db.logdata.findOne({
    game: "mhs",
    playerId: playerId,
    eventKey: {
      $in: [
        "DialogueNodeEvent:70:25",
        "DialogueNodeEvent:70:33"
      ]
    }
  }) !== null
    ? "yellow"
    : "green";

color;
```

---

## Unit 1, Point 4

**Trigger:** `QuestFinishEvent:34`

Always green.

```js
// unit 1, point 4
// Trigger eventKey: "QuestFinishEvent:34"

let color = "green";
color;
```

---

## Unit 2, Point 1

**Trigger:** `QuestFinishEvent:21`

Rule:

- Must have `DialogueNodeEvent:68:29`
- Must have **none** of the yellow nodes

```js
// unit 2, point 1
// Trigger eventKey: "QuestFinishEvent:21"

const playerId = "wenyi10@mhs.mhs";

const color =
  (
    db.logdata.findOne({
      game: "mhs",
      playerId: playerId,
      eventKey: "DialogueNodeEvent:68:29"
    }) !== null &&

    db.logdata.findOne({
      game: "mhs",
      playerId: playerId,
      eventKey: {
        $in: [
          "DialogueNodeEvent:68:23",
          "DialogueNodeEvent:68:27",
          "DialogueNodeEvent:68:28",
          "DialogueNodeEvent:68:31"
        ]
      }
    }) === null
  )
    ? "green"
    : "yellow";

color;
```

---

## Unit 2, Point 2

**Trigger:** `DialogueNodeEvent:20:26`

This rule uses a **start/end window** and therefore includes a **duration guard** to handle bad client time values.

```js
// unit 2, point 2
// Trigger eventKey: "DialogueNodeEvent:20:26"

const playerId = "wenyi10@mhs.mhs";

// Start and end markers defining the evaluation window
const START_KEY = "DialogueNodeEvent:20:1";
const END_KEY   = "DialogueNodeEvent:19:46";

// Event keys we are counting inside the window
const TARGET_KEYS = [
  "DialogueNodeEvent:18:99",
  "DialogueNodeEvent:28:179",
  "DialogueNodeEvent:59:179",
  "DialogueNodeEvent:18:223",
  "DialogueNodeEvent:28:182",
  "DialogueNodeEvent:59:182",
  "DialogueNodeEvent:18:224",
  "DialogueNodeEvent:28:183",
  "DialogueNodeEvent:59:183"
];

// Duration guard configuration
const MAX_DURATION_SECONDS = 2 * 60 * 60; // 2 hours

// Default to green unless evidence forces yellow
let color = "green";

function parseTimestamp(ts) {
  const t = Date.parse(ts);
  return Number.isFinite(t) ? t : null;
}

// Step 1: earliest START
const startDoc = db.logdata
  .find({
    game: "mhs",
    playerId: playerId,
    eventKey: START_KEY,
    timestamp: { $type: "string" }
  })
  .sort({ timestamp: 1 })
  .limit(1)
  .next();

if (startDoc) {
  const startIso = startDoc.timestamp;
  const startMs  = parseTimestamp(startIso);

  // Step 2: earliest END at/after START
  const endDoc = db.logdata
    .find({
      game: "mhs",
      playerId: playerId,
      eventKey: END_KEY,
      timestamp: { $gte: startIso, $type: "string" }
    })
    .sort({ timestamp: 1 })
    .limit(1)
    .next();

  if (endDoc) {
    const endIso = endDoc.timestamp;
    const endMs  = parseTimestamp(endIso);

    // Step 3: duration sanity check
    if (
      startMs === null ||
      endMs === null ||
      endMs < startMs ||
      (endMs - startMs) / 1000 > MAX_DURATION_SECONDS
    ) {
      // Suspicious duration (bad clock / malformed time) → yellow
      color = "yellow";
    } else {
      // Step 4: count targets in window
      const countTargets = db.logdata.countDocuments({
        game: "mhs",
        playerId: playerId,
        eventKey: { $in: TARGET_KEYS },
        timestamp: { $gte: startIso, $lte: endIso }
      });

      // Step 5: apply rule (<= 1 green else yellow)
      color = (countTargets <= 1) ? "green" : "yellow";
    }
  }
}

color;
```

---

## Unit 2, Point 3

**Trigger:** `DialogueNodeEvent:22:18`

This rule uses a **start/end window** and therefore includes a **duration guard**.

```js
// unit 2, point 3
// Trigger eventKey: "DialogueNodeEvent:22:18"

const playerId = "wenyi11@mhs.mhs";

const START_KEY = "DialogueNodeEvent:20:33";
const END_KEY   = "DialogueNodeEvent:22:1";

const TARGET_KEYS = [
  "DialogueNodeEvent:18:225", "DialogueNodeEvent:28:185", "DialogueNodeEvent:59:185",
  "DialogueNodeEvent:28:184", "DialogueNodeEvent:28:191", "DialogueNodeEvent:59:184", "DialogueNodeEvent:59:191",
  "DialogueNodeEvent:18:226", "DialogueNodeEvent:18:227", "DialogueNodeEvent:28:186", "DialogueNodeEvent:59:186",
  "DialogueNodeEvent:18:228", "DialogueNodeEvent:28:187", "DialogueNodeEvent:59:187",
  "DialogueNodeEvent:18:229", "DialogueNodeEvent:28:188", "DialogueNodeEvent:59:188",
  "DialogueNodeEvent:18:230", "DialogueNodeEvent:28:180", "DialogueNodeEvent:59:180",
  "DialogueNodeEvent:18:233", "DialogueNodeEvent:28:192", "DialogueNodeEvent:59:192",
  "DialogueNodeEvent:18:234", "DialogueNodeEvent:28:193", "DialogueNodeEvent:59:193",
  "DialogueNodeEvent:18:235", "DialogueNodeEvent:28:194", "DialogueNodeEvent:59:194",
  "DialogueNodeEvent:18:236", "DialogueNodeEvent:18:237", "DialogueNodeEvent:28:190", "DialogueNodeEvent:59:190"
];

const MAX_DURATION_SECONDS = 2 * 60 * 60; // 2 hours

let color = "green";

function parseTimestamp(ts) {
  const t = Date.parse(ts);
  return Number.isFinite(t) ? t : null;
}

// Step 1: earliest START
const startDoc = db.logdata
  .find({
    game: "mhs",
    playerId: playerId,
    eventKey: START_KEY,
    timestamp: { $type: "string" }
  })
  .sort({ timestamp: 1 })
  .limit(1)
  .next();

if (startDoc) {
  const startIso = startDoc.timestamp;
  const startMs  = parseTimestamp(startIso);

  // Step 2: earliest END at/after START
  const endDoc = db.logdata
    .find({
      game: "mhs",
      playerId: playerId,
      eventKey: END_KEY,
      timestamp: { $gte: startIso, $type: "string" }
    })
    .sort({ timestamp: 1 })
    .limit(1)
    .next();

  if (endDoc) {
    const endIso = endDoc.timestamp;
    const endMs  = parseTimestamp(endIso);

    // Step 3: duration sanity check
    if (
      startMs === null ||
      endMs === null ||
      endMs < startMs ||
      (endMs - startMs) / 1000 > MAX_DURATION_SECONDS
    ) {
      color = "yellow";
    } else {
      // Step 4: count targets in window
      const countTargets = db.logdata.countDocuments({
        game: "mhs",
        playerId: playerId,
        eventKey: { $in: TARGET_KEYS },
        timestamp: { $gte: startIso, $lte: endIso }
      });

      // Step 5: apply rule (<= 6 green else yellow)
      color = (countTargets <= 6) ? "green" : "yellow";
    }
  }
}

color;
```

---

## Unit 2, Point 4

**Trigger:** `DialogueNodeEvent:23:17`

Rule: green only if success exists and no bad feedback exists.

```js
// unit 2, point 4
// Trigger eventKey: "DialogueNodeEvent:23:17"

const playerId = "wenyi10@mhs.mhs";

const success_key = "DialogueNodeEvent:74:21";

const bad_keys = [
  "DialogueNodeEvent:74:16",
  "DialogueNodeEvent:74:17",
  "DialogueNodeEvent:74:20",
  "DialogueNodeEvent:74:22"
];

let color = "yellow";

const has_success =
  db.logdata.findOne({
    game: "mhs",
    playerId: playerId,
    eventKey: success_key
  }) !== null;

const has_bad_feedback =
  db.logdata.findOne({
    game: "mhs",
    playerId: playerId,
    eventKey: { $in: bad_keys }
  }) !== null;

color = (has_success && !has_bad_feedback) ? "green" : "yellow";

color;
```

---

## Unit 2, Point 5

**Trigger:** `DialogueNodeEvent:23:42`

Rule: compute a score from positive and negative evidence.

- `score = pos_count - (neg_count / 3.0)`
- green if `score >= 4`

```js
// unit 2, point 5
// Trigger eventKey: "DialogueNodeEvent:23:42"

const playerId = "wenyi10@mhs.mhs";

const POS_KEYS = [
  "DialogueNodeEvent:23:140", "DialogueNodeEvent:23:142", "DialogueNodeEvent:23:143",
  "DialogueNodeEvent:23:146", "DialogueNodeEvent:23:147", "DialogueNodeEvent:23:148",
  "DialogueNodeEvent:23:165", "DialogueNodeEvent:23:166", "DialogueNodeEvent:23:167",
  "DialogueNodeEvent:23:168", "DialogueNodeEvent:23:169", "DialogueNodeEvent:23:170",
  "DialogueNodeEvent:23:172", "DialogueNodeEvent:23:173", "DialogueNodeEvent:23:174",
  "DialogueNodeEvent:23:175", "DialogueNodeEvent:23:177", "DialogueNodeEvent:23:178",
  "DialogueNodeEvent:23:179", "DialogueNodeEvent:23:180", "DialogueNodeEvent:23:181",
  "DialogueNodeEvent:23:182", "DialogueNodeEvent:23:183", "DialogueNodeEvent:23:184",
  "DialogueNodeEvent:23:185", "DialogueNodeEvent:23:186"
];

const NEG_KEYS = [
  "DialogueNodeEvent:26:137", "DialogueNodeEvent:26:144", "DialogueNodeEvent:26:145",
  "DialogueNodeEvent:26:187", "DialogueNodeEvent:26:188", "DialogueNodeEvent:26:189",
  "DialogueNodeEvent:26:191", "DialogueNodeEvent:26:192", "DialogueNodeEvent:26:193",
  "DialogueNodeEvent:26:194", "DialogueNodeEvent:26:195", "DialogueNodeEvent:26:196",
  "DialogueNodeEvent:26:197", "DialogueNodeEvent:26:198", "DialogueNodeEvent:26:199",
  "DialogueNodeEvent:26:200", "DialogueNodeEvent:26:201", "DialogueNodeEvent:26:202",
  "DialogueNodeEvent:26:203", "DialogueNodeEvent:26:204", "DialogueNodeEvent:26:205",
  "DialogueNodeEvent:26:206", "DialogueNodeEvent:26:207", "DialogueNodeEvent:26:208",
  "DialogueNodeEvent:26:209", "DialogueNodeEvent:26:210", "DialogueNodeEvent:26:211"
];

let color = "yellow";

const pos_count = db.logdata.countDocuments({
  game: "mhs",
  playerId: playerId,
  eventKey: { $in: POS_KEYS }
});

const neg_count = db.logdata.countDocuments({
  game: "mhs",
  playerId: playerId,
  eventKey: { $in: NEG_KEYS }
});

const score = pos_count - (neg_count / 3.0);

color = (score >= 4) ? "green" : "yellow";

color;
```

---

## Unit 2, Point 6

**Trigger:** `DialogueNodeEvent:18:284`

Rule:

- If pass node is missing → yellow
- Else if any yellow nodes exist → yellow
- Else → green

```js
// unit 2, point 6
// Trigger eventKey: "DialogueNodeEvent:18:284"

const playerId = "wenyi10@mhs.mhs";

const yellow_nodes = ["dialogue:20:44", "dialogue:20:45"];
const pass_node_key = "dialogue:20:43";

let color = "yellow";

const passDoc = db.logdata.findOne({
  game: "mhs",
  playerId: playerId,
  eventKey: pass_node_key
});

if (!passDoc) {
  color = "yellow";
} else {
  const yellowDoc = db.logdata.findOne({
    game: "mhs",
    playerId: playerId,
    eventKey: { $in: yellow_nodes }
  });

  color = yellowDoc ? "yellow" : "green";
}

color;
```

---

## Unit 2, Point 7

**Trigger:** `DialogueNodeEvent:20:74` **or** `DialogueNodeEvent:20:75`

Rule: green only if success exists and negative count is within limit.

```js
// unit 2, point 7
// Trigger eventKey: "DialogueNodeEvent:20:74" OR "DialogueNodeEvent:20:75"

const playerId = "wenyi11@mhs.mhs";

const SUCCESS_KEY = "DialogueNodeEvent:27:7";

const NEG_KEYS = [
  "DialogueNodeEvent:27:11", "DialogueNodeEvent:27:12", "DialogueNodeEvent:27:13", "DialogueNodeEvent:27:14",
  "DialogueNodeEvent:27:15", "DialogueNodeEvent:27:16", "DialogueNodeEvent:27:17", "DialogueNodeEvent:27:18",
  "DialogueNodeEvent:27:19", "DialogueNodeEvent:27:20", "DialogueNodeEvent:27:21", "DialogueNodeEvent:27:22",
  "DialogueNodeEvent:27:23", "DialogueNodeEvent:27:24", "DialogueNodeEvent:27:25", "DialogueNodeEvent:27:26",
  "DialogueNodeEvent:27:27", "DialogueNodeEvent:27:28", "DialogueNodeEvent:27:29", "DialogueNodeEvent:27:30"
];

let color = "yellow";

const has_success =
  db.logdata.findOne({
    game: "mhs",
    playerId: playerId,
    eventKey: SUCCESS_KEY
  }) !== null;

const neg_count = db.logdata.countDocuments({
  game: "mhs",
  playerId: playerId,
  eventKey: { $in: NEG_KEYS }
});

color = (has_success && neg_count <= 3) ? "green" : "yellow";

color;
```

---