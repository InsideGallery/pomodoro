package systems

import (
	"testing"

	"github.com/InsideGallery/game-core/geometry/shapes"
	"github.com/InsideGallery/game-core/rtree"

	"github.com/InsideGallery/pomodoro/internal/ecs/components"
)

func TestNewInputSystem(t *testing.T) {
	tree := rtree.NewRTree(rtree.DefaultMinRTreeOption, rtree.DefaultMaxRTreeOption)
	s := NewInputSystem(tree)

	if s == nil {
		t.Fatal("expected non-nil input system")
	}
}

func TestAddClickableInsertsIntoRTree(t *testing.T) {
	tree := rtree.NewRTree(rtree.DefaultMinRTreeOption, rtree.DefaultMaxRTreeOption)
	s := NewInputSystem(tree)

	spatial := shapes.NewSphere(shapes.NewPoint(100, 100), 20)
	c := &components.Clickable{
		Spatial:  spatial,
		OnClick:  func() {},
		EntityID: 1,
	}

	s.AddClickable(c)

	if tree.Size() != 1 {
		t.Fatalf("expected 1 entry in RTree, got %d", tree.Size())
	}
}

func TestClearClickables(t *testing.T) {
	tree := rtree.NewRTree(rtree.DefaultMinRTreeOption, rtree.DefaultMaxRTreeOption)
	s := NewInputSystem(tree)

	for i := range 5 {
		spatial := shapes.NewSphere(shapes.NewPoint(float64(i*100), 100), 20)
		s.AddClickable(&components.Clickable{
			Spatial:  spatial,
			OnClick:  func() {},
			EntityID: uint64(i),
		})
	}

	if tree.Size() != 5 {
		t.Fatalf("expected 5 entries, got %d", tree.Size())
	}

	s.ClearClickables()

	if tree.Size() != 0 {
		t.Fatalf("expected 0 entries after clear, got %d", tree.Size())
	}

	if len(s.clickables) != 0 {
		t.Fatalf("expected empty clickables slice, got %d", len(s.clickables))
	}
}

func TestRTreeCollisionFindsClickable(t *testing.T) {
	tree := rtree.NewRTree(rtree.DefaultMinRTreeOption, rtree.DefaultMaxRTreeOption)

	spatial := shapes.NewSphere(shapes.NewPoint(100, 100), 30)
	tree.Insert(spatial)

	// Query point inside the sphere
	queryPoint := shapes.NewSphere(shapes.NewPoint(110, 110), 1)
	hits := tree.Collision(queryPoint, nil)

	if len(hits) == 0 {
		t.Fatal("expected at least one hit")
	}
}

func TestRTreeCollisionMisses(t *testing.T) {
	tree := rtree.NewRTree(rtree.DefaultMinRTreeOption, rtree.DefaultMaxRTreeOption)

	spatial := shapes.NewSphere(shapes.NewPoint(100, 100), 10)
	tree.Insert(spatial)

	// Query point far from the sphere
	queryPoint := shapes.NewSphere(shapes.NewPoint(500, 500), 1)
	hits := tree.Collision(queryPoint, nil)

	if len(hits) != 0 {
		t.Fatalf("expected 0 hits, got %d", len(hits))
	}
}
