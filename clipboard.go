package wui

import (
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

// ClipboardText returns the text on the system clipboard. ok is false if the
// clipboard does not contain text.
func ClipboardText() (text string, ok bool) {
	if !w32.IsClipboardFormatAvailable(w32.CF_UNICODETEXT) {
		return "", false
	}
	if !w32.OpenClipboard(0) {
		return "", false
	}
	defer w32.CloseClipboard()

	h := w32.GetClipboardData(w32.CF_UNICODETEXT)
	if h == 0 {
		return "", false
	}
	p := w32.GlobalLock(w32.HGLOBAL(h))
	if p == nil {
		return "", false
	}
	defer w32.GlobalUnlock(w32.HGLOBAL(h))

	var s []uint16
	for i := uintptr(0); ; i += 2 {
		c := *(*uint16)(unsafe.Pointer(uintptr(p) + i))
		if c == 0 {
			break
		}
		s = append(s, c)
	}
	return utf16ToString(s), true
}

// SetClipboardText replaces the clipboard contents with the given text.
func SetClipboardText(text string) bool {
	u := stringToUTF16(text)
	if !w32.OpenClipboard(0) {
		return false
	}
	defer w32.CloseClipboard()
	w32.EmptyClipboard()

	h := w32.GlobalAlloc(w32.GMEM_MOVEABLE, uint32(len(u)*2))
	if h == 0 {
		return false
	}
	p := w32.GlobalLock(h)
	if p == nil {
		w32.GlobalFree(h)
		return false
	}
	for i, c := range u {
		*(*uint16)(unsafe.Pointer(uintptr(p) + uintptr(i)*2)) = c
	}
	w32.GlobalUnlock(h)
	if w32.SetClipboardData(w32.CF_UNICODETEXT, w32.HANDLE(h)) == 0 {
		w32.GlobalFree(h)
		return false
	}
	return true
}
