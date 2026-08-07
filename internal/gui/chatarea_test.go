package gui

import (
	"context"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/cindyhuang123/hylbscode/internal/app"
	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/llm/agent"
	"github.com/cindyhuang123/hylbscode/internal/llm/models"
	"github.com/cindyhuang123/hylbscode/internal/message"
	"github.com/cindyhuang123/hylbscode/internal/pubsub"
)

// fakeAgentService implements agent.Service with a closable event channel so
// ChatArea.Send exercises the full start path and tests can finish the run.
type fakeAgentService struct {
	runCh chan agent.AgentEvent
}

func (f *fakeAgentService) Subscribe(ctx context.Context) <-chan pubsub.Event[agent.AgentEvent] {
	return make(chan pubsub.Event[agent.AgentEvent])
}
func (f *fakeAgentService) Model() models.Model { return models.Model{} }
func (f *fakeAgentService) Run(ctx context.Context, sessionID string, content string, attachments ...message.Attachment) (<-chan agent.AgentEvent, error) {
	f.runCh = make(chan agent.AgentEvent)
	return f.runCh, nil
}
func (f *fakeAgentService) Cancel(sessionID string)             {}
func (f *fakeAgentService) IsSessionBusy(sessionID string) bool { return false }
func (f *fakeAgentService) IsBusy() bool                        { return false }
func (f *fakeAgentService) Update(agentName config.AgentName, modelID models.ModelID) (models.Model, error) {
	return models.Model{}, nil
}
func (f *fakeAgentService) Summarize(ctx context.Context, sessionID string) error { return nil }

// TestSendBlockedWhileStreaming verifies that sending while the agent is
// still responding shows the blocked hint on the status line instead of
// silently dropping the keypress.
func TestSendBlockedWhileStreaming(t *testing.T) {
	test.NewApp()
	core := &app.App{Messages: &fakeMessageService{}}
	c := NewChatArea(core, context.Background())

	c.streaming.Store(true)
	c.Send()

	seg := c.status.Segments[0].(*widget.TextSegment)
	if seg.Text != "⏳ 正在应答中，请等应答完毕后再发送" {
		t.Fatalf("expected blocked hint, got %q", seg.Text)
	}
}

// TestCancelResponse verifies Esc cancels the running response: the stored
// cancel func fires and the cancelled flag is set so the final status line
// shows the cancelled state instead of "done".
func TestCancelResponse(t *testing.T) {
	test.NewApp()
	core := &app.App{Messages: &fakeMessageService{}}
	c := NewChatArea(core, context.Background())

	called := false
	c.cancelFunc = func() { called = true }
	c.streaming.Store(true)

	c.CancelResponse()
	if !called {
		t.Fatal("expected cancelFunc to be called")
	}
	if !c.cancelled.Load() {
		t.Fatal("expected cancelled flag to be set")
	}

	// Idle state: CancelResponse must be a no-op.
	c.streaming.Store(false)
	c.CancelResponse()
}

// TestSetCancelledUI verifies the status line switches to the cancelled
// state after a response was aborted.
func TestSetCancelledUI(t *testing.T) {
	test.NewApp()
	core := &app.App{Messages: &fakeMessageService{}}
	c := NewChatArea(core, context.Background())

	c.setCancelledUI()
	seg := c.status.Segments[0].(*widget.TextSegment)
	if seg.Text != "✖ 已取消" {
		t.Fatalf("expected cancelled status, got %q", seg.Text)
	}
}

// TestSendAppendsHistory verifies sent messages are recorded for the
// Up/Down history navigation.
func TestSendAppendsHistory(t *testing.T) {
	test.NewApp()
	svc := &fakeAgentService{}
	core := &app.App{Messages: &fakeMessageService{}, CoderAgent: svc}
	c := NewChatArea(core, context.Background())
	c.current = "s1"

	c.input.SetText("first question")
	c.Send()
	waitAgentIdle(t, c, svc)
	c.input.SetText("second question")
	c.Send()

	if len(c.history) != 2 || c.history[0] != "first question" || c.history[1] != "second question" {
		t.Fatalf("expected 2 history entries, got %v", c.history)
	}
}

// TestHistoryDedup verifies an exact repeat of the previous message is not
// recorded twice.
func TestHistoryDedup(t *testing.T) {
	test.NewApp()
	svc := &fakeAgentService{}
	core := &app.App{Messages: &fakeMessageService{}, CoderAgent: svc}
	c := NewChatArea(core, context.Background())
	c.current = "s1"

	c.input.SetText("same")
	c.Send()
	waitAgentIdle(t, c, svc)
	c.input.SetText("same")
	c.Send()

	if len(c.history) != 1 {
		t.Fatalf("expected 1 history entry after repeat, got %v", c.history)
	}
}

// waitAgentIdle closes the fake event channel and waits until the consuming
// goroutine has reset the streaming flag.
func waitAgentIdle(t *testing.T, c *ChatArea, svc *fakeAgentService) {
	t.Helper()
	close(svc.runCh)
	deadline := time.Now().Add(2 * time.Second)
	for c.streaming.Load() {
		if time.Now().After(deadline) {
			t.Fatal("agent run did not finish")
		}
		time.Sleep(time.Millisecond)
	}
}

// TestHistoryNavigation verifies Up/Down recall previous messages and restore
// the current draft at the newest end.
func TestHistoryNavigation(t *testing.T) {
	test.NewApp()
	core := &app.App{Messages: &fakeMessageService{}}
	c := NewChatArea(core, context.Background())
	c.history = []string{"q1", "q2"}
	c.historyIdx = -1
	c.input.SetText("draft")

	c.historyUp()
	if c.input.Text() != "q2" {
		t.Fatalf("expected q2 after Up, got %q", c.input.Text())
	}
	c.historyUp()
	if c.input.Text() != "q1" {
		t.Fatalf("expected q1 after second Up, got %q", c.input.Text())
	}
	c.historyUp() // already at the oldest entry
	if c.input.Text() != "q1" {
		t.Fatalf("expected to stay at q1, got %q", c.input.Text())
	}
	c.historyDown()
	if c.input.Text() != "q2" {
		t.Fatalf("expected q2 after Down, got %q", c.input.Text())
	}
	c.historyDown()
	if c.input.Text() != "draft" {
		t.Fatalf("expected draft after second Down, got %q", c.input.Text())
	}
}

// TestScrollHomeEnd verifies Home/End scroll the message list to the top and
// bottom (Home/End target the output area, unlike Up/Down history).
func TestScrollHomeEnd(t *testing.T) {
	test.NewApp()
	core := &app.App{Messages: &fakeMessageService{}}
	c := NewChatArea(core, context.Background())

	c.scroll.Offset.Y = 100
	c.scrollHome()
	if c.scroll.Offset.Y != 0 {
		t.Fatalf("expected offset 0 at Home, got %f", c.scroll.Offset.Y)
	}
	c.scrollEnd()
	if c.scroll.Offset.Y != 0 {
		t.Fatalf("expected offset 0 at End with empty content, got %f", c.scroll.Offset.Y)
	}
}
