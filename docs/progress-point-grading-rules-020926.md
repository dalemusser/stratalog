
# Grading Rules (mongosh scripts)

**All scripts below are intended to be run individually** (not as one combined script).

Each script:

- sets a `color` variable to `"green"` or `"yellow"`
- evaluates data in `db.logdata`
- always filters to `game: "mhs"`

> **Important:** All event key strings below have been verified to contain **no leading/trailing spaces**.

---

## Unit 1, Point 1

**Title**: Getting Your Space Legs

**Trigger:** `DialogueNodeEvent:31:29`

### Data Analytics Script

```python
If has eventKey:
    color = "green"
```

Rule statement (matches analytics exactly)
- If the player has a log entry with eventKey = DialogueNodeEvent:31:29 → green

The following script matches analytics intent.

```js
// Unit 1, Point 1 — Analytics-matching script
// Trigger eventKey: "DialogueNodeEvent:31:29"

const color = "green";
color;
```

### Unit 1 Point 1 Production Grading Script

Attempt-based standalone production script (latest attempt only)

Rule statement (latest attempt)
- Find the latest trigger DialogueNodeEvent:31:29 for the player.
- If it exists → green

Note: Because the rule has no other evidence keys, "attempt-based" doesn’t really change anything — the latest trigger existing is sufficient.

```js
// Unit 1, Point 1 — Attempt-based standalone production script (latest attempt)
// Trigger eventKey: "DialogueNodeEvent:31:29"

const color = "green";
color;
```

---

## Unit 1, Point 2

**Title**: Info and Intros

**Trigger:** `DialogueNodeEvent:30:98`

### Data Analytics Script

```python
If has eventKey:
    color = 1 # (“green”)
```

Rule statement (matches analytics exactly)
- If the player has a log entry with
eventKey = DialogueNodeEvent:30:98
→ green

There are no additional success/failure keys. This is a pure “did they reach this node?” check.

The following script matches analytics intent.

```js
// Unit 1, Point 2 — Analytics-matching script
// Trigger eventKey: "DialogueNodeEvent:30:98"

const color = "green";
color;
```

### Unit 1 Point 2 Production Grading Script

Attempt-based standalone production script (latest attempt only)

Rule statement (latest attempt)
- Find the latest DialogueNodeEvent:30:98 for the player.
- If it exists → green

Because there are no other evidence keys, "latest attempt" semantics do not change the outcome — existence of the most recent trigger is sufficient.

```js
// Unit 1, Point 2 — Attempt-based standalone production script
// Trigger eventKey: "DialogueNodeEvent:30:98"

const color = "green";
color;
```

---

## Unit 1, Point 3

**Title**: Defend the Expedition (Argumentation: Identify a claim)

**Trigger:** `questActiveEvent:34`

### Data Analytics Script

```python
If has eventKey:
     has_yellow_trigger = coll.count_documents({
    "playerId": pid,
    "eventKey": {"$in": [
    "DialogueNodeEvent:70:25",
    "DialogueNodeEvent:70:33"
     ]}
     }) > 0
    if has_yellow_trigger:
        color = 2 # ("yellow")
    else:
        color = 1 # ("green")

```

Rule statement (matches analytics exactly)
- If the player has any of the following:
- DialogueNodeEvent:70:25
- DialogueNodeEvent:70:33
→ yellow
- Otherwise → green

Important:
The analytics script does not require the trigger to be inside a window. It simply checks for the existence of the yellow nodes anywhere in the player’s history once the trigger condition is met.

This is a lifetime existence rule, not attempt-based.

The following script matches analytics intent.

```js
// Unit 1, Point 3 — Analytics-matching script
// Trigger eventKey: "questActiveEvent:34"

const playerId = "wenyi10@mhs.mhs";

const YELLOW_KEYS = [
  "DialogueNodeEvent:70:25",
  "DialogueNodeEvent:70:33"
];

const hasYellow =
  db.logdata.findOne({
    game: "mhs",
    playerId: playerId,
    eventKey: { $in: YELLOW_KEYS }
  }) !== null;

const color = hasYellow ? "yellow" : "green";
color;
```

### Unit 1 Point 3 Production Grading Script

Attempt-based standalone production script (latest attempt only)

Rule statement (latest attempt)
- Define attempt window as:
- previous questActiveEvent:34 (exclusive)
- latest questActiveEvent:34 (inclusive)
- If any yellow node (70:25, 70:33) exists within that attempt window → yellow
- Otherwise → green
- If no trigger exists → yellow

This prevents a past failed attempt from permanently poisoning a later clean attempt.

```js
// Unit 1, Point 3 — Attempt-based standalone production script
// Trigger eventKey: "questActiveEvent:34"

const playerId = "wenyi10@mhs.mhs";

const TRIGGER_KEY = "questActiveEvent:34";

const YELLOW_KEYS = [
  "DialogueNodeEvent:70:25",
  "DialogueNodeEvent:70:33"
];

// 1) Latest trigger (end anchor)
const latestTrigger = db.logdata.findOne(
  { game: "mhs", playerId: playerId, eventKey: TRIGGER_KEY },
  { sort: { _id: -1 } }
);

if (!latestTrigger) {
  "yellow";
} else {
  // 2) Previous trigger (attempt boundary)
  const prevTrigger = db.logdata.findOne(
    {
      game: "mhs",
      playerId: playerId,
      eventKey: TRIGGER_KEY,
      _id: { $lt: latestTrigger._id }
    },
    { sort: { _id: -1 } }
  );

  const windowStartId = prevTrigger ? prevTrigger._id : ObjectId("000000000000000000000000");
  const windowEndId = latestTrigger._id;

  const hasYellow =
    db.logdata.findOne({
      game: "mhs",
      playerId: playerId,
      eventKey: { $in: YELLOW_KEYS },
      _id: { $gt: windowStartId, $lte: windowEndId }
    }) !== null;

  hasYellow ? "yellow" : "green";
}
```

#### Subtle architectural note

This is a perfect example of why attempt-based grading matters.

Without windowing:
- One mistake early in the game play = permanently yellow.

With windowing:
- The student can improve, replay, and earn green on a later attempt.

That’s not just a database choice — that’s a learning philosophy choice.

---

## Unit 1, Point 4

**Title:** Unexpected Turbulence

**Trigger:** `DialogueNodeEvent:33:19`

### Data Analytics Script

```python
If has eventKey:
    color = 1 # (“green”)

```

Rule statement (matches analytics exactly)
- If the player has a log entry with eventKey = DialogueNodeEvent:33:19 → green

The following script matches analytics intent.

```js
// Unit 1, Point 4 — Analytics-matching script
// Trigger eventKey: "DialogueNodeEvent:33:19"

const color = "green";
color;
```

### Unit 1 Point 4 Production Grading Script

Attempt-based standalone production script (latest attempt only)

Rule statement (latest attempt)
- Find the latest DialogueNodeEvent:33:19 for the player.
- If it exists → green

(There are no other evidence keys, so attempt-based semantics don’t change the outcome.)

```js
// Unit 1, Point 4 — Attempt-based standalone production script
// Trigger eventKey: "DialogueNodeEvent:33:19"

const color = "green";
color;
```

---

## Unit 2, Point 1

**Title**: Escape the Ruins + Topographic Glyph (Identify Topographic map features)

**Trigger:** `questFinishEvent:21`

### Data Analytics Script

```python
yellow_nodes = ["DialogueNodeEvent:68:23", "DialogueNodeEvent:68:27", "DialogueNodeEvent:68:28", "DialogueNodeEvent:68:31"]
success_node = "DialogueNodeEvent:68:29"
if has eventKey:
    has_29 = coll.find_one({
"playerId": pid, 
"eventKey": success_node}) is not None
    has_any_yellow = coll.find_one({
"playerId": pid, 
"eventKey": {"$in": yellow_nodes}}) is not None
    if has_29 and not has_any_yellow:
        color = 1 # ("green")
    else:
        color = 2 # ("yellow")

```

Rule statement (matches analytics exactly)
- Must have DialogueNodeEvent:68:29 (success node)
- Must have none of:
 - DialogueNodeEvent:68:23
 - DialogueNodeEvent:68:27
 - DialogueNodeEvent:68:28
 - DialogueNodeEvent:68:31
 - If success present and no yellow nodes → green
 - Otherwise → yellow

Analytics logic is lifetime-based (no window).

The following script matches analytics intent.

```js
// Unit 2, Point 1 — Analytics-matching script
// Trigger eventKey: "questFinishEvent:21"

const playerId = "wenyi10@mhs.mhs";

const successKey = "DialogueNodeEvent:68:29";

const yellowNodes = [
  "DialogueNodeEvent:68:23",
  "DialogueNodeEvent:68:27",
  "DialogueNodeEvent:68:28",
  "DialogueNodeEvent:68:31"
];

const hasSuccess =
  db.logdata.findOne({
    game: "mhs",
    playerId: playerId,
    eventKey: successKey
  }) !== null;

const hasAnyYellow =
  db.logdata.findOne({
    game: "mhs",
    playerId: playerId,
    eventKey: { $in: yellowNodes }
  }) !== null;

const color =
  hasSuccess && !hasAnyYellow
    ? "green"
    : "yellow";

color;
```

