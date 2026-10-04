// ColorDialog example: pick a color and show it in a paint box.
package main

import "github.com/2dprototype/wui"

func main() {
	w := wui.NewWindow()
	w.SetTitle("ColorDialog")
	w.SetInnerSize(300, 160)
	w.SetCenterOnShow(true)

	dlg := wui.NewColorDialog()
	dlg.SetColor(wui.RGB(255, 128, 0))

	preview := wui.NewPaintBox()
	preview.SetBounds(10, 10, 280, 80)
	color := dlg.Color()
	preview.SetOnPaint(func(c *wui.Canvas) {
		c.FillRect(0, 0, 280, 80, color)
	})
	w.Add(preview)

	b := wui.NewButton()
	b.SetText("Choose color...")
	b.SetBounds(95, 110, 110, 30)
	b.SetOnClick(func() {
		if dlg.Execute(w) {
			color = dlg.Color()
			preview.Paint()
		}
	})
	w.Add(b)

	w.Show()
}
