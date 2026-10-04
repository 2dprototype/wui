// TabControl example: one Panel per tab, switched in OnChange.
package main

import "github.com/2dprototype/wui"

func main() {
	w := wui.NewWindow()
	w.SetTitle("TabControl")
	w.SetInnerSize(400, 260)
	w.SetCenterOnShow(true)

	tabs := wui.NewTabControl()
	tabs.SetBounds(10, 10, 380, 240)
	tabs.AddTab("General")
	tabs.AddTab("Network")
	tabs.AddTab("About")
	w.Add(tabs)

	// Panels are added after the tab control so they appear on top of it.
	x, y, width, height := tabs.ContentBounds()
	titles := []string{"General settings", "Network settings", "About this app"}
	var panels []*wui.Panel
	for i, t := range titles {
		p := wui.NewPanel()
		p.SetBounds(x+2, y+2, width-4, height-4)
		l := wui.NewLabel()
		l.SetText(t)
		l.SetBounds(10, 10, 250, 25)
		p.Add(l)
		p.SetVisible(i == 0)
		w.Add(p)
		panels = append(panels, p)
	}

	tabs.SetOnChange(func(index int) {
		for i, p := range panels {
			p.SetVisible(i == index)
		}
	})

	w.Show()
}
