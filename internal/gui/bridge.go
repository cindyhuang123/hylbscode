package gui

import (
	"context"
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"

	"github.com/cindyhuang123/hylbscode/internal/history"
	"github.com/cindyhuang123/hylbscode/internal/logging"
	"github.com/cindyhuang123/hylbscode/internal/pubsub"
)

// subscribe runs handler on the Fyne main thread for every event from sub.
func subscribe[T any](
	ctx context.Context,
	wg *sync.WaitGroup,
	name string,
	subscribeFn func(context.Context) <-chan pubsub.Event[T],
	handler func(pubsub.Event[T]),
) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer logging.RecoverPanic(fmt.Sprintf("subscription-%s", name), nil)

		subCh := subscribeFn(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-subCh:
				if !ok {
					return
				}
				fyne.Do(func() {
					defer logging.RecoverPanic(fmt.Sprintf("handler-%s", name), nil)
					handler(event)
				})
			}
		}
	}()
}

// SetupSubscriptions wires the core services' events to the GUI. It returns
// a cancel func that stops all subscription goroutines.
func SetupSubscriptions(g *MainWindow, parentCtx context.Context) func() {
	ctx, cancel := context.WithCancel(parentCtx)
	var wg sync.WaitGroup

	subscribe(ctx, &wg, "history", g.core.History.Subscribe, func(ev pubsub.Event[history.File]) {
		// M2: file history panel
	})
	subscribe(ctx, &wg, "sessions", g.core.Sessions.Subscribe, g.onSessionEvent)
	subscribe(ctx, &wg, "messages", g.core.Messages.Subscribe, g.chat.OnMessageEvent)
	subscribe(ctx, &wg, "permissions", g.core.Permissions.Subscribe, g.showPermission)
	subscribe(ctx, &wg, "todos", g.core.Todos.Subscribe, g.todo.OnTodoEvent)
	subscribe(ctx, &wg, "coderAgent", g.core.CoderAgent.Subscribe, g.chat.OnAgentEvent)

	return func() {
		logging.Info("Cancelling all GUI subscriptions")
		cancel()
		done := make(chan struct{})
		go func() {
			defer logging.RecoverPanic("subscription-cleanup", nil)
			wg.Wait()
			close(done)
		}()
		select {
		case <-done:
			logging.Info("All GUI subscription goroutines completed")
		case <-time.After(5 * time.Second):
			logging.Warn("Timed out waiting for GUI subscription goroutines")
		}
	}
}
