// Center example: the main window centers on the screen, the modal dialog
// centers over its parent.
package main

import "github.com/2dprototype/wui"

func main() {
	w := wui.NewWindow()
	w.SetTitle("Main window")
	w.SetInnerSize(400, 250)
	w.SetCenterOnShow(true) // centered on the screen

	b := wui.NewButton()
	b.SetText("Open modal dialog")
	b.SetBounds(125, 100, 150, 30)
	b.SetOnClick(func() {
		d := wui.NewWindow()
		d.SetTitle("Modal")
		d.SetInnerSize(220, 100)
		d.SetCenterOnShow(true) // centered over the main window
		l := wui.NewLabel()
		l.SetText("I am centered on my parent.")
		l.SetBounds(10, 30, 200, 25)
		d.Add(l)
		d.ShowModal()
	})
	w.Add(b)

	b2 := wui.NewButton()
	b2.SetText("Center main window")
	b2.SetBounds(125, 140, 150, 30)
	b2.SetOnClick(func() {
		w.CenterOnScreen()
	})
	w.Add(b2)

	w.Show()
}
