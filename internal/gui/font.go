package gui

import (
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"golang.org/x/image/font/sfnt"

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

// fontSupportsGlyphs reports whether the font carries the glyphs the UI and
// typical tool/model output rely on: CJK ideographs, full-width punctuation,
// arrows and check marks. Fyne does NOT fall back to system fonts for a
// custom theme font, so a custom font missing any of these would render
// replacement characters ('?') instead — exactly the mojibake reported with
// DroidSansFallbackFull.ttf.
func fontSupportsGlyphs(data []byte) bool {
	f, err := sfnt.Parse(data)
	if err != nil {
		return false
	}
	var b sfnt.Buffer
	for _, r := range []rune{'汉', '。', '←', '→', '↑', '↓', '✓', '✗'} {
		idx, err := f.GlyphIndex(&b, r)
		if err != nil || idx == 0 {
			return false
		}
	}
	return true
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
	if !fontSupportsGlyphs(res.Content()) {
		logging.Warn("custom font rejected: missing required glyphs (CJK/arrows/check marks), using built-in font", "path", fontPath)
		return base
	}
	return &fontTheme{Theme: base, font: res}
}

// applyThemeFont sets the base Fyne theme (light/dark/auto) while keeping any
// configured custom font active.
func (g *MainWindow) applyThemeFont(base fyne.Theme) {
	g.fyneApp.Settings().SetTheme(withCustomFont(base))
}
