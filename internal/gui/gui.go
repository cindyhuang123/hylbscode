package gui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/cindyhuang123/hylbscode/internal/app"
	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/llm/models"
	"github.com/cindyhuang123/hylbscode/internal/logging"
	"github.com/cindyhuang123/hylbscode/internal/pubsub"
	"github.com/cindyhuang123/hylbscode/internal/session"
	"github.com/cindyhuang123/hylbscode/internal/version"
)

type MainWindow struct {
	fyneApp fyne.App
	win     fyne.Window
	core    *app.App
	ctx     context.Context
	sidebar *SessionSidebar
	chat    *ChatArea
	todo    *TodoPanel

	sessionPanel  *SessionPanel
	status        *widget.Label
	contextLabel  *widget.Label
	costLabel     *widget.Label
	wdLabel       *widget.Label
	contextText   string
	costText      string
	sessionTokens int64
	outer         *container.Split
	inner         *container.Split

	themeAuto, themeLight, themeDark *fyne.MenuItem
	themeMenu                        *fyne.Menu
	viewMenu                         *fyne.Menu
	viewLeftItem, viewRightItem      *fyne.MenuItem
}

func NewMainWindow(fyneApp fyne.App, core *app.App, ctx context.Context) *MainWindow {
	g := &MainWindow{fyneApp: fyneApp, core: core, ctx: ctx}
	g.sidebar = NewSessionSidebar(core, ctx, g.selectSession)
	g.chat = NewChatArea(core, ctx)
	g.chat.SetOnSessionCreated(g.selectSession)
	g.todo = NewTodoPanel(core, ctx)
	g.sidebar.SetOnDelete(g.onSessionDeleted)
	g.sessionPanel = NewSessionPanel(core, ctx)

	g.win = fyneApp.NewWindow("HyLbsCode")
	g.win.SetCloseIntercept(g.requestQuit)
	g.chat.SetWindow(g.win)
	g.sidebar.SetWindow(g.win)
	g.buildLayout()

	cfg := config.Get()
	if w, h := cfg.GUI.Width, cfg.GUI.Height; w > 0 && h > 0 {
		g.win.Resize(fyne.NewSize(float32(w), float32(h)))
	} else {
		g.win.Resize(fyne.NewSize(1100, 720))
	}
	switch cfg.GUI.Theme {
	case "light":
		g.applyThemeFont(theme.LightTheme())
	case "dark":
		g.applyThemeFont(theme.DarkTheme())
	}
	return g
}

// buildLayout assembles the sidebar/chat/right-panel/status-bar layout and the
// main menu. It is called once at startup and again when the language changes
// so the UI text updates immediately.
func (g *MainWindow) buildLayout() {
	tr := config.Tr()

	// Border: top = session+todo headers, bottom = version, center = todo
	// list. The center slot stretches the List to fill the panel height; a
	// VBox would collapse it to a single row.
	versionLabel := widget.NewLabel(version.Version)
	right := container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle(tr.SessionLabel, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			g.sessionPanel.Content(),
			widget.NewLabelWithStyle(tr.GUITodoLabel, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		),
		versionLabel,
		nil, nil,
		g.todo.Content(),
	)
	rightPanel := right
	g.inner = container.NewHSplit(g.chat.Content(), rightPanel)
	g.inner.SetOffset(0.72)
	g.outer = container.NewHSplit(g.sidebar.Content(), g.inner)
	g.outer.SetOffset(0.2)

	if g.contextLabel == nil {
		g.status = widget.NewLabel(g.modelLabel())
		g.contextLabel = widget.NewLabel("")
		g.costLabel = widget.NewLabel("")
		g.wdLabel = widget.NewLabel("")
		g.contextText = "-"
		g.costText = "-"
	}
	g.refreshStatus()

	infoRow := container.NewHBox(g.contextLabel, g.costLabel, g.wdLabel, g.status)
	bar := container.NewBorder(nil, nil, nil, nil, infoRow)
	g.win.SetContent(container.NewBorder(nil, bar, nil, nil, g.outer))
	g.win.SetMainMenu(g.Menu())
}

