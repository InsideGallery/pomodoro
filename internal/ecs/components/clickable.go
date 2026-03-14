package components

import "github.com/InsideGallery/game-core/geometry/shapes"

// Clickable marks an entity as having a click zone.
// Spatial is the collision shape inserted into RTree.
type Clickable struct {
	Spatial  shapes.Spatial
	OnClick  func()
	EntityID uint64
}
