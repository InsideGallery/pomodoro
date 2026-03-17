//go:build android

package platform

// HideWindow is a no-op on Android.
func HideWindow(title string) {}

// ShowWindow is a no-op on Android.
func ShowWindow(title string) {}

// SetAlwaysOnTop is a no-op on Android.
func SetAlwaysOnTop(title string, enable bool) {}

// RaiseWindow is a no-op on Android.
func RaiseWindow(title string) {}
