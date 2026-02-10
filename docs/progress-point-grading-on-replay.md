What changes when you support “go back / replay”

Once teachers can rewind a student, the log stream becomes multiple attempts at the same progress point. If you keep using “earliest start ever,” you’ll grade the first attempt forever, which is wrong once rewinds exist.

So the grader needs to grade the attempt that just completed — i.e., the one whose end-trigger event just arrived.

The key idea: anchor windows to the trigger log

Because the trigger eventKey is emitted at the end of the progress point:
- the relevant gameplay evidence is before the trigger record
- the trigger record itself gives you:
- a stable arrival-order anchor: _id
- a gameplay-time anchor: timestamp

So for any windowed rule, you can define “this attempt” as:
1. End anchor = the trigger log record (the one that caused grading)
2. Start anchor = the most recent START_KEY before that end anchor
3. Evaluate targets between start and end (by timestamp), and optionally also constrain by _id to avoid cross-attempt bleed.

Why _id is useful even though timing uses timestamp
- timestamp tells you game time ordering (good for duration/sequence)
- _id tells you arrival order (good for incremental processing + isolating “the attempt that just finished”)

When a teacher rewinds, the “new attempt” will generate new log entries with new _ids, even if some timestamps overlap weirdly (or if the game replays an older segment).

So _id is your “newest attempt boundary” signal.

A robust pattern for these scripts (windowed points)

When grading is triggered by an end event triggerDoc:
- Find endDoc = the specific log record that triggered grading (you already have it in the service)
- Find startDoc = most recent START_KEY with:
- timestamp <= endDoc.timestamp (game-time)
- and (optionally) _id <= endDoc._id (arrival-time safety)
- sort by timestamp: -1 (or _id: -1 as a backup)

Then count target events with:
- timestamp between [startDoc.timestamp, endDoc.timestamp]
- and _id between [startDoc._id, endDoc._id] (optional but very helpful once rewinds exist)

That last _id bracket is the “anti-smearing” fence.

Concrete example adjustment (conceptual, not rewriting all scripts yet)

Instead of:
- “earliest START”
- “earliest END after it”

You’ll want:
- “END = the trigger that just happened”
- “START = nearest START before that END”

That matches your statement: “trigger happens at the end; evidence is prior.”

Practical implication for how we write your mongosh scripts

Right now, our standalone scripts take only playerId. For rewind-proof grading, the script ideally also takes:
- triggerLogId (the _id of the end trigger record)

Because “the newest” is defined relative to which trigger instance we’re grading.

So I suggest two versions for each windowed rule going forward:
1. Standalone/debug script (playerId-only): okay for manual checking, but ambiguous across attempts
2. Production/grader script (playerId + triggerLogId): unambiguous and rewind-ready

The production version of the grading script uses the trigger record as the end anchor.

