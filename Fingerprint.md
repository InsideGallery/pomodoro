# Fingerprint Lab — Game Design & Asset Guide

## Overview

A standalone forensic puzzle game set in the "Muldrow Police Department" universe.
The player is a fingerprint analyst who identifies suspects by assembling
fingerprint puzzles and matching them against a database.

**NOT** a pomodoro break minigame. Separate binary: `make build-fingerprint`

---

## Visual Design

All scenes render INSIDE a retro CRT monitor. The monitor frame is the constant
background. UI elements appear on the CRT "screen" area.

### Color Palette (from Карта кольорів та шрифтів.png)

| Color | Hex | Usage |
|-------|-----|-------|
| Teal tint | #d5f2f1 | Window backgrounds, UI chrome |
| White | #ffffff | Text on dark elements, buttons |
| Dark grey | #4d4b4b | Body text, labels |

### Fonts

| Font | Size | Usage |
|------|------|-------|
| Sylfaen Regular | 44pt | Fingerprint code digits (middle column) |
| Georgia Regular | 43pt | Labels, descriptions |
| Georgia Bold | 72pt | Headings, titles |

### Custom Cursor

`курсор.png` (125×124) — teal arrow, replaces system cursor in ALL game scenes.
Hide system cursor via `ebiten.SetCursorMode(CursorModeHidden)`.
Draw cursor image at mouse position as the last draw call.

---

## Asset Inventory

### All images are 8328×4320 (except noted)

These are rendered as LAYERS. Each is a separate full-frame PNG that overlays
on top of the previous. They are NOT cropped regions — they are full-size
with transparent areas where the layer below shows through.

| Key | File | Size | Description |
|-----|------|------|-------------|
| bg_off | екран (понижена яскравість).png | 20MB | CRT monitor OFF (dark screen) |
| bg_on | екран (підвищена яскраввість).png | 30MB | CRT monitor ON (bright screen glow) |
| bg_static | Фон (не анімований).png | 83MB | CRT monitor static frame (no screen content) |
| wallpaper | робочий стіл (фон).png | 11MB | Desktop wallpaper ("Muldrow Police Department") |
| frame | рама.png | 13MB | Generic window frame (used inside CRT screen) |
| workspace | Робоче поле Дактилоскопії.png | 18MB | Puzzle workspace (grey fingerprint area) |
| grid | Робоче поле Дактилоскопії (сітка 0-9).png | 18MB | Same with 10×10 grid lines visible |
| app_window | Вікно вибору відбитка.png | 16MB | 3-column app layout (empty) |
| app_full | Вікно вибору відбитка (повне).png | 16MB | 3-column app layout (full version) |

### Small UI Assets

| Key | File | Dimensions | Description |
|-----|------|-----------|-------------|
| cursor | курсор.png | 125×124 | Custom mouse cursor |
| app_icon | fingerprinting.png | 343×343 | Desktop app icon (teal circle) |
| btn_place | place button.png | 949×137 | "Place" button (normal) |
| btn_place_hover | place button - активовано.png | ~same | "Place" button (hover/active) |
| btn_code | code button.png | 554×118 | "Code" button (normal) |
| btn_code_hover | code button - активовано.png | ~same | "Code" button (hover/active) |
| btn_send | send button.png | 588×154 | "Send" button (normal) |
| btn_send_hover | send button- активовано.png | ~same | "Send" button (hover/active) |
| btn_back_hover | back - активовано.png | ~small | Back arrow (hover state only) |
| btn_exit_hover | exit - активовано.png | ~small | Exit text (hover state only) |
| stamp_success | success button.png | 653×169 | Green "SUCCESS!" stamp overlay |
| stamp_fail | fail button.png | 653×169 | Red "FAIL" stamp overlay |
| highlighter | Відбитки/highlighter.png | ~small | Tile selection highlight |

### Design Reference (NOT loaded at runtime)

| File | Description |
|------|-------------|
| приклад.png | Full example of 3-column layout with data filled in |
| Трасування колонок.png | Wireframe showing column/row layout |
| трасування колонок приклад.png | Wireframe with content example |
| робочий стіл приклад.png | Desktop example screenshot |
| Карта кольорів та шрифтів.png | Font and color specification |

### Avatars

