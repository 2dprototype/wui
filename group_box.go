package wui

import "github.com/2dprototype/wui/w32"

// NewGroupBox creates a framed box with a caption. A GroupBox is a visual
// grouping only: place other controls on top of it (add the GroupBox to its
// parent first, then the controls that should appear inside it).
func NewGroupBox() *GroupBox {
	return &GroupBox{}
}

type GroupBox struct {
	textControl
}

var _ Control = (*GroupBox)(nil)

func (*GroupBox) canFocus() bool { return false }

func (*GroupBox) eatsTabs() bool { return false }

func (g *GroupBox) create(id int) {
	g.textControl.create(id, 0, "BUTTON", w32.BS_GROUPBOX|w32.WS_CLIPSIBLINGS)
	g.themeForColor()
}

// SetTextColor changes the caption color. Visual styles are switched off for
// this group box because themed group boxes ignore the text color.
func (g *GroupBox) SetTextColor(col Color) {
	g.control.SetTextColor(col)
	g.themeForColor()
}
