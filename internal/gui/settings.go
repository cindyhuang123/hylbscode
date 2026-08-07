package gui

import (
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/llm/models"
	"github.com/cindyhuang123/hylbscode/internal/logging"
)

// ShowSettings opens the settings dialog: pick a provider, then a model from
// that provider.
func (g *MainWindow) ShowSettings() {
	logging.Info("menu: show settings")
	tr := config.Tr()
	cfg := config.Get()
	provs, provNames := providerNames()
	if len(provs) == 0 {
		dialog.ShowInformation(tr.GUISettingsItem, "No models available.", g.win)
		return
	}
	currentModel := cfg.Agents[config.AgentCoder].Model
	modelsOf := func(p models.ModelProvider) []models.Model {
		var ms []models.Model
		for _, m := range models.SupportedModels {
			if m.Provider == p {
				ms = append(ms, m)
			}
		}
		sort.Slice(ms, func(i, j int) bool { return ms[i].Name < ms[j].Name })
		return ms
	}

	selectedProv := provs[0]
	modelSelect := widget.NewSelect(nil, nil)
	refreshModels := func(p models.ModelProvider) {
		selectedProv = p
		ms := modelsOf(p)
		names := make([]string, 0, len(ms))
		for _, m := range ms {
			names = append(names, m.Name)
		}
		modelSelect.Options = names
		modelSelect.ClearSelected()
		for i, m := range ms {
			if m.ID == currentModel {
				modelSelect.SetSelectedIndex(i)
				break
			}
		}
		if modelSelect.SelectedIndex() < 0 && len(names) > 0 {
			modelSelect.SetSelectedIndex(0)
		}
	}

	providerSelect := widget.NewSelect(provNames, func(name string) {
		for _, p := range provs {
			if displayProviderName(p) == name {
				refreshModels(p)
				return
			}
		}
	})
	providerSelect.SetSelectedIndex(0)
	refreshModels(selectedProv)
	for provIdx, p := range provs {
		for _, m := range models.SupportedModels {
			if m.ID == currentModel && m.Provider == p {
				providerSelect.SetSelectedIndex(provIdx)
				refreshModels(p)
				break
			}
		}
	}

	var selectedModel models.ModelID
	modelSelect.OnChanged = func(name string) {
		for _, m := range modelsOf(selectedProv) {
			if m.Name == name {
				selectedModel = m.ID
				break
			}
		}
	}
	selectedModel = currentModel

	content := container.NewVBox(
		widget.NewLabelWithStyle(tr.GUIProviderSelect, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		providerSelect,
		widget.NewLabelWithStyle(tr.GUIModelLabel, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		modelSelect,
	)
	dlg := dialog.NewCustomConfirm(tr.GUISettingsItem, tr.GUIDone, tr.GUIDismiss, content, func(ok bool) {
		if ok && selectedModel != "" {
			if err := config.UpdateAgentModel(config.AgentCoder, selectedModel); err != nil {
				logging.Error("failed to update agent model: %v", err)
			}
		}
	}, g.win)
	dlg.Resize(fyne.NewSize(450, 320))
	dlg.Show()
}

// applyTheme switches the Fyne native theme, persists it, and updates the
// Theme menu check marks.
func (g *MainWindow) applyTheme(sel string) {
	logging.Info("menu: apply theme", "theme", sel)
	if sel == "" {
		return
	}
	if err := config.UpdateGUITheme(sel); err != nil {
		logging.Error("failed to update GUI theme: %v", err)
		return
	}
	switch sel {
	case "light":
		g.applyThemeFont(theme.LightTheme())
	case "dark":
		g.applyThemeFont(theme.DarkTheme())
	default:
		g.applyThemeFont(nil)
	}
	g.themeAuto.Checked = sel == "auto"
	g.themeLight.Checked = sel == "light"
	g.themeDark.Checked = sel == "dark"
	g.themeMenu.Refresh()
}

// Menu builds the main menu for the window.
func (g *MainWindow) Menu() *fyne.MainMenu {
	tr := config.Tr()

	settings := fyne.NewMenuItem(tr.GUISettingsItem, g.ShowSettings)
	settings.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyO, Modifier: fyne.KeyModifierControl}
	providerCfg := fyne.NewMenuItem(tr.GUIProviderMenu, g.ShowProviderSetup)
	quit := fyne.NewMenuItem(tr.GUIQuitItem, g.requestQuit)
	quit.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyQ, Modifier: fyne.KeyModifierControl}
	file := fyne.NewMenu(tr.GUIFileMenu, settings, providerCfg, quit)

	g.themeAuto = fyne.NewMenuItem(tr.GUIThemeAuto, func() { g.applyTheme("auto") })
	g.themeLight = fyne.NewMenuItem(tr.GUIThemeLight, func() { g.applyTheme("light") })
	g.themeDark = fyne.NewMenuItem(tr.GUIThemeDark, func() { g.applyTheme("dark") })
	switch config.Get().GUI.Theme {
	case "light":
		g.themeLight.Checked = true
	case "dark":
		g.themeDark.Checked = true
	default:
		g.themeAuto.Checked = true
	}
	cycleTheme := fyne.NewMenuItem(tr.GUICycleTheme, g.cycleTheme)
	cycleTheme.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyT, Modifier: fyne.KeyModifierControl}
	g.themeMenu = fyne.NewMenu(tr.GUIThemeMenu, g.themeAuto, g.themeLight, g.themeDark, cycleTheme)

	g.viewLeftItem = fyne.NewMenuItem(tr.GUIToggleLeftBar, g.toggleLeftBar)
	g.viewLeftItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyB, Modifier: fyne.KeyModifierControl}
	g.viewLeftItem.Checked = true
	g.viewRightItem = fyne.NewMenuItem(tr.GUIToggleRightBar, g.toggleRightBar)
	g.viewRightItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyB, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift}
	g.viewRightItem.Checked = true
	selectSession := fyne.NewMenuItem(tr.GUISelectSession, g.showSessionPicker)
	selectSession.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierControl}
	attachFile := fyne.NewMenuItem(tr.GUIAttachFile, func() { g.chat.PickAttachment() })
	attachFile.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: fyne.KeyModifierControl}
	compactCtx := fyne.NewMenuItem(tr.GUIContextCompact, g.compactContext)
	compactCtx.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyM, Modifier: fyne.KeyModifierControl}
	g.viewMenu = fyne.NewMenu(tr.GUIViewMenu, g.viewLeftItem, g.viewRightItem, selectSession, attachFile, compactCtx)

	enItem := fyne.NewMenuItem(tr.GUIEnglish, func() { g.applyLanguage(config.LangEnglish) })
	zhItem := fyne.NewMenuItem(tr.GUIChinese, func() { g.applyLanguage(config.LangChinese) })
	switch config.CurrentLanguage() {
	case config.LangChinese:
		zhItem.Checked = true
	default:
		enItem.Checked = true
	}
	langMenu := fyne.NewMenu(tr.GUILanguageMenu, enItem, zhItem)

	helpItem := fyne.NewMenuItem(tr.GUIHelpShortcuts, g.showShortcuts)
	helpItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeySlash, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift}
	helpMenu := fyne.NewMenu(tr.GUIHelpMenu, helpItem)

	return fyne.NewMainMenu(file, g.themeMenu, g.viewMenu, langMenu, helpMenu)
}
