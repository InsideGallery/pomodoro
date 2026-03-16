// Package tilemap provides Tiled TMX map loading and rendering utilities.
// Reuses patterns from InsideGallery/detective.
package tilemap

import (
	"fmt"
	"image"
	"log/slog"
	"os"
	"path/filepath"

	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder

	"github.com/InsideGallery/game-core/geometry/shapes"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/lafriks/go-tiled"
)

// Map wraps a tiled.Map with loaded image cache and helper methods.
type Map struct {
	*tiled.Map
	basePath string
	images   map[string]*ebiten.Image // source path → loaded image
}

// Load parses a TMX file from disk and pre-loads all referenced images.
func Load(tmxPath string) (*Map, error) {
	gameMap, err := tiled.LoadFile(tmxPath)
	if err != nil {
		return nil, fmt.Errorf("load tmx %s: %w", tmxPath, err)
	}

	m := &Map{
		Map:      gameMap,
		basePath: filepath.Dir(tmxPath),
		images:   make(map[string]*ebiten.Image),
	}

	// Pre-load all image layer images
	for _, il := range gameMap.ImageLayers {
		if il.Image != nil && il.Image.Source != "" {
			if err := m.loadImage(il.Image.Source); err != nil {
				slog.Warn("tilemap: load image layer", "source", il.Image.Source, "error", err)
			}
		}
	}

	// Pre-load all tileset images
	for _, ts := range gameMap.Tilesets {
		if ts.Image != nil && ts.Image.Source != "" {
			if err := m.loadImage(ts.Image.Source); err != nil {
				slog.Warn("tilemap: load tileset", "source", ts.Image.Source, "error", err)
			}
		}

		// Collection-of-images tilesets
		for _, t := range ts.Tiles {
			if t.Image != nil && t.Image.Source != "" {
				if err := m.loadImage(t.Image.Source); err != nil {
					slog.Warn("tilemap: load tile image", "source", t.Image.Source, "error", err)
				}
			}
		}
	}

	return m, nil
}

// GetImage returns a pre-loaded image by its source path.
func (m *Map) GetImage(source string) *ebiten.Image {
	return m.images[source]
}

// ImageLayerImage returns the ebiten.Image for an image layer.
func (m *Map) ImageLayerImage(layer *tiled.ImageLayer) *ebiten.Image {
	if layer == nil || layer.Image == nil {
		return nil
	}

	return m.images[layer.Image.Source]
}

// FindImageLayer finds an image layer by name.
func (m *Map) FindImageLayer(name string) *tiled.ImageLayer {
	for _, il := range m.ImageLayers {
		if il.Name == name {
			return il
		}
	}

	return nil
}

// FindObjectGroup finds an object group by name.
func (m *Map) FindObjectGroup(name string) *tiled.ObjectGroup {
	for _, og := range m.ObjectGroups {
		if og.Name == name {
			return og
		}
	}

	return nil
}

// FindTileLayer finds a tile layer by name.
func (m *Map) FindTileLayer(name string) *tiled.Layer {
	for _, l := range m.Layers {
		if l.Name == name {
			return l
		}
	}

	return nil
}

// FindObject finds an object by name within an object group.
func FindObject(og *tiled.ObjectGroup, name string) *tiled.Object {
	if og == nil {
		return nil
	}

	for _, obj := range og.Objects {
		if obj.Name == name {
			return obj
		}
	}

	return nil
}

// ObjectToSpatial converts a Tiled object to a shapes.Spatial for RTree.
// Reused from InsideGallery/detective.
func ObjectToSpatial(obj *tiled.Object) shapes.Spatial { //nolint:ireturn // spatial for RTree
	if obj.Polygons != nil {
		var points []shapes.Point

		for _, l := range obj.Polygons {
			ul := *l.Points
			for _, p := range ul {
				points = append(points, shapes.NewPoint(obj.X+p.X, obj.Y+p.Y))
			}
		}

		return shapes.NewPolyhedron(points...)
	}

	if obj.PolyLines != nil {
		var points []shapes.Point

		for _, l := range obj.PolyLines {
			ul := *l.Points
			for _, p := range ul {
				points = append(points, shapes.NewPoint(obj.X+p.X, obj.Y+p.Y))
			}
		}

		return shapes.NewPolyhedron(points...)
	}

	return shapes.NewBox(shapes.NewPoint(obj.X, obj.Y), obj.Width, obj.Height)
}

// DrawImageLayer draws an image layer at its position, scaled to fit the target.
func DrawImageLayer(
	dst *ebiten.Image, _ *tiled.ImageLayer, img *ebiten.Image,
	scaleX, scaleY, offsetX, offsetY float64,
) {
	if img == nil {
		return
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Translate(offsetX, offsetY)
	dst.DrawImage(img, op)
}

// DrawTileLayer renders a tile layer using tileset images.
func (m *Map) DrawTileLayer(dst *ebiten.Image, layer *tiled.Layer, scaleX, scaleY, offsetX, offsetY float64) {
	if layer == nil || layer.Tiles == nil {
		return
	}

	tileW := m.TileWidth
	tileH := m.TileHeight

	for i, tile := range layer.Tiles {
		if tile.IsNil() {
			continue
		}

		col := i % m.Width
		row := i / m.Width

		tileImg := m.getTileImage(tile)
		if tileImg == nil {
			continue
		}

		// Large tiles anchor at bottom-left of grid cell (Tiled convention)
		imgH := float64(tileImg.Bounds().Dy())
		yOffset := (imgH - float64(tileH)) * scaleY

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scaleX, scaleY)
		op.GeoM.Translate(
			offsetX+float64(col)*float64(tileW)*scaleX,
			offsetY+float64(row)*float64(tileH)*scaleY-yOffset,
		)

		dst.DrawImage(tileImg, op)
	}
}

func (m *Map) getTileImage(tile *tiled.LayerTile) *ebiten.Image {
	if tile.IsNil() {
		return nil
	}

	ts := tile.Tileset
	if ts == nil {
		return nil
	}

	localID := tile.ID // already local (go-tiled subtracts FirstGID)

	// Collection-of-images tileset
	for _, t := range ts.Tiles {
		if t.ID == localID && t.Image != nil {
			return m.images[t.Image.Source]
		}
	}

	// Single-image tileset — extract sub-rectangle
	if ts.Image == nil {
		return nil
	}

	srcImg := m.images[ts.Image.Source]
	if srcImg == nil {
		return nil
	}

	cols := ts.Columns
	if cols == 0 {
		cols = 1
	}

	tileX := int(localID) % cols
	tileY := int(localID) / cols
	rect := image.Rect(
		tileX*ts.TileWidth, tileY*ts.TileHeight,
		(tileX+1)*ts.TileWidth, (tileY+1)*ts.TileHeight,
	)

	return srcImg.SubImage(rect).(*ebiten.Image)
}

func (m *Map) loadImage(source string) error {
	if _, ok := m.images[source]; ok {
		return nil // already loaded
	}

	fullPath := filepath.Join(m.basePath, source)

	f, err := os.Open(fullPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", fullPath, err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("decode %s: %w", fullPath, err)
	}

	m.images[source] = ebiten.NewImageFromImage(img)

	slog.Debug("tilemap: loaded image", "source", source,
		"size", fmt.Sprintf("%dx%d", img.Bounds().Dx(), img.Bounds().Dy()))

	return nil
}

// MapPixelWidth returns the total map width in pixels.
func (m *Map) MapPixelWidth() int {
	return m.Width * m.TileWidth
}

// MapPixelHeight returns the total map height in pixels.
func (m *Map) MapPixelHeight() int {
	return m.Height * m.TileHeight
}
