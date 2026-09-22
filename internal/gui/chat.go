package gui

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/cindyhuang123/hylbscode/internal/app"
	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/llm/agent"
	"github.com/cindyhuang123/hylbscode/internal/logging"
	"github.com/cindyhuang123/hylbscode/internal/message"
	"github.com/cindyhuang123/hylbscode/internal/pubsub"
)

type ChatArea struct {
	core *app.App
	ctx  context.Context

	current string
	output  *fyne.Container
	scroll  *container.Scroll
	input   *ChatInput
	sched   *renderScheduler

	streaming   atomic.Bool
	cancelled   atomic.Bool
	errState    atomic.Bool
	errMsg      string
	cancelFunc  context.CancelFunc
	onSessionID func(string)
	onSetup     func()
	toolMu      sync.Mutex
	toolBlocks  map[string]*ToolBlock
	renderCache map[string]fyne.CanvasObject
	dirty       map[string]bool
	win         fyne.Window
	pending     []string

	attachments []message.Attachment
	attachLabel *widget.Label
	status      *widget.RichText

	history       []string
	historyIdx    int
	historyDraft  string
	historyLoaded bool
}

func NewChatArea(core *app.App, ctx context.Context) *ChatArea {
	c := &ChatArea{core: core, ctx: ctx}
	c.output = container.NewVBox()
	c.scroll = container.NewVScroll(c.output)
	c.input = NewChatInput(func(string) { c.Send() }, c.CancelResponse)
	c.input.SetPageHandlers(c.pageUp, c.pageDown, c.scrollHome, c.scrollEnd)
	c.input.SetHistoryHandlers(c.historyUp, c.historyDown)
	c.toolBlocks = make(map[string]*ToolBlock)
	c.renderCache = make(map[string]fyne.CanvasObject)
	c.attachLabel = widget.NewLabel("")
	c.attachLabel.Wrapping = fyne.TextWrapWord
	c.status = widget.NewRichText()
	c.sched = newRenderScheduler(80*time.Millisecond, func() {
		fyne.Do(c.renderNow)
	})
	return c
}

func (c *ChatArea) FocusInput() {
	if cv := fyne.CurrentApp().Driver().CanvasForObject(c.input); cv != nil {
		cv.Focus(c.input)
		logging.Info("chatarea: focus input requested")
	} else {
		logging.Warn("chatarea: focus input failed, canvas not found")
	}
}

func (c *ChatArea) Content() fyne.CanvasObject {
	attachBtn := widget.NewButton(config.Tr().GUIAttachFile, c.PickAttachment)
	attachRow := container.NewHBox(attachBtn, c.attachLabel)
	inputBox := container.NewBorder(nil, nil, attachRow, nil, c.input)
	bottom := container.NewVBox(c.status, inputBox)
	return container.NewBorder(nil, bottom, nil, nil, c.scroll)
}

// setStreamingUI reflects the agent's running state in a status line between
// the message list and the input, so the user can tell whether the assistant
// is still responding or has finished.
func (c *ChatArea) setStreamingUI(streaming bool) {
	tr := config.Tr()
	if streaming {
		c.status.Segments = []widget.RichTextSegment{&widget.TextSegment{
			Text:  tr.ChatStreaming,
			Style: widget.RichTextStyle{ColorName: theme.ColorNamePrimary, TextStyle: fyne.TextStyle{Bold: true}},
		}}
	} else {
		c.status.Segments = []widget.RichTextSegment{&widget.TextSegment{
			Text:  tr.ChatDone,
			Style: widget.RichTextStyle{ColorName: theme.ColorNameDisabled},
		}}
	}
	c.status.Refresh()
}

// showBlockedHint switches the status line to a hint when the user tries to
// send while the agent is still responding; the input text is left intact so
// nothing typed is lost.
func (c *ChatArea) showBlockedHint() {
	c.status.Segments = []widget.RichTextSegment{&widget.TextSegment{
		Text:  config.Tr().ChatBlocked,
		Style: widget.RichTextStyle{ColorName: theme.ColorNamePrimary, TextStyle: fyne.TextStyle{Bold: true}},
	}}
	c.status.Refresh()
}

