// GroupBox example: a captioned frame around related controls.
package main

import "github.com/2dprototype/wui"

func main() {
	w := wui.NewWindow()
	w.SetTitle("GroupBox")
	w.SetInnerSize(300, 180)
	w.SetCenterOnShow(true)

	// Add the group box first so the controls on top of it are drawn above.
	g := wui.NewGroupBox()
	g.SetText("Options")
	g.SetBounds(10, 10, 280, 120)
	w.Add(g)

	a := wui.NewCheckBox()
	a.SetText("Enable logging")
	a.SetBounds(25, 35, 200, 25)
	w.Add(a)

	b := wui.NewCheckBox()
	b.SetText("Start minimized")
	b.SetBounds(25, 70, 200, 25)
	w.Add(b)

	w.Show()
}
