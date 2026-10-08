package wui

import (
	"errors"
	"syscall"
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

const (
	wmDPIChanged = 0x02E0
)

// HotKeyMod are the modifier keys of a global hot key.
type HotKeyMod uint

const (
	ModAlt      HotKeyMod = 0x0001
	ModControl  HotKeyMod = 0x0002
	ModShift    HotKeyMod = 0x0004
	ModWin      HotKeyMod = 0x0008
	ModNoRepeat HotKeyMod = 0x4000
)

type globalHotKey struct {
	id         int
	mods       HotKeyMod
	key        int
	f          func()
	registered bool
}

// windowFeatures holds the state of the window features added on top of the
// original library: extra events, hot keys, full screen and layouts.
type windowFeatures struct {
	onMove        func(x, y int)
	onActivate    func(active bool)
	onStateChange func(WindowState)
	onDPIChanged  func(dpi int)
	lastState     WindowState
	hotkeys       []*globalHotKey
	nextHotkeyID  int
	fullscreen    bool
	savedStyle    int32
	savedRect     w32.RECT
	taskbar       *taskbarList
	layout        Layout
	tabOrder      []Control
	defaultButton *Button
	cancelButton  *Button
}

// SetOnMove sets the function called when the window is moved. x and y are
// the screen position of the window's top left corner.
func (w *Window) SetOnMove(f func(x, y int)) { w.ext.feat.onMove = f }

// SetOnActivate sets the function called when the window becomes the active
// window (true) or stops being it (false).
func (w *Window) SetOnActivate(f func(active bool)) { w.ext.feat.onActivate = f }

// SetOnStateChange sets the function called when the window is maximized,
// minimized or restored.
func (w *Window) SetOnStateChange(f func(WindowState)) { w.ext.feat.onStateChange = f }

// SetOnDPIChanged sets the function called when the window is moved to a
// monitor with a different DPI (or the user changes the scaling). The window is
// already resized to the size Windows suggests. Needs EnableDPIAwareness.
func (w *Window) SetOnDPIChanged(f func(dpi int)) { w.ext.feat.onDPIChanged = f }

// SetLayout makes the window arrange its direct children with the layout
// every time it is resized. Anchors of the children are overridden by it.
func (w *Window) SetLayout(l Layout) {
	w.ext.feat.layout = l
	w.applyLayout()
}

// Layout returns the window's layout or nil.
func (w *Window) Layout() Layout { return w.ext.feat.layout }

// Relayout arranges the children again, call it after adding or removing
// controls or changing a layout's settings while the window is open.
func (w *Window) Relayout() { w.applyLayout() }

func (w *Window) applyLayout() {
	l := w.ext.feat.layout
	if l == nil || w.handle == 0 {
		return
	}
	iw, ih := w.InnerSize()
	if iw <= 0 || ih <= 0 {
		return
	}
	l.Arrange(w.children, 0, 0, iw, ih)
}

// SetFullscreen turns full screen mode on or off. In full screen mode the
// window has no caption and border and covers the whole monitor it is on.
func (w *Window) SetFullscreen(on bool) {
	f := &w.ext.feat
	if w.handle == 0 || on == f.fullscreen {
		return
	}
	if on {
		if w.state == WindowMaximized {
			w32.ShowWindow(w.handle, w32.SW_RESTORE)
		}
		f.savedStyle = w32.GetWindowLong(w.handle, w32.GWL_STYLE)
		if r := w32.GetWindowRect(w.handle); r != nil {
			f.savedRect = *r
		}
		var mi w32.MONITORINFO
		mon := w32.MonitorFromWindow(w.handle, w32.MONITOR_DEFAULTTONEAREST)
		if !w32.GetMonitorInfo(mon, &mi) {
			return
		}
		style := uint(f.savedStyle) &^ w32.WS_OVERLAPPEDWINDOW
		w32.SetWindowLong(w.handle, w32.GWL_STYLE, int32(style))
		w32.SetWindowPos(
			w.handle, 0,
			int(mi.RcMonitor.Left), int(mi.RcMonitor.Top),
			int(mi.RcMonitor.Right-mi.RcMonitor.Left),
			int(mi.RcMonitor.Bottom-mi.RcMonitor.Top),
			w32.SWP_FRAMECHANGED|w32.SWP_NOOWNERZORDER,
		)
		f.fullscreen = true
	} else {
		w32.SetWindowLong(w.handle, w32.GWL_STYLE, f.savedStyle)
		r := f.savedRect
		w32.SetWindowPos(
			w.handle, 0,
			int(r.Left), int(r.Top), int(r.Right-r.Left), int(r.Bottom-r.Top),
			w32.SWP_FRAMECHANGED|w32.SWP_NOOWNERZORDER|w32.SWP_NOZORDER,
		)
		f.fullscreen = false
	}
	w.readBounds()
}

// Fullscreen tells whether the window is in full screen mode.
func (w *Window) Fullscreen() bool { return w.ext.feat.fullscreen }

// SetOwner makes parent the owner of the window: it stays above its owner
// and is minimized and closed with it. Call it after the window was shown.
func (w *Window) SetOwner(parent *Window) {
	if w.handle == 0 {
		return
	}
	var h uintptr
	if parent != nil {
		h = uintptr(parent.handle)
	}
	w32.SetWindowLongPtr(w.handle, w32.GWL_HWNDPARENT, h)
}

type flashInfo struct {
	cbSize  uint32
	hwnd    uintptr
	flags   uint32
	count   uint32
	timeout uint32
}

// Flash makes the taskbar button and the caption of the window blink count
// times, to get the user's attention. It stops when the window is activated.
func (w *Window) Flash(count int) {
	if w.handle == 0 {
		return
	}
	fi := flashInfo{hwnd: uintptr(w.handle), flags: 3 | 0xC, count: uint32(count)}
	fi.cbSize = uint32(unsafe.Sizeof(fi))
	nxFlashWindowEx.Call(uintptr(unsafe.Pointer(&fi)))
}

// StopFlashing stops what Flash started.
func (w *Window) StopFlashing() {
	if w.handle == 0 {
		return
	}
	fi := flashInfo{hwnd: uintptr(w.handle)}
	fi.cbSize = uint32(unsafe.Sizeof(fi))
	nxFlashWindowEx.Call(uintptr(unsafe.Pointer(&fi)))
}

// RegisterHotKey registers a system wide hot key: f runs on the GUI thread
// whenever the key combination is pressed, even when the window is not in the
// foreground. The returned id can be given to UnregisterHotKey. When the
// window is not shown yet, the registration happens at show time and a failure
// then is silently ignored.
func (w *Window) RegisterHotKey(mods HotKeyMod, key int, f func()) (int, error) {
	feat := &w.ext.feat
	feat.nextHotkeyID++
	hk := &globalHotKey{id: feat.nextHotkeyID, mods: mods, key: key, f: f}
	if w.handle != 0 {
		if err := hk.register(w.handle); err != nil {
			return 0, err
		}
	}
	feat.hotkeys = append(feat.hotkeys, hk)
	return hk.id, nil
}

// UnregisterHotKey removes a hot key made with RegisterHotKey.
func (w *Window) UnregisterHotKey(id int) {
	feat := &w.ext.feat
	for i, hk := range feat.hotkeys {
		if hk.id == id {
			hk.unregister(w.handle)
			feat.hotkeys = append(feat.hotkeys[:i], feat.hotkeys[i+1:]...)
			return
		}
	}
}

func (hk *globalHotKey) register(h w32.HWND) error {
	r, _, err := nxRegisterHotKey.Call(uintptr(h), uintptr(hk.id), uintptr(hk.mods), uintptr(hk.key))
	if r == 0 {
		if err == nil {
			err = errors.New("RegisterHotKey failed")
		}
		return err
	}
	hk.registered = true
	return nil
}

func (hk *globalHotKey) unregister(h w32.HWND) {
	if hk.registered && h != 0 {
		nxUnregisterHotKey.Call(uintptr(h), uintptr(hk.id))
	}
	hk.registered = false
}

// SingleInstance returns true for the first process that calls it with the
// given name and false for every later one. Use it at the start of main to
// stop a second copy of the program from starting. The name should be unique
// to your program, e.g. "com.example.myapp".
func SingleInstance(name string) bool {
	r, _, err := nxCreateMutex.Call(0, 0, uintptr(unsafe.Pointer(utf16Ptr("Local\\"+name))))
	if r == 0 {
		return true // could not tell, do not block the program
	}
	if errno, ok := err.(syscall.Errno); ok && errno == 183 { // ERROR_ALREADY_EXISTS
		return false
	}
	return true
}

// TaskbarProgressState is the look of the progress bar on the taskbar button.
type TaskbarProgressState int

const (
	TaskbarNoProgress    TaskbarProgressState = 0
	TaskbarIndeterminate TaskbarProgressState = 1
	TaskbarNormal        TaskbarProgressState = 2
	TaskbarError         TaskbarProgressState = 4
	TaskbarPaused        TaskbarProgressState = 8
)

// SetTaskbarProgress shows progress on the window's taskbar button. fraction
// goes from 0 to 1 and is ignored for TaskbarNoProgress and
// TaskbarIndeterminate.
func (w *Window) SetTaskbarProgress(state TaskbarProgressState, fraction float64) {
	if w.handle == 0 {
		return
	}
	if w.ext.feat.taskbar == nil {
		w.ext.feat.taskbar = newTaskbarList()
	}
	t := w.ext.feat.taskbar
	if t == nil {
		return
	}
	t.setState(w.handle, state)
	if state == TaskbarNormal || state == TaskbarError || state == TaskbarPaused {
		if fraction < 0 {
			fraction = 0
		}
		if fraction > 1 {
			fraction = 1
		}
		t.setValue(w.handle, uint64(fraction*1000+0.5), 1000)
	}
}

// taskbarList wraps the COM interface ITaskbarList3.
type taskbarList struct {
	obj uintptr
}

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	clsidTaskbarList = guid{0x56FDF344, 0xFD6D, 0x11D0, [8]byte{0x95, 0x8A, 0x00, 0x60, 0x97, 0xC9, 0xA0, 0x90}}
	iidTaskbarList3  = guid{0xEA1AFB91, 0x9E28, 0x4B86, [8]byte{0x90, 0xE9, 0x9E, 0x9F, 0x8A, 0x5E, 0xEF, 0xAF}}
)

