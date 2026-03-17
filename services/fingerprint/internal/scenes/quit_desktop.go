//go:build !android

package scenes

import "os"

// QuitGame exits the application. On desktop: os.Exit. On mobile: no-op.
func QuitGame() {
	os.Exit(0)
}
