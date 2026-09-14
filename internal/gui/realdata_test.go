package gui

import (
	"context"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/cindyhuang123/hylbscode/internal/app"
	"github.com/cindyhuang123/hylbscode/internal/message"
)

// TestRenderRealPersistedToolCall uses the exact shape of a persisted
// assistant message from the live database (bash ToolCall whose input is the
// {command: ...} JSON) and verifies the rendered block shows the command.
func TestRenderRealPersistedToolCall(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.ToolCall{
				ID:       "call_00_rqZWxlah3NRxLCsk2vna6593",
				Name:     "bash",
				Input:    `{"command": "cd /home/cindy/TMP/lbs_code_dir/20260914_1646 \u0026\u0026 find . -type d -not -path \"./.hylbscode*\" | sort"}`,
				Type:     "function",
				Finished: true,
			},
		},
	}
	_, used := renderMessage(m, map[string]*ToolBlock{}, nil, false)
	block, ok := used["call_00_rqZWxlah3NRxLCsk2vna6593"]
	if !ok {
		t.Fatal("expected block registered for the persisted call id")
	}
	if block.TitleText() != "bash" {
		t.Fatalf("expected tool name title, got %q", block.TitleText())
	}
	want := "find . -type d -not -path"
	if !strings.Contains(block.output.Text, want) {
		t.Fatalf("expected rendered output to contain the command %q, got %q", want, block.output.Text)
	}
}

// TestRenderNowShowsPersistedCommand drives the real renderNow pipeline with
// the two messages that the live database actually stored for a bash call
// (assistant ToolCall with a {command: ...} input + tool result), and checks
// the rendered tool blocks display the command text.
func TestRenderNowShowsPersistedCommand(t *testing.T) {
	test.NewApp()
	svc := &fakeMessageService{msgs: []message.Message{
		{ID: "m1", SessionID: "s1", Role: message.Assistant, Parts: []message.ContentPart{
			message.ToolCall{
				ID:       "call_00_WGMYBpDInVK4DWgfx7zl2556",
				Name:     "bash",
				Input:    `{"command": "find /home/cindy/TMP/lbs_code_dir/20260914_1814 -type f | wc -l"}`,
				Type:     "function",
				Finished: true,
			},
			message.Finish{Reason: "tool_use"},
		}},
		{ID: "m2", SessionID: "s1", Role: message.Tool, Parts: []message.ContentPart{
			message.ToolResult{ToolCallID: "call_00_WGMYBpDInVK4DWgfx7zl2556", Name: "bash", Content: "18", IsError: false},
			message.Finish{Reason: "stop"},
		}},
	}}
	core := &app.App{Messages: svc}
	c := NewChatArea(core, context.Background())
	c.current = "s1"

	c.renderNow()

	var texts []string
	collectToolBlockTexts(c.output, &texts)
	joined := strings.Join(texts, "\n")
	if !strings.Contains(joined, "find /home/cindy/TMP/lbs_code_dir/20260914_1814") {
		t.Fatalf("expected the bash command text in rendered output, got:\n%s", joined)
	}
}

func collectToolBlockTexts(obj fyne.CanvasObject, out *[]string) {
	if tb, ok := obj.(*ToolBlock); ok {
		if tb.output.Text != "" {
			*out = append(*out, tb.output.Text)
		}
		if len(tb.title.Segments) > 0 {
			if seg, ok := tb.title.Segments[0].(*widget.TextSegment); ok {
				*out = append(*out, seg.Text)
			}
		}
		return
	}
	if cc, ok := obj.(*fyne.Container); ok {
		for _, child := range cc.Objects {
			collectToolBlockTexts(child, out)
		}
	}
}
