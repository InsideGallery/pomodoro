# Fingerprint Lab — Implementation Guide

## Source of Truth

**`fingerprint.tmx`** is the ONLY source for layout, positions, and layer structure.
Load it with `github.com/lafriks/go-tiled` (same library as InsideGallery/detective).
Do NOT hardcode any positions — read everything from TMX objects.

## TMX Structure

Map: 125×68 tiles, 32×32px each = **4000×2176 pixels**

### Tilesets
- `fingerprinting-icon.tsx` — app icon (128×128)
- `avatars.tsx` — 6 avatars (5 suspects + 1 "unknown")
- `buttons.tsx` — all UI buttons (back, code, send, place, exit, success, fail, cursor)

### Layer Groups (states of the game)

**State: "disabled" (PC off)**
- `disabled` imagelayer → `background/background-disabled.png` (4000×2176)

**State: "enabled" (desktop)**
- `enabled` imagelayer → `background/background-enabled.png` (4000×2176)
- `enabled` tilelayer → fingerprinting icon placed at tile position
- `enabled` objectgroup:
  - `button-run-fingerprint` (type: button-play) — x:1376, y:416, 126×126
  - `button-quit-os` (type: button-quit) — x:1305, y:1571 (polygon)

**State: "application-layout" (case selection)**
- Keep `enabled` imagelayer
- `application-layout` imagelayer → `background/fingerprint-select.png`
- `application-layout` tilelayer → UI buttons/avatars placed by tiles
- `application-layout` objectgroup:
  - `list-of-cases` — x:1452, y:486, 346×996 (room for case buttons)
  - `fingerprints-user-names` — x:1884, y:516, 290×932 (room for fingerprint buttons)
  - `avatar` — x:2273, y:492, 330×313 (avatar display area)
  - `description` — x:2260, y:857, 362×564 (person description area)
  - `exit` — x:2547, y:361, 141×47 (exit button)
  - `play-puzzle` — x:2260, y:1425, 362×58 (open puzzle button)

**State: "application-net-layout" (puzzle workspace)**
- Keep `enabled` imagelayer
- `application-net-layout` imagelayer → `background/fingerprint-ui-net.png`
- `application-net-layout` tilelayer → UI elements
- `application-net-layout` objectgroup:
  - `puzzle` — x:1692, y:562, 680×684 (10×10 fingerprint grid area)
  - `pieces` — x:2393, y:566, 269×672 (piece tray area)
  - `hash` — x:1533, y:357, 606×45 (hash display)
  - `back` — x:1364, y:357, 141×47 (back to cases)
  - `exit` — x:2546, y:357, 142×46 (exit to desktop)
  - `drag-and-drop-zone` — x:1380, y:440, 1288×1088

**Global**
- `main` objectgroup: `cursor-room` — x:1248, y:318, 1527×1303

### Success/Fail tile layers
- `application-net-layout-success` tilelayer (visible=0)
- `application-net-layout-fail` tilelayer (visible=0)

## Asset Paths (restructured)

```
background/
  background-disabled.png     (4000×2176, PC off)
  background-enabled.png      (4000×2176, PC on/desktop)
  fingerprint-select.png      (4000×2176, case selection app)
  fingerprint-ui-net.png      (4000×2176, puzzle workspace)

ui/
  cursor.png                  (63×62)
  fingerprinting-icon.png     (128×128)
  back-activated.png          (376×176)
  exit-activated.png          (128×65)
  code -button.png            (554×118)
  code -button-activated.png  (554×118)
  place-button.png            (949×137)
  place-button-activated.png  (949×137)
  send-button.png             (489×128)
  send-button-  activated.png (588×154)
  success-button.png          (653×169)
  fail-button.png             (653×169)
  highlighter.png

avatars/
  1-5.jpg + unkown.jpg        (311×311 or 660×660)

fingerprints/
  {color}.{1-4}.png           (full fingerprint images)
  grey.{1-4}.png              (grey variants)
```

## Implementation Steps

### Step 1: TMX Loading
- Add `github.com/lafriks/go-tiled` dependency
- Parse `fingerprint.tmx` in the preloader
- Extract layer references, object positions, tile data
- Reuse patterns from `InsideGallery/detective/internal/tilemap/`

### Step 2: Single Scene with State Machine
Instead of multiple scenes, use ONE scene with layer visibility toggling:
- State: disabled → enabled → application-layout → application-net-layout
- Each state shows/hides specific layers
- Object groups for each state define clickable zones

### Step 3: Rendering
- Image layers: draw background PNGs at (0,0) scaled to screen
- Tile layers: iterate non-zero tiles, draw tileset images at tile positions
- Object layers: create RTree zones from object positions

### Step 4: Game Logic
Per README.md instructions:
- Choose color (G/R/Y/B), fingerprint variant (1-4), rotation, mirror
- Load fingerprint from `fingerprints/{color}.{variant}.png`
- Scale to 690×690, apply rotation + mirror
- Cut into 10×10 grid (69×69 each piece)
- Generate uint32 per piece from x,y coordinates
- Compute CRC64 hash of all pieces = correct hash
- Remove {pieces-to-solve} random pieces (4-16 depending on case)
- Add decoy pieces from other fingerprints (5 random from each other variant)
- Show removed pieces in piece tray with random rotation
- Player drags pieces to grid, can rotate with mouse wheel
- Hash computed live as pieces are placed
- Color may be hidden (grey fingerprint, colored pieces only)
- Submit → compare hash → SUCCESS or FAIL

### Step 5: Camera
- Zoom hotkey around center of desktop
- Reset hotkey
- Use pkg/core/Camera for world matrix
