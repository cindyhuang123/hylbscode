package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExternalPathInCommand(t *testing.T) {
	cases := []struct {
		name string
		cmd  string
		want string
	}{
		{"absolute external path", "ls -la /etc", "/etc"},
		{"external file read", "cat /etc/hostname", "/etc/hostname"},
		{"with trailing punct", "ls /etc;", "/etc"},
		{"redirect target", "echo hi > /etc/test.txt", "/etc"},
		{"quoted external", "tail -n 5 \"/etc/hosts\"", "/etc/hosts"},
		{"subdir under external root", "cat /etc/ssl/certs/ca-certificates.crt", "/etc/ssl/certs/ca-certificates.crt"},
		{"internal relative path", "go build ./...", ""},
		{"internal dot path", "ls .", ""},
		{"flag value internal", "git -C . status", ""},
		{"root-relative home", "cat ~/.bashrc", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := externalPathInCommand(tc.cmd)
			if tc.want == "" {
				assert.Equal(t, tc.want, got, "command %q should not be flagged", tc.cmd)
			} else {
				assert.Equal(t, tc.want, got, "command %q should be flagged", tc.cmd)
			}
		})
	}
}
