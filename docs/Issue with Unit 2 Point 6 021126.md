# Issue with Unit 2 Point 6

Issue Identified – Unit 2, Point 6 (Drone Tutorial + Data Collection)

A mismatch has been identified between the grading logic as written and the actual ordering of gameplay events in the log data.

## Intended Design

For this progress point:
- Trigger: DialogueNodeEvent:20:35
- Evidence required for grading:
  - Pass node: DialogueNodeEvent:20:43
  - Yellow nodes: DialogueNodeEvent:20:44, DialogueNodeEvent:20:45

The intended design assumption (which the replay-safe grading architecture depends on) is:

The trigger eventKey should occur after all activity relevant to grading this progress point.

This allows the trigger to be treated as the end-of-attempt anchor, meaning:
- All evidence used for grading should occur before the trigger in log order.

---

## What Was Observed in Actual Log Data

In real records for a student (wenyi12@mhs.mhs), the following was observed:

```
Trigger (20:35):  _id 6985303fe93b16888f5f27f2
Pass node (20:43): _id 69853044e93b16888f5f27fc
```

The pass node occurs after the trigger in _id order.

This means:
- The grading evidence (20:43) is being logged after the trigger (20:35).
- When a single attempt is correctly isolated (to support replay), and the attempt window is defined as:
  (previous trigger, latest trigger]
  the pass node is not included in that window.
- As a result:
  - Lifetime analytics grading = green
  - Replay-safe production grading = yellow

This discrepancy is caused entirely by event ordering.

---

## Why This Matters

The grading system supports replay and attempt isolation. To do that safely, the following is required:
- Define a grading window per attempt.
- Anchor that window using the trigger event.

For that model to work reliably:

The trigger must occur after all grading-relevant events.

If evidence occurs after the trigger:
- The trigger is effectively acting as a start marker, not an end marker.
- That breaks the grading window assumption.
- It causes replay-safe grading to disagree with analytics.

---

## What Needs Clarification / Correction

One of the following must be true for Unit 2, Point 6:

1. The trigger event (DialogueNodeEvent:20:35) is firing too early.
   It should be moved to occur after pass/yellow nodes.
2. The pass/yellow nodes are firing too late.
   They should occur before the trigger event.
3. The trigger is conceptually a start marker, not an end marker.
   If that is the case, confirmation is needed so this point can be treated differently in the grading system.

---

## Requested Clarification

For Unit 2, Point 6:
- Is DialogueNodeEvent:20:35 intended to represent:
  - Completion of the progress point?
  - Or the beginning of the evaluated activity?

If it is intended to represent completion, then the ordering of events in gameplay needs to be adjusted so that:

```
[activity events]
(pass / yellow evidence)
--> trigger (20:35)
```

instead of:

```
trigger (20:35)
--> pass / yellow evidence
```

## Why This Is Important Going Forward

As grading is expanded with replay support:
- All progress points must follow a consistent pattern.
- Either triggers are end anchors (preferred), or
- It must be explicitly documented which points use start anchors.

Without that consistency, replay-safe grading will produce inconsistent results compared to analytics.

---

If helpful, additional examples of specific log sequences demonstrating the issue can be provided.
