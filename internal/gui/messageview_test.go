package gui

import (
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
	if got := block.TitleText(); got != "bash" {
		t.Fatalf("expected running state title to be the tool name, got %q", got)
	}
	if block.output.Text != "ls" {
		t.Fatalf("expected bash command rendered in output, got %q", block.output.Text)
	}
}

func TestSummarizeToolInput(t *testing.T) {
	cases := []struct {
		name, tool, input, want string
	}{
		{"bash cmd", "bash", `{"cmd":"go test ./..."}`, "go test ./..."},
		{"bash non-json", "bash", "ls", "ls"},
		{"other tool json kept", "glob", `{"pattern":"*.go"}`, `{"pattern":"*.go"}`},
	}
	for _, c := range cases {
		if got := summarizeToolInput(c.tool, c.input); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
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

func TestRenderMessageToolResultCreatesNewBlock(t *testing.T) {
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
	if used["call_1"] == live {
		t.Fatal("expected a fresh block for the tool result, not the reused live one")
	}
	if used["call_1"].output.Text != "out" {
		t.Fatalf("expected tool result output rendered, got %q", used["call_1"].output.Text)
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

func TestRenderMessageToolResultErrorFallbackName(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role: message.Tool,
		Parts: []message.ContentPart{
			message.ToolResult{ToolCallID: "call_xx", Name: "", Content: "err", IsError: true},
		},
	}
	_, used := renderMessage(m, map[string]*ToolBlock{}, nil, false)
	block, ok := used["call_xx"]
	if !ok {
		t.Fatal("expected used map to contain call_xx")
	}
	if got := block.TitleText(); got != "error" {
		t.Fatalf("expected error fallback title 'error', got %q", got)
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
func TestRenderMessageToolResultShowsOutput(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role: message.Tool,
		Parts: []message.ContentPart{
			message.ToolResult{ToolCallID: "call_1", Name: "bash", Content: "ok"},
		},
	}
	_, used := renderMessage(m, map[string]*ToolBlock{}, nil, false)
	block, ok := used["call_1"]
	if !ok {
		t.Fatal("expected used map to contain call_1")
	}
	if block.output.Text != "ok" {
		t.Fatalf("expected bash output content to be shown, got %q", block.output.Text)
	}
	if !block.title.Hidden {
		t.Fatal("expected successful tool title to be hidden")
	}
}

func TestRenderMessageCodeToolCallCompact(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.ToolCall{ID: "call_2", Name: "read", Input: "/path/file.go"},
		},
	}
	_, used := renderMessage(m, map[string]*ToolBlock{}, nil, false)
	block, ok := used["call_2"]
	if !ok {
		t.Fatal("expected used map to contain call_2")
	}
	if got := block.TitleText(); got != "read" {
		t.Fatalf("expected compact title to be the tool name, got %q", got)
	}
	if block.output.Text != "" {
		t.Fatalf("expected no tool input rendered for code tool, got %q", block.output.Text)
	}
}

func TestRenderMessageCodeToolResultCompact(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role: message.Tool,
		Parts: []message.ContentPart{
			message.ToolResult{ToolCallID: "call_2", Name: "read", Content: "package main\n"},
		},
	}
	_, used := renderMessage(m, map[string]*ToolBlock{}, nil, false)
	block, ok := used["call_2"]
	if !ok {
		t.Fatal("expected used map to contain call_2")
	}
	if got := block.TitleText(); got != "read" {
		t.Fatalf("expected compact title to be the tool name, got %q", got)
	}
	if block.output.Text != "" {
		t.Fatalf("expected no file content rendered for code tool, got %q", block.output.Text)
	}
}

func TestRenderMessageToolCallCompactTitleOnly(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.ToolCall{ID: "call_1", Name: "bash", Input: `{"cmd":"ls"}`},
		},
	}
	_, used := renderMessage(m, map[string]*ToolBlock{}, nil, true)
	block, ok := used["call_1"]
	if !ok {
		t.Fatal("expected used map to contain call_1")
	}
	if got := block.TitleText(); got != "bash" {
		t.Fatalf("expected compact title to be the tool name, got %q", got)
	}
	if block.output.Text != "" {
		t.Fatalf("expected no tool input rendered in compact mode, got %q", block.output.Text)
	}
	if !block.outputBox.Hidden {
		t.Fatal("expected compact block output area to be hidden")
	}
}

func TestRenderMessageToolResultCompactTitleOnly(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role: message.Tool,
		Parts: []message.ContentPart{
			message.ToolResult{ToolCallID: "call_1", Name: "bash", Content: "out"},
		},
	}
	_, used := renderMessage(m, map[string]*ToolBlock{}, nil, true)
	block, ok := used["call_1"]
	if !ok {
		t.Fatal("expected used map to contain call_1")
	}
	if got := block.TitleText(); got != "bash" {
		t.Fatalf("expected compact title to be the tool name, got %q", got)
	}
	if block.output.Text != "" {
		t.Fatalf("expected no tool result content rendered in compact mode, got %q", block.output.Text)
	}
	if !block.outputBox.Hidden {
		t.Fatal("expected compact block output area to be hidden")
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
