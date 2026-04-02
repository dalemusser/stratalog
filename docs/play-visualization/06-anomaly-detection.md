# Anomaly Detection

## Purpose

Automatically detect and report issues in a student's gameplay data that explain pencil icons, empty cells, and other dashboard problems. The anomaly summary appears at the top of the student detail view and in the student list as counts.

## Anomaly Types

### 1. Stuck Active (Pencil)

**Detection**: A progress point has a grade with status "active" as the latest attempt, meaning the start event was received but the end/trigger event never arrived.

**Severity**: Error

**Information to show**:
- Which progress point
- When the start event was received
- What end eventKey was expected
- Whether the expected end eventKey exists anywhere in the logs after the start (could indicate an ordering/cursor issue vs. truly missing)
- Time elapsed since the start event
- What events came after the start event (last few events in that unit)

**Example output**:
```
U3P2 [pencil]: Start event seen at 16:24:45 (DialogueNodeEvent:11:34).
Expected end: questFinishEvent:18. NOT FOUND in logs after start.
Last events in Unit 3: DialogueEvent at 16:25:10, DialogueEvent at 16:25:14,
then no more Unit 3 events. Student likely quit the activity.
```

### 2. Missing Grade (Empty Cell)

**Detection**: A progress point has no grade entry at all, but contextual evidence suggests the student should have encountered it:
- Prior progress points in the same unit have grades
- The student has grades for later progress points (skipped this one)
- The student reached EndOfUnit for this unit

**Severity**: Warning (if student appears to have reached that point) or Info (if student hasn't reached it yet)

**Information to show**:
- Which progress point
- What start eventKey was expected
- Whether that eventKey exists in the logs (indicates grader missed it)
- What the student's furthest progress point in this unit is
- Whether EndOfUnit exists for this unit

**Example output**:
```
U4P6 [empty]: No grade exists. Expected start: questActiveEvent:41.
Start event FOUND in logs at 16:59:48, but grader has no record.
Possible cause: eventKey format mismatch or grader cursor issue.
The student has EndOfUnit for Unit 4 at 17:08:26.
```

Or:
```
U4P6 [empty]: No grade exists. Expected start: questActiveEvent:41.
Start event NOT FOUND in logs. Student may not have reached this activity.
Last graded point in Unit 4: u4p5 (passed at 17:03:54).
```

### 3. Duplicate Events

**Detection**: Same eventKey fired multiple times within a short window (< 30 seconds). Common with EndOfUnit and some quest events.

**Severity**: Warning

**Information to show**:
- Which eventKey
- How many times it fired
- Time span of the duplicates
- Whether this affected grading (multiple grade entries for same point)

**Example output**:
```
EndOfUnit (Unit 4): Fired 3 times in 18 seconds
  17:08:08.965, 17:08:26.384, 17:08:26.522
Likely game bug — multiple exit triggers in same scene.
Grade impact: None (grader handles idempotently).
```

### 4. Large Time Gaps

**Detection**: Gap of > 10 minutes between consecutive events for the same student within a session (same day, same unit).

**Severity**: Info

**Information to show**:
- Duration of the gap
- Events before and after the gap
- Whether a progress point was active during the gap

**Example output**:
```
Gap: 45 minutes between events in Unit 3
  Before: DialogueEvent at 16:25:14 (during u3p2)
  After: DialogueEvent at 17:10:30 (during u3p2)
  Progress point u3p2 was active during this gap.
  Student likely left and returned.
```

### 5. Event Key Present but No Grade

**Detection**: A start eventKey exists in the logs but the grader has no corresponding grade entry. This is different from "Missing Grade" — here we're specifically finding the event in the log data.

**Severity**: Error (indicates a grader issue)

**Information to show**:
- The eventKey found in logs
- Its timestamp
- The grader's cursor position (if available) relative to this event
- Whether the event has the expected `eventKey` field format

### 6. Version Mismatch

**Detection**: Events in a progress point's window come from different game versions (the `version` field changes).

**Severity**: Info

**Information to show**:
- Which versions appear
- Where the version boundary is
- Whether this correlates with any grading anomalies

## Implementation

### Anomaly Detector

```go
type AnomalyDetector struct {
    rules       *GradingRulesConfig
    ruleIndex   map[string][]GradingRule  // eventKey → rules
}

func NewAnomalyDetector(rules *GradingRulesConfig) *AnomalyDetector

// DetectAnomalies analyzes grades and log events for a single student.
func (d *AnomalyDetector) DetectAnomalies(
    grades map[string][]ProgressGradeItem,
    events []LogEntry,
) []DebugAnomaly
```

### Detection Algorithm

```
For each progress point in order (u1p1 → u5p4):
  1. Check if grades exist for this point
     - If no grades:
       - Search logs for the start eventKey
       - If found: ANOMALY "Event Key Present but No Grade"
       - If not found but later points have grades: ANOMALY "Missing Grade (skipped)"
       - If not found and no later grades: NORMAL (student hasn't reached it)

  2. If grades exist, check latest attempt:
     - If status == "active":
       - Search logs for the end eventKey after the start time
       - Report findings as "Stuck Active" anomaly

  3. Check for duplicate start/end events within 30-second windows

For EndOfUnit events:
  - Count occurrences per unit
  - Flag if > 1 within 60 seconds

For time gaps:
  - Scan events chronologically
  - Flag gaps > 10 minutes within same-day activity
```

### Performance Considerations

- Loading all events for a student can be expensive (2500+ events for a full playthrough)
- For the student list view, only compute anomaly counts from grades (fast, no log queries)
- For the detail view, load events on demand (when student is clicked)
- Unit filtering reduces the event set significantly
- Consider caching the anomaly analysis for a student if it's accessed repeatedly
