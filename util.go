package wui

import (
	"unicode/utf16"
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

var unsafeSizeofMonitorInfo = unsafe.Sizeof(w32.MONITORINFO{})

// stringToUTF16 returns a zero-terminated UTF-16 encoding of s.
func stringToUTF16(s string) []uint16 {
	return append(utf16.Encode([]rune(s)), 0)
}

func utf16ToString(s []uint16) string {
	return string(utf16.Decode(s))
}

var commonControlsInitialized bool

// initCommonControls registers the common control window classes.
func initCommonControls() {
	if commonControlsInitialized {
		return
	}
	commonControlsInitialized = true
	w32.InitCommonControlsEx(&w32.INITCOMMONCONTROLSEX{
		ICC: w32.ICC_DATE_CLASSES | w32.ICC_TAB_CLASSES | w32.ICC_BAR_CLASSES |
			w32.ICC_LISTVIEW_CLASSES | w32.ICC_PROGRESS_CLASS | w32.ICC_UPDOWN_CLASS |
			0x2 | 0x40 | 0x800 | 0x4000 | 0x8000, // tree view, hot key, internet (IP address), standard, link
	})
}

