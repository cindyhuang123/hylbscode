package tools

import (
	"context"
	"time"
)

const TimeToolName = "time"

type timeTool struct{}

func NewTimeTool() BaseTool {
	return &timeTool{}
}

func (t *timeTool) Info() ToolInfo {
	return ToolInfo{
		Name:        TimeToolName,
		Description: "Get the current local date, time, and timezone in RFC3339 format with day-of-week. Use this whenever you need the current time for logging, scheduling, timestamps, or to know today's date.",
		Parameters:  map[string]any{},
	}
}

func (t *timeTool) Run(_ context.Context, _ ToolCall) (ToolResponse, error) {
	now := time.Now()
	return NewTextResponse(now.Format("2006-01-02 15:04:05 MST Monday")), nil
}
