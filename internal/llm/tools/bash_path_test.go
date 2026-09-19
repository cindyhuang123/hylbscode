package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnalyzeScriptExecution(t *testing.T) {
	// B: executing a script through anything that is not bash always requires
	// confirmation, even when the script lives inside the working directory,
	// because analyzeScriptExecution cannot audit the script body.
	cases := []struct {
		name     string
		cmd      string
		wantNeed bool
	}{
		{"bash script internal", "bash ./gen.sh", false},
		{"sh script", "sh ./gen.sh", true},
		{"direct script", "./gen.sh", true},
		{"python script", "python3 ./gen.py", true},
		{"node script", "node app.js", true},
		{"ruby script", "ruby x.rb", true},
		{"perl script", "perl x.pl", true},
		{"php script", "php x.php", true},
		{"bash -c inline", "bash -c 'ls /etc'", false},
		{"python -c inline", "python3 -c 'print(1)'", true},
		{"unresolvable var", "python3 $UNKNOWN/x.py", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, gotNeed := analyzeScriptExecution(tc.cmd)
			assert.Equal(t, tc.wantNeed, gotNeed, "need for %q", tc.cmd)
		})
	}
}

func TestAnalyzeCommandPaths(t *testing.T) {
	home, err := os.UserHomeDir()
	assert.NoError(t, err)
	cases := []struct {
		name     string
		cmd      string
		wantPath string
		wantNeed bool
	}{
		{"absolute external path", "ls -la /etc", "/etc", true},
		{"external file read", "cat /etc/hostname", "/etc/hostname", true},
		{"with trailing punct", "ls /etc;", "/etc", true},
		{"redirect target", "echo hi > /etc/test.txt", "/etc", true},
		{"quoted external", "tail -n 5 \"/etc/hosts\"", "/etc/hosts", true},
		{"subdir under external root", "cat /etc/ssl/certs/ca-certificates.crt", "/etc/ssl/certs/ca-certificates.crt", true},
		{"find home", "find ~ -maxdepth 2 -name x", home, true},
		{"ls home var", "ls $HOME", home, true},
		{"home subpath", "ls ~/.config", filepath.Join(home, ".config"), true},
		{"unresolved variable path", "ls $UNKNOWN_VAR/x", "", true},
		{"env var without path", "echo $PATH", "", false},
		{"internal relative path", "go build ./...", "", false},
		{"internal dot path", "ls .", "", false},
		{"flag value internal", "git -C . status", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotPath, gotNeed := analyzeCommandPaths(tc.cmd)
			assert.Equal(t, tc.wantNeed, gotNeed, "needConfirm for %q", tc.cmd)
			if tc.wantPath != "" {
				assert.Equal(t, tc.wantPath, gotPath, "path for %q", tc.cmd)
			}
		})
	}
}
