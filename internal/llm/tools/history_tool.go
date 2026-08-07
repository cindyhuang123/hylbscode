package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/cindyhuang123/hylbscode/internal/history"
)

type HistoryParams struct {
	// Path filters history to a specific file. Empty returns all files touched this session.
	Path string `json:"path,omitempty"`
	// Limit caps how many versions are shown per file (default 10).
	Limit int `json:"limit,omitempty"`
}

type historyTool struct {
	history history.Service
}

const (
	HistoryToolName = "history"
	// historyDescription explains how the AI can inspect file versions tracked for the current session.
	historyDescription = `Lists files modified during the current session and shows their recorded versions.

WHEN TO USE THIS TOOL:
- Use to recall which files the assistant has changed in this session
- Use to inspect earlier versions of a file when a mistake was made and you want to see or recover prior content
- Helpful before undoing an edit to confirm what the previous version looked like

HOW TO USE:
- Call with no arguments to list all files touched in this session (returns path + version count)
- Call with "path" set to view the version list (id, version) for that file
- Optionally set "limit" to cap how many versions are shown (default 10, max 50)

LIMITATIONS:
- History is scoped to the CURRENT session only; it does not span sessions
- Content is not returned in bulk by default. Use the path form and review the returned version ids/content carefully
- Records are created when the assistant edits/writes files during this session

TIPS:
- After a bad edit, list the file's versions to find the one before the change
- If a version looks right, tell the user and use edit/write to restore its content`
)

func NewHistoryTool(h history.Service) BaseTool {
	return &historyTool{history: h}
}

func (ht *historyTool) Info() ToolInfo {
	return ToolInfo{
		Name:        HistoryToolName,
		Description: historyDescription,
		Parameters: map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Optional file path. When set, lists that file's history; when empty, lists all files modified this session.",
			},
			"limit": map[string]any{
				"type":        "number",
				"description": "Maximum number of versions to show per file (default 10, max 50)",
			},
		},
	}
}

func (ht *historyTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params HistoryParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("error parsing parameters: " + err.Error()), nil
	}

	sessionID, _ := ctx.Value(SessionIDContextKey).(string)
	if sessionID == "" {
		return NewTextErrorResponse("no current session available"), nil
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	if params.Path != "" {
		return ht.renderFileHistory(ctx, sessionID, params.Path, limit)
	}
	return ht.renderSessionFiles(ctx, sessionID)
}

// renderSessionFiles lists every file modified in the session.
func (ht *historyTool) renderSessionFiles(ctx context.Context, sessionID string) (ToolResponse, error) {
	files, err := ht.history.ListLatestSessionFiles(ctx, sessionID)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("error listing session files: %w", err)
	}
	if len(files) == 0 {
		return NewTextResponse("No files have been modified in this session yet."), nil
	}

	// Deduplicate by path, tracking the number of versions recorded.
	byPath := map[string]map[string]bool{}
	var paths []string
	for _, f := range files {
		if byPath[f.Path] == nil {
			byPath[f.Path] = map[string]bool{}
			paths = append(paths, f.Path)
		}
		byPath[f.Path][f.Version] = true
	}
	sort.Strings(paths)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Files modified this session (%d):\n", len(paths)))
	for _, p := range paths {
		b.WriteString(fmt.Sprintf("- %s (%d version(s))\n", p, len(byPath[p])))
	}
	b.WriteString("\nUse the \"path\" parameter to inspect a specific file's versions.")
	return NewTextResponse(b.String()), nil
}

// renderFileHistory lists versions of a single file in this session.
func (ht *historyTool) renderFileHistory(ctx context.Context, sessionID, path string, limit int) (ToolResponse, error) {
	files, err := ht.history.ListBySession(ctx, sessionID)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("error listing session files: %w", err)
	}

	var matched []history.File
	for _, f := range files {
		if f.Path == path {
			matched = append(matched, f)
		}
	}
	if len(matched) == 0 {
		return NewTextErrorResponse(fmt.Sprintf("No history found for %q in the current session.", path)), nil
	}

	// Newest first.
	sort.Slice(matched, func(i, j int) bool { return matched[i].UpdatedAt > matched[j].UpdatedAt })
	if len(matched) > limit {
		matched = matched[:limit]
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("History for %s (%d shown):\n", path, len(matched)))
	for _, f := range matched {
		b.WriteString(fmt.Sprintf("- version %s (id: %s)\n", f.Version, f.ID))
	}
	b.WriteString("\nTo restore a version, tell the user the id and use edit/write with the prior content.")
	return NewTextResponse(b.String()), nil
}
