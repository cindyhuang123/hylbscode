package gui

import (
	"encoding/json"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/llm/models"
	"github.com/cindyhuang123/hylbscode/internal/logging"
)

// ShowExtraModelEditor opens a dialog for editing the extraModels JSON array.
// On save the JSON is syntax-checked, then field-validated by
// config.UpdateExtraModels and applied to the in-memory model registry
// immediately; invalid input keeps the dialog open so the user can fix it.
func (g *MainWindow) ShowExtraModelEditor() {
	tr := config.Tr()
	entry := widget.NewMultiLineEntry()
	entry.SetText(extraModelsJSON())
	entry.SetPlaceHolder(tr.GUIExtraModelHint)
	entry.SetMinRowsVisible(14)

	hint := widget.NewLabelWithStyle(tr.GUIExtraModelHint, fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	hint.Wrapping = fyne.TextWrapWord

	var dlg *dialog.CustomDialog
	example := widget.NewButton(tr.GUIExtraModelExample, func() {
		text := strings.TrimSpace(entry.Text)
		if text == "" || text == "[]" {
			entry.SetText(extraModelsExample)
			return
		}
		dialog.ShowConfirm(tr.GUIExtraModelTitle, tr.GUIExtraModelExampleReplace, func(ok bool) {
			if ok {
				entry.SetText(extraModelsExample)
			}
		}, g.win)
	})
	save := widget.NewButton(tr.GUIProviderSave, func() {
		if err := g.saveExtraModelsFromEditor(entry); err != nil {
			return
		}
		dlg.Hide()
		g.refreshStatus()
		dialog.ShowInformation(tr.GUIExtraModelTitle, tr.GUIExtraModelSaved, g.win)
	})
	actions := container.NewHBox(example, save)

	content := container.NewBorder(container.NewVBox(hint), actions, nil, nil,
		container.NewScroll(entry))

	dlg = dialog.NewCustom(tr.GUIExtraModelTitle, tr.GUIDismiss, content, g.win)
	dlg.Resize(fyne.NewSize(640, 560))
	dlg.Show()
}

// saveExtraModelsFromEditor parses the editor text as a JSON array of models
// and persists it. On any error an information dialog is shown and the editor
// stays open; on success the change is already live in the model registry.
func (g *MainWindow) saveExtraModelsFromEditor(entry *widget.Entry) error {
	tr := config.Tr()
	text := strings.TrimSpace(entry.Text)
	var extras []models.Model
	if text != "" {
		if err := json.Unmarshal([]byte(text), &extras); err != nil {
			dialog.ShowInformation(tr.GUIExtraModelTitle, tr.GUIExtraModelInvalid+err.Error(), g.win)
			return err
		}
	}
	if err := config.UpdateExtraModels(extras); err != nil {
		dialog.ShowInformation(tr.GUIExtraModelTitle, err.Error(), g.win)
		return err
	}
	logging.Info("extra models updated", "count", len(extras))
	return nil
}

// extraModelsJSON returns the current extraModels as an indented JSON array.
func extraModelsJSON() string {
	cfg := config.Get()
	if cfg == nil || len(cfg.ExtraModels) == 0 {
		return "[]"
	}
	data, err := json.MarshalIndent(cfg.ExtraModels, "", "  ")
	if err != nil {
		return "[]"
	}
	return string(data)
}

const extraModelsExample = `[
  {
    "id": "my-model",
    "name": "我的自定义模型",
    "provider": "deepseek",
    "api_model": "deepseek-reasoner",
    "cost_per_1m_in": 1.2,
    "cost_per_1m_out": 2.4,
    "context_window": 65536,
    "default_max_tokens": 8192
  }
]`
