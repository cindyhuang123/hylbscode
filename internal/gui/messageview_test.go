package gui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"strings"

	"github.com/cindyhuang123/hylbscode/internal/message"
)

func TestRenderMessageToolCallCreatesBlock(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.ToolCall{ID: "call_1", Name: "bash", Input: `{"command":"ls"}`},
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
	if block.output.Text != "" {
		t.Fatalf("expected no output shown while the tool is still running, got %q", block.output.Text)
	}
	if !block.outputBox.Hidden {
		t.Fatal("expected running tool block output area to be hidden")
	}
	if block.expandBtn.Hidden {
		t.Fatal("expected expand button visible while the tool is running")
	}
}

func TestRenderMessageFinishedReplacesRunningCompactBlock(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.ToolCall{ID: "call_1", Name: "bash", Input: `{"command":"ls"}`, Finished: true},
		},
	}
	_, used := renderMessage(m, map[string]*ToolBlock{}, nil, false)
	block, ok := used["call_1"]
	if !ok {
		t.Fatal("expected used map to contain call_1")
	}
	if block.outputBox.Hidden {
		t.Fatal("expected finished tool block output area to be visible")
	}
}

func TestSummarizeToolInput(t *testing.T) {
	cases := []struct {
		name, tool, input, want string
	}{
		{"bash cmd", "bash", `{"command":"go test ./..."}`, "go test ./..."},
		{"bash non-json", "bash", "ls", "ls"},
		{"other tool json kept", "glob", `{"pattern":"*.go"}`, `{"pattern":"*.go"}`},
	}
	for _, c := range cases {
		if got := summarizeToolInput(c.tool, c.input); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

// TestRenderMessageReusedLiveBlockGetsCommand reproduces the streaming flow:
// the live block created while the tool ran (ToolStart carries no input, so
// it has no command), then the finished ToolCall re-renders and must replace
// the reused block's output with the actual command.
func TestRenderMessageToolCallCompactShowsCommand(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.ToolCall{ID: "call_1", Name: "bash", Input: `{"command":"find x | wc -l"}`, Finished: true},
		},
	}
	_, used := renderMessage(m, map[string]*ToolBlock{}, nil, true)
	block, ok := used["call_1"]
	if !ok {
		t.Fatal("expected used map to contain call_1")
	}
	if got := block.TitleText(); got != "bash  find x | wc -l" {
		t.Fatalf("expected compact title to carry the command, got %q", got)
	}
	if block.output.Text != "" {
		t.Fatalf("expected no output area in compact mode, got %q", block.output.Text)
	}
}

func TestRenderMessageReusedLiveBlockGetsCommand(t *testing.T) {
	test.NewApp()
	live := NewToolBlock("bash")
	active := map[string]*ToolBlock{"call_1": live}
	m := message.Message{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.ToolCall{ID: "call_1", Name: "bash", Input: `{"command":"find x | wc -l"}`, Finished: true},
		},
	}
	_, used := renderMessage(m, active, nil, false)
	if used["call_1"] != live {
		t.Fatal("expected the live block to be reused")
	}
	if live.output.Text != "find x | wc -l" {
		t.Fatalf("expected command in reused block output, got %q", live.output.Text)
	}
	if got := live.TitleText(); got != "bash" {
		t.Fatalf("expected tool name title, got %q", got)
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
	if used["call_1"].title.Hidden {
		t.Fatal("expected the tool name title to stay visible next to the command")
	}
	if got := used["call_1"].TitleText(); got != "bash" {
		t.Fatalf("expected title to be the tool name, got %q", got)
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
			message.ToolCall{ID: "call_1", Name: "bash", Input: `{"command":"ls"}`},
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
		t.Fatalf("expected no tool result content rendered in collapsed compact mode, got %q", block.output.Text)
	}
	if !block.outputBox.Hidden {
		t.Fatal("expected compact block output area to be hidden")
	}
	if block.expandBtn.Hidden {
		t.Fatal("expected expand button to be visible when compact block has output")
	}
}

func TestRenderMessageToolResultCompactExpand(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role: message.Tool,
		Parts: []message.ContentPart{
			message.ToolResult{ToolCallID: "call_1", Name: "bash", Content: "line1\nline2"},
		},
	}
	_, used := renderMessage(m, map[string]*ToolBlock{}, nil, true)
	block := used["call_1"]
	if block == nil {
		t.Fatal("expected used map to contain call_1")
	}
	if !block.outputBox.Hidden {
		t.Fatal("expected output area hidden before expand")
	}
	block.toggleExpand()
	if block.outputBox.Hidden {
		t.Fatal("expected output area visible after expand")
	}
	if block.output.Text != "line1\nline2" {
		t.Fatalf("expected expanded output to show content, got %q", block.output.Text)
	}
	block.toggleExpand()
	if !block.outputBox.Hidden {
		t.Fatal("expected output area hidden again after collapse")
	}
}

