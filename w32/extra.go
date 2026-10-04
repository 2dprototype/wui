package w32

import (
	"unsafe"
)

// Additions used by the wui TabControl, StatusBar and Timer.

var killTimer = user32.NewProc("KillTimer")

// KillTimer destroys a timer created with SetTimer.
func KillTimer(window HWND, idEvent uintptr) bool {
	ret, _, _ := killTimer.Call(uintptr(window), idEvent)
	return ret != 0
}

// TCITEM is the Unicode tab control item structure (TCITEMW).
type TCITEM struct {
	Mask        uint32
	DwState     uint32
	DwStateMask uint32
	PszText     *uint16
	CchTextMax  int32
	IImage      int32
	LParam      uintptr
}

const (
	TCIF_TEXT        = 0x0001
	TCM_INSERTITEMW  = 0x133E
	TCM_SETITEMW     = 0x133D
	TCN_SELCHANGE    = 0xFFFFFDD9 // -551 as uint32
	TAB_CLASS        = "SysTabControl32"
	STATUS_CLASS     = "msctls_statusbar32"
	SBARS_SIZEGRIP   = 0x0100
	SB_SETTEXTW_MSG  = 0x040B
	SB_SETPARTS_MSG  = 0x0404
	// SB_GETTEXTLENGTHW = 0x040C
)

// SendMessagePtr sends a message whose lParam is a pointer.
func SendMessagePtr(hwnd HWND, msg uint32, wParam uintptr, lParam unsafe.Pointer) uintptr {
	return SendMessage(hwnd, msg, wParam, uintptr(lParam))
}

// Color dialog.

var chooseColor = comdlg32.NewProc("ChooseColorW")

const (
	CC_RGBINIT  = 0x00000001
	CC_FULLOPEN = 0x00000002
)

// CHOOSECOLOR is the CHOOSECOLORW structure.
type CHOOSECOLOR struct {
	StructSize    uint32
	HwndOwner     HWND
	HInstance     HWND
	RgbResult     uint32
	CustColors    *uint32
	Flags         uint32
	LCustData     uintptr
	LpfnHook      uintptr
	LpTemplateName *uint16
}

// ChooseColor shows the system color dialog.
func ChooseColor(cc *CHOOSECOLOR) bool {
	cc.StructSize = uint32(unsafe.Sizeof(*cc))
	ret, _, _ := chooseColor.Call(uintptr(unsafe.Pointer(cc)))
	return ret != 0
}

// Date time picker.

const (
	DATETIMEPICK_CLASS  = "SysDateTimePick32"
	DTS_UPDOWN          = 0x0001
	DTS_SHORTDATEFORMAT = 0x0000
	DTS_LONGDATEFORMAT  = 0x0004
	DTS_TIMEFORMAT      = 0x0009
	DTN_DATETIMECHANGE  = 0xFFFFFD02 // -766 as uint32
	GDT_VALID           = 0
	DTM_GETSYSTEMTIME_MSG = 0x1001
	DTM_SETSYSTEMTIME_MSG = 0x1002
)
