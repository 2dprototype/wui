package wui

import (
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

// EnableDPIAwareness tells Windows that the program scales itself on high DPI
// monitors, so the window is no longer blurry-stretched. Call it once at the
// start of main, before the first window is created. It uses per monitor v2
// awareness where available (Windows 10 1703+) and falls back to system DPI
// awareness. It returns false if nothing could be set, e.g. because the
// awareness was already fixed by a manifest.
//
// With DPI awareness on, sizes you give are real pixels. Use Window.Scale to
// turn design sizes (made for 96 DPI) into pixels, and SetOnDPIChanged to
// react when the window moves to another monitor.
func EnableDPIAwareness() bool {
	if nxSetProcessDpiAwarenessContext.Find() == nil {
		// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 is the handle -4.
		if r, _, _ := nxSetProcessDpiAwarenessContext.Call(^uintptr(3)); r != 0 {
			return true
		}
	}
	if nxSetProcessDPIAware.Find() == nil {
		if r, _, _ := nxSetProcessDPIAware.Call(); r != 0 {
			return true
		}
	}
	return false
}

// DPI returns the dots per inch of the monitor the window is on, 96 means 100%
// scaling. Without DPI awareness it is the system value.
func (w *Window) DPI() int {
	if w.handle != 0 && nxGetDpiForWindow.Find() == nil {
		if r, _, _ := nxGetDpiForWindow.Call(uintptr(w.handle)); r != 0 {
			return int(r)
		}
	}
	hdc := w32.GetDC(0)
	defer w32.ReleaseDC(0, hdc)
	if dpi := w32.GetDeviceCaps(hdc, w32.LOGPIXELSX); dpi > 0 {
		return dpi
	}
	return 96
}

// ScaleFactor returns DPI()/96, e.g. 1.5 for 150% scaling.
func (w *Window) ScaleFactor() float64 {
	return float64(w.DPI()) / 96
}

// Scale converts a size designed for 96 DPI to pixels on this window's
// monitor.
func (w *Window) Scale(px int) int {
	return ScaleForDPI(px, w.DPI())
}

// ScaleForDPI converts a size designed for 96 DPI to pixels at the given DPI.
func ScaleForDPI(px, dpi int) int {
	return (px*dpi + 48) / 96
}

// DarkTheme is the name of the control theme that gives scroll bars, lists,
// trees, edit boxes and similar controls a dark look on Windows 10 and 11.
const DarkTheme = "DarkMode_Explorer"

// SetDarkTitleBar switches the window's title bar between the light and the
// dark look (Windows 10 1809 and newer, older versions ignore it).
func (w *Window) SetDarkTitleBar(dark bool) {
	if w.handle == 0 || nxDwmSetWindowAttribute.Find() != nil {
		return
	}
	v := int32(0)
	if dark {
		v = 1
	}
	// 20 is DWMWA_USE_IMMERSIVE_DARK_MODE, 19 was used before Windows 10 20H1.
	for _, attr := range []uintptr{20, 19} {
		r, _, _ := nxDwmSetWindowAttribute.Call(
			uintptr(w.handle), attr, uintptr(unsafe.Pointer(&v)), unsafe.Sizeof(v),
		)
		if r == 0 {
			break
		}
	}
}

// SetControlTheme gives one control a visual style by name, e.g. DarkTheme or
// "Explorer". An empty name switches visual styles off for the control. It
// does nothing before the window is shown.
func SetControlTheme(c Control, theme string) {
	h := c.Handle()
	if h == 0 {
		return
	}
	var name uintptr
	if theme == "" {
		name = uintptr(unsafe.Pointer(emptyUTF16))
	} else {
		name = uintptr(unsafe.Pointer(utf16Ptr(theme)))
	}
	procSetWindowTheme.Call(h, name, 0)
	w32.InvalidateRect(w32.HWND(h), nil, true)
}

// DarkPalette are the colors ApplyDarkMode uses.
type DarkPalette struct {
	Background Color // window and panels
	Text       Color
	EditBack   Color // edit boxes
}

// DefaultDarkPalette is a neutral dark gray palette.
var DefaultDarkPalette = DarkPalette{
	Background: RGB(32, 32, 32),
	Text:       RGB(235, 235, 235),
	EditBack:   RGB(45, 45, 45),
}

type textColorSetter interface{ SetTextColor(Color) }
type backgroundColorSetter interface{ SetBackgroundColor(Color) }

// ApplyDarkMode gives the window and all controls in it a dark look: dark
// title bar, dark control themes, and dark background and light text colors
// where a control supports them. Call it after the window was created, e.g.
// in SetOnShow. Not every system control can be darkened completely, the
// built in check boxes and radio buttons lose their visual style when they get
// a text color.
func (w *Window) ApplyDarkMode(p DarkPalette) {
	w.SetDarkTitleBar(true)
	w.SetBackgroundColor(p.Background)
	darkenControls(w.children, p)
	w32.InvalidateRect(w.handle, nil, true)
}

func darkenControls(children []Control, p DarkPalette) {
	for _, c := range children {
		SetControlTheme(c, DarkTheme)
		if s, ok := c.(textColorSetter); ok {
			s.SetTextColor(p.Text)
		}
		if s, ok := c.(backgroundColorSetter); ok {
			switch c.(type) {
			case *EditLine, *TextEdit, *ComboBox:
				s.SetBackgroundColor(p.EditBack)
			default:
				s.SetBackgroundColor(p.Background)
			}
		}
		if cont, ok := c.(Container); ok {
			darkenControls(cont.Children(), p)
		}
	}
}