### Unit 2 Point 1 Production Grading Script

Attempt-based standalone production script (latest attempt only)

Rule statement (latest attempt)
- Define attempt window as:
 - previous questFinishEvent:21 (exclusive)
 - latest questFinishEvent:21 (inclusive)
- Within that window:
 - Must have DialogueNodeEvent:68:29
 - Must have none of the yellow nodes
- If success and no yellow nodes → green
- Else → yellow
- If no latest trigger → white (but in production this script only runs when trigger exists)

```js
// Unit 2, Point 1 — Attempt-based standalone production grading script
// Trigger eventKey: "questFinishEvent:21"

const playerId = "wenyi10@mhs.mhs";

const TRIGGER_KEY = "questFinishEvent:21";

const successKey = "DialogueNodeEvent:68:29";

const yellowNodes = [
  "DialogueNodeEvent:68:23",
  "DialogueNodeEvent:68:27",
  "DialogueNodeEvent:68:28",
  "DialogueNodeEvent:68:31"
];

// 1) Latest trigger (end anchor)
const latestTrigger = db.logdata.findOne(
  { game: "mhs", playerId: playerId, eventKey: TRIGGER_KEY },
  { sort: { _id: -1 } }
);

if (!latestTrigger) {
  "yellow";
} else {

  // 2) Previous trigger (attempt boundary)
  const prevTrigger = db.logdata.findOne(
    {
      game: "mhs",
      playerId: playerId,
      eventKey: TRIGGER_KEY,
      _id: { $lt: latestTrigger._id }
    },
    { sort: { _id: -1 } }
  );

  const windowStartId = prevTrigger ? prevTrigger._id : ObjectId("000000000000000000000000");
  const windowEndId = latestTrigger._id;

  const hasSuccess =
    db.logdata.findOne({
      game: "mhs",
      playerId: playerId,
      eventKey: successKey,
      _id: { $gt: windowStartId, $lte: windowEndId }
    }) !== null;

  const hasAnyYellow =
    db.logdata.findOne({
      game: "mhs",
      playerId: playerId,
      eventKey: { $in: yellowNodes },
      _id: { $gt: windowStartId, $lte: windowEndId }
    }) !== null;

  hasSuccess && !hasAnyYellow ? "green" : "yellow";
}
```

---

## Unit 2, Point 2

**Title**: Foraged Forging + Finding Toppo (Find geographic locations on topographic map)

**Trigger:** `DialogueNodeEvent:20:26`

Rule statement (matches analytics exactly)
- Find the earliest START_KEY = questFinishEvent:21 (by timestamp ascending).
- Find the earliest END_KEY = DialogueNodeEvent:20:26 with timestamp >= start.timestamp (by timestamp ascending).
- Count how many events with eventKey in TARGET_KEYS occur with timestamp between start and end (inclusive).
- If count_targets <= 1 → green
- Else → yellow

Notes:
- This is a windowed rule and must use client timestamp.
- Analytics intent is "earliest start → earliest matching end," not "latest attempt."

The following script matches analytics intent.

```js
// Unit 2, Point 2 — Analytics-matching script
// Trigger eventKey: "DialogueNodeEvent:20:26"

const playerId = "wenyi10@mhs.mhs";

const START_KEY = "questFinishEvent:21";
const END_KEY = "DialogueNodeEvent:20:26";

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

// 1) Earliest start by timestamp
const startDoc = db.logdata.findOne(
  { game: "mhs", playerId: playerId, eventKey: START_KEY },
  { sort: { timestamp: 1 } }
);

if (!startDoc || !startDoc.timestamp) {
  "yellow";
} else {
  const startIso = startDoc.timestamp;

  // 2) Earliest end after start by timestamp
  const endDoc = db.logdata.findOne(
    {
      game: "mhs",
      playerId: playerId,
      eventKey: END_KEY,
      timestamp: { $gte: startIso }
    },
    { sort: { timestamp: 1 } }
  );

  if (!endDoc || !endDoc.timestamp) {
    "yellow";
  } else {
    const endIso = endDoc.timestamp;

    // 3) Count targets in [startIso, endIso]
    const countTargets = db.logdata.countDocuments({
      game: "mhs",
      playerId: playerId,
      eventKey: { $in: TARGET_KEYS },
      timestamp: { $gte: startIso, $lte: endIso }
    });

    countTargets <= 1 ? "green" : "yellow";
  }
}
```

### Unit 2 Point 2 Production Grading Script

Attempt-based standalone production script (latest attempt only)

Rule statement (latest attempt)
- Let end anchor be the latest trigger DialogueNodeEvent:20:26 (by _id).
- Find the most recent START_KEY = questFinishEvent:21 that occurred before that end (by _id), and use its timestamp as start time.
- Count target keys between start and end (inclusive), fenced by _id range to isolate the attempt.
- If count_targets <= 1 → green else yellow.
- If start or end missing → yellow.

This aligns with "teacher rewind / replay" semantics: grade the most recently completed attempt.

```js
// Unit 2, Point 2 — Attempt-based standalone production grading script (latest attempt)
// Trigger eventKey: "DialogueNodeEvent:20:26"

const playerId = "wenyi10@mhs.mhs";

const END_KEY = "DialogueNodeEvent:20:26";     // trigger
const START_KEY = "questFinishEvent:21";

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

// 1) Latest end trigger by arrival order
const endDoc = db.logdata.findOne(
  { game: "mhs", playerId: playerId, eventKey: END_KEY },
  { sort: { _id: -1 } }
);

if (!endDoc || !endDoc.timestamp) {
  "yellow";
} else {
  const endIso = endDoc.timestamp;

  // 2) Latest start before this end (same attempt) by arrival order
  const startDoc = db.logdata.findOne(
    {
      game: "mhs",
      playerId: playerId,
      eventKey: START_KEY,
      _id: { $lte: endDoc._id }
    },
    { sort: { _id: -1 } }
  );

  if (!startDoc || !startDoc.timestamp) {
    "yellow";
  } else {
    const startIso = startDoc.timestamp;

    // 3) Count targets within [startIso, endIso] and fence by _id window
    const countTargets = db.logdata.countDocuments({
      game: "mhs",
      playerId: playerId,
      eventKey: { $in: TARGET_KEYS },
      timestamp: { $gte: startIso, $lte: endIso },
      _id: { $gte: startDoc._id, $lte: endDoc._id }
    });

    countTargets <= 1 ? "green" : "yellow";
  }
}
```

---

## Unit 2, Point 3

**Title:** Finding Tera & Aryn (Find geographic locations on topographic map)

**Trigger:** `DialogueNodeEvent:22:18`

### Data Analytics Script

```python
START_KEY = " DialogueNodeEvent:20:33"   
END_KEY   = " DialogueNodeEvent:22:1"
TARGET_KEYS = [
    " DialogueNodeEvent:18:225", "DialogueNodeEvent:28:185", "DialogueNodeEvent:59:185",
    "DialogueNodeEvent:28:184", "DialogueNodeEvent:28:191", "DialogueNodeEvent:59:184", "DialogueNodeEvent:59:191",
    "DialogueNodeEvent:18:226", "DialogueNodeEvent:18:227", "DialogueNodeEvent:28:186", "DialogueNodeEvent:59:186",
    "DialogueNodeEvent:18:228", "DialogueNodeEvent:28:187", "DialogueNodeEvent:59:187",
    "DialogueNodeEvent:18:229", "DialogueNodeEvent:28:188", "DialogueNodeEvent:59:188",
    "DialogueNodeEvent:18:230", "DialogueNodeEvent:28:180", "DialogueNodeEvent:59:180",
    "DialogueNodeEvent:18:233", "DialogueNodeEvent:28:192", "DialogueNodeEvent:59:192",
    "DialogueNodeEvent:18:234", "DialogueNodeEvent:28:193", "DialogueNodeEvent:59:193",
    "DialogueNodeEvent:18:235", "DialogueNodeEvent:28:194", "DialogueNodeEvent:59:194",
    "DialogueNodeEvent:18:236", "DialogueNodeEvent:18:237", "DialogueNodeEvent:28:190", "DialogueNodeEvent:59:190"
]

If has eventKey:
    start_doc = coll.find_one(
        {
            "playerId": pid,
            "eventKey": START_KEY
        },
        sort=[("timestamp", 1)])

     start_ts = datetime.fromisoformat(start_doc["timestamp"].replace("Z", "+00:00"))
     end_doc = coll.find_one(
        {
            "playerId": pid,
            "eventKey": END_KEY,
            "timestamp": {"$gte": start_doc["timestamp"]}
        },
        sort=[("timestamp", 1)]
)

    end_ts = datetime.fromisoformat(end_doc["timestamp"].replace("Z", "+00:00"))
    start_iso = start_ts.isoformat().replace("+00:00", "Z")
    end_iso   = end_ts.isoformat().replace("+00:00", "Z")
    count_targets = coll.count_documents({
        "playerId": pid,
        "eventKey": {"$in": TARGET_KEYS},
        "timestamp": {
            "$gte": start_iso,
            "$lte": end_iso
        }
})
    if count_targets <= 6:
        color = 1 #("green")
    else:
        color = 2 #("yellow")
```

