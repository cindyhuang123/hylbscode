package gui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/cindyhuang123/hylbscode/internal/config"
	"github.com/cindyhuang123/hylbscode/internal/logging"
)

// ChatInput 是聊天输入控件：Enter 发送、Shift+Enter 换行。
//
// Fyne 多行 Entry 的原生行为恰好相反（Enter 插入换行、Shift+Enter 触发
// OnSubmitted 提交），且裸 Enter / Shift+Enter 都无法被 canvas 级 shortcut
// 拦截（triggersShortcut 只为非零且非 Shift 的修饰键构造 CustomShortcut）。
// 因此这里组合 widget.Entry，并在两层拦截 Enter：焦点在 ChatInput 时由
// desktop.Keyable.KeyDown 处理；点击输入框后焦点常驻内层 chatEntry（Fyne
// 键盘事件只发给当前焦点对象，此时 ChatInput.KeyDown 收不到事件），由
// chatEntry 覆盖 KeyDown/TypedKey 处理。Shift 状态用
// desktop.Driver.CurrentKeyModifiers 读取。
//
// chatEntry 必须用零值 Entry 构造并把自己的 BaseWidget.impl 指向 chatEntry
// 本身：Entry.requestFocus 通过 super() 找焦点对象，若 impl 是内嵌的
// *widget.Entry（widget.NewEntry 的默认行为），CanvasForObject 找不到对象
// （对象树里是 chatEntry），点击输入框将完全无法获得焦点。
//
// 焦点处理：Entry 被点击时其内部 requestFocus 会把焦点抢走（之后按键会走
// Entry 原生逻辑）。这里实现 fyne.Focusable，在 FocusLost 时通过 fyne.Do
// 排到下一帧把焦点夺回——FocusLost 阶段焦点管理器尚未切换焦点，直接
// canvas.Focus 会被等值检查短路，延迟一帧后 f.focused 已指向 Entry，
// 此时夺回才能生效。窗口失焦等场景下 f.focused 仍指向本控件，夺回为空操作。
type ChatInput struct {
	widget.BaseWidget
	entry         *chatEntry
	onSubmit      func(string)
	onCancel      func()
	onPageUp      func()
	onPageDown    func()
	onPageHome    func()
	onPageEnd     func()
	onHistoryUp   func()
	onHistoryDown func()
}

// SetPageHandlers registers PageUp/PageDown/Home/End handlers so the chat
// area can scroll the message list when the input has focus.
func (c *ChatInput) SetPageHandlers(pageUp, pageDown, home, end func()) {
	c.onPageUp = pageUp
	c.onPageDown = pageDown
	c.onPageHome = home
	c.onPageEnd = end
}

// SetHistoryHandlers registers Up/Down handlers so the chat area can recall
// previously sent messages while the input has focus.
func (c *ChatInput) SetHistoryHandlers(up, down func()) {
	c.onHistoryUp = up
	c.onHistoryDown = down
}

// NewChatInput 创建一个 Enter 发送、Shift+Enter 换行、Esc 取消（onCancel）
// 的多行输入控件。
func NewChatInput(onSubmit func(string), onCancel func()) *ChatInput {
	c := &ChatInput{onSubmit: onSubmit, onCancel: onCancel}
	c.entry = &chatEntry{Entry: &widget.Entry{}, parent: c}
	c.entry.ExtendBaseWidget(c.entry)
	c.entry.MultiLine = true
	c.entry.SetPlaceHolder(config.Tr().GUIInputPlaceholder)
	c.ExtendBaseWidget(c)
	return c
}

// CreateRenderer 渲染内部 Entry。
func (c *ChatInput) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.entry)
}

// chatEntry 包装 widget.Entry，在 Entry 自身持有焦点时拦截 Enter 提交。
// Fyne 键盘事件只发给当前焦点对象：点击输入框后焦点常驻 Entry（外层无
// 焦点时 FocusLost 夺回不会触发），此时 Enter 走 Entry 原生换行逻辑，
// 必须在这里拦截（见 ChatInput 文档）。
type chatEntry struct {
	*widget.Entry
	parent *ChatInput
}

func (e *chatEntry) KeyDown(ev *fyne.KeyEvent) {
	e.Entry.KeyDown(ev)
	if ev.Name != fyne.KeyReturn && ev.Name != fyne.KeyEnter {
		return
	}
	if e.Disabled() || e.parent.shiftHeld() {
		return // Shift+Enter 换行，由 TypedKey 放行给 Entry
	}
	logging.Info("chatinput: entry keydown enter -> submit")
	e.parent.onSubmit(e.Text)
}

func (e *chatEntry) TypedKey(ev *fyne.KeyEvent) {
	switch ev.Name {
	case fyne.KeyReturn, fyne.KeyEnter:
		// press 已由 KeyDown 提交；长按 repeat 只走 TypedKey，直接吞掉
		// 防止 Entry 插入换行。Shift+Enter 放行给 Entry 插入换行。
		if e.parent.shiftHeld() {
			e.Entry.TypedKey(ev)
		}
	case fyne.KeyPageUp:
		if e.parent.onPageUp != nil {
			e.parent.onPageUp()
		}
	case fyne.KeyPageDown:
		if e.parent.onPageDown != nil {
			e.parent.onPageDown()
		}
	case fyne.KeyEscape:
		// 输入框聚焦时 Esc 取消当前应答。fyne v2.8 的 Entry 与 dialog 均不
		// 处理 Esc；popup menu 的 Esc 在菜单自身焦点下处理，不受影响。
		if e.parent.onCancel != nil {
			e.parent.onCancel()
		}
	case fyne.KeyUp, fyne.KeyDown:
		// 单行输入时上下键翻历史；多行（含换行）时让给 Entry 做光标移动。
		if strings.Contains(e.Text, "\n") {
			e.Entry.TypedKey(ev)
			return
		}
		if ev.Name == fyne.KeyUp {
			if e.parent.onHistoryUp != nil {
				e.parent.onHistoryUp()
			}
		} else if e.parent.onHistoryDown != nil {
			e.parent.onHistoryDown()
		}
	case fyne.KeyHome:
		if e.parent.onPageHome != nil {
			e.parent.onPageHome()
		}
	case fyne.KeyEnd:
		if e.parent.onPageEnd != nil {
			e.parent.onPageEnd()
		}
	default:
		e.Entry.TypedKey(ev)
	}
}

