# Status Report

## Test Summary

| Package | Tests | Status |
|---------|-------|--------|
| services/pomodoro/internal/timer | 25 | PASS |
| pkg/config | 11 | PASS |
| pkg/core | 5 | PASS |
| pkg/event | 9 | PASS |
| pkg/scene | 16 | PASS |
| pkg/systems | 6 | PASS |
| pkg/plugins/fingerprint | 8 | PASS |
| pkg/plugins/fingerprint/domain | 16 | PASS |
| pkg/plugins/lockscreen | 10 | PASS |
| pkg/plugins/metrics | 9 | PASS |
| pkg/plugins/minigame | 11 | PASS |
| **Total** | **126** | **ALL PASS** |

Coverage: 84.1% | Lint: 0 issues

## What's Irrelevant / Dead Code

1. **pkg/ecs/components.go** — has entity types (ButtonEntity, TimerTextEntity, etc.) used only
   by timer scene. Settings scene defines its own entity types in systems/entities.go.
   Not truly "shared" as intended. Could be moved to timer/systems/ or deleted.

2. **pkg/plugins/fingerprint/generate.go** — procedural fingerprint IMAGE generator (whorl/loop/arch).
   This was built before understanding the real game design. The actual game uses pre-made
   fingerprint IMAGES from assets/, not procedural generation. The domain model's tile encoding
   is what matters for gameplay. generate.go + generate_test.go could be removed.

3. **pkg/plugins/fingerprint/puzzle.go** — the "matching" puzzle scene where you pick from 6
   candidates. This was the WRONG interpretation of the game. The real game is a tile-placement
   puzzle on a 10x10 grid. This file should be rewritten or replaced.

4. **plugins/ directory** (.so entry points) — has example/, minigame/, lockscreen/, metrics/
   wrappers. These duplicate the logic in pkg/plugins/. With builtin/ compiled-in, .so plugins
   are rarely needed. Consider removing or documenting as optional.

## Fingerprint Asset Analysis

### Background Layers (draw order)

1. `Фон (не анімований).png` (86MB, 8328x4320) — CRT monitor + desk, fullscreen
2. Inside CRT screen area (~22-78% of width, ~8-86% of height):
   - `робочий стіл (фон).png` — Desktop wallpaper ("Muldrow Police Department")
   - `fingerprinting.png` — App icon (top-left of desktop area)

### App Window Chrome

3. `рама.png` — Generic window frame (used for sub-windows)
4. `Вікно вибору відбитка.png` — 3-column app window (main layout)
5. Title bar: "FINGERPRINTING" text + badge icon (top-left) + "exit" button (top-right)

### 3-Column Layout (from Трасування колонок.png)

**Left column (cases):**
- 4 case slots (MOTEL, CAR WASH, EDEN, HOUSE MCQUEEN)
- Each is a clickable row
- Below: large area for selected case details

**Middle column (codes):**
- 3-4 code slots showing fingerprint codes
- Format: letter + 16 digits (e.g., A404583556801156)
- Color-coded: different letter prefixes = different fingerprint colors
- Below: workspace or additional info

**Right column (suspect):**
- Top: suspect photo (avatars 1-5.jpg, or "???" placeholder)
- Below photo: suspect name or "???"
- Below name: detailed profile (DOB, height, hair, eyes, occupation, criminal record)

### Buttons (position from приклад.png)

- `place button` — bottom of middle column (places a piece)
- `code button` — shows computed fingerprint code
- `send button` — submits identification
- `back` — top-left (returns to previous screen)
- `exit` — top-right title bar
- `success button` / `fail button` — overlay stamps for result

### Puzzle Workspace

- `Робоче поле Дактилоскопії.png` — grey field where fingerprint goes
- `Робоче поле Дактилоскопії (сітка 0-9).png` — same with visible 10x10 grid
- Grid is ~10x10 tiles = 100 pieces total per fingerprint
- `highlighter.png` — semi-transparent overlay for selected tile position

### Fingerprint Assets (16 colored + 4 grey)

4 colors × 4 variants:
- blue 1-4, green 1-4, red 1-4, yellow 1-4
- Each has: full image (centered.png), raw image (.png), 100 pre-cut pieces (images/)
- grey: 4 variants (G1-G4), no pre-cut pieces (grey = player hasn't chosen color yet)

### Loading Animation

8 frames: loading 1.png through loading 4а.png
- Hand scanning/analyzing a fingerprint
- Alternates between states (1/1а, 2/2а, etc.)

### Custom Cursor

- `курсор.png` — teal arrow cursor, replaces system cursor in-game

### Design Spec (from Карта кольорів та шрифтів.png)

- Fonts: Serif Sylfaen Regular 44pt (code digits), Serif Georgia Regular/Bold 72pt + 43pt
- Colors: #d5f2f1 (teal tint), #ffffff (white), #4d4b4b (dark grey)

## Architecture Issues

1. **Shared Resources pattern** — currently uses SetResources() injection which is fragile.
   Better: pass Resources as parameter to scene constructors, or use a global resource
   registry accessible by all scenes.

2. **Timer scene still uses widget self-detection** for ring drag instead of RTree.
   The angular math is specific to the ring shape. Could be modeled as a DragZone
   but the current approach works.

3. **Settings scene entity types** defined in systems/entities.go instead of pkg/ecs/.
   Not a problem functionally but inconsistent with the architecture description.

## What Needs Building (Fingerprint Game)

1. **App scene** (3-column layout) — the main game UI, not yet implemented
2. **Tile placement system** — drag pieces from tray to grid, rotate, snap
3. **Color selection** — choose fingerprint color (grey → colored)
4. **Code display** — compute and show the 9-digit hash from current tile state
5. **Submit/verify flow** — send code, DB lookup, success/fail stamp
6. **Integration with real assets** — use actual fingerprint images + pre-cut pieces
7. **Custom cursor** rendering in all fingerprint scenes (partially done in desktop)

## Testing Strategy for Ebiten Applications

### Unit Tests (pure logic)
- Domain models, state machines, generators — already done extensively
- Config persistence round-trips — done
- Event bus pub/sub — done

### Integration Tests
- Scene lifecycle (Init/Load/Unload) — done via lifecycle_test.go
- Scene switching with callbacks — done via callback_test.go
- Registry operations within scenes — possible to test

### Visual / E2E Tests (research needed)
- **Screenshot testing**: render a frame to *ebiten.Image, compare against golden image
- **Ebiten testing mode**: `ebiten.RunGameWithOptions` with `InitUnfocused: true` for headless
- **Image comparison**: pixel-diff libraries (go-test-image, pixelmatch-go)
- **Headless rendering**: use offscreen image as render target, no window needed
- **GitHub Actions**: use Xvfb for Linux CI, or virtual framebuffer
