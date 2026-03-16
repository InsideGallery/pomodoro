# Fingerprint Lab — Implementation Guide

## Source of Truth

- **`fingerprint.tmx`** — all layout, positions, layer structure (4000×2176, 125×68 tiles at 32px)
- **`stories.json`** — 50 case narratives for city of Muldrow
- **`assets/external/fingerprint/README.md`** — game logic and behavior spec
- TMX loaded with `github.com/lafriks/go-tiled` via `pkg/tilemap/`

## TMX Structure

### Tilesets
- `fingerprinting-icon.tsx` — app icon
- `avatars.tsx` — character avatars + unknown placeholder
- `buttons.tsx` — UI buttons (back, send, exit, success, fail, cursor, etc.)

### Game States (layer visibility)

**StateLoading** — deferred asset loading with progress bar
- No TMX layers — just "Loading..." text + animated progress bar
- Heavy work in goroutines: TMX (~119MB backgrounds), DB, puzzles

**StateDisabled** — PC off → boot animation
- imagelayer "disabled" → `background/background-disabled.png`
- Cross-fade to enabled over 90 frames (~1.5s)

**StateEnabled** — desktop
- imagelayer "enabled" → `background/background-enabled.png`
- tilelayer "enabled" → fingerprinting app icon
- objectgroup "enabled":
  - `button-run-fingerprint` (ellipse, 1376,416 126×126) → open app
  - `button-quit-os` (polygon, 1305,1571) → quit game (drawn programmatically 200×50 red)

**StateApplicationLayout** — case selection
- KEEP imagelayer "enabled"
- imagelayer "application-layout" → `background/fingerprint-select.png`
- tilelayer "application-layout" → UI elements
- objectgroup "application-layout":
  - `list-of-cases` (1452,486 346×921) — scrollable case buttons (50 cases, rowH=45)
  - `fingerprints-user-names` (1884,516 290×932) — scrollable puzzle buttons (20/case, rowH=50)
  - `avatar` (2273,492 330×313) — unkown.jpg or character avatar when solved
  - `description` (2260,857 362×564) — word-wrapped, scrollable narrative
  - `exit` (2547,361 141×47) → back to desktop
  - `play-puzzle` (2260,1425 362×58) → open puzzle (drawn programmatically, teal)
  - `regenerate-puzzles` (1454,1408 341×73) → regenerate all puzzles (drawn programmatically, orange)

**StateApplicationNet** — puzzle workspace
- KEEP imagelayer "enabled"
- imagelayer "application-net-layout" → `background/fingerprint-ui-net.png`
- tilelayer "application-net-layout" → UI elements (includes send button design)
- objectgroup "application-net-layout":
  - `puzzle` (1692,562 680×684) — 10×10 fingerprint grid (forced square, cell ≈ 68px)
  - `pieces` (2393,566 263×939) — tray room #1 for draggable pieces
  - `pieces` (1401,566 269×930) — tray room #2 for draggable pieces
  - `hash` (1533,357 606×45) — live hash display
  - `button-send-puzzle` (1790,1315 518×93) — submit button
  - `back` (1364,357 141×47) → back to case selection
  - `exit` (2546,357 142×46) → back to desktop
  - `drag-and-drop-zone` (1380,440 1288×1088) — full working area

**Result overlays** (tile layers, shown 3 seconds on submit):
- `application-net-layout-success`
- `application-net-layout-fail`

**Global**: objectgroup "main" → `cursor-room` (1248,318 1527×1303)

## Assets

```
background/                     — 4 large PNGs (~30MB each, 4000×2176)
ui/                             — cursor.png, buttons, icons
avatars/                        — m.{Rob Malfoy}.jpg, m.{Steve Gilber}.jpg,
                                  w.{Elizabet Queen}.jpg, w.{May Forty}.jpg, unkown.jpg
fingerprints/                   — {color}.{1-4}.png, grey.{1-4}.png (16+4 images)
stories.json                    — 50 case narratives
```

## Database: 256 Fingerprints (db.json)

Generated deterministically on first run, only regenerated if db.json deleted.

```
4 colors × 4 variants × 8 rotations × 2 mirror = 256 records
```

Each record:
```json
{
  "id": 1,
  "color": "green",
  "variant": 2,
  "rotation": 45,
  "mirrored": true,
  "hash": "G14976251236816614454",
  "pieces": [{"x":0,"y":0,"value":...}, ... 100 items],
  "person_name": "Rob Malfoy",
  "avatar_key": "m.{Rob Malfoy}.jpg"
}
```

