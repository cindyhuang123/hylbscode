// Package skills 提供内置技能库（skills/<类别>/Skill.md）的扫描与元信息解析。
// Skill.md 以 YAML frontmatter 开头，包含 name 与 description（带触发词），
// 用于生成常驻的技能目录，以及按需加载技能全文。
package skills

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Skill 描述一个技能库条目。
type Skill struct {
	// Name 取自 frontmatter 的 name，缺失时回退为目录名。
	Name string
	// Description 取自 frontmatter 的 description（含触发词），用于技能目录。
	Description string
	// Path 是 Skill.md 的完整路径，供按需读取全文。
	Path string
}

// Scan 扫描 baseDir 下所有 <类别>/Skill.md，解析 frontmatter 并按名称排序。
// baseDir 不存在时返回空列表而非错误，方便没有技能库时静默跳过。
func Scan(baseDir string) []Skill {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil
	}

	result := make([]Skill, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillFile := filepath.Join(baseDir, entry.Name(), "Skill.md")
		content, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}
		fm, ok := parseFrontmatter(string(content))
		if !ok {
			continue
		}
		name := fm["name"]
		if name == "" {
			name = entry.Name()
		}
		result = append(result, Skill{
			Name:        name,
			Description: fm["description"],
			Path:        skillFile,
		})
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// Find 在扫描结果中按名称或所属目录名查找技能。
func Find(list []Skill, name string) (Skill, bool) {
	for _, s := range list {
		if s.Name == name || filepath.Base(filepath.Dir(s.Path)) == name {
			return s, true
		}
	}
	return Skill{}, false
}

// parseFrontmatter 解析 Skill.md 头部的 YAML frontmatter（--- 包裹的 key/value）。
// 仅支持简单键值对与 > 折叠多行块（description 使用该格式），返回 map。
func parseFrontmatter(content string) (map[string]string, bool) {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return nil, false
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return nil, false
	}

	fm := make(map[string]string)
	var blockKey string
	var blockLines []string
	for i := 1; i < end; i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if blockKey != "" {
			if trimmed == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				if trimmed != "" {
					blockLines = append(blockLines, trimmed)
				}
				continue
			}
			// 折叠块结束，转入新的键值对
			fm[blockKey] = strings.Join(blockLines, " ")
			blockKey = ""
			blockLines = nil
		}
		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if val == ">" || val == "|" {
			blockKey = key
			blockLines = nil
			continue
		}
		fm[key] = strings.Trim(val, `"'`)
	}
	if blockKey != "" {
		fm[blockKey] = strings.Join(blockLines, " ")
	}
	return fm, true
}
