package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/permission"
)

type ToolInfo struct {
	Name        string
	Description string
	Parameters  map[string]any
	Required    []string
}

type toolResponseType string

type (
	sessionIDContextKey string
	messageIDContextKey string
	streamCallbackKey   string
)

const (
	ToolResponseTypeText  toolResponseType = "text"
	ToolResponseTypeImage toolResponseType = "image"

	SessionIDContextKey sessionIDContextKey = "session_id"
	MessageIDContextKey messageIDContextKey = "message_id"
	StreamCallbackKey   streamCallbackKey   = "stream_callback"
)

type ToolResponse struct {
	Type     toolResponseType `json:"type"`
	Content  string           `json:"content"`
	Metadata string           `json:"metadata,omitempty"`
	IsError  bool             `json:"is_error"`
}

func NewTextResponse(content string) ToolResponse {
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: content,
	}
}

func WithResponseMetadata(response ToolResponse, metadata any) ToolResponse {
	if metadata != nil {
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return response
		}
		response.Metadata = string(metadataBytes)
	}
	return response
}

func NewTextErrorResponse(content string) ToolResponse {
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: content,
		IsError: true,
	}
}

type ToolCall struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input string `json:"input"`
}

type BaseTool interface {
	Info() ToolInfo
	Run(ctx context.Context, params ToolCall) (ToolResponse, error)
}

func GetContextValues(ctx context.Context) (string, string) {
	sessionID := ctx.Value(SessionIDContextKey)
	messageID := ctx.Value(MessageIDContextKey)
	if sessionID == nil {
		return "", ""
	}
	if messageID == nil {
		return sessionID.(string), ""
	}
	return sessionID.(string), messageID.(string)
}

// StreamOutputFunc receives incremental output while a tool executes.
type StreamOutputFunc func(chunk string)

// GetStreamCallback returns the streaming callback set in the context, if any.
func GetStreamCallback(ctx context.Context) StreamOutputFunc {
	if cb, ok := ctx.Value(StreamCallbackKey).(StreamOutputFunc); ok {
		return cb
	}
	return nil
}

// workingDirectory returns the configured working directory, falling back to
// the process working directory when the config is not loaded (unit tests).
func workingDirectory() string {
	if config.IsLoaded() {
		return config.WorkingDirectory()
	}
	wd, _ := os.Getwd()
	return wd
}

// confirmPathAccess asks the user for permission when the given absolute path
// lies outside the working directory and /tmp. It returns (true, nil) when the
// access is allowed without prompting or granted by the user; (false, nil)
// when denied; or an error when the permission service is unavailable.
func confirmPathAccess(ctx context.Context, perms permission.Service, toolName, action, absPath string) (bool, error) {
	if inWorkingDir(absPath) {
		return true, nil
	}
	if perms == nil {
		return false, nil
	}
	sessionID, messageID := GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return false, nil
	}
	dir := filepath.Dir(absPath)
	p := perms.Request(permission.CreatePermissionRequest{
		SessionID:   sessionID,
		Path:        dir,
		ToolName:    toolName,
		Action:      action,
		Description: "Access path outside working directory: " + absPath,
		Params: EditPermissionsParams{
			FilePath: absPath,
		},
	})
	return p, nil
}
