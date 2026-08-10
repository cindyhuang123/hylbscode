package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFrontmatterFoldedDescription(t *testing.T) {
	content := `---
name: coding-workflow
description: >
  处理任何代码编写、重构、Debug、代码审查任务时套用本技能。
  触发词: "写代码", "修 bug", "重构", "code", "review"。
---

# 标题
正文`
	fm, ok := parseFrontmatter(content)
	if !ok {
		t.Fatal("expected frontmatter to parse")
	}
	if fm["name"] != "coding-workflow" {
		t.Errorf("name = %q, want coding-workflow", fm["name"])
	}
	want := `处理任何代码编写、重构、Debug、代码审查任务时套用本技能。 触发词: "写代码", "修 bug", "重构", "code", "review"。`
	if fm["description"] != want {
		t.Errorf("description = %q, want %q", fm["description"], want)
	}
}

func TestParseFrontmatterNoBlock(t *testing.T) {
	fm, ok := parseFrontmatter("---\nname: simple\nfoo: bar\n---\nbody")
	if !ok {
		t.Fatal("expected frontmatter to parse")
	}
	if fm["name"] != "simple" || fm["foo"] != "bar" {
		t.Errorf("got %#v", fm)
	}
}

func TestParseFrontmatterMalformed(t *testing.T) {
	if _, ok := parseFrontmatter("no frontmatter here"); ok {
		t.Fatal("expected no frontmatter")
	}
	if _, ok := parseFrontmatter("---\nnever closed"); ok {
		t.Fatal("expected no frontmatter for unterminated block")
	}
}

func TestScanAndFind(t *testing.T) {
	base := t.TempDir()
	mustWrite(t, filepath.Join(base, "coding", "Skill.md"), `---
name: coding-workflow
description: 处理编码任务。
---
# 正文`)
	mustWrite(t, filepath.Join(base, "planning", "Skill.md"), `---
name: planning-skill
description: 制定计划。
---
# 正文`)
	// 目录存在但缺 Skill.md，应被跳过
	if err := os.MkdirAll(filepath.Join(base, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	list := Scan(base)
	if len(list) != 2 {
		t.Fatalf("Scan returned %d skills, want 2", len(list))
	}
	// 按名称排序：coding-workflow < planning-skill
	if list[0].Name != "coding-workflow" {
		t.Errorf("list[0].Name = %q", list[0].Name)
	}

	if _, ok := Find(list, "coding-workflow"); !ok {
		t.Error("Find by name failed")
	}
	if _, ok := Find(list, "planning"); !ok {
		t.Error("Find by directory name failed")
	}
	if _, ok := Find(list, "nope"); ok {
		t.Error("Find returned ok for unknown name")
	}
}

func TestScanMissingDir(t *testing.T) {
	if got := Scan(filepath.Join(t.TempDir(), "no-such-dir")); got != nil {
		t.Errorf("Scan of missing dir = %v, want nil", got)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