// refreshStatus re-renders the status bar labels with the active language.
func (g *MainWindow) refreshStatus() {
	tr := config.Tr()
	wd := ""
	if cfg := config.Get(); cfg != nil {
		wd = cfg.WorkingDir
	}
	g.status.SetText(g.modelLabel())
	g.contextLabel.SetText(tr.ContextLabel + ": " + g.contextText)
	g.costLabel.SetText(tr.CostLabel + ": " + g.costText)
	g.wdLabel.SetText(tr.GUIWDLabel + ": " + wd)
}

// applyLanguage switches the UI language, persists it, and rebuilds the layout
// so the new language takes effect immediately.
func (g *MainWindow) applyLanguage(lang string) {
	logging.Info("menu: apply language", "lang", lang)
	if err := config.UpdateLanguage(lang); err != nil {
		logging.Error("failed to update language: %v", err)
		return
	}
	g.buildLayout()
}

func (g *MainWindow) modelLabel() string {
	tr := config.Tr()
	cfg := config.Get()
	modelID := cfg.Agents[config.AgentCoder].Model
	if m, ok := models.SupportedModels[modelID]; ok {
		return tr.GUIModelLabel + ": " + m.Name
	}
	return tr.GUIModelLabel + ": " + string(modelID)
}

func (g *MainWindow) contextSummary() string {
	cfg := config.Get()
	modelID := cfg.Agents[config.AgentCoder].Model
	window := int64(0)
	if m, ok := models.SupportedModels[modelID]; ok {
		window = m.ContextWindow
	}
	if window > 0 {
		return fmt.Sprintf("%d / %d tokens", g.sessionTokens, window)
	}
	return fmt.Sprintf("%d tokens", g.sessionTokens)
}

func (g *MainWindow) updateCost(s session.Session) {
	g.sessionTokens = s.PromptTokens
	g.contextText = g.contextSummary()
	// Session costs are normalized to CNY by the agent layer (cnyRate), so the
	// display reads them directly without an extra conversion.
	g.costText = fmt.Sprintf("¥%.2f", s.Cost)
	g.refreshStatus()
}

func (g *MainWindow) selectSession(sessionID string) {
	g.chat.SetCurrent(sessionID)
	g.sidebar.SetCurrent(sessionID)
	g.todo.SetSession(sessionID)
	g.sessionPanel.SetSession(sessionID)
	if sess, err := g.core.Sessions.Get(g.ctx, sessionID); err == nil {
		g.updateCost(sess)
	}
	// sidebar List steals focus after click; return it to input so ↑/↓ work
	g.chat.FocusInput()
}

// onSessionEvent fans session events out to the sidebar and the session/cost
// status labels so they stay fresh without extra DB round-trips.
func (g *MainWindow) onSessionEvent(ev pubsub.Event[session.Session]) {
	g.sidebar.OnSessionEvent(ev)
	if ev.Payload.ID == g.chat.Current() {
		g.sessionPanel.SetSessionInfo(ev.Payload)
		g.updateCost(ev.Payload)
	}
}

func (g *MainWindow) onSessionDeleted(sessionID string) {
	g.chat.SetCurrent("")
	g.todo.SetSession("")
	g.sessionPanel.SetSession("")
	g.updateCost(session.Session{})
}

// cycleTheme rotates the GUI theme: auto -> light -> dark -> auto.
func (g *MainWindow) cycleTheme() {
	order := []string{"auto", "light", "dark"}
	cur := config.Get().GUI.Theme
	next := "auto"
	for i, t := range order {
		if t == cur {
			next = order[(i+1)%len(order)]
			break
		}
	}
	logging.Info("menu: cycle theme", "from", cur, "to", next)
	g.applyTheme(next)
}

// compactContext manually summarizes and compresses the current session's
// context via the agent's Summarize operation.
func (g *MainWindow) compactContext() {
	logging.Info("menu: compact context")
	sessionID := g.chat.Current()
	if sessionID == "" {
		return
	}
	tr := config.Tr()
	dialog.ShowConfirm(tr.GUIContextCompact, tr.GUIContextCompact+"?", func(ok bool) {
		if !ok {
			return
		}
		if err := g.core.CoderAgent.Summarize(g.ctx, sessionID); err != nil {
			logging.Error("failed to compact context: %v", err)
		}
	}, g.win)
}

