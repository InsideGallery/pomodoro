package systems

import (
	"testing"

	"github.com/InsideGallery/game-core/geometry/shapes"
	"github.com/InsideGallery/game-core/rtree"
)

func TestNewInputSystem(t *testing.T) {
	tree := rtree.NewRTree(rtree.DefaultMinRTreeOption, rtree.DefaultMaxRTreeOption)
	s := NewInputSystem(tree)

	if s == nil {
		t.Fatal("expected non-nil input system")
	}
}

func TestAddZoneInsertsIntoRTree(t *testing.T) {
	tree := rtree.NewRTree(rtree.DefaultMinRTreeOption, rtree.DefaultMaxRTreeOption)
	s := NewInputSystem(tree)

	s.AddZone(&Zone{
		Spatial: shapes.NewSphere(shapes.NewPoint(100, 100), 20),
		OnClick: func() {},
	})

	if tree.Size() != 1 {
		t.Fatalf("expected 1 entry in RTree, got %d", tree.Size())
	}
}

func TestClearZones(t *testing.T) {
	tree := rtree.NewRTree(rtree.DefaultMinRTreeOption, rtree.DefaultMaxRTreeOption)
	s := NewInputSystem(tree)

	for i := range 5 {
		s.AddZone(&Zone{
			Spatial: shapes.NewSphere(shapes.NewPoint(float64(i*100), 100), 20),
			OnClick: func() {},
		})
	}

	if tree.Size() != 5 {
		t.Fatalf("expected 5 entries, got %d", tree.Size())
	}

	s.ClearZones()

	if tree.Size() != 0 {
		t.Fatalf("expected 0 entries after clear, got %d", tree.Size())
	}

	if len(s.zones) != 0 {
		t.Fatalf("expected empty zones slice, got %d", len(s.zones))
	}
}

func TestRTreeCollisionFindsZone(t *testing.T) {
	tree := rtree.NewRTree(rtree.DefaultMinRTreeOption, rtree.DefaultMaxRTreeOption)

	spatial := shapes.NewSphere(shapes.NewPoint(100, 100), 30)
	tree.Insert(spatial)

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

	queryPoint := shapes.NewSphere(shapes.NewPoint(500, 500), 1)
	hits := tree.Collision(queryPoint, nil)

	if len(hits) != 0 {
		t.Fatalf("expected 0 hits, got %d", len(hits))
	}
}

func TestFindZoneAtWithPriority(t *testing.T) {
	tree := rtree.NewRTree(rtree.DefaultMinRTreeOption, rtree.DefaultMaxRTreeOption)
	s := NewInputSystem(tree)

	// Two overlapping zones at same location
	z1 := &Zone{
		Spatial:  shapes.NewSphere(shapes.NewPoint(100, 100), 50),
		OnClick:  func() {},
		Priority: 10, // lower priority
	}

	z2 := &Zone{
		Spatial:  shapes.NewSphere(shapes.NewPoint(100, 100), 30),
		OnClick:  func() {},
		Priority: 1, // higher priority (lower number)
	}

	s.AddZone(z1)
	s.AddZone(z2)

	found := s.findZoneAt(100, 100)

	if found != z2 {
		t.Fatal("expected higher priority zone (z2) to be selected")
	}
}
