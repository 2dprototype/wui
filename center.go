package wui

import "github.com/2dprototype/wui/w32"

// SetCenterOnShow makes the window center itself when it is shown. A window
// shown with ShowModal is centered over its parent window, any other window is
// centered on the screen (work area of its monitor).
func (w *Window) SetCenterOnShow(center bool) {
	w.centerOnShow = center
}

// CenterOnShow returns whether the window centers itself when shown.
func (w *Window) CenterOnShow() bool {
	return w.centerOnShow
}

// Center moves the window to the center of the screen or, for a modal window,
// to the center of its parent. It can be called before the window is shown (the
// window size must be set already) and while it is shown.
func (w *Window) Center() {
	w.centerOn(w.parent)
}

// CenterOnScreen moves the window to the center of the monitor it is on.
func (w *Window) CenterOnScreen() {
	w.centerOn(nil)
}

// CenterOnParent moves a modal window to the center of its parent. If there is
// no parent, the window is centered on the screen.
func (w *Window) CenterOnParent() {
	w.centerOn(w.parent)
}

func (w *Window) centerOn(parent *Window) {
	width, height := w.width, w.height
	if w.handle != 0 {
		r := w32.GetWindowRect(w.handle)
		width, height = int(r.Right-r.Left), int(r.Bottom-r.Top)
	}

	var left, top, areaW, areaH int
	if parent != nil && parent.handle != 0 {
		r := w32.GetWindowRect(parent.handle)
		left, top = int(r.Left), int(r.Top)
		areaW, areaH = int(r.Right-r.Left), int(r.Bottom-r.Top)
	} else {
		left, top = 0, 0
		areaW, areaH = w32.GetSystemMetrics(w32.SM_CXSCREEN), w32.GetSystemMetrics(w32.SM_CYSCREEN)
		var h w32.HWND = w.handle
		if parent == nil && h == 0 {
			h = w32.GetDesktopWindow()
		}
		if mon := w32.MonitorFromWindow(h, w32.MONITOR_DEFAULTTONEAREST); mon != 0 {
			var mi w32.MONITORINFO
			mi.CbSize = uint32(unsafeSizeofMonitorInfo)
			if w32.GetMonitorInfo(mon, &mi) {
				left, top = int(mi.RcWork.Left), int(mi.RcWork.Top)
				areaW = int(mi.RcWork.Right - mi.RcWork.Left)
				areaH = int(mi.RcWork.Bottom - mi.RcWork.Top)
			}
		}
	}

	x := left + (areaW-width)/2
	y := top + (areaH-height)/2
	if w.handle != 0 {
		w32.SetWindowPos(
			w.handle, 0, x, y, width, height,
			w32.SWP_NOOWNERZORDER|w32.SWP_NOZORDER|w32.SWP_NOSIZE,
		)
		w.readBounds()
	} else {
		w.x, w.y = x, y
	}
}
