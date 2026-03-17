package fsystems

import (
	"github.com/InsideGallery/core/memory/registry"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/InsideGallery/pomodoro/pkg/core"
	"github.com/InsideGallery/pomodoro/pkg/systems"
	"github.com/InsideGallery/pomodoro/pkg/tilemap"
)

// SceneAccessor provides access to scene infrastructure for systems.
// All mutable game state lives in GameData component — accessed via Registry.
// Systems draw in world (map) coordinates. Camera handles world→screen transform.
type SceneAccessor interface {
	// Infrastructure
	GetRegistry() *registry.Registry[string, uint64, any]
	GetCamera() *core.Camera
	GetInputSystem() *systems.InputSystem
	GetTileMap() *tilemap.Map
	GetScreenSize() (int, int)

	// Infrastructure mutation (loading phase)
	SetTileMap(*tilemap.Map)
	SetupWorld() // creates World image + configures Camera after TMX loads
	SetCursorPos(x, y int)

	// App lifecycle
	RequestQuit()

	// Camera
	GetBaseZoom() float64
	ResetCameraZoom()

	// Screen↔World coordinate conversion (uses Camera)
	ScreenToWorld(screenX, screenY float64) (float64, float64)

	// Persistence
	LoadGameState()
	SaveGameState()

	// Zone registration (called by StateSystem on transitions)
	RegisterEnabledZones()
	RegisterAppLayoutZones()
	RegisterPuzzleZones()

	// Puzzle image loading
	EnsureCurrentPuzzleImages()
	InitTrayPositions()

	// Image access (lazy-loaded caches)
	GetTargetPieceImage(recordID, pieceIdx int) *ebiten.Image
	GetGreyPieceImage(recordID, pieceIdx int) *ebiten.Image
	GetDecoyPieceImage(color string, variant, rotation int, mirrored bool, pieceIdx int) *ebiten.Image
	GetAvatarImage(filename string) *ebiten.Image
}
