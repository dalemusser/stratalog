# Reasons and Messages for Progress Points

## Overview

The MHS Dashboard displays student progress as colored squares:
- **Green** = Successfully completed
- **Yellow** = Needs review (completed but with issues)
- **White/Empty** = Not yet attempted

While the grading service determines green vs yellow, **teachers need to understand WHY a student received yellow** so they can provide targeted support.

This document describes the system for providing human-readable explanations for progress point grades.

---

## The Need

### Current State

The grading service evaluates gameplay logs and produces a grade:
- `color: "green"` or `color: "yellow"`
- `reasonCode: "TOO_MANY_TARGETS"` (machine-readable)
- `metrics: { countTargets: 9, threshold: 6 }` (raw data)

### The Gap

Teachers see a yellow square but don't know:
- What the student struggled with
- What specific misconception or error occurred
- How to help the student improve

### The Goal

When a teacher clicks or hovers on a yellow square, they see a message like:

> "The student selected 9 incorrect water sources during the watershed identification activity. The threshold for success is 6 or fewer incorrect selections. Consider reviewing the difference between surface water and groundwater sources."

---

## What Is Required

For each grading rule that can produce yellow, we need:

### 1. Reason Codes

A finite set of machine-readable codes that categorize why a grade is yellow.

**Example for Unit 2, Point 3:**
```
TOO_MANY_TARGETS     - Student exceeded the error threshold
DURATION_EXCEEDED    - Activity took too long (possible clock issue)
MISSING_END_EVENT    - Student didn't complete the activity
```

### 2. Message Templates

Human-readable templates that incorporate metrics to explain the grade.

**Example:**
```
Code: TOO_MANY_TARGETS
Template: "The student made {countTargets} incorrect selections during {activityName}.
           The threshold for success is {threshold} or fewer.
           {teacherGuidance}"

Variables:
  - countTargets: from metrics.countTargets
  - threshold: from metrics.threshold
  - activityName: "the watershed identification activity"
  - teacherGuidance: "Consider reviewing the difference between surface water and groundwater sources."
```

### 3. Activity Context

Information about what each progress point represents in the curriculum:
- Activity name/description
- Learning objectives
- Common misconceptions
- Remediation suggestions

---

## Information Needed from Curriculum Team

For each progress point that can be yellow, please provide:

### A. Reason Code Definitions

| Progress Point | Reason Code | When This Occurs |
|----------------|-------------|------------------|
| U1P3 | WRONG_PATH_SELECTED | Student chose dialogue path 70:25 or 70:33 |
| U2P1 | MISSING_SUCCESS_NODE | Student never reached DialogueNodeEvent:68:29 |
| U2P1 | HIT_YELLOW_NODE | Student triggered one of the "wrong answer" dialogue nodes |
| U2P2 | TOO_MANY_TARGETS | More than 1 incorrect selection in the activity window |
| U2P2 | BAD_DURATION | Activity duration was negative or exceeded 2 hours |
| U2P3 | TOO_MANY_TARGETS | More than 6 incorrect selections in the activity window |
| U2P4 | MISSING_SUCCESS | Student never reached the success dialogue node |
| U2P4 | BAD_FEEDBACK | Student received negative feedback during the activity |
| U2P5 | LOW_SCORE | Calculated score was below 4 |
| U2P6 | MISSING_PASS_NODE | Student never passed through dialogue:20:43 |
| U2P6 | HIT_YELLOW_NODE | Student triggered dialogue:20:44 or dialogue:20:45 |
| U2P7 | MISSING_SUCCESS | Student never reached DialogueNodeEvent:27:7 |
| U2P7 | TOO_MANY_NEGATIVES | More than 3 negative events occurred |

### B. Teacher-Facing Messages

For each reason code, provide:

1. **Short Description** (for tooltip, ~50 chars max)
   > "Too many incorrect water source selections"

2. **Full Explanation** (for detail modal, 1-2 sentences)
   > "The student selected 9 incorrect water sources during the watershed identification activity. The threshold for success is 6 or fewer incorrect selections."

3. **Teacher Guidance** (optional, for remediation suggestions)
   > "Consider reviewing the difference between surface water and groundwater sources with this student."

### C. Activity Names

Map each progress point to a human-readable activity name:

