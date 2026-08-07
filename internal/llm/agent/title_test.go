package agent

import "testing"

func TestIsDefaultTitle(t *testing.T) {
	cases := []struct {
		title string
		want  bool
	}{
		{"New Session", true},
		{"Non-interactive: 什么是 goroutine", true},
		{"解释闭包概念", false},
		{"Rust语言核心特性简介", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isDefaultTitle(c.title); got != c.want {
			t.Errorf("isDefaultTitle(%q) = %v, want %v", c.title, got, c.want)
		}
	}
}
