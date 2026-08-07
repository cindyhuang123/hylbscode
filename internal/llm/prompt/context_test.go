package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessContextPathsReadsExistingFileWithCaseTwin(t *testing.T) {
	// Create a temp dir with only the uppercase file present
	dir := t.TempDir()
	content := "project instructions here"
	if err := os.WriteFile(filepath.Join(dir, "HyLbsCode.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Mirror defaultContextPaths: both case variants present, only one exists
	paths := []string{"hylbscode.md", "HyLbsCode.md"}

	// Run repeatedly to surface the race (nonexistent twin must not win)
	for i := 0; i < 50; i++ {
		result := processContextPaths(dir, paths)
		if !strings.Contains(result, content) {
			t.Fatalf("iteration %d: expected context from HyLbsCode.md, got %q", i, result)
		}
	}
}