// setCancelledUI marks the status line to show the response was cancelled.
func (c *ChatArea) setCancelledUI() {
	c.status.Segments = []widget.RichTextSegment{&widget.TextSegment{
		Text:  config.Tr().ChatCancelled,
		Style: widget.RichTextStyle{ColorName: theme.ColorNameWarning, TextStyle: fyne.TextStyle{Bold: true}},
	}}
	c.status.Refresh()
}

// setQueuedUI marks the status line when /bty messages are queued behind the
// current response; the queued count is included so the state is visible.
func (c *ChatArea) setQueuedUI() {
	c.status.Segments = []widget.RichTextSegment{&widget.TextSegment{
		Text:  fmt.Sprintf("%s (%d)", config.Tr().ChatQueued, len(c.pending)),
		Style: widget.RichTextStyle{ColorName: theme.ColorNamePrimary, TextStyle: fyne.TextStyle{Bold: true}},
	}}
	c.status.Refresh()
}

// setErrorUI marks the status line to show a failed request.
func (c *ChatArea) setErrorUI(msg string) {
	c.status.Segments = []widget.RichTextSegment{&widget.TextSegment{
		Text:  config.Tr().ChatErrorPrefix + msg,
		Style: widget.RichTextStyle{ColorName: theme.ColorNameError, TextStyle: fyne.TextStyle{Bold: true}},
	}}
	c.status.Refresh()
}

// showError surfaces a failed agent request to the user: a red error banner in
// the message list plus a warning status line, so failures like an invalid API
// key are visible instead of silently disappearing into the log.
func (c *ChatArea) showError(err error) {
	if err == nil {
		return
	}
	msg := err.Error()
	if len(msg) > 300 {
		msg = msg[:300] + "..."
	}
	logging.Warn("chatarea: agent error", "error", err)
	c.errState.Store(true)
	c.errMsg = msg
	fyne.Do(func() {
		label := widget.NewLabel(msg)
		label.Wrapping = fyne.TextWrapWord
		label.TextStyle = fyne.TextStyle{Bold: true}
		c.output.Add(label)
		c.scroll.ScrollToBottom()
		c.setErrorUI(msg)
	})
}

// CancelResponse aborts the running agent response, if any. It is wired to the
// Esc key while the input has focus (see chatinput.go). The agent marks the
// in-flight message with FinishReasonCanceled; once its event channel closes
// the status line switches to the cancelled state.
func (c *ChatArea) CancelResponse() {
	if !c.streaming.Load() || c.cancelFunc == nil {
		return
	}
	logging.Info("chatarea: cancelling running response")
	c.cancelled.Store(true)
	c.cancelFunc()
}

// remember appends a sent message to the input history, skipping an exact
// repeat of the previous entry, and resets any in-progress navigation.
func (c *ChatArea) remember(text string) {
	if n := len(c.history); n > 0 && c.history[n-1] == text {
		return
	}
	c.history = append(c.history, text)
	c.historyIdx = -1
	c.historyDraft = ""
}

// historyUp recalls the previous sent message into the input; the current
// draft is kept so historyDown can restore it. The history is backed by the
// database, so it survives restarts and spans sessions; it is loaded lazily
// on first use.
func (c *ChatArea) historyUp() {
	if !c.historyLoaded {
		c.loadHistory()
	}
	logging.Debug("chatarea: history up", "len", len(c.history), "idx", c.historyIdx)
	if len(c.history) == 0 {
		return
	}
	if c.historyIdx < 0 {
		c.historyDraft = c.input.Text()
		c.historyIdx = len(c.history) - 1
	} else if c.historyIdx > 0 {
		c.historyIdx--
	} else {
		return
	}
	c.input.SetText(c.history[c.historyIdx])
}

// loadHistory merges the persisted user messages from the database with the
// in-memory entries recorded this run, keeps at most 100 newest unique texts
// and marks the history as loaded.
func (c *ChatArea) loadHistory() {
	c.historyLoaded = true
	var merged []string
	seen := make(map[string]bool)
	for _, text := range c.dbUserMessages() {
		if !seen[text] {
			seen[text] = true
			merged = append(merged, text)
		}
	}
	for _, text := range c.history {
		if !seen[text] {
			seen[text] = true
			merged = append(merged, text)
		}
	}
	if len(merged) > 100 {
		merged = merged[len(merged)-100:]
	}
	c.history = merged
	c.historyIdx = -1
	c.historyDraft = ""
	logging.Info("chatarea: history loaded", "count", len(c.history))
}

