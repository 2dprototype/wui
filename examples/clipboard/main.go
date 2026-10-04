// Clipboard example: copy the edit text to the clipboard and paste it back.
package main

import "github.com/2dprototype/wui"

func main() {
	w := wui.NewWindow()
	w.SetTitle("Clipboard")
	w.SetInnerSize(320, 110)
	w.SetCenterOnShow(true)

	edit := wui.NewEditLine()
	edit.SetText("Hello clipboard")
	edit.SetBounds(10, 10, 300, 25)
	w.Add(edit)

	copyBtn := wui.NewButton()
	copyBtn.SetText("Copy")
	copyBtn.SetBounds(10, 55, 100, 30)
	copyBtn.SetOnClick(func() {
		wui.SetClipboardText(edit.Text())
	})
	w.Add(copyBtn)

	pasteBtn := wui.NewButton()
	pasteBtn.SetText("Paste")
	pasteBtn.SetBounds(120, 55, 100, 30)
	pasteBtn.SetOnClick(func() {
		if text, ok := wui.ClipboardText(); ok {
			edit.SetText(text)
		}
	})
	w.Add(pasteBtn)

	w.Show()
}
