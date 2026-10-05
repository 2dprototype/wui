package wui

import (
	"syscall"
	"unicode/utf16"
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

const (
	wmTrayCallback = w32.WM_APP + 1

	nimAdd    = 0
	nimModify = 1
	nimDelete = 2

	nifMessage = 0x01
	nifIcon    = 0x02
	nifTip     = 0x04
	nifInfo    = 0x10

	wmLButtonUp     = 0x0202
	wmLButtonDblClk = 0x0203
	wmRButtonUp     = 0x0205
	wmMButtonUp     = 0x0208
	ninBalloonClick = 0x0405 // WM_USER + 5
)

var (
	extShell32           = syscall.NewLazyDLL("shell32.dll")
	procShellNotifyIcon  = extShell32.NewProc("Shell_NotifyIconW")
	extUser32            = syscall.NewLazyDLL("user32.dll")
	procRegisterWindowMs = extUser32.NewProc("RegisterWindowMessageW")
	procSetWindowRgn     = extUser32.NewProc("SetWindowRgn")
	extGdi32             = syscall.NewLazyDLL("gdi32.dll")
	procCreateRoundRgn   = extGdi32.NewProc("CreateRoundRectRgn")
)

// notifyIconData is the NOTIFYICONDATAW structure.
type notifyIconData struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UTimeout         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         [16]byte
	HBalloonIcon     uintptr
}

func copyUTF16(dst []uint16, s string) {
	u := utf16.Encode([]rune(s))
	if len(u) > len(dst)-1 {
		u = u[:len(dst)-1]
	}
	n := copy(dst, u)
	dst[n] = 0
}

// BalloonKind selects the icon shown in a tray balloon notification.
type BalloonKind int

const (
	BalloonNone BalloonKind = iota
	BalloonInfo
	BalloonWarning
	BalloonError
)

// TrayIcon is an icon in the system tray (notification area). Create it with
// NewTrayIcon and attach it to a window with Window.SetTrayIcon. It is shown
// as soon as the window is shown.
//
//	tray := wui.NewTrayIcon()
//	tray.SetToolTip("My App")
//	menu := wui.NewPopupMenu()
//	menu.Add(quit)
//	tray.SetMenu(menu)
//	window.SetTrayIcon(tray)
type TrayIcon struct {
	window         *Window
	icon           *Icon
	tip            string
	menu           *PopupMenu
	hidden         bool
	added          bool
	restoreOnClick bool
	onClick        func()
	onDoubleClick  func()
	onRightClick   func()
	onBalloonClick func()
}

// NewTrayIcon creates a tray icon that uses the window's icon. By default a
// left click restores the window if it was hidden in the tray.
func NewTrayIcon() *TrayIcon {
	return &TrayIcon{restoreOnClick: true}
}

// SetIcon sets the tray icon image. By default the icon of the window is used.
func (t *TrayIcon) SetIcon(icon *Icon) {
	t.icon = icon
	t.update()
}

// Icon returns the icon set with SetIcon.
func (t *TrayIcon) Icon() *Icon { return t.icon }

// SetToolTip sets the text that appears when hovering over the tray icon (at
// most 127 characters).
func (t *TrayIcon) SetToolTip(text string) {
	t.tip = text
	t.update()
}

// ToolTip returns the hover text.
func (t *TrayIcon) ToolTip() string { return t.tip }

// SetMenu sets the menu that opens on right click.
func (t *TrayIcon) SetMenu(m *PopupMenu) { t.menu = m }

// Menu returns the right click menu.
func (t *TrayIcon) Menu() *PopupMenu { return t.menu }

// SetVisible shows or hides the icon in the tray.
func (t *TrayIcon) SetVisible(v bool) {
	t.hidden = !v
	if v {
		t.update()
	} else {
		t.remove()
	}
}

// Visible returns true if the icon is meant to be shown.
func (t *TrayIcon) Visible() bool { return !t.hidden }

// SetRestoreOnClick controls whether a left click on the icon brings back a
// window that was hidden with HideToTray, MinimizeToTray or CloseToTray. The
// default is true.
func (t *TrayIcon) SetRestoreOnClick(restore bool) { t.restoreOnClick = restore }

// SetOnClick sets the function that is called on a left click.
func (t *TrayIcon) SetOnClick(f func()) { t.onClick = f }

// SetOnDoubleClick sets the function that is called on a double click.
func (t *TrayIcon) SetOnDoubleClick(f func()) { t.onDoubleClick = f }

// SetOnRightClick sets the function that is called on a right click, before
// the menu opens.
func (t *TrayIcon) SetOnRightClick(f func()) { t.onRightClick = f }

// SetOnBalloonClick sets the function that is called when the user clicks a
// balloon notification.
func (t *TrayIcon) SetOnBalloonClick(f func()) { t.onBalloonClick = f }

func (t *TrayIcon) data(flags uint32) notifyIconData {
	var d notifyIconData
	d.CbSize = uint32(unsafe.Sizeof(d))
	d.HWnd = uintptr(t.window.handle)
	d.UID = 1
	d.UFlags = flags
	d.UCallbackMessage = wmTrayCallback
	icon := t.icon
	if icon == nil {
		icon = t.window.icon
	}
	if icon == nil {
		icon = IconApplication
	}
	d.HIcon = uintptr(icon.handle)
	copyUTF16(d.SzTip[:], t.tip)
	return d
}

