package wui

import "github.com/2dprototype/wui/w32"

// ColorDialog is the system color chooser.
type ColorDialog struct {
	color  Color
	custom [16]uint32
}

func NewColorDialog() *ColorDialog {
	return &ColorDialog{}
}

// SetColor sets the initially selected color.
func (d *ColorDialog) SetColor(c Color) { d.color = c }

// Color returns the color chosen in the last successful Execute.
func (d *ColorDialog) Color() Color { return d.color }

// Execute shows the dialog (parent may be nil) and returns true if the user
// confirmed a color, which is then available through Color.
func (d *ColorDialog) Execute(parent *Window) bool {
	cc := w32.CHOOSECOLOR{
		RgbResult:  uint32(d.color),
		CustColors: &d.custom[0],
		Flags:      w32.CC_RGBINIT | w32.CC_FULLOPEN,
	}
	if parent != nil {
		cc.HwndOwner = parent.handle
	}
	if w32.ChooseColor(&cc) {
		d.color = Color(cc.RgbResult)
		return true
	}
	return false
}
