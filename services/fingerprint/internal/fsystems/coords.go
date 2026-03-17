package fsystems

// Coords provides map↔screen coordinate conversion.
// All systems use these helpers instead of manual scaleX/offsetX math.
type Coords struct {
	Scale   float64 // uniform scale factor (fit-to-screen)
	OffsetX float64 // horizontal centering offset
	OffsetY float64 // vertical centering offset (for future portrait screens)
}

// CoordsFromScene builds Coords from the scene accessor.
func CoordsFromScene(scene SceneAccessor) Coords {
	return Coords{
		Scale:   scene.GetScaleX(), // uniform: scaleX == scaleY
		OffsetX: scene.GetOffsetX(),
	}
}

// MapToScreenX converts map X coordinate to screen pixel.
func (c Coords) MapToScreenX(mapX float64) float64 {
	return mapX*c.Scale + c.OffsetX
}

// MapToScreenY converts map Y coordinate to screen pixel.
func (c Coords) MapToScreenY(mapY float64) float64 {
	return mapY*c.Scale + c.OffsetY
}

// MapToScreenSize converts a map dimension (width or height) to screen pixels.
func (c Coords) MapToScreenSize(mapSize float64) float64 {
	return mapSize * c.Scale
}

// ScreenToMapX converts screen pixel X to map coordinate.
func (c Coords) ScreenToMapX(screenX float64) float64 {
	return (screenX - c.OffsetX) / c.Scale
}

// ScreenToMapY converts screen pixel Y to map coordinate.
func (c Coords) ScreenToMapY(screenY float64) float64 {
	return (screenY - c.OffsetY) / c.Scale
}

// MapRect converts a map-space rectangle to screen-space values.
func (c Coords) MapRect(mapX, mapY, mapW, mapH float64) (sx, sy, sw, sh float64) {
	return c.MapToScreenX(mapX), c.MapToScreenY(mapY), c.MapToScreenSize(mapW), c.MapToScreenSize(mapH)
}
