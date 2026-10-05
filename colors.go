package wui

import (
	"syscall"
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

// colorState holds the optional custom colors of a control. It is embedded in
// the base control so every control has TextColor/BackgroundColor methods.
// They only have a visible effect on controls that are drawn by Windows with
// the WM_CTLCOLOR messages: Label, CheckBox, RadioButton, GroupBox, EditLine,
// TextEdit, ComboBox, StringList and the background of a Panel. Push buttons,
// sliders and progress bars ignore them, a PaintBox draws its own colors.
type colorState struct {
	textColor    Color
	bgColor      Color
	hasTextColor bool
	hasBgColor   bool
}

// TextColor returns the color set with SetTextColor. If no color was set it
// returns the system window text color.
func (c *control) TextColor() Color {
	if c.hasTextColor {
		return c.textColor
	}
	return ColorWindowText
}

// SetTextColor changes the color of the control's text.
func (c *control) SetTextColor(col Color) {
	c.textColor = col
	c.hasTextColor = true
	c.colorsChanged()
}

// ResetTextColor goes back to the system text color.
func (c *control) ResetTextColor() {
	c.textColor = 0
	c.hasTextColor = false
	c.colorsChanged()
}

// HasTextColor returns true if SetTextColor was called.
func (c *control) HasTextColor() bool {
	return c.hasTextColor
}

// BackgroundColor returns the color set with SetBackgroundColor. If no color
// was set it returns the system button face color.
func (c *control) BackgroundColor() Color {
	if c.hasBgColor {
		return c.bgColor
	}
	return ColorButtonFace
}

// SetBackgroundColor changes the color behind the control's text (or the fill
// color of a Panel).
func (c *control) SetBackgroundColor(col Color) {
	c.bgColor = col
	c.hasBgColor = true
	c.colorsChanged()
}

// ResetBackgroundColor goes back to the default background.
func (c *control) ResetBackgroundColor() {
	c.bgColor = 0
	c.hasBgColor = false
	c.colorsChanged()
}

// HasBackgroundColor returns true if SetBackgroundColor was called.
func (c *control) HasBackgroundColor() bool {
	return c.hasBgColor
}

func (c *control) colorSpec() (text Color, hasText bool, bg Color, hasBg bool) {
	return c.textColor, c.hasTextColor, c.bgColor, c.hasBgColor
}

func (c *control) colorsChanged() {
	if c.handle != 0 {
		w32.InvalidateRect(c.handle, nil, true)
		if c.parent != nil {
			w32.InvalidateRect(c.parent.getHandle(), nil, true)
		}
	}
}

var (
	extUxtheme            = syscall.NewLazyDLL("uxtheme.dll")
	procSetWindowTheme    = extUxtheme.NewProc("SetWindowTheme")
	emptyUTF16, _         = syscall.UTF16PtrFromString("")
	ctlColorBrushes       = make(map[Color]w32.HBRUSH)
)

// themeForColor switches off visual styles for a control whose text color is
// customized. Themed check boxes, radio buttons and group boxes ignore the text
// color, the classic look honors it.
func (c *control) themeForColor() {
	if c.handle != 0 && c.hasTextColor {
		procSetWindowTheme.Call(
			uintptr(c.handle),
			uintptr(unsafe.Pointer(emptyUTF16)),
			uintptr(unsafe.Pointer(emptyUTF16)),
		)
		w32.InvalidateRect(c.handle, nil, true)
	}
}

func brushFor(c Color) w32.HBRUSH {
	if b, ok := ctlColorBrushes[c]; ok {
		return b
	}
	b := w32.CreateSolidBrush(uint32(c))
	ctlColorBrushes[c] = b
	return b
}

// containerBackground returns the color a container paints behind its
// children.
func containerBackground(c Container) Color {
	switch p := c.(type) {
	case *Window:
		return p.background
	case *Panel:
		if p.hasBgColor {
			return p.bgColor
		}
	}
	return ColorButtonFace
}

// ctlColor answers the WM_CTLCOLOR* messages that a window or panel receives
// from its children. It returns false if the control has no custom colors, in
// which case the default processing must run.
func ctlColor(children []Control, parentBg Color, msg uint32, wParam, lParam uintptr) (uintptr, bool) {
	ctl := findControlByHandle(children, lParam)
	if ctl == nil {
		return 0, false
	}
	spec, ok := ctl.(interface {
		colorSpec() (Color, bool, Color, bool)
	})
	if !ok {
		return 0, false
	}
	text, hasText, bg, hasBg := spec.colorSpec()
	if !hasText && !hasBg {
		return 0, false
	}
	if !hasBg {
		if msg == w32.WM_CTLCOLOREDIT || msg == w32.WM_CTLCOLORLISTBOX {
			bg = ColorWindow
		} else {
			bg = parentBg
		}
	}
	if !hasText {
		text = ColorWindowText
	}
	hdc := w32.HDC(wParam)
	w32.SetTextColor(hdc, w32.COLORREF(text))
	w32.SetBkColor(hdc, w32.COLORREF(bg))
	return uintptr(brushFor(bg)), true
}

func isCtlColorMsg(msg uint32) bool {
	return msg == w32.WM_CTLCOLORSTATIC ||
		msg == w32.WM_CTLCOLOREDIT ||
		msg == w32.WM_CTLCOLORBTN ||
		msg == w32.WM_CTLCOLORLISTBOX
}
