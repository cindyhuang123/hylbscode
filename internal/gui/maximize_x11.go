//go:build x11

package gui

import (
	"fyne.io/fyne/v2"
	"github.com/go-gl/glfw/v3.4/glfw"

	"github.com/cindyhuang123/hylbscode/internal/logging"
)

// maximizedSize returns a near-fullscreen size for the primary monitor,
// because Fyne has no window maximize API. A margin keeps the window inside
// the desktop work area (title bar / taskbar). Falls back to the caller's
// default size when the monitor cannot be queried.
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
