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
// This interface is for infrastructure only: TileMap, Camera, InputSystem,
// image caches, scaling, and persistence.
type SceneAccessor interface {
	// Infrastructure
	GetRegistry() *registry.Registry[string, uint64, any]
	GetCamera() *core.Camera
	GetInputSystem() *systems.InputSystem
	GetTileMap() *tilemap.Map
	GetScreenSize() (int, int)

	// Scaling (read-only, computed by Layout)
	GetScaleX() float64
	GetScaleY() float64
	GetOffsetX() float64

	// Infrastructure mutation (loading phase only)
	SetTileMap(*tilemap.Map)
	SetScale(scaleX, scaleY, offsetX float64)
	SetCursorPos(x, y int)

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

	// Image access (lazy-loaded caches, not game state)
	GetTargetPieceImage(recordID, pieceIdx int) *ebiten.Image
	GetGreyPieceImage(recordID, pieceIdx int) *ebiten.Image
	GetDecoyPieceImage(color string, variant, rotation int, mirrored bool, pieceIdx int) *ebiten.Image
	GetAvatarImage(filename string) *ebiten.Image
}
