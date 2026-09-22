package provider

import (
	"testing"

	"github.com/cindyhuang123/hylbscode/internal/llm/models"
	"github.com/cindyhuang123/hylbscode/internal/message"
)

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

type deepSeekToggleCase struct {
	name       string
	provider   models.ModelProvider
	thinking   string
	effort     string
	messages   []message.Message
	wantThk    string
	wantEffort string
	wantOK     bool
}

func runDeepSeekToggle(t *testing.T, c deepSeekToggleCase) {
	t.Helper()
	o := &openaiClient{
		providerOptions: providerClientOptions{model: models.Model{Provider: c.provider}},
		options:         openaiOptions{thinking: c.thinking, reasoningEffort: c.effort},
	}
	thk, effort, ok := o.deepSeekThinkingToggle(c.messages)
	if ok != c.wantOK {
		t.Fatalf("ok = %v, want %v", ok, c.wantOK)
	}
	if thk != c.wantThk {
		t.Fatalf("thinking = %q, want %q", thk, c.wantThk)
	}
	if effort != c.wantEffort {
		t.Fatalf("effort = %q, want %q", effort, c.wantEffort)
	}
}

func TestDeepSeekThinkingToggle(t *testing.T) {
	userMsg := func() []message.Message { return []message.Message{{Role: message.User}} }
	toolMsg := func() []message.Message { return []message.Message{{Role: message.Tool}} }

	runDeepSeekToggle(t, deepSeekToggleCase{
		name: "non-deepseek provider returns ok=false",
		provider: models.ProviderOpenAI, thinking: "enabled", effort: "high",
		messages: userMsg(), wantOK: false,
	})
	runDeepSeekToggle(t, deepSeekToggleCase{
		name: "enabled + user tail injects effort",
		provider: models.ProviderDeepSeek, thinking: "enabled", effort: "high",
		messages: userMsg(), wantThk: "enabled", wantEffort: "high", wantOK: true,
	})
	runDeepSeekToggle(t, deepSeekToggleCase{
		name: "enabled + empty effort falls back to medium",
		provider: models.ProviderDeepSeek, thinking: "enabled",
		messages: userMsg(), wantThk: "enabled", wantEffort: "medium", wantOK: true,
	})
	runDeepSeekToggle(t, deepSeekToggleCase{
		name: "enabled + tool tail downgrades and drops effort",
		provider: models.ProviderDeepSeek, thinking: "enabled", effort: "high",
		messages: toolMsg(), wantThk: "disabled", wantEffort: "", wantOK: true,
	})
	runDeepSeekToggle(t, deepSeekToggleCase{
		name: "disabled keeps disabled and drops effort",
		provider: models.ProviderDeepSeek, thinking: "disabled", effort: "high",
		messages: userMsg(), wantThk: "disabled", wantEffort: "", wantOK: true,
	})
}