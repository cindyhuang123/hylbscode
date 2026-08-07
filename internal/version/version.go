package version

import (
	"runtime/debug"

	"github.com/cindyhuang123/hylbscode/internal/logging"
)

// Build-time parameters set via -ldflags
var Version = "1.0.0"

// A user may install pug using `go install hylbscode@latest`.
// without -ldflags, in which case the version above is unset. As a workaround
// we use the embedded build version that *is* set when using `go install` (and
// is only set for `go install` and not for `go build`).
func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		// < go v1.18
		return
	}
	mainVersion := info.Main.Version
	if mainVersion == "" || mainVersion == "(devel)" {
		// bin not built using `go install`
		return
	}
	logging.Info("Build info", "version", mainVersion)
	// bin built using `go install`
	//Version = mainVersion
}
