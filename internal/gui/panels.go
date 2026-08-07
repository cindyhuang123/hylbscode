package gui

import (
	"context"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/cindyhuang123/hylbscode/internal/app"
	"github.com/cindyhuang123/hylbscode/internal/logging"
	"github.com/cindyhuang123/hylbscode/internal/session"
)

// SessionPanel shows the title of the currently selected session in a single
// line.
type SessionPanel struct {
	core  *app.App
	ctx   context.Context
	mu    sync.Mutex
	sess  session.Session
	title *widget.Label
}

func NewSessionPanel(core *app.App, ctx context.Context) *SessionPanel {
	p := &SessionPanel{core: core, ctx: ctx}
	p.title = widget.NewLabel("-")
	p.title.Wrapping = fyne.TextWrapWord
	return p
}

// SetSession reloads the panel for the given session ID.
func (p *SessionPanel) SetSession(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if id == "" {
		p.sess = session.Session{}
		p.title.SetText("-")
		return
	}
	sess, err := p.core.Sessions.Get(p.ctx, id)
	if err != nil {
		logging.Error("failed to get session: %v", err)
		return
	}
	p.sess = sess
	p.title.SetText(sess.Title)
}

// SetSessionInfo updates the panel from a session event without a DB round-trip.
func (p *SessionPanel) SetSessionInfo(s session.Session) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sess = s
	p.title.SetText(s.Title)
}

func (p *SessionPanel) Content() fyne.CanvasObject {
	return p.title
}
