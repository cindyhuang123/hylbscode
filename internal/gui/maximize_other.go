//go:build !x11

package gui

import "fyne.io/fyne/v2"

// maximizeWindowNow is a no-op on non-X11 backends (Fyne has no maximize API
// and the GLFW window handle is X11-specific).
func maximizeWindowNow() bool {
	return false
}

// maximizedSize is not supported on non-X11 backends, so callers use their
// default size.
func maximizedSize() (fyne.Size, bool) {
	return fyne.Size{}, false
}
