package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/skills"
)

type ReadSkillParams struct {
	Name string `json:"name"`
}

const (
	ReadSkillToolName = "read_skill"

	readSkillDescription = `按需加载技能库的完整指令，返回该技能的完整工作流/规范内容。

WHEN TO USE THIS TOOL:
- 常驻系统提示中的"可用技能库"列出了所有技能及其触发词
- 当用户请求命中某技能的触发词（如"写代码"、"制定计划"、"写会议纪要"）时
- 加载技能后再按其中规范执行任务，可显著提升输出质量与一致性

HOW TO USE:
- 传入技能名称（name 字段，即技能目录中列出的名称）

LIMITATIONS:
- 一次只加载一个技能
- 技能内容仅供当前任务参考，不会修改系统提示`
)

type readSkillTool struct{}

func NewReadSkillTool() BaseTool {
	return &readSkillTool{}
}

func (r *readSkillTool) Info() ToolInfo {
	return ToolInfo{
		Name:        ReadSkillToolName,
		Description: readSkillDescription,
		Parameters: map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "要加载的技能名称",
			},
		},
		Required: []string{"name"},
	}
}

func (r *readSkillTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params ReadSkillParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}
	if params.Name == "" {
		return NewTextErrorResponse("missing required parameter: name"), nil
	}

	baseDir := filepath.Join(config.WorkingDirectory(), "skills")
	list := skills.Scan(baseDir)
	skill, ok := skills.Find(list, params.Name)
	if !ok {
		names := make([]string, 0, len(list))
		for _, s := range list {
			names = append(names, s.Name)
		}
		return NewTextErrorResponse(fmt.Sprintf("技能 %q 不存在。可用技能: %s", params.Name, strings.Join(names, ", "))), nil
	}

	content, err := os.ReadFile(skill.Path)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("failed to read skill %q: %v", skill.Name, err)), nil
	}
	return NewTextResponse("# From:" + skill.Path + "\n" + string(content)), nil
}
