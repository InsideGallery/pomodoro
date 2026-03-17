package components

import "github.com/InsideGallery/game-core/geometry/shapes"

// Clickable marks an entity as interactive in the RTree.
type Clickable struct {
	Spatial shapes.Spatial
}