### Hash Algorithm
- Each piece: `uint32 = x | (y<<8) | (rotation<<16) | (content<<24)`
- Content: random uint8 per piece, unique per record
- Hash: CRC64-ECMA over all 100 uint32 values in byte order
- Full hash: `"{LETTER}{CRC64}"` where LETTER = G/R/Y/B (or ? for hidden color)

### 4 Characters (cycle by record ID)
| ID mod 4 | Name | Avatar |
|-----------|------|--------|
| 1 | Rob Malfoy | m.{Rob Malfoy}.jpg |
| 2 | Steve Gilber | m.{Steve Gilber}.jpg |
| 3 | Elizabet Queen | w.{Elizabet Queen}.jpg |
| 0 | May Forty | w.{May Forty}.jpg |

## Puzzles: 50 Cases × 20 Puzzles (puzzles.json)

### Difficulty Tiers
| Cases | Difficulty | Missing Pieces | Color |
|-------|-----------|---------------|-------|
| 0–19 | EASY | 3 | random show/hide |
| 20–34 | MEDIUM | 6 | random show/hide |
| 35–49 | HARD | 12 | always hidden (grey) |

### Corner Exclusion
12 corner pieces never removed (L-shape at each corner):
`{0, 1, 10, 8, 9, 19, 80, 90, 91, 89, 98, 99}`
→ 88 valid piece indices

### Decoy Groups (fake pieces)
For N missing pieces, the tray contains exactly **3×N** pieces:
- **N correct** pieces from the target fingerprint
- **N fake** pieces from source fingerprint A (different variant, any color)
- **N fake** pieces from source fingerprint B (different variant, any color)

Decoy selection uses its own deterministic RNG seeded from `seed ^ targetID`.
Fake fingerprint images loaded with the TARGET's rotation+mirror angle so pieces are visually consistent.

### Grey (Hidden Color) Puzzle
- Grid shows grey version of fingerprint
- Tray pieces show real colors (all groups)
- Hash prefix = "?"
- On submit: tries all 4 color letters (G/R/Y/B). If match found, reveals color.

## Image Pipeline

```
Source PNG
  → cropCentered(480×480)
  → scaleImage(690×690)            [nearest-neighbor upscale]
  → rotateImage(degrees)           [0°/45°/90°/.../315°]
  → mirrorImage(if mirrored)
  → cropCentered(690×690)          [for 45° angles: larger intermediate]
  → cut 10×10 grid                 [100 pieces, 69×69 each]
```

For 45° angles: crop 980×980, upscale to ~976×976, rotate, then crop center 690×690.

## Virtual Cursor
Delta-based to prevent stickiness at cursor-room edges:
```
dx = rawX - prevRawX; dy = rawY - prevRawY
cursorX += dx; cursorY += dy
clamp(cursorX, room.minX, room.maxX)
clamp(cursorY, room.minY, room.maxY)
```
Fed to InputSystem via `CursorOverride`.

## Drag-and-Drop
- **Mouse press** on tray piece → start dragging
- **Mouse press** on placed grid piece → unplace, start dragging
- **Mouse wheel** while dragging → rotate piece (up=CW, down=CCW, 8 steps of 45°)
- **Mouse release** on empty missing grid cell → place piece
- **Mouse release** elsewhere → drop at cursor position in drag-and-drop-zone

Pieces have free-form TrayX/TrayY positions (map coordinates), not grid-aligned.

## Save System
Three files in `~/.config/pomodoro/fingerprint/`:
- **db.json** — 256 fingerprint records (static)
- **puzzles.json** — 50×20 puzzle configs (regenerated by REGENERATE button)
- **save.json** — per-puzzle progress: solved/failed, placed pieces + tray positions

## Lazy Loading
`Load()` only sets `StateLoading`. Heavy work runs in background goroutines:
1. stories.json (instant) + TMX loading goroutine (~119MB PNGs)
2. DB loading/generation goroutine
3. Puzzle loading/generation goroutine
4. Ready → StateDisabled

Fingerprint images loaded on demand:
- Target: `ensureCurrentPuzzleImages()` when entering puzzle workspace
- Decoys: `getPieceImage()` lazy-loads on first render
- Avatars: `getAvatar()` lazy-loads from disk with cache

## Scrollable UI Areas
Three independently scrollable areas in application-layout (mouse wheel over area):
- **Cases list** (left column) — 50 cases with progress counters
- **Puzzle names** (middle column) — 20 puzzles per case, Name or "Unknown"
- **Description** (right panel) — word-wrapped narrative text
