package gui

import "testing"

func TestFormatTokens(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0k"},
		{12800, "12.8k"},
		{102400, "102.4k"},
		{1_000_000, "1.0M"},
		{2_500_000, "2.5M"},
	}
	for _, c := range cases {
		if got := formatTokens(c.in); got != c.want {
			t.Errorf("formatTokens(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestShortenPath(t *testing.T) {
	short := "/home/u/proj"
	if got := shortenPath(short); got != short {
		t.Errorf("short path modified: %q", got)
	}
	long := "/home/cindy/TMP/lbs_code_dir/20260914_1814/very/long/sub/dir/path/here"
	got := shortenPath(long)
	if len(got) > 44+3+1 {
		t.Errorf("shortenPath result too long: %d chars (%q)", len(got), got)
	}
	if got[:18] != long[:18] || got[len(got)-20:] != long[len(long)-20:] {
		t.Errorf("shortenPath lost head/tail: %q", got)
	}
}
