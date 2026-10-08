package wui

import (
	"runtime"
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

// NativeToolBar is the real Windows tool bar: flat buttons with icons from an
// ImageList, optional text, toggle buttons, buttons with a drop down menu and
// separators. (ToolBar, the older control, is only a row of ordinary buttons.)
//
//	il := wui.NewImageList(24, 24)
//	open := il.AddIcon(openIcon)
//	tb := wui.NewNativeToolBar()
//	tb.SetImages(il)
//	tb.AddButton("Open", open, func() { ... })
//	tb.AddSeparator()
//	tb.AddToggle("Bold", bold, func(on bool) { ... })
type NativeToolBar struct {
	control
	id       int
	buttons  []*NativeToolButton
	images   *ImageList
	showText bool
	nextID   int
}

var _ Control = (*NativeToolBar)(nil)

// NativeToolButton is one entry of a NativeToolBar.
type NativeToolButton struct {
	tb       *NativeToolBar
	cmd      int
	text     string
	image    int
	sep      bool
	toggle   bool
	checked  bool
	disabled bool
	hidden   bool
	onClick  func()
	onToggle func(on bool)
	menu     *PopupMenu
	Tag      interface{}
}

const (
	tbClassName = "ToolbarWindow32"

	tbEnableButton     = 0x0401
	tbCheckButton      = 0x0402
	tbIsButtonChecked  = 0x040A
	tbHideButton       = 0x0404
	tbButtonStructSize = 0x041E
	tbGetItemRect      = 0x041D
	tbAutoSize         = 0x0421
	tbSetImageList     = 0x0430
	tbAddButtonsW      = 0x0444
	tbSetExtendedStyle = 0x0454
	tbGetButtonSize    = 0x043A

	tbstyleTooltips    = 0x0100
	tbstyleFlat        = 0x0800
	tbstyleList        = 0x1000
	tbstyleTransparent = 0x8000
	ccsNoParentAlign   = 0x0008
	ccsNoResize        = 0x0004
	ccsNoDivider       = 0x0040

	tbstyleExDrawDDArrows  = 0x0001
	tbstyleExMixedButtons  = 0x0008
	tbstyleExHideClipped   = 0x0010

	tbsStateChecked = 0x01
	tbsStateEnabled = 0x04
	tbsStateHidden  = 0x08

	tbsButton    = 0x00
	tbsSep       = 0x01
	tbsCheck     = 0x02
	tbsDropDown  = 0x08
	tbsAutoSize  = 0x10
	tbsShowText  = 0x40

	tbnDropDown uint32 = 0xFFFFFD3A // TBN_FIRST - 10
)

type tbButtonW struct {
	iBitmap   int32
	idCommand int32
	fsState   byte
	fsStyle   byte
	_         [unsafe.Sizeof(uintptr(0)) - 2]byte
	dwData    uintptr
	iString   uintptr
}

// NewNativeToolBar creates an empty tool bar that shows button texts next to
// the icons. Put it at the top of a window with anchors AnchorMinAndMax
// horizontally and set its height to PreferredHeight (or about 28).
func NewNativeToolBar() *NativeToolBar {
	return &NativeToolBar{showText: true, nextID: 1}
}

func (*NativeToolBar) canFocus() bool { return false }
func (*NativeToolBar) eatsTabs() bool { return false }

func (t *NativeToolBar) create(id int) {
	t.id = id
	initCommonControls()
	style := uint(w32.WS_CHILD|w32.WS_VISIBLE) | tbstyleFlat | tbstyleTooltips | tbstyleTransparent |
		ccsNoParentAlign | ccsNoResize | ccsNoDivider
	if t.showText {
		style |= tbstyleList
	}
	t.control.create(id, 0, tbClassName, style)
	w32.SendMessage(t.handle, tbButtonStructSize, unsafe.Sizeof(tbButtonW{}), 0)
	w32.SendMessage(t.handle, tbSetExtendedStyle, 0, tbstyleExDrawDDArrows|tbstyleExMixedButtons)
	if t.images != nil {
		w32.SendMessage(t.handle, tbSetImageList, 0, t.images.handle)
	}
	for _, b := range t.buttons {
		t.insert(b)
	}
	w32.SendMessage(t.handle, tbAutoSize, 0, 0)
}

func (t *NativeToolBar) insert(b *NativeToolButton) {
	var bt tbButtonW
	var text *uint16
	bt.iBitmap = int32(b.image)
	bt.idCommand = int32(b.cmd)
	if !b.disabled {
		bt.fsState |= tbsStateEnabled
	}
	if b.checked {
		bt.fsState |= tbsStateChecked
	}
	if b.hidden {
		bt.fsState |= tbsStateHidden
	}
	switch {
	case b.sep:
		bt.fsStyle = tbsSep
		bt.iBitmap = 8 // width of the separator
	default:
		bt.fsStyle = tbsAutoSize
		if b.toggle {
			bt.fsStyle |= tbsCheck
		}
		if b.menu != nil {
			bt.fsStyle |= tbsDropDown
		}
		if t.showText && b.text != "" {
			bt.fsStyle |= tbsShowText
			text = utf16Ptr(b.text)
			bt.iString = uintptr(unsafe.Pointer(text))
		}
	}
	w32.SendMessage(t.handle, tbAddButtonsW, 1, uintptr(unsafe.Pointer(&bt)))
	runtime.KeepAlive(text)
}

func (t *NativeToolBar) add(b *NativeToolButton) *NativeToolButton {
	b.tb = t
	b.cmd = t.nextID
	t.nextID++
	t.buttons = append(t.buttons, b)
	if t.handle != 0 {
		t.insert(b)
		w32.SendMessage(t.handle, tbAutoSize, 0, 0)
	}
	return b
}

// SetImages gives the tool bar the icons the buttons refer to by index.
func (t *NativeToolBar) SetImages(il *ImageList) {
	t.images = il
	if t.handle != 0 && il != nil {
		w32.SendMessage(t.handle, tbSetImageList, 0, il.handle)
	}
}

// SetShowText chooses whether the buttons show their text. Without text only
// icons are shown. It must be called before the window is shown.
func (t *NativeToolBar) SetShowText(on bool) { t.showText = on }

// AddButton appends a push button with a text, an icon (index into the image
// list, -1 for none) and a click function.
func (t *NativeToolBar) AddButton(text string, image int, onClick func()) *NativeToolButton {
	return t.add(&NativeToolButton{text: text, image: image, onClick: onClick})
}

// AddToggle appends a button that stays pressed until clicked again.
func (t *NativeToolBar) AddToggle(text string, image int, onToggle func(on bool)) *NativeToolButton {
	return t.add(&NativeToolButton{text: text, image: image, toggle: true, onToggle: onToggle})
}

// AddDropDown appends a button with a small arrow that opens the menu. A click
// on the button itself calls onClick, which may be nil.
func (t *NativeToolBar) AddDropDown(text string, image int, onClick func(), menu *PopupMenu) *NativeToolButton {
	return t.add(&NativeToolButton{text: text, image: image, onClick: onClick, menu: menu})
}

// AddSeparator appends a thin gap between groups of buttons.
func (t *NativeToolBar) AddSeparator() {
	t.add(&NativeToolButton{sep: true, image: -1})
}

// Buttons returns the buttons in order, including separators.
func (t *NativeToolBar) Buttons() []*NativeToolButton {
	return append([]*NativeToolButton(nil), t.buttons...)
}

// PreferredHeight returns the height that fits the buttons.
func (t *NativeToolBar) PreferredHeight() int {
	if t.handle == 0 {
		return 28
	}
	r := w32.SendMessage(t.handle, tbGetButtonSize, 0, 0)
	return hiword(r) + 4
}

// SetEnabled enables or disables the button.
func (b *NativeToolButton) SetEnabled(on bool) {
	b.disabled = !on
	if b.tb != nil && b.tb.handle != 0 {
		w32.SendMessage(b.tb.handle, tbEnableButton, uintptr(b.cmd), boolToUintptr(on))
	}
}

// Enabled tells whether the button can be clicked.
func (b *NativeToolButton) Enabled() bool { return !b.disabled }

// SetVisible shows or hides the button.
func (b *NativeToolButton) SetVisible(on bool) {
	b.hidden = !on
	if b.tb != nil && b.tb.handle != 0 {
		w32.SendMessage(b.tb.handle, tbHideButton, uintptr(b.cmd), boolToUintptr(!on))
	}
}

// SetChecked presses or releases a toggle button.
func (b *NativeToolButton) SetChecked(on bool) {
	b.checked = on
	if b.tb != nil && b.tb.handle != 0 {
		w32.SendMessage(b.tb.handle, tbCheckButton, uintptr(b.cmd), boolToUintptr(on))
	}
}

// Checked tells whether a toggle button is pressed.
func (b *NativeToolButton) Checked() bool {
	if b.tb != nil && b.tb.handle != 0 && b.toggle {
		return w32.SendMessage(b.tb.handle, tbIsButtonChecked, uintptr(b.cmd), 0) != 0
	}
	return b.checked
}

// SetOnClick replaces the click function.
func (b *NativeToolButton) SetOnClick(f func()) { b.onClick = f }

func (t *NativeToolBar) button(cmd int) *NativeToolButton {
	for _, b := range t.buttons {
		if b.cmd == cmd {
			return b
		}
	}
	return nil
}

func (t *NativeToolBar) handleCommand(id, code int) {
	b := t.button(id)
	if b == nil {
		return
	}
	if b.toggle {
		b.checked = w32.SendMessage(t.handle, tbIsButtonChecked, uintptr(id), 0) != 0
		if b.onToggle != nil {
			b.onToggle(b.checked)
		}
	}
	if b.onClick != nil {
		b.onClick()
	}
}

func (t *NativeToolBar) handleNotify(code uint32, lParam uintptr) bool {
	if code != tbnDropDown {
		return false
	}
	nm := (*struct {
		hdr   w32.NMHDR
		iItem int32
	})(unsafe.Pointer(lParam))
	b := t.button(int(nm.iItem))
	if b == nil || b.menu == nil {
		return true
	}
	var r w32.RECT
	w32.SendMessage(t.handle, tbGetItemRect, indexOfButton(t, b), uintptr(unsafe.Pointer(&r)))
	x, y := w32.ClientToScreen(t.handle, int(r.Left), int(r.Bottom))
	if win := windowOfControl(t.parent); win != nil {
		b.menu.ShowAtScreen(win, x, y)
	}
	return true
}

// indexOfButton is the zero based position of b as the tool bar counts it.
func indexOfButton(t *NativeToolBar, b *NativeToolButton) uintptr {
	for i, o := range t.buttons {
		if o == b {
			return uintptr(i)
		}
	}
	return 0
}

// windowOfControl finds the window a container belongs to.
func windowOfControl(c Container) *Window {
	for c != nil {
		if w, ok := c.(*Window); ok {
			return w
		}
		c = c.Parent()
	}
	return nil
}
