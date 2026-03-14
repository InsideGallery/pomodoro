package systems

import (
	"context"

	"github.com/InsideGallery/game-core/geometry/shapes"
	"github.com/InsideGallery/game-core/rtree"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/InsideGallery/pomodoro/internal/ecs/components"
)

// clickableSpatial wraps a Clickable to associate it with its spatial shape in the RTree.
type clickableSpatial struct {
	shapes.Spatial
	Clickable *components.Clickable
}

// InputSystem handles mouse click detection via RTree spatial queries.
// Register clickable entities by calling AddClickable; the system queries
// the RTree on each mouse click to find which entity was hit.
type InputSystem struct {
	tree       *rtree.RTree
	clickables []*clickableSpatial
}

// NewInputSystem creates an InputSystem backed by the given RTree.
func NewInputSystem(tree *rtree.RTree) *InputSystem {
	return &InputSystem{tree: tree}
}

// AddClickable registers a clickable component in the spatial index.
func (s *InputSystem) AddClickable(c *components.Clickable) {
	cs := &clickableSpatial{
		Spatial:   c.Spatial,
		Clickable: c,
	}

	s.clickables = append(s.clickables, cs)
	s.tree.Insert(cs.Spatial)
}

// ClearClickables removes all registered clickables from the spatial index.
func (s *InputSystem) ClearClickables() {
	for _, cs := range s.clickables {
		s.tree.Delete(cs.Spatial)
	}

	s.clickables = nil
}

// Update checks for mouse clicks and queries the RTree for hit entities.
func (s *InputSystem) Update(_ context.Context) error {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return nil
	}

	mx, my := ebiten.CursorPosition()
	clickPoint := shapes.NewSphere(shapes.NewPoint(float64(mx), float64(my)), 1)

	hits := s.tree.Collision(clickPoint, nil)

	for _, hit := range hits {
		for _, cs := range s.clickables {
			if cs.Spatial == hit && cs.Clickable.OnClick != nil {
				cs.Clickable.OnClick()

				return nil
			}
		}
	}

	return nil
}

// Draw is a no-op; InputSystem has no visual representation.
func (s *InputSystem) Draw(_ context.Context, _ *ebiten.Image) {}
