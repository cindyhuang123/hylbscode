//go:build x11

package gui

import (
	"fyne.io/fyne/v2"
	"github.com/go-gl/glfw/v3.4/glfw"

	"github.com/cindyhuang123/hylbscode/internal/logging"
)

// maximizeWindowNow performs a real system-level maximize (identical to
// clicking the window's maximize button) on the current GLFW context, which is
// the Fyne window when run on the UI thread. Fyne has no maximize API, so this
// goes through GLFW directly. Returns false when the window cannot be found.
func maximizeWindowNow() (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			logging.Warn("maximize failed, falling back to sized window", "err", r)
			ok = false
		}
	}()
	if err := glfw.Init(); err != nil {
		return false
	}
	w := glfw.GetCurrentContext()
	if w == nil {
		return false
	}
	w.Maximize()
	return true
}

// maximizedSize returns a near-fullscreen size for the primary monitor, used
// as a fallback when the GLFW window handle is unavailable. A margin keeps the
// window inside the desktop work area (title bar / taskbar).
func maximizedSize() (fyne.Size, bool) {
	defer func() {
		if r := recover(); r != nil {
			logging.Warn("maximized size lookup failed, using default size", "err", r)
		}
	}()
	if err := glfw.Init(); err != nil {
		return fyne.Size{}, false
	}
	mon := glfw.GetPrimaryMonitor()
	if mon == nil {
		return fyne.Size{}, false
	}
	mode := mon.GetVideoMode()
	if mode == nil {
		return fyne.Size{}, false
	}
	w := float32(mode.Width) - 120
	h := float32(mode.Height) - 120
	if w < 100 || h < 100 {
		return fyne.Size{}, false
	}
	return fyne.NewSize(w, h), true
}
