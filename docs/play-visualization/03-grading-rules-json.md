# Embedded Grading Rules JSON

## Purpose

The debug timeline needs to know which log events correspond to progress point boundaries (start/end triggers) and which events are graded within each progress point window. This information currently lives in mhsgrader's Go rule implementations. We extract the relevant eventKeys into a JSON config file embedded in stratahub.

## File Location

```
stratahub/internal/app/resources/mhs_grading_rules.json
```

Embedded alongside the existing `mhs_progress_points.json` via the `//go:embed` directive in `resources.go`.

## JSON Structure

```json
{
  "unit_start_events": {
    "unit1": "questActiveEvent:28",
    "unit2": "DialogueNodeEvent:18:1",
    "unit3": "DialogueNodeEvent:10:1",
    "unit4": "DialogueNodeEvent:88:0",
    "unit5": "questActiveEvent:43"
  },
  "rules": [
    {
      "rule_id": "u1p1_v2",
      "point_id": "u1p1",
      "unit": 1,
      "point": 1,
      "activity_name": "Getting Your Space Legs",
      "start_keys": ["questActiveEvent:28"],
      "trigger_keys": ["DialogueNodeEvent:31:29"],
      "grading_type": "completion",
      "evaluated_keys": {}
    },
    {
      "rule_id": "u1p3_v2",
      "point_id": "u1p3",
      "unit": 1,
      "point": 3,
      "activity_name": "Defend the Expedition",
      "start_keys": ["DialogueNodeEvent:30:98"],
      "trigger_keys": ["questActiveEvent:34"],
      "grading_type": "yellow_count",
      "evaluated_keys": {
        "yellow": ["DialogueNodeEvent:70:25", "DialogueNodeEvent:70:33"]
      }
    }
  ]
}
```

### Field Definitions

| Field | Description |
|-------|-------------|
| `rule_id` | Matches the rule ID in mhsgrader (e.g., `u1p3_v2`) |
| `point_id` | Progress point ID matching `mhs_progress_points.json` (e.g., `u1p3`) |
| `unit` | Unit number (1-5) |
| `point` | Point number within unit |
| `activity_name` | Human-readable activity name |
| `start_keys` | eventKey(s) that set the grade to "active" |
| `trigger_keys` | eventKey(s) that trigger grading evaluation |
| `grading_type` | Classification: `completion`, `yellow_count`, `score_based`, `timestamp_window` |
| `evaluated_keys` | Map of key categories used during evaluation |

### Evaluated Key Categories

Depending on `grading_type`, the `evaluated_keys` map may contain:

- `yellow` — events that count against the student (wrong answers, wrong directions)
- `positive` / `success` — events that indicate correct behavior
- `negative` — events that indicate incorrect behavior
- `bonus_event_type` — non-eventKey event types checked during evaluation (e.g., `argumentationToolEvent`)
- `custom_event_type` — event types queried by type rather than eventKey (e.g., `soilMachine`, `WaterChamberEvent`, `TerasGardenBox`, `Soil Key Puzzle`)

## Complete Rules Data

All 24 rules with their eventKeys are documented below. This serves as the source for generating the JSON file.

### Unit 1

| Rule | Start Key | Trigger Key | Grading | Evaluated Keys |
|------|-----------|-------------|---------|----------------|
| u1p1_v2 | `questActiveEvent:28` | `DialogueNodeEvent:31:29` | completion | — |
| u1p2_v2 | `DialogueNodeEvent:31:29` | `DialogueNodeEvent:30:98` | completion | — |
| u1p3_v2 | `DialogueNodeEvent:30:98` | `questActiveEvent:34` | yellow_count | yellow: `70:25`, `70:33` |
| u1p4_v2 | `questActiveEvent:34` | `DialogueNodeEvent:33:19` | completion | — |

### Unit 2

| Rule | Start Key | Trigger Key | Grading | Evaluated Keys |
|------|-----------|-------------|---------|----------------|
| u2p1_v2 | `DialogueNodeEvent:18:1` | `questFinishEvent:21` | yellow_count | success: `68:29`; yellow: `68:22`, `68:23`, `68:27`, `68:28`, `68:31` |
| u2p2_v2 | `questFinishEvent:21` | `DialogueNodeEvent:20:26` | yellow_count | yellow: `18:99`, `28:179`, `59:179`, `18:223`, `28:182`, `59:182`, `18:224`, `28:183`, `59:183` |
| u2p3_v2 | `DialogueNodeEvent:20:33` | `DialogueNodeEvent:22:18` | timestamp_window | yellow: 31 DialogueNodeEvent keys (wrong-direction prompts across dialogues 18, 28, 59) |
| u2p4_v2 | `DialogueNodeEvent:22:18` | `DialogueNodeEvent:23:17` | yellow_count | success: `74:21`; yellow: `74:16`, `74:17`, `74:20`, `74:22` |
| u2p5_v2 | `DialogueNodeEvent:23:17` | `DialogueNodeEvent:23:42` | yellow_count | positive: 22 keys (`26:165`–`26:186`); negative: 25 keys (`26:187`–`26:211`) |
| u2p6_v2 | `DialogueNodeEvent:23:42` | `DialogueNodeEvent:20:46` | yellow_count | pass: `20:43`; yellow: `20:44`, `20:45` |
| u2p7_v2 | `DialogueNodeEvent:20:46` | `questFinishEvent:54` | yellow_count | success: `27:7`; negative: 20 keys (`27:11`–`27:30`) |

