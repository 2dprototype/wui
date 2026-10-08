package wui

import (
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

const (
	sbHorz = 0
	sbVert = 1
	sbCtl  = 2

	sifRange = 0x1
	sifPage  = 0x2
	sifPos   = 0x4
	sifTrack = 0x10
	sifAll   = 0x17

	sbLineUp        = 0
	sbLineDown      = 1
	sbPageUp        = 2
	sbPageDown      = 3
	sbThumbPosition = 4
	sbThumbTrack    = 5
	sbTop           = 6
	sbBottom        = 7

	wsHScroll = 0x00100000
	wsVScroll = 0x00200000
)

type scrollInfo struct {
	cbSize    uint32
	fMask     uint32
	nMin      int32
	nMax      int32
	nPage     uint32
	nPos      int32
	nTrackPos int32
}

func setScrollInfo(h w32.HWND, bar int, min, max, page, pos int) {
	si := scrollInfo{
		fMask: sifRange | sifPage | sifPos,
		nMin:  int32(min), nMax: int32(max), nPage: uint32(page), nPos: int32(pos),
	}
	si.cbSize = uint32(unsafe.Sizeof(si))
	nxSetScrollInfo.Call(uintptr(h), uintptr(bar), uintptr(unsafe.Pointer(&si)), 1)
}

func getScrollInfo(h w32.HWND, bar int) scrollInfo {
	si := scrollInfo{fMask: sifAll}
	si.cbSize = uint32(unsafe.Sizeof(si))
	nxGetScrollInfo.Call(uintptr(h), uintptr(bar), uintptr(unsafe.Pointer(&si)))
	return si
}

// scrollRequest turns a scroll bar request code into the new position.
func scrollRequest(req int, si scrollInfo, line int) int {
	pos := int(si.nPos)
	switch req {
	case sbLineUp:
		pos -= line
	case sbLineDown:
		pos += line
	case sbPageUp:
		pos -= int(si.nPage)
	case sbPageDown:
		pos += int(si.nPage)
	case sbThumbPosition, sbThumbTrack:
		pos = int(si.nTrackPos)
	case sbTop:
		pos = int(si.nMin)
	case sbBottom:
		pos = int(si.nMax)
	}
	return pos
}

// ScrollBar is a stand-alone horizontal or vertical scroll bar. The
// position goes from the minimum to maximum-page+1, where page is the size of
// the visible part (the thumb). Use it for custom scrolled views; for
// scrolling a group of controls ScrollPanel does everything itself.
type ScrollBar struct {
	control
	vertical          bool
	min, max, page    int
	pos               int
	line              int
	onChange          func(pos int)
}

var _ Control = (*ScrollBar)(nil)

// NewScrollBar creates a scroll bar with the range 0 to 100 and a page of 10.
func NewScrollBar(vertical bool) *ScrollBar {
	return &ScrollBar{vertical: vertical, max: 100, page: 10, line: 1}
}

func (*ScrollBar) canFocus() bool { return true }
func (*ScrollBar) eatsTabs() bool { return false }

func (s *ScrollBar) create(id int) {
	var style uint = w32.WS_TABSTOP
	if s.vertical {
		style |= 1 // SBS_VERT
	}
	s.control.create(id, 0, "SCROLLBAR", style)
	setScrollInfo(s.handle, sbCtl, s.min, s.max, s.page, s.pos)
}

// SetRange sets the smallest and largest value, including the page.
func (s *ScrollBar) SetRange(min, max int) {
	s.min, s.max = min, max
	s.apply()
}

// SetPage sets the size of the visible part, which is the size of the thumb.
func (s *ScrollBar) SetPage(page int) {
	s.page = page
	s.apply()
}

// SetLineSize sets how far the arrow buttons move, 1 by default.
func (s *ScrollBar) SetLineSize(n int) { s.line = n }

// Position returns the current position.
func (s *ScrollBar) Position() int { return s.pos }

// SetPosition moves the thumb.
func (s *ScrollBar) SetPosition(pos int) {
	s.pos = s.clamp(pos)
	s.apply()
}

// SetOnChange sets the function called with the new position when the user
// scrolls.
func (s *ScrollBar) SetOnChange(f func(pos int)) { s.onChange = f }

func (s *ScrollBar) clamp(pos int) int {
	hi := s.max
	if s.page > 0 {
		hi = s.max - s.page + 1
	}
	if pos > hi {
		pos = hi
	}
	if pos < s.min {
		pos = s.min
	}
	return pos
}

func (s *ScrollBar) apply() {
	if s.handle != 0 {
		setScrollInfo(s.handle, sbCtl, s.min, s.max, s.page, s.pos)
	}
}

func (s *ScrollBar) handleScroll(req int) {
	si := getScrollInfo(s.handle, sbCtl)
	pos := s.clamp(scrollRequest(req, si, s.line))
	if pos != s.pos || req == sbThumbTrack {
		s.pos = pos
		s.apply()
		if s.onChange != nil {
			s.onChange(pos)
		}
	}
}

// ScrollPanel is a Panel that shows scroll bars when its children do not fit,
// like a scrolled window. Add controls at positions in a large virtual area;
// the panel scrolls them into view. The mouse wheel scrolls it vertically.
//
// While scrolling, the Bounds of the children change because they are moved.
// ContentPosition tells how far the content is scrolled, adding it to a
// child's Bounds gives its position in the virtual area.
type ScrollPanel struct {
	Panel
	autoContent        bool
	contentW, contentH int
	posX, posY         int
}

var _ Control = (*ScrollPanel)(nil)
var _ Container = (*ScrollPanel)(nil)

// NewScrollPanel creates a scrolling panel whose virtual size follows its
// children.
func NewScrollPanel() *ScrollPanel {
	return &ScrollPanel{autoContent: true}
}

func (s *ScrollPanel) create(id int) {
	s.Panel.create(id)
	style := w32.GetWindowLong(s.handle, w32.GWL_STYLE)
	w32.SetWindowLong(s.handle, w32.GWL_STYLE, style|wsHScroll|wsVScroll)
	w32.SetWindowPos(s.handle, 0, 0, 0, 0, 0,
		w32.SWP_FRAMECHANGED|w32.SWP_NOMOVE|w32.SWP_NOSIZE|w32.SWP_NOZORDER|w32.SWP_NOACTIVATE)
	s.control.ev.raw = s.rawProc
	s.updateScroll()
}

// SetContentSize fixes the size of the virtual area instead of following the
// children.
func (s *ScrollPanel) SetContentSize(width, height int) {
	s.autoContent = false
	s.contentW, s.contentH = width, height
	s.updateScroll()
}

// SetAutoContentSize makes the virtual area as large as the children need,
// the default.
func (s *ScrollPanel) SetAutoContentSize() {
	s.autoContent = true
	s.updateScroll()
}

// ContentPosition returns how far the content is scrolled.
func (s *ScrollPanel) ContentPosition() (x, y int) { return s.posX, s.posY }

// ContentSize returns the size of the virtual area.
func (s *ScrollPanel) ContentSize() (w, h int) { return s.contentW, s.contentH }

// Add adds a control and updates the scroll bars.
func (s *ScrollPanel) Add(c Control) {
	s.Panel.Add(c)
	s.updateScroll()
}

// Remove removes a control and updates the scroll bars.
func (s *ScrollPanel) Remove(c Control) {
	s.Panel.Remove(c)
	s.updateScroll()
}

// SetBounds moves or resizes the panel and updates the scroll bars.
func (s *ScrollPanel) SetBounds(x, y, width, height int) {
	s.Panel.SetBounds(x, y, width, height)
	s.updateScroll()
}

// Update recalculates the scroll bars, call it after you moved or resized
// children yourself.
func (s *ScrollPanel) Update() { s.updateScroll() }

func (s *ScrollPanel) viewSize() (w, h int) {
	if s.handle == 0 {
		return 0, 0
	}
	r := w32.GetClientRect(s.handle)
	return int(r.Right - r.Left), int(r.Bottom - r.Top)
}

func (s *ScrollPanel) updateScroll() {
	if s.handle == 0 {
		return
	}
	if s.autoContent {
		w, h := 0, 0
		for _, c := range s.children {
			if !c.Visible() {
				continue
			}
			x, y, cw, ch := c.Bounds()
			if r := x + cw + s.posX; r > w {
				w = r
			}
			if b := y + ch + s.posY; b > h {
				h = b
			}
		}
		s.contentW, s.contentH = w, h
	}
	vw, vh := s.viewSize()
	setScrollInfo(s.handle, sbHorz, 0, maxInt(s.contentW-1, 0), vw, s.posX)
	setScrollInfo(s.handle, sbVert, 0, maxInt(s.contentH-1, 0), vh, s.posY)
	// The bars may have changed the client size, and a smaller content may
	// need a smaller scroll offset.
	s.ScrollTo(s.posX, s.posY)
}

// ScrollTo scrolls so that the point (x, y) of the virtual area is at the top
// left corner of the panel.
func (s *ScrollPanel) ScrollTo(x, y int) {
	if s.handle == 0 {
		return
	}
	vw, vh := s.viewSize()
	x = clampInt(x, 0, maxInt(s.contentW-vw, 0))
	y = clampInt(y, 0, maxInt(s.contentH-vh, 0))
	dx, dy := x-s.posX, y-s.posY
	if dx != 0 || dy != 0 {
		s.posX, s.posY = x, y
		for _, c := range s.children {
			cx, cy, cw, ch := c.Bounds()
			c.SetBounds(cx-dx, cy-dy, cw, ch)
		}
		w32.InvalidateRect(s.handle, nil, true)
	}
	setScrollInfo(s.handle, sbHorz, 0, maxInt(s.contentW-1, 0), vw, s.posX)
	setScrollInfo(s.handle, sbVert, 0, maxInt(s.contentH-1, 0), vh, s.posY)
}

// ScrollIntoView scrolls the minimum needed to make the child fully visible.
func (s *ScrollPanel) ScrollIntoView(c Control) {
	x, y, w, h := c.Bounds()
	vw, vh := s.viewSize()
	nx, ny := s.posX, s.posY
	if x < 0 {
		nx += x
	} else if x+w > vw {
		nx += x + w - vw
	}
	if y < 0 {
		ny += y
	} else if y+h > vh {
		ny += y + h - vh
	}
	s.ScrollTo(nx, ny)
}

func clampInt(v, lo, hi int) int {
	if v > hi {
		v = hi
	}
	if v < lo {
		v = lo
	}
	return v
}

func (s *ScrollPanel) rawProc(msg uint32, wParam, lParam uintptr) (uintptr, bool) {
	switch msg {
	case w32.WM_HSCROLL:
		if lParam == 0 {
			si := getScrollInfo(s.handle, sbHorz)
			s.ScrollTo(scrollRequest(loword(wParam), si, 16), s.posY)
			return 0, true
		}
	case w32.WM_VSCROLL:
		if lParam == 0 {
			si := getScrollInfo(s.handle, sbVert)
			s.ScrollTo(s.posX, scrollRequest(loword(wParam), si, 16))
			return 0, true
		}
	case w32.WM_MOUSEWHEEL:
		_, vh := s.viewSize()
		if s.contentH > vh {
			delta := int(int16(wParam >> 16))
			s.ScrollTo(s.posX, s.posY-delta*48/120)
			return 0, true
		}
	}
	return 0, false
}

// SetVertical makes the scroll bar vertical instead of horizontal. Call it
// before the window is shown.
func (s *ScrollBar) SetVertical(vertical bool) { s.vertical = vertical }