| Progress Point | Activity Name |
|----------------|---------------|
| U1P1 | "Get Your Space Legs" tutorial |
| U1P2 | "Meet the Team" introduction |
| U1P3 | "First Mission Briefing" |
| U2P1 | "Water Cycle Basics" |
| U2P2 | "Watershed Identification" |
| U2P3 | "Water Source Classification" |
| ... | ... |

---

## How It Works

### Data Flow

```
1. Grading Service evaluates a progress point
   └─> Produces: color, reasonCode, metrics

2. Grade is stored in progress_point_grades collection
   └─> { playerId, unit, point, color, reasonCode, metrics }

3. Dashboard loads grades for a class
   └─> Renders green/yellow/white squares

4. Teacher clicks a yellow square
   └─> Dashboard looks up reasonCode in message catalog
   └─> Substitutes metrics into template
   └─> Displays human-readable message
```

### Message Catalog Structure

The message catalog is a configuration file (JSON or embedded in code):

```json
{
  "u2p3": {
    "activityName": "Water Source Classification",
    "reasons": {
      "TOO_MANY_TARGETS": {
        "short": "Too many incorrect classifications",
        "template": "The student made {countTargets} incorrect classifications. The threshold is {threshold} or fewer.",
        "guidance": "Review the properties that distinguish different water source types."
      },
      "BAD_DURATION": {
        "short": "Activity timing issue",
        "template": "The activity duration was invalid (possibly due to clock synchronization). The student should retry this activity.",
        "guidance": null
      }
    }
  }
}
```

### Template Variable Substitution

The dashboard (or a helper service) substitutes values from `metrics`:

```go
// Pseudocode
message := template
message = strings.Replace(message, "{countTargets}", fmt.Sprint(metrics.CountTargets), -1)
message = strings.Replace(message, "{threshold}", fmt.Sprint(metrics.Threshold), -1)
// etc.
```

---

## Storage Design

### In `progress_point_grades` Document

```js
{
  playerId: "student@mhs.mhs",
  grades: {
    "u2p3": {
      color: "yellow",
      reasonCode: "TOO_MANY_TARGETS",  // Machine-readable
      metrics: {
        countTargets: 9,
        threshold: 6
      },
      computedAt: ISODate("...")
    }
  }
}
```

### Message Catalog (Separate Config)

Stored as:
- JSON file in the application (`resources/mhs_reason_messages.json`)
- Or in a database collection (`reason_message_catalog`)

The catalog is **not duplicated** in every grade record. Only the `reasonCode` and `metrics` are stored per grade. The human-readable text is looked up at display time.

---

## Implementation Phases

### Phase 1: Grading (Current Focus)
- Grading service produces `color` values
- Dashboard displays green/yellow/white squares
- `reasonCode` and `metrics` are stored but not yet displayed

### Phase 2: Basic Reasons
- Add `reasonCode` to grading rules
- Store reason codes with grades
- Dashboard shows reason code as tooltip (raw, not pretty)

### Phase 3: Human-Readable Messages
- Create message catalog with templates
- Dashboard substitutes metrics into templates
- Teachers see meaningful explanations

### Phase 4: Teacher Guidance
- Add remediation suggestions to catalog
- Dashboard shows "How to help" section
- Link to relevant curriculum resources

---

## Questions for Curriculum Team

1. For each yellow reason, what should the teacher understand about the student's performance?

2. Are there common misconceptions associated with each activity that should be mentioned?

3. Should some yellow grades link to specific remediation activities or resources?

4. Are there cases where yellow should be accompanied by specific data (e.g., "the student selected: X, Y, Z" listing their actual wrong choices)?

5. Should teachers be able to override a yellow to green manually? If so, under what circumstances?

---

## Next Steps

1. **Curriculum team**: Review the reason codes table above and provide teacher-facing messages for each.

2. **Development team**: Implement grading service with `reasonCode` and `metrics` capture.

3. **Design team**: Create UI mockups for the "why yellow" tooltip/modal.

4. **Integration**: Connect message catalog to dashboard display.

---

## Contact

For questions about this system:
- **Technical implementation**: [Development team contact]
- **Curriculum content**: [Curriculum team contact]
- **Dashboard UX**: [Design team contact]