// dbUserMessages returns the text of every user message across all sessions,
// ordered oldest first.
func (c *ChatArea) dbUserMessages() []string {
	if c.core.Sessions == nil {
		return nil
	}
	type item struct {
		ts   int64
		text string
	}
	var items []item
	sessions, err := c.core.Sessions.List(c.ctx)
	if err != nil {
		logging.Warn("chatarea: history load sessions failed", "err", err)
		return nil
	}
	for _, sess := range sessions {
		msgs, err := c.core.Messages.List(c.ctx, sess.ID)
		if err != nil {
			logging.Debug("chatarea: history load session messages failed", "session", sess.ID, "err", err)
			continue
		}
		for _, m := range msgs {
			if m.Role != message.User {
				continue
			}
			for _, p := range m.Parts {
				if tc, ok := p.(message.TextContent); ok {
					if text := strings.TrimSpace(tc.Text); text != "" {
						items = append(items, item{m.CreatedAt, text})
					}
				}
			}
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ts < items[j].ts })
	texts := make([]string, len(items))
	for i, it := range items {
		texts[i] = it.text
	}
	return texts
}

// historyDown moves forward through the input history, restoring the draft
// after the newest entry.
func (c *ChatArea) historyDown() {
	if c.historyIdx < 0 {
		return
	}
	c.historyIdx++
	if c.historyIdx >= len(c.history) {
		c.historyIdx = -1
		c.input.SetText(c.historyDraft)
		return
	}
	c.input.SetText(c.history[c.historyIdx])
}

// scrollHome scrolls the message list to the top.
func (c *ChatArea) scrollHome() {
	c.scroll.Offset.Y = 0
	c.scroll.Refresh()
}

// scrollEnd scrolls the message list to the bottom.
func (c *ChatArea) scrollEnd() {
	maxY := c.scroll.Content.Size().Height - c.scroll.Size().Height
	c.scroll.Offset.Y = fyne.Max(0, maxY)
	c.scroll.Refresh()
}

// SetWindow gives the chat area the parent window for file dialogs.
func (c *ChatArea) SetWindow(win fyne.Window) {
	c.win = win
}

// SetOnSetup registers a callback invoked when the user needs to configure an
// LLM provider before sending a message.
func (c *ChatArea) SetOnSetup(fn func()) {
	c.onSetup = fn
}

// showProviderSetupPrompt 提示用户先配置 API Key，并拉起设置弹窗。
func (c *ChatArea) showProviderSetupPrompt() {
	if c.onSetup != nil {
		c.onSetup()
		return
	}
	if c.win != nil {
		dialog.ShowInformation("API Key 未配置", "请先在设置中配置 API Key。", c.win)
	}
}

func (c *ChatArea) PickAttachment() {
	if c.win == nil {
		logging.Warn("chatarea: no window for file dialog")
		return
	}
	d := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			logging.Error("file dialog error: %v", err)
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()
		data, err := io.ReadAll(reader)
		if err != nil {
			logging.Error("failed to read attachment: %v", err)
			return
		}
		c.attachments = append(c.attachments, message.Attachment{
			FilePath: reader.URI().Path(),
			FileName: filepath.Base(reader.URI().Path()),
			Content:  data,
		})
		c.updateAttachLabel()
	}, c.win)
	d.Show()
}

func (c *ChatArea) updateAttachLabel() {
	if len(c.attachments) == 0 {
		c.attachLabel.SetText("")
		return
	}
	names := make([]string, 0, len(c.attachments))
	for _, a := range c.attachments {
		names = append(names, a.FileName)
	}
	c.attachLabel.SetText(fmt.Sprintf("%d file(s): %s", len(c.attachments), strings.Join(names, ", ")))
}

func (c *ChatArea) SetOnSessionCreated(fn func(string)) {
	c.onSessionID = fn
}