func shellNotify(msg uint32, d *notifyIconData) bool {
	r, _, _ := procShellNotifyIcon.Call(uintptr(msg), uintptr(unsafe.Pointer(d)))
	return r != 0
}

func (t *TrayIcon) ready() bool {
	return t.window != nil && t.window.handle != 0
}

func (t *TrayIcon) add() {
	if !t.ready() || t.hidden {
		return
	}
	d := t.data(nifMessage | nifIcon | nifTip)
	if shellNotify(nimAdd, &d) {
		t.added = true
	}
}

func (t *TrayIcon) update() {
	if !t.ready() || t.hidden {
		return
	}
	if !t.added {
		t.add()
		return
	}
	d := t.data(nifMessage | nifIcon | nifTip)
	shellNotify(nimModify, &d)
}

func (t *TrayIcon) remove() {
	if !t.ready() || !t.added {
		return
	}
	d := t.data(0)
	shellNotify(nimDelete, &d)
	t.added = false
}

// ShowBalloon shows a balloon notification at the tray icon.
func (t *TrayIcon) ShowBalloon(title, text string, kind BalloonKind) {
	if !t.ready() {
		return
	}
	if !t.added {
		t.add()
	}
	d := t.data(nifInfo)
	copyUTF16(d.SzInfoTitle[:], title)
	copyUTF16(d.SzInfo[:], text)
	d.DwInfoFlags = uint32(kind)
	shellNotify(nimModify, &d)
}

func (t *TrayIcon) onMessage(lParam uintptr) {
	switch uint32(lParam) {
	case wmLButtonUp:
		if t.onClick != nil {
			t.onClick()
		}
		if t.restoreOnClick && t.window != nil {
			t.window.RestoreFromTray()
		}
	case wmLButtonDblClk:
		if t.onDoubleClick != nil {
			t.onDoubleClick()
		}
	case wmRButtonUp:
		if t.onRightClick != nil {
			t.onRightClick()
		}
		if t.menu != nil && t.window != nil {
			t.menu.ShowAtCursor(t.window)
			// Makes the menu close when the user clicks somewhere else.
			w32.PostMessage(t.window.handle, w32.WM_NULL, 0, 0)
		}
	case ninBalloonClick:
		if t.onBalloonClick != nil {
			t.onBalloonClick()
		}
	}
}

// SetTrayIcon puts an icon for the window into the system tray. Pass nil to
// remove it.
func (w *Window) SetTrayIcon(t *TrayIcon) {
	if w.ext.tray != nil && w.ext.tray != t {
		w.ext.tray.remove()
		w.ext.tray.window = nil
	}
	w.ext.tray = t
	if t != nil {
		t.window = w
		t.add()
	}
}

// TrayIcon returns the tray icon of the window or nil.
func (w *Window) TrayIcon() *TrayIcon {
	return w.ext.tray
}

// SetTrayEnabled adds a default tray icon (using the window icon) or removes
// the tray icon. Use SetTrayIcon for more control.
func (w *Window) SetTrayEnabled(on bool) {
	if on && w.ext.tray == nil {
		w.SetTrayIcon(NewTrayIcon())
	} else if !on && w.ext.tray != nil {
		w.SetTrayIcon(nil)
	}
}

// TrayEnabled returns true if the window has a tray icon.
func (w *Window) TrayEnabled() bool {
	return w.ext.tray != nil
}

// SetTrayToolTip sets the tray icon's hover text and creates the tray icon if
// the window does not have one yet.
func (w *Window) SetTrayToolTip(text string) {
	if w.ext.tray == nil {
		if text == "" {
			return
		}
		w.SetTrayIcon(NewTrayIcon())
	}
	w.ext.tray.SetToolTip(text)
}

// TrayToolTip returns the tray icon's hover text.
func (w *Window) TrayToolTip() string {
	if w.ext.tray == nil {
		return ""
	}
	return w.ext.tray.tip
}

// SetMinimizeToTray makes the window disappear into the tray (instead of the
// taskbar) when it is minimized. It needs a tray icon.
func (w *Window) SetMinimizeToTray(on bool) { w.ext.minimizeToTray = on }

// MinimizeToTray returns true if minimizing hides the window in the tray.
func (w *Window) MinimizeToTray() bool { return w.ext.minimizeToTray }

// SetCloseToTray makes the close button hide the window in the tray instead of
// closing it. Window.Close (and Quit) still close it for real. It needs a tray
// icon.
func (w *Window) SetCloseToTray(on bool) { w.ext.closeToTray = on }

// CloseToTray returns true if the close button hides the window in the tray.
func (w *Window) CloseToTray() bool { return w.ext.closeToTray }

// HideToTray hides the window completely, including its taskbar button. Bring
// it back with RestoreFromTray, a left click on the tray icon does that by
// default.
func (w *Window) HideToTray() {
	if w.handle != 0 {
		w32.ShowWindow(w.handle, w32.SW_HIDE)
	}
}

// RestoreFromTray shows a window that was hidden with HideToTray and brings it
// to the front.
func (w *Window) RestoreFromTray() {
	if w.handle != 0 {
		w32.ShowWindow(w.handle, w32.SW_RESTORE)
		w32.SetForegroundWindow(w.handle)
	}
}

// Quit closes the window for real, even if CloseToTray is set.
func (w *Window) Quit() {
	w.Close()
}
