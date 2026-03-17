# Fingerprint Lab — ECS Refactor + Cross-Platform Plan

## Why ECS

The game is ~2000 lines of procedural code in one file. All drawing, input,
state management, and game logic are interleaved. Adding features means touching
everything. ECS separates data (components) from behavior (systems):

- **Components** = pure data structs, no logic, no callbacks
- **Systems** = stateless functions that iterate component groups
- **Entities** = just an ID linking components together
- **Composition over inheritance** — entity behavior defined by which components it has

This makes every feature isolated, testable, and reusable across platforms.

---

## Existing ECS Infrastructure

Already in the codebase (used by pomodoro timer/settings scenes):

```
pkg/core/system.go      — System interface (Update + Draw)
pkg/core/systems.go     — Systems ordered collection (Add/Remove/Get/Clean)
pkg/core/camera.go      — Camera with WorldMatrix, zoom, pan, rotation
pkg/scene/base.go       — BaseScene: Systems + Registry + RTree + Camera + Bus
pkg/systems/input.go    — InputSystem with RTree zones + CursorOverride
InsideGallery/core/ecs/ — Entity/Component/System interfaces
InsideGallery/core/memory/registry/ — Registry[group, id, value]
```

The Registry is an Indexed ECS: groups = archetypes, entity ID = index,
value = component data. Systems iterate groups to process entities.

---

## Part A: Component Design (Step 1)

Create `services/fingerprint/internal/components/`.

Components are pure data structs. No methods, no callbacks, no logic.
Following the article: "Components have no logic, systems have no state,
and entities have nothing."

```
components/
  transform.go      — position, size in map coordinates
  sprite.go         — ebiten.Image reference, alpha, visible flag
  renderable.go     — rotation angle (radians), scale, z-order
  button.go         — label text, bg/text colors, face reference
  clickable.go      — spatial bounds for RTree (shapes.Spatial)
  scrollable.go     — scroll offset, row height, item count, max visible
  text_block.go     — text string, max width/height, scroll offset
  avatar.go         — filename, cached image pointer
  layer.go          — TMX layer name, alpha (for image/tile layers)
  puzzle_grid.go    — pointer to PuzzleConfig, cell size, origin
  puzzle_piece.go   — pointer to TrayPiece, in-tray flag
  draggable.go      — being-dragged flag, piece index
  cursor.go         — x/y, prev raw x/y, room bounds, inited flag
  state.go          — current GameState enum value
  progress_bar.go   — progress float64, status text
```

Each component is a plain struct with exported fields.
No pointers to other components. No function fields.

---

## Part B: Entity Groups / Archetypes (Step 2)

Registry groups act as archetypes. An entity belongs to a group
based on its role. Systems query by group name.

```
Group               Entities    Components
─────────────────────────────────────────────────────
"game_state"        1           State
"cursor"            1           Cursor + Transform
"progress"          1           ProgressBar
"bg_layer"          1-4         Layer + Renderable (image layers per state)
"tile_layer"        1-4         Layer + Renderable (tile layers per state)
"button"            5-8         Button + Transform + Clickable
"case_list"         1           Scrollable + Transform
"puzzle_list"       1           Scrollable + Transform
"avatar"            1           Avatar + Transform
"description"       1           TextBlock + Transform
"grid"              1           PuzzleGrid + Transform
"grid_cell"         100         Sprite + Transform + Renderable
"tray_piece"        3N-36       PuzzlePiece + Sprite + Transform + Draggable
"held_piece"        0-1         Sprite + Transform + Renderable
"hash_display"      1           TextBlock + Transform
"result_overlay"    0-1         Layer + Renderable
```

---

## Part C: Systems (Steps 3-9)

Systems registered in execution order. Each system iterates one or more
entity groups and processes their components. Systems are stateless —
all state lives in components.

### C1: StateSystem (step 3)
- Reads "game_state" entity
- Manages transitions: Loading → Disabled → Enabled → AppLayout → AppNet
- On state enter: creates entities for that state via assemblage functions
- On state exit: removes entities from previous state groups
- Boot animation: updates Layer alpha on "bg_layer" entities over 90 frames
- Loading: runs deferred goroutines, updates "progress" entity

### C2: CursorSystem (step 4)
- Reads "cursor" entity (Cursor component)
- Desktop: reads `ebiten.CursorPosition()`, applies delta-based clamping
- Mobile: reads `ebiten.TouchPosition()` for single touch
- Writes CursorOverride to InputSystem
- Platform-specific via build tags:
  - `cursor_desktop.go` (`//go:build !android`)
  - `cursor_mobile.go` (`//go:build android`)

### C3: ScrollSystem (step 5)
- Iterates "case_list", "puzzle_list", "description" entities
- Reads Scrollable/TextBlock components
- Desktop: mouse wheel when cursor over the entity's Transform bounds
- Mobile: touch swipe gesture
- Updates scroll offset, clamps to valid range

