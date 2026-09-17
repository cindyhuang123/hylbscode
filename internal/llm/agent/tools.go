package agent

import (
	"context"

	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/history"
	"github.com/cindyhuang123/hylbscode/internal/llm/tools"
	"github.com/cindyhuang123/hylbscode/internal/lsp"
	"github.com/cindyhuang123/hylbscode/internal/message"
	"github.com/cindyhuang123/hylbscode/internal/permission"
	"github.com/cindyhuang123/hylbscode/internal/search"
	"github.com/cindyhuang123/hylbscode/internal/session"
	"github.com/cindyhuang123/hylbscode/internal/todo"
)

func CoderAgentTools(
	permissions permission.Service,
	sessions session.Service,
	messages message.Service,
	history history.Service,
	search search.Service,
	todos todo.Service,
	lspClients map[string]*lsp.Client,
) []tools.BaseTool {
	ctx := context.Background()
	otherTools := GetMcpTools(ctx, permissions)
	if len(lspClients) > 0 {
		otherTools = append(otherTools, tools.NewDiagnosticsTool(lspClients))
	}
	core := []tools.BaseTool{
		tools.NewBashTool(permissions),
		tools.NewEditTool(lspClients, permissions, history),
		tools.NewGitTool(permissions),
		tools.NewGlobTool(),
		tools.NewGrepTool(),
		tools.NewHistoryTool(history),
		tools.NewLsTool(),
		tools.NewSearchHistoryTool(search),
		tools.NewTodoTool(todos),
		tools.NewViewTool(lspClients),
		tools.NewPatchTool(lspClients, permissions, history),
		tools.NewReadSkillTool(),
		tools.NewWriteTool(lspClients, permissions, history),
		NewAgentTool(sessions, messages, lspClients),
	}
	core = append(core, optionalTools(permissions)...)
	return append(core, otherTools...)
}

// optionalTools returns tools that are omitted from every request unless the
// user opts in via config "tools.enabled" (their schema costs prompt tokens).
func optionalTools(permissions permission.Service) []tools.BaseTool {
	cfg := config.Get()
	enabled := make(map[string]bool, len(cfg.Tools.Enabled))
	for _, name := range cfg.Tools.Enabled {
		enabled[name] = true
	}
	var out []tools.BaseTool
	if enabled[tools.SourcegraphToolName] {
		out = append(out, tools.NewSourcegraphTool())
	}
	if enabled[tools.FetchToolName] {
		out = append(out, tools.NewFetchTool(permissions))
	}
	return out
}

func TaskAgentTools(lspClients map[string]*lsp.Client) []tools.BaseTool {
	cfg := config.Get()
	sourcegraphEnabled := false
	for _, name := range cfg.Tools.Enabled {
		if name == tools.SourcegraphToolName {
			sourcegraphEnabled = true
			break
		}
	}
	result := []tools.BaseTool{
		tools.NewGlobTool(),
		tools.NewGrepTool(),
		tools.NewLsTool(),
		tools.NewReadSkillTool(),
		tools.NewViewTool(lspClients),
	}
	if sourcegraphEnabled {
		result = append(result, tools.NewSourcegraphTool())
	}
	return result
}
