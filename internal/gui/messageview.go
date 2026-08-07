package gui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/charmbracelet/x/ansi"
	"github.com/cindyhuang123/hylbscode/internal/message"
)

const maxInlineAttachmentLen = 120

// textBlock is one styled paragraph produced by markdownBlocks.
type textBlock struct {
	text  string
	style fyne.TextStyle
}

// markdownBlocks splits a markdown string into styled plain-text blocks:
// fenced code blocks become monospace, heading lines become bold, everything
// else stays regular. Consecutive same-style lines are merged into a single
// multi-line block so a whole paragraph is selectable at once — each block
// renders as one selectable label, and fyne only supports cross-line drag
// selection within a single label. Blank lines, headings and code fences act
// as block boundaries.
func markdownBlocks(text string) []textBlock {
	var blocks []textBlock
	var code strings.Builder
	var plain strings.Builder
	inCode := false

	flushCode := func() {
		if code.Len() == 0 {
			return
		}
		blocks = append(blocks, textBlock{code.String(), fyne.TextStyle{Monospace: true}})
		code.Reset()
	}
	flushPlain := func() {
		if plain.Len() == 0 {
			return
		}
		blocks = append(blocks, textBlock{strings.TrimSuffix(plain.String(), "\n"), fyne.TextStyle{}})
		plain.Reset()
	}
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			flushPlain()
			if inCode {
				inCode = false
				flushCode()
			} else {
				inCode = true
			}
			continue
		}
		if inCode {
			code.WriteString(line)
			code.WriteString("\n")
			continue
		}
		if strings.TrimSpace(line) == "" {
			flushPlain()
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			flushPlain()
			blocks = append(blocks, textBlock{line, fyne.TextStyle{Bold: true}})
			continue
		}
		plain.WriteString(line)
		plain.WriteString("\n")
	}
	flushPlain()
	if inCode {
		flushCode()
	}
	return blocks
}

// inlineText is one parsed segment of a prose line: plain text, inline code
// or a markdown link (text holds the link's display label).
type inlineText struct {
	text string
	code bool
	link bool
}

// parseInline splits one line into plain / inline-code / link segments. It
// reports false when the line carries no inline markdown so the caller can
// keep rendering it as a plain selectable label.
func parseInline(line string) ([]inlineText, bool) {
	var segs []inlineText
	var plain strings.Builder
	flush := func() {
		if plain.Len() > 0 {
			segs = append(segs, inlineText{text: plain.String()})
			plain.Reset()
		}
	}
	hasInline := false
	for i := 0; i < len(line); {
		switch line[i] {
		case '`':
			if end := strings.IndexByte(line[i+1:], '`'); end >= 0 {
				flush()
				segs = append(segs, inlineText{text: line[i+1 : i+1+end], code: true})
				i += end + 2
				hasInline = true
				continue
			}
		case '[':
			if close := strings.IndexByte(line[i+1:], ']'); close >= 0 {
				open := i + 1 + close + 1
				if open < len(line) && line[open] == '(' {
					if end := strings.IndexByte(line[open+1:], ')'); end >= 0 {
						flush()
						segs = append(segs, inlineText{text: line[i+1 : i+1+close], link: true})
						i = open + 1 + end + 1
						hasInline = true
						continue
					}
				}
			}
		}
		plain.WriteByte(line[i])
		i++
	}
	flush()
	return segs, hasInline
}

// inlineRichText renders one parsed line as rich text: inline code is
// monospace/primary, links are primary, the rest stays default. Only used for
// lines carrying inline markup, where color beats selectability.
func inlineRichText(line string) *widget.RichText {
	segs, _ := parseInline(line)
	rich := make([]widget.RichTextSegment, 0, len(segs))
	for _, s := range segs {
		style := widget.RichTextStyle{Inline: true}
		switch {
		case s.code:
			style.ColorName = theme.ColorNamePrimary
			style.TextStyle.Monospace = true
		case s.link:
			style.ColorName = theme.ColorNamePrimary
		}
		rich = append(rich, &widget.TextSegment{Text: s.text, Style: style})
	}
	rt := widget.NewRichText(rich...)
	rt.Wrapping = fyne.TextWrapWord
	return rt
}

// codeBlock wraps monospace text in a rounded background so code stands out
// from prose; the label stays selectable.
func codeBlock(text string) fyne.CanvasObject {
	rect := canvas.NewRectangle(theme.InputBackgroundColor())
	rect.CornerRadius = 4
	lbl := widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
	lbl.Wrapping = fyne.TextWrapWord
	lbl.Selectable = true
	return container.NewStack(rect, container.NewPadded(lbl))
}

// userMessageBackground tints the whole user message with a faint primary
// tint so the conversation direction is visible at a glance.
func userMessageBackground(content fyne.CanvasObject) fyne.CanvasObject {
	c := color.NRGBAModel.Convert(theme.Color(theme.ColorNamePrimary)).(color.NRGBA)
	rect := canvas.NewRectangle(color.NRGBA{R: c.R, G: c.G, B: c.B, A: 0x14})
	rect.CornerRadius = 6
	return container.NewStack(rect, container.NewPadded(content))
}