5 suspect photos (660×660 each):
`1.jpg`, `2.jpg`, `3.jpg`, `4.jpg`, `5.jpg`
Vintage style with teal tint. Used in the right column of the app.

### Fingerprints

**4 colors × 4 variants = 16 colored + 4 grey = 20 fingerprints**

Each colored fingerprint directory contains:
- Full image: `*N centered.png` (1332×1335) — the complete fingerprint, centered
- Raw image: `*N.png` — same fingerprint, not centered
- `images/` directory: 100 pre-cut pieces (133×134 each)

Piece naming: `{Color}{Variant}_{01-100}.png`
Examples: `Blue1_01.png`, `green-1_50.png`, `red-3_99.png`

**Note**: naming is inconsistent between colors:
- blue 1: `Blue1_01.png` (capitalized)
- blue 2-4: `blue-2_01.png` (lowercase with dash)
- green: `green-1_01.png`
- red: `red-1_01.png`
- yellow: `yellow-1_01.png`

The 100 pieces form a **10×10 grid**. Piece 01 = top-left, piece 100 = bottom-right.
Layout: row-major order (01-10 = row 0, 11-20 = row 1, ..., 91-100 = row 9).

**Grey fingerprints** (Відбитки/шматочки пазлу/grey/):
4 variants: G1-G4. Only full images, no pre-cut pieces.
Grey = the fingerprint before the player chooses a color.

### Loading Animation

8 frames alternating between two states:
`loading 1.png` → `loading 1а.png` → `loading 2.png` → `loading 2а.png` → ...
Shows a hand scanning/analyzing a fingerprint.

---

## Scene Flow

```
[Preloader] → [Desktop] → [App (3-column)] → [Puzzle Workspace]
                  ↑              ↓ (back)           ↓ (back)
                  └──────────────┘                   │
                  └──────────────────────────────────┘
```

### Scene 1: Preloader

**Background**: black screen or `bg_off` (CRT monitor powered off)
**Content**: loading animation (8 frames) + progress bar
**Logic**: load ALL resources asynchronously. Track progress.
**Transition**: when 100% loaded → Desktop scene

### Scene 2: Desktop

**Boot animation** (1.5 seconds):
1. Frame 0-45: Show `bg_off` (dark/powered-off CRT monitor)
2. Frame 45-90: Cross-fade `bg_off` → `bg_on` (monitor turning on)
3. Frame 90+: Show `wallpaper` on the CRT screen area
4. Fade in app icon + cursor → interactive

**Interactive elements**:
- "Fingerprinting" app icon (top-left of CRT screen) → click opens App scene
- "Quit" label (bottom-right) → exit game
- Custom cursor

**Background layers** (draw order):
1. `bg_on` (full frame — CRT monitor with bright screen)
2. `wallpaper` (drawn ONLY in the CRT screen rect area)
3. App icon + labels on top

### Scene 3: App (3-Column Forensic Application)

**Background layers**:
1. `bg_on` (CRT monitor)
2. `app_window` or `app_full` (3-column app chrome, drawn in CRT screen area)

**Layout** (from Трасування колонок.png and приклад.png):

```
┌─────────────────────────────────────────────────────────────┐
│ [badge] ─── FINGERPRINTING ──────────────────────── [exit]  │  Title bar
├───────────────┬─────────────────┬───────────────────────────┤
│ MOTEL         │ A40458355680... │ ┌─────────┐              │
│ CAR WASH      │ B12484380012... │ │ PHOTO   │              │
│ EDEN          │ C70484385112... │ │ 660×660 │              │
│ HOUSE MCQUEEN │                 │ └─────────┘              │
│               │                 │                           │
│ Left column   │ Middle column   │ Name: ???                 │
│ (cases list)  │ (codes)         │ Known as: ???             │
│               │                 │ DOB: ???                  │
│               │                 │ Place of Birth: ???       │
│               │                 │ ...                       │
│               │                 │                           │
│               │ [place] [code]  │ Physical Description      │
│               │ [send]          │ Occupation                │
│               │                 │ Criminal Record           │
│               │                 │ Right column (suspect)    │
└───────────────┴─────────────────┴───────────────────────────┘
```

**Left column**: Case list. Each case = clickable row.
- Unsolved: teal/normal background
- Solved: green highlight
- Click → selects case, shows data in middle + right columns

