package gui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/cindyhuang123/hylbscode/internal/message"
)

func TestRenderMessageToolCallCreatesBlock(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.ToolCall{ID: "call_1", Name: "bash", Input: `{"cmd":"ls"}`},
		},
	}
	_, used := renderMessage(m, map[string]*ToolBlock{}, nil, false)
	block, ok := used["call_1"]
	if !ok {
		t.Fatal("expected used map to contain call_1")
	}
	if got := block.TitleText(); !strings.HasPrefix(got, "⏳") {
		t.Fatalf("expected running state title, got %q", got)
	}
	if block.output.Text == "" {
		t.Fatal("expected input rendered in output")
	}
}

func TestRenderMessageToolCallFinished(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.ToolCall{ID: "call_1", Name: "bash", Input: "ls", Finished: true},
		},
	}
	_, used := renderMessage(m, map[string]*ToolBlock{}, nil, false)
	if !used["call_1"].title.Hidden {
		t.Fatal("expected successful tool title to be hidden")
	}
}

func TestRenderMessageToolCallReusesLiveBlock(t *testing.T) {
	test.NewApp()
	live := NewToolBlock("bash")
	active := map[string]*ToolBlock{"call_1": live}
	m := message.Message{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.ToolCall{ID: "call_1", Name: "bash", Finished: true},
		},
	}
	_, used := renderMessage(m, active, nil, false)
	if used["call_1"] != live {
		t.Fatal("expected the live block to be reused")
	}
}

func TestRenderMessageToolResultReusesLiveBlock(t *testing.T) {
	test.NewApp()
	live := NewToolBlock("bash")
	active := map[string]*ToolBlock{"call_1": live}
	m := message.Message{
		Role: message.Tool,
		Parts: []message.ContentPart{
			message.ToolResult{ToolCallID: "call_1", Name: "bash", Content: "out", IsError: true},
		},
	}
	_, used := renderMessage(m, active, nil, false)
	if used["call_1"] != live {
		t.Fatal("expected the live block to be reused")
	}
	if got := live.TitleText(); !strings.HasPrefix(got, "✗") {
		t.Fatalf("expected error title, got %q", got)
	}
	if live.output.Text == "" {
		t.Fatal("expected result content to replace output")
	}
}

func TestRenderMessageToolResultNewBlock(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role: message.Tool,
		Parts: []message.ContentPart{
			message.ToolResult{ToolCallID: "call_1", Name: "bash", Content: "out"},
		},
	}
	_, used := renderMessage(m, map[string]*ToolBlock{}, nil, false)
	if used["call_1"] == nil {
		t.Fatal("expected a new block for an unknown tool result")
	}
	if !used["call_1"].title.Hidden {
		t.Fatal("expected successful tool title to be hidden")
	}
}

func TestRenderMessageToolCallSkippedWhenResultStored(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.ToolCall{ID: "call_1", Name: "bash", Input: "ls", Finished: true},
		},
	}
	_, got := renderMessage(m, map[string]*ToolBlock{}, map[string]bool{"call_1": true}, false)
	if len(got) != 0 {
		t.Fatal("expected the tool call block to be skipped when the result is stored")
	}
}
func TestRenderMessageStripsANSI(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.TextContent{Text: "hello \x1b[31mred\x1b[0m"},
		},
	}
	obj, used := renderMessage(m, map[string]*ToolBlock{}, nil, false)
	if obj == nil || len(used) != 0 {
		t.Fatal("expected a view with no tool blocks")
	}
}

func TestParseInlinePlainLine(t *testing.T) {
	segs, has := parseInline("just plain text")
	if has {
		t.Fatal("plain line should not report inline markup")
	}
	if len(segs) != 1 || segs[0].text != "just plain text" || segs[0].code || segs[0].link {
		t.Fatalf("unexpected segments: %+v", segs)
	}
}

func TestParseInlineCode(t *testing.T) {
	segs, has := parseInline("run `go build` now")
	if !has {
		t.Fatal("inline code should report markup")
	}
	if len(segs) != 3 {
		t.Fatalf("expected 3 segments, got %d: %+v", len(segs), segs)
	}
	if segs[1].text != "go build" || !segs[1].code {
		t.Fatalf("expected code segment, got %+v", segs[1])
	}
}

func TestParseInlineLink(t *testing.T) {
	segs, has := parseInline("see [docs](https://example.com) here")
	if !has {
		t.Fatal("link should report markup")
	}
	if len(segs) != 3 {
		t.Fatalf("expected 3 segments, got %d: %+v", len(segs), segs)
	}
	if segs[1].text != "docs" || !segs[1].link {
		t.Fatalf("expected link segment, got %+v", segs[1])
	}
}

func TestParseInlineUnclosedBacktick(t *testing.T) {
	segs, has := parseInline("oops `unclosed")
	if has {
		t.Fatal("unclosed backtick should not report markup")
	}
	if len(segs) != 1 || segs[0].text != "oops `unclosed" {
		t.Fatalf("expected single plain segment, got %+v", segs)
	}
}
