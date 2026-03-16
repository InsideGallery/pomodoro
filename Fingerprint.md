# Fingerprint Lab — Implementation Guide

## Source of Truth

- **`fingerprint.tmx`** — all layout, positions, layer structure
- **`assets/external/fingerprint/README.md`** — game logic and behavior spec
- Load TMX with `github.com/lafriks/go-tiled` (reused from detective)

## TMX Structure (4000×2176, 125×68 tiles at 32px)

### Tilesets
- `fingerprinting-icon.tsx` — app icon (128×128)
- `avatars.tsx` — 6 images (5 suspects + "unknown")
- `buttons.tsx` — 12 UI buttons (back, code, send, place, exit, success, fail, cursor, icon)

### Game States (layer visibility)

**disabled** — PC off
- imagelayer "disabled" → `background/background-disabled.png`

**enabled** — desktop
- imagelayer "enabled" → `background/background-enabled.png`
- tilelayer "enabled" → fingerprinting icon at tile position
- objectgroup "enabled":
  - `button-run-fingerprint` (1376,416 126×126) → open app
  - `button-quit-os` (1305,1571 polygon) → quit game

**application-layout** — case selection (3-column)
- KEEP imagelayer "enabled"
- imagelayer "application-layout" → `background/fingerprint-select.png`
- tilelayer "application-layout" → UI buttons/avatars
- objectgroup "application-layout":
  - `list-of-cases` (1452,486 346×996) — room for case buttons
  - `fingerprints-user-names` (1884,516 290×932) — room for name buttons
  - `avatar` (2273,492 330×313) — avatar display
  - `description` (2260,857 362×564) — person details
  - `exit` (2547,361 141×47) → back to desktop
  - `play-puzzle` (2260,1425 362×58) → open puzzle

**application-net-layout** — puzzle workspace
- KEEP imagelayer "enabled"
- imagelayer "application-net-layout" → `background/fingerprint-ui-net.png`
- tilelayer "application-net-layout" → UI elements
- objectgroup "application-net-layout":
  - `puzzle` (1692,562 680×684) — 10×10 fingerprint grid
  - `pieces` (2393,566 269×672) — draggable piece tray
  - `hash` (1533,357 606×45) — hash display
  - `back` (1364,357 141×47) → back to cases
  - `exit` (2546,357 142×46) → back to desktop
  - `drag-and-drop-zone` (1380,440 1288×1088)

**Result overlays** (tile layers, shown on submit):
- `application-net-layout-success`
- `application-net-layout-fail`

**Global**: objectgroup "main" → `cursor-room` (1248,318 1527×1303)

## Assets

```
background/                     — 4 image layers (4000×2176 each)
ui/                             — buttons, cursor, icon, highlighter
avatars/                        — 1-5.jpg + unkown.jpg
fingerprints/                   — {color}.{1-4}.png + grey.{1-4}.png
```

## Implementation Steps

### Step 1: TMX Loading + State Machine ✅ DONE
- pkg/tilemap/ — TMX loader with image caching
- Single GameScene with 4 states
- Layer rendering per state
- Object groups → RTree click zones
- Boot animation (disabled → enabled cross-fade)
- Custom cursor

### Step 2: DB Generation (preloader)
On first run, generate `db.json` with 100 fingerprint records:
```json
{
  "id": 1,
  "color": "green",
  "variant": 2,
  "rotation": 90,
  "mirrored": true,
  "hash": "G<crc64>",
  "pieces": [{"x":0,"y":0,"uint32":...}, ...],
  "person_name": "...",
  "avatar_key": "1"
}
```
Each record: pick color, variant (1-4), rotation (0/90/180/270), mirror (bool).
Load fingerprint image, scale to 690×690, apply rotation+mirror, cut 10×10 grid.
Compute CRC64 from piece uint32s. Generate person name + avatar.
Store as db.json. On subsequent runs, load from disk.

### Step 3: Case Generation
3 hardcoded cases. Each case:
- Pick a DB record as the target fingerprint
- Choose {pieces-to-solve}: case 1 = 4-8, case 2 = 8-12, case 3 = 12-16
- Remove that many pieces from the grid
- Add decoy pieces: 5 random pieces from each OTHER fingerprint variant
- Randomly decide: show color (0) or hide color (1)
  - If hidden: draw fingerprint as grey, colored pieces in tray
  - If shown: use original color everywhere, first hash letter = G/R/Y/B
  - If hidden: first hash letter = "?"

### Step 4: Application Layout Scene Content
Draw dynamic content in the TMX object rooms:
- `list-of-cases`: 3 case buttons (hardcoded names), first selected by default
- `fingerprints-user-names`: person names or "Unknown" / "?"
- `avatar`: draw avatar image or "unknown" placeholder
- `description`: case details text
- `play-puzzle` button → open puzzle for selected case
- `exit` → back to desktop

### Step 5: Puzzle Workspace
- Draw fingerprint grid (690×690 area, 10×10 = 69×69 per cell)
- Pre-filled tiles in place, missing tiles as empty slots
- Piece tray: show draggable pieces with random rotation
- Click piece → attaches to cursor
- Mouse wheel → rotate attached piece
- Click grid slot → place piece (detach from cursor)
- Live hash display: compute from current uint32 values
- `send` button → compare hash → show success/fail tile layer

### Step 6: Save/Load State
- Persist to disk: solved cases, placed pieces, current case state
- Restore on game restart
- Each piece placement saved immediately

### Step 7: Camera + Polish
- Zoom hotkey (around center of screen)
- Reset hotkey
- Use pkg/core/Camera for WorldMatrix
- Cursor limited to `cursor-room` object bounds

## Game Logic (from README.md)

### Fingerprint Generation
1. Choose: color (G/R/Y/B), variant (1-4), rotation (0/90/180/270), mirror (bool)
2. Load `fingerprints/{color}.{variant}.png`
3. Scale to 690×690
4. Apply rotation, then mirror
5. Cut into 10×10 grid (69×69 each)
6. Each piece gets uint32 from (x,y) coordinates
7. CRC64 hash of all 100 uint32s = correct hash
8. Full hash = `{COLOR_LETTER}{CRC64}`

### Puzzle Setup
1. From 100 pieces, remove {pieces-to-solve} random ones
2. Removed pieces go to tray with random rotation
3. Add 5 decoy pieces from each other variant (3 variants × 5 = 15 decoys)
4. If color hidden: grey fingerprint on grid, colored pieces in tray
5. If color shown: original color everywhere

### Verification
1. Player places pieces and optionally selects color
2. Hash computed live from current grid state
3. Send → lookup hash in DB → found person or no match
4. Show success/fail overlay tile layer
5. Store result (fail = no person found, valid outcome)
