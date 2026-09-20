// Package desktop installs the platform-specific integration files that let
// the desktop environment (dock/taskbar/launcher) display the application
// icon, because the mechanisms differ per OS:
//
//   - Linux: GNOME/KDE/XFCE docks and taskbars resolve the icon from a
//     freedesktop .desktop entry (Icon + StartupWMClass), not from the
//     window _NET_WM_ICON property. We install ~/.local/share/icons/... and
//     ~/.local/share/applications/hylbscode.desktop pointing at the currently
//     running binary, so the custom icon appears after a re-login.
//   - Windows: the taskbar icon follows the window icon (WM_SETICON) set by
//     Fyne, so no extra files are needed.
//   - macOS: a standalone binary cannot change the Dock icon at runtime; the
//     .app bundle must ship a .icns. Nothing to do here.
package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
)

const (
	appID    = "hylbscode"
	appName  = "HyLbsCode"
	wmClass  = "HyLbsCode"
	iconName = "hylbscode"
)

// EnsureIcon makes the desktop environment show the given resource as the
// application icon. It is idempotent and safe to call on every startup.
// Failures are returned to the caller, which should log and continue.
func EnsureIcon(res fyne.Resource) error {
	if res == nil || len(res.Content()) == 0 {
		return nil
	}
	if runtime.GOOS != "linux" {
		return nil
	}
	return installLinux(res)
}

func installLinux(res fyne.Resource) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	iconDir := filepath.Join(home, ".local", "share", "icons", "hicolor", "apps")
	iconPath := filepath.Join(iconDir, iconName+".png")
	if err := os.MkdirAll(iconDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(iconPath, res.Content(), 0o644); err != nil {
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	appsDir := filepath.Join(home, ".local", "share", "applications")
	if err := os.MkdirAll(appsDir, 0o755); err != nil {
		return err
	}
	entry := desktopEntry(exe, iconPath)
	return os.WriteFile(filepath.Join(appsDir, appID+".desktop"), []byte(entry), 0o644)
}

// desktopEntry renders the freedesktop .desktop content. Exec points at the
// currently running binary (quoted when it contains spaces) so launchers and
// the dock start the same build that is running; StartupWMClass must match
// the window class set by the GUI so the window group gets the icon.
func desktopEntry(exe, icon string) string {
	exec := exe
	if strings.ContainsAny(exec, " \t") {
		exec = "\"" + exec + "\""
	}
	return "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=" + appName + "\n" +
		"Comment=Desktop AI coding assistant\n" +
		"Exec=" + exec + "\n" +
		"Icon=" + icon + "\n" +
		"Terminal=false\n" +
		"Categories=Development;\n" +
		"StartupWMClass=" + wmClass + "\n"
}