### Unit 3

| Rule | Start Key | Trigger Key | Grading | Evaluated Keys |
|------|-----------|-------------|---------|----------------|
| u3p1_v2 | `DialogueNodeEvent:10:1` | `DialogueNodeEvent:11:22` | yellow_count | yellow: `10:30` |
| u3p2_v2 | `questFinishEvent:17` | `DialogueNodeEvent:11:34` | yellow_count | yellow: `11:27`, `11:29`, `11:230` |
| u3p3_v2 | `DialogueNodeEvent:11:34` | `questFinishEvent:18` | score_based | yellow: 18 keys (`84:20`–`84:47`); bonus_event_type: `argumentationToolEvent` |
| u3p4_v2 | `questFinishEvent:18` | `DialogueNodeEvent:73:200` | yellow_count | gate: `78:24`; yellow: 8 keys (`78:3`–`78:23`) |
| u3p5_v2 | `DialogueNodeEvent:73:200` | `DialogueNodeEvent:10:194` | yellow_count | positive: `73:163`; negative: `73:164`, `73:168`, `73:171` |

### Unit 4

| Rule | Start Key | Trigger Key | Grading | Evaluated Keys |
|------|-----------|-------------|---------|----------------|
| u4p1_v2 | `DialogueNodeEvent:88:0` | `questActiveEvent:39` | score_based | success: `88:5`; custom_event_type: `Soil Key Puzzle` |
| u4p2_v2 | `questActiveEvent:39` | `questActiveEvent:48` | score_based | success: `88:11`; yellow: `102:9`, `102:10`, `102:12`, `102:18`, `102:23` |
| u4p3_v2 | `questActiveEvent:48` | `questActiveEvent:50` | score_based | custom_event_type: `soilMachine` (floor 3-4) |
| u4p4_v2 | `questActiveEvent:50` | `questActiveEvent:36` | score_based | custom_event_type: `soilMachine` (floor 5); success: `107:4`, `107:5`; negative: `107:2`, `107:3`, `107:6` |
| u4p5_v2 | `questActiveEvent:36` | `questActiveEvent:41` | yellow_count | positive: `90:50`, `90:57`; negative: 13 keys (`90:25`–`90:61`) |
| u4p6_v2 | `questActiveEvent:41` | `questFinishEvent:56` | score_based | custom_event_type: `TerasGardenBox` |

### Unit 5

| Rule | Start Key | Trigger Key | Grading | Evaluated Keys |
|------|-----------|-------------|---------|----------------|
| u5p1_v2 | `questActiveEvent:43` | `questFinishEvent:43` | score_based | success: `100:44`; negative: `100:38`, `100:39`, `100:43` |
| u5p2_v2 | `questFinishEvent:43` | `DialogueNodeEvent:96:1` | score_based | custom_event_type: `WaterChamberEvent` (floor 3-4) |
| u5p3_v2 | `DialogueNodeEvent:96:1` | `questFinishEvent:44` | yellow_count | negative: 31 keys (`108:25`–`108:91`) |
| u5p4_v2 | `questFinishEvent:44` | `questFinishEvent:45` | yellow_count | success: `106:35`; negative: 11 keys (`106:4`–`106:34`) |

## Updating the JSON

When mhsgrader rules change (new version, new eventKeys), update `mhs_grading_rules.json` to match. The rule_id version suffix (e.g., `_v2`) provides a way to verify alignment between the JSON and the running grader.

## Go Types

```go
// GradingRulesConfig is loaded from mhs_grading_rules.json
type GradingRulesConfig struct {
    UnitStartEvents map[string]string `json:"unit_start_events"`
    Rules           []GradingRule     `json:"rules"`
}

type GradingRule struct {
    RuleID        string              `json:"rule_id"`
    PointID       string              `json:"point_id"`
    Unit          int                 `json:"unit"`
    Point         int                 `json:"point"`
    ActivityName  string              `json:"activity_name"`
    StartKeys     []string            `json:"start_keys"`
    TriggerKeys   []string            `json:"trigger_keys"`
    GradingType   string              `json:"grading_type"`
    EvaluatedKeys map[string][]string `json:"evaluated_keys"`
}
```

Loaded via `sync.Once` alongside the existing `ProgressConfig`, with a helper to build an `eventKey → []GradingRule` index for fast lookup when annotating the event timeline.
