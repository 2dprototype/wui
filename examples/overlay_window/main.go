package main

import "github.com/2dprototype/wui"

// A headless (borderless), semi-transparent, always-on-top window with a
// color key. Drag it anywhere, double-check the opacity with the slider.
func main() {
	window := wui.NewWindow()
	window.SetInnerSize(300, 160)
	window.SetCenterOnShow(true)
	window.SetHeadless(true)
	window.SetResizable(false)
	window.SetDragByBackground(true)
	window.SetTopMost(true)
	window.SetShowInTaskbar(false)
	window.SetAlpha(230)

	// Everything painted in the key color is see-through.
	key := wui.RGB(255, 0, 255)
	window.SetTransparentColor(key)
	window.SetBackground(key)

	panel := wui.NewPanel()
	panel.SetBackgroundColor(wui.RGB(40, 40, 60))
	panel.SetBounds(0, 0, 300, 160)
	window.Add(panel)

	title := wui.NewLabel()
	title.SetText("Overlay window")
	title.SetTextColor(wui.RGB(255, 255, 255))
	title.SetBackgroundColor(wui.RGB(40, 40, 60))
	title.SetBounds(16, 14, 200, 20)
	panel.Add(title)

	opacity := wui.NewSlider()
	opacity.SetMinMax(60, 255)
	opacity.SetCursorPosition(230)
	opacity.SetBounds(16, 50, 268, 40)
	opacity.SetOnChange(func(pos int) { window.SetAlpha(uint8(pos)) })
	panel.Add(opacity)

	quit := wui.NewButton()
	quit.SetText("Close")
	quit.SetBounds(200, 115, 84, 28)
	quit.SetOnClick(window.Close)
	panel.Add(quit)

	window.SetCornerRadius(18)
	window.SetShortcut(window.Close, wui.KeyEscape)
	window.Show()
}
