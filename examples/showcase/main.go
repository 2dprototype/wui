// Showcase of the controls and features added to wui: native tool bar with
// icons, ListView, RichEdit, layouts, ScrollPanel, Invoke from a goroutine,
// default/cancel buttons, task dialog and dark mode.
package main

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"github.com/2dprototype/wui"
)

func square(c color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func main() {
	wui.EnableDPIAwareness()

	w := wui.NewWindow()
	w.SetTitle("wui showcase")
	w.SetInnerSize(820, 520)

	icons := wui.NewImageList(16, 16)
	green, _ := icons.AddImage(square(color.RGBA{60, 170, 80, 255}))
	blue, _ := icons.AddImage(square(color.RGBA{60, 110, 200, 255}))
	red, _ := icons.AddImage(square(color.RGBA{210, 60, 60, 255}))

	status := wui.NewLabel()
	status.SetText("Ready")

	// The list on the left.
	lv := wui.NewListView()
	lv.SetSmallImages(icons)
	lv.AddColumn("Name", 140)
	lv.AddColumn("Size", 70)
	lv.SetColumnAlignment(1, wui.ColumnRight)
	lv.SetCheckBoxes(true)
	lv.SetSortable(true)
	lv.SetMultiSelect(true)
	for i := 1; i <= 8; i++ {
		it := lv.AddItem(fmt.Sprintf("file%d.txt", i), fmt.Sprint(i*i*7))
		it.SetImage((i % 3))
	}
	lv.SetOnSelect(func(index int) {
		status.SetText(fmt.Sprintf("selected row %d", index))
	})

	// The editor on the right.
	rich := wui.NewRichEdit()
	rich.SetText("Select some text and use the tool bar.\r\nLinks like https://github.com are clickable.")
	rich.SetOnLinkClick(func(url string) { wui.OpenURL(url) })

	// The tool bar.
	tb := wui.NewNativeToolBar()
	tb.SetImages(icons)
	tb.SetBounds(0, 0, 820, 30)
	tb.AddButton("Add", green, func() {
		lv.AddItem(fmt.Sprintf("new%d.txt", lv.Count()+1), "0")
	})
	tb.AddButton("Dialog", blue, func() {
		wui.TaskDialog(w, "wui", "Task dialog", "The modern message box.", wui.TaskOK, wui.TaskIconInformation)
	})
	tb.AddSeparator()
	tb.AddToggle("Bold", red, func(on bool) { rich.SetSelectionBold(on) })
	tb.AddButton("Job", green, func() {
		status.SetText("working...")
		go func() {
			for i := 1; i <= 5; i++ {
				time.Sleep(300 * time.Millisecond)
				n := i
				// Controls may only be touched on the GUI thread.
				w.Invoke(func() { status.SetText(fmt.Sprintf("job %d/5", n)) })
			}
			w.Invoke(func() { status.SetText("job done") })
		}()
	})

	// A panel laying its children out side by side.
	middle := wui.NewPanel()
	middle.SetBounds(0, 0, 820, 400)
	hbox := wui.NewHBox(6, 0)
	hbox.SetWeight(lv, 1).SetWeight(rich, 2)
	middle.SetLayout(hbox)
	middle.Add(lv)
	middle.Add(rich)

	status.SetBounds(0, 0, 820, 22)

	// The window stacks tool bar, middle part and status line.
	vbox := wui.NewVBox(4, 4)
	vbox.SetWeight(middle, 1)
	w.SetLayout(vbox)
	w.Add(tb)
	w.Add(middle)
	w.Add(status)

	w.SetOnShow(func() { w.SetDarkTitleBar(false) })
	w.Show()
}
