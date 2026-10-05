// ToolTip example: hover over the controls.
package main

import "github.com/2dprototype/wui"

func main() {
	w := wui.NewWindow()
	w.SetTitle("ToolTip")
	w.SetInnerSize(300, 140)
	w.SetCenterOnShow(true)

	b := wui.NewButton()
	b.SetText("Save")
	b.SetBounds(20, 20, 100, 30)
	b.SetToolTip("Saves the document")
	w.Add(b)

	e := wui.NewEditLine()
	e.SetBounds(20, 70, 240, 25)
	e.SetToolTip("Type your name here")
	w.Add(e)

	w.Show()
}
