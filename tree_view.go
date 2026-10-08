package wui

import (
	"syscall"
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

const (
	treeViewClassName = "SysTreeView32"

	tvsHasButtons    = 0x0001
	tvsHasLines      = 0x0002
	tvsLinesAtRoot   = 0x0004
	tvsShowSelAlways = 0x0020

	tvmDeleteItem  = 0x1101
	tvmExpand      = 0x1102
	tvmGetNextItem = 0x110A
	tvmSelectItem  = 0x110B
	tvmInsertItemW = 0x1132
	tvmSetItemW    = 0x113F

	tvifText = 0x0001

	tveCollapse = 1
	tveExpand   = 2

	tvgnCaret = 9

	tviRoot = ^uintptr(0xFFFF) // -0x10000
	tviLast = ^uintptr(0xFFFD) // -0xFFFE

	tvnSelChangedW = 0xFFFFFE3D // TVN_FIRST-51
	nmDblClkCode   = 0xFFFFFFFD // NM_FIRST-3
)

type tvItem struct {
	mask           uint32
	hItem          uintptr
	state          uint32
	stateMask      uint32
	pszText        *uint16
	cchTextMax     int32
	iImage         int32
	iSelectedImage int32
	cChildren      int32
	lParam         uintptr
	iIntegral      int32
	uStateEx       uint32
	hwnd           uintptr
}

type tvInsertStruct struct {
	hParent      uintptr
	hInsertAfter uintptr
	item         tvItem
}

// NewTreeView creates a hierarchical list. Add top level entries with
// TreeView.Add and children with TreeNode.Add.
func NewTreeView() *TreeView {
	return &TreeView{nodes: make(map[uintptr]*TreeNode)}
}

type TreeView struct {
	textControl
	roots         []*TreeNode
	nodes         map[uintptr]*TreeNode
	onSelect      func(*TreeNode)
	onDoubleClick func(*TreeNode)
	images        *ImageList
	checkBoxes    bool
	editable      bool
	onExpand      func(*TreeNode)
	onCollapse    func(*TreeNode)
	onCheck       func(*TreeNode, bool)
	onLabelEdit   func(*TreeNode, string) bool
}

var _ Control = (*TreeView)(nil)

func (*TreeView) canFocus() bool { return true }

func (*TreeView) eatsTabs() bool { return false }

// TreeNode is one entry of a TreeView.
type TreeNode struct {
	// Tag can hold any value you want to associate with the node.
	Tag      interface{}
	tree     *TreeView
	parent   *TreeNode
	children []*TreeNode
	text     string
	handle   uintptr
	image    int
	selImage int
	checked  bool
}

func (t *TreeView) create(id int) {
	t.textControl.create(
		id, w32.WS_EX_CLIENTEDGE, treeViewClassName,
		w32.WS_TABSTOP|tvsHasButtons|tvsHasLines|tvsLinesAtRoot|tvsShowSelAlways|t.extraStyle(),
	)
	t.nodes = make(map[uintptr]*TreeNode)
	t.applyImages()
	for _, r := range t.roots {
		t.insert(r, tviRoot)
	}
	t.applyToolTip()
}

func (t *TreeView) insert(n *TreeNode, parent uintptr) {
	var is tvInsertStruct
	is.hParent = parent
	is.hInsertAfter = tviLast
	is.item.mask = tvifText
	is.item.pszText = syscall.StringToUTF16Ptr(n.text)
	if t.images != nil {
		is.item.mask |= tvifImage | tvifSelectedImage
		is.item.iImage = int32(n.image)
		is.item.iSelectedImage = int32(n.selImage)
	}
	n.handle = w32.SendMessage(t.handle, tvmInsertItemW, 0, uintptr(unsafe.Pointer(&is)))
	t.nodes[n.handle] = n
	if t.checkBoxes && n.checked {
		t.setNodeChecked(n, true)
	}
	for _, c := range n.children {
		t.insert(c, n.handle)
	}
}

// Add appends a top level node.
func (t *TreeView) Add(text string) *TreeNode {
	n := &TreeNode{tree: t, text: text}
	t.roots = append(t.roots, n)
	if t.handle != 0 {
		t.insert(n, tviRoot)
	}
	return n
}

// Roots returns the top level nodes.
func (t *TreeView) Roots() []*TreeNode {
	return append([]*TreeNode(nil), t.roots...)
}

// Clear removes all nodes.
func (t *TreeView) Clear() {
	t.roots = nil
	t.nodes = make(map[uintptr]*TreeNode)
	if t.handle != 0 {
		w32.SendMessage(t.handle, tvmDeleteItem, 0, tviRoot)
	}
}

// Selected returns the selected node or nil.
func (t *TreeView) Selected() *TreeNode {
	if t.handle == 0 {
		return nil
	}
	h := w32.SendMessage(t.handle, tvmGetNextItem, tvgnCaret, 0)
	return t.nodes[h]
}

// ExpandAll opens every node.
func (t *TreeView) ExpandAll() {
	for _, r := range t.roots {
		r.expandRec(true)
	}
}

// CollapseAll closes every node.
func (t *TreeView) CollapseAll() {
	for _, r := range t.roots {
		r.expandRec(false)
	}
}

// SetOnSelect sets the function called when the selected node changes.
func (t *TreeView) SetOnSelect(f func(*TreeNode)) { t.onSelect = f }

func (t *TreeView) OnSelect() func(*TreeNode) { return t.onSelect }

// SetOnDoubleClick sets the function called when a node is double clicked.
func (t *TreeView) SetOnDoubleClick(f func(*TreeNode)) { t.onDoubleClick = f }

func (t *TreeView) OnDoubleClick() func(*TreeNode) { return t.onDoubleClick }

func (t *TreeView) notify(code uint32) {
	switch code {
	case tvnSelChangedW:
		if t.onSelect != nil {
			if n := t.Selected(); n != nil {
				t.onSelect(n)
			}
		}
	case nmDblClkCode:
		if t.onDoubleClick != nil {
			if n := t.Selected(); n != nil {
				t.onDoubleClick(n)
			}
		}
	}
}

// Add appends a child node.
func (n *TreeNode) Add(text string) *TreeNode {
	c := &TreeNode{tree: n.tree, parent: n, text: text}
	n.children = append(n.children, c)
	if n.tree != nil && n.tree.handle != 0 && n.handle != 0 {
		n.tree.insert(c, n.handle)
	}
	return c
}

func (n *TreeNode) Text() string { return n.text }

func (n *TreeNode) SetText(text string) {
	n.text = text
	if n.tree != nil && n.tree.handle != 0 && n.handle != 0 {
		var item tvItem
		item.mask = tvifText
		item.hItem = n.handle
		item.pszText = syscall.StringToUTF16Ptr(text)
		w32.SendMessage(n.tree.handle, tvmSetItemW, 0, uintptr(unsafe.Pointer(&item)))
	}
}

func (n *TreeNode) Parent() *TreeNode { return n.parent }

func (n *TreeNode) Children() []*TreeNode {
	return append([]*TreeNode(nil), n.children...)
}

// Select makes this node the selected one.
func (n *TreeNode) Select() {
	if n.tree != nil && n.tree.handle != 0 && n.handle != 0 {
		w32.SendMessage(n.tree.handle, tvmSelectItem, tvgnCaret, n.handle)
	}
}

// Expand opens the node.
func (n *TreeNode) Expand() {
	if n.tree != nil && n.tree.handle != 0 && n.handle != 0 {
		w32.SendMessage(n.tree.handle, tvmExpand, tveExpand, n.handle)
	}
}

// Collapse closes the node.
func (n *TreeNode) Collapse() {
	if n.tree != nil && n.tree.handle != 0 && n.handle != 0 {
		w32.SendMessage(n.tree.handle, tvmExpand, tveCollapse, n.handle)
	}
}

func (n *TreeNode) expandRec(open bool) {
	for _, c := range n.children {
		c.expandRec(open)
	}
	if open {
		n.Expand()
	} else {
		n.Collapse()
	}
}

func (n *TreeNode) forget() {
	if n.tree != nil {
		delete(n.tree.nodes, n.handle)
	}
	for _, c := range n.children {
		c.forget()
	}
}

// Remove deletes the node and all its children from the tree.
func (n *TreeNode) Remove() {
	t := n.tree
	if t == nil {
		return
	}
	if t.handle != 0 && n.handle != 0 {
		w32.SendMessage(t.handle, tvmDeleteItem, 0, n.handle)
	}
	n.forget()
	list := &t.roots
	if n.parent != nil {
		list = &n.parent.children
	}
	for i, c := range *list {
		if c == n {
			*list = append((*list)[:i], (*list)[i+1:]...)
			break
		}
	}
	n.tree = nil
}
