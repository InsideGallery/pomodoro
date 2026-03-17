package components

// Scrollable marks an entity as a scrollable list container.
type Scrollable struct {
	Scroll     int     // current scroll offset (items)
	RowH       float64 // row height in map units
	TotalItems int     // total number of items
}
