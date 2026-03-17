package fsystems

import (
	"context"
	"log/slog"
	"os"

	"github.com/InsideGallery/core/memory/registry"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/InsideGallery/pomodoro/pkg/plugins/fingerprint/domain"
	"github.com/InsideGallery/pomodoro/pkg/tilemap"
	c "github.com/InsideGallery/pomodoro/services/fingerprint/internal/components"
	"github.com/InsideGallery/pomodoro/services/fingerprint/internal/entities"
)

// StateSystem manages the game state machine and entity lifecycle.
// StateSystem manages the game state machine. Fully stateless —
// loading state (loadDone, loadStep) lives in GameData component.
type StateSystem struct {
	scene SceneAccessor
}

func NewStateSystem(scene SceneAccessor) *StateSystem {
	return &StateSystem{scene: scene}
}

func (s *StateSystem) Update(_ context.Context) error {
	reg := s.scene.GetRegistry()
	state := s.getOrCreateState(reg)

	if state == nil {
		return nil
	}

	switch state.Current {
	case c.StateLoading:
		s.updateLoading(reg, state)

	case c.StateDisabled:
		state.BootTick++
		if state.BootTick > 90 {
			state.Current = c.StateEnabled
			s.enterEnabled(reg)
		}

	case c.StateEnabled:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
			os.Exit(0)
		}

	case c.StateApplicationLayout:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			state.Current = c.StateEnabled
			s.enterEnabled(reg)
		}

	case c.StateApplicationNet:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			state.Current = c.StateApplicationLayout
			s.enterAppLayout(reg)
		}

		// Result overlay timer
		gd := s.getGameData(reg)
		if gd != nil && gd.ResultTick > 0 {
			gd.ResultTick--
			if gd.ResultTick == 0 {
				gd.ShowResult = 0
			}
		}
	}

	return nil
}

func (s *StateSystem) Draw(_ context.Context, _ *ebiten.Image) {}

func (s *StateSystem) getOrCreateState(reg RegType) *c.State {
	val, err := reg.Get(c.GroupGameState, 0)
	if err != nil {
		entity := &c.Entity{State: &c.State{Current: c.StateLoading}}

		if addErr := reg.Add(c.GroupGameState, 0, entity); addErr != nil {
			slog.Error("add state entity", "error", addErr)

			return nil
		}

		// Also create cursor and progress entities
		entities.CreateCursorEntity(reg, 0, 0, 1920, 1080)
		entities.CreateProgressEntity(reg)

		return entity.State
	}

	if entity, ok := val.(*c.Entity); ok && entity.State != nil {
		return entity.State
	}

	return nil
}

func (s *StateSystem) updateLoading(reg RegType, state *c.State) {
	gd := s.getGameData(reg)
	if gd == nil {
		return
	}

	if gd.LoadDone != nil {
		select {
		case <-gd.LoadDone:
			gd.LoadDone = nil
			gd.LoadStep++
		default:
			if gd.LoadProgress < 0.95 {
				gd.LoadProgress += 0.003
			}

			return
		}
	}

	switch gd.LoadStep {
	case 0:
		gd.LoadStatus = "Loading scene assets..."
		gd.LoadProgress = 0.1
		gd.LoadDone = make(chan struct{})

		go func() {
			defer close(gd.LoadDone)

			if gd.AssetsDir != "" {
				if err := domain.LoadStories(gd.AssetsDir); err != nil {
					slog.Warn("load stories", "error", err)
				}
			}

			tmxPath := findTMXPath()
			if tmxPath == "" {
				slog.Error("fingerprint.tmx not found")

				return
			}

			m, err := tilemap.Load(tmxPath)
			if err != nil {
				slog.Error("load tmx", "error", err)

				return
			}

			s.scene.SetTileMap(m)
			slog.Info("tmx loaded")
		}()

	case 1:
		// TMX loaded — setup World image + Camera
		s.scene.SetupWorld()

		gd.LoadStatus = "Loading fingerprint database..."
		gd.LoadProgress = 0.6
		gd.LoadDone = make(chan struct{})

		go func() {
			defer close(gd.LoadDone)

			dbPath := domain.DefaultDBPath()

			db, dbErr := domain.LoadDB(dbPath)
			if dbErr != nil {
				slog.Info("generating fingerprint DB")

				db = domain.GenerateDB(42)
				if err := db.Save(dbPath); err != nil {
					slog.Warn("save db", "error", err)
				}
			}

			gd.DB = db
		}()

	case 2:
		gd.LoadStatus = "Generating puzzles..."
		gd.LoadProgress = 0.8
		gd.LoadDone = make(chan struct{})

		go func() {
			defer close(gd.LoadDone)

			if gd.DB == nil {
				return
			}

			puzzlesPath := domain.DefaultPuzzlesPath()
			loadedCases, loadedSeed, loadErr := domain.LoadPuzzles(puzzlesPath, gd.DB)

			if loadErr != nil {
				gd.PuzzleSeed = 99
				gd.Cases = domain.GenerateCases(gd.DB, 99)

				if err := domain.SavePuzzles(gd.Cases, 99, puzzlesPath); err != nil {
					slog.Warn("save puzzles", "error", err)
				}
			} else {
				gd.PuzzleSeed = loadedSeed
				gd.Cases = loadedCases
			}
		}()

	case 3:
		gd.LoadStatus = "Ready"
		gd.LoadProgress = 1.0
		gd.SelectedCase = 0
		s.scene.LoadGameState()
		state.Current = c.StateDisabled
		state.BootTick = 0
		slog.Info("game ready", "cases", len(gd.Cases))
	}
}

func (s *StateSystem) getGameData(reg RegType) *c.GameData {
	return GetGameData(reg)
}

func (s *StateSystem) enterEnabled(_ RegType) {
	s.scene.RegisterEnabledZones()
	slog.Info("entered enabled state", "inputZones", s.scene.GetInputSystem() != nil)
}

func (s *StateSystem) enterAppLayout(reg RegType) {
	s.scene.RegisterAppLayoutZones()
}

// RegType is a convenience alias.
type RegType = *registry.Registry[string, uint64, any]

// findTMXPath locates the fingerprint.tmx file.
func findTMXPath() string {
	candidates := []string{
		"assets/external/fingerprint/fingerprint.tmx",
		"../assets/external/fingerprint/fingerprint.tmx",
		"../../assets/external/fingerprint/fingerprint.tmx",
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}
