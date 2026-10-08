// System features: single instance, global hot key, fullscreen, taskbar
// progress, flashing, clipboard images and files, and a virtual ListView with
// 1,000,000 rows.
package main

import (
	"fmt"
	"image"
	"image/color"
	"os"

	"github.com/2dprototype/wui"
)

func main() {
	if !wui.SingleInstance("com.example.wui.system_features") {
		wui.MessageBox("wui", "The program is already running.")
		os.Exit(0)
	}

	w := wui.NewWindow()
	w.SetTitle("system features")
	w.SetInnerSize(620, 460)

	out := wui.NewLabel()
	out.SetBounds(10, 430, 600, 22)

	// Virtual list: no row objects exist, text is produced on demand.
	lv := wui.NewListView()
	lv.AddColumn("Row", 120)
	lv.AddColumn("Square", 160)
	lv.SetVirtual(1000000, func(row, col int) string {
		if col == 0 {
			return fmt.Sprint(row)
		}
		return fmt.Sprint(row * row)
	})
	lv.SetBounds(10, 150, 600, 270)

	// Global hot key Ctrl+Shift+F9 works even when the window is in the
	// background.
	w.RegisterHotKey(wui.ModControl|wui.ModShift, 0x78, func() {
		out.SetText("global hot key pressed")
		w.Flash(3)
	})

	full := wui.NewButton()
	full.SetText("Fullscreen on/off")
	full.SetBounds(10, 10, 150, 28)
	full.SetOnClick(func() { w.SetFullscreen(!w.Fullscreen()) })

	progress := 0.0
	task := wui.NewButton()
	task.SetText("Taskbar progress +10%")
	task.SetBounds(170, 10, 170, 28)
	task.SetOnClick(func() {
		progress += 0.1
		if progress > 1 {
			progress = 0
			w.SetTaskbarProgress(wui.TaskbarNoProgress, 0)
			return
		}
		w.SetTaskbarProgress(wui.TaskbarNormal, progress)
	})

	img := wui.NewButton()
	img.SetText("Copy image to clipboard")
	img.SetBounds(10, 46, 170, 28)
	img.SetOnClick(func() {
		pic := image.NewRGBA(image.Rect(0, 0, 64, 64))
		for y := 0; y < 64; y++ {
			for x := 0; x < 64; x++ {
				pic.SetRGBA(x, y, color.RGBA{uint8(x * 4), uint8(y * 4), 128, 255})
			}
		}
		out.SetText(fmt.Sprint("image copied: ", wui.SetClipboardImage(pic)))
	})
	paste := wui.NewButton()
	paste.SetText("Paste image / files")
	paste.SetBounds(190, 46, 150, 28)
	paste.SetOnClick(func() {
		if pic, ok := wui.ClipboardImage(); ok {
			b := pic.Bounds()
			out.SetText(fmt.Sprintf("clipboard image %dx%d", b.Dx(), b.Dy()))
		} else if files, ok := wui.ClipboardFiles(); ok {
			out.SetText(fmt.Sprintf("%d file(s), first: %s", len(files), files[0]))
		} else {
			out.SetText("clipboard has no image or files")
		}
	})

	info := wui.NewLabel()
	info.SetBounds(10, 90, 600, 40)
	info.SetText("Press Ctrl+Shift+F9 from any program. DPI is shown after start.")

	w.SetOnShow(func() {
		info.SetText(fmt.Sprintf("Press Ctrl+Shift+F9 from any program. DPI: %d (%.0f%%)", w.DPI(), w.ScaleFactor()*100))
	})
	w.SetOnStateChange(func(s wui.WindowState) { out.SetText(fmt.Sprint("state changed: ", s)) })

	for _, c := range []wui.Control{full, task, img, paste, info, lv, out} {
		w.Add(c)
	}
	w.Show()
}