Rule statement (Unit 2, Point 3)
- Find the earliest START_KEY = "DialogueNodeEvent:20:33" for the player (sort by timestamp ascending).
- Find the earliest END_KEY = "DialogueNodeEvent:22:1" that occurs at or after the start time (sort by timestamp ascending).
- Count how many events with eventKey in TARGET_KEYS occur with timestamp between start and end (inclusive).
- If count_targets <= 6 → green
- Else → yellow

Notes
- Window logic uses client timestamp (string ISO time) for correct gameplay timing.
- If start or end is missing, treat as yellow (needs review / incomplete window).
- Trim issue: analytics team provided leading spaces; this version assumes eventKeys in DB do not have leading spaces.

The following script matches analytics intent.

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

### Unit 2 Point 3 Production Grading Script 

The following is the production script for Unit 2 Point 3 that has the ability to handle replays.

```js
// Unit 2, Point 3 — Standalone production grading script
// Trigger eventKey (end-of-point): "DialogueNodeEvent:22:18"
//
// Rule statement:
// - Let endDoc be the latest trigger record for this player (by _id).
// - Let startDoc be the latest START_KEY record with _id <= endDoc._id.
// - Count TARGET_KEYS records between startDoc.timestamp and endDoc.timestamp (inclusive),
//   and fence by _id range [startDoc._id, endDoc._id].
// - If count_targets <= 6 => green else yellow.
// - If endDoc or startDoc missing => yellow.

const playerId = "wenyi10@mhs.mhs";

const TRIGGER_KEY = "DialogueNodeEvent:22:18";
const START_KEY = "DialogueNodeEvent:20:33";

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

// 1) Latest trigger (end anchor) by arrival order
const endDoc = db.logdata.findOne(
  { game: "mhs", playerId: playerId, eventKey: TRIGGER_KEY },
  { sort: { _id: -1 } }
);

if (!endDoc || !endDoc.timestamp) {
  "yellow";
} else {

  // 2) Latest start before that end (same attempt) by arrival order
  const startDoc = db.logdata.findOne(
    {
      game: "mhs",
      playerId: playerId,
      eventKey: START_KEY,
      _id: { $lte: endDoc._id }
    },
    { sort: { _id: -1 } }
  );

  if (!startDoc || !startDoc.timestamp) {
    "yellow";
  } else {

    const startIso = startDoc.timestamp;
    const endIso = endDoc.timestamp;

    // 3) Count targets within the bounded attempt window
    const countTargets = db.logdata.countDocuments({
      game: "mhs",
      playerId: playerId,
      eventKey: { $in: TARGET_KEYS },
      timestamp: { $gte: startIso, $lte: endIso },
      _id: { $gte: startDoc._id, $lte: endDoc._id }
    });

    countTargets <= 6 ? "green" : "yellow";
  }
}
```

---

## Unit 2, Point 4

**Title:** Investigate the Temple & Watershed Glyph (Relate watershed size to flow rate through main river)

**Trigger:** `DialogueNodeEvent:23:17`

### Data Analytics Script

```python
success_key = " DialogueNodeEvent:74:21"

bad_keys = [
    " DialogueNodeEvent:74:16",
    " DialogueNodeEvent:74:17",
    " DialogueNodeEvent:74:20",
    " DialogueNodeEvent:74:22"
]

If has eventKey:
    has_74_21 = coll.find_one(
        {
            "playerId": pid,
            "eventKey": success_key
        }
) is not None
    has_bad_feedback = coll.find_one(
        {
            "playerId": pid,
            "eventKey": {"$in": bad_keys}
        }
) is not None
    if has_74_21 and not has_bad_feedback:
        color = 1 #("green")
    else:
        color = 2 #("yellow")

```

Rule statement
- Must have DialogueNodeEvent:74:21 (success)
- Must have none of the bad feedback nodes:
- DialogueNodeEvent:74:16
- DialogueNodeEvent:74:17
- DialogueNodeEvent:74:20
- DialogueNodeEvent:74:22
- If success is present and no bad feedback is present → green
- Otherwise → yellow

Note: The data analytics script shows leading spaces in several keys. Based on the constraint (“no leading/trailing spaces”), the keys below are trimmed.

The following script matches analytics intent.

```js
// Unit 2, Point 4
// Trigger eventKey: "DialogueNodeEvent:23:17"

// Player identifier to evaluate
const playerId = "wenyi10@mhs.mhs";

const successKey = "DialogueNodeEvent:74:21";

const badKeys = [
  "DialogueNodeEvent:74:16",
  "DialogueNodeEvent:74:17",
  "DialogueNodeEvent:74:20",
  "DialogueNodeEvent:74:22"
];

const hasSuccess =
  db.logdata.findOne({
    game: "mhs",
    playerId: playerId,
    eventKey: successKey
  }) !== null;

const hasBadFeedback =
  db.logdata.findOne({
    game: "mhs",
    playerId: playerId,
    eventKey: { $in: badKeys }
  }) !== null;

const color =
  hasSuccess && !hasBadFeedback
    ? "green"
    : "yellow";

color;
```

### Unit 2 Point 4 Production Grading Script 

The following is the production script for Unit 2 Point 4 that has the ability to handle replays.

```js
// Unit 2, Point 4 — Standalone replay-aware grading (latest attempt heuristic)
// Trigger eventKey: "DialogueNodeEvent:23:17"

const playerId = "wenyi10@mhs.mhs";

const TRIGGER_KEY = "DialogueNodeEvent:23:17";

const successKey = "DialogueNodeEvent:74:21";
const badKeys = [
  "DialogueNodeEvent:74:16",
  "DialogueNodeEvent:74:17",
  "DialogueNodeEvent:74:20",
  "DialogueNodeEvent:74:22"
];

// 1) Latest trigger
const latestTrigger = db.logdata.findOne(
  { game: "mhs", playerId: playerId, eventKey: TRIGGER_KEY },
  { sort: { _id: -1 } }
);

if (!latestTrigger) {
  "yellow";
} else {
  // 2) Previous trigger (defines prior attempt boundary)
  const prevTrigger = db.logdata.findOne(
    { game: "mhs", playerId: playerId, eventKey: TRIGGER_KEY, _id: { $lt: latestTrigger._id } },
    { sort: { _id: -1 } }
  );

  const windowStartId = prevTrigger ? prevTrigger._id : ObjectId("000000000000000000000000");
  const windowEndId = latestTrigger._id;

  // 3) Check success/bad within this attempt window
  const hasSuccess =
    db.logdata.findOne({
      game: "mhs",
      playerId: playerId,
      eventKey: successKey,
      _id: { $gt: windowStartId, $lte: windowEndId }
    }) !== null;

  const hasBad =
    db.logdata.findOne({
      game: "mhs",
      playerId: playerId,
      eventKey: { $in: badKeys },
      _id: { $gt: windowStartId, $lte: windowEndId }
    }) !== null;

  hasSuccess && !hasBad ? "green" : "yellow";
}
```

Notes:
- If Point 4 is lifetime/ever → your original script is already “production.”
- If Point 4 is per-attempt (latest attempt) → yes, you need a replay-aware version like above or (better) you need the game to emit a clean START_KEY / attemptId so the attempt window is explicit.

Given the possible implementation of teacher rewind plan, I’d strongly lean toward attempt-based grading.

---

## Unit 2, Point 5

**Title:** Getting the Band Back Together (Identify the parts of a scientific argument)

**Trigger:** `DialogueNodeEvent:23:42`

### Original Data Analytics Script

```python
POS_KEYS = [
"DialogueNodeEvent:26:165", 
"DialogueNodeEvent:26:166",
"DialogueNodeEvent:26:167",
"DialogueNodeEvent:26:168",
"DialogueNodeEvent:26:169",
"DialogueNodeEvent:26:170",
"DialogueNodeEvent:26:171",
"DialogueNodeEvent:26:172",
"DialogueNodeEvent:26:173",
"DialogueNodeEvent:26:174",
"DialogueNodeEvent:26:175",
"DialogueNodeEvent:26:176",
"DialogueNodeEvent:26:177",
"DialogueNodeEvent:26:178",
"DialogueNodeEvent:26:179",
"DialogueNodeEvent:26:180",
"DialogueNodeEvent:26:181",
"DialogueNodeEvent:26:182",
"DialogueNodeEvent:26:183",
"DialogueNodeEvent:26:184",
"DialogueNodeEvent:26:185",
"DialogueNodeEvent:26:186"
]

NEG_KEYS = [
    "DialogueNodeEvent:26:187",
"DialogueNodeEvent:26:188",
"DialogueNodeEvent:26:189",
"DialogueNodeEvent:26:190",
"DialogueNodeEvent:26:191",
"DialogueNodeEvent:26:192",
"DialogueNodeEvent:26:193",
"DialogueNodeEvent:26:194",
"DialogueNodeEvent:26:195",
"DialogueNodeEvent:26:196",
"DialogueNodeEvent:26:197",
"DialogueNodeEvent:26:198",
"DialogueNodeEvent:26:199",
"DialogueNodeEvent:26:200",
"DialogueNodeEvent:26:201",
"DialogueNodeEvent:26:202",
"DialogueNodeEvent:26:203",
"DialogueNodeEvent:26:204",
"DialogueNodeEvent:26:205",
"DialogueNodeEvent:26:206",
"DialogueNodeEvent:26:207",
"DialogueNodeEvent:26:208",
"DialogueNodeEvent:26:209",
"DialogueNodeEvent:26:210",
"DialogueNodeEvent:26:211"
]

If has eventKey:
    pos_count = coll.count_documents({
        "playerId": pid,
        "eventKey": {"$in": POS_KEYS}
    })
    neg_count = coll.count_documents({
        "playerId": pid,
        "eventKey": {"$in": NEG_KEYS}
    })
    score = pos_count - (neg_count / 3.0)
     if score >= 4:
          color = 1 #("green")
    else:
          color = 2 #("yellow")

```

