// TreeView with icons, check boxes and renaming, and a TabControl whose pages
// show and hide automatically.
package main

import (
	"fmt"
	"image"
	"image/color"

	"github.com/2dprototype/wui"
)

func square(c color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func main() {
	w := wui.NewWindow()
	w.SetTitle("tree and tabs")
	w.SetInnerSize(640, 400)

	icons := wui.NewImageList(16, 16)
	folder, _ := icons.AddImage(square(color.RGBA{230, 190, 60, 255}))
	file, _ := icons.AddImage(square(color.RGBA{90, 140, 220, 255}))

	status := wui.NewLabel()
	status.SetBounds(8, 372, 620, 22)

	tree := wui.NewTreeView()
	tree.SetBounds(8, 8, 220, 356)
	tree.SetImages(icons)
	tree.SetCheckBoxes(true)
	tree.SetEditable(true)
	root := tree.Add("Project")
	root.SetImage(folder, folder)
	for i := 1; i <= 3; i++ {
		n := root.Add(fmt.Sprintf("item %d", i))
		n.SetImage(file, file)
	}
	tree.SetOnCheck(func(n *wui.TreeNode, checked bool) {
		status.SetText(fmt.Sprintf("%s checked=%v", n.Text(), checked))
	})
	tree.SetOnExpand(func(n *wui.TreeNode) { status.SetText("expanded " + n.Text()) })
	tree.SetOnLabelEdit(func(n *wui.TreeNode, text string) bool { return text != "" })
	tree.ExpandAll()

	// A tab control with real pages.
	tabs := wui.NewTabControl()
	tabs.SetBounds(240, 8, 392, 356)
	tabs.SetImages(icons)

	general := wui.NewPanel()
	l := wui.NewLabel()
	l.SetText("General page")
	l.SetBounds(10, 10, 200, 20)
	general.Add(l)

	advanced := wui.NewPanel()
	cb := wui.NewCheckBox()
	cb.SetText("Advanced option")
	cb.SetBounds(10, 10, 200, 20)
	advanced.Add(cb)

	w.Add(tree)
	w.Add(tabs)
	w.Add(general)
	w.Add(advanced)
	w.Add(status)
	tabs.AddPage("General", general, file)
	tabs.AddPage("Advanced", advanced, folder)
	w.Show()
}
