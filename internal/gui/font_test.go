package gui

import (
	"os"
	"testing"
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

func readFontFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Logf("cannot read %s: %v", path, err)
		return nil
	}
	return data
}
