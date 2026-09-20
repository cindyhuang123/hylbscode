package desktop

import (
	"strings"
	"testing"
)

func TestDesktopEntryBasic(t *testing.T) {
	got := desktopEntry("/home/cindy/go/bin/hylbscode", "/home/cindy/.local/share/icons/hicolor/apps/hylbscode.png")
	want := "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=HyLbsCode\n" +
		"Comment=Desktop AI coding assistant\n" +
		"Exec=/home/cindy/go/bin/hylbscode\n" +
		"Icon=/home/cindy/.local/share/icons/hicolor/apps/hylbscode.png\n" +
		"Terminal=false\n" +
		"Categories=Development;\n" +
		"StartupWMClass=HyLbsCode\n"
	if got != want {
		t.Fatalf("desktop entry mismatch\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestDesktopEntryQuotesExecWithSpaces(t *testing.T) {
	got := desktopEntry("/home/user/my apps/hylbscode", "hylbscode.png")
	want := `Exec="/home/user/my apps/hylbscode"`
	if !strings.Contains(got, want) {
		t.Fatalf("expected quoted Exec with spaces, got:\n%s", got)
	}
}