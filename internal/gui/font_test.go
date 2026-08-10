package gui

import (
	"os"
	"testing"

	"github.com/cindyhuang123/hylbscode/internal/config"
)

// TestFontSupportsGlyphs verifies the custom-font gate: the Droid
// SansFallback font on this system lacks arrows/check marks, so it must be
// rejected to avoid replacement characters; malformed data is rejected too.
func TestFontSupportsGlyphs(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want bool
	}{
		{"empty data", nil, false},
		{"garbage", []byte("not a font at all"), false},
		{"droid-sans-fallback-missing-symbols", readFontFile(t, "/usr/share/fonts/google-droid-sans-fonts/DroidSansFallbackFull.ttf"), false},
	}
	for _, c := range cases {
		if c.name == "droid-sans-fallback-missing-symbols" && len(c.data) == 0 {
			t.Skip("droid font not present on this system")
		}
		if got := fontSupportsGlyphs(c.data); got != c.want {
			t.Errorf("%s: fontSupportsGlyphs = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestEmbeddedCJKFont verifies the bundled Noto Sans SC font passes the
// glyph gate and becomes the default when no custom font is configured.
func TestEmbeddedCJKFont(t *testing.T) {
	if _, err := config.Load(t.TempDir(), false); err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if len(embeddedCJKFont) == 0 {
		t.Fatal("embedded CJK font is empty")
	}
	if !fontSupportsGlyphs(embeddedCJKFont) {
		t.Error("embedded CJK font failed the glyph check")
	}
	th := withCustomFont(nil)
	ft, ok := th.(*fontTheme)
	if !ok || ft.font == nil {
		t.Fatal("withCustomFont(nil) should apply the embedded CJK font")
	}
	if ft.font.Name() != "NotoSansSC-Regular.otf" {
		t.Errorf("embedded font name = %q", ft.font.Name())
	}
}

func readFontFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Logf("cannot read %s: %v", path, err)
		return nil
	}
	return data
}