**Middle column**: Fingerprint codes for selected case.
- Each code: letter prefix (A/B/C = color) + 16 digits
- Color of the code bar: matches fingerprint color (green/blue/red/yellow)
- Buttons at bottom: Place, Code, Send
- Place → opens Puzzle Workspace
- Code → shows computed code from current puzzle state
- Send → submits the fingerprint for identification

**Right column**: Suspect profile for selected case.
- Top: avatar photo (660×660) or "???" placeholder
- Below: name, alias, DOB, place of birth, citizenship, age, sex
- Physical description, occupation, criminal record
- All "???" until case is solved

### Scene 4: Puzzle Workspace

**Background layers**:
1. `bg_on` (CRT monitor)
2. `workspace` or `grid` (puzzle field with window chrome)

**Layout**: a window with 10×10 grid in the center.
- Title bar: [back] ─── title ──── [exit]
- Center: 10×10 grid (each cell ~133×134 pixels at native resolution)
- Pre-filled tiles shown in their positions
- Empty slots shown as grey/dark cells
- Available pieces shown in a tray area (bottom or side)

**Interaction**:
- Click a piece from the tray → pick it up
- Click a grid cell → place the piece
- Right-click or button → rotate piece 90°
- `highlighter.png` overlay on hovered cell

---

## Game Logic (Domain Model)

### Tile Encoding

Each tile value is `uint32` = `EncodeTile(x, y, rotation, content)`:
- byte 0: x position (0-9)
- byte 1: y position (0-9)
- byte 2: rotation (0-3 = 0°/90°/180°/270°)
- byte 3: content (random 1-255, unique per tile)

**Only when x, y match the correct position AND rotation = 0 does the
tile have its "correct" value.**

### Fingerprint Hash

Hash = SHA256 of all 100 tile uint32 values → truncated to 9-digit decimal.
UniqueID = color_byte + "-" + 9_digit_hash.

Example: `2-384729105` (green, hash 384729105)

The hash is **pre-computed** from the solved state before removing pieces.
Only placing ALL pieces correctly (right position + rotation 0) reproduces it.

### Color Selection

Grey fingerprints → player must choose a color (yellow/green/red/blue).
The chosen color becomes byte 0 of the UniqueID.
Wrong color = wrong UniqueID = no match in database.

### Person Database

Each person has: name, avatar, fingerprint UniqueID.
Lookup by UniqueID → returns person or "unknown".
Wrong color OR wrong tile arrangement → "unknown" (case failed).

### Case Flow

1. Select case from list
2. See grey fingerprint code
3. Click "Place" → open puzzle workspace
4. Place pieces on grid, rotate to correct orientation
5. Choose fingerprint color
6. Click "Code" → compute hash from current state
7. Click "Send" → submit UniqueID to database
8. Match found → SUCCESS stamp, reveal suspect profile
9. No match → FAIL stamp

---

## Implementation Plan (Step by Step)

### Step 1: Layer-Based Rendering

All 8328×4320 images are **layers** drawn on top of each other.
Downscale all to fit screen (e.g., 2560×1600) preserving aspect ratio.
Each scene draws specific layers in order.

### Step 2: Desktop Scene

- Layer 1: `bg_off` (boot) → cross-fade to `bg_on`
- Layer 2: `wallpaper` (in CRT screen area only)
- Interactive: app icon, quit, cursor

### Step 3: App Scene (3-Column)

- Layer 1: `bg_on`
- Layer 2: `app_window` (the 3-column chrome)
- Dynamic content: case list, codes, suspect profile rendered as text/entities
- Buttons: `btn_place`, `btn_code`, `btn_send` drawn at specific positions

### Step 4: Puzzle Workspace

- Layer 1: `bg_on`
- Layer 2: `grid` (workspace with 10×10 grid)
- Tiles: draw pre-cut piece images at grid positions
- Tray: show available pieces
- Interaction: click to place, right-click to rotate

### Step 5: Game Logic Integration

- Generate cases with domain model
- Pre-cut fingerprint pieces loaded from `images/` directories
- Track placed pieces, compute hash on demand
- Database lookup on submit
- Success/fail stamps

### Step 6: Polish

- Loading animation spritesheet
- Button hover states
- Sound effects
- Case persistence