Rule statement
- Count how many log entries for the player have eventKey in POS_KEYS → pos_count
- Count how many log entries for the player have eventKey in NEG_KEYS → neg_count
- Compute: score = pos_count - (neg_count / 3.0)
- If score >= 4 → green
- Else → yellow

For "latest attempt only," we’ll need an attempt window (start key, attemptId, or trigger-to-trigger windowing). This conversion matches the analytics intent exactly.

The following script matches analytics intent.

```js
// Unit 2, Point 5
// Trigger eventKey: "DialogueNodeEvent:23:42"

// Player identifier to evaluate
const playerId = "wenyi10@mhs.mhs";

const POS_KEYS = [
  "DialogueNodeEvent:26:165",
  "DialogueNodeEvent:26:166",
  "DialogueNodeEvent:26:167",
  "DialogueNodeEvent:26:168",
  "DialogueNodeEvent:26:169",
  "DialogueNodeEvent:26:170",
  "DialogueNodeEvent:26:171",
  "DialogueNodeEvent:26:172",
  "DialogueNodeEvent:26:173",
  "DialogueNodeEvent:26:174",
  "DialogueNodeEvent:26:175",
  "DialogueNodeEvent:26:176",
  "DialogueNodeEvent:26:177",
  "DialogueNodeEvent:26:178",
  "DialogueNodeEvent:26:179",
  "DialogueNodeEvent:26:180",
  "DialogueNodeEvent:26:181",
  "DialogueNodeEvent:26:182",
  "DialogueNodeEvent:26:183",
  "DialogueNodeEvent:26:184",
  "DialogueNodeEvent:26:185",
  "DialogueNodeEvent:26:186"
];

const NEG_KEYS = [
  "DialogueNodeEvent:26:187",
  "DialogueNodeEvent:26:188",
  "DialogueNodeEvent:26:189",
  "DialogueNodeEvent:26:190",
  "DialogueNodeEvent:26:191",
  "DialogueNodeEvent:26:192",
  "DialogueNodeEvent:26:193",
  "DialogueNodeEvent:26:194",
  "DialogueNodeEvent:26:195",
  "DialogueNodeEvent:26:196",
  "DialogueNodeEvent:26:197",
  "DialogueNodeEvent:26:198",
  "DialogueNodeEvent:26:199",
  "DialogueNodeEvent:26:200",
  "DialogueNodeEvent:26:201",
  "DialogueNodeEvent:26:202",
  "DialogueNodeEvent:26:203",
  "DialogueNodeEvent:26:204",
  "DialogueNodeEvent:26:205",
  "DialogueNodeEvent:26:206",
  "DialogueNodeEvent:26:207",
  "DialogueNodeEvent:26:208",
  "DialogueNodeEvent:26:209",
  "DialogueNodeEvent:26:210",
  "DialogueNodeEvent:26:211"
];

const posCount = db.logdata.countDocuments({
  game: "mhs",
  playerId: playerId,
  eventKey: { $in: POS_KEYS }
});

const negCount = db.logdata.countDocuments({
  game: "mhs",
  playerId: playerId,
  eventKey: { $in: NEG_KEYS }
});

// Keep it explicitly floating-point like the Python (neg_count / 3.0)
const score = posCount - (negCount / 3.0);

const color = score >= 4 ? "green" : "yellow";

color;
```

### Unit 2 Point 5 Production Grading Script 

Unit 2, Point 5 — Attempt-based (standalone) Production Script

Trigger eventKey: DialogueNodeEvent:23:42

Rule statement (attempt-based)
- Define the current attempt window as the log records between:
- the previous trigger DialogueNodeEvent:23:42 (exclusive), and
- the latest trigger DialogueNodeEvent:23:42 (inclusive),
for the same player (using _id ordering).
- Within that window:
- pos_count = count of records with eventKey in POS_KEYS
- neg_count = count of records with eventKey in NEG_KEYS
- score = pos_count - (neg_count / 3.0)
- If score >= 4 → green
- Else → yellow
- If the latest trigger doesn’t exist → yellow

This mirrors the attempt-window heuristic we used earlier: trigger-to-trigger defines an attempt when no explicit START_KEY / attemptId exists.

```js
// Unit 2, Point 5 — Attempt-based standalone production grading script
// Trigger eventKey: "DialogueNodeEvent:23:42"

const playerId = "wenyi10@mhs.mhs";

const TRIGGER_KEY = "DialogueNodeEvent:23:42";

const POS_KEYS = [
  "DialogueNodeEvent:26:165",
  "DialogueNodeEvent:26:166",
  "DialogueNodeEvent:26:167",
  "DialogueNodeEvent:26:168",
  "DialogueNodeEvent:26:169",
  "DialogueNodeEvent:26:170",
  "DialogueNodeEvent:26:171",
  "DialogueNodeEvent:26:172",
  "DialogueNodeEvent:26:173",
  "DialogueNodeEvent:26:174",
  "DialogueNodeEvent:26:175",
  "DialogueNodeEvent:26:176",
  "DialogueNodeEvent:26:177",
  "DialogueNodeEvent:26:178",
  "DialogueNodeEvent:26:179",
  "DialogueNodeEvent:26:180",
  "DialogueNodeEvent:26:181",
  "DialogueNodeEvent:26:182",
  "DialogueNodeEvent:26:183",
  "DialogueNodeEvent:26:184",
  "DialogueNodeEvent:26:185",
  "DialogueNodeEvent:26:186"
];

const NEG_KEYS = [
  "DialogueNodeEvent:26:187",
  "DialogueNodeEvent:26:188",
  "DialogueNodeEvent:26:189",
  "DialogueNodeEvent:26:190",
  "DialogueNodeEvent:26:191",
  "DialogueNodeEvent:26:192",
  "DialogueNodeEvent:26:193",
  "DialogueNodeEvent:26:194",
  "DialogueNodeEvent:26:195",
  "DialogueNodeEvent:26:196",
  "DialogueNodeEvent:26:197",
  "DialogueNodeEvent:26:198",
  "DialogueNodeEvent:26:199",
  "DialogueNodeEvent:26:200",
  "DialogueNodeEvent:26:201",
  "DialogueNodeEvent:26:202",
  "DialogueNodeEvent:26:203",
  "DialogueNodeEvent:26:204",
  "DialogueNodeEvent:26:205",
  "DialogueNodeEvent:26:206",
  "DialogueNodeEvent:26:207",
  "DialogueNodeEvent:26:208",
  "DialogueNodeEvent:26:209",
  "DialogueNodeEvent:26:210",
  "DialogueNodeEvent:26:211"
];

// 1) Latest trigger (end anchor) by arrival order
const latestTrigger = db.logdata.findOne(
  { game: "mhs", playerId: playerId, eventKey: TRIGGER_KEY },
  { sort: { _id: -1 } }
);

if (!latestTrigger) {
  "yellow";
} else {
  // 2) Previous trigger (defines attempt start boundary)
  const prevTrigger = db.logdata.findOne(
    {
      game: "mhs",
      playerId: playerId,
      eventKey: TRIGGER_KEY,
      _id: { $lt: latestTrigger._id }
    },
    { sort: { _id: -1 } }
  );

  // Window is (prevTrigger._id, latestTrigger._id]
  const windowStartId = prevTrigger ? prevTrigger._id : ObjectId("000000000000000000000000");
  const windowEndId = latestTrigger._id;

  // 3) Count POS/NEG inside window
  const posCount = db.logdata.countDocuments({
    game: "mhs",
    playerId: playerId,
    eventKey: { $in: POS_KEYS },
    _id: { $gt: windowStartId, $lte: windowEndId }
  });

  const negCount = db.logdata.countDocuments({
    game: "mhs",
    playerId: playerId,
    eventKey: { $in: NEG_KEYS },
    _id: { $gt: windowStartId, $lte: windowEndId }
  });

  const score = posCount - (negCount / 3.0);

  score >= 4 ? "green" : "yellow";
}
```

Notes:
If later a true attemptId (or a START_KEY for U2P5) is introduced, we can tighten the window to that explicit boundary instead of "trigger-to-trigger," but this is the best standalone attempt-based production approach with the data provided.


---

## Unit 2, Point 6

