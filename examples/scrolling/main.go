// ScrollPanel (scrolls controls), Splitter, ScrollBar and ImageView.
package main

import (
	"fmt"
	"image"
	"image/color"

	"github.com/2dprototype/wui"
)

func gradient() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 200, 120))
	for y := 0; y < 120; y++ {
		for x := 0; x < 200; x++ {
			img.SetRGBA(x, y, color.RGBA{uint8(x), uint8(y * 2), 160, 255})
		}
	}
	return img
}

func main() {
	w := wui.NewWindow()
	w.SetTitle("scrolling")
	w.SetInnerSize(700, 420)

	// Left: a ScrollPanel with 30 rows, far more than fit.
	sp := wui.NewScrollPanel()
	sp.SetBounds(0, 0, 300, 420)
	for i := 0; i < 30; i++ {
		l := wui.NewLabel()
		l.SetText(fmt.Sprintf("Row %d", i+1))
		l.SetBounds(8, 8+i*30, 60, 22)
		sp.Add(l)
		e := wui.NewEditLine()
		e.SetText("value")
		e.SetBounds(76, 8+i*30, 190, 22)
		sp.Add(e)
	}
	w.Add(sp)

	// Right: an image whose view is resized by dragging the splitter, with a
	// separate scroll bar underneath that changes the fit mode.
	iv := wui.NewImageView()
	iv.SetBounds(306, 0, 394, 380)
	iv.SetImage(gradient())
	iv.SetMode(wui.ImageFit)

	split := wui.NewSplitter(sp, iv, wui.SplitVertical)
	split.SetBounds(300, 0, 6, 420)

	bar := wui.NewScrollBar(false)
	bar.SetBounds(306, 390, 394, 18)
	bar.SetRange(0, 4)
	bar.SetPage(1)
	bar.SetOnChange(func(pos int) {
		iv.SetMode(wui.ImageMode(pos)) // Normal, Center, Stretch, Fit, Fill
	})

	w.Add(split)
	w.Add(iv)
	w.Add(bar)
	w.Show()
}
