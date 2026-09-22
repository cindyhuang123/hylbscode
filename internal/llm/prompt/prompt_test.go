package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/llm/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetContextFromPaths(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	_, err := config.Load(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	cfg := config.Get()
	cfg.WorkingDir = tmpDir
	cfg.ContextPaths = []string{
		"file.txt",
		"directory/",
	}
	testFiles := []string{
		"file.txt",
		"directory/file_a.txt",
		"directory/file_b.txt",
		"directory/file_c.txt",
	}

	createTestFiles(t, tmpDir, testFiles)

	context := getContextFromPaths()
	expectedContext := fmt.Sprintf("# From:%s/file.txt\nfile.txt: test content\n# From:%s/directory/file_a.txt\ndirectory/file_a.txt: test content\n# From:%s/directory/file_b.txt\ndirectory/file_b.txt: test content\n# From:%s/directory/file_c.txt\ndirectory/file_c.txt: test content", tmpDir, tmpDir, tmpDir, tmpDir)
	assert.Equal(t, expectedContext, context)
}

func createTestFiles(t *testing.T, tmpDir string, testFiles []string) {
	t.Helper()
	for _, path := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		if path[len(path)-1] == '/' {
			err := os.MkdirAll(fullPath, 0755)
			require.NoError(t, err)
		} else {
			dir := filepath.Dir(fullPath)
			err := os.MkdirAll(dir, 0755)
			require.NoError(t, err)
			err = os.WriteFile(fullPath, []byte(path+": test content"), 0644)
			require.NoError(t, err)
		}
	}
}

func TestCoderPromptProviderSelection(t *testing.T) {
	tmpDir := t.TempDir()
	if _, err := config.Load(tmpDir, false); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	cfg := config.Get()
	cfg.WorkingDir = tmpDir

	openaiCompat := []models.ModelProvider{
		models.ProviderOpenAI,
		models.ProviderDeepSeek,
		models.ProviderGLM,
	}
	for _, p := range openaiCompat {
		out := CoderPrompt(p)
		if !strings.Contains(out, "OpenAI-compatible models") {
			t.Errorf("provider %q: expected OpenAI-compatible prompt marker, got:\n%s", p, out)
		}
		if strings.Contains(out, "built by OpenAI") {
			t.Errorf("provider %q: prompt still contains OpenAI-specific identity", p)
		}
	}
	// Anthropic 仍走 Anthropic 版提示词
	anthropic := CoderPrompt(models.ProviderAnthropic)
	if !strings.Contains(anthropic, "interactive CLI tool that helps users") {
		t.Errorf("anthropic: expected Anthropic prompt marker, got:\n%s", anthropic)
	}
	if strings.Contains(anthropic, "OpenAI-compatible models") {
		t.Errorf("anthropic: prompt wrongly contains OpenAI-compatible marker")
	}
}