**Title:** Drone Tutorial +Data Collection (Relate watershed size to flow rate through main river)

**Trigger:** `DialogueNodeEvent:20:35`

### Data Analytics Script

```python
yellow_node = ["DialogueNodeEvent:20:44", " DialogueNodeEvent:20:45"]
pass_node = " DialogueNodeEvent:20:43"
If has eventKey:
    pass_node = coll.find_one(
        {
            "playerId": pid,
            "eventKey": pass_node,
        }
    )
    if not pass_node:
          color = 2 ("yellow")
          continue
    yellow_44_45 = coll.find_one(
        {
            "playerId": pid,
            "eventKey": {"$in": yellow_node},
        })
    if yellow_44_45:
        color = 2 ("yellow")
    else:
        color = 1 ("green")
```

Rule statement
- Must have DialogueNodeEvent:20:43 (pass node)
- Must have none of:
- DialogueNodeEvent:20:44
- DialogueNodeEvent:20:45
- If pass node is missing → yellow
- Else if any yellow node is present → yellow
- Else → green

Note: Analytics script includes leading spaces on 20:45 and 20:43. Assuming eventKeys in DB have no leading/trailing spaces, these are trimmed below.

The following script matches analytics intent.

```js
// Unit 2, Point 6
// Trigger eventKey: "DialogueNodeEvent:20:35"

// Player identifier to evaluate
const playerId = "wenyi10@mhs.mhs";

const passKey = "DialogueNodeEvent:20:43";

const yellowKeys = [
  "DialogueNodeEvent:20:44",
  "DialogueNodeEvent:20:45"
];

const hasPass =
  db.logdata.findOne({
    game: "mhs",
    playerId: playerId,
    eventKey: passKey
  }) !== null;

if (!hasPass) {
  "yellow";
} else {
  const hasYellow =
    db.logdata.findOne({
      game: "mhs",
      playerId: playerId,
      eventKey: { $in: yellowKeys }
    }) !== null;

  const color = hasYellow ? "yellow" : "green";
  color;
}
```

The above as written (matching analytics intent), this is lifetime existence-based. If you want "latest attempt only," the following is the trigger-to-trigger (or start-to-trigger) windowing pattern.

### Unit 2 Point 6 Production Grading Script 

Unit 2, Point 6 — Attempt-based (latest attempt) Production Script

Trigger eventKey: DialogueNodeEvent:20:35

Rule statement (latest attempt only)
- Define the current attempt window as the records between:
- the previous trigger DialogueNodeEvent:20:35 (exclusive), and
- the latest trigger DialogueNodeEvent:20:35 (inclusive),
for the same player (using _id ordering).
- Within that window:
- Must have DialogueNodeEvent:20:43 (pass node)
- Must have none of DialogueNodeEvent:20:44 or DialogueNodeEvent:20:45
- If pass node is missing → yellow
- Else if any yellow node present → yellow
- Else → green

(Trim note: analytics had leading spaces on some keys; script uses trimmed keys.)

```js
// Unit 2, Point 6 — Attempt-based standalone production grading script
// Trigger eventKey: "DialogueNodeEvent:20:35"

const playerId = "wenyi10@mhs.mhs";

const TRIGGER_KEY = "DialogueNodeEvent:20:35";
const PASS_KEY = "DialogueNodeEvent:20:43";
const YELLOW_KEYS = ["DialogueNodeEvent:20:44", "DialogueNodeEvent:20:45"];

// 1) Latest trigger (end anchor)
const latestTrigger = db.logdata.findOne(
  { game: "mhs", playerId: playerId, eventKey: TRIGGER_KEY },
  { sort: { _id: -1 } }
);

if (!latestTrigger) {
  "yellow";
} else {
  // 2) Previous trigger (attempt boundary)
  const prevTrigger = db.logdata.findOne(
    {
      game: "mhs",
      playerId: playerId,
      eventKey: TRIGGER_KEY,
      _id: { $lt: latestTrigger._id }
    },
    { sort: { _id: -1 } }
  );

  const windowStartId = prevTrigger ? prevTrigger._id : ObjectId("000000000000000000000000");
  const windowEndId = latestTrigger._id;

  // 3) Must have PASS_KEY within attempt window
  const hasPass =
    db.logdata.findOne({
      game: "mhs",
      playerId: playerId,
      eventKey: PASS_KEY,
      _id: { $gt: windowStartId, $lte: windowEndId }
    }) !== null;

  if (!hasPass) {
    "yellow";
  } else {
    // 4) Must have none of YELLOW_KEYS within attempt window
    const hasYellow =
      db.logdata.findOne({
        game: "mhs",
        playerId: playerId,
        eventKey: { $in: YELLOW_KEYS },
        _id: { $gt: windowStartId, $lte: windowEndId }
      }) !== null;

    hasYellow ? "yellow" : "green";
  }
}
```

---

## Unit 2, Point 7

**Title:** Watershed Argument (Support a claim with evidence)

**Trigger:** `questFinishEvent:54`

### Data Analytics Script

```python
SUCCESS_KEY = " DialogueNodeEvent:27:7"

NEG_KEYS = [
    "DialogueNodeEvent:27:11", "DialogueNodeEvent:27:12", "DialogueNodeEvent:27:13", "DialogueNodeEvent:27:14",
    "DialogueNodeEvent:27:15", "DialogueNodeEvent:27:16", "DialogueNodeEvent:27:17", "DialogueNodeEvent:27:18",
    "DialogueNodeEvent:27:19", "DialogueNodeEvent:27:20", "DialogueNodeEvent:27:21", "DialogueNodeEvent:27:22",
    "DialogueNodeEvent:27:23", "DialogueNodeEvent:27:24", "DialogueNodeEvent:27:25", "DialogueNodeEvent:27:26",
    "DialogueNodeEvent:27:27", "DialogueNodeEvent:27:28", "DialogueNodeEvent:27:29", "DialogueNodeEvent:27:30"
]

If has eventKey:
    has_success = coll.find_one(
        {
            "playerId": pid,
            "eventKey": SUCCESS_KEY
        }) is not None
    neg_count = coll.count_documents(
        {
            "playerId": pid,
            "eventKey": {"$in": NEG_KEYS}
        }
)
    if has_success and neg_count <= 3:
        color = 1 ("green")
    else:
        color = 2 ("yellow")

```

Trigger eventKey: questFinishEvent:54

Rule statement (matches analytics script exactly)
- Must have SUCCESS_KEY = DialogueNodeEvent:27:7 (anywhere in history)
- Count neg_count = number of events with eventKey in NEG_KEYS (anywhere in history)
- If has_success and neg_count <= 3 → green
- Else → yellow

(Trim note: analytics shows a leading space on SUCCESS_KEY; script uses trimmed key.)

The following script matches analytics intent.

```js
// Unit 2, Point 7 — Analytics-matching script (lifetime)
// Trigger eventKey: "questFinishEvent:54"

const playerId = "wenyi10@mhs.mhs";

const SUCCESS_KEY = "DialogueNodeEvent:27:7";

const NEG_KEYS = [
  "DialogueNodeEvent:27:11", "DialogueNodeEvent:27:12", "DialogueNodeEvent:27:13", "DialogueNodeEvent:27:14",
  "DialogueNodeEvent:27:15", "DialogueNodeEvent:27:16", "DialogueNodeEvent:27:17", "DialogueNodeEvent:27:18",
  "DialogueNodeEvent:27:19", "DialogueNodeEvent:27:20", "DialogueNodeEvent:27:21", "DialogueNodeEvent:27:22",
  "DialogueNodeEvent:27:23", "DialogueNodeEvent:27:24", "DialogueNodeEvent:27:25", "DialogueNodeEvent:27:26",
  "DialogueNodeEvent:27:27", "DialogueNodeEvent:27:28", "DialogueNodeEvent:27:29", "DialogueNodeEvent:27:30"
];

const hasSuccess =
  db.logdata.findOne({
    game: "mhs",
    playerId: playerId,
    eventKey: SUCCESS_KEY
  }) !== null;

const negCount = db.logdata.countDocuments({
  game: "mhs",
  playerId: playerId,
  eventKey: { $in: NEG_KEYS }
});

const color = hasSuccess && negCount <= 3 ? "green" : "yellow";
color;
```

### Unit 2 Point 7 Production Grading Script 

Attempt-based standalone production script (latest attempt only)

Rule statement (latest attempt)
- Define attempt window as:
- previous questFinishEvent:54 (exclusive) → latest questFinishEvent:54 (inclusive)
- Within that window:
- Must have DialogueNodeEvent:27:7
- neg_count = count of NEG_KEYS
- If has_success and neg_count <= 3 → green else yellow
- If no latest trigger exists → yellow

