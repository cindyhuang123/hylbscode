package skills

import (
	"embed"
	"os"
	"path/filepath"
)

//go:embed data
var embeddedData embed.FS

// EnsureDefault 确保工作目录下存在技能库：
// - baseDir 已存在：跳过，保留用户自定义或已有的技能，不做任何改动；
// - baseDir 不存在：用内置模板（internal/skills/data）生成完整的技能库。
// 返回 nil 表示技能库已就绪（存在或生成成功）。
func EnsureDefault(baseDir string) error {
	if _, err := os.Stat(baseDir); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	entries, err := embeddedData.ReadDir("data")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		content, err := embeddedData.ReadFile(filepath.ToSlash(filepath.Join("data", entry.Name(), "Skill.md")))
		if err != nil {
			continue
		}
		catDir := filepath.Join(baseDir, entry.Name())
		if err := os.MkdirAll(catDir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(catDir, "Skill.md"), content, 0o644); err != nil {
			return err
		}
	}
	return nil
}
