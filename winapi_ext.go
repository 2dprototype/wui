package wui

import (
	"syscall"
	"unsafe"
)

// Lazily loaded Win32 functions that the w32 package does not wrap. All names
// start with nx so they cannot clash with the older ext... and proc... names.
var (
	nxUser32   = syscall.NewLazyDLL("user32.dll")
	nxKernel32 = syscall.NewLazyDLL("kernel32.dll")
	nxShell32  = syscall.NewLazyDLL("shell32.dll")
	nxOle32    = syscall.NewLazyDLL("ole32.dll")
	nxDwmapi   = syscall.NewLazyDLL("dwmapi.dll")
	nxComctl32 = syscall.NewLazyDLL("comctl32.dll")
	nxComdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	nxGdi32    = syscall.NewLazyDLL("gdi32.dll")

	nxGetCurrentThreadID            = nxKernel32.NewProc("GetCurrentThreadId")
	nxCreateMutex                   = nxKernel32.NewProc("CreateMutexW")
	nxLoadLibrary                   = nxKernel32.NewProc("LoadLibraryW")
	nxRegisterHotKey                = nxUser32.NewProc("RegisterHotKey")
	nxUnregisterHotKey              = nxUser32.NewProc("UnregisterHotKey")
	nxFlashWindowEx                 = nxUser32.NewProc("FlashWindowEx")
	nxGetDpiForWindow               = nxUser32.NewProc("GetDpiForWindow")
	nxSetProcessDPIAware            = nxUser32.NewProc("SetProcessDPIAware")
	nxSetProcessDpiAwarenessContext = nxUser32.NewProc("SetProcessDpiAwarenessContext")
	nxSetScrollInfo                 = nxUser32.NewProc("SetScrollInfo")
	nxGetScrollInfo                 = nxUser32.NewProc("GetScrollInfo")
	nxScrollWindowEx                = nxUser32.NewProc("ScrollWindowEx")
	nxSetParent                     = nxUser32.NewProc("SetParent")
	nxDwmSetWindowAttribute         = nxDwmapi.NewProc("DwmSetWindowAttribute")
	nxShellExecute                  = nxShell32.NewProc("ShellExecuteW")
	nxCoInitializeEx                = nxOle32.NewProc("CoInitializeEx")
	nxCoCreateInstance              = nxOle32.NewProc("CoCreateInstance")
	nxImageListCreate               = nxComctl32.NewProc("ImageList_Create")
	nxImageListDestroy              = nxComctl32.NewProc("ImageList_Destroy")
	nxImageListReplaceIcon          = nxComctl32.NewProc("ImageList_ReplaceIcon")
	nxImageListGetImageCount        = nxComctl32.NewProc("ImageList_GetImageCount")
	nxTaskDialog                    = nxComctl32.NewProc("TaskDialog")
	nxChooseFont                    = nxComdlg32.NewProc("ChooseFontW")
	nxGetStockObject                = nxGdi32.NewProc("GetStockObject")
	nxGlobalSize                    = nxKernel32.NewProc("GlobalSize")
)

func boolToUintptr(b bool) uintptr {
	if b {
		return 1
	}
	return 0
}

// utf16Ptr returns a pointer to the zero terminated UTF-16 form of s.
func utf16Ptr(s string) *uint16 {
	return &stringToUTF16(s)[0]
}

// ptrOf converts a pointer to the uintptr that Win32 calls expect.
func ptrOf(p unsafe.Pointer) uintptr {
	return uintptr(p)
}

// loword and hiword split a 32 bit value, loInt and hiInt sign extend them,
// which is what mouse coordinates need.
func loword(v uintptr) int { return int(v & 0xFFFF) }
func hiword(v uintptr) int { return int((v >> 16) & 0xFFFF) }
func loInt(v uintptr) int  { return int(int16(v & 0xFFFF)) }
func hiInt(v uintptr) int  { return int(int16((v >> 16) & 0xFFFF)) }

func currentThreadID() uint32 {
	r, _, _ := nxGetCurrentThreadID.Call()
	return uint32(r)
}
