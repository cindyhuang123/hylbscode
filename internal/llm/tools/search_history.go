package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cindyhuang123/hylbscode/internal/search"
)

type SearchHistoryParams struct {
	Query     string `json:"query"`
	SessionID string `json:"session_id"`
	Limit     int    `json:"limit"`
}

type searchHistoryTool struct {
	search search.Service
}

const (
	SearchHistoryToolName    = "search_history"
	searchHistoryDescription = `Search tool that finds text in past conversation messages, returning matching message snippets.

WHEN TO USE THIS TOOL:
- Use when you need details from earlier in the conversation that may have been summarized or compacted
- Great for recalling decisions, requirements, file paths, or prices discussed previously

HOW TO USE:
- Provide a search query with the text you want to find
- By default searches the current session; set session_id to search another session
- Optionally set limit (default 5, max 20)

LIMITATIONS:
- Only searches stored message content, not file contents (use Grep for files)
- Results are truncated snippets; use the session switch dialog to view full messages`
)

func NewSearchHistoryTool(search search.Service) BaseTool {
	return &searchHistoryTool{search: search}
}

func (t *searchHistoryTool) Info() ToolInfo {
	return ToolInfo{
		Name:        SearchHistoryToolName,
		Description: searchHistoryDescription,
		Parameters: map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The text to search for in past conversation messages",
			},
			"session_id": map[string]any{
				"type":        "string",
				"description": "Session to search. Defaults to the current session.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results to return (default 5, max 20)",
			},
		},
		Required: []string{"query"},
	}
}

func (t *searchHistoryTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params SearchHistoryParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}
	if params.Query == "" {
		return NewTextErrorResponse("query is required"), nil
	}
	sessionID := params.SessionID
	if sessionID == "" {
		if sid, ok := ctx.Value(SessionIDContextKey).(string); ok {
			sessionID = sid
		}
	}
	if sessionID == "" {
		return NewTextErrorResponse("no session specified and no current session available"), nil
	}

	results, err := t.search.Search(ctx, sessionID, params.Query, params.Limit)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("error searching history: %w", err)
	}

	if len(results) == 0 {
		return NewTextResponse("No matching messages found"), nil
	}

	var output string
	output = fmt.Sprintf("Found %d matching messages\n", len(results))
	for _, r := range results {
		ts := time.Unix(r.CreatedAt, 0).Format("2006-01-02 15:04")
		output += fmt.Sprintf("\n[%s] %s\n%s\n", ts, r.Role, r.Snippet)
	}

	return NewTextResponse(output), nil
}
