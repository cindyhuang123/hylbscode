package gui

import (
	"context"
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/cindyhuang123/hylbscode/internal/app"
	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/logging"
	"github.com/cindyhuang123/hylbscode/internal/pubsub"
	"github.com/cindyhuang123/hylbscode/internal/session"
)

type SessionSidebar struct {
	core     *app.App
	ctx      context.Context
	mu       sync.Mutex
	sessions []session.Session
	current  string
	list     *widget.List
	onSelect func(sessionID string)
	onDelete func(sessionID string)
	win      fyne.Window
}

func NewSessionSidebar(core *app.App, ctx context.Context, onSelect func(string)) *SessionSidebar {
	v := &SessionSidebar{
		core:     core,
		ctx:      ctx,
		onSelect: onSelect,
	}
	if sessions, err := core.Sessions.List(ctx); err == nil {
		for _, s := range sessions {
			if s.ParentSessionID == "" {
				v.sessions = append(v.sessions, s)
			}
		}
	}
	v.list = widget.NewList(
		v.length,
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			s := v.itemAt(int(id))
			label := obj.(*widget.Label)
			label.SetText(s.Title)
			if s.ID == v.current {
				label.TextStyle = fyne.TextStyle{Bold: true}
			} else {
				label.TextStyle = fyne.TextStyle{}
			}
		},
	)
	v.list.OnSelected = func(id widget.ListItemID) {
		if v.onSelect != nil {
			v.onSelect(v.itemAt(int(id)).ID)
		}
	}
	return v
}

func (v *SessionSidebar) length() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return len(v.sessions)
}

func (v *SessionSidebar) itemAt(i int) session.Session {
	v.mu.Lock()
	defer v.mu.Unlock()
	if i < 0 || i >= len(v.sessions) {
		return session.Session{}
	}
	return v.sessions[i]
}

func (v *SessionSidebar) SetCurrent(id string) {
	v.mu.Lock()
	v.current = id
	v.mu.Unlock()
	v.list.Refresh()
}

func (v *SessionSidebar) OnSessionEvent(ev pubsub.Event[session.Session]) {
	s := ev.Payload
	if s.ParentSessionID != "" {
		return
	}
	v.mu.Lock()
	switch ev.Type {
	case pubsub.CreatedEvent:
		v.insertNewLocked(s)
	case pubsub.UpdatedEvent:
		if !v.upsertLocked(s) {
			logging.Info("sidebar: updated session not in list, inserting at top", "id", s.ID, "title", s.Title)
			v.insertNewLocked(s)
		}
	case pubsub.DeletedEvent:
		v.removeLocked(s.ID)
	}
	v.mu.Unlock()
	v.list.Refresh()
}

// upsertLocked replaces an existing session entry; it reports false when the
// session is not in the list yet (creation event raced with a title update).
func (v *SessionSidebar) upsertLocked(s session.Session) bool {
	for i, existing := range v.sessions {
		if existing.ID == s.ID {
			v.sessions[i] = s
			return true
		}
	}
	return false
}

// insertNewLocked puts a newly created session at the top of the list so the
// most recent session is always visible first.
func (v *SessionSidebar) insertNewLocked(s session.Session) {
	for i, existing := range v.sessions {
		if existing.ID == s.ID {
			v.sessions[i] = s
			return
		}
	}
	v.sessions = append([]session.Session{s}, v.sessions...)
}

func (v *SessionSidebar) removeLocked(id string) {
	for i, existing := range v.sessions {
		if existing.ID == id {
			v.sessions = append(v.sessions[:i], v.sessions[i+1:]...)
			return
		}
	}
}

func (v *SessionSidebar) Content() fyne.CanvasObject {
	newBtn := widget.NewButton(config.Tr().GUINewSession, v.createNew)
	delBtn := widget.NewButton(config.Tr().GUIDelete, v.deleteCurrent)
	row := container.NewHBox(newBtn, delBtn)
	return container.NewBorder(row, nil, nil, nil, v.list)
}

// SetOnDelete registers a callback fired after a session is deleted so the
// caller can clear any state tied to it.
func (v *SessionSidebar) SetOnDelete(fn func(string)) {
	v.onDelete = fn
}

// SetWindow gives the sidebar the parent window for confirmation dialogs.
func (v *SessionSidebar) SetWindow(win fyne.Window) {
	v.win = win
}

func (v *SessionSidebar) createNew() {
	sess, err := v.core.Sessions.Create(v.ctx, "New Session")
	if err != nil {
		logging.Error("Failed to create session: %v", err)
		return
	}
	if v.onSelect != nil {
		v.onSelect(sess.ID)
	}
}

func (v *SessionSidebar) deleteCurrent() {
	v.mu.Lock()
	id := v.current
	title := ""
	for _, s := range v.sessions {
		if s.ID == id {
			title = s.Title
			break
		}
	}
	v.mu.Unlock()
	if id == "" {
		return
	}
	doDelete := func() {
		if err := v.core.Sessions.Delete(v.ctx, id); err != nil {
			logging.Error("Failed to delete session: %v", err)
			return
		}
		v.mu.Lock()
		if v.current == id {
			v.current = ""
		}
		v.mu.Unlock()
		if v.onDelete != nil {
			v.onDelete(id)
		}
	}
	if v.win == nil {
		doDelete()
		return
	}
	tr := config.Tr()
	dialog.ShowConfirm(tr.GUIDeleteSessionTitle, fmt.Sprintf(tr.GUIDeleteSessionMsg, title), func(confirmed bool) {
		if confirmed {
			doDelete()
		}
	}, v.win)
}