```js
// Unit 2, Point 7 — Attempt-based standalone production grading script
// Trigger eventKey: "questFinishEvent:54"

const playerId = "wenyi10@mhs.mhs";

const TRIGGER_KEY = "questFinishEvent:54";
const SUCCESS_KEY = "DialogueNodeEvent:27:7";

const NEG_KEYS = [
  "DialogueNodeEvent:27:11", "DialogueNodeEvent:27:12", "DialogueNodeEvent:27:13", "DialogueNodeEvent:27:14",
  "DialogueNodeEvent:27:15", "DialogueNodeEvent:27:16", "DialogueNodeEvent:27:17", "DialogueNodeEvent:27:18",
  "DialogueNodeEvent:27:19", "DialogueNodeEvent:27:20", "DialogueNodeEvent:27:21", "DialogueNodeEvent:27:22",
  "DialogueNodeEvent:27:23", "DialogueNodeEvent:27:24", "DialogueNodeEvent:27:25", "DialogueNodeEvent:27:26",
  "DialogueNodeEvent:27:27", "DialogueNodeEvent:27:28", "DialogueNodeEvent:27:29", "DialogueNodeEvent:27:30"
];

// 1) Latest trigger (end anchor)
const latestTrigger = db.logdata.findOne(
  { game: "mhs", playerId: playerId, eventKey: TRIGGER_KEY },
  { sort: { _id: -1 } }
);

if (!latestTrigger) {
  "yellow";
} else {
  // 2) Previous trigger (attempt boundary)
  const prevTrigger = db.logdata.findOne(
    {
      game: "mhs",
      playerId: playerId,
      eventKey: TRIGGER_KEY,
      _id: { $lt: latestTrigger._id }
    },
    { sort: { _id: -1 } }
  );

  const windowStartId = prevTrigger ? prevTrigger._id : ObjectId("000000000000000000000000");
  const windowEndId = latestTrigger._id;

  const hasSuccess =
    db.logdata.findOne({
      game: "mhs",
      playerId: playerId,
      eventKey: SUCCESS_KEY,
      _id: { $gt: windowStartId, $lte: windowEndId }
    }) !== null;

  const negCount = db.logdata.countDocuments({
    game: "mhs",
    playerId: playerId,
    eventKey: { $in: NEG_KEYS },
    _id: { $gt: windowStartId, $lte: windowEndId }
  });

  (hasSuccess && negCount <= 3) ? "green" : "yellow";
}
```

---

## Unit 3, Point 1

**Title:** Good morning cadet + Establishing a foothold (Identify the direction of water flow based on a map of the watershed)

**Trigger:** `questFinishEvent:17`

### Data Analytics Script

```python
if has eventKey:
    
    cnt = coll.count_documents({"playerId": pid, "eventKey": "DialogueNodeEvent:10:30"})

    if cnt > 1:
        return 1  # green
    else:
        return 2  # yellow

```

Rule statement (matches analytics intent)
- Count how many log entries exist for the player with:
- eventKey = DialogueNodeEvent:10:30
- If cnt > 1 → green
- Else → yellow

Notes:
- This is count-based and (in the analytics version) counts across the player’s full history.

The following script matches analytics intent.

```js
// Unit 3, Point 1 — Analytics-matching script
// Trigger eventKey: "questFinishEvent:17"

const playerId = "wenyi10@mhs.mhs";

const cnt = db.logdata.countDocuments({
  game: "mhs",
  playerId: playerId,
  eventKey: "DialogueNodeEvent:10:30"
});

const color = cnt > 1 ? "green" : "yellow";
color;
```

### Unit 3 Point 1 Production Grading Script 

Rule statement (latest attempt only, replay-safe)
- Define the attempt window as:
- previous questFinishEvent:17 (exclusive) to latest questFinishEvent:17 (inclusive), by _id
- Within that window, count:
- eventKey = DialogueNodeEvent:10:30
- If cnt > 1 → green
- Else → yellow
- If the latest trigger does not exist → yellow (though in your trigger-queued pipeline this script would normally not run)

Attempt-based standalone production script (player can replay)

```js
// Unit 3, Point 1 — Attempt-based standalone production script (latest attempt)
// Trigger eventKey: "questFinishEvent:17"

const playerId = "wenyi10@mhs.mhs";

const TRIGGER_KEY = "questFinishEvent:17";
const TARGET_KEY = "DialogueNodeEvent:10:30";

// 1) Latest trigger (end anchor)
const latestTrigger = db.logdata.findOne(
  { game: "mhs", playerId: playerId, eventKey: TRIGGER_KEY },
  { sort: { _id: -1 } }
);

if (!latestTrigger) {
  "yellow";
} else {
  // 2) Previous trigger (attempt boundary)
  const prevTrigger = db.logdata.findOne(
    {
      game: "mhs",
      playerId: playerId,
      eventKey: TRIGGER_KEY,
      _id: { $lt: latestTrigger._id }
    },
    { sort: { _id: -1 } }
  );

  const windowStartId = prevTrigger ? prevTrigger._id : ObjectId("000000000000000000000000");
  const windowEndId = latestTrigger._id;

  // 3) Count target occurrences within attempt window
  const cnt = db.logdata.countDocuments({
    game: "mhs",
    playerId: playerId,
    eventKey: TARGET_KEY,
    _id: { $gt: windowStartId, $lte: windowEndId }
  });

  cnt > 1 ? "green" : "yellow";
}
```

---

## Unit 3, Point 2

**Title:** Pollution solution (Predict the spread of dissolved materials through a watershed.)

**Trigger:** `DialogueNodeEvent:11:34`

### Data Analytics Script

```python
If has eventKey:
    c27 = coll.count_documents({"playerId": pid, "eventKey": "DialogueNodeEvent:11:27"})
    c29 = coll.count_documents({"playerId": pid, "eventKey": "DialogueNodeEvent:11:29"})
    c230 = coll.count_documents({"playerId": pid, "eventKey": "DialogueNodeEvent:11:230"})
    cSum = c29 + c230
    
    def capped_penalty(cnt: int) -> int:
        if cnt <= 1:
            return 0
        elif cnt <= 3:
            return 1
        else:
            return 2

    score = 5
    score -= capped_penalty(c27)
    score -= capped_penalty(cSum)

    return 2 # yellow if score < 3 else 1 # green
```

Rule statement (matches analytics intent)
- Count (over the player’s full history):
- c27 = count of DialogueNodeEvent:11:27
- c29 = count of DialogueNodeEvent:11:29
- c230 = count of DialogueNodeEvent:11:230
- cSum = c29 + c230
- Define capped_penalty(cnt):
- 0 if cnt <= 1
- 1 if cnt <= 3
- 2 if cnt >= 4
- Compute:
- score = 5 - capped_penalty(c27) - capped_penalty(cSum)
- If score < 3 → yellow
- Else → green


The following script matches analytics intent.

```js
// Unit 3, Point 2 — Analytics-matching script
// Trigger eventKey: "DialogueNodeEvent:11:34"

const playerId = "wenyi10@mhs.mhs";

function cappedPenalty(cnt) {
  if (cnt <= 1) return 0;
  if (cnt <= 3) return 1;
  return 2;
}

const c27 = db.logdata.countDocuments({
  game: "mhs",
  playerId: playerId,
  eventKey: "DialogueNodeEvent:11:27"
});

const c29 = db.logdata.countDocuments({
  game: "mhs",
  playerId: playerId,
  eventKey: "DialogueNodeEvent:11:29"
});

const c230 = db.logdata.countDocuments({
  game: "mhs",
  playerId: playerId,
  eventKey: "DialogueNodeEvent:11:230"
});

const cSum = c29 + c230;

let score = 5;
score -= cappedPenalty(c27);
score -= cappedPenalty(cSum);

const color = score < 3 ? "yellow" : "green";
color;
```

### Unit 3 Point 2 Production Grading Script 

Rule statement (latest attempt only, replay-safe)
- Define the attempt window as:
- previous DialogueNodeEvent:11:34 (exclusive) to latest DialogueNodeEvent:11:34 (inclusive), by _id
- Within that window, count:
- c27 for 11:27
- c29 for 11:29
- c230 for 11:230
- cSum = c29 + c230
- Use the same capped_penalty and score logic as above.
- If score < 3 → yellow else green.
- If no latest trigger exists → yellow (normally won’t run in your trigger-driven pipeline)

```js
// Unit 3, Point 2 — Attempt-based standalone production script (latest attempt)
// Trigger eventKey: "DialogueNodeEvent:11:34"

const playerId = "wenyi10@mhs.mhs";

const TRIGGER_KEY = "DialogueNodeEvent:11:34";

function cappedPenalty(cnt) {
  if (cnt <= 1) return 0;
  if (cnt <= 3) return 1;
  return 2;
}

// 1) Latest trigger (end anchor)
const latestTrigger = db.logdata.findOne(
  { game: "mhs", playerId: playerId, eventKey: TRIGGER_KEY },
  { sort: { _id: -1 } }
);

if (!latestTrigger) {
  "yellow";
} else {
  // 2) Previous trigger (attempt boundary)
  const prevTrigger = db.logdata.findOne(
    {
      game: "mhs",
      playerId: playerId,
      eventKey: TRIGGER_KEY,
      _id: { $lt: latestTrigger._id }
    },
    { sort: { _id: -1 } }
  );

  const windowStartId = prevTrigger ? prevTrigger._id : ObjectId("000000000000000000000000");
  const windowEndId = latestTrigger._id;

  const c27 = db.logdata.countDocuments({
    game: "mhs",
    playerId: playerId,
    eventKey: "DialogueNodeEvent:11:27",
    _id: { $gt: windowStartId, $lte: windowEndId }
  });

  const c29 = db.logdata.countDocuments({
    game: "mhs",
    playerId: playerId,
    eventKey: "DialogueNodeEvent:11:29",
    _id: { $gt: windowStartId, $lte: windowEndId }
  });

  const c230 = db.logdata.countDocuments({
    game: "mhs",
    playerId: playerId,
    eventKey: "DialogueNodeEvent:11:230",
    _id: { $gt: windowStartId, $lte: windowEndId }
  });

  const cSum = c29 + c230;

  let score = 5;
  score -= cappedPenalty(c27);
  score -= cappedPenalty(cSum);

  score < 3 ? "yellow" : "green";
}
```

