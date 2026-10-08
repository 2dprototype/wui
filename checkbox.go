package wui

import "github.com/2dprototype/wui/w32"

func NewCheckBox() *CheckBox {
	return &CheckBox{}
}

type CheckBox struct {
	textControl
	checked       bool
	onChange      func(bool)
	threeState    bool
	indeterminate bool
	pushLike      bool
}

var _ Control = (*CheckBox)(nil)

func (*CheckBox) canFocus() bool {
	return true
}

func (c *CheckBox) OnTabFocus() func() {
	return c.onTabFocus
}

func (c *CheckBox) SetOnTabFocus(f func()) {
	c.onTabFocus = f
}

func (*CheckBox) eatsTabs() bool {
	return false
}

func (c *CheckBox) create(id int) {
	c.textControl.create(id, 0, "BUTTON", w32.WS_TABSTOP|c.styleBits())
	w32.SendMessage(c.handle, w32.BM_SETCHECK, c.checkState(), 0)
	c.themeForColor()
}

// SetTextColor changes the text color. Visual styles are switched off for this
// check box because themed check boxes ignore the text color.
func (c *CheckBox) SetTextColor(col Color) {
	c.control.SetTextColor(col)
	c.themeForColor()
}

func (c *CheckBox) Checked() bool {
	return c.checked
}

func (c *CheckBox) SetChecked(checked bool) {
	if checked == c.checked {
		return
	}
	c.checked = checked
	if c.handle != 0 {
		w32.SendMessage(c.handle, w32.BM_SETCHECK, toCheckState(c.checked), 0)
	}
	if c.onChange != nil {
		c.onChange(c.checked)
	}
	return
}

func toCheckState(checked bool) uintptr {
	if checked {
		return w32.BST_CHECKED
	}
	return w32.BST_UNCHECKED
}

func (c *CheckBox) SetOnChange(f func(checked bool)) {
	c.onChange = f
}

func (c *CheckBox) handleNotification(cmd uintptr) {
	if cmd == w32.BN_CLICKED {
		st := w32.SendMessage(c.handle, w32.BM_GETCHECK, 0, 0)
		c.checked = st == w32.BST_CHECKED
		c.indeterminate = st == w32.BST_INDETERMINATE
		if c.onChange != nil {
			c.onChange(c.checked)
		}
	}
}
