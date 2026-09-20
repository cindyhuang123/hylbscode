package gui

import (
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/cindyhuang123/hylbscode/internal/llm/agent"
)

func TestMcpPanelRendersServerWithTools(t *testing.T) {
	test.NewApp()
	p := NewMcpPanel()
	p.mu.Lock()
	p.servers = []agent.McpServerInfo{
		{Name: "stdio-server", Type: "stdio", Tools: []string{"b", "a_tool"}},
	}
	p.mu.Unlock()
	p.refreshVisibility()

	if len(p.servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(p.servers))
	}
	box := container.NewVBox(widget.NewLabel(""), widget.NewLabel(""))
	p.update(0, box)

	name := box.Objects[0].(*widget.Label).Text
	if name != "stdio-server" {
		t.Fatalf("expected server name label, got %q", name)
	}
	detail := box.Objects[1].(*widget.Label).Text
	if detail != "STDIO: a_tool, b" {
		t.Fatalf("expected sorted tool list, got %q", detail)
	}
}

func TestMcpPanelRendersServerWithError(t *testing.T) {
	test.NewApp()
	p := NewMcpPanel()
	p.mu.Lock()
	p.servers = []agent.McpServerInfo{
		{Name: "sse-server", Type: "sse", Err: "connect: refused"},
	}
	p.mu.Unlock()
	p.refreshVisibility()

	box := container.NewVBox(widget.NewLabel(""), widget.NewLabel(""))
	p.update(0, box)

	detail := box.Objects[1].(*widget.Label).Text
	if detail != "✗ connect: refused" {
		t.Fatalf("expected error detail, got %q", detail)
	}
}

func TestMcpPanelEmptyPlaceholderVisibility(t *testing.T) {
	test.NewApp()
	p := NewMcpPanel()
	p.refreshVisibility()
	if p.empty.Hidden {
		t.Fatal("expected placeholder visible when no servers")
	}
	p.mu.Lock()
	p.servers = []agent.McpServerInfo{{Name: "s1"}}
	p.mu.Unlock()
	p.refreshVisibility()
	if !p.empty.Hidden {
		t.Fatal("expected placeholder hidden when servers exist")
	}
}