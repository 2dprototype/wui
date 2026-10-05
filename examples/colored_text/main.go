package main

import "github.com/2dprototype/wui"

func main() {
	window := wui.NewWindow()
	window.SetTitle("Colored text")
	window.SetInnerSize(360, 220)

	red := wui.NewLabel()
	red.SetText("Red text")
	red.SetTextColor(wui.RGB(220, 0, 0))
	red.SetBounds(10, 10, 160, 20)
	window.Add(red)

	banner := wui.NewLabel()
	banner.SetText("White on blue")
	banner.SetTextColor(wui.RGB(255, 255, 255))
	banner.SetBackgroundColor(wui.RGB(0, 90, 200))
	banner.SetAlignment(wui.AlignCenter)
	banner.SetBounds(10, 40, 340, 24)
	window.Add(banner)

	check := wui.NewCheckBox()
	check.SetText("Green check box")
	check.SetTextColor(wui.RGB(0, 140, 0))
	check.SetBounds(10, 80, 200, 20)
	window.Add(check)

	edit := wui.NewEditLine()
	edit.SetText("Yellow on dark")
	edit.SetTextColor(wui.RGB(255, 230, 0))
	edit.SetBackgroundColor(wui.RGB(30, 30, 30))
	edit.SetBounds(10, 110, 340, 22)
	window.Add(edit)

	panel := wui.NewPanel()
	panel.SetBackgroundColor(wui.RGB(255, 240, 200))
	panel.SetBounds(10, 145, 340, 60)
	window.Add(panel)

	inPanel := wui.NewLabel()
	inPanel.SetText("Label on a colored panel")
	inPanel.SetTextColor(wui.RGB(120, 0, 120))
	inPanel.SetBackgroundColor(wui.RGB(255, 240, 200))
	inPanel.SetBounds(10, 10, 300, 20)
	panel.Add(inPanel)

	window.Show()
}
