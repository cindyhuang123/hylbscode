package gui

import (
	"sync/atomic"
	"time"

	"github.com/cindyhuang123/hylbscode/internal/logging"
)

// renderScheduler coalesces bursts of refresh requests into one execution a
// fixed interval after the first request. Every burst is executed at least
// once; a pending request is never dropped (regression guard for the TUI
// chat-list debounce bug fixed in 199b7d6).
type renderScheduler struct {
	interval time.Duration
	do       func()
	pending  atomic.Bool
}

func newRenderScheduler(interval time.Duration, do func()) *renderScheduler {
	return &renderScheduler{interval: interval, do: do}
}

// Schedule starts an execution if none is pending, otherwise coalesces into
// the pending one. The action runs after the interval; it is never dropped.
func (s *renderScheduler) Schedule() {
	if !s.pending.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer logging.RecoverPanic("render-scheduler", nil)
		time.Sleep(s.interval)
		s.pending.Store(false)
		s.do()
	}()
}