---

## Unit 3, Point 3

**Title:** Pollution argument (Construct an argument with reasoning that links a claim with evidence.)

**Trigger:** `questFinishEvent:18`

### Data Analytics Script

```python
TARGET_KEYS = [
        "DialogueNodeEvent:84:20", "DialogueNodeEvent:84:25", "DialogueNodeEvent:84:32",
        "DialogueNodeEvent:84:33", "DialogueNodeEvent:84:34", "DialogueNodeEvent:84:35",
        "DialogueNodeEvent:84:37", "DialogueNodeEvent:84:39", "DialogueNodeEvent:84:40",
        "DialogueNodeEvent:84:41", "DialogueNodeEvent:84:42", "DialogueNodeEvent:84:43",
        "DialogueNodeEvent:84:44", "DialogueNodeEvent:84:45", "DialogueNodeEvent:84:46",
        "DialogueNodeEvent:84:47"
]

If has eventKey:
    sum_count = coll.count_documents({
        "playerId": pid,
        "eventKey": {"$in": TARGET_KEYS}
})

if sum_count <= 3:
        base_score = 3
    elif sum_count == 4:
        base_score = 2
    elif sum_count == 5:
        base_score = 1
    else:
        base_score = 0

has_bonus = coll.find_one(
        {
            "playerId": pid,
            "eventType": "argumentationToolEvent",
            "data.toolName": "BackingInfoPanel - Pollution Site Data"}
        },
        projection={"_id": 1}
    ) is not None

total_score = base_score + (1 if has_bonus else 0)

return 1 if total_score >= 3 else 2
```

Rule statement (matches analytics intent)
- Count sum_count = number of log entries for the player with eventKey in TARGET_KEYS (full history).
- Compute base_score from sum_count:
- if sum_count <= 3 → base_score = 3
- else if sum_count == 4 → base_score = 2
- else if sum_count == 5 → base_score = 1
- else (sum_count >= 6) → base_score = 0
- Compute has_bonus (full history): true if there exists a log entry with:
- eventType = "argumentationToolEvent" and
- data.toolName = "BackingInfoPanel - Pollution Site Data"
- total_score = base_score + (1 if has_bonus else 0)
- If total_score >= 3 → green
- Else → yellow

The following script matches analytics intent.

```js
// Unit 3, Point 3 — Analytics-matching script
// Trigger eventKey: "questFinishEvent:18"

const playerId = "wenyi10@mhs.mhs";

const TARGET_KEYS = [
  "DialogueNodeEvent:84:20", "DialogueNodeEvent:84:25", "DialogueNodeEvent:84:32",
  "DialogueNodeEvent:84:33", "DialogueNodeEvent:84:34", "DialogueNodeEvent:84:35",
  "DialogueNodeEvent:84:37", "DialogueNodeEvent:84:39", "DialogueNodeEvent:84:40",
  "DialogueNodeEvent:84:41", "DialogueNodeEvent:84:42", "DialogueNodeEvent:84:43",
  "DialogueNodeEvent:84:44", "DialogueNodeEvent:84:45", "DialogueNodeEvent:84:46",
  "DialogueNodeEvent:84:47"
];

const sumCount = db.logdata.countDocuments({
  game: "mhs",
  playerId: playerId,
  eventKey: { $in: TARGET_KEYS }
});

let baseScore;
if (sumCount <= 3) baseScore = 3;
else if (sumCount === 4) baseScore = 2;
else if (sumCount === 5) baseScore = 1;
else baseScore = 0;

const hasBonus =
  db.logdata.findOne(
    {
      game: "mhs",
      playerId: playerId,
      eventType: "argumentationToolEvent",
      "data.toolName": "BackingInfoPanel - Pollution Site Data"
    },
    { projection: { _id: 1 } }
  ) !== null;

const totalScore = baseScore + (hasBonus ? 1 : 0);

const color = totalScore >= 3 ? "green" : "yellow";
color;
```

### Unit 3 Point 3 Production Grading Script 

Rule statement (latest attempt only, replay-safe)
- Define the attempt window as:
 - previous questFinishEvent:18 (exclusive) to latest questFinishEvent:18 (inclusive), by _id
- Within that window:
 - Count sum_count for TARGET_KEYS
 - Determine base_score using the same mapping
 - Determine has_bonus based on an argumentationToolEvent with data.toolName matching exactly
 - total_score = base_score + bonus
- If total_score >= 3 → green else yellow
- If no latest trigger exists → yellow (normally won’t run in your trigger-driven pipeline)

Attempt-based standalone production script (player can replay)

```js
// Unit 3, Point 3 — Attempt-based standalone production script (latest attempt)
// Trigger eventKey: "questFinishEvent:18"

const playerId = "wenyi10@mhs.mhs";

const TRIGGER_KEY = "questFinishEvent:18";

const TARGET_KEYS = [
  "DialogueNodeEvent:84:20", "DialogueNodeEvent:84:25", "DialogueNodeEvent:84:32",
  "DialogueNodeEvent:84:33", "DialogueNodeEvent:84:34", "DialogueNodeEvent:84:35",
  "DialogueNodeEvent:84:37", "DialogueNodeEvent:84:39", "DialogueNodeEvent:84:40",
  "DialogueNodeEvent:84:41", "DialogueNodeEvent:84:42", "DialogueNodeEvent:84:43",
  "DialogueNodeEvent:84:44", "DialogueNodeEvent:84:45", "DialogueNodeEvent:84:46",
  "DialogueNodeEvent:84:47"
];

// 1) Latest trigger (end anchor)
const latestTrigger = db.logdata.findOne(
  { game: "mhs", playerId: playerId, eventKey: TRIGGER_KEY },
  { sort: { _id: -1 } }
);

if (!latestTrigger) {
  "yellow";
} else {
  // 2) Previous trigger (attempt boundary)
  const prevTrigger = db.logdata.findOne(
    {
      game: "mhs",
      playerId: playerId,
      eventKey: TRIGGER_KEY,
      _id: { $lt: latestTrigger._id }
    },
    { sort: { _id: -1 } }
  );

  const windowStartId = prevTrigger ? prevTrigger._id : ObjectId("000000000000000000000000");
  const windowEndId = latestTrigger._id;

  const sumCount = db.logdata.countDocuments({
    game: "mhs",
    playerId: playerId,
    eventKey: { $in: TARGET_KEYS },
    _id: { $gt: windowStartId, $lte: windowEndId }
  });

  let baseScore;
  if (sumCount <= 3) baseScore = 3;
  else if (sumCount === 4) baseScore = 2;
  else if (sumCount === 5) baseScore = 1;
  else baseScore = 0;

  const hasBonus =
    db.logdata.findOne(
      {
        game: "mhs",
        playerId: playerId,
        eventType: "argumentationToolEvent",
        "data.toolName": "BackingInfoPanel - Pollution Site Data",
        _id: { $gt: windowStartId, $lte: windowEndId }
      },
      { projection: { _id: 1 } }
    ) !== null;

  const totalScore = baseScore + (hasBonus ? 1 : 0);

  totalScore >= 3 ? "green" : "yellow";
}
```

---

## Unit 3, Point 4

**Title:** Forsaken Facility (Predict the spread of dissolved materials)

**Trigger:** `DialogueNodeEvent:73:200`

### Data Analytics Script

```python
TARGET_KEYS = [
        "DialogueNodeEvent:78:3", "DialogueNodeEvent:78:4", "DialogueNodeEvent:78:7",
        "DialogueNodeEvent:78:9", "DialogueNodeEvent:78:10", "DialogueNodeEvent:78:12",
        "DialogueNodeEvent:78:18", "DialogueNodeEvent:78:23"
]

If has eventKey:
    has_7824 = coll.find_one(
        {"playerId": pid, "eventKey": "DialogueNodeEvent:78:24"},
        projection={"_id": 1}
) is not None
if not has_7824:
        return 2 #yellow

    total_count = coll.count_documents({
        "playerId": pid,
        "eventKey": {"$in": TARGET_KEYS}
})

if total_count == 0:
        score = 2
    elif total_count <= 2:
        score = 1
    else:
        score = 0

return 2 #yellow if score == 0 else 1 #green
```

Rule statement (matches analytics intent)
- First, require the player to have:
- DialogueNodeEvent:78:24
- If missing → yellow
- Then count total_count = number of log entries with eventKey in TARGET_KEYS (full history).
- Compute score from total_count:
- if total_count == 0 → score = 2
- else if total_count <= 2 → score = 1
- else (total_count >= 3) → score = 0
- If score == 0 → yellow
- Else → green


