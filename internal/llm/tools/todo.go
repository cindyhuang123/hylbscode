package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cindyhuang123/hylbscode/internal/todo"
)

type TodoItem struct {
	Content  string `json:"content"`
	Status   string `json:"status"`
	Priority int64  `json:"priority"`
}

type TodoToolParams struct {
	Todos []TodoItem `json:"todos"`
}

type todoTool struct {
	todos todo.Service
}

const (
	TodoToolName    = "todo"
	todoDescription = `Tool that manages the task list for the current session.

WHEN TO USE THIS TOOL:
- Use when starting a multi-step task that benefits from tracking progress
- Use to update the list as work progresses (mark items complete, adjust plan)

HOW TO USE:
- Always pass the FULL list of todos as the "todos" parameter - this is a snapshot-based tool
- The server replaces the entire list with what you send; items you omit are deleted
- Each item: content (required), status ("pending", "in_progress", or "completed"), priority (0-3, higher = more important)
- Reorder items by changing their position in the array
- Call again whenever the task list changes; keep it current throughout the task

LIMITATIONS:
- The list is scoped to the current session
- Keep items concise and actionable; one task per item`
)

func NewTodoTool(todos todo.Service) BaseTool {
	return &todoTool{todos: todos}
}

func (t *todoTool) Info() ToolInfo {
	return ToolInfo{
		Name:        TodoToolName,
		Description: todoDescription,
		Parameters: map[string]any{
			"todos": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"content": map[string]any{
							"type":        "string",
							"description": "Task description",
						},
						"status": map[string]any{
							"type":        "string",
							"description": "Task status: pending, in_progress, or completed",
						},
						"priority": map[string]any{
							"type":        "integer",
							"description": "Priority 0-3, higher is more important",
						},
					},
					"required": []string{"content"},
				},
				"description": "The complete list of todos (snapshot-based; omitted items are removed)",
			},
		},
		Required: []string{"todos"},
	}
}

func (t *todoTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params TodoToolParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}

	sessionID, _ := ctx.Value(SessionIDContextKey).(string)
	if sessionID == "" {
		return NewTextErrorResponse("no current session available"), nil
	}

	items := make([]todo.Todo, 0, len(params.Todos))
	for _, p := range params.Todos {
		content := strings.TrimSpace(p.Content)
		if content == "" {
			continue
		}
		status := p.Status
		if status == "" {
			status = "pending"
		}
		items = append(items, todo.Todo{
			Content:  content,
			Status:   status,
			Priority: p.Priority,
		})
	}

	if _, err := t.todos.BulkSet(ctx, sessionID, items); err != nil {
		return ToolResponse{}, fmt.Errorf("error saving todos: %w", err)
	}

	return NewTextResponse(formatTodoList(items)), nil
}

func formatTodoList(items []todo.Todo) string {
	if len(items) == 0 {
		return "Todo list is now empty."
	}
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Todo list updated (%d items):\n", len(items)))
	for i, item := range items {
		mark := " "
		switch item.Status {
		case "completed":
			mark = "x"
		case "in_progress":
			mark = ">"
		}
		output.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, mark, item.Content))
	}
	return output.String()
}
