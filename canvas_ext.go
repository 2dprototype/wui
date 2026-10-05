package wui

import (
	"math"
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

// LineStyle is the dash pattern of lines, see Canvas.SetStroke.
type LineStyle int

const (
	LineSolid LineStyle = iota
	LineDash
	LineDot
	LineDashDot
	LineDashDotDot
)

// LineCap is the shape of the ends of wide lines.
type LineCap int

const (
	CapRound LineCap = iota
	CapSquare
	CapFlat
)

// LineJoin is the shape of the corners of wide lines.
type LineJoin int

const (
	JoinRound LineJoin = iota
	JoinBevel
	JoinMiter
)

// Stroke describes how outlines are drawn. The zero value is a solid line of
// width 1. A Width of 0 or 1 draws the fast default pen.
type Stroke struct {
	Width int
	Style LineStyle
	Cap   LineCap
	Join  LineJoin
}

type canvasState struct {
	dc      uintptr
	stroke  Stroke
	regions int
}

// SetStroke sets the pen used by all outline drawing functions (DrawRect,
// Line, DrawEllipse, Polyline, Arc, DrawPie, DrawPolygon, DrawRoundRect,
// Bezier, Curve and the *Outline functions).
func (c *Canvas) SetStroke(s Stroke) { c.stroke = s }

// Stroke returns the current stroke.
func (c *Canvas) Stroke() Stroke { return c.stroke }

// SetLineWidth sets the outline width in pixels.
func (c *Canvas) SetLineWidth(width int) { c.stroke.Width = width }

// SetLineStyle sets the dash pattern of outlines.
func (c *Canvas) SetLineStyle(s LineStyle) { c.stroke.Style = s }

// usePen selects a pen of the given color and the current stroke and returns a
// function that restores the default pen and frees the created one.
func (c *Canvas) usePen(color Color) func() {
	s := c.stroke
	if s.Width <= 1 && s.Style == LineSolid {
		w32.SelectObject(c.hdc, w32.GetStockObject(w32.DC_PEN))
		w32.SetDCPenColor(c.hdc, w32.COLORREF(color))
		return func() {}
	}
	width := s.Width
	if width < 1 {
		width = 1
	}
	var style uint = w32.PS_GEOMETRIC
	switch s.Style {
	case LineDash:
		style |= w32.PS_DASH
	case LineDot:
		style |= w32.PS_DOT
	case LineDashDot:
		style |= w32.PS_DASHDOT
	case LineDashDotDot:
		style |= w32.PS_DASHDOTDOT
	default:
		style |= w32.PS_SOLID
	}
	switch s.Cap {
	case CapSquare:
		style |= w32.PS_ENDCAP_SQUARE
	case CapFlat:
		style |= w32.PS_ENDCAP_FLAT
	default:
		style |= w32.PS_ENDCAP_ROUND
	}
	switch s.Join {
	case JoinBevel:
		style |= w32.PS_JOIN_BEVEL
	case JoinMiter:
		style |= w32.PS_JOIN_MITER
	default:
		style |= w32.PS_JOIN_ROUND
	}
	lb := w32.LOGBRUSH{LbStyle: w32.BS_SOLID, LbColor: w32.COLORREF(color)}
	pen := w32.ExtCreatePen(style, uint(width), &lb, 0, nil)
	if pen == 0 {
		w32.SelectObject(c.hdc, w32.GetStockObject(w32.DC_PEN))
		w32.SetDCPenColor(c.hdc, w32.COLORREF(color))
		return func() {}
	}
	w32.SelectObject(c.hdc, w32.HGDIOBJ(pen))
	return func() {
		w32.SelectObject(c.hdc, w32.GetStockObject(w32.DC_PEN))
		w32.DeleteObject(w32.HGDIOBJ(pen))
	}
}

func (c *Canvas) stockPen(color Color) {
	w32.SelectObject(c.hdc, w32.GetStockObject(w32.DC_PEN))
	w32.SetDCPenColor(c.hdc, w32.COLORREF(color))
}

func (c *Canvas) selectFill(color Color) {
	w32.SelectObject(c.hdc, w32.GetStockObject(w32.DC_BRUSH))
	w32.SetDCBrushColor(c.hdc, w32.COLORREF(color))
}

func (c *Canvas) selectNoBrush() {
	w32.SelectObject(c.hdc, w32.GetStockObject(w32.NULL_BRUSH))
}

// Clear fills the whole canvas with the given color.
func (c *Canvas) Clear(color Color) {
	c.FillRect(0, 0, c.width, c.height, color)
}

// SetPixel sets a single pixel.
func (c *Canvas) SetPixel(x, y int, color Color) {
	xSetPixel.Call(uintptr(c.hdc), uintptr(x), uintptr(y), uintptr(color))
}

// Pixel returns the color of a single pixel (0xFFFFFFFF outside the canvas).
func (c *Canvas) Pixel(x, y int) Color {
	ret, _, _ := xGetPixel.Call(uintptr(c.hdc), uintptr(x), uintptr(y))
	return Color(ret)
}

func (c *Canvas) roundRect(x, y, width, height, radius int) {
	xRoundRect.Call(
		uintptr(c.hdc),
		uintptr(x), uintptr(y), uintptr(x+width), uintptr(y+height),
		uintptr(2*radius), uintptr(2*radius),
	)
}

// DrawRoundRect draws the outline of a rectangle with rounded corners.
func (c *Canvas) DrawRoundRect(x, y, width, height, radius int, color Color) {
	defer c.usePen(color)()
	c.selectNoBrush()
	c.roundRect(x, y, width, height, radius)
}

// FillRoundRect fills a rectangle with rounded corners.
func (c *Canvas) FillRoundRect(x, y, width, height, radius int, color Color) {
	c.stockPen(color)
	c.selectFill(color)
	c.roundRect(x, y, width, height, radius)
}

// FillRoundRectOutline fills a rounded rectangle and draws its outline in a
// different color using the current stroke.
func (c *Canvas) FillRoundRectOutline(x, y, width, height, radius int, fill, outline Color) {
	defer c.usePen(outline)()
	c.selectFill(fill)
	c.roundRect(x, y, width, height, radius)
}

// FillRectOutline fills a rectangle and draws its outline in a different color.
func (c *Canvas) FillRectOutline(x, y, width, height int, fill, outline Color) {
	defer c.usePen(outline)()
	c.selectFill(fill)
	w32.Rectangle(c.hdc, x, y, x+width, y+height)
}

// FillEllipseOutline fills an ellipse and draws its outline in a different color.
func (c *Canvas) FillEllipseOutline(x, y, width, height int, fill, outline Color) {
	defer c.usePen(outline)()
	c.selectFill(fill)
	w32.Ellipse(c.hdc, x, y, x+width, y+height)
}

// DrawPolygon draws only the outline of a polygon.
func (c *Canvas) DrawPolygon(p []Point, color Color) {
	if len(p) < 2 {
		return
	}
	defer c.usePen(color)()
	c.selectNoBrush()
	w32.PolygonMem(c.hdc, unsafe.Pointer(&p[0]), len(p))
}

// FillPolygonOutline fills a polygon and draws its outline in a different color.
func (c *Canvas) FillPolygonOutline(p []Point, fill, outline Color) {
	if len(p) < 2 {
		return
	}
	defer c.usePen(outline)()
	c.selectFill(fill)
	w32.PolygonMem(c.hdc, unsafe.Pointer(&p[0]), len(p))
}

// Bezier draws connected cubic Bezier curves. The first point is the start
// point, every following group of three points is two control points and an end
// point, so len(p) must be 3n+1.
func (c *Canvas) Bezier(p []Point, color Color) {
	if len(p) < 4 || (len(p)-1)%3 != 0 {
		return
	}
	defer c.usePen(color)()
	c.selectNoBrush()
	w32.PolyBezierMem(c.hdc, unsafe.Pointer(&p[0]), len(p))
}

// Curve draws a smooth curve through all given points (Catmull-Rom spline).
func (c *Canvas) Curve(p []Point, color Color) {
	if len(p) < 2 {
		return
	}
	if len(p) == 2 {
		c.Line(int(p[0].X), int(p[0].Y), int(p[1].X), int(p[1].Y), color)
		return
	}
	at := func(i int) Point {
		if i < 0 {
			i = 0
		}
		if i >= len(p) {
			i = len(p) - 1
		}
		return p[i]
	}
	pts := make([]Point, 0, 3*(len(p)-1)+1)
	pts = append(pts, p[0])
	for i := 0; i < len(p)-1; i++ {
		p0, p1, p2, p3 := at(i-1), at(i), at(i+1), at(i+2)
		c1 := Point{p1.X + (p2.X-p0.X)/6, p1.Y + (p2.Y-p0.Y)/6}
		c2 := Point{p2.X - (p3.X-p1.X)/6, p2.Y - (p3.Y-p1.Y)/6}
		pts = append(pts, c1, c2, p2)
	}
	c.Bezier(pts, color)
}

// FillRectAlpha fills a rectangle with a semi-transparent color. Alpha 0 is
// fully transparent, 255 is opaque.
func (c *Canvas) FillRectAlpha(x, y, width, height int, color Color, alpha uint8) {
	if width <= 0 || height <= 0 {
		return
	}
	mem := w32.CreateCompatibleDC(c.hdc)
	bmp := w32.CreateCompatibleBitmap(c.hdc, width, height)
	old := w32.SelectObject(mem, w32.HGDIOBJ(bmp))
	brush := w32.CreateSolidBrush(uint32(color))
	r := w32.RECT{Left: 0, Top: 0, Right: int32(width), Bottom: int32(height)}
	w32.FillRect(mem, &r, brush)
	w32.AlphaBlend(
		c.hdc, x, y, width, height,
		mem, 0, 0, width, height,
		w32.BLENDFUNC{
			BlendOp:             w32.AC_SRC_OVER,
			SourceConstantAlpha: alpha,
		},
	)
	w32.SelectObject(mem, old)
	w32.DeleteObject(w32.HGDIOBJ(bmp))
	w32.DeleteObject(w32.HGDIOBJ(brush))
	w32.DeleteDC(mem)
}

type gradientVertex struct {
	X, Y                       int32
	Red, Green, Blue, Alpha    uint16
}

type gradientRect struct {
	UpperLeft, LowerRight uint32
}

func (c *Canvas) fillGradient(x, y, width, height int, from, to Color, mode uintptr) {
	if width <= 0 || height <= 0 {
		return
	}
	vertex := func(px, py int, col Color) gradientVertex {
		return gradientVertex{
			X:     int32(px),
			Y:     int32(py),
			Red:   uint16(uint8(col)) << 8,
			Green: uint16(uint8(col>>8)) << 8,
			Blue:  uint16(uint8(col>>16)) << 8,
		}
	}
	v := [2]gradientVertex{
		vertex(x, y, from),
		vertex(x+width, y+height, to),
	}
	r := gradientRect{UpperLeft: 0, LowerRight: 1}
	xGradientFill.Call(
		uintptr(c.hdc),
		uintptr(unsafe.Pointer(&v[0])), 2,
		uintptr(unsafe.Pointer(&r)), 1,
		mode,
	)
}

// FillGradientH fills a rectangle with a horizontal gradient from left to right.
func (c *Canvas) FillGradientH(x, y, width, height int, left, right Color) {
	c.fillGradient(x, y, width, height, left, right, 0)
}

// FillGradientV fills a rectangle with a vertical gradient from top to bottom.
func (c *Canvas) FillGradientV(x, y, width, height int, top, bottom Color) {
	c.fillGradient(x, y, width, height, top, bottom, 1)
}

func (c *Canvas) drawImageBlend(img *Image, src, dest Rectangle, alpha uint8) {
	if img == nil {
		return
	}
	if src.Width == 0 {
		src.Width = img.width
	}
	if src.Height == 0 {
		src.Height = img.height
	}
	if dest.Width == 0 {
		dest.Width = src.Width
	}
	if dest.Height == 0 {
		dest.Height = src.Height
	}
	hdcMem := w32.CreateCompatibleDC(c.hdc)
	old := w32.SelectObject(hdcMem, w32.HGDIOBJ(img.bitmap))
	w32.AlphaBlend(
		c.hdc,
		dest.X, dest.Y, dest.Width, dest.Height,
		hdcMem,
		src.X, src.Y, src.Width, src.Height,
		w32.BLENDFUNC{
			BlendOp:             w32.AC_SRC_OVER,
			BlendFlags:          0,
			SourceConstantAlpha: alpha,
			AlphaFormat:         w32.AC_SRC_ALPHA,
		},
	)
	w32.SelectObject(hdcMem, old)
	w32.DeleteDC(hdcMem)
}

// DrawImageScaled draws the src part of the image stretched into dest. A zero
// width or height in src means the full image, in dest it means the src size.
func (c *Canvas) DrawImageScaled(img *Image, src, dest Rectangle) {
	c.drawImageBlend(img, src, dest, 255)
}

// DrawImageAlpha draws the src part of the image with an additional
// transparency, 0 is invisible and 255 is opaque.
func (c *Canvas) DrawImageAlpha(img *Image, src Rectangle, destX, destY int, alpha uint8) {
	c.drawImageBlend(img, src, Rectangle{X: destX, Y: destY}, alpha)
}

func (c *Canvas) pushRegion(r w32.HRGN) {
	if len(c.regions) > 0 {
		w32.CombineRgn(r, r, c.regions[len(c.regions)-1], w32.RGN_AND)
	}
	c.regions = append(c.regions, r)
	w32.SelectClipRgn(c.hdc, r)
}

// PushDrawEllipse restricts drawing to an ellipse. Undo with PopDrawRegion.
func (c *Canvas) PushDrawEllipse(x, y, width, height int) {
	ret, _, _ := xCreateEllipticRgn.Call(
		uintptr(x), uintptr(y), uintptr(x+width), uintptr(y+height),
	)
	c.pushRegion(w32.HRGN(ret))
}

// PushDrawRoundRect restricts drawing to a rounded rectangle. Undo with
// PopDrawRegion.
func (c *Canvas) PushDrawRoundRect(x, y, width, height, radius int) {
	ret, _, _ := xCreateRoundRectRgn.Call(
		uintptr(x), uintptr(y), uintptr(x+width), uintptr(y+height),
		uintptr(2*radius), uintptr(2*radius),
	)
	c.pushRegion(w32.HRGN(ret))
}

// PushDrawPolygon restricts drawing to a polygon. Undo with PopDrawRegion.
func (c *Canvas) PushDrawPolygon(p []Point) {
	if len(p) < 3 {
		return
	}
	ret, _, _ := xCreatePolygonRgn.Call(
		uintptr(unsafe.Pointer(&p[0])), uintptr(len(p)), 1,
	)
	c.pushRegion(w32.HRGN(ret))
}

// Save remembers the current stroke, font and clip region. Restore goes back to
// it. Calls can be nested.
func (c *Canvas) Save() {
	ret, _, _ := xSaveDC.Call(uintptr(c.hdc))
	c.states = append(c.states, canvasState{
		dc:      ret,
		stroke:  c.stroke,
		regions: len(c.regions),
	})
}

// Restore returns to the state of the last Save.
func (c *Canvas) Restore() {
	n := len(c.states)
	if n == 0 {
		return
	}
	s := c.states[n-1]
	c.states = c.states[:n-1]
	for len(c.regions) > s.regions {
		c.PopDrawRegion()
	}
	xRestoreDC.Call(uintptr(c.hdc), s.dc)
	c.stroke = s.stroke
}

// DrawFocusRect draws a dotted focus rectangle.
func (c *Canvas) DrawFocusRect(x, y, width, height int) {
	r := w32.RECT{
		Left: int32(x), Top: int32(y),
		Right: int32(x + width), Bottom: int32(y + height),
	}
	xDrawFocusRect.Call(uintptr(c.hdc), uintptr(unsafe.Pointer(&r)))
}

// TextRectEllipsis draws a single line of text, vertically centered in the
// rectangle, ending with "..." if it does not fit.
func (c *Canvas) TextRectEllipsis(x, y, width, height int, s string, color Color) {
	w32.SetBkMode(c.hdc, w32.TRANSPARENT)
	c.selectNoBrush()
	w32.SetTextColor(c.hdc, w32.COLORREF(color))
	r := w32.RECT{
		Left: int32(x), Top: int32(y),
		Right: int32(x + width), Bottom: int32(y + height),
	}
	w32.DrawText(
		c.hdc, s, &r,
		w32.DT_SINGLELINE|w32.DT_VCENTER|w32.DT_END_ELLIPSIS|w32.DT_NOPREFIX|w32.DT_LEFT,
	)
	w32.SetBkMode(c.hdc, w32.OPAQUE)
}

func roundToInt32(f float64) int32 {
	return int32(math.Floor(f + 0.5))
}

// RegularPolygonPoints returns the corners of a regular polygon (3 sides is a
// triangle, 6 a hexagon, ...). With rotationDegrees 0 the first corner points
// up. Use the result with Polygon, DrawPolygon or FillPolygonOutline.
func RegularPolygonPoints(cx, cy, radius float64, sides int, rotationDegrees float64) []Point {
	if sides < 3 {
		return nil
	}
	pts := make([]Point, sides)
	for i := range pts {
		a := (rotationDegrees - 90 + 360*float64(i)/float64(sides)) * math.Pi / 180
		pts[i] = Point{
			X: roundToInt32(cx + radius*math.Cos(a)),
			Y: roundToInt32(cy + radius*math.Sin(a)),
		}
	}
	return pts
}

// StarPoints returns the corners of a star with the given number of tips.
func StarPoints(cx, cy, outerRadius, innerRadius float64, tips int, rotationDegrees float64) []Point {
	if tips < 3 {
		return nil
	}
	pts := make([]Point, 2*tips)
	for i := range pts {
		r := outerRadius
		if i%2 == 1 {
			r = innerRadius
		}
		a := (rotationDegrees - 90 + 180*float64(i)/float64(tips)) * math.Pi / 180
		pts[i] = Point{
			X: roundToInt32(cx + r*math.Cos(a)),
			Y: roundToInt32(cy + r*math.Sin(a)),
		}
	}
	return pts
}
