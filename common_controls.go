package wui

import (
	"errors"
	"fmt"
	"time"
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

// OpenURL opens a web address, file or folder with the program Windows has
// registered for it, e.g. the default browser.
func OpenURL(url string) {
	nxShellExecute.Call(0, uintptr(unsafe.Pointer(utf16Ptr("open"))),
		uintptr(unsafe.Pointer(utf16Ptr(url))), 0, 0, 1)
}

// ---------------------------------------------------------------- LinkLabel

// LinkLabel is a label whose text can contain clickable links written in HTML
// style: `Visit <a href="https://example.com">our site</a> or
// <a id="help">get help</a>`.
type LinkLabel struct {
	textControl
	onClick func(url, id string)
}

var _ Control = (*LinkLabel)(nil)

// NewLinkLabel creates a label with links. Set its text with SetText.
func NewLinkLabel() *LinkLabel { return &LinkLabel{} }

func (*LinkLabel) canFocus() bool       { return true }
func (*LinkLabel) eatsTabs() bool       { return false }
func (l *LinkLabel) OnTabFocus() func() { return l.onTabFocus }
func (l *LinkLabel) SetOnTabFocus(f func()) { l.onTabFocus = f }

// SetOnClick sets the function called with the href and id of a clicked link.
// Without it, the href is opened with OpenURL.
func (l *LinkLabel) SetOnClick(f func(url, id string)) { l.onClick = f }

func (l *LinkLabel) create(id int) {
	initCommonControls()
	l.textControl.create(id, 0, "SysLink", w32.WS_TABSTOP)
}

const (
	nmClickCode  uint32 = 0xFFFFFFFE // NM_CLICK
	nmReturnCode uint32 = 0xFFFFFFFC // NM_RETURN
)

type nmLink struct {
	hdr       w32.NMHDR
	mask      uint32
	iLink     int32
	state     uint32
	stateMask uint32
	szID      [48]uint16
	szURL     [2084]uint16
}

func utf16Z(s []uint16) string {
	n := 0
	for n < len(s) && s[n] != 0 {
		n++
	}
	return utf16ToString(s[:n])
}

func (l *LinkLabel) handleNotify(code uint32, lParam uintptr) bool {
	if code != nmClickCode && code != nmReturnCode {
		return false
	}
	nm := (*nmLink)(unsafe.Pointer(lParam))
	url, id := utf16Z(nm.szURL[:]), utf16Z(nm.szID[:])
	if l.onClick != nil {
		l.onClick(url, id)
	} else if url != "" {
		OpenURL(url)
	}
	return true
}

// ------------------------------------------------------------ MonthCalendar

// MonthCalendar shows a month and lets the user pick a day.
type MonthCalendar struct {
	textControl
	date         time.Time
	weekNumbers  bool
	onChange     func(date time.Time)
}

var _ Control = (*MonthCalendar)(nil)

type systemTime struct {
	year, month, dayOfWeek, day, hour, minute, second, ms uint16
}

const (
	mcmGetCurSel      = 0x1001
	mcmSetCurSel      = 0x1002
	mcmGetMinReqRect  = 0x1009
	mcsWeekNumbers    = 0x0004
	mcnSelChange uint32 = 0xFFFFFD13
	mcnSelect    uint32 = 0xFFFFFD16
)

// NewMonthCalendar creates a calendar showing today's month.
func NewMonthCalendar() *MonthCalendar {
	return &MonthCalendar{date: time.Now()}
}

func (*MonthCalendar) canFocus() bool       { return true }
func (*MonthCalendar) eatsTabs() bool       { return false }
func (m *MonthCalendar) OnTabFocus() func() { return m.onTabFocus }
func (m *MonthCalendar) SetOnTabFocus(f func()) { m.onTabFocus = f }

func (m *MonthCalendar) create(id int) {
	initCommonControls()
	var style uint = w32.WS_TABSTOP
	if m.weekNumbers {
		style |= mcsWeekNumbers
	}
	m.textControl.create(id, 0, "SysMonthCal32", style)
	m.SetDate(m.date)
}

// SetShowWeekNumbers shows the week numbers at the left of the days. Call it
// before the window is shown.
func (m *MonthCalendar) SetShowWeekNumbers(on bool) { m.weekNumbers = on }

// Date returns the selected day, with the time of day set to midnight.
func (m *MonthCalendar) Date() time.Time {
	if m.handle != 0 {
		var st systemTime
		if w32.SendMessage(m.handle, mcmGetCurSel, 0, uintptr(unsafe.Pointer(&st))) != 0 {
			m.date = time.Date(int(st.year), time.Month(st.month), int(st.day), 0, 0, 0, 0, time.Local)
		}
	}
	return m.date
}

// SetDate selects a day.
func (m *MonthCalendar) SetDate(t time.Time) {
	m.date = t
	if m.handle != 0 {
		st := systemTime{
			year: uint16(t.Year()), month: uint16(t.Month()), day: uint16(t.Day()),
			dayOfWeek: uint16(t.Weekday()),
		}
		w32.SendMessage(m.handle, mcmSetCurSel, 0, uintptr(unsafe.Pointer(&st)))
	}
}

// SetOnChange sets the function called when the selected day changes.
func (m *MonthCalendar) SetOnChange(f func(date time.Time)) { m.onChange = f }

// PreferredSize returns the size that shows the whole month, to use with
// SetSize after the window is shown.
func (m *MonthCalendar) PreferredSize() (width, height int) {
	if m.handle == 0 {
		return 230, 160
	}
	var r w32.RECT
	w32.SendMessage(m.handle, mcmGetMinReqRect, 0, uintptr(unsafe.Pointer(&r)))
	return int(r.Right - r.Left), int(r.Bottom - r.Top)
}

func (m *MonthCalendar) handleNotify(code uint32, lParam uintptr) bool {
	if code == mcnSelChange || code == mcnSelect {
		d := m.Date()
		if m.onChange != nil {
			m.onChange(d)
		}
		return true
	}
	return false
}

// ------------------------------------------------------------------ HotKey

// HotKeyEdit is a box in which the user presses a key combination, like
// Ctrl+Shift+K, and sees it written out. Use it for configurable shortcuts.
// The result can be given to Window.RegisterHotKey.
type HotKeyEdit struct {
	textControl
	key      int
	mods     HotKeyMod
	onChange func()
}

var _ Control = (*HotKeyEdit)(nil)

// NewHotKeyEdit creates an empty hot key box.
func NewHotKeyEdit() *HotKeyEdit { return &HotKeyEdit{} }

func (*HotKeyEdit) canFocus() bool       { return true }
func (*HotKeyEdit) eatsTabs() bool       { return false }
func (h *HotKeyEdit) OnTabFocus() func() { return h.onTabFocus }
func (h *HotKeyEdit) SetOnTabFocus(f func()) { h.onTabFocus = f }

const (
	hkmSetHotKey = 0x0401
	hkmGetHotKey = 0x0402
	hotkeyfShift = 1
	hotkeyfCtrl  = 2
	hotkeyfAlt   = 4
)

func (h *HotKeyEdit) create(id int) {
	initCommonControls()
	h.textControl.create(id, w32.WS_EX_CLIENTEDGE, "msctls_hotkey32", w32.WS_TABSTOP)
	h.SetHotKey(h.key, h.mods)
}

// SetHotKey shows a key combination: key is a virtual key code, e.g. 'K' or
// 0x70 for F1, mods the modifiers.
func (h *HotKeyEdit) SetHotKey(key int, mods HotKeyMod) {
	h.key, h.mods = key, mods
	if h.handle == 0 {
		return
	}
	var f int
	if mods&ModShift != 0 {
		f |= hotkeyfShift
	}
	if mods&ModControl != 0 {
		f |= hotkeyfCtrl
	}
	if mods&ModAlt != 0 {
		f |= hotkeyfAlt
	}
	w32.SendMessage(h.handle, hkmSetHotKey, uintptr(key&0xFF|f<<8), 0)
}

// HotKey returns the key combination the user entered. key is 0 if nothing
// was entered.
func (h *HotKeyEdit) HotKey() (key int, mods HotKeyMod) {
	if h.handle == 0 {
		return h.key, h.mods
	}
	r := int(w32.SendMessage(h.handle, hkmGetHotKey, 0, 0))
	key = r & 0xFF
	f := (r >> 8) & 0xFF
	if f&hotkeyfShift != 0 {
		mods |= ModShift
	}
	if f&hotkeyfCtrl != 0 {
		mods |= ModControl
	}
	if f&hotkeyfAlt != 0 {
		mods |= ModAlt
	}
	return key, mods
}

// SetOnChange sets the function called when the combination changes.
func (h *HotKeyEdit) SetOnChange(f func()) { h.onChange = f }

func (h *HotKeyEdit) handleNotification(cmd uintptr) {
	if cmd == 0x0300 && h.onChange != nil { // EN_CHANGE
		h.onChange()
	}
}

// --------------------------------------------------------------- IPAddress

// IPAddressEdit is the box for entering an IPv4 address with four fields.
type IPAddressEdit struct {
	textControl
	value    uint32
	onChange func()
}

var _ Control = (*IPAddressEdit)(nil)

const (
	ipmClearAddress = 0x0464
	ipmSetAddress   = 0x0465
	ipmGetAddress   = 0x0466
	ipmIsBlank      = 0x0469
)

// NewIPAddressEdit creates an empty IP address box.
func NewIPAddressEdit() *IPAddressEdit { return &IPAddressEdit{} }

func (*IPAddressEdit) canFocus() bool       { return true }
func (*IPAddressEdit) eatsTabs() bool       { return false }
func (e *IPAddressEdit) OnTabFocus() func() { return e.onTabFocus }
func (e *IPAddressEdit) SetOnTabFocus(f func()) { e.onTabFocus = f }

func (e *IPAddressEdit) create(id int) {
	initCommonControls()
	e.textControl.create(id, w32.WS_EX_CLIENTEDGE, "SysIPAddress32", w32.WS_TABSTOP)
	if e.value != 0 {
		w32.SendMessage(e.handle, ipmSetAddress, 0, uintptr(e.value))
	}
}

// SetAddress shows an address like "192.168.0.1". An empty string clears the
// box.
func (e *IPAddressEdit) SetAddress(s string) error {
	if s == "" {
		e.value = 0
		if e.handle != 0 {
			w32.SendMessage(e.handle, ipmClearAddress, 0, 0)
		}
		return nil
	}
	var a, b, c, d uint8
	if n, err := fmt.Sscanf(s, "%d.%d.%d.%d", &a, &b, &c, &d); err != nil || n != 4 {
		return errors.New("not an IPv4 address: " + s)
	}
	e.value = uint32(a)<<24 | uint32(b)<<16 | uint32(c)<<8 | uint32(d)
	if e.handle != 0 {
		w32.SendMessage(e.handle, ipmSetAddress, 0, uintptr(e.value))
	}
	return nil
}

// Address returns the entered address as text and false if some field is
// still empty.
func (e *IPAddressEdit) Address() (string, bool) {
	v := e.value
	if e.handle != 0 {
		if w32.SendMessage(e.handle, ipmIsBlank, 0, 0) != 0 {
			return "", false
		}
		var got uint32
		n := w32.SendMessage(e.handle, ipmGetAddress, 0, uintptr(unsafe.Pointer(&got)))
		if n != 4 {
			return "", false
		}
		v = got
	}
	return fmt.Sprintf("%d.%d.%d.%d", v>>24&0xFF, v>>16&0xFF, v>>8&0xFF, v&0xFF), true
}

// SetOnChange sets the function called when any field changes.
func (e *IPAddressEdit) SetOnChange(f func()) { e.onChange = f }

func (e *IPAddressEdit) handleNotification(cmd uintptr) {
	if cmd == 0x0300 && e.onChange != nil { // EN_CHANGE
		e.onChange()
	}
}
