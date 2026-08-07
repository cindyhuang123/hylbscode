package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/llm/tools/shell"
	"github.com/cindyhuang123/hylbscode/internal/permission"
)

type GitParams struct {
	// Args is the raw git subcommand and its arguments, e.g. ["status"], ["diff", "--stat"], ["log", "-5"].
	Args []string `json:"args"`
	// WorkingDirectory overrides where git runs. Defaults to the current project directory.
	WorkingDirectory string `json:"working_directory,omitempty"`
}

type gitTool struct {
	permissions permission.Service
}

const (
	GitToolName = "git"
	gitTimeout  = 60000
)

// gitDescription explains when to use the git tool and lists the allowed
// subcommands. Read-only commands run without extra confirmation; commands
// that modify state (commit, reset, clean, push, etc.) require permission.
const gitDescription = `Runs git commands against the current project, returning the output to help inspect or modify repository state.

WHEN TO USE THIS TOOL:
- Use instead of ` + "`bash`" + ` whenever the task is purely about git (status, diff, log, add, commit, stash, etc.)
- Helps track working-tree changes, review diffs, understand history, and manage commits
- Prefer this over navigating with General purpose search tools for git-specific queries

HOW TO USE:
- Provide the full git command as an argument list, WITHOUT the leading "git", e.g.:
  - ["status"] -> git status
  - ["diff", "--stat"] -> git diff --stat
  - ["log", "--oneline", "-5"] -> git log --oneline -5
  - ["add", "file.go"] -> git add file.go
  - ["commit", "-m", "fix: thing"] -> git commit -m "fix: thing"
- Optional field working_directory overrides where git runs (defaults to the project directory)

ALLOWED SUBCOMMANDS READ-ONLY (run without confirmation):
- status, diff, log, show, branch, ls-files, rev-parse, remote, tag, stash, config --get, blame
- If you only need to inspect state, use one of these first.

MUTATING SUBCOMMANDS (require user approval):
- add, commit, reset, clean, checkout, restore, mv, merge, rebase, cherry-pick, revert, push, pull, stash, config, reset-hard equivalents, etc.

LIMITATIONS:
- Do NOT push to the remote unless the user explicitly asks.
- Never use interactive flags (e.g. rebase -i) — they are not supported.
- The tool runs git in a persistent shell; commands must be non-interactive and must not require a pager. Prefer --no-pager / --no-color for large output.

TIPS:
- For commits: run status, diff, and log first, then commit, and verify with status.
- Human-readable, concise output is usually best (e.g. git status -sb, git diff --stat).`

func NewGitTool(permission permission.Service) BaseTool {
	return &gitTool{
		permissions: permission,
	}
}

// readOnlySubcommands run without a permission prompt. Everything else requires approval.
var readOnlySubcommands = map[string]bool{
	"status": true, "diff": true, "log": true, "show": true, "branch": true,
	"ls-files": true, "rev-parse": true, "remote": true, "tag": true,
	"stash": true, "blame": true, "shortlog": true, "describe": true, "name-rev": true,
}

func (g *gitTool) Info() ToolInfo {
	return ToolInfo{
		Name:        GitToolName,
		Description: gitDescription,
		Parameters: map[string]any{
			"args": map[string]any{
				"type":        "array",
				"description": "The git subcommand and its arguments, without the leading 'git'. e.g. [\"status\", \"-sb\"]",
				"items":       map[string]any{"type": "string"},
			},
			"working_directory": map[string]any{
				"type":        "string",
				"description": "Directory where git runs. Defaults to the current project directory.",
			},
		},
		Required: []string{"args"},
	}
}

func (g *gitTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params GitParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("error parsing parameters: " + err.Error()), nil
	}

	if len(params.Args) == 0 || strings.TrimSpace(params.Args[0]) == "" {
		return NewTextErrorResponse("args is required: provide the git subcommand and its arguments"), nil
	}

	// Build a safe command string. Args come from the model; escape each to
	// avoid shell injection since we run through the persistent shell.
	escaped := make([]string, len(params.Args))
	for i, a := range params.Args {
		escaped[i] = shellQuote(a)
	}
	command := "git " + strings.Join(escaped, " ")

	// Classify as read-only vs mutating to decide permission gating.
	sub := params.Args[0]
	isReadOnly := readOnlySubcommands[sub]

	workDir := params.WorkingDirectory
	if workDir == "" {
		workDir = config.WorkingDirectory()
	}

	if !isReadOnly {
		sessionID, messageID := GetContextValues(ctx)
		if sessionID == "" || messageID == "" {
			return ToolResponse{}, fmt.Errorf("session ID and message ID are required")
		}
		p := g.permissions.Request(
			permission.CreatePermissionRequest{
				SessionID:   sessionID,
				Path:        workDir,
				ToolName:    GitToolName,
				Action:      "git " + sub,
				Description: "Run git command: " + command,
				Params:      GitParams(params),
			},
		)
		if !p {
			return ToolResponse{}, permission.ErrorPermissionDenied
		}
	}

	sh := shell.GetPersistentShell(workDir)
	stdout, stderr, exitCode, interrupted, err := sh.Exec(ctx, command+" --no-pager --no-color", gitTimeout)
	if err != nil {
		return ToolResponse{}, fmt.Errorf("error executing git: %w", err)
	}

	stdout = truncateOutput(stdout)
	stderr = truncateOutput(stderr)

	msg := stdout
	if interrupted {
		msg = "Command was aborted before completion"
	}
	if exitCode != 0 {
		if msg != "" {
			msg += "\n"
		}
		msg += strings.TrimSpace(stderr)
		if msg == "" {
			msg += fmt.Sprintf("git exited with code %d", exitCode)
		} else {
			msg += fmt.Sprintf("\nexit code %d", exitCode)
		}
		// Return errors as an error response so the model sees them clearly.
		return NewTextErrorResponse(msg), nil
	}

	if strings.TrimSpace(msg) == "" {
		msg = "no output"
	}
	return NewTextResponse(msg), nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
