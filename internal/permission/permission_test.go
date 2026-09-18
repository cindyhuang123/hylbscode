package permission

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathWithin(t *testing.T) {
	cases := []struct {
		name   string
		target string
		root   string
		want   bool
	}{
		{"same dir", "/a/b", "/a/b", true},
		{"child dir", "/a/b/c", "/a/b", true},
		{"deep child", "/a/b/c/d/e.txt", "/a/b", true},
		{"sibling dir", "/a/x", "/a/b", false},
		{"parent dir", "/a", "/a/b", false},
		{"unrelated", "/other", "/a/b", false},
		{"empty root", "/a/b", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, pathWithin(tc.target, tc.root))
		})
	}
}