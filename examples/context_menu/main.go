// Context menu example: right-click the window or the button.
package main

import "github.com/2dprototype/wui"

func main() {
	w := wui.NewWindow()
	w.SetTitle("Context menu")
	w.SetInnerSize(360, 200)
	w.SetCenterOnShow(true)

	status := wui.NewLabel()
	status.SetBounds(10, 160, 340, 25)
	status.SetText("Right-click somewhere")
	w.Add(status)

	// Menu for the window background.
	bg := wui.NewPopupMenu()
	bg.Add(wui.NewMenuString("Say hello").SetOnClick(func() {
		status.SetText("Hello from the window menu")
	}))
	bg.Add(wui.NewMenuSeparator())
	more := wui.NewMenu("More")
	more.Add(wui.NewMenuString("Sub item").SetOnClick(func() {
		status.SetText("Sub item chosen")
	}))
	bg.Add(more)
	w.SetContextMenu(bg)

	// Separate menu for the button.
	b := wui.NewButton()
	b.SetText("Right-click me")
	b.SetBounds(110, 60, 140, 32)
	menu := wui.NewPopupMenu()
	menu.Add(wui.NewMenuString("Button action").SetOnClick(func() {
		status.SetText("Button menu action")
	}))
	b.SetContextMenu(menu)
	w.Add(b)

	w.Show()
}
