//go:build !x11

package gui

import "fyne.io/fyne/v2"

// maximizedSize is not supported on non-X11 backends (Fyne has no maximize
// API and the GLFW query is X11-specific), so callers use their default size.
func maximizedSize() (fyne.Size, bool) {
	return fyne.Size{}, false
}
