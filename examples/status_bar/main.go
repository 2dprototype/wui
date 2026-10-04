// StatusBar example: a multi-part status bar at the bottom of the window.
package main

import (
	"fmt"

	"github.com/2dprototype/wui"
)

func main() {
	w := wui.NewWindow()
	w.SetTitle("StatusBar")
	w.SetInnerSize(400, 200)
	w.SetCenterOnShow(true)

	status := wui.NewStatusBar()
	status.SetBounds(0, 178, 400, 22)
	status.SetAnchors(wui.AnchorMinAndMax, wui.AnchorMax)
	status.SetParts(250, -1)
	status.SetPartText(0, "Ready")
	status.SetPartText(1, "Clicks: 0")
	w.Add(status)

	clicks := 0
	b := wui.NewButton()
	b.SetText("Click me")
	b.SetBounds(150, 70, 100, 30)
	b.SetOnClick(func() {
		clicks++
		status.SetPartText(0, "Button clicked")
		status.SetPartText(1, fmt.Sprintf("Clicks: %d", clicks))
	})
	w.Add(b)

	w.Show()
}