### C4: InputSystem (step 6 — extend existing)
- Already exists in `pkg/systems/input.go`
- Extend: on entity creation, Clickable components auto-register in RTree
- On entity removal, auto-unregister from RTree
- Reads CursorOverride from CursorSystem
- Fires click callbacks from Button components

### C5: DragDropSystem (step 7)
- Iterates "tray_piece" and "grid_cell" entities
- Reads Cursor position from "cursor" entity
- On press over tray piece → creates "held_piece" entity, removes from tray
- On press over placed grid cell → unplace, create "held_piece"
- While dragging → updates "held_piece" Transform to cursor position
- Desktop: mouse wheel rotates held piece (updates Renderable.Rotation)
- Mobile: two-finger twist rotates
- On release over empty missing grid cell → place piece, save state
- On release elsewhere → return to tray at cursor position

### C6: CameraSystem (step 8)
- Reads Camera from BaseScene
- Sets Camera.ViewPort from Layout dimensions
- Desktop: Z/X/C keys → zoom in/out/reset
- Mobile: pinch gesture → zoom
- Only active when game state = AppNet
- RenderSystem reads Camera.WorldMatrix() for world-space entities

### C7: RenderSystem (step 9)
Implements both `Draw()` and `ScreenDraw()`:

**Draw() — world space** (Camera transform applied):
- "grid" entities: puzzle grid lines + pre-filled pieces
- "grid_cell" entities: placed pieces or empty slot highlights
- "tray_piece" entities: pieces at TrayX/TrayY positions
- "held_piece" entity: piece following cursor

**ScreenDraw() — screen space** (no camera, UI overlay):
- "bg_layer" entities: background images with alpha
- "tile_layer" entities: TMX tile layers
- "button" entities: rounded rects + centered text
- "case_list" / "puzzle_list": scrollable button lists
- "avatar" entity: character image
- "description" entity: word-wrapped scrollable text
- "hash_display" entity: live hash text
- "result_overlay" entity: success/fail tile layer
- "cursor" entity: cursor image (always on top)
- "progress" entity: loading bar (during StateLoading)

---

## Part D: Assemblage Functions (Step 10)

Create `services/fingerprint/internal/entities/`.

Factory functions that populate Registry for each state.
Called by StateSystem on transitions.

```go
// entities/loading.go
func CreateLoadingEntities(reg, ...) {
    // "progress" entity: ProgressBar{Progress: 0, Status: ""}
    // "cursor" entity: Cursor{} + Transform{}
}

// entities/enabled.go
func CreateEnabledEntities(reg, tmap, scaleX, scaleY) {
    // "bg_layer": enabled background
    // "tile_layer": enabled tiles
    // "button": run-fingerprint, quit-os
    // "cursor": virtual cursor
}

// entities/app_layout.go
func CreateAppLayoutEntities(reg, cases, selectedCase, selectedPuzzle, tmap, scale) {
    // "bg_layer": enabled + app-layout backgrounds
    // "tile_layer": app-layout tiles
    // "case_list": Scrollable with 50 cases
    // "puzzle_list": Scrollable with 20 puzzles
    // "avatar": Avatar with unknown or character filename
    // "description": TextBlock with story narrative
    // "button": play-puzzle, regenerate, exit
}

// entities/puzzle_net.go
func CreatePuzzleEntities(reg, puzzle, targetImgs, greyImgs, tmap, scale) {
    // "bg_layer": enabled + net-layout backgrounds
    // "tile_layer": net-layout tiles
    // "grid": PuzzleGrid
    // "grid_cell" × 100: pre-filled pieces + empty slots
    // "tray_piece" × 3N: correct + decoy pieces
    // "hash_display": live hash
    // "button": back, exit, send
}
```

StateSystem calls `CleanGroups()` then `CreateXxxEntities()` on each transition.

---

## Part E: Refactor game.go (Step 11)

game.go shrinks from ~2000 lines to ~300 lines:

```go
type GameScene struct {
    *scene.BaseScene
    input    *systems.InputSystem
    tmap     *tilemap.Map
    db       *domain.FingerprintDB
    cases    []*domain.CaseConfig
    platform platform.Platform
    // ... minimal state: selectedCase, selectedPuzzle, puzzleSeed
}

func (s *GameScene) Init(ctx context.Context, plat platform.Platform) {
    s.BaseScene = scene.NewBaseScene(ctx, nil)
    s.platform = plat

    s.Systems.Add("state",    fsystems.NewStateSystem(s))
    s.Systems.Add("cursor",   fsystems.NewCursorSystem(s))
    s.Systems.Add("input",    s.input)
    s.Systems.Add("scroll",   fsystems.NewScrollSystem(s))
    s.Systems.Add("dragdrop", fsystems.NewDragDropSystem(s))
    s.Systems.Add("camera",   fsystems.NewCameraSystem(s))
    s.Systems.Add("render",   fsystems.NewRenderSystem(s))
}

func (s *GameScene) Update() error { return s.BaseScene.Update() }
func (s *GameScene) Draw(screen *ebiten.Image) { s.BaseScene.Draw(screen) }
```

