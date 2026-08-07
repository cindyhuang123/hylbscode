package gui

import (
	"context"
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/cindyhuang123/hylbscode/internal/app"
	"github.com/cindyhuang123/hylbscode/internal/pubsub"
	"github.com/cindyhuang123/hylbscode/internal/todo"
)

type fakeTodoService struct {
	statusCalls []string
}

func (f *fakeTodoService) List(ctx context.Context, sessionID string) ([]todo.Todo, error) {
	return nil, nil
}
func (f *fakeTodoService) BulkSet(ctx context.Context, sessionID string, todos []todo.Todo) ([]todo.Todo, error) {
	return nil, nil
}
func (f *fakeTodoService) SetStatus(ctx context.Context, id, status string) (todo.Todo, error) {
	f.statusCalls = append(f.statusCalls, status)
	return todo.Todo{}, nil
}
func (f *fakeTodoService) Delete(ctx context.Context, id string) error { return nil }
func (f *fakeTodoService) Subscribe(ctx context.Context) <-chan pubsub.Event[todo.Todo] {
	return make(chan pubsub.Event[todo.Todo])
}

func TestTodoPanelUpdateRendersItem(t *testing.T) {
	test.NewApp()
	fake := &fakeTodoService{}
	p := &TodoPanel{
		core: &app.App{Todos: fake},
		todos: []todo.Todo{
			{ID: "t1", Content: "implement feature", Status: "pending"},
			{ID: "t2", Content: "done task", Status: "completed"},
		},
	}
	label := widget.NewLabel("")
	check := widget.NewCheck("", nil)
	del := widget.NewButton("✕", nil)
	border := container.NewBorder(nil, nil, check, del, label)

	p.update(0, border)
	if label.Text != "implement feature" {
		t.Fatalf("expected item label, got %q", label.Text)
	}
	if check.Checked {
		t.Fatal("expected pending todo unchecked")
	}
	check.SetChecked(true)
	if len(fake.statusCalls) != 1 || fake.statusCalls[0] != "completed" {
		t.Fatalf("expected SetStatus(completed), got %v", fake.statusCalls)
	}

	p.update(1, border)
	if label.Text != "done task" {
		t.Fatalf("expected updated label, got %q", label.Text)
	}
	if !check.Checked {
		t.Fatal("expected completed todo checked")
	}
	check.SetChecked(false)
	if len(fake.statusCalls) != 2 || fake.statusCalls[1] != "pending" {
		t.Fatalf("expected SetStatus(pending), got %v", fake.statusCalls)
	}
}

func TestTodoPanelEmptyPlaceholderVisibility(t *testing.T) {
	test.NewApp()
	p := NewTodoPanel(nil, nil)
	p.sessionID = "s1"
	p.refreshVisibility()
	if p.empty.Hidden {
		t.Fatal("expected placeholder visible when no todos")
	}
	p.mu.Lock()
	p.todos = []todo.Todo{{ID: "t1", Content: "x"}}
	p.mu.Unlock()
	p.refreshVisibility()
	if !p.empty.Hidden {
		t.Fatal("expected placeholder hidden when todos exist")
	}
}
