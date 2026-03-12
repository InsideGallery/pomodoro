package platform

// HideWindow is a no-op on macOS.
func HideWindow(title string) {}

// ShowWindow is a no-op on macOS.
func ShowWindow(title string) {}

// SetAlwaysOnTop is a no-op on macOS.
func SetAlwaysOnTop(title string, enable bool) {}
