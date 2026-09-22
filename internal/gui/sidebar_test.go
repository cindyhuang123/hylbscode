package gui

import (
	"context"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/cindyhuang123/hylbscode/internal/app"
	"github.com/cindyhuang123/hylbscode/internal/pubsub"
	"github.com/cindyhuang123/hylbscode/internal/session"
)

// fakeSessionService is an in-memory session.Service for sidebar tests.
type fakeSessionService struct {
	byID   map[string]session.Session
	saves  int
}

func newFakeSessionService(sessions ...session.Session) *fakeSessionService {
	f := &fakeSessionService{byID: make(map[string]session.Session)}
	for _, s := range sessions {
		f.byID[s.ID] = s
	}
	return f
}

func (f *fakeSessionService) Subscribe(ctx context.Context) <-chan pubsub.Event[session.Session] {
	return make(chan pubsub.Event[session.Session])
}
func (f *fakeSessionService) Create(ctx context.Context, title string) (session.Session, error) {
	return session.Session{}, nil
}
func (f *fakeSessionService) CreateTitleSession(ctx context.Context, parentSessionID string) (session.Session, error) {
	return session.Session{}, nil
}
func (f *fakeSessionService) CreateTaskSession(ctx context.Context, toolCallID, parentSessionID, title string) (session.Session, error) {
	return session.Session{}, nil
}
func (f *fakeSessionService) Get(ctx context.Context, id string) (session.Session, error) {
	return f.byID[id], nil
}
func (f *fakeSessionService) List(ctx context.Context) ([]session.Session, error) {
	out := make([]session.Session, 0, len(f.byID))
	for _, s := range f.byID {
		out = append(out, s)
	}
	return out, nil
}
func (f *fakeSessionService) Save(ctx context.Context, s session.Session) (session.Session, error) {
	f.saves++
	f.byID[s.ID] = s
	return s, nil
}
func (f *fakeSessionService) Delete(ctx context.Context, id string) error {
	delete(f.byID, id)
	return nil
}

// TestSessionRowTapDispatch verifies a single tap fires the select callback
// and a double tap fires the rename callback.
func TestSessionRowTapDispatch(t *testing.T) {
	test.NewApp()
	var tapped, doubleTapped bool
	row := newSessionRow("title", func() { tapped = true }, func() { doubleTapped = true })

	row.Tapped(&fyne.PointEvent{})
	if !tapped {
		t.Fatal("expected single tap to call onTap")
	}
	if doubleTapped {
		t.Fatal("single tap must not trigger double tap")
	}

	row.DoubleTapped(&fyne.PointEvent{})
	if !doubleTapped {
		t.Fatal("expected double tap to call onDoubleTap")
	}
}

// TestSessionRowTextAndBold verifies the label text and bold style are
// available for the list update callback to mutate.
func TestSessionRowTextAndBold(t *testing.T) {
	test.NewApp()
	row := newSessionRow("old", nil, nil)
	row.SetText("new title")
	if row.Text != "new title" {
		t.Fatalf("expected updated text, got %q", row.Text)
	}
	row.TextStyle = fyne.TextStyle{Bold: true}
	if !row.TextStyle.Bold {
		t.Fatal("expected bold style to be set")
	}
}

// TestApplyRenamePersists verifies a non-empty, changed title is saved and
// the session list is the source of truth for the new title.
func TestApplyRenamePersists(t *testing.T) {
	test.NewApp()
	svc := newFakeSessionService(session.Session{ID: "s1", Title: "Old"})
	core := &app.App{Sessions: svc}
	v := NewSessionSidebar(core, context.Background(), nil)

	if !v.applyRename("s1", "  New Title  ") {
		t.Fatal("expected rename to apply")
	}
	got, _ := svc.Get(context.Background(), "s1")
	if got.Title != "New Title" {
		t.Fatalf("expected trimmed title saved, got %q", got.Title)
	}
}

// TestApplyRenameNoop verifies empty, unchanged, and unknown sessions do not
// trigger a save.
func TestApplyRenameNoop(t *testing.T) {
	test.NewApp()
	svc := newFakeSessionService(session.Session{ID: "s1", Title: "Same"})
	core := &app.App{Sessions: svc}
	v := NewSessionSidebar(core, context.Background(), nil)

	for _, tc := range []struct {
		id, name string
	}{
		{"s1", ""},
		{"s1", "  "},
		{"s1", "Same"},
		{"nope", "Anything"},
	} {
		if v.applyRename(tc.id, tc.name) {
			t.Fatalf("expected no-op for id=%q name=%q", tc.id, tc.name)
		}
	}
	if svc.saves != 0 {
		t.Fatalf("expected no saves, got %d", svc.saves)
	}
}