// ---- fyne.Focusable ----

func (c *ChatInput) FocusGained() {
	logging.Info("chatinput: focus gained", "focused", c.focusType())
	c.entry.FocusGained()
}

func (c *ChatInput) FocusLost() {
	logging.Info("chatinput: focus lost", "focused", c.focusType())
	c.entry.FocusLost()
	if cv := fyne.CurrentApp().Driver().CanvasForObject(c); cv != nil {
		fyne.Do(func() {
			f := cv.Focused()
			logging.Debug("chatinput: focus check", "focused", c.focusTypeOf(f))
			switch {
			case f == c.entry:
				// Clicking the input focuses the inner entry; reclaim the outer
				// widget so Enter interception still works.
				cv.Focus(c)
				logging.Info("chatinput: focus reclaimed (inner entry)")
			case f != nil && f != c:
				// Focus moved to another widget (text selection, buttons,
				// dialogs); let it keep focus so selection/copy works.
				logging.Info("chatinput: focus stays on other widget")
			default:
				cv.Focus(c)
				logging.Info("chatinput: focus reclaimed")
			}
		})
	} else {
		logging.Warn("chatinput: focus lost, canvas not found")
	}
}

func (c *ChatInput) TypedRune(r rune) {
	logging.Debug("chatinput: typedrune", "rune", fmt.Sprintf("%q", r))
	c.entry.TypedRune(r)
}

func (c *ChatInput) TypedKey(ev *fyne.KeyEvent) {
	logging.Debug("chatinput: typedkey", "key", ev.Name)
	if ev.Name == fyne.KeyReturn || ev.Name == fyne.KeyEnter {
		logging.Info("chatinput: typedkey swallowed enter")
		return
	}
	c.entry.TypedKey(ev)
}

// focusType returns a short description of the currently focused object, or
// "nil" when nothing is focused; used in logs to trace focus movement.
func (c *ChatInput) focusType() string {
	cv := fyne.CurrentApp().Driver().CanvasForObject(c)
	if cv == nil {
		return "no-canvas"
	}
	return c.focusTypeOf(cv.Focused())
}

func (c *ChatInput) focusTypeOf(f fyne.Focusable) string {
	switch {
	case f == nil:
		return "nil"
	case f == c:
		return "ChatInput"
	case f == c.entry:
		return "entry"
	default:
		return fmt.Sprintf("%T", f)
	}
}

// ---- desktop.Keyable ----

func (c *ChatInput) KeyDown(ev *fyne.KeyEvent) {
	logging.Debug("chatinput: keydown", "key", ev.Name, "mod", c.modifierSummary())
	if ev.Name == fyne.KeyReturn || ev.Name == fyne.KeyEnter {
		logging.Info("chatinput: keydown enter/return", "shift", c.shiftHeld(), "disabled", c.entry.Disabled())
		if c.entry.Disabled() {
			return
		}
		if c.shiftHeld() {
			logging.Info("chatinput: shift+enter -> newline")
			c.entry.TypedKey(ev)
			return
		}
		logging.Info("chatinput: enter -> submit")
		c.onSubmit(c.entry.Text)
		return
	}
	c.entry.KeyDown(ev)
}

// modifierSummary renders the current key modifiers for debug logging.
func (c *ChatInput) modifierSummary() string {
	d, ok := fyne.CurrentApp().Driver().(desktop.Driver)
	if !ok {
		return "?"
	}
	mods := d.CurrentKeyModifiers()
	var parts []string
	if mods&fyne.KeyModifierControl != 0 {
		parts = append(parts, "Ctrl")
	}
	if mods&fyne.KeyModifierShift != 0 {
		parts = append(parts, "Shift")
	}
	if mods&fyne.KeyModifierAlt != 0 {
		parts = append(parts, "Alt")
	}
	if mods&fyne.KeyModifierSuper != 0 {
		parts = append(parts, "Super")
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, "+")
}

func (c *ChatInput) KeyUp(ev *fyne.KeyEvent) {
	c.entry.KeyUp(ev)
}

func (c *ChatInput) shiftHeld() bool {
	d, ok := fyne.CurrentApp().Driver().(desktop.Driver)
	if !ok {
		return false
	}
	return d.CurrentKeyModifiers()&fyne.KeyModifierShift != 0
}

// ---- 便捷委托 ----

func (c *ChatInput) Text() string { return c.entry.Text }

func (c *ChatInput) SetText(text string) { c.entry.SetText(text) }

func (c *ChatInput) Disabled() bool { return c.entry.Disabled() }

func (c *ChatInput) Disable() { c.entry.Disable() }

func (c *ChatInput) Enable() { c.entry.Enable() }
