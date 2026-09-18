package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/llm/tools"
	"github.com/cindyhuang123/hylbscode/internal/lsp"
	"github.com/cindyhuang123/hylbscode/internal/message"
	"github.com/cindyhuang123/hylbscode/internal/permission"
	"github.com/cindyhuang123/hylbscode/internal/session"
)

type agentTool struct {
	sessions    session.Service
	messages    message.Service
	lspClients  map[string]*lsp.Client
	permissions permission.Service
}

const (
	AgentToolName = "agent"
)

type AgentParams struct {
	Prompt string `json:"prompt"`
}

func (b *agentTool) Info() tools.ToolInfo {
	return tools.ToolInfo{
		Name:        AgentToolName,
		Description: "Launch stateless sub-agents (tools: Glob, Grep, Ls, View) to search the codebase when a first query may miss the match. Run multiple agents concurrently in one message. Each returns a single final report, which the user does not see, so summarize it in your reply. Agents cannot run commands or modify files.",
		Parameters: map[string]any{
			"prompt": map[string]any{
				"type":        "string",
				"description": "The task for the agent to perform",
			},
		},
		Required: []string{"prompt"},
	}
}

func (b *agentTool) Run(ctx context.Context, call tools.ToolCall) (tools.ToolResponse, error) {
	var params AgentParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return tools.NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}
	if params.Prompt == "" {
		return tools.NewTextErrorResponse("prompt is required"), nil
	}

	sessionID, messageID := tools.GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return tools.ToolResponse{}, fmt.Errorf("session_id and message_id are required")
	}

	agent, err := NewAgent(config.AgentTask, b.sessions, b.messages, TaskAgentTools(b.lspClients, b.permissions))
	if err != nil {
		return tools.ToolResponse{}, fmt.Errorf("error creating agent: %s", err)
	}

	session, err := b.sessions.CreateTaskSession(ctx, call.ID, sessionID, "New Agent Session")
	if err != nil {
		return tools.ToolResponse{}, fmt.Errorf("error creating session: %s", err)
	}

	done, err := agent.Run(ctx, session.ID, params.Prompt)
	if err != nil {
		return tools.ToolResponse{}, fmt.Errorf("error generating agent: %s", err)
	}
	result := <-done
	if result.Error != nil {
		return tools.ToolResponse{}, fmt.Errorf("error generating agent: %s", result.Error)
	}

	response := result.Message
	if response.Role != message.Assistant {
		return tools.NewTextErrorResponse("no response"), nil
	}

	updatedSession, err := b.sessions.Get(ctx, session.ID)
	if err != nil {
		return tools.ToolResponse{}, fmt.Errorf("error getting session: %s", err)
	}
	parentSession, err := b.sessions.Get(ctx, sessionID)
	if err != nil {
		return tools.ToolResponse{}, fmt.Errorf("error getting parent session: %s", err)
	}

	parentSession.Cost += updatedSession.Cost

	_, err = b.sessions.Save(ctx, parentSession)
	if err != nil {
		return tools.ToolResponse{}, fmt.Errorf("error saving parent session: %s", err)
	}
	return tools.NewTextResponse(response.Content().String()), nil
}

func NewAgentTool(
	Sessions session.Service,
	Messages message.Service,
	LspClients map[string]*lsp.Client,
	Permissions permission.Service,
) tools.BaseTool {
	return &agentTool{
		sessions:    Sessions,
		messages:    Messages,
		lspClients:  LspClients,
		permissions: Permissions,
	}
}
