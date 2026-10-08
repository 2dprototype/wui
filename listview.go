package wui

import (
	"sort"
	"strconv"
	"strings"
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

// ListView is the full featured list control: several columns with headers,
// small or large icons, check boxes, multiple selection, sorting by clicking
// the column headers, editable labels and a virtual mode for huge data sets.
// It replaces StringTable where you need more than plain text rows.
//
//	lv := wui.NewListView()
//	lv.AddColumn("Name", 200)
//	lv.AddColumn("Size", 80)
//	lv.AddItem("readme.txt", "1 KB")
type ListView struct {
	textControl
	id      int
	columns []*listColumn
	items   []*ListViewItem

	view         ListViewView
	multiSelect  bool
	checkBoxes   bool
	fullRow      bool
	fullRowSet   bool
	gridLines    bool
	noHeader     bool
	editable     bool
	sortable     bool
	sortCol      int
	sortAsc      bool
	smallImages  *ImageList
	largeImages  *ImageList
	compare      func(a, b string) int
	populating   bool
	virtual      bool
	virtualCount int
	virtualText  func(row, col int) string
	virtualImage func(row int) int

	onSelect      func(index int)
	onActivate    func(index int)
	onColumnClick func(column int)
	onItemCheck   func(item *ListViewItem, checked bool)
	onLabelEdit   func(item *ListViewItem, newText string) bool
}

var _ Control = (*ListView)(nil)

type listColumn struct {
	title string
	width int
	align ColumnAlign
}

// ColumnAlign is the text alignment of a ListView column.
type ColumnAlign int

const (
	ColumnLeft ColumnAlign = iota
	ColumnRight
	ColumnCenter
)

// ListViewView is how a ListView shows its items.
type ListViewView int

const (
	// ListViewDetails shows columns with headers (the default).
	ListViewDetails ListViewView = iota
	// ListViewList shows small icons with text in columns that flow.
	ListViewList
	// ListViewIcons shows large icons with text below.
	ListViewIcons
	// ListViewSmallIcons shows small icons with text on the right.
	ListViewSmallIcons
	// ListViewTiles shows large icons with several text lines on the right.
	ListViewTiles
)

// ListViewItem is one row of a ListView. Tag can hold anything you like.
type ListViewItem struct {
	Tag     interface{}
	lv      *ListView
	cells   []string
	image   int
	checked bool
	index   int
}

// NewListView creates an empty list view in details mode.
func NewListView() *ListView {
	return &ListView{fullRow: true, sortAsc: true, sortCol: -1}
}

func (*ListView) canFocus() bool  { return true }
func (*ListView) eatsTabs() bool  { return false }
func (l *ListView) OnTabFocus() func() { return l.onTabFocus }
func (l *ListView) SetOnTabFocus(f func()) { l.onTabFocus = f }

const (
	lvmSetImageList      = 0x1003
	lvmGetItemCount      = 0x1004
	lvmDeleteItem        = 0x1008
	lvmDeleteAllItems    = 0x1009
	lvmGetNextItem       = 0x100C
	lvmEnsureVisible     = 0x1013
	lvmSetColumnWidth    = 0x101E
	lvmSetItemCount      = 0x102F
	lvmSetItemState      = 0x102B
	lvmGetItemState      = 0x102C
	lvmGetSelectedCount  = 0x1032
	lvmSetExtStyle       = 0x1036
	lvmSetItemW          = 0x104C
	lvmInsertItemW       = 0x104D
	lvmSetColumnW        = 0x1060
	lvmInsertColumnW     = 0x1061
	lvmSetItemTextW      = 0x1074
	lvmEditLabelW        = 0x1076
	lvmSetView           = 0x108E

	lvsReport         = 0x0001
	lvsSingleSel      = 0x0004
	lvsShowSelAlways  = 0x0008
	lvsEditLabels     = 0x0200
	lvsShareImgLists  = 0x0040
	lvsNoColumnHeader = 0x4000
	lvsOwnerData      = 0x1000

	lvsExGridLines     = 0x00000001
	lvsExCheckBoxes    = 0x00000004
	lvsExHeaderDrag    = 0x00000010
	lvsExFullRowSelect = 0x00000020
	lvsExLabelTip      = 0x00004000
	lvsExDoubleBuffer  = 0x00010000

	lvifText  = 0x0001
	lvifImage = 0x0002
	lvifParam = 0x0004
	lvifState = 0x0008

	lvisFocused   = 0x0001
	lvisSelected  = 0x0002
	lvisStateMask = 0xF000

	lvniSelected = 0x0002

	lvcfFmt     = 0x0001
	lvcfWidth   = 0x0002
	lvcfText    = 0x0004
	lvcfSubItem = 0x0008

	lvnItemChanged   uint32 = 0xFFFFFF9B
	lvnColumnClick   uint32 = 0xFFFFFF94
	lvnItemActivate  uint32 = 0xFFFFFF8E
	lvnGetDispInfoW  uint32 = 0xFFFFFF4F
	lvnEndLabelEditW uint32 = 0xFFFFFF50
)

type lvItem struct {
	mask       uint32
	iItem      int32
	iSubItem   int32
	state      uint32
	stateMask  uint32
	pszText    *uint16
	cchTextMax int32
	iImage     int32
	lParam     uintptr
	iIndent    int32
	iGroupID   int32
	cColumns   uint32
	puColumns  *uint32
	piColFmt   *int32
	iGroup     int32
}

type lvColumnW struct {
	mask       uint32
	fmt        int32
	cx         int32
	pszText    *uint16
	cchTextMax int32
	iSubItem   int32
	iImage     int32
	iOrder     int32
	cxMin      int32
	cxDefault  int32
	cxIdeal    int32
}

type nmLVDispInfo struct {
	hdr  w32.NMHDR
	item lvItem
}

func (l *ListView) create(id int) {
	initCommonControls()
	l.id = id
	l.recreate()
}

func (l *ListView) recreate() {
	if l.handle != 0 {
		w32.DestroyWindow(l.handle)
		l.handle = 0
	}
	var style uint = w32.WS_TABSTOP | lvsReport | lvsShowSelAlways | lvsShareImgLists
	if !l.multiSelect {
		style |= lvsSingleSel
	}
	if l.noHeader {
		style |= lvsNoColumnHeader
	}
	if l.editable {
		style |= lvsEditLabels
	}
	if l.virtual {
		style |= lvsOwnerData
	}
	l.textControl.create(l.id, w32.WS_EX_CLIENTEDGE, "SysListView32", style)
	l.applyExtStyle()
	if l.smallImages != nil {
		w32.SendMessage(l.handle, lvmSetImageList, 1, l.smallImages.handle)
	}
	if l.largeImages != nil {
		w32.SendMessage(l.handle, lvmSetImageList, 0, l.largeImages.handle)
	}
	for i, c := range l.columns {
		l.insertColumn(i, c)
	}
	if l.view != ListViewDetails {
		w32.SendMessage(l.handle, lvmSetView, lvViewCode(l.view), 0)
	}
	l.populate()
}

func (l *ListView) applyExtStyle() {
	if l.handle == 0 {
		return
	}
	ex := uintptr(lvsExDoubleBuffer | lvsExLabelTip | lvsExHeaderDrag)
	if l.fullRow {
		ex |= lvsExFullRowSelect
	}
	if l.gridLines {
		ex |= lvsExGridLines
	}
	if l.checkBoxes && !l.virtual {
		ex |= lvsExCheckBoxes
	}
	all := uintptr(lvsExDoubleBuffer | lvsExLabelTip | lvsExHeaderDrag |
		lvsExFullRowSelect | lvsExGridLines | lvsExCheckBoxes)
	w32.SendMessage(l.handle, lvmSetExtStyle, all, ex)
}

func (l *ListView) insertColumn(i int, c *listColumn) {
	col := lvColumnW{
		mask:     lvcfFmt | lvcfWidth | lvcfText | lvcfSubItem,
		fmt:      int32(c.align),
		cx:       int32(c.width),
		pszText:  utf16Ptr(c.title),
		iSubItem: int32(i),
	}
	w32.SendMessage(l.handle, lvmInsertColumnW, uintptr(i), uintptr(unsafe.Pointer(&col)))
}

// populate fills the native control from the Go side item list.
func (l *ListView) populate() {
	if l.handle == 0 {
		return
	}
	l.populating = true
	defer func() { l.populating = false }()
	w32.SendMessage(l.handle, lvmDeleteAllItems, 0, 0)
	if l.virtual {
		w32.SendMessage(l.handle, lvmSetItemCount, uintptr(l.virtualCount), 0)
		return
	}
	for i, it := range l.items {
		it.index = i
		l.insertItem(it)
	}
}

func (l *ListView) insertItem(it *ListViewItem) {
	text := ""
	if len(it.cells) > 0 {
		text = it.cells[0]
	}
	lvi := lvItem{
		mask:     lvifText | lvifImage | lvifParam,
		iItem:    int32(it.index),
		pszText:  utf16Ptr(text),
		iImage:   int32(it.image),
		lParam:   uintptr(it.index),
	}
	if l.checkBoxes {
		lvi.mask |= lvifState
		lvi.stateMask = lvisStateMask
		if it.checked {
			lvi.state = 0x2000
		} else {
			lvi.state = 0x1000
		}
	}
	w32.SendMessage(l.handle, lvmInsertItemW, 0, uintptr(unsafe.Pointer(&lvi)))
	for c := 1; c < len(it.cells); c++ {
		l.setCellText(it.index, c, it.cells[c])
	}
}

func (l *ListView) setCellText(row, col int, text string) {
	if l.handle == 0 || l.virtual {
		return
	}
	lvi := lvItem{iSubItem: int32(col), pszText: utf16Ptr(text)}
	w32.SendMessage(l.handle, lvmSetItemTextW, uintptr(row), uintptr(unsafe.Pointer(&lvi)))
}

func (l *ListView) reindex() {
	for i, it := range l.items {
		it.index = i
	}
}

// AddColumn appends a column with a header and a width in pixels and returns
// its index.
func (l *ListView) AddColumn(title string, width int) int {
	c := &listColumn{title: title, width: width}
	l.columns = append(l.columns, c)
	if l.handle != 0 {
		l.insertColumn(len(l.columns)-1, c)
	}
	return len(l.columns) - 1
}

// SetColumnTitle changes the header text of a column.
func (l *ListView) SetColumnTitle(col int, title string) {
	if col < 0 || col >= len(l.columns) {
		return
	}
	l.columns[col].title = title
	if l.handle != 0 {
		c := lvColumnW{mask: lvcfText, pszText: utf16Ptr(title)}
		w32.SendMessage(l.handle, lvmSetColumnW, uintptr(col), uintptr(unsafe.Pointer(&c)))
	}
}

// SetColumnAlignment aligns the text of a column. The first column is always
// left aligned by Windows.
func (l *ListView) SetColumnAlignment(col int, a ColumnAlign) {
	if col < 0 || col >= len(l.columns) {
		return
	}
	l.columns[col].align = a
	if l.handle != 0 {
		c := lvColumnW{mask: lvcfFmt, fmt: int32(a)}
		w32.SendMessage(l.handle, lvmSetColumnW, uintptr(col), uintptr(unsafe.Pointer(&c)))
	}
}

// SetColumnWidth sets the width of a column in pixels.
func (l *ListView) SetColumnWidth(col, width int) {
	if col < 0 || col >= len(l.columns) {
		return
	}
	l.columns[col].width = width
	if l.handle != 0 {
		w32.SendMessage(l.handle, lvmSetColumnWidth, uintptr(col), uintptr(width))
	}
}

// AutoSizeColumns sizes every column to fit its widest cell, or its header
// when there are no items.
func (l *ListView) AutoSizeColumns() {
	if l.handle == 0 {
		return
	}
	for i := range l.columns {
		mode := ^uintptr(0) // LVSCW_AUTOSIZE
		if len(l.items) == 0 {
			mode = ^uintptr(1) // LVSCW_AUTOSIZE_USEHEADER
		}
		w32.SendMessage(l.handle, lvmSetColumnWidth, uintptr(i), mode)
	}
}

// ColumnCount returns the number of columns.
func (l *ListView) ColumnCount() int { return len(l.columns) }

// AddItem appends a row. The cells are the texts of the columns from left to
// right.
func (l *ListView) AddItem(cells ...string) *ListViewItem {
	it := &ListViewItem{lv: l, cells: append([]string(nil), cells...), index: len(l.items)}
	l.items = append(l.items, it)
	if l.handle != 0 && !l.virtual {
		l.populating = true
		l.insertItem(it)
		l.populating = false
	}
	return it
}

// Item returns the row at index or nil.
func (l *ListView) Item(index int) *ListViewItem {
	if index < 0 || index >= len(l.items) {
		return nil
	}
	return l.items[index]
}

// Items returns all rows in their current order.
func (l *ListView) Items() []*ListViewItem {
	return append([]*ListViewItem(nil), l.items...)
}

// Count returns the number of rows.
func (l *ListView) Count() int {
	if l.virtual {
		return l.virtualCount
	}
	return len(l.items)
}

// RemoveItem deletes the row at index.
func (l *ListView) RemoveItem(index int) {
	if index < 0 || index >= len(l.items) {
		return
	}
	l.items[index].lv = nil
	l.items = append(l.items[:index], l.items[index+1:]...)
	l.reindex()
	if l.handle != 0 {
		w32.SendMessage(l.handle, lvmDeleteItem, uintptr(index), 0)
	}
}

// Clear removes all rows.
func (l *ListView) Clear() {
	for _, it := range l.items {
		it.lv = nil
	}
	l.items = nil
	if l.virtual {
		l.virtualCount = 0
	}
	if l.handle != 0 {
		l.populating = true
		w32.SendMessage(l.handle, lvmDeleteAllItems, 0, 0)
		l.populating = false
	}
}

// Index returns the current position of the row, or -1 if it was removed.
func (it *ListViewItem) Index() int {
	if it.lv == nil {
		return -1
	}
	return it.index
}

// Cell returns the text in a column of this row.
func (it *ListViewItem) Cell(col int) string {
	if col < 0 || col >= len(it.cells) {
		return ""
	}
	return it.cells[col]
}

// SetCell changes the text in a column of this row.
func (it *ListViewItem) SetCell(col int, text string) {
	if col < 0 {
		return
	}
	for len(it.cells) <= col {
		it.cells = append(it.cells, "")
	}
	it.cells[col] = text
	if it.lv != nil {
		it.lv.setCellText(it.index, col, text)
	}
}

// Image returns the image list index shown for this row.
func (it *ListViewItem) Image() int { return it.image }

// SetImage chooses the image (an index into the ListView's image list).
func (it *ListViewItem) SetImage(index int) {
	it.image = index
	if it.lv != nil && it.lv.handle != 0 && !it.lv.virtual {
		lvi := lvItem{mask: lvifImage, iItem: int32(it.index), iImage: int32(index)}
		w32.SendMessage(it.lv.handle, lvmSetItemW, 0, uintptr(unsafe.Pointer(&lvi)))
	}
}

// Checked tells whether the row's check box is ticked. It needs
// SetCheckBoxes(true).
func (it *ListViewItem) Checked() bool { return it.checked }

// SetChecked ticks or clears the row's check box.
func (it *ListViewItem) SetChecked(checked bool) {
	it.checked = checked
	if it.lv != nil && it.lv.handle != 0 && it.lv.checkBoxes && !it.lv.virtual {
		state := uint32(0x1000)
		if checked {
			state = 0x2000
		}
		lvi := lvItem{stateMask: lvisStateMask, state: state}
		w32.SendMessage(it.lv.handle, lvmSetItemState, uintptr(it.index), uintptr(unsafe.Pointer(&lvi)))
	}
}

// Select makes this row the only selected row and scrolls it into view.
func (it *ListViewItem) Select() {
	if it.lv != nil {
		it.lv.SetSelectedIndex(it.index)
		it.lv.EnsureVisible(it.index)
	}
}

// EditLabel starts in-place editing of the row's first cell. It needs
// SetEditable(true).
func (it *ListViewItem) EditLabel() {
	if it.lv != nil && it.lv.handle != 0 {
		w32.SendMessage(it.lv.handle, lvmEditLabelW, uintptr(it.index), 0)
	}
}

// SelectedIndex returns the first selected row, or -1.
func (l *ListView) SelectedIndex() int {
	idx := l.SelectedIndices()
	if len(idx) == 0 {
		return -1
	}
	return idx[0]
}

// SelectedIndices returns all selected rows in ascending order.
func (l *ListView) SelectedIndices() []int {
	var out []int
	if l.handle == 0 {
		return out
	}
	next := ^uintptr(0)
	for {
		r := int(int32(w32.SendMessage(l.handle, lvmGetNextItem, next, lvniSelected)))
		if r < 0 {
			return out
		}
		out = append(out, r)
		next = uintptr(r)
	}
}

// SelectedItems returns the selected rows.
func (l *ListView) SelectedItems() []*ListViewItem {
	var out []*ListViewItem
	for _, i := range l.SelectedIndices() {
		if it := l.Item(i); it != nil {
			out = append(out, it)
		}
	}
	return out
}

// SetSelectedIndex selects only the given row, -1 clears the selection.
func (l *ListView) SetSelectedIndex(index int) {
	if l.handle == 0 {
		return
	}
	none := lvItem{stateMask: lvisSelected | lvisFocused}
	w32.SendMessage(l.handle, lvmSetItemState, ^uintptr(0), uintptr(unsafe.Pointer(&none)))
	if index >= 0 {
		sel := lvItem{state: lvisSelected | lvisFocused, stateMask: lvisSelected | lvisFocused}
		w32.SendMessage(l.handle, lvmSetItemState, uintptr(index), uintptr(unsafe.Pointer(&sel)))
	}
}

// SelectAll selects every row, if multiple selection is on.
func (l *ListView) SelectAll() {
	if l.handle == 0 || !l.multiSelect {
		return
	}
	sel := lvItem{state: lvisSelected, stateMask: lvisSelected}
	w32.SendMessage(l.handle, lvmSetItemState, ^uintptr(0), uintptr(unsafe.Pointer(&sel)))
}

// EnsureVisible scrolls the row into view.
func (l *ListView) EnsureVisible(index int) {
	if l.handle != 0 {
		w32.SendMessage(l.handle, lvmEnsureVisible, uintptr(index), 0)
	}
}

// SetView chooses how the items are shown.
func (l *ListView) SetView(v ListViewView) {
	l.view = v
	if l.handle != 0 {
		w32.SendMessage(l.handle, lvmSetView, lvViewCode(v), 0)
	}
}

// lvViewCode converts a ListViewView to the LV_VIEW_ value Windows uses.
func lvViewCode(v ListViewView) uintptr {
	switch v {
	case ListViewList:
		return 3
	case ListViewIcons:
		return 0
	case ListViewSmallIcons:
		return 2
	case ListViewTiles:
		return 4
	}
	return 1 // details
}

// View returns the current view.
func (l *ListView) View() ListViewView { return l.view }

// SetSmallImages sets the icons used by details, list and small icon views.
func (l *ListView) SetSmallImages(il *ImageList) {
	l.smallImages = il
	if l.handle != 0 && il != nil {
		w32.SendMessage(l.handle, lvmSetImageList, 1, il.handle)
	}
}

// SetLargeImages sets the icons used by the large icon and tile views.
func (l *ListView) SetLargeImages(il *ImageList) {
	l.largeImages = il
	if l.handle != 0 && il != nil {
		w32.SendMessage(l.handle, lvmSetImageList, 0, il.handle)
	}
}

// SetMultiSelect allows selecting several rows with Ctrl and Shift.
func (l *ListView) SetMultiSelect(on bool) { l.setStyle(&l.multiSelect, on) }

// MultiSelect tells whether several rows can be selected.
func (l *ListView) MultiSelect() bool { return l.multiSelect }

// SetEditable allows renaming the first cell of a row in place: click an
// already selected row or press F2.
func (l *ListView) SetEditable(on bool) { l.setStyle(&l.editable, on) }

// SetHeaderVisible shows or hides the column headers in details view.
func (l *ListView) SetHeaderVisible(on bool) { l.setStyle(&l.noHeader, !on) }

// SetCheckBoxes shows a check box in front of each row.
func (l *ListView) SetCheckBoxes(on bool) {
	if l.checkBoxes == on {
		return
	}
	l.checkBoxes = on
	if l.handle != 0 {
		l.applyExtStyle()
		l.populate()
	}
}

// SetFullRowSelect makes a click anywhere in the row select it (default).
func (l *ListView) SetFullRowSelect(on bool) {
	l.fullRow = on
	l.applyExtStyle()
}

// SetGridLines draws lines between the cells in details view.
func (l *ListView) SetGridLines(on bool) {
	l.gridLines = on
	l.applyExtStyle()
}

func (l *ListView) setStyle(field *bool, on bool) {
	if *field == on {
		return
	}
	*field = on
	if l.handle != 0 {
		l.recreate() // these window styles cannot be changed after creation
	}
}

// SetSortable sorts the rows when the user clicks a column header. Clicking
// the same header again reverses the order.
func (l *ListView) SetSortable(on bool) { l.sortable = on }

// SetCompare replaces the function that orders two cell texts when sorting.
// It returns a negative number, zero or a positive number. The default sorts
// numbers by value and other text without regard to case.
func (l *ListView) SetCompare(f func(a, b string) int) { l.compare = f }

func defaultListCompare(a, b string) int {
	fa, ea := strconv.ParseFloat(strings.TrimSpace(a), 64)
	fb, eb := strconv.ParseFloat(strings.TrimSpace(b), 64)
	if ea == nil && eb == nil {
		switch {
		case fa < fb:
			return -1
		case fa > fb:
			return 1
		}
		return 0
	}
	return strings.Compare(strings.ToLower(a), strings.ToLower(b))
}

// Sort orders the rows by the text of a column.
func (l *ListView) Sort(col int, ascending bool) {
	if l.virtual || col < 0 || col >= len(l.columns) {
		return
	}
	cmp := l.compare
	if cmp == nil {
		cmp = defaultListCompare
	}
	selected := l.SelectedItems()
	sort.SliceStable(l.items, func(i, j int) bool {
		r := cmp(l.items[i].Cell(col), l.items[j].Cell(col))
		if ascending {
			return r < 0
		}
		return r > 0
	})
	l.sortCol, l.sortAsc = col, ascending
	l.reindex()
	l.populate()
	for _, it := range selected {
		it.reselect()
	}
}

// reselect adds the row to the selection, used after sorting.
func (it *ListViewItem) reselect() {
	if it.lv == nil || it.lv.handle == 0 {
		return
	}
	sel := lvItem{state: lvisSelected, stateMask: lvisSelected}
	w32.SendMessage(it.lv.handle, lvmSetItemState, uintptr(it.index), uintptr(unsafe.Pointer(&sel)))
}

// SetVirtual switches to virtual mode for very large lists: no row objects
// exist, the list asks text(row, column) for what it needs to draw, so a
// million rows cost no memory. Rows cannot be added, sorted or checked in this
// mode. Call SetVirtualCount when the number of rows changes and Refresh when
// the data changed.
func (l *ListView) SetVirtual(count int, text func(row, col int) string) {
	l.virtualText = text
	l.virtualCount = count
	if !l.virtual {
		l.virtual = true
		l.items = nil
		if l.handle != 0 {
			l.recreate()
		}
		return
	}
	l.populate()
}

// SetVirtualImage sets the function that returns the image list index of a
// row in virtual mode.
func (l *ListView) SetVirtualImage(f func(row int) int) { l.virtualImage = f }

// SetVirtualCount changes the number of rows in virtual mode.
func (l *ListView) SetVirtualCount(count int) {
	l.virtualCount = count
	if l.handle != 0 && l.virtual {
		w32.SendMessage(l.handle, lvmSetItemCount, uintptr(count), 0)
	}
}

// Refresh redraws all rows, use it in virtual mode after the data changed.
func (l *ListView) Refresh() {
	if l.handle != 0 {
		w32.InvalidateRect(l.handle, nil, true)
	}
}

// SetOnSelect sets the function called with the row index when a row gets
// selected.
func (l *ListView) SetOnSelect(f func(index int)) { l.onSelect = f }

// SetOnActivate sets the function called when a row is double clicked or
// Enter is pressed on it.
func (l *ListView) SetOnActivate(f func(index int)) { l.onActivate = f }

// SetOnColumnClick sets the function called when a column header is clicked,
// in addition to sorting if that is on.
func (l *ListView) SetOnColumnClick(f func(column int)) { l.onColumnClick = f }

// SetOnItemCheck sets the function called when the user ticks or clears the
// check box of a row.
func (l *ListView) SetOnItemCheck(f func(item *ListViewItem, checked bool)) { l.onItemCheck = f }

// SetOnLabelEdit sets the function called when the user finished editing a
// label. Return false to reject the new text.
func (l *ListView) SetOnLabelEdit(f func(item *ListViewItem, newText string) bool) {
	l.onLabelEdit = f
}

func (l *ListView) handleNotify(code uint32, lParam uintptr) bool {
	switch code {
	case lvnItemChanged:
		if l.populating {
			return true
		}
		nm := (*w32.NMLISTVIEW)(unsafe.Pointer(lParam))
		if nm.UChanged&lvifState == 0 {
			return true
		}
		if (nm.UNewState^nm.UOldState)&lvisSelected != 0 && nm.UNewState&lvisSelected != 0 {
			if l.onSelect != nil {
				l.onSelect(int(nm.IItem))
			}
		}
		if !l.virtual && (nm.UNewState^nm.UOldState)&lvisStateMask != 0 && l.checkBoxes &&
			nm.UOldState&lvisStateMask != 0 {
			checked := nm.UNewState&lvisStateMask == 0x2000
			if it := l.Item(int(nm.IItem)); it != nil && it.checked != checked {
				it.checked = checked
				if l.onItemCheck != nil {
					l.onItemCheck(it, checked)
				}
			}
		}
		return true
	case lvnColumnClick:
		nm := (*w32.NMLISTVIEW)(unsafe.Pointer(lParam))
		col := int(nm.ISubItem)
		if l.sortable && !l.virtual {
			asc := true
			if l.sortCol == col {
				asc = !l.sortAsc
			}
			l.Sort(col, asc)
		}
		if l.onColumnClick != nil {
			l.onColumnClick(col)
		}
		return true
	case lvnItemActivate:
		nm := (*w32.NMLISTVIEW)(unsafe.Pointer(lParam))
		if l.onActivate != nil && nm.IItem >= 0 {
			l.onActivate(int(nm.IItem))
		}
		return true
	case lvnGetDispInfoW:
		if !l.virtual {
			return true
		}
		di := (*nmLVDispInfo)(unsafe.Pointer(lParam))
		row, col := int(di.item.iItem), int(di.item.iSubItem)
		if di.item.mask&lvifText != 0 && l.virtualText != nil && di.item.pszText != nil &&
			di.item.cchTextMax > 0 {
			n := int(di.item.cchTextMax)
			buf := (*[1 << 20]uint16)(unsafe.Pointer(di.item.pszText))[:n:n]
			text := stringToUTF16(l.virtualText(row, col))
			if len(text) > n {
				text = text[:n]
				text[n-1] = 0
			}
			copy(buf, text)
		}
		if di.item.mask&lvifImage != 0 {
			di.item.iImage = 0
			if l.virtualImage != nil {
				di.item.iImage = int32(l.virtualImage(row))
			}
		}
		return true
	case lvnEndLabelEditW:
		di := (*nmLVDispInfo)(unsafe.Pointer(lParam))
		if di.item.pszText == nil {
			return true // editing was cancelled
		}
		buf := (*[1 << 20]uint16)(unsafe.Pointer(di.item.pszText))[:]
		n := 0
		for n < len(buf) && buf[n] != 0 {
			n++
		}
		text := utf16ToString(buf[:n])
		if it := l.Item(int(di.item.iItem)); it != nil {
			if l.onLabelEdit == nil || l.onLabelEdit(it, text) {
				it.SetCell(0, text)
			}
		}
		return true
	}
	return false
}

// SetColumns adds columns with the given titles, each 100 pixels wide, for use
// from wml. Use AddColumn for control over the widths.
func (l *ListView) SetColumns(titles []string) {
	for _, t := range titles {
		l.AddColumn(t, 100)
	}
}

// SetItems adds one row per text, each with the text in the first column, for
// use from wml.
func (l *ListView) SetItems(rows []string) {
	for _, r := range rows {
		l.AddItem(r)
	}
}
