package agent

import (
	"context"

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
	return append(
		[]tools.BaseTool{
			tools.NewBashTool(permissions),
			tools.NewEditTool(lspClients, permissions, history),
			tools.NewFetchTool(permissions),
			tools.NewGitTool(permissions),
			tools.NewGlobTool(),
			tools.NewGrepTool(),
			tools.NewHistoryTool(history),
			tools.NewLsTool(),
			tools.NewSearchHistoryTool(search),
			tools.NewSourcegraphTool(),
			tools.NewTodoTool(todos),
			tools.NewViewTool(lspClients),
			tools.NewPatchTool(lspClients, permissions, history),
			tools.NewWriteTool(lspClients, permissions, history),
			NewAgentTool(sessions, messages, lspClients),
		}, otherTools...,
	)
}

func TaskAgentTools(lspClients map[string]*lsp.Client) []tools.BaseTool {
	return []tools.BaseTool{
		tools.NewGlobTool(),
		tools.NewGrepTool(),
		tools.NewLsTool(),
		tools.NewSourcegraphTool(),
		tools.NewViewTool(lspClients),
	}
}
