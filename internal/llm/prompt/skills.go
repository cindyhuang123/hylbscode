package prompt

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/skills"
)

// skillsBaseDir 返回技能库所在目录（与内置 contextPaths 的相对路径语义一致，
// 即工作目录下的 skills/）。
func skillsBaseDir() string {
	return filepath.Join(config.WorkingDirectory(), "skills")
}

// BuildSkillIndex 扫描技能库，生成一份常驻的技能目录。完整技能内容不再
// 全部注入 prompt，而是由模型按需通过 read_skill 工具加载。
func BuildSkillIndex() string {
	list := skills.Scan(skillsBaseDir())
	if len(list) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("# 可用技能库（按需加载）\n")
	sb.WriteString("当用户请求命中下面某个技能的触发词时，先调用 read_skill 工具加载其完整指令，再按该技能的规范执行任务。\n")
	for _, s := range list {
		fmt.Fprintf(&sb, "- %s: %s\n", s.Name, s.Description)
	}
	return sb.String()
}
