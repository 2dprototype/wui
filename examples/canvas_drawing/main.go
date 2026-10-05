// Canvas drawing example: strokes, rounded rectangles, curves, gradients,
// transparency, polygons, stars and clipping.
package main

import "github.com/2dprototype/wui"

func main() {
	w := wui.NewWindow()
	w.SetTitle("Canvas drawing")
	w.SetInnerSize(640, 420)
	w.SetCenterOnShow(true)

	pb := wui.NewPaintBox()
	pb.SetBounds(0, 0, 640, 420)
	pb.SetAnchors(wui.AnchorMinAndMax, wui.AnchorMinAndMax)
	pb.SetOnPaint(func(c *wui.Canvas) {
		c.Clear(wui.RGB(250, 250, 250))

		// Wide and dashed lines.
		c.SetStroke(wui.Stroke{Width: 6, Cap: wui.CapRound})
		c.Line(20, 30, 200, 30, wui.RGB(200, 40, 40))
		c.SetStroke(wui.Stroke{Width: 3, Style: wui.LineDash})
		c.Line(20, 60, 200, 60, wui.RGB(40, 40, 200))
		c.SetStroke(wui.Stroke{Width: 2, Style: wui.LineDot})
		c.Line(20, 85, 200, 85, wui.RGB(40, 140, 40))
		c.SetStroke(wui.Stroke{}) // back to the default pen

		// Rounded rectangles, with and without outline.
		c.FillRoundRect(20, 110, 180, 60, 14, wui.RGB(90, 160, 230))
		c.SetStroke(wui.Stroke{Width: 3})
		c.FillRoundRectOutline(20, 185, 180, 60, 14, wui.RGB(255, 230, 150), wui.RGB(160, 110, 0))
		c.SetStroke(wui.Stroke{})

		// Gradients.
		c.FillGradientH(230, 20, 180, 60, wui.RGB(255, 80, 80), wui.RGB(255, 220, 80))
		c.FillGradientV(230, 95, 180, 60, wui.RGB(60, 60, 200), wui.RGB(160, 220, 255))

		// Transparency: two overlapping translucent squares.
		c.FillRect(230, 175, 80, 70, wui.RGB(220, 60, 60))
		c.FillRectAlpha(270, 190, 100, 70, wui.RGB(40, 90, 220), 120)

		// Bezier and smooth curves.
		c.SetStroke(wui.Stroke{Width: 3})
		c.Bezier([]wui.Point{{440, 100}, {480, 10}, {540, 10}, {600, 100}}, wui.RGB(170, 40, 170))
		c.Curve([]wui.Point{{440, 150}, {480, 120}, {520, 190}, {560, 130}, {610, 170}}, wui.RGB(0, 130, 130))
		c.SetStroke(wui.Stroke{})

		// Polygons, stars and a pixel.
		tri := wui.RegularPolygonPoints(60, 320, 45, 3, 0)
		c.FillPolygonOutline(tri, wui.RGB(120, 200, 120), wui.RGB(20, 100, 20))
		hex := wui.RegularPolygonPoints(160, 320, 45, 6, 0)
		c.DrawPolygon(hex, wui.RGB(100, 100, 100))
		star := wui.StarPoints(270, 320, 48, 20, 5, 0)
		c.FillPolygonOutline(star, wui.RGB(255, 210, 40), wui.RGB(180, 120, 0))
		c.SetPixel(10, 10, wui.RGB(0, 0, 0))

		// Clipping to an ellipse: the gradient is only visible inside it.
		c.PushDrawEllipse(380, 250, 220, 140)
		c.FillGradientH(380, 250, 220, 140, wui.RGB(255, 0, 128), wui.RGB(0, 200, 255))
		c.TextRectFormat(380, 250, 220, 140, "Clipped", wui.FormatCenter, wui.RGB(255, 255, 255))
		c.PopDrawRegion()

		c.TextRectEllipsis(20, 390, 300, 22, "A long text that is cut with an ellipsis when it does not fit", wui.RGB(0, 0, 0))
	})
	w.Add(pb)

	w.Show()
}
