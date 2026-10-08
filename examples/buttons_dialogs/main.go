// Special buttons, check boxes, new input controls and dialogs.
package main

import (
	"fmt"
	"time"

	"github.com/2dprototype/wui"
)

func main() {
	w := wui.NewWindow()
	w.SetTitle("buttons and dialogs")
	w.SetInnerSize(520, 440)

	out := wui.NewLabel()
	out.SetBounds(10, 410, 500, 22)

	// Split button with a drop down menu.
	menu := wui.NewPopupMenu()
	for _, name := range []string{"Save as PDF", "Save as text"} {
		item := wui.NewMenuString(name)
		n := name
		item.SetOnClick(func() { out.SetText(n) })
		menu.Add(item)
	}
	split := wui.NewButton()
	split.SetKind(wui.ButtonSplit)
	split.SetText("Save")
	split.SetDropDownMenu(menu)
	split.SetBounds(10, 10, 140, 30)
	split.SetOnClick(func() { out.SetText("Save clicked") })

	// Command link with a note, and OK / Cancel with Enter and Esc.
	link := wui.NewButton()
	link.SetKind(wui.ButtonCommandLink)
	link.SetText("Install now")
	link.SetNote("Installs for all users")
	link.SetBounds(160, 10, 220, 60)

	ok := wui.NewButton()
	ok.SetText("OK")
	ok.SetBounds(300, 400, 80, 26)
	ok.SetOnClick(func() { out.SetText("OK (Enter)") })
	cancel := wui.NewButton()
	cancel.SetText("Cancel")
	cancel.SetBounds(390, 400, 80, 26)
	cancel.SetOnClick(func() { out.SetText("Cancel (Esc)") })
	w.SetDefaultButton(ok)
	w.SetCancelButton(cancel)

	three := wui.NewCheckBox()
	three.SetText("Three state")
	three.SetThreeState(true)
	three.SetIndeterminate(true)
	three.SetBounds(10, 80, 150, 22)

	toggle := wui.NewCheckBox()
	toggle.SetText("Toggle button")
	toggle.SetPushLike(true)
	toggle.SetBounds(170, 80, 120, 26)

	// Inputs.
	name := wui.NewEditLine()
	name.SetCueBanner("type your name...", true)
	name.SetBounds(10, 120, 200, 24)
	num := wui.NewEditLine()
	num.SetNumbersOnly(true)
	num.SetTextAlign(wui.EditAlignRight)
	num.SetBounds(220, 120, 80, 24)
	combo := wui.NewComboBox()
	combo.SetEditable(true)
	combo.SetItems([]string{"red", "green", "blue"})
	combo.SetBounds(310, 120, 120, 24)

	hk := wui.NewHotKeyEdit()
	hk.SetBounds(10, 156, 140, 24)
	hk.SetOnChange(func() {
		key, mods := hk.HotKey()
		out.SetText(fmt.Sprintf("hot key %d mods %d", key, mods))
	})
	ip := wui.NewIPAddressEdit()
	ip.SetAddress("192.168.0.1")
	ip.SetBounds(160, 156, 150, 24)
	cal := wui.NewMonthCalendar()
	cal.SetBounds(10, 190, 230, 160)
	cal.SetOnChange(func(t time.Time) { out.SetText(t.Format("2006-01-02")) })
	lbl := wui.NewLinkLabel()
	lbl.SetText(`See <a href="https://github.com">GitHub</a> or <a id="help">help</a>`)
	lbl.SetBounds(260, 190, 250, 22)
	lbl.SetOnClick(func(url, id string) {
		if url != "" {
			wui.OpenURL(url)
		} else {
			out.SetText("link id " + id)
		}
	})

	// Dialogs.
	dlg := wui.NewButton()
	dlg.SetText("Input...")
	dlg.SetBounds(260, 230, 80, 26)
	dlg.SetOnClick(func() {
		if s, ok := wui.InputDialog("Name", "Your name:", "Ann"); ok {
			out.SetText("hello " + s)
		}
	})
	fnt := wui.NewButton()
	fnt.SetText("Font...")
	fnt.SetBounds(350, 230, 80, 26)
	fnt.SetOnClick(func() {
		fd := wui.NewFontDialog()
		if fd.Execute(w) {
			out.SetText(fmt.Sprintf("%s %dpt", fd.Font().Name, fd.Points()))
		}
	})
	yn := wui.NewButton()
	yn.SetText("Ask...")
	yn.SetBounds(260, 266, 80, 26)
	yn.SetOnClick(func() {
		switch wui.MessageBoxYesNoCancel("wui", "Save changes?") {
		case wui.DialogYes:
			out.SetText("yes")
		case wui.DialogNo:
			out.SetText("no")
		default:
			out.SetText("cancel")
		}
	})

	for _, c := range []wui.Control{split, link, ok, cancel, three, toggle, name, num, combo, hk, ip, cal, lbl, dlg, fnt, yn, out} {
		w.Add(c)
	}
	w.Show()
}
