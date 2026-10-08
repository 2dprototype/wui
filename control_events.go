package wui

import (
	"syscall"
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

// controlEvents holds the event handlers every control has. They are set with
// the SetOn... methods below. Controls that had their own handler of the same
// name before (PaintBox) keep it, theirs wins.
type controlEvents struct {
	onMouseDown   func(x, y int, button MouseButton)
	onMouseUp     func(x, y int, button MouseButton)
	onDoubleClick func(x, y int, button MouseButton)
	onMouseMove   func(x, y int)
	onMouseWheel  func(x, y int, delta float64)
	onMouseEnter  func()
	onMouseLeave  func()
	onKeyDown     func(key int)
	onKeyUp       func(key int)
	onChar        func(r rune)
	onFocus       func()
	onBlur        func()
	tracking      bool
	tabStop       int // 0 keep the control's default, 1 force on, 2 force off
	cursor        *Cursor
	// raw sees every message before the handlers; controls use it for
	// behavior that needs the window procedure (scroll panels).
	raw func(msg uint32, wParam, lParam uintptr) (uintptr, bool)
}

func (ev *controlEvents) wantsMouse() bool {
	return ev.onMouseDown != nil || ev.onMouseUp != nil || ev.onDoubleClick != nil ||
		ev.onMouseMove != nil || ev.onMouseWheel != nil || ev.onMouseEnter != nil ||
		ev.onMouseLeave != nil || ev.cursor != nil
}

// SetCursor sets the mouse cursor shown over the control, nil goes back to
// the default. Labels, panels and paint boxes pass mouse input through to their
// parent, setting a cursor or a mouse handler makes them keep it.
func (c *control) SetCursor(cur *Cursor) { c.ev.cursor = cur }

// SetOnMouseDown sets the function called when a mouse button goes down over
// the control. x and y are relative to the control.
func (c *control) SetOnMouseDown(f func(x, y int, button MouseButton)) { c.ev.onMouseDown = f }

// SetOnMouseUp sets the function called when a mouse button is released.
func (c *control) SetOnMouseUp(f func(x, y int, button MouseButton)) { c.ev.onMouseUp = f }

// SetOnDoubleClick sets the function called on a mouse double click.
func (c *control) SetOnDoubleClick(f func(x, y int, button MouseButton)) { c.ev.onDoubleClick = f }

// SetOnMouseMove sets the function called when the mouse moves over the
// control.
func (c *control) SetOnMouseMove(f func(x, y int)) { c.ev.onMouseMove = f }

// SetOnMouseWheel sets the function called when the wheel turns while the
// control has the focus. A delta of 1 is one notch away from the user.
func (c *control) SetOnMouseWheel(f func(x, y int, delta float64)) { c.ev.onMouseWheel = f }

// SetOnMouseEnter sets the function called when the mouse enters the control.
func (c *control) SetOnMouseEnter(f func()) { c.ev.onMouseEnter = f }

// SetOnMouseLeave sets the function called when the mouse leaves the control.
func (c *control) SetOnMouseLeave(f func()) { c.ev.onMouseLeave = f }

// SetOnKeyDown sets the function called when a key goes down while the
// control has the focus. The key is a virtual key code, see the Key... values.
func (c *control) SetOnKeyDown(f func(key int)) { c.ev.onKeyDown = f }

// SetOnKeyUp sets the function called when a key is released.
func (c *control) SetOnKeyUp(f func(key int)) { c.ev.onKeyUp = f }

// SetOnChar sets the function called for typed characters.
func (c *control) SetOnChar(f func(r rune)) { c.ev.onChar = f }

// SetOnFocus sets the function called when the control gets the keyboard
// focus.
func (c *control) SetOnFocus(f func()) { c.ev.onFocus = f }

// SetOnBlur sets the function called when the control loses the keyboard
// focus.
func (c *control) SetOnBlur(f func()) { c.ev.onBlur = f }

// Focus gives the control the keyboard focus. It does nothing before the
// window is shown.
func (c *control) Focus() {
	if c.handle != 0 {
		w32.SetFocus(c.handle)
	}
}

// HasFocus tells whether the control has the keyboard focus.
func (c *control) HasFocus() bool {
	return c.handle != 0 && w32.GetFocus() == c.handle
}

// SetTabStop chooses whether the Tab key stops at this control. The default
// depends on the control type.
func (c *control) SetTabStop(on bool) {
	if on {
		c.ev.tabStop = 1
	} else {
		c.ev.tabStop = 2
	}
	c.applyTabStop()
}

// TabStop tells whether the Tab key stops at this control.
func (c *control) TabStop() bool {
	if c.handle == 0 {
		return c.ev.tabStop == 1
	}
	return uint(w32.GetWindowLong(c.handle, w32.GWL_STYLE))&w32.WS_TABSTOP != 0
}

func (c *control) applyTabStop() {
	if c.handle == 0 || c.ev.tabStop == 0 {
		return
	}
	style := uint(w32.GetWindowLong(c.handle, w32.GWL_STYLE))
	if c.ev.tabStop == 1 {
		style |= w32.WS_TABSTOP
	} else {
		style &^= w32.WS_TABSTOP
	}
	w32.SetWindowLong(c.handle, w32.GWL_STYLE, int32(style))
}

const controlEventsSubclassID = 0x57A1

func (c *control) installEvents() {
	if c.handle == 0 {
		return
	}
	c.ev.tracking = false
	c.applyTabStop()
	w32.SetWindowSubclass(c.handle, controlEventsProc, controlEventsSubclassID, uintptr(unsafe.Pointer(c)))
}

var controlEventsProc = syscall.NewCallback(func(
	window w32.HWND,
	msg uint32,
	wParam, lParam uintptr,
	subclassID uintptr,
	refData uintptr,
) uintptr {
	c := (*control)(unsafe.Pointer(refData))
	ev := &c.ev
	if ev.raw != nil {
		if r, ok := ev.raw(msg, wParam, lParam); ok {
			return r
		}
	}
	switch msg {
	case w32.WM_NCHITTEST:
		if ev.wantsMouse() {
			return w32.HTCLIENT
		}
	case w32.WM_SETCURSOR:
		if ev.cursor != nil && loword(lParam) == w32.HTCLIENT {
			w32.SetCursor(ev.cursor.handle)
			return 1
		}
	case w32.WM_LBUTTONDOWN:
		if ev.onMouseDown != nil {
			ev.onMouseDown(loInt(lParam), hiInt(lParam), MouseButtonLeft)
		}
	case w32.WM_MBUTTONDOWN:
		if ev.onMouseDown != nil {
			ev.onMouseDown(loInt(lParam), hiInt(lParam), MouseButtonMiddle)
		}
	case w32.WM_RBUTTONDOWN:
		if ev.onMouseDown != nil {
			ev.onMouseDown(loInt(lParam), hiInt(lParam), MouseButtonRight)
		}
	case w32.WM_LBUTTONUP:
		if ev.onMouseUp != nil {
			ev.onMouseUp(loInt(lParam), hiInt(lParam), MouseButtonLeft)
		}
	case w32.WM_MBUTTONUP:
		if ev.onMouseUp != nil {
			ev.onMouseUp(loInt(lParam), hiInt(lParam), MouseButtonMiddle)
		}
	case w32.WM_RBUTTONUP:
		if ev.onMouseUp != nil {
			ev.onMouseUp(loInt(lParam), hiInt(lParam), MouseButtonRight)
		}
	case w32.WM_LBUTTONDBLCLK:
		if ev.onDoubleClick != nil {
			ev.onDoubleClick(loInt(lParam), hiInt(lParam), MouseButtonLeft)
		}
	case w32.WM_MBUTTONDBLCLK:
		if ev.onDoubleClick != nil {
			ev.onDoubleClick(loInt(lParam), hiInt(lParam), MouseButtonMiddle)
		}
	case w32.WM_RBUTTONDBLCLK:
		if ev.onDoubleClick != nil {
			ev.onDoubleClick(loInt(lParam), hiInt(lParam), MouseButtonRight)
		}
	case w32.WM_MOUSEMOVE:
		if ev.onMouseMove != nil {
			ev.onMouseMove(loInt(lParam), hiInt(lParam))
		}
		if !ev.tracking && (ev.onMouseEnter != nil || ev.onMouseLeave != nil) {
			ev.tracking = true
			w32.TrackMouseEvent(&w32.TRACKMOUSEEVENT{
				CbSize:    uint32(unsafe.Sizeof(w32.TRACKMOUSEEVENT{})),
				DwFlags:   w32.TME_LEAVE,
				HwndTrack: window,
			})
			if ev.onMouseEnter != nil {
				ev.onMouseEnter()
			}
		}
	case w32.WM_MOUSELEAVE:
		ev.tracking = false
		if ev.onMouseLeave != nil {
			ev.onMouseLeave()
		}
	case w32.WM_MOUSEWHEEL:
		if ev.onMouseWheel != nil {
			x, y, _ := w32.ScreenToClient(window, loInt(lParam), hiInt(lParam))
			delta := float64(int16((wParam&0xFFFF0000)>>16)) / 120
			ev.onMouseWheel(x, y, delta)
		}
	case w32.WM_KEYDOWN, w32.WM_SYSKEYDOWN:
		if ev.onKeyDown != nil {
			ev.onKeyDown(int(wParam))
		}
	case w32.WM_KEYUP, w32.WM_SYSKEYUP:
		if ev.onKeyUp != nil {
			ev.onKeyUp(int(wParam))
		}
	case w32.WM_CHAR:
		if ev.onChar != nil {
			ev.onChar(rune(wParam))
		}
	case w32.WM_SETFOCUS:
		if ev.onFocus != nil {
			ev.onFocus()
		}
	case w32.WM_KILLFOCUS:
		if ev.onBlur != nil {
			ev.onBlur()
		}
	}
	return w32.DefSubclassProc(window, msg, wParam, lParam)
})
