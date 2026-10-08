package wui

import (
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

const (
	tvsEditLabels  = 0x0008
	tvsCheckBoxes  = 0x0100
	tvmSetImageList = 0x1109
	tvmGetItemState = 0x1127
	tvifImage         = 0x0002
	tvifState         = 0x0008
	tvifSelectedImage = 0x0020
	tvisStateImageMask = 0xF000

	tvnItemExpandedW  uint32 = 0xFFFFFE39 // TVN_FIRST-55
	tvnEndLabelEditW  uint32 = 0xFFFFFE34 // TVN_FIRST-60
	tvnItemChangedW   uint32 = 0xFFFFFE5D // TVN_FIRST-19
)

type nmTreeView struct {
	hdr     w32.NMHDR
	action  uint32
	itemOld tvItem
	itemNew tvItem
	ptDrag  w32.POINT
}

type nmTVDispInfo struct {
	hdr  w32.NMHDR
	item tvItem
}

type nmTVItemChange struct {
	hdr       w32.NMHDR
	uChanged  uint32
	hItem     uintptr
	uStateNew uint32
	uStateOld uint32
	lParam    uintptr
}

func (t *TreeView) extraStyle() uint {
	var s uint
	if t.checkBoxes {
		s |= tvsCheckBoxes
	}
	if t.editable {
		s |= tvsEditLabels
	}
	return s
}

func (t *TreeView) applyImages() {
	if t.handle != 0 && t.images != nil {
		w32.SendMessage(t.handle, tvmSetImageList, 0, t.images.handle)
	}
}

// SetImages gives the tree icons. Nodes show the image with the index set by
// TreeNode.SetImage.
func (t *TreeView) SetImages(il *ImageList) {
	t.images = il
	t.applyImages()
}

// SetCheckBoxes shows a check box in front of every node. Call it before the
// window is shown.
func (t *TreeView) SetCheckBoxes(on bool) { t.checkBoxes = on }

// SetEditable lets the user rename nodes in place (click a selected node or
// press F2). Call it before the window is shown.
func (t *TreeView) SetEditable(on bool) { t.editable = on }

// SetOnExpand sets the function called after a node was opened.
func (t *TreeView) SetOnExpand(f func(*TreeNode)) { t.onExpand = f }

// SetOnCollapse sets the function called after a node was closed.
func (t *TreeView) SetOnCollapse(f func(*TreeNode)) { t.onCollapse = f }

// SetOnCheck sets the function called when the user ticks or clears the check
// box of a node.
func (t *TreeView) SetOnCheck(f func(n *TreeNode, checked bool)) { t.onCheck = f }

// SetOnLabelEdit sets the function called when the user finished renaming a
// node. Return false to reject the new text.
func (t *TreeView) SetOnLabelEdit(f func(n *TreeNode, text string) bool) { t.onLabelEdit = f }

func (t *TreeView) setNodeChecked(n *TreeNode, on bool) {
	item := tvItem{mask: tvifState, hItem: n.handle, stateMask: tvisStateImageMask}
	if on {
		item.state = 0x2000
	} else {
		item.state = 0x1000
	}
	w32.SendMessage(t.handle, tvmSetItemW, 0, uintptr(unsafe.Pointer(&item)))
}

// SetImage chooses the icons of the node: image is shown normally, selected
// when the node is selected. They are indexes into the tree's image list.
func (n *TreeNode) SetImage(image, selected int) {
	n.image, n.selImage = image, selected
	if n.tree != nil && n.tree.handle != 0 && n.handle != 0 && n.tree.images != nil {
		item := tvItem{
			mask: tvifImage | tvifSelectedImage, hItem: n.handle,
			iImage: int32(image), iSelectedImage: int32(selected),
		}
		w32.SendMessage(n.tree.handle, tvmSetItemW, 0, uintptr(unsafe.Pointer(&item)))
	}
}

// Checked tells whether the node's check box is ticked.
func (n *TreeNode) Checked() bool {
	if n.tree != nil && n.tree.handle != 0 && n.handle != 0 && n.tree.checkBoxes {
		st := w32.SendMessage(n.tree.handle, tvmGetItemState, n.handle, tvisStateImageMask)
		n.checked = (st&tvisStateImageMask)>>12 == 2
	}
	return n.checked
}

// SetChecked ticks or clears the node's check box.
func (n *TreeNode) SetChecked(on bool) {
	n.checked = on
	if n.tree != nil && n.tree.handle != 0 && n.handle != 0 && n.tree.checkBoxes {
		n.tree.setNodeChecked(n, on)
	}
}

func (t *TreeView) handleNotify(code uint32, lParam uintptr) bool {
	switch code {
	case tvnItemExpandedW:
		nm := (*nmTreeView)(unsafe.Pointer(lParam))
		if n := t.nodes[nm.itemNew.hItem]; n != nil {
			if nm.action == tveExpand && t.onExpand != nil {
				t.onExpand(n)
			} else if nm.action == tveCollapse && t.onCollapse != nil {
				t.onCollapse(n)
			}
		}
		return true
	case tvnItemChangedW:
		nm := (*nmTVItemChange)(unsafe.Pointer(lParam))
		if nm.uChanged&tvifState != 0 && t.checkBoxes &&
			nm.uStateOld&tvisStateImageMask != 0 &&
			(nm.uStateNew^nm.uStateOld)&tvisStateImageMask != 0 {
			if n := t.nodes[nm.hItem]; n != nil {
				checked := nm.uStateNew&tvisStateImageMask == 0x2000
				if n.checked != checked {
					n.checked = checked
					if t.onCheck != nil {
						t.onCheck(n, checked)
					}
				}
			}
		}
		return true
	case tvnEndLabelEditW:
		nm := (*nmTVDispInfo)(unsafe.Pointer(lParam))
		if nm.item.pszText == nil {
			return true // cancelled
		}
		buf := (*[1 << 20]uint16)(unsafe.Pointer(nm.item.pszText))[:]
		k := 0
		for k < len(buf) && buf[k] != 0 {
			k++
		}
		text := utf16ToString(buf[:k])
		if n := t.nodes[nm.item.hItem]; n != nil {
			if t.onLabelEdit == nil || t.onLabelEdit(n, text) {
				n.SetText(text)
			}
		}
		return true
	}
	return false
}
