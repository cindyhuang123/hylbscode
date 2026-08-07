package gui

import (
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/cindyhuang123/hylbscode/internal/session"
)

func TestSessionPanelSetSessionInfo(t *testing.T) {
	test.NewApp()
	p := NewSessionPanel(nil, nil)
	p.SetSessionInfo(session.Session{
		ID:    "s1",
		Title: "Test",
	})
	if got := p.title.Text; got != "Test" {
		t.Fatalf("expected title, got %q", got)
	}
}

func TestSessionPanelEmptySession(t *testing.T) {
	test.NewApp()
	p := NewSessionPanel(nil, nil)
	p.SetSession("")
	if got := p.title.Text; got != "-" {
		t.Fatalf("expected placeholder title, got %q", got)
	}
}
