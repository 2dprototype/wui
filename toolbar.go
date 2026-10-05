package wui

// NewToolBar creates a horizontal row of buttons inside a window or panel. It
// is a layout helper: every button is a normal Button placed next to the
// previous one.
//
//	tb := wui.NewToolBar(window, 0, 0, 28)
//	tb.AddButton("New", 60, func() { ... })
//	tb.AddSeparator()
//	tb.AddButton("Open", 60, func() { ... })
func NewToolBar(parent Container, x, y, height int) *ToolBar {
	return &ToolBar{parent: parent, x: x, y: y, nextX: x, height: height, gap: 2}
}

type ToolBar struct {
	parent  Container
	x, y    int
	nextX   int
	height  int
	gap     int
	buttons []*Button
}

// AddButton appends a button with the given text and width.
func (t *ToolBar) AddButton(text string, width int, onClick func()) *Button {
	b := NewButton()
	b.SetText(text)
	b.SetBounds(t.nextX, t.y, width, t.height)
	if onClick != nil {
		b.SetOnClick(onClick)
	}
	t.parent.Add(b)
	t.buttons = append(t.buttons, b)
	t.nextX += width + t.gap
	return b
}

// AddSeparator leaves a gap between button groups.
func (t *ToolBar) AddSeparator() {
	t.nextX += 10
}

// SetGap sets the space between buttons for the buttons added afterwards.
func (t *ToolBar) SetGap(gap int) { t.gap = gap }

// Buttons returns all buttons added so far.
func (t *ToolBar) Buttons() []*Button {
	return append([]*Button(nil), t.buttons...)
}

// Width returns the width used by the toolbar so far.
func (t *ToolBar) Width() int { return t.nextX - t.x }
