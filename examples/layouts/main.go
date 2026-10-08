// Dock, Grid, HBox and VBox layouts. Resize the window to see them work.
package main

import "github.com/2dprototype/wui"

func main() {
	w := wui.NewWindow()
	w.SetTitle("layouts")
	w.SetInnerSize(640, 420)

	// DockLayout on the window: top bar, bottom bar, left list, rest fill.
	top := wui.NewLabel()
	top.SetText("Top (docked, keeps its height)")
	top.SetBounds(0, 0, 100, 24)
	bottom := wui.NewLabel()
	bottom.SetText("Bottom")
	bottom.SetBounds(0, 0, 100, 24)
	left := wui.NewButton()
	left.SetText("Left")
	left.SetBounds(0, 0, 120, 10)

	// The fill area holds a grid of 3 columns.
	grid := wui.NewPanel()
	grid.SetBorderStyle(wui.PanelBorderSunken)
	gl := wui.NewGridLayout(3, 6, 8)
	gl.SetColumnWeights(1, 2, 1)
	grid.SetLayout(gl)
	var wide *wui.Button
	for i := 1; i <= 7; i++ {
		b := wui.NewButton()
		b.SetText("Cell " + string(rune('0'+i)))
		grid.Add(b)
		if i == 2 {
			wide = b
		}
	}
	gl.SetSpan(wide, 2, 1) // this cell covers two columns

	dock := wui.NewDockLayout(4, 4)
	dock.Set(top, wui.DockTop).Set(bottom, wui.DockBottom).Set(left, wui.DockLeft).Set(grid, wui.DockFill)
	w.SetLayout(dock)
	w.Add(top)
	w.Add(bottom)
	w.Add(left)
	w.Add(grid)
	w.Show()
}
