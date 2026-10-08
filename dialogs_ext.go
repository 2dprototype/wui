package wui

import (
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

// ------------------------------------------------------------ input dialog

// InputDialog asks the user for a line of text in a small modal window.
// parent is not used for placement (the dialog opens over the current top
// window); it is accepted so calls read naturally. ok is false if the user
// cancelled.
func InputDialog(title, prompt, initial string) (text string, ok bool) {
	return inputDialog(title, prompt, initial, false)
}

// PasswordDialog is InputDialog with the typed text hidden.
func PasswordDialog(title, prompt string) (text string, ok bool) {
	return inputDialog(title, prompt, "", true)
}

func inputDialog(title, prompt, initial string, password bool) (string, bool) {
	d := NewWindow()
	d.SetTitle(title)
	d.SetInnerSize(360, 118)
	d.SetHasMinButton(false)
	d.SetHasMaxButton(false)
	d.SetResizable(false)
	d.SetCenterOnShow(true)

	l := NewLabel()
	l.SetBounds(12, 12, 336, 20)
	l.SetText(prompt)
	d.Add(l)

	e := NewEditLine()
	e.SetBounds(12, 38, 336, 24)
	e.SetText(initial)
	e.SetIsPassword(password)
	d.Add(e)

	var result string
	accepted := false

	ok := NewButton()
	ok.SetText("OK")
	ok.SetBounds(188, 80, 78, 26)
	ok.SetOnClick(func() {
		result = e.Text()
		accepted = true
		d.Close()
	})
	d.Add(ok)

	cancel := NewButton()
	cancel.SetText("Cancel")
	cancel.SetBounds(272, 80, 78, 26)
	cancel.SetOnClick(func() { d.Close() })
	d.Add(cancel)

	d.SetDefaultButton(ok)
	d.SetCancelButton(cancel)
	d.SetOnShow(func() {
		e.Focus()
		e.SelectAll()
	})
	d.ShowModal()
	return result, accepted
}

// DialogResult is the button the user chose in a dialog.
type DialogResult int

const (
	DialogCancel DialogResult = iota
	DialogYes
	DialogNo
	DialogOK
	DialogRetry
	DialogClose
)

func resultFromID(id int) DialogResult {
	switch id {
	case 1:
		return DialogOK
	case 2:
		return DialogCancel
	case 4:
		return DialogRetry
	case 6:
		return DialogYes
	case 7:
		return DialogNo
	case 8:
		return DialogClose
	}
	return DialogCancel
}

// MessageBoxYesNoCancel asks a question with three answers.
func MessageBoxYesNoCancel(caption, text string) DialogResult {
	return resultFromID(msgBox(caption, text, w32.MB_YESNOCANCEL|w32.MB_ICONQUESTION))
}

// MessageBoxRetryCancel reports a failure the user may retry.
func MessageBoxRetryCancel(caption, text string) DialogResult {
	return resultFromID(msgBox(caption, text, w32.MB_RETRYCANCEL|w32.MB_ICONWARNING))
}

// MessageBoxFor shows a message box that belongs to the given window, so it
// stays on top of it and blocks only it. flags are the MB_ values of
// MessageBoxCustom; the result is the ID of the pressed button.
func MessageBoxFor(parent *Window, caption, text string, flags uint) int {
	var h w32.HWND
	if parent != nil {
		h = parent.handle
	}
	return w32.MessageBox(h, text, caption, flags)
}

// ------------------------------------------------------------- task dialog

// TaskButtons is the set of standard buttons of a task dialog, combine them
// with |.
type TaskButtons int

const (
	TaskOK     TaskButtons = 0x01
	TaskYes    TaskButtons = 0x02
	TaskNo     TaskButtons = 0x04
	TaskCancel TaskButtons = 0x08
	TaskRetry  TaskButtons = 0x10
	TaskClose  TaskButtons = 0x20
)

// TaskIcon is the icon of a task dialog.
type TaskIcon int

const (
	TaskIconNone        TaskIcon = 0
	TaskIconWarning     TaskIcon = 0xFFFF
	TaskIconError       TaskIcon = 0xFFFE
	TaskIconInformation TaskIcon = 0xFFFD
	TaskIconShield      TaskIcon = 0xFFFC
)

// TaskDialog shows the modern message box with a big blue main instruction
// and a smaller explanation below it. On systems or setups where it is not
// available it falls back to a classic message box.
func TaskDialog(parent *Window, title, instruction, content string, buttons TaskButtons, icon TaskIcon) DialogResult {
	if nxTaskDialog.Find() != nil {
		flags := uint(w32.MB_OK)
		switch buttons {
		case TaskYes | TaskNo:
			flags = w32.MB_YESNO
		case TaskYes | TaskNo | TaskCancel:
			flags = w32.MB_YESNOCANCEL
		case TaskOK | TaskCancel:
			flags = w32.MB_OKCANCEL
		case TaskRetry | TaskCancel:
			flags = w32.MB_RETRYCANCEL
		}
		return resultFromID(MessageBoxFor(parent, title, instruction+"\n\n"+content, flags))
	}
	var owner uintptr
	if parent != nil {
		owner = uintptr(parent.handle)
	}
	if buttons == 0 {
		buttons = TaskOK
	}
	var pressed int32
	nxTaskDialog.Call(
		owner, 0,
		uintptr(unsafe.Pointer(utf16Ptr(title))),
		uintptr(unsafe.Pointer(utf16Ptr(instruction))),
		uintptr(unsafe.Pointer(utf16Ptr(content))),
		uintptr(buttons), uintptr(icon),
		uintptr(unsafe.Pointer(&pressed)),
	)
	return resultFromID(int(pressed))
}

// ------------------------------------------------------------- font dialog

// FontDialog is the standard dialog for choosing a font.
type FontDialog struct {
	desc   FontDesc
	color  Color
	points int
}

// NewFontDialog creates a font dialog that starts with Segoe UI 9 pt.
func NewFontDialog() *FontDialog {
	return &FontDialog{desc: FontDesc{Name: "Segoe UI", Height: -12}}
}

// SetFont sets the font the dialog starts with. Height is in pixels, negative
// for the character height as in FontDesc.
func (d *FontDialog) SetFont(f FontDesc) { d.desc = f }

// Font returns the chosen font, ready for NewFont.
func (d *FontDialog) Font() FontDesc { return d.desc }

// Points returns the chosen size in points.
func (d *FontDialog) Points() int { return d.points }

// SetColor sets the text color the dialog starts with.
func (d *FontDialog) SetColor(c Color) { d.color = c }

// Color returns the chosen text color.
func (d *FontDialog) Color() Color { return d.color }

type chooseFont struct {
	structSize   uint32
	owner        uintptr
	dc           uintptr
	logFont      *w32.LOGFONT
	pointSize    int32
	flags        uint32
	colors       uint32
	custData     uintptr
	hook         uintptr
	templateName *uint16
	instance     uintptr
	style        *uint16
	fontType     uint16
	_            uint16
	sizeMin      int32
	sizeMax      int32
}

// Execute shows the dialog (parent may be nil) and returns true if the user
// pressed OK.
func (d *FontDialog) Execute(parent *Window) bool {
	var weight int32 = w32.FW_NORMAL
	if d.desc.Bold {
		weight = w32.FW_BOLD
	}
	b := func(v bool) byte {
		if v {
			return 1
		}
		return 0
	}
	lf := w32.LOGFONT{
		Height: int32(d.desc.Height), Weight: weight,
		Italic: b(d.desc.Italic), Underline: b(d.desc.Underlined), StrikeOut: b(d.desc.StrikedOut),
		CharSet: w32.DEFAULT_CHARSET,
	}
	lf.SetFaceName(d.desc.Name)
	cf := chooseFont{
		logFont: &lf,
		flags:   0x1 | 0x40 | 0x100, // CF_SCREENFONTS | CF_INITTOLOGFONTSTRUCT | CF_EFFECTS
		colors:  uint32(d.color),
	}
	cf.structSize = uint32(unsafe.Sizeof(cf))
	if parent != nil {
		cf.owner = uintptr(parent.handle)
	}
	r, _, _ := nxChooseFont.Call(uintptr(unsafe.Pointer(&cf)))
	if r == 0 {
		return false
	}
	d.desc = FontDesc{
		Name:       utf16Z(lf.FaceName[:]),
		Height:     int(lf.Height),
		Bold:       lf.Weight >= 700,
		Italic:     lf.Italic != 0,
		Underlined: lf.Underline != 0,
		StrikedOut: lf.StrikeOut != 0,
	}
	d.points = int(cf.pointSize) / 10
	d.color = Color(cf.colors)
	return true
}
