//go:build x11

package gui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/go-gl/glfw/v3.4/glfw"
)

func screenSize(t *testing.T) (int, int) {
	t.Helper()
	if err := glfw.Init(); err != nil {
		t.Fatalf("glfw init: %v", err)
	}
	mon := glfw.GetPrimaryMonitor()
	if mon == nil {
		t.Fatal("no primary monitor")
	}
	mode := mon.GetVideoMode()
	return mode.Width, mode.Height
}

func windowSize(t *testing.T) (int, int) {
	t.Helper()
	w := glfw.GetCurrentContext()
	if w == nil {
		t.Fatal("no current context")
	}
	x, y := w.GetSize()
	return x, y
}

func wait(t *testing.T, d time.Duration) {
	t.Helper()
	time.Sleep(d)
}

func TestMaximizeWithRightBar(t *testing.T) {
	sw, sh := screenSize(t)
	_ = sw
	_ = sh

	a := app.New()
	win := a.NewWindow("maximize-test")

	right := container.NewVBox(
		widget.NewButton("r1", func() {}),
		widget.NewButton("r2", func() {}),
		widget.NewLabel("right panel content"),
	)
	inner := container.NewHSplit(widget.NewLabel("chat area"), right)
	inner.SetOffset(0.72)
	outer := container.NewHSplit(widget.NewLabel("sidebar"), inner)
	outer.SetOffset(0.2)

	win.SetContent(outer)
	win.Resize(fyne.NewSize(1100, 720))
	win.Show()

	wait(t, 300*time.Millisecond)

	// 首次最大化
	if !maximizeWindowNow() {
		t.Fatal("maximize failed")
	}
	wait(t, 500*time.Millisecond)
	w, h := windowSize(t)
	t.Logf("after first maximize: %dx%d screen=%dx%d", w, h, sw, sh)
	if w > sw || h > sh {
		t.Fatalf("maximized window %dx%d exceeds screen %dx%d", w, h, sw, sh)
	}

	// 隐藏右栏
	inner.Trailing.Hide()
	inner.Refresh()
	wait(t, 400*time.Millisecond)

	// 显示右栏
	inner.Trailing.Show()
	inner.Refresh()
	wait(t, 400*time.Millisecond)

	// 再次最大化
	if !maximizeWindowNow() {
		t.Fatal("second maximize failed")
	}
	wait(t, 500*time.Millisecond)
	w, h = windowSize(t)
	if w > sw || h > sh {
		t.Fatalf("window exceeds screen after toggle+maximize: %dx%d > %dx%d", w, h, sw, sh)
	}
}