func newTaskbarList() *taskbarList {
	nxCoInitializeEx.Call(0, 2) // COINIT_APARTMENTTHREADED, S_FALSE is fine
	var obj uintptr
	hr, _, _ := nxCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidTaskbarList)), 0, 1, // CLSCTX_INPROC_SERVER
		uintptr(unsafe.Pointer(&iidTaskbarList3)),
		uintptr(unsafe.Pointer(&obj)),
	)
	if int32(hr) < 0 || obj == 0 {
		return nil
	}
	t := &taskbarList{obj: obj}
	syscall.Syscall(t.method(3), 1, obj, 0, 0) // HrInit
	return t
}

func (t *taskbarList) method(index int) uintptr {
	vtbl := *(*uintptr)(unsafe.Pointer(t.obj))
	return *(*uintptr)(unsafe.Pointer(vtbl + uintptr(index)*unsafe.Sizeof(uintptr(0))))
}

func (t *taskbarList) setState(h w32.HWND, state TaskbarProgressState) {
	syscall.Syscall(t.method(10), 3, t.obj, uintptr(h), uintptr(state))
}

func (t *taskbarList) setValue(h w32.HWND, completed, total uint64) {
	if unsafe.Sizeof(uintptr(0)) == 4 {
		syscall.Syscall6(t.method(9), 6, t.obj, uintptr(h),
			uintptr(completed&0xFFFFFFFF), uintptr(completed>>32),
			uintptr(total&0xFFFFFFFF), uintptr(total>>32))
		return
	}
	syscall.Syscall6(t.method(9), 4, t.obj, uintptr(h), uintptr(completed), uintptr(total), 0, 0)
}

