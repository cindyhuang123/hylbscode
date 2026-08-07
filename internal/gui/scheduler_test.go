package gui

import (
	"sync"
	"testing"
	"time"
)

func TestRenderSchedulerCoalescesBurst(t *testing.T) {
	var mu sync.Mutex
	count := 0
	s := newRenderScheduler(20*time.Millisecond, func() {
		mu.Lock()
		count++
		mu.Unlock()
	})
	for range 5 {
		s.Schedule()
	}
	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if count != 1 {
		t.Fatalf("expected 1 execution for a burst of 5, got %d", count)
	}
}

func TestRenderSchedulerExecutesAgainAfterCoalescedRun(t *testing.T) {
	var mu sync.Mutex
	count := 0
	s := newRenderScheduler(20*time.Millisecond, func() {
		mu.Lock()
		count++
		mu.Unlock()
	})
	s.Schedule()
	time.Sleep(60 * time.Millisecond)
	mu.Lock()
	first := count
	mu.Unlock()
	if first != 1 {
		t.Fatalf("expected 1 execution after first schedule, got %d", first)
	}
	s.Schedule()
	time.Sleep(60 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if count != 2 {
		t.Fatalf("expected 2 executions after second schedule, got %d", count)
	}
}

func TestRenderSchedulerNeverDropsSingleSchedule(t *testing.T) {
	var mu sync.Mutex
	count := 0
	s := newRenderScheduler(10*time.Millisecond, func() {
		mu.Lock()
		count++
		mu.Unlock()
	})
	s.Schedule()
	time.Sleep(80 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if count != 1 {
		t.Fatalf("expected scheduled execution to always run, got %d", count)
	}
}
