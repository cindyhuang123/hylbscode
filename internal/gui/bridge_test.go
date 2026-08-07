package gui

import (
	"context"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"

	"github.com/cindyhuang123/hylbscode/internal/pubsub"
)

// TestSubscribeDispatchesEvent verifies that a published event reaches the
// handler through the subscribe bridge (which routes it via fyne.Do).
func TestSubscribeDispatchesEvent(t *testing.T) {
	test.NewApp()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	ch := make(chan pubsub.Event[int], 1)
	done := make(chan struct{})
	var got int

	subscribe(ctx, &wg, "test", func(context.Context) <-chan pubsub.Event[int] {
		return ch
	}, func(ev pubsub.Event[int]) {
		got = ev.Payload
		close(done)
	})

	ch <- pubsub.Event[int]{Payload: 7}

	select {
	case <-done:
		if got != 7 {
			t.Fatalf("expected payload 7, got %d", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("event was not dispatched to the handler")
	}
}

// TestSubscribeStopsOnCancel verifies that cancelling the context terminates
// the subscription goroutine instead of leaking it.
func TestSubscribeStopsOnCancel(t *testing.T) {
	test.NewApp()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	subscribe(ctx, &wg, "test", func(context.Context) <-chan pubsub.Event[int] {
		return make(chan pubsub.Event[int])
	}, func(pubsub.Event[int]) {})

	cancel()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("subscription goroutine did not stop after cancel")
	}
}
