package gui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"
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
