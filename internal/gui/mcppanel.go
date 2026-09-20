package gui

import (
	"sort"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/llm/agent"
)

// McpPanel lists the configured MCP servers and their registered tools so the
// user can see at a glance what the model can call. Servers that failed to
// register show their connection error instead of a tool list.
type McpPanel struct {
	mu      sync.Mutex
	servers []agent.McpServerInfo
	list    *widget.List
	empty   *widget.Label
}

func NewMcpPanel() *McpPanel {
	p := &McpPanel{}
	p.list = widget.NewList(
		p.length,
		func() fyne.CanvasObject {
			name := widget.NewLabel("")
			name.TextStyle = fyne.TextStyle{Bold: true}
			detail := widget.NewLabel("")
			detail.Wrapping = fyne.TextWrapWord
			return container.NewVBox(name, detail)
		},
		p.update,
	)
	p.empty = widget.NewLabel(config.Tr().GUIMcpEmpty)
	p.empty.Alignment = fyne.TextAlignCenter
	p.empty.Hide()
	return p
}

func (p *McpPanel) length() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.servers)
}

func (p *McpPanel) itemAt(i int) agent.McpServerInfo {
	p.mu.Lock()
	defer p.mu.Unlock()
	if i < 0 || i >= len(p.servers) {
		return agent.McpServerInfo{}
	}
	return p.servers[i]
}

func (p *McpPanel) update(id widget.ListItemID, obj fyne.CanvasObject) {
	info := p.itemAt(int(id))
	box := obj.(*fyne.Container)
	box.Objects[0].(*widget.Label).SetText(info.Name)
	var detail string
	if info.Err != "" {
		detail = "✗ " + info.Err
	} else if len(info.Tools) == 0 {
		detail = strings.ToUpper(string(info.Type)) + ": (no tools)"
	} else {
		sorted := make([]string, len(info.Tools))
		copy(sorted, info.Tools)
		sort.Strings(sorted)
		detail = strings.ToUpper(string(info.Type)) + ": " + strings.Join(sorted, ", ")
	}
	box.Objects[1].(*widget.Label).SetText(detail)
}

// Reload re-reads the registered MCP servers from the agent package. Call it
// after the agent registers tools (e.g. after provider setup) or when the
// config changes.
func (p *McpPanel) Reload() {
	p.mu.Lock()
	p.servers = agent.ListMcpServers()
	p.mu.Unlock()
	p.refreshVisibility()
}

// refreshVisibility shows the list only when servers exist, otherwise the
// empty placeholder, so the two never overlap.
func (p *McpPanel) refreshVisibility() {
	empty := p.length() == 0
	p.empty.Hidden = !empty
	p.empty.Refresh()
	p.list.Hidden = empty
	p.list.Refresh()
}

func (p *McpPanel) Content() fyne.CanvasObject {
	placeholder := container.NewCenter(p.empty)
	return container.NewStack(p.list, placeholder)
}