// windowStarted is called once the native window and its controls exist.
func (w *Window) windowStarted() {
	w.invokeStarted()
	w.ext.feat.lastState = w.state
	for _, hk := range w.ext.feat.hotkeys {
		hk.register(w.handle)
	}
	w.applyLayout()
}

// windowStopped is called when the native window is destroyed.
func (w *Window) windowStopped() {
	for _, hk := range w.ext.feat.hotkeys {
		hk.unregister(w.handle)
	}
	w.invokeStopped()
}

// ext2Msg handles the messages of the newer window features. Messages that
// the rest of the window code needs too are reported as not handled.
func (w *Window) ext2Msg(window w32.HWND, msg uint32, wParam, lParam uintptr) (uintptr, bool) {
	feat := &w.ext.feat
	switch msg {
	case wmInvoke:
		w.drainInvoked()
		return 0, true
	case w32.WM_HOTKEY:
		for _, hk := range feat.hotkeys {
			if uintptr(hk.id) == wParam && hk.f != nil {
				hk.f()
			}
		}
		return 0, true
	case w32.WM_MOVE:
		if feat.onMove != nil {
			if r := w32.GetWindowRect(window); r != nil {
				feat.onMove(int(r.Left), int(r.Top))
			}
		}
	case w32.WM_ACTIVATE:
		if feat.onActivate != nil {
			feat.onActivate(wParam&0xFFFF != 0)
		}
	case w32.WM_SIZE:
		if feat.onStateChange != nil {
			st := feat.lastState
			switch wParam {
			case w32.SIZE_MAXIMIZED:
				st = WindowMaximized
			case w32.SIZE_MINIMIZED:
				st = WindowMinimized
			case w32.SIZE_RESTORED:
				st = WindowNormal
			}
			if st != feat.lastState {
				feat.lastState = st
				feat.onStateChange(st)
			}
		}
	case wmDPIChanged:
		if r := (*w32.RECT)(unsafe.Pointer(lParam)); r != nil {
			w32.SetWindowPos(
				window, 0,
				int(r.Left), int(r.Top), int(r.Right-r.Left), int(r.Bottom-r.Top),
				w32.SWP_NOZORDER|w32.SWP_NOACTIVATE,
			)
		}
		if feat.onDPIChanged != nil {
			feat.onDPIChanged(loword(wParam))
		}
		return 0, true
	case w32.WM_DESTROY:
		w.windowStopped()
	}
	return 0, false
}