func (c *ChatArea) SetCurrent(sessionID string) {
	logging.Info("chatarea: switch session", "from", c.current, "to", sessionID)
	c.current = sessionID
	c.errState.Store(false)
	c.errMsg = ""
	// Drop tool blocks from the previous session so a re-render does not
	// re-attach stale blocks (e.g. git output) to the new session's view.
	c.toolMu.Lock()
	stale := len(c.toolBlocks)
	c.toolBlocks = make(map[string]*ToolBlock)
	c.renderCache = make(map[string]fyne.CanvasObject)
	c.dirty = nil
	c.toolMu.Unlock()
	if stale > 0 {
		logging.Info("chatarea: dropped stale tool blocks", "count", stale)
	}
	c.sched.Schedule()
}

// Current returns the active session ID, or "" if none is selected.
func (c *ChatArea) Current() string {
	return c.current
}

// pageUp scrolls the chat output up by one viewport height.
func (c *ChatArea) pageUp() {
	viewH := c.scroll.Size().Height
	if viewH <= 0 {
		return
	}
	c.scroll.Offset.Y = fyne.Max(0, c.scroll.Offset.Y-viewH)
	c.scroll.Refresh()
}

// pageDown scrolls the chat output down by one viewport height.
func (c *ChatArea) pageDown() {
	viewH := c.scroll.Size().Height
	if viewH <= 0 {
		return
	}
	maxY := c.scroll.Content.Size().Height - viewH
	c.scroll.Offset.Y = fyne.Min(maxY, c.scroll.Offset.Y+viewH)
	c.scroll.Refresh()
}

func (c *ChatArea) Send() {
	logging.Info("send: entered", "streaming", c.streaming.Load())
	text := strings.TrimSpace(c.input.Text())
	if c.streaming.Load() {
		if strings.HasPrefix(text, "/bty") && len(text) > len("/bty") {
			content := strings.TrimSpace(strings.TrimPrefix(text, "/bty"))
			if content != "" {
				c.pending = append(c.pending, content)
				c.input.SetText("")
				c.setQueuedUI()
				logging.Info("send: queued bty", "content", content)
				return
			}
		}
		c.showBlockedHint()
		return
	}
	if text == "" {
		logging.Info("send: empty text, ignored")
		return
	}
	c.sendText(text, true)
}

// sendText starts an agent turn; clearInput=false preserves the user's typed text (bty flush).
func (c *ChatArea) sendText(text string, clearInput bool) {
	if clearInput {
		c.input.SetText("")
	}
	logging.Info("send: text", "text", text)

	if c.core.CoderAgent == nil {
		logging.Warn("send: coder agent not configured, prompting provider setup")
		c.showProviderSetupPrompt()
		return
	}

	sessionID := c.current
	if sessionID == "" {
		sess, err := c.core.Sessions.Create(c.ctx, "New Session")
		if err != nil {
			logging.Error("failed to create session: %v", err)
			return
		}
		sessionID = sess.ID
		c.current = sessionID
		logging.Info("send: created session", "id", sessionID)
		if c.onSessionID != nil {
			c.onSessionID(sessionID)
		}
	}

	// The user message is persisted by agent.Run itself (createUserMessage),
	// so creating it here as well would store it twice and render it twice.
	c.streaming.Store(true)
	c.cancelled.Store(false)
	runCtx, cancel := context.WithCancel(c.ctx)
	attachments := c.attachments
	c.attachments = nil
	c.updateAttachLabel()
	evCh, err := c.core.CoderAgent.Run(runCtx, sessionID, text, attachments...)
	if err != nil {
		cancel()
		c.streaming.Store(false)
		c.setStreamingUI(false)
		logging.Error("failed to start agent run: %v", err)
		return
	}
	c.cancelFunc = cancel
	c.remember(text)
	c.setStreamingUI(true)
	logging.Info("send: agent run started")
	go func() {
		defer logging.RecoverPanic("consume-agent-events", nil)
		defer fyne.Do(func() {
			c.streaming.Store(false)
			if c.cancelled.Load() {
				c.pending = nil
				c.setCancelledUI()
			} else if c.errState.Load() {
				c.pending = nil
				c.setErrorUI(c.errMsg)
			} else {
				c.setStreamingUI(false)
				c.flushPending()
			}
		})
		for range evCh {
		}
	}()
	c.sched.Schedule()
}

// flushPending sends queued /bty messages, one per finished turn, so a burst
// of /bty inputs drains one-by-one after each response completes.
func (c *ChatArea) flushPending() {
	if len(c.pending) == 0 {
		return
	}
	text := c.pending[0]
	c.pending = c.pending[1:]
	logging.Info("send: flushing queued bty", "content", text)
	c.sendText(text, false)
}

