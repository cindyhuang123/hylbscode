package gui

import (
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/logging"
)

// fontTheme wraps a base Fyne theme and serves a custom font resource for
// every text style. Colors, sizes and icons keep coming from the base theme,
// so light/dark switching still works unchanged.
type fontTheme struct {
	fyne.Theme
	font fyne.Resource
}

// Font satisfies the fyne.Theme interface. When a custom font is configured,
// every style (regular, bold, italic, monospace) resolves to that same
// resource; Fyne falls back to its own synthesis for the style variants.
func (t *fontTheme) Font(style fyne.TextStyle) fyne.Resource {
	if t.font != nil {
		return t.font
	}
	return t.Theme.Font(style)
}

// withCustomFont applies the configured font file on top of the base theme.
// A nil base means "follow the system theme"; it is resolved to the default
// theme so the wrapper always has a concrete base to delegate to. If the font
// path is empty or cannot be loaded, the base theme is returned untouched so
// a broken font setting never breaks the UI.
func withCustomFont(base fyne.Theme) fyne.Theme {
	if base == nil {
		base = theme.DefaultTheme()
	}
	fontPath := config.Get().GUI.Font
	if fontPath == "" {
		return base
	}
	// .ttc collection files crash Fyne (nil face dereference during glyph
	// resolution), so they are rejected before loading.
	if ext := strings.ToLower(filepath.Ext(fontPath)); ext == ".ttc" {
		logging.Warn("custom font rejected: .ttc collections are not supported, using built-in font", "path", fontPath)
		return base
	}
	res, err := fyne.LoadResourceFromPath(fontPath)
	if err != nil {
		logging.Warn("custom font load failed, using built-in font", "path", fontPath, "error", err)
		return base
	}
	return &fontTheme{Theme: base, font: res}
}

// applyThemeFont sets the base Fyne theme (light/dark/auto) while keeping any
// configured custom font active.
func (g *MainWindow) applyThemeFont(base fyne.Theme) {
	g.fyneApp.Settings().SetTheme(withCustomFont(base))
}