// roleColor maps a message role to a theme color for the role header line.
func roleColor(role message.MessageRole) fyne.ThemeColorName {
	switch role {
	case message.User:
		return theme.ColorNamePrimary
	case message.Tool:
		return theme.ColorNameWarning
	default:
		return theme.ColorNameForeground
	}
}

// renderMessage builds the canvas object for a stored message. ToolCall parts
// reuse the matching live block from active (keyed by tool call ID) so a
// running tool keeps streaming; used maps the consumed call IDs to their
// blocks. doneTools holds call IDs whose result is stored in a ToolResult
// message: those blocks are rendered only by the ToolResult part, otherwise
// the same block would be added to both messages and drawn twice. The caller
// must not mutate active.
func renderMessage(m message.Message, active map[string]*ToolBlock, doneTools map[string]bool, compact bool) (fyne.CanvasObject, map[string]*ToolBlock) {
	role := roleLabel(m.Role)
	var style fyne.TextStyle
	switch m.Role {
	case message.User:
		style = fyne.TextStyle{Bold: true}
	case message.Tool:
		style = fyne.TextStyle{Bold: true, Italic: true}
	}
	header := widget.NewRichText(&widget.TextSegment{
		Text:  role,
		Style: widget.RichTextStyle{ColorName: roleColor(m.Role), TextStyle: style},
	})

	body := container.NewVBox()
	used := make(map[string]*ToolBlock)
	hasText := false
	for _, part := range m.Parts {
		switch p := part.(type) {
		case message.TextContent:
			text := ansi.Strip(p.Text)
			if text == "" {
				continue
			}
			hasText = true
			for _, b := range markdownBlocks(text) {
				if b.style.Monospace {
					body.Add(codeBlock(b.text))
					continue
				}
				if _, has := parseInline(b.text); has {
					// A block with inline markup loses whole-block selectability;
					// render each line as rich text so code/link colors show.
					for _, line := range strings.Split(b.text, "\n") {
						if strings.TrimSpace(line) == "" {
							continue
						}
						body.Add(inlineRichText(line))
					}
					continue
				}
				lbl := widget.NewLabelWithStyle(b.text, fyne.TextAlignLeading, b.style)
				lbl.Wrapping = fyne.TextWrapWord
				lbl.Selectable = true
				body.Add(lbl)
			}
		case message.ReasoningContent:
			if trim := ansi.Strip(p.Thinking); trim != "" {
				rt := widget.NewRichText(&widget.TextSegment{
					Text:  trim,
					Style: widget.RichTextStyle{ColorName: theme.ColorNameDisabled, TextStyle: fyne.TextStyle{Italic: true}},
				})
				rt.Wrapping = fyne.TextWrapWord
				body.Add(rt)
			}
		case message.ToolCall:
			if doneTools[p.ID] {
				continue
			}
			var block *ToolBlock
			var ok bool
			if !compact {
				block, ok = active[p.ID]
			}
			if !ok {
				block = NewToolBlock(p.Name)
				block.SetOutput(p.Input)
			}
			if p.Finished {
				block.SetResult(p.Name, false)
			}
			if p.ID != "" {
				used[p.ID] = block
			}
			body.Add(block)
		case message.ToolResult:
			name := p.Name
			if name == "" && p.IsError {
				name = p.ToolCallID
			}
			var block *ToolBlock
			if compact && p.IsError {
				if p.ToolCallID != "" {
					used[p.ToolCallID] = nil
				}
				continue
			}
			block = active[p.ToolCallID]
			if block == nil {
				block = NewToolBlock(name)
			}
			block.SetResult(name, p.IsError)
			block.SetOutput(p.Content)
			if p.ToolCallID != "" {
				used[p.ToolCallID] = block
			}
			body.Add(block)
		case message.ImageURLContent:
			lbl := widget.NewLabel("🖼 " + p.URL)
			lbl.Selectable = true
			body.Add(lbl)
		case message.BinaryContent:
			lbl := widget.NewLabel(truncateAttachment(p))
			lbl.Selectable = true
			body.Add(lbl)
		case message.Finish:
			// Terminal marker; for text-bearing assistant replies, render the
			// LLM latency (finish timestamp minus message creation time) so the
			// final answer shows how long the model took to generate.
			if m.Role == message.Assistant && hasText && p.Time > m.CreatedAt {
				secs := p.Time - m.CreatedAt
				rt := widget.NewRichText(&widget.TextSegment{
					Text:  "⏱ " + (time.Duration(secs) * time.Second).Truncate(time.Second).String(),
					Style: widget.RichTextStyle{ColorName: theme.ColorNameDisabled, TextStyle: fyne.TextStyle{Italic: true}},
				})
				body.Add(rt)
			}
		}
	}

	var view fyne.CanvasObject = container.NewVBox(header, body)
	if m.Role == message.User {
		view = userMessageBackground(view)
	}
	return view, used
}

func truncateAttachment(p message.BinaryContent) string {
	label := fmt.Sprintf("📎 %s (%s)", p.MIMEType, humanBytes(len(p.Data)))
	if p.Path != "" {
		label = fmt.Sprintf("📎 %s (%s)", p.Path, humanBytes(len(p.Data)))
	}
	if len(label) > maxInlineAttachmentLen {
		return label[:maxInlineAttachmentLen] + "..."
	}
	return label
}

func humanBytes(n int) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
