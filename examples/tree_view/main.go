// TreeView example: a small hierarchy that shows the selected node.
package main

import "github.com/2dprototype/wui"

func main() {
	w := wui.NewWindow()
	w.SetTitle("TreeView")
	w.SetInnerSize(420, 300)
	w.SetCenterOnShow(true)

	tree := wui.NewTreeView()
	tree.SetBounds(10, 10, 200, 280)
	tree.SetAnchors(wui.AnchorMin, wui.AnchorMinAndMax)

	fruits := tree.Add("Fruits")
	fruits.Add("Apple")
	fruits.Add("Banana")
	veg := tree.Add("Vegetables")
	veg.Add("Carrot")
	roots := veg.Add("Root vegetables")
	roots.Add("Beet")
	roots.Add("Radish")
	w.Add(tree)

	info := wui.NewLabel()
	info.SetBounds(225, 15, 190, 25)
	info.SetText("Select a node")
	w.Add(info)

	tree.SetOnSelect(func(n *wui.TreeNode) {
		info.SetText("Selected: " + n.Text())
	})
	tree.SetOnDoubleClick(func(n *wui.TreeNode) {
		info.SetText("Double click: " + n.Text())
	})

	expand := wui.NewButton()
	expand.SetText("Expand all")
	expand.SetBounds(225, 60, 120, 28)
	expand.SetOnClick(tree.ExpandAll)
	w.Add(expand)

	collapse := wui.NewButton()
	collapse.SetText("Collapse all")
	collapse.SetBounds(225, 95, 120, 28)
	collapse.SetOnClick(tree.CollapseAll)
	w.Add(collapse)

	w.Show()
}
