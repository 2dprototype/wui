package wui

// Layout places the children of a Panel or Window. Set one with
// Panel.SetLayout or Window.SetLayout and the children are arranged again
// whenever the container is resized or children are added or removed. Anchors
// of the children are overridden by the layout.
//
// Arrange gets the children and the area (x, y, width, height) in the
// container's client coordinates. Children that are not Visible are skipped.
type Layout interface {
	Arrange(children []Control, x, y, width, height int)
}

// SetLayout makes the panel arrange its children with l.
func (p *Panel) SetLayout(l Layout) {
	p.layout = l
	p.applyLayout()
}

// Layout returns the panel's layout or nil.
func (p *Panel) Layout() Layout { return p.layout }

// Relayout arranges the children again, call it after changing the settings
// of a layout while the window is open.
func (p *Panel) Relayout() { p.applyLayout() }

func (p *Panel) applyLayout() {
	if p.layout == nil {
		return
	}
	_, _, w, h := p.InnerBounds()
	if w <= 0 || h <= 0 {
		return
	}
	p.layout.Arrange(p.children, 0, 0, w, h)
}

func visibleControls(children []Control) []Control {
	var out []Control
	for _, c := range children {
		if c.Visible() {
			out = append(out, c)
		}
	}
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// BoxLayout puts the children in a row (NewHBox) or a column (NewVBox).
//
// Along the row or column a child keeps its own size unless it has a weight:
// the space that is left over is shared between the weighted children in
// proportion to their weights. Across, every child is stretched to fill the
// available space unless SetKeepCrossSize is used for it.
type BoxLayout struct {
	Horizontal bool
	// Spacing is the gap between two children, Padding the gap between the
	// children and the edge of the container.
	Spacing, Padding int

	weights map[Control]float64
	keep    map[Control]bool
}

// NewHBox returns a layout that arranges children from left to right.
func NewHBox(spacing, padding int) *BoxLayout {
	return &BoxLayout{Horizontal: true, Spacing: spacing, Padding: padding}
}

// NewVBox returns a layout that arranges children from top to bottom.
func NewVBox(spacing, padding int) *BoxLayout {
	return &BoxLayout{Spacing: spacing, Padding: padding}
}

// SetWeight lets c grow with the free space, relative to the other weights.
// A weight of 0 (the default) keeps the size c has.
func (l *BoxLayout) SetWeight(c Control, weight float64) *BoxLayout {
	if l.weights == nil {
		l.weights = make(map[Control]float64)
	}
	l.weights[c] = weight
	return l
}

// SetKeepCrossSize keeps the size of c across the layout direction (height in
// a row, width in a column) instead of stretching it.
func (l *BoxLayout) SetKeepCrossSize(c Control, keep bool) *BoxLayout {
	if l.keep == nil {
		l.keep = make(map[Control]bool)
	}
	l.keep[c] = keep
	return l
}

// Arrange implements Layout.
func (l *BoxLayout) Arrange(children []Control, x, y, width, height int) {
	cs := visibleControls(children)
	if len(cs) == 0 {
		return
	}
	var mainAvail, crossAvail int
	if l.Horizontal {
		mainAvail, crossAvail = width-2*l.Padding, height-2*l.Padding
	} else {
		mainAvail, crossAvail = height-2*l.Padding, width-2*l.Padding
	}
	fixed := l.Spacing * (len(cs) - 1)
	total := 0.0
	for _, c := range cs {
		wt := l.weights[c]
		if wt > 0 {
			total += wt
			continue
		}
		_, _, cw, ch := c.Bounds()
		if l.Horizontal {
			fixed += cw
		} else {
			fixed += ch
		}
	}
	free := mainAvail - fixed
	if free < 0 {
		free = 0
	}
	pos := l.Padding
	left := free
	seenWeighted := 0.0
	for _, c := range cs {
		_, _, cw, ch := c.Bounds()
		size := ch
		if l.Horizontal {
			size = cw
		}
		if wt := l.weights[c]; wt > 0 {
			seenWeighted += wt
			if seenWeighted >= total {
				size = left // the last weighted child takes the rounding rest
			} else {
				size = int(float64(free) * wt / total)
				left -= size
			}
		}
		cross := crossAvail
		if l.keep[c] {
			if l.Horizontal {
				cross = ch
			} else {
				cross = cw
			}
		}
		if l.Horizontal {
			c.SetBounds(x+pos, y+l.Padding, size, cross)
		} else {
			c.SetBounds(x+l.Padding, y+pos, cross, size)
		}
		pos += size + l.Spacing
	}
}

// DockStyle says to which side of the free space a child is docked.
type DockStyle int

const (
	// DockNone leaves the child where it is.
	DockNone DockStyle = iota
	DockTop
	DockBottom
	DockLeft
	DockRight
	// DockFill takes everything that is left after the other children.
	DockFill
)

// DockLayout docks children to the sides of the container in the order they
// were added, like the Dock property of other GUI toolkits: a child docked to
// the top keeps its height and gets the full width of the space that is left,
// and so on. Add the DockFill child last.
type DockLayout struct {
	Spacing, Padding int
	styles           map[Control]DockStyle
}

// NewDockLayout returns an empty dock layout.
func NewDockLayout(spacing, padding int) *DockLayout {
	return &DockLayout{Spacing: spacing, Padding: padding}
}

// Set chooses how c is docked.
func (l *DockLayout) Set(c Control, d DockStyle) *DockLayout {
	if l.styles == nil {
		l.styles = make(map[Control]DockStyle)
	}
	l.styles[c] = d
	return l
}

// Arrange implements Layout.
func (l *DockLayout) Arrange(children []Control, x, y, width, height int) {
	left, top := x+l.Padding, y+l.Padding
	right, bottom := x+width-l.Padding, y+height-l.Padding
	for _, c := range visibleControls(children) {
		_, _, cw, ch := c.Bounds()
		w, h := right-left, bottom-top
		switch l.styles[c] {
		case DockTop:
			c.SetBounds(left, top, w, ch)
			top += ch + l.Spacing
		case DockBottom:
			c.SetBounds(left, bottom-ch, w, ch)
			bottom -= ch + l.Spacing
		case DockLeft:
			c.SetBounds(left, top, cw, h)
			left += cw + l.Spacing
		case DockRight:
			c.SetBounds(right-cw, top, cw, h)
			right -= cw + l.Spacing
		case DockFill:
			c.SetBounds(left, top, maxInt(w, 0), maxInt(h, 0))
		}
	}
}

// GridLayout puts the children into the cells of a grid, row by row. The
// number of columns is fixed, rows are added as needed. Columns and rows are
// equally sized unless weights are set.
type GridLayout struct {
	Columns          int
	Spacing, Padding int

	colWeights []float64
	rowWeights []float64
	spans      map[Control][2]int
}

// NewGridLayout returns a grid with the given number of columns.
func NewGridLayout(columns, spacing, padding int) *GridLayout {
	if columns < 1 {
		columns = 1
	}
	return &GridLayout{Columns: columns, Spacing: spacing, Padding: padding}
}

// SetColumnWeights sets the relative widths of the columns. Missing entries
// count as 1.
func (l *GridLayout) SetColumnWeights(w ...float64) *GridLayout {
	l.colWeights = w
	return l
}

// SetRowWeights sets the relative heights of the rows. Missing entries count
// as 1.
func (l *GridLayout) SetRowWeights(w ...float64) *GridLayout {
	l.rowWeights = w
	return l
}

// SetSpan lets c cover several columns and rows.
func (l *GridLayout) SetSpan(c Control, columns, rows int) *GridLayout {
	if l.spans == nil {
		l.spans = make(map[Control][2]int)
	}
	if columns < 1 {
		columns = 1
	}
	if rows < 1 {
		rows = 1
	}
	l.spans[c] = [2]int{columns, rows}
	return l
}

type gridCell struct {
	c                   Control
	col, row, cols, rws int
}

func weightAt(w []float64, i int) float64 {
	if i < len(w) && w[i] > 0 {
		return w[i]
	}
	return 1
}

// split cuts total into n parts with the given weights, the parts add up to
// exactly total.
func split(total, n int, weight func(int) float64) []int {
	sum := 0.0
	for i := 0; i < n; i++ {
		sum += weight(i)
	}
	parts := make([]int, n)
	used := 0
	acc := 0.0
	for i := 0; i < n; i++ {
		acc += weight(i)
		end := int(float64(total)*acc/sum + 0.5)
		if i == n-1 {
			end = total
		}
		parts[i] = end - used
		used = end
	}
	return parts
}

// Arrange implements Layout.
func (l *GridLayout) Arrange(children []Control, x, y, width, height int) {
	cs := visibleControls(children)
	if len(cs) == 0 {
		return
	}
	occupied := make(map[[2]int]bool)
	var cells []gridCell
	rows := 0
	col, row := 0, 0
	for _, c := range cs {
		span := l.spans[c]
		cols, rws := span[0], span[1]
		if cols == 0 {
			cols, rws = 1, 1
		}
		if cols > l.Columns {
			cols = l.Columns
		}
		for {
			if col+cols > l.Columns {
				col = 0
				row++
			}
			free := true
			for dc := 0; dc < cols && free; dc++ {
				for dr := 0; dr < rws; dr++ {
					if occupied[[2]int{col + dc, row + dr}] {
						free = false
						break
					}
				}
			}
			if free {
				break
			}
			col++
		}
		for dc := 0; dc < cols; dc++ {
			for dr := 0; dr < rws; dr++ {
				occupied[[2]int{col + dc, row + dr}] = true
			}
		}
		cells = append(cells, gridCell{c, col, row, cols, rws})
		if row+rws > rows {
			rows = row + rws
		}
		col += cols
	}
	availW := width - 2*l.Padding - l.Spacing*(l.Columns-1)
	availH := height - 2*l.Padding - l.Spacing*(rows-1)
	colW := split(maxInt(availW, 0), l.Columns, func(i int) float64 { return weightAt(l.colWeights, i) })
	rowH := split(maxInt(availH, 0), rows, func(i int) float64 { return weightAt(l.rowWeights, i) })
	colX := make([]int, l.Columns+1)
	for i := 0; i < l.Columns; i++ {
		colX[i+1] = colX[i] + colW[i] + l.Spacing
	}
	rowY := make([]int, rows+1)
	for i := 0; i < rows; i++ {
		rowY[i+1] = rowY[i] + rowH[i] + l.Spacing
	}
	for _, cell := range cells {
		cw := colX[cell.col+cell.cols] - l.Spacing - colX[cell.col]
		ch := rowY[cell.row+cell.rws] - l.Spacing - rowY[cell.row]
		cell.c.SetBounds(x+l.Padding+colX[cell.col], y+l.Padding+rowY[cell.row], cw, ch)
	}
}
