package wui

import (
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

// EditAlign is the horizontal text alignment of an edit control.
type EditAlign int

const (
	EditAlignLeft EditAlign = iota
	EditAlignCenter
	EditAlignRight
)

const (
	emSetCueBanner = 0x1501
	emGetSel       = 0x00B0
)

// applyEditExtras applies the settings that need a window: placeholder text,
// number only input and alignment.
func (c *textEditControl) applyEditExtras() {
	if c.handle == 0 {
		return
	}
	if c.cue != "" {
		w32.SendMessage(c.handle, emSetCueBanner, boolToUintptr(c.cueAlways),
			uintptr(unsafe.Pointer(utf16Ptr(c.cue))))
	}
	c.applyEditStyle()
}

func (c *textEditControl) applyEditStyle() {
	style := uint(w32.GetWindowLong(c.handle, w32.GWL_STYLE))
	style &^= w32.ES_NUMBER | w32.ES_CENTER | w32.ES_RIGHT
	if c.numbersOnly {
		style |= w32.ES_NUMBER
	}
	switch c.align {
	case EditAlignCenter:
		style |= w32.ES_CENTER
	case EditAlignRight:
		style |= w32.ES_RIGHT
	}
	w32.SetWindowLong(c.handle, w32.GWL_STYLE, int32(style))
	w32.InvalidateRect(c.handle, nil, true)
}

// SetCueBanner sets the grayed hint text shown while an edit is empty, the
// "placeholder". If showWhenFocused is true it stays visible until the user
// types. It works for single line edits (EditLine) only.
func (c *textEditControl) SetCueBanner(text string, showWhenFocused bool) {
	c.cue, c.cueAlways = text, showWhenFocused
	if c.handle != 0 {
		w32.SendMessage(c.handle, emSetCueBanner, boolToUintptr(showWhenFocused),
			uintptr(unsafe.Pointer(utf16Ptr(text))))
	}
}

// SetNumbersOnly allows only the digits 0 to 9 to be typed. Pasting is not
// restricted by Windows.
func (c *textEditControl) SetNumbersOnly(on bool) {
	c.numbersOnly = on
	if c.handle != 0 {
		c.applyEditStyle()
	}
}

// SetTextAlign aligns the text left, centered or right.
func (c *textEditControl) SetTextAlign(a EditAlign) {
	c.align = a
	if c.handle != 0 {
		c.applyEditStyle()
	}
}

// utf16Selection returns the selection as offsets into the UTF-16 form of the
// text, which is what Windows uses.
func (c *textEditControl) utf16Selection() (start, end int) {
	var s, e uint32
	w32.SendMessage(c.handle, emGetSel, uintptr(unsafe.Pointer(&s)), uintptr(unsafe.Pointer(&e)))
	return int(s), int(e)
}

// SelectedText returns the selected text, empty if nothing is selected.
func (c *textEditControl) SelectedText() string {
	if c.handle == 0 {
		return ""
	}
	s, e := c.utf16Selection()
	text := stringToUTF16(c.Text())
	text = text[:len(text)-1] // without the terminating 0
	if s < 0 || e > len(text) || s >= e {
		return ""
	}
	return utf16ToString(text[s:e])
}

// ReplaceSelection replaces the selected text, or inserts at the cursor if
// nothing is selected. The change can be undone.
func (c *textEditControl) ReplaceSelection(text string) {
	if c.handle != 0 {
		w32.SendMessage(c.handle, w32.EM_REPLACESEL, 1, uintptr(unsafe.Pointer(utf16Ptr(text))))
	}
}

// AppendText adds text at the end and scrolls it into view. It is much faster
// than SetText(Text()+...) for logs.
func (c *textEditControl) AppendText(text string) {
	if c.handle == 0 {
		c.text += text
		return
	}
	w32.SendMessage(c.handle, w32.EM_SETSEL, ^uintptr(0), ^uintptr(0)) // caret to the end
	w32.SendMessage(c.handle, w32.EM_REPLACESEL, 0, uintptr(unsafe.Pointer(utf16Ptr(text))))
	w32.SendMessage(c.handle, w32.EM_SCROLLCARET, 0, 0)
}

// Undo reverts the last change and returns false if there was nothing to
// undo.
func (c *textEditControl) Undo() bool {
	if c.handle == 0 {
		return false
	}
	return w32.SendMessage(c.handle, w32.EM_UNDO, 0, 0) != 0
}

// CanUndo tells whether Undo has something to revert.
func (c *textEditControl) CanUndo() bool {
	return c.handle != 0 && w32.SendMessage(c.handle, w32.EM_CANUNDO, 0, 0) != 0
}

// Cut moves the selected text to the clipboard.
func (c *textEditControl) Cut() {
	if c.handle != 0 {
		w32.SendMessage(c.handle, w32.WM_CUT, 0, 0)
	}
}

// Copy copies the selected text to the clipboard.
func (c *textEditControl) Copy() {
	if c.handle != 0 {
		w32.SendMessage(c.handle, w32.WM_COPY, 0, 0)
	}
}

// Paste inserts the clipboard text at the cursor.
func (c *textEditControl) Paste() {
	if c.handle != 0 {
		w32.SendMessage(c.handle, w32.WM_PASTE, 0, 0)
	}
}

// DeleteSelection deletes the selected text without using the clipboard.
func (c *textEditControl) DeleteSelection() {
	if c.handle != 0 {
		w32.SendMessage(c.handle, w32.WM_CLEAR, 0, 0)
	}
}

// LineCount returns the number of lines of a multi line edit.
func (c *textEditControl) LineCount() int {
	if c.handle == 0 {
		return 0
	}
	return int(w32.SendMessage(c.handle, w32.EM_GETLINECOUNT, 0, 0))
}

// CursorLine returns the zero based line the cursor is on.
func (c *textEditControl) CursorLine() int {
	if c.handle == 0 {
		return 0
	}
	return int(w32.SendMessage(c.handle, w32.EM_LINEFROMCHAR, ^uintptr(0), 0))
}

// ScrollToCaret scrolls so that the cursor is visible.
func (c *textEditControl) ScrollToCaret() {
	if c.handle != 0 {
		w32.SendMessage(c.handle, w32.EM_SCROLLCARET, 0, 0)
	}
}
