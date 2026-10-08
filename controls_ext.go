package wui

import (
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

// ------------------------------------------------------------------ Button

// ButtonKind chooses the look and behavior of a Button.
type ButtonKind int

const (
	// ButtonNormal is the ordinary push button.
	ButtonNormal ButtonKind = iota
	// ButtonSplit has a small arrow next to the label that opens a menu, see
	// Button.SetDropDownMenu.
	ButtonSplit
	// ButtonCommandLink is the big flat button with a green arrow, a title
	// and an explanatory note (Button.SetNote) used in wizards.
	ButtonCommandLink
)

const (
	bsSplitButton   = 0x000C
	bsCommandLink   = 0x000E
	bcmSetImageList = 0x1602
	bcmSetNote      = 0x1609
	bcnDropDown uint32 = 0xFFFFFB20
)

type buttonImageList struct {
	himl   uintptr
	margin w32.RECT
	align  uint32
}

func (b *Button) styleBits() uint {
	var style uint
	switch b.kind {
	case ButtonSplit:
		style = bsSplitButton
	case ButtonCommandLink:
		style = bsCommandLink
	}
	if b.isDefault {
		style |= w32.BS_DEFPUSHBUTTON // 1, which makes the other kinds default too
	}
	return style
}

func (b *Button) applyExtras() {
	if b.note != "" {
		w32.SendMessage(b.handle, bcmSetNote, 0, uintptr(unsafe.Pointer(utf16Ptr(b.note))))
	}
	if b.icon != nil {
		bil := buttonImageList{himl: b.icon.handle}
		bil.margin = w32.RECT{Left: 4, Right: 4}
		w32.SendMessage(b.handle, bcmSetImageList, 0, uintptr(unsafe.Pointer(&bil)))
	}
}

// SetKind changes the look of the button. Call it before the window is shown.
func (b *Button) SetKind(k ButtonKind) { b.kind = k }

// SetNote sets the explanation under the title of a ButtonCommandLink.
func (b *Button) SetNote(text string) {
	b.note = text
	if b.handle != 0 {
		w32.SendMessage(b.handle, bcmSetNote, 0, uintptr(unsafe.Pointer(utf16Ptr(text))))
	}
}

// SetIcon shows an icon left of the text.
func (b *Button) SetIcon(icon *Icon) {
	size := w32.GetSystemMetrics(w32.SM_CXSMICON)
	if size <= 0 {
		size = 16
	}
	il := NewImageList(size, size)
	il.AddIcon(icon)
	b.SetImages(il)
}

// SetImages shows the first image of the list left of the text. (An image
// list with six images gives different pictures for the normal, hot, pressed,
// disabled, focused and stylus-hot states.)
func (b *Button) SetImages(il *ImageList) {
	b.icon = il
	if b.handle != 0 {
		b.applyExtras()
		w32.InvalidateRect(b.handle, nil, true)
	}
}

// SetDefault draws the button as the default button. To also make Enter click
// it, use Window.SetDefaultButton.
func (b *Button) SetDefault(on bool) {
	b.isDefault = on
	if b.handle != 0 {
		w32.SendMessage(b.handle, w32.BM_SETSTYLE, uintptr(b.styleBits()), 1)
	}
}

// SetDropDownMenu sets the menu a ButtonSplit opens from its arrow.
func (b *Button) SetDropDownMenu(m *PopupMenu) { b.dropMenu = m }

// SetOnDropDown sets the function called when the arrow of a ButtonSplit is
// clicked. If a drop down menu is set it is shown afterwards.
func (b *Button) SetOnDropDown(f func()) { b.onDropDown = f }

func (b *Button) handleNotify(code uint32, lParam uintptr) bool {
	if code != bcnDropDown {
		return false
	}
	if b.onDropDown != nil {
		b.onDropDown()
	}
	if b.dropMenu != nil {
		nm := (*struct {
			hdr w32.NMHDR
			rc  w32.RECT
		})(unsafe.Pointer(lParam))
		x, y := w32.ClientToScreen(b.handle, int(nm.rc.Left), int(nm.rc.Bottom))
		if win := windowOfControl(b.parent); win != nil {
			b.dropMenu.ShowAtScreen(win, x, y)
		}
	}
	return true
}

// ---------------------------------------------------------------- CheckBox

func (c *CheckBox) styleBits() uint {
	style := uint(w32.BS_AUTOCHECKBOX)
	if c.threeState {
		style = w32.BS_AUTO3STATE
	}
	if c.pushLike {
		style |= w32.BS_PUSHLIKE
	}
	return style
}

func (c *CheckBox) checkState() uintptr {
	if c.threeState && c.indeterminate {
		return w32.BST_INDETERMINATE
	}
	return toCheckState(c.checked)
}

func (c *CheckBox) restyle() {
	if c.handle == 0 {
		return
	}
	style := uint(w32.GetWindowLong(c.handle, w32.GWL_STYLE))
	style &^= 0xF | w32.BS_PUSHLIKE
	style |= c.styleBits()
	w32.SetWindowLong(c.handle, w32.GWL_STYLE, int32(style))
	w32.SendMessage(c.handle, w32.BM_SETCHECK, c.checkState(), 0)
	w32.InvalidateRect(c.handle, nil, true)
}

// SetThreeState lets the check box cycle through checked, unchecked and
// indeterminate (the "some of the children are checked" state).
func (c *CheckBox) SetThreeState(on bool) {
	c.threeState = on
	if !on {
		c.indeterminate = false
	}
	c.restyle()
}

// ThreeState tells whether the check box has the third state.
func (c *CheckBox) ThreeState() bool { return c.threeState }

// Indeterminate tells whether the check box is in the third state.
func (c *CheckBox) Indeterminate() bool { return c.threeState && c.indeterminate }

// SetIndeterminate puts a three state check box in or out of the third state.
func (c *CheckBox) SetIndeterminate(on bool) {
	if !c.threeState {
		return
	}
	c.indeterminate = on
	if c.handle != 0 {
		w32.SendMessage(c.handle, w32.BM_SETCHECK, c.checkState(), 0)
	}
}

// SetPushLike draws the check box as a button that stays pressed when
// checked, a toggle button.
func (c *CheckBox) SetPushLike(on bool) {
	c.pushLike = on
	c.restyle()
}

// ------------------------------------------------------------- ProgressBar

// ProgressState is the color state of a ProgressBar.
type ProgressState int

const (
	// ProgressNormal is the usual green bar.
	ProgressNormal ProgressState = iota
	// ProgressError turns the bar red.
	ProgressError
	// ProgressPaused turns the bar yellow.
	ProgressPaused
)

const pbmSetState = 0x0410

// SetState turns the bar red (error) or yellow (paused), or back to green.
func (p *ProgressBar) SetState(s ProgressState) {
	p.state = s
	p.applyState()
}

// State returns the color state.
func (p *ProgressBar) State() ProgressState { return p.state }

func (p *ProgressBar) applyState() {
	if p.handle != 0 && !p.movesForever {
		w32.SendMessage(p.handle, pbmSetState, uintptr(p.state)+1, 0)
	}
}

// ---------------------------------------------------------------- ComboBox

func (e *ComboBox) dropStyle() uint {
	if e.editable {
		return w32.CBS_DROPDOWN
	}
	return w32.CBS_DROPDOWNLIST
}

// SetEditable lets the user type any text in the combo box instead of only
// choosing from the list. Read the text with Text(). Call it before the
// window is shown.
func (e *ComboBox) SetEditable(on bool) { e.editable = on }

// Editable tells whether the user can type in the combo box.
func (e *ComboBox) Editable() bool { return e.editable }

// SetOnTextChange sets the function called when the user edits the text of an
// editable combo box.
func (e *ComboBox) SetOnTextChange(f func()) { e.onTextChange = f }

// -------------------------------------------------------------------- Menu

func (m *MenuString) applyState() {
	if m.menu == 0 {
		return
	}
	var info w32.MENUITEMINFO
	info.Mask = w32.MIIM_STATE
	if m.checked {
		info.State |= w32.MFS_CHECKED
	}
	if m.disabled {
		info.State |= w32.MFS_DISABLED
	}
	w32.SetMenuItemInfo(m.menu, m.id, false, &info)
}

// applyExtra sets what a menu needs when it is created: the radio look.
func (m *MenuString) applyExtra() {
	if m.menu == 0 {
		return
	}
	if m.radio {
		var info w32.MENUITEMINFO
		info.Mask = w32.MIIM_FTYPE
		info.Type = w32.MFT_RADIOCHECK
		w32.SetMenuItemInfo(m.menu, m.id, false, &info)
	}
	if m.image != 0 {
		var info w32.MENUITEMINFO
		info.Mask = w32.MIIM_BITMAP
		info.BmpItem = m.image
		w32.SetMenuItemInfo(m.menu, m.id, false, &info)
	}
	m.applyState()
}

// SetEnabled enables or grays out the menu item.
func (m *MenuString) SetEnabled(on bool) {
	m.disabled = !on
	m.applyState()
}

// Enabled tells whether the item can be chosen.
func (m *MenuString) Enabled() bool { return !m.disabled }

// SetRadio shows the check mark of this item as a round bullet, for groups of
// items of which only one is checked.
func (m *MenuString) SetRadio(on bool) {
	m.radio = on
	if m.menu != 0 {
		var info w32.MENUITEMINFO
		info.Mask = w32.MIIM_FTYPE
		if on {
			info.Type = w32.MFT_RADIOCHECK
		}
		w32.SetMenuItemInfo(m.menu, m.id, false, &info)
	}
}

// SetImage shows an icon in front of the item. The image is converted to a
// 32 bit bitmap, 16x16 pixels is the usual size. Call it before the window is
// shown.
func (m *MenuString) SetImage(img *Image) {
	if img == nil {
		m.image = 0
		return
	}
	m.image = img.bitmap
}

// SelectRadio checks item and unchecks all others of the group.
func SelectRadio(item *MenuString, group ...*MenuString) {
	for _, o := range group {
		o.SetChecked(o == item)
	}
}
