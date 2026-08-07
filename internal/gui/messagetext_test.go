package gui

import (
	"strings"
	"testing"

	"github.com/cindyhuang123/hylbscode/internal/message"
)

func TestMessageTextConcatenatesTextParts(t *testing.T) {
	m := message.Message{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.TextContent{Text: "Hello "},
			message.TextContent{Text: "world"},
		},
	}
	if got := messageText(m); got != "Hello world" {
		t.Fatalf("expected 'Hello world', got %q", got)
	}
}

func TestMessageTextIncludesToolResult(t *testing.T) {
	m := message.Message{
		Role: message.Tool,
		Parts: []message.ContentPart{
			message.ToolResult{Name: "bash", Content: "done", IsError: true},
		},
	}
	got := messageText(m)
	for _, want := range []string{"bash", "error", "done"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in tool message rendering, got %q", want, got)
		}
	}
}

func TestMessageTextIncludesReasoning(t *testing.T) {
	m := message.Message{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.ReasoningContent{Thinking: "step one"},
			message.TextContent{Text: "answer"},
		},
	}
	got := messageText(m)
	if !strings.Contains(got, "step one") || !strings.Contains(got, "answer") {
		t.Fatalf("unexpected rendering with reasoning part: %q", got)
	}
}

func TestRoleLabel(t *testing.T) {
	cases := map[message.MessageRole]string{
		message.User:      "You",
		message.Assistant: "Assistant",
		message.Tool:      "Tool",
		message.System:    "System",
	}
	for role, want := range cases {
		if got := roleLabel(role); got != want {
			t.Fatalf("roleLabel(%s) = %q, want %q", role, got, want)
		}
	}
}