func (c *ChatArea) OnMessageEvent(ev pubsub.Event[message.Message]) {
	if ev.Payload.SessionID == c.current {
		// Mark the message dirty so renderNow re-renders just this message
		// while reusing cached views for the unchanged ones. Streaming text
		// updates the same message repeatedly (agent.Update), so each update
		// dirties only its own ID, keeping re-render cost O(1) per tick.
		c.toolMu.Lock()
		if c.dirty == nil {
			c.dirty = make(map[string]bool)
		}
		c.dirty[ev.Payload.ID] = true
		c.toolMu.Unlock()
		c.sched.Schedule()
	}
}

func (c *ChatArea) OnAgentEvent(ev pubsub.Event[agent.AgentEvent]) {
	e := ev.Payload
	if e.Type == agent.AgentEventTypeError {
		c.showError(e.Error)
		return
	}
	if e.Type == agent.AgentEventTypeResponse && e.Done {
		// Rebuild every view from the final message state, exactly like
		// switching away and back to the session: the streaming render only
		// refreshed dirty messages, leaving stale cached tool blocks behind.
		c.toolMu.Lock()
		c.toolBlocks = make(map[string]*ToolBlock)
		c.renderCache = make(map[string]fyne.CanvasObject)
		c.dirty = nil
		c.toolMu.Unlock()
		c.sched.Schedule()
		return
	}
	if e.Type == agent.AgentEventTypeToolUse {
		c.toolMu.Lock()
		defer c.toolMu.Unlock()
		switch {
		case e.ToolStart:
			if e.ToolCallID == "" {
				return
			}
			if _, ok := c.toolBlocks[e.ToolCallID]; !ok {
				var block *ToolBlock
				if isCodeTool(e.ToolName) {
					block = NewCompactToolBlock(e.ToolName)
				} else {
					block = NewToolBlock(e.ToolName)
					if e.ToolInput != "" {
						block.SetOutput(summarizeToolInput(e.ToolName, e.ToolInput))
					}
				}
				c.toolBlocks[e.ToolCallID] = block
				c.output.Add(block)
				c.output.Refresh()
				c.scroll.ScrollToBottom()
			}
		case e.StreamContent != "":
			if block, ok := c.toolBlocks[e.ToolCallID]; ok {
				block.AppendOutput(e.StreamContent)
				if !block.noOutput {
					c.scroll.ScrollToBottom()
				}
			}
		case !e.ToolStart:
			// Tool finished; keep the live block in the map so the next
			// re-render's ToolResult part can reuse it (avoiding a duplicate
			// block), then re-render.
			c.sched.Schedule()
		}
	}
}

