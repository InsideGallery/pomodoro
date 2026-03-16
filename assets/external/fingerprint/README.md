# Fingerprint Lab — Game Behavior Specification

## Game Concept

Forensic fingerprint puzzle game set in the fictional city of Muldrow.
Player works as a forensic analyst, matching fingerprint pieces to identify
suspects across 50 crime scene cases.

## UI Flow

All UI defined in `fingerprint.tmx` (4000×2176). The game is a single scene
with states controlling layer visibility.

### 1. Boot Sequence (StateLoading → StateDisabled)
- Show "Loading..." with progress bar while assets load in background
- After loading: cross-fade from "disabled" (PC off) to "enabled" (desktop)
- Animation: 90 frames (~1.5 seconds)

### 2. Desktop (StateEnabled)
- Show enabled Image Layer + Tile Layer (fingerprint app icon visible)
- Custom cursor clamped to `cursor-room` (delta-based, no stickiness)
- `button-run-fingerprint` → open fingerprint application
- `button-quit-os` → quit game (drawn programmatically, red, 200×50)

### 3. Application Layout (StateApplicationLayout)
Three-column layout:
- **Left**: Scrollable list of 50 cases (city locations from stories.json)
  Each button shows: "MOTEL (3/20)" — name + solved count
- **Middle**: Scrollable list of 20 fingerprint puzzles for selected case
  Each button shows: "1. Rob Malfoy" (solved) or "1. Unknown" (unsolved)
- **Right**: Avatar (unkown.jpg until solved, then character photo) +
  Description (word-wrapped, scrollable narrative from stories.json)

Buttons:
- `play-puzzle` → open selected puzzle (teal, programmatic)
- `regenerate-puzzles` → regenerate all 1000 puzzles, keep 256 fingerprints (orange)
- `exit` → back to desktop

Mouse wheel scrolls whichever area the cursor is over (cases, names, or description).

### 4. Puzzle Workspace (StateApplicationNet)
- **Puzzle grid**: 10×10 forced-square area with fingerprint image
  Pre-filled pieces shown, missing pieces as orange-highlighted empty slots
- **Piece tray**: Two rectangular rooms with draggable pieces (free-form positions)
  Pieces same size as grid cells
- **Hash display**: Live CRC64 hash updates as pieces are placed
- **Send button**: Submit current hash for verification (tile layer design)

## Fingerprint Database (256 records)

Exhaustive enumeration of all possible fingerprints:
```
4 colors (green, red, yellow, blue)
× 4 variants (fingerprint images 1-4)
× 8 rotations (0°, 45°, 90°, 135°, 180°, 225°, 270°, 315°)
× 2 mirror states (normal, mirrored)
= 256 unique fingerprints
```

Stored in `db.json`. Only regenerated if file is deleted.

Each record contains:
- Color, variant, rotation angle, mirror flag
- 100 piece records (10×10 grid), each with uint32 value encoding (x, y, rotation, content)
- CRC64 hash of all piece values
- Person name and avatar filename

### Image Processing Pipeline
```
fingerprints/{color}.{variant}.png
  → crop centered 480×480
  → upscale to 690×690 (nearest-neighbor)
  → rotate by record's angle
  → mirror if flagged
  → crop center 690×690 (for 45° angles: uses 980×980 intermediate)
  → cut into 10×10 grid of 69×69 pixel pieces
```

### Hash Computation
Each piece encoded as uint32: `x | (y<<8) | (rotation<<16) | (content<<24)`
CRC64-ECMA computed over all 100 uint32 values in grid order.
Full hash string: `{COLOR_LETTER}{CRC64_NUMBER}` (e.g. "G14976251236816614454")

## Puzzle System (1000 puzzles)

50 cases × 20 puzzles each. Cases loaded from `stories.json`.

### Difficulty
- Cases 0-19 (EASY): 3 missing pieces, random color visibility
- Cases 20-34 (MEDIUM): 6 missing pieces, random color visibility
- Cases 35-49 (HARD): 12 missing pieces, always grey/hidden color

### Piece Selection
12 corner pieces excluded (L-shape at each corner of 10×10 grid):
indices {0, 1, 10, 8, 9, 19, 80, 90, 91, 89, 98, 99}
Remaining 88 valid positions used for piece removal.

### Decoy Pieces
For N missing pieces, the tray always contains 3×N total:
- N correct pieces from the target fingerprint
- N fake pieces from random fingerprint A (different variant than target)
- N fake pieces from random fingerprint B (different variant than target)

Fake pieces loaded with TARGET's rotation+mirror so they're visually indistinguishable by angle.
Color of fake groups is random — can be same as target or different.

### Grey (Hidden Color) Mode
- Puzzle grid shows grey version of fingerprint
- All tray pieces (correct + fake) show in their actual colors
- Hash displays "?" prefix instead of color letter
- On submit: system tries all 4 color letters. If match found, color is revealed.

## Drag-and-Drop

- Mouse press on piece in tray → pick up (attaches to cursor)
- Mouse press on placed piece in grid → unplace, pick up
- Mouse wheel → rotate held piece (up=clockwise, down=counter-clockwise, 8 steps of 45°)
- Mouse release on empty missing grid cell → place piece there
- Mouse release elsewhere → drop at cursor position within drag-and-drop-zone
- Pieces have free-form positions in tray (not grid-aligned)

## Submit & Verification

- Click send button → compute current hash from grid state
- If color hidden: try G, R, Y, B prefixes against DB
- If match found → mark puzzle SOLVED, show success overlay, reveal avatar + name
- If no match → mark puzzle FAILED (valid outcome), show fail overlay
- Solved/failed state persisted to save.json

## Persistence

Three files in `~/.config/pomodoro/fingerprint/`:
- `db.json` — 256 fingerprint records (static, regenerate = delete file)
- `puzzles.json` — 50×20 puzzle configs (regenerate via button)
- `save.json` — per-puzzle: solved/failed flags, placed piece positions, tray positions

## Characters

4 suspects cycle across 256 fingerprint records by ID:
- Rob Malfoy — `m.{Rob Malfoy}.jpg`
- Steve Gilber — `m.{Steve Gilber}.jpg`
- Elizabet Queen — `w.{Elizabet Queen}.jpg`
- May Forty — `w.{May Forty}.jpg`
- Unknown (unsolved) — `unkown.jpg`

## Stories (stories.json)

50 locations in city of Muldrow. Each case has:
- Location name (MOTEL, CAR WASH, EDEN, DOCKS, CASINO, etc.)
- Intro text describing the crime
- 20 evidence locations (one per puzzle: "on the door handle", "on the bedside table", etc.)
- Solved templates with {name} placeholder for identified suspect

Description text is word-wrapped to fit the TMX description box and scrollable.
