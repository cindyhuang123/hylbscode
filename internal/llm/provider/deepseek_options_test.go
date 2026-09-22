package provider

import "testing"

func TestWithThinkingDefaults(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty means disabled", "", "disabled"},
		{"blank means disabled", "  ", "disabled"},
		{"enabled", "enabled", "enabled"},
		{"enable alias", "enable", "enabled"},
		{"disabled", "disabled", "disabled"},
		{"case insensitive", "Enabled", "enabled"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := &openaiOptions{}
			WithThinking(tc.input)(opts)
			if opts.thinking != tc.expected {
				t.Fatalf("thinking = %q, want %q", opts.thinking, tc.expected)
			}
		})
	}
}

func TestWithReasoningEffortDefaults(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty means medium", "", "medium"},
		{"low", "low", "low"},
		{"medium", "medium", "medium"},
		{"high", "high", "high"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := &openaiOptions{}
			WithReasoningEffort(tc.input)(opts)
			if opts.reasoningEffort != tc.expected {
				t.Fatalf("reasoningEffort = %q, want %q", opts.reasoningEffort, tc.expected)
			}
		})
	}
}