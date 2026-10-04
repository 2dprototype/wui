// Timer example: a clock updated every second plus a start/stop button.
package main

import (
	"time"

	"github.com/2dprototype/wui"
)

func main() {
	w := wui.NewWindow()
	w.SetTitle("Timer")
	w.SetInnerSize(260, 120)
	w.SetCenterOnShow(true)

	label := wui.NewLabel()
	label.SetAlignment(wui.AlignCenter)
	label.SetText(time.Now().Format("15:04:05"))
	label.SetBounds(30, 15, 200, 30)
	w.Add(label)

	t := w.AddTimer(1000, func() {
		label.SetText(time.Now().Format("15:04:05"))
	})

	b := wui.NewButton()
	b.SetText("Stop")
	b.SetBounds(80, 65, 100, 30)
	b.SetOnClick(func() {
		if t.Running() {
			t.Stop()
			b.SetText("Start")
		} else {
			t.Start()
			b.SetText("Stop")
		}
	})
	w.Add(b)

	w.Show()
}
