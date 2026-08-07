package gui

import (
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/llm/models"
	"github.com/cindyhuang123/hylbscode/internal/logging"
)

// providerNames returns the providers present in SupportedModels, ordered by
// popularity.
func providerNames() ([]models.ModelProvider, []string) {
	seen := make(map[models.ModelProvider]bool)
	var provs []models.ModelProvider
	for _, m := range models.SupportedModels {
		if !seen[m.Provider] {
			seen[m.Provider] = true
			provs = append(provs, m.Provider)
		}
	}
	sort.Slice(provs, func(i, j int) bool {
		return models.ProviderPopularity[provs[i]] < models.ProviderPopularity[provs[j]]
	})
	names := make([]string, 0, len(provs))
	for _, p := range provs {
		names = append(names, displayProviderName(p))
	}
	return provs, names
}

func displayProviderName(p models.ModelProvider) string {
	s := string(p)
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// ShowProviderSetup opens the provider configuration dialog: pick a provider,
// enter the API key (and optional base URL), then save.
func (g *MainWindow) ShowProviderSetup() {
	logging.Info("menu: show provider setup")
	tr := config.Tr()
	provs, names := providerNames()
	if len(provs) == 0 {
		dialog.ShowInformation(tr.GUIProviderTitle, "No providers available.", g.win)
		return
	}
	providerSelect := widget.NewSelect(names, nil)
	providerSelect.SetSelectedIndex(0)
	keyEntry := widget.NewPasswordEntry()
	keyEntry.SetPlaceHolder(tr.GUIProviderAPIKey)
	baseURLEntry := widget.NewEntry()
	baseURLEntry.SetPlaceHolder(tr.GUIProviderBaseURL)

	content := container.NewVBox(
		widget.NewLabelWithStyle(tr.GUIProviderSelect, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		providerSelect,
		widget.NewLabelWithStyle(tr.GUIProviderAPIKey, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		keyEntry,
		widget.NewLabelWithStyle(tr.GUIProviderBaseURL, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		baseURLEntry,
	)
	dlg := dialog.NewCustomConfirm(tr.GUIProviderTitle, tr.GUIProviderSave, tr.GUIDismiss, content, func(save bool) {
		if !save {
			return
		}
		if strings.TrimSpace(keyEntry.Text) == "" {
			dialog.ShowInformation(tr.GUIProviderTitle, tr.GUIProviderNoKey, g.win)
			return
		}
		idx := providerSelect.SelectedIndex()
		if idx < 0 || idx >= len(provs) {
			return
		}
		if err := config.UpdateProviderConfig(provs[idx], keyEntry.Text, baseURLEntry.Text); err != nil {
			logging.Error("failed to save provider config: %v", err)
			dialog.ShowInformation(tr.GUIProviderTitle, err.Error(), g.win)
			return
		}
		// 配置保存成功后重建 coder agent，使用户无需重启即可使用。
		if err := g.core.EnsureCoderAgent(); err != nil {
			logging.Error("failed to create coder agent after provider setup: %v", err)
			dialog.ShowInformation(tr.GUIProviderTitle, "API Key 已保存，但创建模型代理失败: "+err.Error(), g.win)
			return
		}
		g.subscribeCoderAgent()
		dialog.ShowInformation(tr.GUIProviderTitle, tr.GUIProviderSaved, g.win)
	}, g.win)
	dlg.Resize(fyne.NewSize(450, 300))
	dlg.Show()
}
