package gui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/charmbracelet/x/ansi"
)

// ToolBlock renders one tool invocation: a title line plus monospace output
// on a rounded background. It is used both for streamed tool execution
// (AppendOutput while running) and for completed ToolCall/ToolResult message
// parts. The output label is selectable so tool output can be copied.
type ToolBlock struct {
	widget.BaseWidget
	name       string
	title      *widget.RichText
	outputText strings.Builder
	output     *widget.Label
	outputRect *canvas.Rectangle
	outputBox  *fyne.Container
	box        *fyne.Container
	noOutput   bool
	expanded   bool
	maxLines   int
	expandBtn  *widget.Button
}

// NewToolBlock creates a tool block in the "running" state.
func NewToolBlock(name string) *ToolBlock {
	return newToolBlock(name, false)
}

// NewCompactToolBlock creates a tool block that only shows the title line,
// hiding the output area entirely (for intermediate tool rounds where the
// user only needs to see which tool ran, not its full output).
func NewCompactToolBlock(name string) *ToolBlock {
	t := newToolBlock(name, true)
	t.outputBox.Hide()
	return t
}

func newToolBlock(name string, noOutput bool) *ToolBlock {
	t := &ToolBlock{
		name:     name,
		noOutput: noOutput,
		maxLines: 10,
		title: widget.NewRichText(&widget.TextSegment{
			Text:  "⏳ " + name,
			Style: widget.RichTextStyle{ColorName: theme.ColorNameForeground, TextStyle: fyne.TextStyle{Bold: true}},
		}),
		output: widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}),
	}
	t.output.Wrapping = fyne.TextWrapWord
	t.output.Selectable = true
	t.outputRect = canvas.NewRectangle(theme.InputBackgroundColor())
	t.outputRect.CornerRadius = 4
	if name == "" {
		t.title.Hide()
	}
	t.expandBtn = widget.NewButton("▼ 展开", t.toggleExpand)
	t.expandBtn.Alignment = widget.ButtonAlignLeading
	t.expandBtn.Hide()
	t.outputBox = container.NewStack(t.outputRect, container.NewPadded(t.output))
	t.box = container.NewVBox(t.title, t.outputBox, t.expandBtn)
	t.ExtendBaseWidget(t)
	return t
}

// TitleText returns the current title text (used by tests).
func (t *ToolBlock) TitleText() string {
	if len(t.title.Segments) == 0 {
		return ""
	}
	if seg, ok := t.title.Segments[0].(*widget.TextSegment); ok {
		return seg.Text
	}
	return ""
}

// SetResult transitions the block to its final state after the tool finishes.
// Successful tools hide the title line entirely (the green "✓ <call-id>"
// header is visual noise; only the output stays). Errors keep a red title so
// failures stay easy to spot. In compact mode (noOutput), successful results
// show a green ✓ check so the user can still see the tool completed.
func (t *ToolBlock) SetResult(name string, isError bool) {
	if !isError {
		if t.noOutput {
			t.title.Show()
			if name == "" {
				name = t.name
			}
			t.title.Segments = []widget.RichTextSegment{
				&widget.TextSegment{
					Text:  "✓   " + name,
					Style: widget.RichTextStyle{ColorName: theme.ColorNameSuccess, TextStyle: fyne.TextStyle{}},
				},
			}
			t.title.Refresh()
			return
		}
		t.title.Hide()
		t.title.Segments = nil
		t.title.Refresh()
		return
	}
	if name == "" {
		name = "error"
	}
	t.title.Show()
	t.title.Segments = []widget.RichTextSegment{
		&widget.TextSegment{
			Text:  "✗ " + name,
			Style: widget.RichTextStyle{ColorName: theme.ColorNameError, TextStyle: fyne.TextStyle{Bold: true}},
		},
	}
	t.title.Refresh()
	ec := color.NRGBAModel.Convert(theme.Color(theme.ColorNameError)).(color.NRGBA)
	t.outputRect.FillColor = color.NRGBA{R: ec.R, G: ec.G, B: ec.B, A: 0x1E}
	t.outputRect.Refresh()
}

// AppendOutput strips ANSI escapes and appends a chunk to the output.
func (t *ToolBlock) AppendOutput(text string) {
	if text == "" {
		return
	}
	t.outputText.WriteString(ansi.Strip(text))
	t.refreshOutput()
}

// SetOutput replaces the output content with a single monospace block.
func (t *ToolBlock) SetOutput(text string) {
	t.outputText.Reset()
	t.outputText.WriteString(ansi.Strip(text))
	t.refreshOutput()
}

func (t *ToolBlock) toggleExpand() {
	t.expanded = !t.expanded
	if t.expanded {
		t.expandBtn.SetText("▲ 收起")
	} else {
		t.expandBtn.SetText("▼ 展开")
	}
	t.refreshOutput()
}

func (t *ToolBlock) refreshOutput() {
	full := t.outputText.String()
	if t.maxLines <= 0 {
		t.output.SetText(full)
		return
	}
	if t.expanded {
		t.output.SetText(full)
		t.expandBtn.SetText("▲ 收起")
		t.expandBtn.Show()
		return
	}
	lines := strings.Split(full, "\n")
	if len(lines) <= t.maxLines {
		t.output.SetText(full)
		t.expandBtn.Hide()
		return
	}
	truncated := strings.Join(lines[:t.maxLines], "\n") + "\n..."
	t.output.SetText(truncated)
	t.expandBtn.SetText("▼ 展开")
	t.expandBtn.Show()
}

func (t *ToolBlock) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.box)
}