ALL drawing code moves to RenderSystem.
ALL input handling moves to InputSystem/DragDropSystem/ScrollSystem.
ALL state transitions move to StateSystem.
game.go becomes a thin shell providing accessor methods for systems.

---

## Part F: Platform Abstraction (Steps 12-15)

### F1: Platform Interfaces (step 12)

Create `pkg/platform/`:

```go
// platform.go
type Platform interface {
    Storage() Storage
    Assets() Assets
    CanExit() bool    // true on desktop, false on mobile
}

type Storage interface {
    ReadFile(name string) ([]byte, error)
    WriteFile(name string, data []byte) error
    Remove(name string) error
    DataDir() string
}

type Assets interface {
    Open(path string) (io.ReadCloser, error)
    ReadFile(path string) ([]byte, error)
}
```

### F2: Desktop Implementation (step 13)

```
pkg/platform/
  desktop.go         (//go:build !android)
  desktop_storage.go (//go:build !android) — wraps os.* with DataDir = ~/.config/pomodoro/fingerprint/
  desktop_assets.go  (//go:build !android) — wraps os.Open with path resolution
```

### F3: Embed Assets (step 14)

```go
// assets/embed.go
//go:embed external/fingerprint/stories.json
//go:embed external/fingerprint/fingerprints/*.png
//go:embed external/fingerprint/avatars/*.jpg
var FingerprintAssets embed.FS
```

Background PNGs (119MB) NOT embedded on desktop — loaded from filesystem.
For mobile: create scaled 2000×1088 versions, embed those.

```
assets/external/fingerprint/background_mobile/  — scaled backgrounds
assets/embed_mobile.go   (//go:build android)   — embeds mobile backgrounds
assets/embed_desktop.go  (//go:build !android)   — embeds only small assets
```

### F4: Refactor Domain for Storage (step 15)

All domain I/O functions accept `platform.Storage`:

```go
func LoadDB(store Storage) (*FingerprintDB, error)
func (db *FingerprintDB) Save(store Storage) error
func LoadPuzzles(store Storage, db *FingerprintDB) ([]*CaseConfig, uint64, error)
func SavePuzzles(cases []*CaseConfig, seed uint64, store Storage) error
func SaveGame(save *GameSave, store Storage) error
func LoadGame(store Storage) (*GameSave, error)
func LoadStories(assets Assets) error
```

Image loading accepts `platform.Assets`:
```go
func LoadFingerprintImages(assets Assets, rec *FingerprintRecord) (*FingerprintImages, error)
func loadStdImage(assets Assets, path string) (image.Image, error)
```

Remove: `FindTMXPath()`, `FindFingerprintAssetsDir()`, `os.Exit()`, `os.UserHomeDir()`.
Replace `os.Exit(0)` with `s.Bus.Emit("quit")` — desktop app.go handles exit.

---

## Part G: Touch Input (Steps 16-18)

### G1: Touch Adapter (step 16)

Create `pkg/systems/touch.go`:

```go
type TouchState struct {
    Touches     []TouchPoint
    PrevTouches []TouchPoint
}

type TouchPoint struct {
    ID   ebiten.TouchID
    X, Y int
}

func DetectGesture(curr, prev []TouchPoint) (GestureType, GestureData)
```

Gesture types: Tap, Drag, Pinch (zoom), Twist (rotate), Swipe (scroll).

### G2: Integrate into CursorSystem + DragDropSystem (step 17)

CursorSystem:
- `//go:build !android`: mouse position + delta clamping (current behavior)
- `//go:build android`: single touch → cursor position, touch start/end → press/release

DragDropSystem:
- Mouse wheel rotation → also accept two-finger twist
- Both via unified input reading from CursorSystem

### G3: Platform-Specific Input Config (step 18)

```
fsystems/input_desktop.go  (//go:build !android)
  — Escape key, Z/X/C zoom, mouse wheel
fsystems/input_mobile.go   (//go:build android)
  — Back button, pinch zoom, swipe scroll, minimum 48dp touch targets
```

---

## Part H: Mobile Build (Steps 19-22)

### H1: Android Platform Implementation (step 19)

```
pkg/platform/
  android.go         (//go:build android)
  android_storage.go (//go:build android) — app-internal storage
  android_assets.go  (//go:build android) — reads from embed.FS
```

### H2: Scale Background Images (step 20)

