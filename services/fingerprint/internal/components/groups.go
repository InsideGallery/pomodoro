package components

// Registry group names. Each group acts as an archetype —
// entities in the same group share the same component composition.
const (
	GroupGameState     = "game_state"     // 1 entity: State + GameData
	GroupCursor        = "cursor"         // 1 entity: Cursor
	GroupProgress      = "progress"       // 1 entity: ProgressBar (legacy, merged into GameData)
	GroupImageLayer    = "image_layer"    // N entities: ImageLayerComp + Renderable
	GroupTileLayer     = "tile_layer"     // N entities: TileLayerComp + Renderable
	GroupButton        = "button"         // N entities: Button + Transform + Clickable
	GroupCaseList      = "case_list"      // 1 entity: Scrollable + Transform
	GroupPuzzleList    = "puzzle_list"    // 1 entity: Scrollable + Transform
	GroupAvatar        = "avatar"         // 1 entity: Avatar + Transform
	GroupDescription   = "description"    // 1 entity: TextBlock + Transform
	GroupPuzzleGrid    = "puzzle_grid"    // 1 entity: PuzzleGrid + Transform
	GroupGridCell      = "grid_cell"      // 100 entities: Sprite + Transform + Renderable
	GroupTrayPiece     = "tray_piece"     // 3N entities: PuzzlePiece + Sprite + Transform + Draggable
	GroupHeldPiece     = "held_piece"     // 0-1 entity: Sprite + Transform + Renderable
	GroupHashDisplay   = "hash_display"   // 1 entity: TextBlock + Transform
	GroupResultOverlay = "result_overlay" // 0-1 entity: TileLayerComp + Renderable
)

// Entity is a container for components associated with one entity.
// Systems access components by type assertion from the Registry value.
type Entity struct {
	Transform   *Transform
	Sprite      *Sprite
	Renderable  *Renderable
	Button      *Button
	Clickable   *Clickable
	Scrollable  *Scrollable
	TextBlock   *TextBlock
	Avatar      *Avatar
	ImageLayer  *ImageLayerComp
	TileLayer   *TileLayerComp
	PuzzleGrid  *PuzzleGrid
	PuzzlePiece *PuzzlePiece
	Draggable   *Draggable
	Cursor      *Cursor
	State       *State
	Progress    *ProgressBar
	GameData    *GameData
}