func (c *ChatArea) renderNow() {
	if c.current == "" {
		return
	}
	start := time.Now()
	msgs, err := c.core.Messages.List(c.ctx, c.current)
	if err != nil {
		logging.Error("failed to list messages: %v", err)
		return
	}
	logging.Debug("chatarea: renderNow", "session", c.current, "msgs", len(msgs))

	const maxMessages = 100
	if len(msgs) > maxMessages {
		msgs = msgs[len(msgs)-maxMessages:]
	}

	c.toolMu.Lock()
	active := c.toolBlocks
	c.toolBlocks = make(map[string]*ToolBlock)
	dirty := c.dirty
	c.dirty = nil
	cache := c.renderCache
	c.toolMu.Unlock()

	views := make([]fyne.CanvasObject, 0, len(msgs)*2+2)
	doneTools := make(map[string]bool)
	for _, m := range msgs {
		if m.Role != message.Tool {
			continue
		}
		for _, p := range m.Parts {
			if tr, ok := p.(message.ToolResult); ok && tr.ToolCallID != "" && isCodeTool(tr.Name) {
				doneTools[tr.ToolCallID] = true
			}
		}
	}
	seen := make(map[string]bool, len(msgs))
	// Find the last assistant message index so intermediate tool-use
	// rounds can be tagged as compact-mode (tool blocks keep the title
	// line so the user sees which tool ran, but the output area is hidden
	// to avoid flooding the conversation with intermediate command output).
	// Compact only applies when there are multiple assistant rounds; a
	// single round with tools shows everything inline.
	lastAssistant := -1
	assistantCount := 0
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == message.Assistant {
			assistantCount++
			if lastAssistant < 0 {
				lastAssistant = i
			}
		}
	}
	hasMultipleRounds := assistantCount > 1
	for i, m := range msgs {
		seen[m.ID] = true
		compact := false
		if hasMultipleRounds {
			compact = m.Role == message.Assistant && i != lastAssistant
		}
		if !dirty[m.ID] && !hasToolCallParts(m) {
			if v, ok := cache[m.ID]; ok {
				views = append(views, v)
				continue
			}
		}
		view, used := renderMessage(m, active, doneTools, compact)
		cache[m.ID] = view
		for id, block := range used {
			if block == nil {
				continue
			}
			c.toolMu.Lock()
			c.toolBlocks[id] = block
			c.toolMu.Unlock()
		}
		views = append(views, view)
	}
	for id := range cache {
		if !seen[id] {
			delete(cache, id)
		}
	}
	// Blocks that were not consumed by a persisted ToolCall part are still
	// running. Attach each one right after the view of the assistant message
	// that issued the call instead of the list tail: appending at the end
	// pushed live tool output below the newest streamed answer, and map
	// iteration made several running blocks shuffle on every tick. Blocks
	// whose owning message was trimmed away fall back to the tail.
	toolSeq := make(map[string]int)
	{
		seq := 0
		for _, m := range msgs {
			if m.Role != message.Assistant {
				continue
			}
			for _, p := range m.Parts {
				if tc, ok := p.(message.ToolCall); ok {
					toolSeq[tc.ID] = seq
					seq++
				}
			}
		}
	}
	type pendingBlock struct {
		order int
		id    string
		block fyne.CanvasObject
	}
	var pendings []pendingBlock
	c.toolMu.Lock()
	const maxReattach = 20
	for id, block := range active {
		if block == nil {
			continue
		}
		if _, consumed := c.toolBlocks[id]; consumed {
			continue
		}
		if len(pendings) >= maxReattach {
			break
		}
		order, owned := toolSeq[id]
		if !owned {
			order = len(msgs)
		}
		pendings = append(pendings, pendingBlock{order, id, block})
		c.toolBlocks[id] = block
	}
	c.toolMu.Unlock()
	sort.Slice(pendings, func(i, j int) bool {
		if pendings[i].order != pendings[j].order {
			return pendings[i].order < pendings[j].order
		}
		return pendings[i].id < pendings[j].id
	})
	if len(pendings) > 0 {
		merged := make([]fyne.CanvasObject, 0, len(views)+len(pendings))
		pi := 0
		for i, v := range views {
			merged = append(merged, v)
			for pi < len(pendings) && pendings[pi].order == i {
				merged = append(merged, pendings[pi].block)
				pi++
			}
		}
		for ; pi < len(pendings); pi++ {
			merged = append(merged, pendings[pi].block)
		}
		views = merged
		logging.Debug("chatarea: renderNow reattached running blocks", "count", len(pendings))
	}

	c.output.Objects = views
	c.output.Refresh()
	// Layout recomputes on the next paint, so a direct ScrollToBottom here
	// would use the stale Content size; queue a second one to land at the
	// true bottom once the new height is known.
	c.scroll.ScrollToBottom()
	fyne.Do(c.scroll.ScrollToBottom)
	if min := c.output.MinSize(); min.Width > 900 || min.Height > 6000 {
		logging.Warn("chatarea: wide/tall content", "session", c.current, "views", len(views),
			"min_w", min.Width, "min_h", min.Height, "took", time.Since(start))
	}
	logging.Debug("chatarea: renderNow done", "session", c.current, "views", len(views), "took", time.Since(start))
}

// hasToolCallParts reports whether the message carries a ToolCall part. Such
// messages are never cached across re-renders: whether the block is drawn
// inside them depends on doneTools (result availability), which changes as
// tool results arrive, so a stale cached view would duplicate the block that
// the ToolResult message renders.
func hasToolCallParts(m message.Message) bool {
	for _, p := range m.Parts {
		if _, ok := p.(message.ToolCall); ok {
			return true
		}
	}
	return false
}
