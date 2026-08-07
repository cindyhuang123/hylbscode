package tools

import (
	"testing"
)

func TestGitReadOnlySubcommands(t *testing.T) {
	readonly := []string{"status", "diff", "log", "show", "branch", "ls-files", "rev-parse", "remote", "tag", "stash", "blame"}
	mutating := []string{"add", "commit", "reset", "clean", "checkout", "restore", "push", "pull", "merge", "rebase"}

	for _, sub := range readonly {
		if !readOnlySubcommands[sub] {
			t.Errorf("expected %q to be marked read-only", sub)
		}
	}
	for _, sub := range mutating {
		if readOnlySubcommands[sub] {
			t.Errorf("expected %q to require permission", sub)
		}
	}
}

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"status":       "'status'",
		"fix: thing":   `'fix: thing'`,
		"it's":         `'it'\''s'`,
		"a b":          "'a b'",
		"foo\"bar":     `'foo"bar'`,
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %q, want %q", in, got, want)
		}
	}
}
