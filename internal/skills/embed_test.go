package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDefaultGenerates(t *testing.T) {
	base := filepath.Join(t.TempDir(), "skills")
	if err := EnsureDefault(base); err != nil {
		t.Fatalf("EnsureDefault: %v", err)
	}
	// 应生成内置的全部类别
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 10 {
		t.Errorf("generated %d categories, want >= 10", len(entries))
	}
	// 每个类别应有可解析的 Skill.md
	list := Scan(base)
	if len(list) != len(entries) {
		t.Errorf("Scan found %d skills, want %d", len(list), len(entries))
	}
	// 再次调用应幂等
	if err := EnsureDefault(base); err != nil {
		t.Fatalf("EnsureDefault second call: %v", err)
	}
}

func TestEnsureDefaultKeepsExisting(t *testing.T) {
	base := filepath.Join(t.TempDir(), "skills")
	// 用户自定义目录：只有一个自定义类别
	custom := filepath.Join(base, "my-custom")
	if err := os.MkdirAll(custom, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(custom, "Skill.md"), []byte("---\nname: custom-skill\ndescription: 用户自定义。\n---\n正文"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureDefault(base); err != nil {
		t.Fatalf("EnsureDefault: %v", err)
	}
	// 已存在则不应生成内置技能，也不应动自定义内容
	entries, _ := os.ReadDir(base)
	if len(entries) != 1 {
		t.Fatalf("existing dir was modified: got %d entries, want 1", len(entries))
	}
	if _, err := os.Stat(filepath.Join(custom, "Skill.md")); err != nil {
		t.Errorf("custom skill was removed: %v", err)
	}
}
