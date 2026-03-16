package systems

import (
	"context"

	"github.com/InsideGallery/game-core/geometry/shapes"
	"github.com/InsideGallery/game-core/rtree"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ClickHandler is called when a clickable zone is clicked.
type ClickHandler func()

// DragHandler is called each frame while dragging, with mouse position.
type DragHandler func(mx, my int)

// HoverHandler is called each frame with whether the zone is hovered.
type HoverHandler func(hovered bool)

// Zone is a clickable/draggable area registered with the InputSystem.
type Zone struct {
	Spatial shapes.Spatial

	// OnClick is called on mouse release while still hovering (button behavior).
	OnClick ClickHandler

	// OnDragStart is called when a drag begins on this zone.
	OnDragStart func()

	// OnDrag is called each frame during drag with current mouse X.
	OnDrag DragHandler

	// OnDragEnd is called when the drag ends.
	OnDragEnd func()

	// OnHover is called each frame with hover state (for visual feedback).
	OnHover HoverHandler

	// Priority: lower value = checked first (for overlapping zones).
	Priority int
}

// InputSystem handles mouse interaction via RTree spatial queries.
// Supports click (press+release), drag, and hover detection.
type InputSystem struct {
	tree  *rtree.RTree
	zones []*Zone

	// Drag state
	dragging    bool
	dragZone    *Zone
	pressedZone *Zone

	// Scroll offset: subtracted from mouse Y to convert screen→content space.
	scrollOffsetY float64

	// CursorOverride: if set, use these coordinates instead of ebiten.CursorPosition().
	CursorOverride *[2]int
}

// NewInputSystem creates an InputSystem backed by the given RTree.
func NewInputSystem(tree *rtree.RTree) *InputSystem {
	return &InputSystem{tree: tree}
}

// AddZone registers an interactive zone in the spatial index.
func (s *InputSystem) AddZone(z *Zone) {
	s.zones = append(s.zones, z)
	s.tree.Insert(z.Spatial)
}

// ClearZones removes all registered zones from the spatial index.
func (s *InputSystem) ClearZones() {
	for _, z := range s.zones {
		s.tree.Delete(z.Spatial)
	}

	s.zones = nil
	s.dragging = false
	s.dragZone = nil
	s.pressedZone = nil
}

// Update processes mouse events: hover, press, drag, release.
func (s *InputSystem) Update(_ context.Context) error {
	var rawMX, rawMY int
	if s.CursorOverride != nil {
		rawMX, rawMY = s.CursorOverride[0], s.CursorOverride[1]
	} else {
		rawMX, rawMY = ebiten.CursorPosition()
	}

	mx := rawMX
	my := rawMY - int(s.scrollOffsetY)

	// Find hovered zone
	hovered := s.findZoneAt(mx, my)

	// Update hover state for all zones
	for _, z := range s.zones {
		if z.OnHover != nil {
			z.OnHover(z == hovered)
		}
	}

	// Handle drag continuation
	if s.dragging && s.dragZone != nil {
		if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			// Drag ended
			if s.dragZone.OnDragEnd != nil {
				s.dragZone.OnDragEnd()
			}

			s.dragging = false
			s.dragZone = nil
		} else if s.dragZone.OnDrag != nil {
			s.dragZone.OnDrag(mx, my)
		}

		return nil
	}

	// Handle press
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && hovered != nil {
		s.pressedZone = hovered

		if hovered.OnDragStart != nil {
			s.dragging = true
			s.dragZone = hovered
			hovered.OnDragStart()
		}
	}

	// Handle release (click = press + release on same zone)
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		if s.pressedZone != nil && s.pressedZone == hovered && s.pressedZone.OnClick != nil && !s.dragging {
			s.pressedZone.OnClick()
		}

		s.pressedZone = nil
	}

	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		s.pressedZone = nil
		s.dragging = false
		s.dragZone = nil
	}

	return nil
}

// Draw is a no-op; InputSystem has no visual representation.
func (s *InputSystem) Draw(_ context.Context, _ *ebiten.Image) {}

// SetScrollOffset sets a Y offset subtracted from mouse position before querying.
// Used for scrollable containers where zones are in content-space coordinates.
func (s *InputSystem) SetScrollOffset(y float64) {
	s.scrollOffsetY = y
}

func (s *InputSystem) findZoneAt(mx, my int) *Zone {
	clickPoint := shapes.NewSphere(shapes.NewPoint(float64(mx), float64(my)), 1)

	hits := s.tree.Collision(clickPoint, nil)
	if len(hits) == 0 {
		return nil
	}

	// Find the matching zone with highest priority (lowest value)
	var best *Zone

	for _, hit := range hits {
		for _, z := range s.zones {
			if z.Spatial == hit {
				if best == nil || z.Priority < best.Priority {
					best = z
				}
			}
		}
	}

	return best
}
