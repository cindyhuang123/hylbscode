package gui

import (
	"context"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/cindyhuang123/hylbscode/internal/app"
	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/logging"
	"github.com/cindyhuang123/hylbscode/internal/pubsub"
	"github.com/cindyhuang123/hylbscode/internal/todo"
)

const maxTodoPanelItems = 200

// TodoPanel lists the todos of the current session. Checkbox toggles the
// status between pending/completed; a delete button removes an item.
type TodoPanel struct {
	core      *app.App
	ctx       context.Context
	sessionID string
	mu        sync.Mutex
	todos     []todo.Todo
	list      *widget.List
	empty     *widget.Label
}

func NewTodoPanel(core *app.App, ctx context.Context) *TodoPanel {
	p := &TodoPanel{core: core, ctx: ctx}
	p.list = widget.NewList(
		p.length,
		func() fyne.CanvasObject {
			check := widget.NewCheck("", nil)
			label := widget.NewLabel("")
			label.Wrapping = fyne.TextWrapWord
			del := widget.NewButton("✕", nil)
			return container.NewBorder(nil, nil, check, del, label)
		},
		p.update,
	)
	p.empty = widget.NewLabel(config.Tr().GUITodoEmpty)
	p.empty.Alignment = fyne.TextAlignCenter
	p.empty.Hide()
	return p
}

func (p *TodoPanel) length() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.todos)
}

func (p *TodoPanel) itemAt(i int) todo.Todo {
	p.mu.Lock()
	defer p.mu.Unlock()
	if i < 0 || i >= len(p.todos) {
		return todo.Todo{}
	}
	return p.todos[i]
}

func (p *TodoPanel) update(id widget.ListItemID, obj fyne.CanvasObject) {
	item := p.itemAt(int(id))
	border := obj.(*fyne.Container)
	label := border.Objects[0].(*widget.Label)
	check := border.Objects[1].(*widget.Check)
	del := border.Objects[2].(*widget.Button)

	label.SetText(item.Content)
	check.SetChecked(item.Status == "completed")
	check.OnChanged = func(on bool) {
		status := "pending"
		if on {
			status = "completed"
		}
		if _, err := p.core.Todos.SetStatus(p.ctx, item.ID, status); err != nil {
			logging.Error("failed to update todo status: %v", err)
		}
	}
	del.OnTapped = func() {
		if err := p.core.Todos.Delete(p.ctx, item.ID); err != nil {
			logging.Error("failed to delete todo: %v", err)
		}
	}
}

// SetSession switches the panel to another session and reloads its todos.
func (p *TodoPanel) SetSession(sessionID string) {
	p.mu.Lock()
	p.sessionID = sessionID
	p.mu.Unlock()
	p.reload()
}

func (p *TodoPanel) reload() {
	p.mu.Lock()
	sessionID := p.sessionID
	p.mu.Unlock()
	if sessionID == "" {
		p.mu.Lock()
		p.todos = nil
		p.mu.Unlock()
		p.refreshVisibility()
		return
	}
	todos, err := p.core.Todos.List(p.ctx, sessionID)
	if err != nil {
		logging.Error("failed to list todos: %v", err)
		return
	}
	p.mu.Lock()
	p.todos = todos
	p.mu.Unlock()
	p.refreshVisibility()
}

// refreshVisibility shows the list only when todos exist, otherwise the
// empty placeholder, so the two never overlap.
func (p *TodoPanel) refreshVisibility() {
	empty := p.length() == 0
	p.empty.Hidden = !empty
	p.empty.Refresh()
	p.list.Hidden = empty
	p.list.Refresh()
}

// OnTodoEvent keeps the panel in sync with todo service events for the
// current session.
func (p *TodoPanel) OnTodoEvent(ev pubsub.Event[todo.Todo]) {
	p.mu.Lock()
	sessionID := p.sessionID
	p.mu.Unlock()
	if ev.Payload.SessionID != sessionID {
		return
	}
	p.reload()
}

func (p *TodoPanel) Content() fyne.CanvasObject {
	placeholder := container.NewCenter(p.empty)
	return container.NewStack(p.list, placeholder)
}
