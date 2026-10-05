package wui

import (
	"syscall"
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

// windowExt holds the additional window settings: transparency, headless
// mode, tray and drag and drop.
type windowExt struct {
	transparent      bool
	transparentKey   Color
	topMost          bool
	hideFromTaskbar  bool
	dragByBackground bool
	minW, minH       int
	maxW, maxH       int
	cornerRadius     int
	acceptFiles      bool
	onDropFiles      DropFilesFunc
	tray             *TrayIcon
	minimizeToTray   bool
	closeToTray      bool
	programmaticClose bool
	taskbarCreated   uint32
}

// DefaultTransparentColor is the key color used by SetTransparent when no
// color was chosen: magenta.
var DefaultTransparentColor = RGB(255, 0, 255)

// SetTransparent turns on (or off) color key transparency: every pixel of the
// window that has exactly the TransparentColor is invisible and lets mouse
// clicks pass through. Set the window background and the backgrounds of
// panels to that color to get a window with a free-form shape. Combine it with
// SetHeadless(true) for a window without caption and border. See also
// SetAlpha for a semi-transparent window.
func (w *Window) SetTransparent(on bool) {
	if on && w.ext.transparentKey == 0 {
		w.ext.transparentKey = DefaultTransparentColor
	}
	w.ext.transparent = on
	w.applyLayered()
}

// Transparent returns true if color key transparency is on.
func (w *Window) Transparent() bool {
	return w.ext.transparent
}

// SetTransparentColor chooses the key color and turns color key transparency
// on, see SetTransparent.
func (w *Window) SetTransparentColor(c Color) {
	w.ext.transparentKey = c
	w.ext.transparent = true
	w.applyLayered()
}

// TransparentColor returns the key color, magenta by default.
func (w *Window) TransparentColor() Color {
	if w.ext.transparentKey == 0 {
		return DefaultTransparentColor
	}
	return w.ext.transparentKey
}

func (w *Window) applyLayered() {
	if w.handle == 0 {
		return
	}
	ex := w32.GetWindowLong(w.handle, w32.GWL_EXSTYLE)
	if w.alpha == 255 && !w.ext.transparent {
		if ex&w32.WS_EX_LAYERED != 0 {
			w32.SetWindowLong(w.handle, w32.GWL_EXSTYLE, ex & ^w32.WS_EX_LAYERED)
			w32.RedrawWindow(
				w.handle, nil, 0,
				w32.RDW_ERASE|w32.RDW_INVALIDATE|w32.RDW_FRAME|w32.RDW_ALLCHILDREN,
			)
		}
		return
	}
	if ex&w32.WS_EX_LAYERED == 0 {
		w32.SetWindowLong(w.handle, w32.GWL_EXSTYLE, ex|w32.WS_EX_LAYERED)
	}
	var flags uint32
	if w.alpha != 255 {
		flags |= w32.LWA_ALPHA
	}
	if w.ext.transparent {
		flags |= w32.LWA_COLORKEY
	}
	w32.SetLayeredWindowAttributes(
		w.handle, w32.COLORREF(w.TransparentColor()), w.alpha, flags,
	)
}

// SetHeadless removes the title bar and the border of the window (it becomes a
// borderless popup window). Use SetDragByBackground to still be able to move
// it and SetResizable to keep or remove the resize edges.
func (w *Window) SetHeadless(headless bool) {
	w.SetHasBorder(!headless)
}

// Headless returns true if the window has no title bar and border.
func (w *Window) Headless() bool {
	return !w.HasBorder()
}

// SetDragByBackground lets the user move the window by dragging any free
// area of its background. This is useful for headless windows.
func (w *Window) SetDragByBackground(on bool) {
	w.ext.dragByBackground = on
}

// DragByBackground returns true if the window can be moved by its background.
func (w *Window) DragByBackground() bool {
	return w.ext.dragByBackground
}

// SetTopMost keeps the window above all non-topmost windows.
func (w *Window) SetTopMost(on bool) {
	w.ext.topMost = on
	if w.handle != 0 {
		after := w32.HWND_NOTOPMOST
		if on {
			after = w32.HWND_TOPMOST
		}
		w32.SetWindowPos(
			w.handle, after, 0, 0, 0, 0,
			w32.SWP_NOMOVE|w32.SWP_NOSIZE|w32.SWP_NOACTIVATE,
		)
	}
}

// TopMost returns true if the window stays on top of other windows.
func (w *Window) TopMost() bool {
	return w.ext.topMost
}

// SetShowInTaskbar chooses whether the window has a button in the taskbar. A
// window without taskbar button is a tool window. The default is true.
func (w *Window) SetShowInTaskbar(show bool) {
	if w.ext.hideFromTaskbar == !show {
		return
	}
	w.ext.hideFromTaskbar = !show
	if w.handle != 0 {
		visible := w32.IsWindowVisible(w.handle)
		if visible {
			w32.ShowWindow(w.handle, w32.SW_HIDE)
		}
		ex := w32.GetWindowLong(w.handle, w32.GWL_EXSTYLE)
		if show {
			ex &^= w32.WS_EX_TOOLWINDOW
		} else {
			ex |= w32.WS_EX_TOOLWINDOW
		}
		w32.SetWindowLong(w.handle, w32.GWL_EXSTYLE, ex)
		if visible {
			w32.ShowWindow(w.handle, w32.SW_SHOW)
		}
	}
}

// ShowInTaskbar returns true if the window has a taskbar button.
func (w *Window) ShowInTaskbar() bool {
	return !w.ext.hideFromTaskbar
}

// SetMinSize sets the smallest outer size the user can resize the window to.
// 0 means no limit.
func (w *Window) SetMinSize(width, height int) {
	w.ext.minW, w.ext.minH = width, height
}

// MinSize returns the size set with SetMinSize.
func (w *Window) MinSize() (width, height int) {
	return w.ext.minW, w.ext.minH
}

// SetMaxSize sets the biggest outer size the user can resize the window to.
// 0 means no limit.
func (w *Window) SetMaxSize(width, height int) {
	w.ext.maxW, w.ext.maxH = width, height
}

// MaxSize returns the size set with SetMaxSize.
func (w *Window) MaxSize() (width, height int) {
	return w.ext.maxW, w.ext.maxH
}

// SetCornerRadius gives the window rounded corners with the given radius in
// pixels. 0 means square corners. It is most useful for headless windows.
func (w *Window) SetCornerRadius(radius int) {
	if radius < 0 {
		radius = 0
	}
	w.ext.cornerRadius = radius
	w.applyRegion()
}

// CornerRadius returns the radius set with SetCornerRadius.
func (w *Window) CornerRadius() int {
	return w.ext.cornerRadius
}

func (w *Window) applyRegion() {
	if w.handle == 0 {
		return
	}
	if w.ext.cornerRadius <= 0 || w.state == WindowMaximized {
		procSetWindowRgn.Call(uintptr(w.handle), 0, 1)
		return
	}
	r := w32.GetWindowRect(w.handle)
	width := int(r.Right - r.Left)
	height := int(r.Bottom - r.Top)
	d := uintptr(w.ext.cornerRadius * 2)
	rgn, _, _ := procCreateRoundRgn.Call(
		0, 0, uintptr(width+1), uintptr(height+1), d, d,
	)
	if rgn != 0 {
		// After this call the system owns the region.
		procSetWindowRgn.Call(uintptr(w.handle), rgn, 1)
	}
}

// BackgroundColor returns the window background color.
func (w *Window) BackgroundColor() Color {
	return w.background
}

// SetBackgroundColor changes the window background color, it is the same as
// SetBackground.
func (w *Window) SetBackgroundColor(c Color) {
	w.SetBackground(c)
}

// extAfterCreate is called once the window and all its controls exist.
func (w *Window) extAfterCreate() {
	w.ext.taskbarCreated = registerWindowMessage("TaskbarCreated")
	w.applyDropTarget()
	w.applyRegion()
	if w.ext.tray != nil {
		w.ext.tray.window = w
		w.ext.tray.add()
	}
}

func registerWindowMessage(name string) uint32 {
	p := syscall.StringToUTF16Ptr(name)
	r, _, _ := procRegisterWindowMs.Call(uintptr(unsafe.Pointer(p)))
	return uint32(r)
}

// extMsg handles the messages of the extended window features. It returns true
// if the message was handled and the result must be returned to Windows.
func (w *Window) extMsg(window w32.HWND, msg uint32, wParam, lParam uintptr) (uintptr, bool) {
	if w.ext.taskbarCreated != 0 && msg == w.ext.taskbarCreated {
		// Explorer was restarted, bring the tray icon back.
		if w.ext.tray != nil {
			w.ext.tray.added = false
			w.ext.tray.add()
		}
		return 0, true
	}
	switch msg {
	case wmTrayCallback:
		if w.ext.tray != nil {
			w.ext.tray.onMessage(lParam)
		}
		return 0, true
	case w32.WM_DROPFILES:
		w.onWM_DROPFILES(wParam)
		return 0, true
	case w32.WM_NCHITTEST:
		if w.ext.dragByBackground {
			r := w32.DefWindowProc(window, msg, wParam, lParam)
			if r == w32.HTCLIENT {
				return uintptr(w32.HTCAPTION), true
			}
			return r, true
		}
	case w32.WM_GETMINMAXINFO:
		if w.ext.minW > 0 || w.ext.minH > 0 || w.ext.maxW > 0 || w.ext.maxH > 0 {
			mmi := (*w32.MINMAXINFO)(unsafe.Pointer(lParam))
			if w.ext.minW > 0 {
				mmi.PtMinTrackSize.X = int32(w.ext.minW)
			}
			if w.ext.minH > 0 {
				mmi.PtMinTrackSize.Y = int32(w.ext.minH)
			}
			if w.ext.maxW > 0 {
				mmi.PtMaxTrackSize.X = int32(w.ext.maxW)
			}
			if w.ext.maxH > 0 {
				mmi.PtMaxTrackSize.Y = int32(w.ext.maxH)
			}
			return 0, true
		}
	case w32.WM_SIZE:
		if wParam == w32.SIZE_MINIMIZED {
			if w.ext.minimizeToTray && w.ext.tray != nil {
				w32.ShowWindow(w.handle, w32.SW_HIDE)
			}
		} else if w.ext.cornerRadius > 0 {
			if wParam == w32.SIZE_MAXIMIZED {
				procSetWindowRgn.Call(uintptr(w.handle), 0, 1)
			} else {
				w.applyRegion()
			}
		}
	case w32.WM_CLOSE:
		if w.ext.closeToTray && w.ext.tray != nil && !w.ext.programmaticClose {
			w.HideToTray()
			return 0, true
		}
	case w32.WM_DESTROY:
		if w.ext.tray != nil {
			w.ext.tray.remove()
		}
	}
	return 0, false
}
