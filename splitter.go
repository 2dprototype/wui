package wui

// Splitter is a draggable divider between two controls. Place it between
// them; dragging it resizes both.
//
//	s := wui.NewSplitter(left, right, wui.SplitVertical)
//	s.SetBounds(200, 0, 6, 300) // a thin bar between left and right
//
// A SplitVertical splitter is a vertical bar that divides a left from a right
// control, a SplitHorizontal one is a horizontal bar between a top and a
// bottom control. The two controls must be next to the splitter (in the same
// container); the splitter itself must be added to that container too.
type Splitter struct {
	*PaintBox
	first, second Control
	dir           SplitDirection
	minFirst      int
	minSecond     int
	downX, downY  int
	dragging      bool
	onMove        func()
	back          Color
	hasBack       bool
}

// SplitDirection tells the direction of the splitter bar.
type SplitDirection int

const (
	// SplitVertical is a vertical bar, the controls are left and right of it.
	SplitVertical SplitDirection = iota
	// SplitHorizontal is a horizontal bar, the controls are above and below.
	SplitHorizontal
)

var _ Control = (*Splitter)(nil)

// NewSplitter creates a splitter for the two controls, first is the left or
// top one.
func NewSplitter(first, second Control, dir SplitDirection) *Splitter {
	s := &Splitter{
		PaintBox:  NewPaintBox(),
		first:     first,
		second:    second,
		dir:       dir,
		minFirst:  40,
		minSecond: 40,
	}
	if dir == SplitVertical {
		s.PaintBox.SetCursor(CursorSizeWE)
	} else {
		s.PaintBox.SetCursor(CursorSizeNS)
	}
	s.PaintBox.SetOnPaint(func(c *Canvas) {
		w, h := c.Size()
		if s.hasBack {
			c.Clear(s.back)
		} else {
			c.Clear(ColorButtonFace)
		}
		// A subtle grip in the middle of the bar.
		if dir == SplitVertical {
			c.Line(w/2, h/2-8, w/2, h/2+8, ColorButtonShadow)
		} else {
			c.Line(w/2-8, h/2, w/2+8, h/2, ColorButtonShadow)
		}
	})
	s.PaintBox.SetOnMouseDown(func(x, y int, b MouseButton) {
		if b == MouseButtonLeft {
			s.dragging = true
			s.downX, s.downY = x, y
		}
	})
	s.PaintBox.SetOnMouseUp(func(x, y int, b MouseButton) {
		s.dragging = false
	})
	s.PaintBox.SetOnMouseMove(func(x, y int) {
		if s.dragging {
			s.drag(x-s.downX, y-s.downY)
		}
	})
	return s
}

// SetMinSizes sets the smallest sizes (width for a vertical splitter, height
// for a horizontal one) of the first and second control while dragging. The
// default is 40 pixels each.
func (s *Splitter) SetMinSizes(first, second int) {
	s.minFirst, s.minSecond = first, second
}

// SetBackColor sets the color of the bar.
func (s *Splitter) SetBackColor(c Color) {
	s.back, s.hasBack = c, true
	s.Paint()
}

// SetOnMove sets the function called after the user moved the bar.
func (s *Splitter) SetOnMove(f func()) { s.onMove = f }

func (s *Splitter) drag(dx, dy int) {
	sx, sy, sw, sh := s.Bounds()
	fx, fy, fw, fh := s.first.Bounds()
	nx, ny, nw, nh := s.second.Bounds()
	if s.dir == SplitVertical {
		// Keep the right edge of the second control where it is.
		right := nx + nw
		pos := sx + dx
		if pos < fx+s.minFirst {
			pos = fx + s.minFirst
		}
		if pos > right-sw-s.minSecond {
			pos = right - sw - s.minSecond
		}
		if pos == sx {
			return
		}
		s.first.SetBounds(fx, fy, pos-fx, fh)
		s.SetBounds(pos, sy, sw, sh)
		s.second.SetBounds(pos+sw, ny, right-pos-sw, nh)
	} else {
		bottom := ny + nh
		pos := sy + dy
		if pos < fy+s.minFirst {
			pos = fy + s.minFirst
		}
		if pos > bottom-sh-s.minSecond {
			pos = bottom - sh - s.minSecond
		}
		if pos == sy {
			return
		}
		s.first.SetBounds(fx, fy, fw, pos-fy)
		s.SetBounds(sx, pos, sw, sh)
		s.second.SetBounds(nx, pos+sh, nw, bottom-pos-sh)
	}
	if s.onMove != nil {
		s.onMove()
	}
}
