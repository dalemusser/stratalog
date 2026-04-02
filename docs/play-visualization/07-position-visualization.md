# Position Visualization (Phase 2)

## Purpose

Plot `PlayerPositionEvent` data as 2D movement paths on a canvas, showing where students go during gameplay. This helps understand:
- Whether students are exploring or stuck in one area
- Movement patterns during progress points (e.g., wandering during a puzzle)
- Where students spend time between activities
- Whether movement correlates with gameplay events

## Data Source

Position events have this structure:
```json
{
  "eventType": "PlayerPositionEvent",
  "sceneName": "Unit 3 Dev",
  "serverTimestamp": "2026-03-30T16:24:45.142Z",
  "data": {
    "position": {
      "x": 12.5,
      "y": 0.0,
      "z": -34.2
    }
  }
}
```

The game world is 3D but movement is primarily on the XZ plane (Y is elevation, mostly constant). Plot using X (horizontal) and Z (vertical).

## Scene Bounds

Derive bounds per scene from the actual position data:

```go
// ServeDebugPositions handler
// 1. Query all PlayerPositionEvent entries for the student + unit
// 2. Extract x, z values
// 3. Compute min/max for bounds with padding
// 4. Return as JSON
```

Response format:
```json
{
  "scene": "Unit 3 Dev",
  "bounds": {
    "minX": -120.5,
    "maxX": 85.3,
    "minZ": -90.1,
    "maxZ": 110.7
  },
  "positions": [
    {
      "x": 12.5,
      "z": -34.2,
      "time": "2026-03-30T16:24:45.142Z",
      "scene": "Unit 3 Dev",
      "index": 0
    }
  ],
  "waypoints": [
    {
      "x": 15.2,
      "z": -30.1,
      "pointID": "u3p1",
      "type": "start",
      "time": "2026-03-30T16:22:18.348Z"
    }
  ]
}
```

The `waypoints` array contains positions of the nearest `PlayerPositionEvent` to each progress point start/end event, so we can mark where in the world those activities began/ended.

## Canvas Rendering

### HTML Structure

```html
<div id="mhs-position-container" class="relative">
  <div class="flex gap-2 mb-2">
    <select id="mhs-position-scene">
      <option value="all">All Scenes</option>
      <!-- populated from data -->
    </select>
    <label class="text-xs">
      <input type="checkbox" id="mhs-show-waypoints" checked> Show waypoints
    </label>
    <label class="text-xs">
      <input type="checkbox" id="mhs-show-direction" checked> Show direction
    </label>
  </div>
  <canvas id="mhs-position-canvas" width="800" height="500"></canvas>
  <div id="mhs-position-tooltip" class="hidden absolute bg-gray-800 text-white text-xs p-2 rounded"></div>
</div>
```

### Rendering Logic (JavaScript)

```javascript
function renderPositionPlot(data) {
  const canvas = document.getElementById('mhs-position-canvas');
  const ctx = canvas.getContext('2d');
  const { bounds, positions, waypoints } = data;

  // Scale game coordinates to canvas pixels
  const scaleX = canvas.width / (bounds.maxX - bounds.minX);
  const scaleZ = canvas.height / (bounds.maxZ - bounds.minZ);

  function toCanvas(x, z) {
    return {
      cx: (x - bounds.minX) * scaleX,
      cy: (z - bounds.minZ) * scaleZ
    };
  }

  // Draw movement path as connected line
  ctx.beginPath();
  ctx.strokeStyle = 'rgba(59, 130, 246, 0.6)'; // blue
  ctx.lineWidth = 1.5;
  positions.forEach((p, i) => {
    const { cx, cy } = toCanvas(p.x, p.z);
    if (i === 0) ctx.moveTo(cx, cy);
    else ctx.lineTo(cx, cy);
  });
  ctx.stroke();

  // Draw start point (green dot)
  if (positions.length > 0) {
    const start = toCanvas(positions[0].x, positions[0].z);
    drawDot(ctx, start.cx, start.cy, 6, '#22c55e');
  }

  // Draw end point (red dot)
  if (positions.length > 1) {
    const end = toCanvas(positions[positions.length-1].x, positions[positions.length-1].z);
    drawDot(ctx, end.cx, end.cy, 6, '#ef4444');
  }

  // Draw waypoints (progress point markers)
  if (showWaypoints) {
    waypoints.forEach(wp => {
      const { cx, cy } = toCanvas(wp.x, wp.z);
      const color = wp.type === 'start' ? '#22c55e' : '#3b82f6';
      drawDiamond(ctx, cx, cy, 8, color);
      drawLabel(ctx, cx + 10, cy, wp.pointID);
    });
  }
}
```

### Visual Features

1. **Path line**: Continuous line connecting all positions, with opacity gradient from start (bright) to end (faded) to show temporal direction
2. **Start/end markers**: Green circle at first position, red circle at last
3. **Waypoint markers**: Diamond shapes at positions nearest to progress point events, labeled with point ID
4. **Scene boundaries**: When switching between scenes (e.g., Unit 3 Dev → Unit 3 Dungeon Dev), draw a visual break in the path
5. **Hover tooltip**: Mouse over any point on the path to see timestamp and nearest event
6. **Density heatmap** (optional): Areas where the student spent the most time shown as warmer colors

### Multi-Scene Handling

When "All Scenes" is selected and a unit has multiple scenes (e.g., Unit 4 has Dev, Dungeon, Anderson Base):

Option A: Separate canvases per scene, stacked vertically
Option B: Combined view with different path colors per scene
Option C: Scene selector dropdown (default)

Recommend **Option C** (scene selector) for simplicity. Each scene has its own coordinate system, so combining them on one canvas would be misleading.

## Performance

- Position events are ~20% of total events (~500 for a full playthrough, ~100 per unit)
- Canvas rendering of 500 points is fast
- Larger datasets (thousands of positions) may need point thinning (skip every Nth point)
- Data is loaded via a JSON endpoint, rendered client-side — no server-side image generation needed

## Implementation Order

1. JSON endpoint that returns position data with bounds (server-side)
2. Basic canvas with path rendering (client-side)
3. Scene selector and waypoint overlay
4. Hover tooltips
5. Direction indicators (optional: small arrows along the path)
