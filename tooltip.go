package wui

import (
	"syscall"
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

const (
	ttmAddToolW        = 0x0432 // WM_USER + 50
	ttmUpdateTipTextW  = 0x0439 // WM_USER + 57
	ttmDelToolW        = 0x0433 // WM_USER + 51
	toolTipsClassName  = "tooltips_class32"
)

// SetToolTip sets the hint text shown when the mouse rests over the control.
// An empty text removes the tool tip. It works for every control.
func (c *control) SetToolTip(text string) {
	c.toolTip = text
	c.applyToolTip()
}

// ToolTip returns the hint text of the control.
func (c *control) ToolTip() string {
	return c.toolTip
}

func (c *control) toolInfo() w32.TOOLINFO {
	ti := w32.TOOLINFO{
		UFlags:   w32.TTF_IDISHWND | w32.TTF_SUBCLASS,
		Hwnd:     c.parent.getHandle(),
		UId:      uintptr(c.handle),
		LpszText: syscall.StringToUTF16Ptr(c.toolTip),
	}
	ti.CbSize = uint32(unsafe.Sizeof(ti))
	return ti
}

func (c *control) applyToolTip() {
	if c.handle == 0 || c.parent == nil {
		return
	}
	if c.toolTip == "" {
		if c.toolTipWnd != 0 {
			ti := c.toolInfo()
			w32.SendMessage(c.toolTipWnd, ttmDelToolW, 0, uintptr(unsafe.Pointer(&ti)))
			w32.DestroyWindow(c.toolTipWnd)
			c.toolTipWnd = 0
		}
		return
	}
	ti := c.toolInfo()
	if c.toolTipWnd == 0 {
		c.toolTipWnd = w32.CreateWindowExStr(
			w32.WS_EX_TOPMOST,
			toolTipsClassName,
			"",
			w32.WS_POPUP|w32.TTS_NOPREFIX|w32.TTS_ALWAYSTIP,
			w32.CW_USEDEFAULT, w32.CW_USEDEFAULT, w32.CW_USEDEFAULT, w32.CW_USEDEFAULT,
			c.parent.getHandle(), 0, c.parent.getInstance(), nil,
		)
		if c.toolTipWnd == 0 {
			return
		}
		w32.SendMessage(c.toolTipWnd, ttmAddToolW, 0, uintptr(unsafe.Pointer(&ti)))
	} else {
		w32.SendMessage(c.toolTipWnd, ttmUpdateTipTextW, 0, uintptr(unsafe.Pointer(&ti)))
	}
}