The following script matches analytics intent.

```js
// Unit 3, Point 4 — Analytics-matching script
// Trigger eventKey: "DialogueNodeEvent:73:200"

const playerId = "wenyi10@mhs.mhs";

const TARGET_KEYS = [
  "DialogueNodeEvent:78:3", "DialogueNodeEvent:78:4", "DialogueNodeEvent:78:7",
  "DialogueNodeEvent:78:9", "DialogueNodeEvent:78:10", "DialogueNodeEvent:78:12",
  "DialogueNodeEvent:78:18", "DialogueNodeEvent:78:23"
];

// Gate: must have 78:24
const has7824 =
  db.logdata.findOne(
    { game: "mhs", playerId: playerId, eventKey: "DialogueNodeEvent:78:24" },
    { projection: { _id: 1 } }
  ) !== null;

if (!has7824) {
  "yellow";
} else {
  const totalCount = db.logdata.countDocuments({
    game: "mhs",
    playerId: playerId,
    eventKey: { $in: TARGET_KEYS }
  });

  let score;
  if (totalCount === 0) score = 2;
  else if (totalCount <= 2) score = 1;
  else score = 0;

  score === 0 ? "yellow" : "green";
}
```

(So: missing 78:24 is immediate yellow; otherwise green unless target count is 3+.)

### Unit 3 Point 4 Production Grading Script 

Rule statement (latest attempt only, replay-safe)
- Define the attempt window as:
- previous DialogueNodeEvent:73:200 (exclusive) to latest DialogueNodeEvent:73:200 (inclusive), by _id.
- Within that window:
- Must have DialogueNodeEvent:78:24 or else yellow.
- Count total_count for TARGET_KEYS.
- Score mapping identical to analytics.
- If score == 0 → yellow else green.
- If no latest trigger exists → yellow (normally won’t run in your trigger-driven pipeline).

Attempt-based standalone production script (player can replay)

```js
// Unit 3, Point 4 — Attempt-based standalone production script (latest attempt)
// Trigger eventKey: "DialogueNodeEvent:73:200"

const playerId = "wenyi10@mhs.mhs";

const TRIGGER_KEY = "DialogueNodeEvent:73:200";

const TARGET_KEYS = [
  "DialogueNodeEvent:78:3", "DialogueNodeEvent:78:4", "DialogueNodeEvent:78:7",
  "DialogueNodeEvent:78:9", "DialogueNodeEvent:78:10", "DialogueNodeEvent:78:12",
  "DialogueNodeEvent:78:18", "DialogueNodeEvent:78:23"
];

// 1) Latest trigger (end anchor)
const latestTrigger = db.logdata.findOne(
  { game: "mhs", playerId: playerId, eventKey: TRIGGER_KEY },
  { sort: { _id: -1 } }
);

if (!latestTrigger) {
  "yellow";
} else {
  // 2) Previous trigger (attempt boundary)
  const prevTrigger = db.logdata.findOne(
    {
      game: "mhs",
      playerId: playerId,
      eventKey: TRIGGER_KEY,
      _id: { $lt: latestTrigger._id }
    },
    { sort: { _id: -1 } }
  );

  const windowStartId = prevTrigger ? prevTrigger._id : ObjectId("000000000000000000000000");
  const windowEndId = latestTrigger._id;

  // Gate: must have 78:24 within attempt window
  const has7824 =
    db.logdata.findOne(
      {
        game: "mhs",
        playerId: playerId,
        eventKey: "DialogueNodeEvent:78:24",
        _id: { $gt: windowStartId, $lte: windowEndId }
      },
      { projection: { _id: 1 } }
    ) !== null;

  if (!has7824) {
    "yellow";
  } else {
    const totalCount = db.logdata.countDocuments({
      game: "mhs",
      playerId: playerId,
      eventKey: { $in: TARGET_KEYS },
      _id: { $gt: windowStartId, $lte: windowEndId }
    });

    let score;
    if (totalCount === 0) score = 2;
    else if (totalCount <= 2) score = 1;
    else score = 0;

    score === 0 ? "yellow" : "green";
  }
}
```

This one’s nicely deterministic. The only “gotcha” to watch for in live data is whether 78:24 can occur after the trigger 73:200 in timestamp ordering; but since we’re using _id windowing around the trigger, it’ll be counted only if it occurs in the same attempt window.

---

## Unit 3, Point 5

**Title:** Plant the superfruit seeds (Predict the spread of dissolved materials through a watershed)

**Trigger:** `DialogueNodeEvent:10:194`

### Data Analytics Script

```python
If has “DialogueNodeEvent:10:194":
    pos_count = coll.count_documents({"playerId": pid, "eventKey": "DialogueNodeEvent:73:163"})

    pos_score = pos_count * 1.0

    NEG_KEYS = ["DialogueNodeEvent:73:164", "DialogueNodeEvent:73:168", "DialogueNodeEvent:73:171"]
neg_count = coll.count_documents({"playerId": pid, "eventKey": {"$in": NEG_KEYS}})

    neg_score = neg_count * 0.5

    sum_score = pos_score - neg_score

    return 2 #yellow if sum_score < 3 else 1 #green

```

Rule statement (matches analytics intent)
- Count (over the player’s full history):
- pos_count = count of DialogueNodeEvent:73:163
- neg_count = count of events with eventKey in:
- DialogueNodeEvent:73:164
- DialogueNodeEvent:73:168
- DialogueNodeEvent:73:171
- Compute:
- pos_score = pos_count * 1.0
- neg_score = neg_count * 0.5
- sum_score = pos_score - neg_score
- If sum_score < 3 → yellow
- Else → green

The following script matches analytics intent.

```js
// Unit 3, Point 5 — Analytics-matching script
// Trigger eventKey: "DialogueNodeEvent:10:194"

const playerId = "wenyi10@mhs.mhs";

const POS_KEY = "DialogueNodeEvent:73:163";
const NEG_KEYS = ["DialogueNodeEvent:73:164", "DialogueNodeEvent:73:168", "DialogueNodeEvent:73:171"];

const posCount = db.logdata.countDocuments({
  game: "mhs",
  playerId: playerId,
  eventKey: POS_KEY
});

const negCount = db.logdata.countDocuments({
  game: "mhs",
  playerId: playerId,
  eventKey: { $in: NEG_KEYS }
});

const posScore = posCount * 1.0;
const negScore = negCount * 0.5;

const sumScore = posScore - negScore;

const color = sumScore < 3 ? "yellow" : "green";
color;
```

### Unit 3 Point 5 Production Grading Script

Rule statement (latest attempt only, replay-safe)
- Define the attempt window as:
 - previous DialogueNodeEvent:10:194 (exclusive) to latest DialogueNodeEvent:10:194 (inclusive), by _id
- Within that window:
 - pos_count = count of DialogueNodeEvent:73:163
 - neg_count = count of NEG_KEYS
 - sum_score = (pos_count * 1.0) - (neg_count * 0.5)
- If sum_score < 3 → yellow else green
- If no latest trigger exists → yellow (normally won’t run in your trigger-driven pipeline)

Attempt-based standalone production script (player can replay)

```js
// Unit 3, Point 5 — Attempt-based standalone production script (latest attempt)
// Trigger eventKey: "DialogueNodeEvent:10:194"

const playerId = "wenyi10@mhs.mhs";

const TRIGGER_KEY = "DialogueNodeEvent:10:194";
const POS_KEY = "DialogueNodeEvent:73:163";
const NEG_KEYS = ["DialogueNodeEvent:73:164", "DialogueNodeEvent:73:168", "DialogueNodeEvent:73:171"];

// 1) Latest trigger (end anchor)
const latestTrigger = db.logdata.findOne(
  { game: "mhs", playerId: playerId, eventKey: TRIGGER_KEY },
  { sort: { _id: -1 } }
);

if (!latestTrigger) {
  "yellow";
} else {
  // 2) Previous trigger (attempt boundary)
  const prevTrigger = db.logdata.findOne(
    {
      game: "mhs",
      playerId: playerId,
      eventKey: TRIGGER_KEY,
      _id: { $lt: latestTrigger._id }
    },
    { sort: { _id: -1 } }
  );

  const windowStartId = prevTrigger ? prevTrigger._id : ObjectId("000000000000000000000000");
  const windowEndId = latestTrigger._id;

  const posCount = db.logdata.countDocuments({
    game: "mhs",
    playerId: playerId,
    eventKey: POS_KEY,
    _id: { $gt: windowStartId, $lte: windowEndId }
  });

  const negCount = db.logdata.countDocuments({
    game: "mhs",
    playerId: playerId,
    eventKey: { $in: NEG_KEYS },
    _id: { $gt: windowStartId, $lte: windowEndId }
  });

  const sumScore = (posCount * 1.0) - (negCount * 0.5);

  sumScore < 3 ? "yellow" : "green";
}
```

One minor implementation note: multiplying by 1.0 and 0.5 is enough to force floating-point behavior in JS anyway, but I kept it explicit to mirror the analytics script.
---

