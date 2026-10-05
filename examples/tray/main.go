package main

import "github.com/2dprototype/wui"

func main() {
	window := wui.NewWindow()
	window.SetTitle("Tray example")
	window.SetInnerSize(320, 120)
	window.SetCenterOnShow(true)

	label := wui.NewLabel()
	label.SetText("Minimize or close me: I live in the tray.")
	label.SetBounds(10, 10, 300, 20)
	window.Add(label)

	quit := wui.NewMenuString("Quit")
	quit.SetOnClick(window.Quit)
	show := wui.NewMenuString("Show")
	show.SetOnClick(window.RestoreFromTray)
	menu := wui.NewPopupMenu()
	menu.Add(show)
	menu.Add(wui.NewMenuSeparator())
	menu.Add(quit)

	tray := wui.NewTrayIcon()
	tray.SetToolTip("wui tray example")
	tray.SetMenu(menu)
	window.SetTrayIcon(tray)
	window.SetMinimizeToTray(true)
	window.SetCloseToTray(true)

	hello := wui.NewButton()
	hello.SetText("Notify")
	hello.SetBounds(10, 50, 90, 25)
	hello.SetOnClick(func() {
		tray.ShowBalloon("wui", "Hello from the tray!", wui.BalloonInfo)
	})
	window.Add(hello)

	window.Show()
}