// showSessionPicker opens a dialog listing sessions; selecting one switches
// to it.
func (g *MainWindow) showSessionPicker() {
	logging.Info("menu: show session picker")
	tr := config.Tr()
	sessions, err := g.core.Sessions.List(g.ctx)
	if err != nil {
		logging.Error("failed to list sessions: %v", err)
		return
	}
	var top []session.Session
	for _, s := range sessions {
		if s.ParentSessionID == "" {
			top = append(top, s)
		}
	}
	logging.Info("session picker loaded", "total", len(sessions), "top", len(top))
	if len(top) == 0 {
		dialog.ShowInformation(tr.GUISelectSession, "No sessions yet.", g.win)
		return
	}
	list := widget.NewList(
		func() int { return len(top) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(top[id].Title)
		},
	)
	dlg := dialog.NewCustomConfirm(tr.GUISelectSession, tr.GUIDone, tr.GUIDismiss, list, func(bool) {}, g.win)
	list.OnSelected = func(id widget.ListItemID) {
		if int(id) < len(top) {
			g.selectSession(top[id].ID)
		}
		dlg.Hide()
	}
	dlg.Resize(fyne.NewSize(400, 400))
	dlg.Show()
}

// requestQuit asks for confirmation before quitting (used by the Quit menu
// item, Ctrl+Q, and the window close button).
func (g *MainWindow) requestQuit() {
	logging.Info("menu: request quit")
	tr := config.Tr()
	dialog.ShowConfirm(tr.QuitQuestion, "", func(ok bool) {
		if ok {
			g.fyneApp.Quit()
		}
	}, g.win)
}

// showShortcuts opens the keyboard shortcuts help dialog (Ctrl+?).
func (g *MainWindow) showShortcuts() {
	logging.Info("menu: show shortcuts")
	tr := config.Tr()
	lines := []string{
		"Ctrl+O        " + tr.GUISwitchModel,
		"Ctrl+S        " + tr.GUISelectSession,
		"Ctrl+M        " + tr.GUIContextCompact,
		"Ctrl+T        " + tr.GUICycleTheme,
		"Ctrl+F        " + tr.GUIAttachFile,
		"Ctrl+B        " + tr.GUIToggleLeftBar,
		"Ctrl+Shift+B  " + tr.GUIToggleRightBar,
		"Ctrl+Q        " + tr.GUIQuitItem,
		"Ctrl+?        " + tr.GUIHelpShortcuts,
		"Enter         " + tr.GUISend,
		"Shift+Enter   " + tr.GUINewline,
		"Esc           " + tr.GUICancelResponse,
		"↑/↓           " + tr.GUIHistoryNav,
		"Home/End      " + tr.GUIScrollOutput,
	}
	var col1, col2 []fyne.CanvasObject
	half := (len(lines) + 1) / 2
	for i, line := range lines {
		lbl := widget.NewLabel(line)
		lbl.Selectable = true
		if i < half {
			col1 = append(col1, lbl)
		} else {
			col2 = append(col2, lbl)
		}
	}
	content := container.NewHBox(container.NewVBox(col1...), container.NewVBox(col2...))
	dlg := dialog.NewCustom(tr.GUIHelpShortcuts, tr.GUIDone, content, g.win)
	dlg.Resize(fyne.NewSize(600, 320))
	dlg.Show()
}

// toggleLeftBar shows or hides the session sidebar and syncs the View menu
// check mark.
func (g *MainWindow) toggleLeftBar() {
	logging.Info("menu: toggle left bar")
	if g.outer.Leading.Visible() {
		g.outer.Leading.Hide()
		g.viewLeftItem.Checked = false
	} else {
		g.outer.Leading.Show()
		g.viewLeftItem.Checked = true
	}
	g.outer.Refresh()
	g.viewMenu.Refresh()
}

// toggleRightBar shows or hides the right info panel and syncs the View menu
// check mark.
func (g *MainWindow) toggleRightBar() {
	logging.Info("menu: toggle right bar")
	if g.inner.Trailing.Visible() {
		g.inner.Trailing.Hide()
		g.viewRightItem.Checked = false
	} else {
		g.inner.Trailing.Show()
		g.viewRightItem.Checked = true
	}
	g.inner.Refresh()
	g.viewMenu.Refresh()
}

func (g *MainWindow) Show() {
	g.win.Show()
	g.chat.FocusInput()
}

func (g *MainWindow) Window() fyne.Window {
	return g.win
}
