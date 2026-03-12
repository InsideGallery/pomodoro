package platform

// HideWindow is a no-op on Windows.
func HideWindow(title string) {}

// ShowWindow is a no-op on Windows.
func ShowWindow(title string) {}

// SetAlwaysOnTop is a no-op on Windows.
func SetAlwaysOnTop(title string, enable bool) {}
