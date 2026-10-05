// ToolBar example: a row of buttons above a text box.
package main

import "github.com/2dprototype/wui"

func main() {
	w := wui.NewWindow()
	w.SetTitle("ToolBar")
	w.SetInnerSize(420, 220)
	w.SetCenterOnShow(true)

	text := wui.NewTextEdit()
	text.SetBounds(0, 34, 420, 186)
	text.SetAnchors(wui.AnchorMinAndMax, wui.AnchorMinAndMax)
	w.Add(text)

	tb := wui.NewToolBar(w, 2, 2, 28)
	tb.AddButton("New", 60, func() { text.SetText("") }).SetToolTip("Clear the text")
	tb.AddSeparator()
	tb.AddButton("Copy", 60, func() { wui.SetClipboardText(text.Text()) })
	tb.AddButton("Paste", 60, func() {
		if s, ok := wui.ClipboardText(); ok {
			text.SetText(s)
		}
	})

	w.Show()
}
