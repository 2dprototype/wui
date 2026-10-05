// Canvas mouse example: draw freehand lines with the left mouse button, clear
// with the right button.
package main

import "github.com/2dprototype/wui"

func main() {
	w := wui.NewWindow()
	w.SetTitle("Paint with the mouse")
	w.SetInnerSize(500, 350)
	w.SetCenterOnShow(true)

	var strokes [][]wui.Point
	drawing := false

	pb := wui.NewPaintBox()
	pb.SetBounds(0, 0, 500, 350)
	pb.SetAnchors(wui.AnchorMinAndMax, wui.AnchorMinAndMax)

	pb.SetOnMouseDown(func(x, y int, b wui.MouseButton) {
		if b == wui.MouseButtonRight {
			strokes = nil
			pb.Paint()
			return
		}
		drawing = true
		strokes = append(strokes, []wui.Point{{X: int32(x), Y: int32(y)}})
	})
	pb.SetOnMouseMove(func(x, y int) {
		if drawing && len(strokes) > 0 {
			last := len(strokes) - 1
			strokes[last] = append(strokes[last], wui.Point{X: int32(x), Y: int32(y)})
			pb.Paint()
		}
	})
	pb.SetOnMouseUp(func(x, y int, b wui.MouseButton) {
		drawing = false
	})

	pb.SetOnPaint(func(c *wui.Canvas) {
		c.Clear(wui.RGB(255, 255, 255))
		c.SetStroke(wui.Stroke{Width: 4, Cap: wui.CapRound, Join: wui.JoinRound})
		for _, s := range strokes {
			if len(s) > 1 {
				c.Polyline(s, wui.RGB(30, 30, 160))
			}
		}
		c.SetStroke(wui.Stroke{})
		c.TextOut(8, 8, "Left button: draw   Right button: clear", wui.RGB(120, 120, 120))
	})
	w.Add(pb)

	w.Show()
}
