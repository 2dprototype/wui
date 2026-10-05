package main

import (
	"strings"

	"github.com/2dprototype/wui"
)

func main() {
	window := wui.NewWindow()
	window.SetTitle("Drag and drop files here")
	window.SetInnerSize(480, 300)

	list := wui.NewTextEdit()
	list.SetBounds(10, 10, 460, 280)
	list.SetAnchors(wui.AnchorMinAndMax, wui.AnchorMinAndMax)
	list.SetText("Drop files from the file explorer onto this window.")
	window.Add(list)

	// The whole window accepts files, the handler gets the file paths.
	window.SetOnDropFiles(func(files []string, x, y int) {
		list.SetText(strings.Join(files, "\r\n"))
	})
	window.Show()
}
