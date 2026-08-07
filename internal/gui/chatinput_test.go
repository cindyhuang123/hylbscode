package gui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func TestChatInputEnterSubmits(t *testing.T) {
	test.NewApp()

	var submitted string
	ci := NewChatInput(func(text string) { submitted = text }, nil)
	win := test.NewWindow(ci)
	defer win.Close()

	ci.SetText("hello world")
	ci.KeyDown(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if submitted != "hello world" {
		t.Fatalf("expected submit with %q, got %q", "hello world", submitted)
	}

	ci.SetText("second")
	ci.KeyDown(&fyne.KeyEvent{Name: fyne.KeyEnter})
	if submitted != "second" {
		t.Fatalf("expected submit with %q on KeyEnter, got %q", "second", submitted)
	}
}

func TestChatInputTypedKeySwallowsEnter(t *testing.T) {
	test.NewApp()

	called := false
	ci := NewChatInput(func(string) { called = true }, nil)
	win := test.NewWindow(ci)
	defer win.Close()

	ci.SetText("abc")
	ci.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if called {
		t.Fatal("TypedKey must not trigger submit (already handled in KeyDown)")
	}
	if ci.Text() != "abc" {
		t.Fatalf("TypedKey(Enter) must not mutate text, got %q", ci.Text())
	}
}

func TestChatInputTypedKeyDelegatesNonEnter(t *testing.T) {
	test.NewApp()

	ci := NewChatInput(func(string) {}, nil)
	win := test.NewWindow(ci)
	defer win.Close()

	ci.TypedRune('a')
	ci.TypedRune('b')
	if ci.Text() != "ab" {
		t.Fatalf("expected text %q, got %q", "ab", ci.Text())
	}
}

func TestChatInputFocus(t *testing.T) {
	test.NewApp()

	ci := NewChatInput(func(string) {}, nil)
	win := test.NewWindow(ci)
	defer win.Close()

	win.Canvas().Focus(ci)
	if got := win.Canvas().Focused(); got != ci {
		t.Fatalf("expected ChatInput to be focused, got %T", got)
	}
}

// TestChatEntryFocusVerifies super() wiring: clicking the input must be able
// to focus the inner entry (impl must point at chatEntry, not the embedded
// *widget.Entry, or requestFocus finds no canvas and focus never lands).
func TestChatEntryFocus(t *testing.T) {
	test.NewApp()

	ci := NewChatInput(func(string) {}, nil)
	win := test.NewWindow(ci)
	defer win.Close()

	win.Canvas().Focus(ci.entry)
	if got := win.Canvas().Focused(); got != ci.entry {
		t.Fatalf("expected chatEntry to be focused, got %T", got)
	}
}

// TestChatEntryEnterSubmits covers the focus-on-entry path: after clicking
// the input, focus stays on the inner entry (ChatInput.KeyDown never fires),
// so the entry itself must intercept Enter and submit.
func TestChatEntryEnterSubmits(t *testing.T) {
	test.NewApp()

	var submitted []string
	ci := NewChatInput(func(text string) { submitted = append(submitted, text) }, nil)
	win := test.NewWindow(ci)
	defer win.Close()

	ci.SetText("hello")
	win.Canvas().Focus(ci.entry)

	ci.entry.KeyDown(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if len(submitted) != 1 || submitted[0] != "hello" {
		t.Fatalf("expected 1 submit with %q, got %v", "hello", submitted)
	}

	// TypedKey follows every KeyDown (and repeats only hit TypedKey); it must
	// neither submit again nor insert a newline.
	ci.entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if len(submitted) != 1 {
		t.Fatalf("TypedKey must not re-submit, got %v", submitted)
	}
	if ci.Text() != "hello" {
		t.Fatalf("TypedKey must not insert newline, got %q", ci.Text())
	}
}

// TestChatEntryDelegatesTyping ensures normal editing still works while the
// entry holds focus.
func TestChatEntryDelegatesTyping(t *testing.T) {
	test.NewApp()

	ci := NewChatInput(func(string) {}, nil)
	win := test.NewWindow(ci)
	defer win.Close()

	win.Canvas().Focus(ci.entry)
	ci.entry.TypedRune('a')
	ci.entry.TypedRune('b')
	if ci.Text() != "ab" {
		t.Fatalf("expected text %q, got %q", "ab", ci.Text())
	}
}

// TestArrowKeysTriggerHistory verifies Up/Down fire the history handlers in
// single-line mode and are delegated to the entry for cursor movement in
// multi-line mode.
func TestArrowKeysTriggerHistory(t *testing.T) {
	test.NewApp()

	up, down := 0, 0
	ci := NewChatInput(func(string) {}, nil)
	ci.SetHistoryHandlers(func() { up++ }, func() { down++ })
	win := test.NewWindow(ci)
	defer win.Close()

	ci.SetText("single line")
	ci.entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	ci.entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	if up != 1 || down != 1 {
		t.Fatalf("expected history handlers fired, up=%d down=%d", up, down)
	}

	up, down = 0, 0
	ci.SetText("line1\nline2")
	ci.entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	ci.entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	if up != 0 || down != 0 {
		t.Fatalf("multi-line input must keep arrow keys, up=%d down=%d", up, down)
	}
}

// TestHomeEndScrollMessageList verifies Home/End fire the scroll handlers
// (targeting the output area) regardless of input content.
func TestHomeEndScrollMessageList(t *testing.T) {
	test.NewApp()

	home, end := 0, 0
	ci := NewChatInput(func(string) {}, nil)
	ci.SetPageHandlers(func() {}, func() {}, func() { home++ }, func() { end++ })
	win := test.NewWindow(ci)
	defer win.Close()

	ci.SetText("multi\nline")
	ci.entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyHome})
	ci.entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnd})
	if home != 1 || end != 1 {
		t.Fatalf("expected Home/End scroll handlers fired, home=%d end=%d", home, end)
	}
}

// TestEscTriggersCancel verifies Esc fires onCancel while the entry holds
// focus, and that it is swallowed (not forwarded to the underlying entry).
func TestEscTriggersCancel(t *testing.T) {
	test.NewApp()

	cancelled := 0
	ci := NewChatInput(func(string) {}, func() { cancelled++ })
	win := test.NewWindow(ci)
	defer win.Close()

	win.Canvas().Focus(ci.entry)
	ci.entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if cancelled != 1 {
		t.Fatalf("expected onCancel fired once, got %d", cancelled)
	}
	ci.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if cancelled != 2 {
		t.Fatalf("expected outer ChatInput path to cancel too, got %d", cancelled)
	}
	if ci.Text() != "" {
		t.Fatalf("Esc must not mutate text, got %q", ci.Text())
	}
}
