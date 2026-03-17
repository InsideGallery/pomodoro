package assets

import "embed"

// FingerprintStories is the stories.json for the fingerprint game.
//
//go:embed external/fingerprint/stories.json
var FingerprintStories []byte

// FingerprintImages contains fingerprint source images (4 colors × 4 variants + 4 grey).
//
//go:embed external/fingerprint/fingerprints
var FingerprintImages embed.FS

// FingerprintAvatars contains character avatar images.
//
//go:embed external/fingerprint/avatars
var FingerprintAvatars embed.FS

// FingerprintUI contains UI images (cursor, buttons, icons).
//
//go:embed external/fingerprint/ui
var FingerprintUI embed.FS

// FingerprintTMX is the main TMX map file.
//
//go:embed external/fingerprint/fingerprint.tmx
var FingerprintTMX []byte

// FingerprintTilesets contains tileset definition files.
//
//go:embed external/fingerprint/fingerprinting-icon.tsx
//go:embed external/fingerprint/avatars.tsx
//go:embed external/fingerprint/buttons.tsx
//go:embed external/fingerprint/exit-activated.tsx
var FingerprintTilesets embed.FS
