package gui

import (
	"context"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/cindyhuang123/hylbscode/internal/app"
	"github.com/cindyhuang123/hylbscode/internal/message"
	"github.com/cindyhuang123/hylbscode/internal/pubsub"
)

type fakeMessageService struct {
	msgs []message.Message
}

func (f *fakeMessageService) Create(ctx context.Context, sessionID string, params message.CreateMessageParams) (message.Message, error) {
	return message.Message{}, nil
}
func (f *fakeMessageService) Update(ctx context.Context, m message.Message) error { return nil }
func (f *fakeMessageService) Get(ctx context.Context, id string) (message.Message, error) {
	return message.Message{}, nil
}
func (f *fakeMessageService) List(ctx context.Context, sessionID string) ([]message.Message, error) {
	return f.msgs, nil
}
func (f *fakeMessageService) Delete(ctx context.Context, id string) error { return nil }
func (f *fakeMessageService) DeleteSessionMessages(ctx context.Context, sessionID string) error {
	return nil
}
func (f *fakeMessageService) Subscribe(ctx context.Context) <-chan pubsub.Event[message.Message] {
	return make(chan pubsub.Event[message.Message])
}

// TestRenderNowKeepsRunningToolBlock exercises the renderNow coordination:
// a live block for a tool that is still running (no persisted ToolCall part
// in the message list) must survive a full re-render and stay re-attached to
// the output container.
func TestRenderNowKeepsRunningToolBlock(t *testing.T) {
	test.NewApp()
	core := &app.App{
		Messages: &fakeMessageService{msgs: []message.Message{
			{
				Role: message.Assistant,
				Parts: []message.ContentPart{
					message.TextContent{Text: "hello"},
				},
			},
		}},
	}
	c := NewChatArea(core, context.Background())
	c.current = "s1"

	live := NewToolBlock("bash")
	c.toolBlocks["call_1"] = live
	c.output.Add(live)

	c.renderNow()

	if c.toolBlocks["call_1"] != live {
		t.Fatal("expected running tool block to survive re-render")
	}
	found := false
	for _, obj := range c.output.Objects {
		if obj == live {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected running tool block re-attached to output")
	}
}

// TestRenderNowConsumesFinishedToolCall verifies that a persisted ToolCall
// part reuses the live block (keeping streamed content) and transitions it to
// the finished state during a re-render.
func TestRenderNowConsumesFinishedToolCall(t *testing.T) {
	test.NewApp()
	core := &app.App{
		Messages: &fakeMessageService{msgs: []message.Message{
			{
				Role: message.Assistant,
				Parts: []message.ContentPart{
					message.ToolCall{ID: "call_1", Name: "bash", Input: "ls", Finished: true},
				},
			},
		}},
	}
	c := NewChatArea(core, context.Background())
	c.current = "s1"

	live := NewToolBlock("bash")
	c.toolBlocks["call_1"] = live

	c.renderNow()

	block, ok := c.toolBlocks["call_1"]
	if !ok {
		t.Fatal("expected tool block registered after render")
	}
	if block != live {
		t.Fatal("expected the live block to be reused for the finished call")
	}
	if !block.title.Hidden {
		t.Fatal("expected successful tool title to be hidden")
	}
}

// TestRenderNowReusesCachedViews verifies that unchanged messages keep their
// previously rendered view across re-renders (incremental rendering), so a
// long session does not rebuild every message widget on each streaming tick.
func TestRenderNowReusesCachedViews(t *testing.T) {
	test.NewApp()
	core := &app.App{
		Messages: &fakeMessageService{msgs: []message.Message{
			{ID: "m1", SessionID: "s1", Role: message.User, Parts: []message.ContentPart{
				message.TextContent{Text: "hi"},
			}},
			{ID: "m2", SessionID: "s1", Role: message.Assistant, Parts: []message.ContentPart{
				message.TextContent{Text: "hello"},
			}},
		}},
	}
	c := NewChatArea(core, context.Background())
	c.current = "s1"

	c.renderNow()
	first := c.output.Objects[0]
	second := c.output.Objects[1]

	c.renderNow()
	if c.output.Objects[0] != first || c.output.Objects[1] != second {
		t.Fatal("expected unchanged message views to be reused from cache")
	}
	if len(c.renderCache) != 2 {
		t.Fatalf("expected both messages cached, got %d", len(c.renderCache))
	}
}

// TestRenderNowRedrawsDirtyMessage verifies that a message marked dirty (e.g.
// by a streaming Update event) is re-rendered while the other stays cached.
func TestRenderNowRedrawsDirtyMessage(t *testing.T) {
	test.NewApp()
	core := &app.App{
		Messages: &fakeMessageService{msgs: []message.Message{
			{ID: "m1", SessionID: "s1", Role: message.User, Parts: []message.ContentPart{
				message.TextContent{Text: "hi"},
			}},
		}},
	}
	c := NewChatArea(core, context.Background())
	c.current = "s1"

	c.renderNow()
	cached := c.output.Objects[0]
	// Simulate a streaming update on a different message.
	core.Messages.(*fakeMessageService).msgs = append(core.Messages.(*fakeMessageService).msgs,
		message.Message{ID: "m2", SessionID: "s1", Role: message.Assistant, Parts: []message.ContentPart{
			message.TextContent{Text: "streaming"},
		}})
	c.toolMu.Lock()
	c.dirty = map[string]bool{"m2": true}
	c.toolMu.Unlock()

	c.renderNow()
	if c.output.Objects[0] != cached {
		t.Fatal("expected unchanged m1 view to be reused")
	}
	if len(c.output.Objects) != 2 {
		t.Fatalf("expected 2 views, got %d", len(c.output.Objects))
	}
}

// TestSetStreamingUI verifies the answer-status line switches between the
// streaming and finished states as the agent runs and completes.
func TestSetStreamingUI(t *testing.T) {
	test.NewApp()
	core := &app.App{Messages: &fakeMessageService{}}
	c := NewChatArea(core, context.Background())

	c.setStreamingUI(true)
	seg := c.status.Segments[0].(*widget.TextSegment)
	if seg.Text != "⏳ 正在应答…" {
		t.Fatalf("expected streaming status, got %q", seg.Text)
	}

	c.setStreamingUI(false)
	seg = c.status.Segments[0].(*widget.TextSegment)
	if seg.Text != "✔ 应答完毕" {
		t.Fatalf("expected done status, got %q", seg.Text)
	}
}

// TestRenderNowDoesNotDuplicateToolBlock reproduces the streaming tool flow:
// the assistant message with an unfinished ToolCall is rendered first (its
// view embeds the live block), then the ToolResult message arrives. The
// second render must re-render the ToolCall message (not reuse its cached
// view) so the block appears only once, inside the ToolResult message.
func TestRenderNowDoesNotDuplicateToolBlock(t *testing.T) {
	test.NewApp()
	svc := &fakeMessageService{msgs: []message.Message{
		{ID: "m5", SessionID: "s1", Role: message.Assistant, Parts: []message.ContentPart{
			message.ToolCall{ID: "call_9", Name: "grep", Input: "foo"},
		}},
	}}
	core := &app.App{Messages: svc}
	c := NewChatArea(core, context.Background())
	c.current = "s1"

	c.renderNow()
	cached := c.output.Objects[0]
	if cached == nil {
		t.Fatal("expected a message view")
	}

	svc.msgs = append(svc.msgs, message.Message{
		ID: "m6", SessionID: "s1", Role: message.Tool, Parts: []message.ContentPart{
			message.ToolResult{ToolCallID: "call_9", Name: "grep", Content: "match"},
		},
	})
	c.renderNow()

	if n := countToolBlocks(c.output); n != 1 {
		t.Fatalf("expected exactly 1 tool block in output, got %d", n)
	}
}

// countToolBlocks recursively counts ToolBlock instances inside a widget tree.
func countToolBlocks(obj fyne.CanvasObject) int {
	switch o := obj.(type) {
	case *ToolBlock:
		return 1
	case *fyne.Container:
		n := 0
		for _, child := range o.Objects {
			n += countToolBlocks(child)
		}
		return n
	default:
		return 0
	}
}