Scale 4 backgrounds from 4000×2176 to 2000×1088:
```bash
for f in background/*.png; do
  convert "$f" -resize 2000x1088 "background_mobile/$(basename $f)"
done
```

### H3: Mobile Entry Point (step 21)

```
services/fingerprint/mobile/
  mobile.go  — init() { mobile.SetGame(newGame()) }
  game.go    — same scene setup, AndroidPlatform, landscape-only
```

Build:
```bash
ebitenmobile bind -target android \
  -javapkg com.insidegallery.fingerprint \
  -o fingerprint.aar \
  ./services/fingerprint/mobile/
```

### H4: Android Project (step 22)

```
services/fingerprint/mobile/android/
  app/build.gradle.kts
  app/src/main/AndroidManifest.xml   — landscape, minSdk 21, targetSdk 34
  app/src/main/java/.../MainActivity.java
  app/src/main/res/
```

---

## Part I: UI Scaling (Step 23)

Create `services/fingerprint/internal/fsystems/scale.go`:

- Detect screen size and DPI in Layout()
- Compute: button height, font size, piece size, touch target minimums
- Store as singleton component on "layout" entity
- All systems read from it instead of hardcoded pixel values
- Desktop: current sizes work at 1920×1080
- Mobile: enforce minimum 48dp touch targets, 14sp font size

---

## Part J: Build & Test (Steps 24-25)

### J1: Makefile (step 24)

```makefile
build-desktop:
	go build -o bin/fingerprint ./services/fingerprint/cmd/fingerprint/

build-android:
	ebitenmobile bind -target android -javapkg com.insidegallery.fingerprint \
		-o services/fingerprint/mobile/android/app/libs/fingerprint.aar \
		./services/fingerprint/mobile/

test:
	go test ./pkg/... ./services/fingerprint/...

run:
	go run ./services/fingerprint/cmd/fingerprint/
```

### J2: Tests (step 25)

- Component creation unit tests
- Assemblage functions produce correct Registry state
- StateSystem transition tests (enter/exit create/clean entities)
- DragDropSystem place/pickup logic tests
- ScrollSystem clamp tests
- CameraSystem zoom math tests
- Platform Storage/Assets mock tests
- Existing domain/ tests unchanged
- Integration: build for android target compiles without errors

---

## Execution Order

| # | Step | Depends | Size | Description |
|---|------|---------|------|-------------|
| 1 | A | — | M | Define all component types |
| 2 | B | 1 | S | Define entity groups/archetypes |
| 3 | C1 | 1,2 | L | StateSystem (state machine, entity lifecycle) |
| 4 | C2 | 1 | M | CursorSystem (delta cursor, platform split) |
| 5 | C3 | 1 | M | ScrollSystem |
| 6 | C4 | 1 | S | InputSystem extension (auto RTree register) |
| 7 | C5 | 1,4 | L | DragDropSystem |
| 8 | C6 | 1 | M | CameraSystem (zoom with WorldMatrix) |
| 9 | C7 | 1-8 | XL | RenderSystem (ALL drawing migrated here) |
| 10 | D | 1,2,3 | M | Assemblage functions per state |
| 11 | E | 3-10 | L | Refactor game.go to wire systems |
| 12 | F1 | — | S | Platform interfaces |
| 13 | F2 | 12 | S | Desktop platform implementation |
| 14 | F3 | 12 | M | Embed assets + scale backgrounds |
| 15 | F4 | 12,13 | M | Domain/images accept platform interfaces |
| 16 | G1 | — | M | Touch adapter |
| 17 | G2 | 4,7,16 | M | Touch integrated into Cursor + DragDrop |
| 18 | G3 | 16,17 | S | Platform-specific input config |
| 19 | H1 | 12 | S | Android platform implementation |
| 20 | H2 | — | S | Scale background images |
| 21 | H3 | 11,15,19 | M | Mobile entry point |
| 22 | H4 | 21 | M | Android project structure |
| 23 | I | 9 | M | UI scaling system |
| 24 | J1 | 22 | S | Makefile targets |
| 25 | J2 | 11 | M | Tests |

S = Small (< 1h), M = Medium (1-3h), L = Large (3-6h), XL = Extra Large (6-10h)

**Critical path**: 1 → 2 → 3 → 9 → 10 → 11 → 15 → 21 → 22

**Phase A-E (ECS refactor)**: Steps 1-11. Improves desktop immediately. No mobile dependency.
**Phase F (Platform)**: Steps 12-15. Enables mobile without breaking desktop.
**Phase G (Touch)**: Steps 16-18. Mobile input.
**Phase H (Android)**: Steps 19-22. Build and ship.
**Phase I-J (Polish)**: Steps 23-25. Scaling and automation.

Desktop never breaks — each step compiles and runs on PC before moving to next.