func TestRenderMessageToolResultCompactNoContentNoButton(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role: message.Tool,
		Parts: []message.ContentPart{
			message.ToolResult{ToolCallID: "call_1", Name: "bash", Content: ""},
		},
	}
	_, used := renderMessage(m, map[string]*ToolBlock{}, nil, true)
	block := used["call_1"]
	if block == nil {
		t.Fatal("expected used map to contain call_1")
	}
	if !block.expandBtn.Hidden {
		t.Fatal("expected expand button hidden when compact block has no output")
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

func collectButtons(obj fyne.CanvasObject) []*widget.Button {
	var out []*widget.Button
	var walk func(fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if b, ok := o.(*widget.Button); ok {
			out = append(out, b)
			return
		}
		if c, ok := o.(*fyne.Container); ok {
			for _, child := range c.Objects {
				walk(child)
			}
		}
	}
	walk(obj)
	return out
}

func TestAssistantCopyButtonWritesClipboard(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.ReasoningContent{Thinking: "thinking step"},
			message.TextContent{Text: "answer with `code` and [link](https://x.dev)"},
		},
	}
	view, _ := renderMessage(m, map[string]*ToolBlock{}, nil, false)

	buttons := collectButtons(view)
	if len(buttons) == 0 {
		t.Fatal("assistant message should render a copy button, found none")
	}
	buttons[0].OnTapped()

	got := fyne.CurrentApp().Clipboard().Content()
	if !strings.Contains(got, "thinking step") || !strings.Contains(got, "answer with") {
		t.Fatalf("clipboard should contain reasoning and body, got %q", got)
	}
}

// 纯工具轮 assistant(非最后一条)只有 ToolCall parts, 复制应产出工具摘要并写入剪贴板.
func TestToolUseAssistantCopyWritesClipboard(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role: message.Assistant,
		Parts: []message.ContentPart{
			message.ToolCall{ID: "call_1", Name: "bash", Input: `{"command":"go test ./internal/gui"}`},
			message.ToolCall{ID: "call_2", Name: "view", Input: `{"file_path":"main.go"}`},
		},
	}
	view, _ := renderMessage(m, map[string]*ToolBlock{}, nil, false)

	buttons := collectButtons(view)
	if len(buttons) == 0 {
		t.Fatal("tool-use assistant message should render a copy button, found none")
	}
	buttons[0].OnTapped()

	got := fyne.CurrentApp().Clipboard().Content()
	if !strings.Contains(got, "bash") || !strings.Contains(got, "go test ./internal/gui") {
		t.Fatalf("clipboard should contain the bash tool summary, got %q", got)
	}
	if !strings.Contains(got, "view") {
		t.Fatalf("clipboard should contain the view tool name, got %q", got)
	}
}

// Tool 角色消息(工具返回)也应有复制按钮, 复制 ToolResult 内容.
func TestToolResultCopyWritesClipboard(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role:  message.Tool,
		Parts: []message.ContentPart{message.ToolResult{Name: "bash", Content: "PASS\nok  github.com/cindyhuang123/hylbscode/internal/gui  0.317s"}},
	}
	view, _ := renderMessage(m, map[string]*ToolBlock{}, nil, false)

	buttons := collectButtons(view)
	if len(buttons) == 0 {
		t.Fatal("tool message should render a copy button, found none")
	}
	buttons[0].OnTapped()

	got := fyne.CurrentApp().Clipboard().Content()
	if !strings.Contains(got, "PASS") || !strings.Contains(got, "0.317s") {
		t.Fatalf("clipboard should contain the tool result content, got %q", got)
	}
}

func TestUserCopyButtonWritesClipboard(t *testing.T) {
	test.NewApp()
	m := message.Message{
		Role:  message.User,
		Parts: []message.ContentPart{message.TextContent{Text: "user question"}},
	}
	view, _ := renderMessage(m, map[string]*ToolBlock{}, nil, false)

	buttons := collectButtons(view)
	if len(buttons) == 0 {
		t.Fatal("user message should render a copy button, found none")
	}
	buttons[0].OnTapped()

	got := fyne.CurrentApp().Clipboard().Content()
	if !strings.Contains(got, "user question") {
		t.Fatalf("clipboard should contain the user text, got %q", got)
	}
}

// 复刻 chat.go renderNow 的缓存+渲染循环, 验证复制按钮不串消息.
func TestRenderCycleCopyIsolation(t *testing.T) {
	test.NewApp()
	a1 := message.Message{ID: "a1", Role: message.Assistant, Parts: []message.ContentPart{message.TextContent{Text: "first answer"}}}
	a2 := message.Message{ID: "a2", Role: message.Assistant, Parts: []message.ContentPart{message.TextContent{Text: "second answer"}}}
	msgs := []message.Message{a1, a2}
	cache := make(map[string]fyne.CanvasObject)
	active := map[string]*ToolBlock{}
	views := make([]fyne.CanvasObject, 0, len(msgs))
	for _, m := range msgs {
		if v, ok := cache[m.ID]; ok {
			views = append(views, v)
			continue
		}
		v, _ := renderMessage(m, active, nil, false)
		cache[m.ID] = v
		views = append(views, v)
	}

	btn1 := collectButtons(cache["a1"])[0]
	btn1.OnTapped()
	if got := fyne.CurrentApp().Clipboard().Content(); got != "first answer" {
		t.Fatalf("button of first message should copy its own text, got %q", got)
	}
	btn2 := collectButtons(cache["a2"])[0]
	btn2.OnTapped()
	if got := fyne.CurrentApp().Clipboard().Content(); got != "second answer" {
		t.Fatalf("button of second message should copy its own text, got %q", got)
	}
}
