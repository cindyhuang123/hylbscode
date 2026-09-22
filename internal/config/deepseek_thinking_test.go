package config

import (
	"fmt"
	"testing"

	"github.com/cindyhuang123/hylbscode/internal/llm/models"
	"github.com/spf13/viper"
)

func TestDeepSeekThinkingDefaults(t *testing.T) {
	// isolate github-copilot token lookup from the real HOME
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// clear every provider that could short-circuit before the deepseek branch
	for _, env := range []string{
		"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "OPENROUTER_API_KEY",
		"GEMINI_API_KEY", "GROQ_API_KEY", "XAI_API_KEY", "GLM_API_KEY",
		"AZURE_OPENAI_ENDPOINT", "AZURE_OPENAI_API_KEY", "GITHUB_TOKEN",
		"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_PROFILE",
		"AWS_DEFAULT_PROFILE", "AWS_REGION", "AWS_DEFAULT_REGION",
		"AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", "AWS_CONTAINER_CREDENTIALS_FULL_URI",
		"VERTEXAI_PROJECT", "VERTEXAI_LOCATION", "GOOGLE_CLOUD_PROJECT",
		"GOOGLE_CLOUD_REGION", "GOOGLE_CLOUD_LOCATION",
	} {
		t.Setenv(env, "")
	}
	t.Setenv("DEEPSEEK_API_KEY", "test-deepseek-key")

	viper.Reset()
	configureViper()
	setDefaults(false)
	setProviderDefaults()

	checks := []struct {
		key  string
		want string
	}{
		{"agents.coder.model", string(models.DeepSeekV4Flash)},
		{"agents.task.model", string(models.DeepSeekV4Flash)},
		{"agents.coder.thinking", "enabled"},
		{"agents.task.thinking", "enabled"},
		{"agents.summarizer.thinking", "disabled"},
		{"agents.title.thinking", "disabled"},
	}
	for _, c := range checks {
		got := viper.Get(c.key)
		if got == nil || fmt.Sprint(got) != c.want {
			t.Fatalf("%s = %v, want %q", c.key, got, c.want)
		}
	}
}