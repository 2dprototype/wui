package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/2dprototype/wui/w32"
	"github.com/2dprototype/wui"
)

// TODO Have icon for window.
// TODO Have cursor properties for all controls, first let all controls have changeable cursors.
// TODO Edit main menu.
// TODO Have a way to edit shortcuts.
// TODO Have a way to hide the app icon (WS_EX_DLGMODALFRAME).
// TODO Have a way to hide the border completely but make it still resizeable.

var (
	names      = make(map[interface{}]string)
	buildDir    = "."
	buildPrefix = "wui_designer_preview_"
	buildCount  = 0
	events     = make(map[event]string)
)

type event struct {
	control interface{}
	name    string
}

func main() {
	if dir, err := ioutil.TempDir("", "wui_designer_preview_builds"); err == nil {
		buildDir = dir
		defer os.Remove(dir)
	}
	defer func() {
		if files, err := ioutil.ReadDir(buildDir); err == nil {
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".exe") && strings.HasPrefix(file.Name(), buildPrefix) {
					os.Remove(filepath.Join(buildDir, file.Name()))
				}
			}
			os.Remove(filepath.Join(buildDir, "go.mod"))
			os.Remove(filepath.Join(buildDir, "go.sum"))
		}
	}()

	var (
		innerX, innerY int
		active         node
		preview        = wui.NewPaintBox()
	)

	font, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -11})
	bold, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -11, Bold: true})
	italic, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -11, Italic: true})
	underlined, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -11, Underlined: true})
	strikedOut, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -11, StrikedOut: true})
	w := wui.NewWindow()
	w.SetFont(font)
	w.SetTitle("wui Designer - Untitled")
	w.SetBackground(wui.ColorButtonFace)
	w.SetInnerSize(800, 600)

	theWindow := defaultWindow()
	names[theWindow] = "window"

	// Menu setup
	menu := wui.NewMainMenu()
	fileMenu := wui.NewMenu("&File")
	editMenu := wui.NewMenu("&Edit")
	viewMenu := wui.NewMenu("&View")
	helpMenu := wui.NewMenu("&Help")

	// File menu items
	fileNewMenu := wui.NewMenuString("&New Project\tCtrl+N")
	fileOpenMenu := wui.NewMenuString("&Open Project...\tCtrl+O")
	fileSaveMenu := wui.NewMenuString("&Save Project\tCtrl+S")
	fileSaveAsMenu := wui.NewMenuString("Save Project &As...\tCtrl+Shift+S")
	fileExportGoMenu := wui.NewMenuString("&Export as Go...\tCtrl+E")
	previewMenu := wui.NewMenuString("&Run Preview\tF5")
	exitMenu := wui.NewMenuString("E&xit\tAlt+F4")

	// Edit menu items
	cutMenu := wui.NewMenuString("Cu&t\tCtrl+X")
	copyMenu := wui.NewMenuString("&Copy\tCtrl+C")
	pasteMenu := wui.NewMenuString("&Paste\tCtrl+V")
	deleteMenu := wui.NewMenuString("&Delete\tDel")

	// View menu items
	viewFullScreenMenu := wui.NewMenuString("&Maximize Window\tF11")
	viewToolboxMenu := wui.NewMenuString("Toggle &Toolbox\tF2")
	viewPropsMenu := wui.NewMenuString("Toggle &Properties\tF3")
	viewAutoLayoutMenu := wui.NewMenuString("&Auto Layout\tF4")
	duplicateMenu := wui.NewMenuString("D&uplicate\tCtrl+D")

	// Help menu items
	helpAboutMenu := wui.NewMenuString("&About wui Designer\tF1")
	helpShortcutsMenu := wui.NewMenuString("&Keyboard Shortcuts\tCtrl+F1")

	// Build File menu
	fileMenu.Add(fileNewMenu)
	fileMenu.Add(fileOpenMenu)
	fileMenu.Add(wui.NewMenuSeparator())
	fileMenu.Add(fileSaveMenu)
	fileMenu.Add(fileSaveAsMenu)
	fileMenu.Add(wui.NewMenuSeparator())
	fileMenu.Add(fileExportGoMenu)
	fileMenu.Add(wui.NewMenuSeparator())
	fileMenu.Add(previewMenu)
	fileMenu.Add(wui.NewMenuSeparator())
	fileMenu.Add(exitMenu)

	// Build Edit menu
	editMenu.Add(cutMenu)
	editMenu.Add(copyMenu)
	editMenu.Add(pasteMenu)
	editMenu.Add(duplicateMenu)
	editMenu.Add(wui.NewMenuSeparator())
	editMenu.Add(deleteMenu)

	// Build View menu
	viewMenu.Add(viewToolboxMenu)
	viewMenu.Add(viewPropsMenu)
	viewMenu.Add(viewAutoLayoutMenu)
	viewMenu.Add(wui.NewMenuSeparator())
	viewMenu.Add(viewFullScreenMenu)

	// Build Help menu
	helpMenu.Add(helpAboutMenu)
	helpMenu.Add(helpShortcutsMenu)

	// Add menus to main menu
	menu.Add(fileMenu)
	menu.Add(editMenu)
	menu.Add(viewMenu)
	menu.Add(helpMenu)
	w.SetMenu(menu)

	type uiProp struct {
		panel     *wui.Panel
		setter    string
		getter    string // if set, the property is only shown when the control lists it
		update    func()
		rightType func(t reflect.Type) bool
		// visibleFor, if set, decides whether the property is shown for a
		// control. It replaces the check of the setter.
		visibleFor func(c interface{}) bool
		// isExtra marks the editors of the extra properties of extras.go.
		isExtra bool
		// group overrides the group that the setter name gives.
		group string
	}
	var updateProperties func()
	// updatingProps is true while the editors are filled from the control,
	// so that this does not count as an edit.
	updatingProps := false

	const propMargin = 2

	// File operations
	workingPath := ""
	projectModified := false

	setWorkingPath := func(path string) {
		workingPath = path
		title := "wui Designer"
		if path != "" {
			title += " - " + filepath.Base(path)
		} else {
			title += " - Untitled"
		}
		if projectModified {
			title += " *"
		}
		w.SetTitle(title)
	}

	markModified := func() {
		if !projectModified {
			projectModified = true
			setWorkingPath(workingPath)
		}
	}

	markSaved := func() {
		projectModified = false
		setWorkingPath(workingPath)
	}

	saveProjectFile := func(path string) bool {
		if err := saveProject(theWindow, path); err != nil {
			wui.MessageBoxError("Error", "Failed to save project: "+err.Error())
			return false
		}
		setWorkingPath(path)
		markSaved()
		return true
	}

	// saveAs asks for a file name and saves the project there.
	saveAs := func() bool {
		save := wui.NewFileSaveDialog()
		save.SetAppendExt(true)
		save.SetTitle("Save wui Designer Project")
		save.AddFilter("WML project file", projectExt)
		if accept, path := save.Execute(w); accept {
			return saveProjectFile(path)
		}
		return false
	}

	saveCurrent := func() bool {
		if workingPath != "" {
			return saveProjectFile(workingPath)
		}
		return saveAs()
	}

	// confirmDiscard asks what to do with unsaved changes. It returns false if
	// the user wants to keep working, or if saving did not succeed.
	confirmDiscard := func(question string) bool {
		if !projectModified {
			return true
		}
		switch wui.MessageBoxCustom("Unsaved Changes", question,
			w32.MB_YESNOCANCEL|w32.MB_ICONQUESTION) {
		case w32.IDYES:
			return saveCurrent()
		case w32.IDNO:
			return true
		}
		return false
	}

	boolPanel := func(parent wui.Container, name string) (*wui.CheckBox, *wui.Panel) {
		c := wui.NewCheckBox()
		c.SetText(name)
		c.SetBounds(100, propMargin, 95, 17)
		p := wui.NewPanel()
		p.SetSize(195, c.Height()+2*propMargin)
		parent.Add(p)
		p.Add(c)
		return c, p
	}

	boolProp := func(name, getterFunc string) uiProp {
		c, p := boolPanel(w, name)
		setterFunc := "Set" + getterFunc
		c.SetOnChange(func(on bool) {
			reflect.ValueOf(active).MethodByName(setterFunc).Call(
				[]reflect.Value{reflect.ValueOf(on)},
			)
			updateProperties()
			preview.Paint()
			markModified()
		})
		update := func() {
			on := reflect.ValueOf(active).MethodByName(getterFunc).Call(nil)[0].Bool()
			if c.Checked() != on {
				c.SetChecked(on)
			}
		}
		rightType := func(t reflect.Type) bool {
			return t.Kind() == reflect.Bool
		}
		return uiProp{
			panel:     p,
			setter:    setterFunc,
			update:    update,
			rightType: rightType,
		}
	}

	intPanel := func(parent wui.Container, name string, minmax ...int) (*wui.IntUpDown, *wui.Panel) {
		n := wui.NewIntUpDown()
		n.SetOnTabFocus(n.SelectAll)
		if len(minmax) == 2 {
			n.SetMinMax(minmax[0], minmax[1])
		}
		n.SetBounds(100, propMargin, 90, 22)
		l := wui.NewLabel()
		l.SetText(name)
		l.SetAlignment(wui.AlignRight)
		l.SetBounds(0, propMargin-1, 95, n.Height())
		p := wui.NewPanel()
		p.SetSize(195, n.Height()+2+2*propMargin)
		parent.Add(p)
		p.Add(l)
		p.Add(n)
		return n, p
	}

	intProp := func(name, getterFunc string, minmax ...int) uiProp {
		n, p := intPanel(w, name, minmax...)
		setterFunc := "Set" + getterFunc
		n.SetOnValueChange(func(v int) {
			if active == nil {
				return
			}
			if m, ok := reflect.TypeOf(active).MethodByName(setterFunc); ok {
				reflect.ValueOf(active).MethodByName(setterFunc).Call(
					[]reflect.Value{reflect.ValueOf(v).Convert(m.Type.In(1))},
				)
				updateProperties()
				preview.Paint()
				markModified()
			}
		})
		update := func() {
			v := reflect.ValueOf(active).MethodByName(getterFunc).Call(nil)[0]
			i := v.Convert(reflect.TypeOf(0)).Int()
			newValue := int(i)
			if n.Value() != newValue {
				n.SetValue(newValue)
			}
		}
		rightType := func(t reflect.Type) bool {
			return t.Kind() == reflect.Int || t.Kind() == reflect.Uint8
		}
		return uiProp{
			panel:     p,
			setter:    setterFunc,
			update:    update,
			rightType: rightType,
		}
	}

	floatProp := func(name, getterFunc string, minmax ...float64) uiProp {
		setterFunc := "Set" + getterFunc
		n := wui.NewFloatUpDown()
		n.SetOnTabFocus(n.SelectAll)
		if len(minmax) == 2 {
			n.SetMinMax(minmax[0], minmax[1])
		}
		n.SetPrecision(6)
		n.SetBounds(100, propMargin, 90, 22)
		l := wui.NewLabel()
		l.SetText(name)
		l.SetAlignment(wui.AlignRight)
		l.SetBounds(0, propMargin-1, 95, n.Height())
		p := wui.NewPanel()
		p.SetSize(195, n.Height()+2+2*propMargin)
		w.Add(p)
		p.Add(l)
		p.Add(n)
		n.SetOnValueChange(func(v float64) {
			if active == nil {
				return
			}
			if m, ok := reflect.TypeOf(active).MethodByName(setterFunc); ok {
				reflect.ValueOf(active).MethodByName(setterFunc).Call(
					[]reflect.Value{reflect.ValueOf(v).Convert(m.Type.In(1))},
				)
				updateProperties()
				preview.Paint()
				markModified()
			}
		})
		update := func() {
			v := reflect.ValueOf(active).MethodByName(getterFunc).Call(nil)[0]
			newValue := v.Convert(reflect.TypeOf(0.0)).Float()
			if n.Value() != newValue {
				n.SetValue(newValue)
			}
		}
		rightType := func(t reflect.Type) bool {
			return t.Kind() == reflect.Float32 || t.Kind() == reflect.Float64
		}
		return uiProp{
			panel:     p,
			setter:    setterFunc,
			update:    update,
			rightType: rightType,
		}
	}

	stringPanel := func(parent wui.Container, name string) (*wui.EditLine, *wui.Panel) {
		t := wui.NewEditLine()
		t.SetOnTabFocus(t.SelectAll)
		t.SetBounds(100, propMargin, 90, 22)
		l := wui.NewLabel()
		l.SetText(name)
		l.SetAlignment(wui.AlignRight)
		l.SetBounds(0, propMargin-1, 95, t.Height())
		p := wui.NewPanel()
		p.SetSize(195, t.Height()+2*propMargin)
		parent.Add(p)
		p.Add(l)
		p.Add(t)
		return t, p
	}

	stringProp := func(name, getterFunc string) uiProp {
		t, p := stringPanel(w, name)
		setterFunc := "Set" + getterFunc
		t.SetOnTextChange(func() {
			if active == nil {
				return
			}
			if _, ok := reflect.TypeOf(active).MethodByName(setterFunc); ok {
				reflect.ValueOf(active).MethodByName(setterFunc).Call(
					[]reflect.Value{reflect.ValueOf(t.Text())},
				)
				updateProperties()
				preview.Paint()
				markModified()
			}
		})
		update := func() {
			text := reflect.ValueOf(active).MethodByName(getterFunc).Call(nil)[0].String()
			if t.Text() != text {
				t.SetText(text)
			}
		}
		rightType := func(t reflect.Type) bool {
			return t.Kind() == reflect.String
		}
		return uiProp{
			panel:     p,
			setter:    setterFunc,
			update:    update,
			rightType: rightType,
		}
	}

	stringListProp := func(name, getterFunc string) uiProp {
		setterFunc := "Set" + getterFunc
		l := wui.NewLabel()
		l.SetBounds(10, 5, 180, 13)
		l.SetText(name)
		l.SetAlignment(wui.AlignCenter)
		list := wui.NewTextEdit()
		list.SetBounds(10, 20, 180, 80)
		list.SetAnchors(wui.AnchorMinAndMax, wui.AnchorMinAndMax)
		p := wui.NewPanel()
		p.SetSize(195, list.Height()+2*propMargin)
		w.Add(p)
		p.Add(l)
		p.Add(list)
		list.SetOnTextChange(func() {
			if active == nil {
				return
			}
			if _, ok := reflect.TypeOf(active).MethodByName(setterFunc); ok {
				items := strings.Split(list.Text(), "\r\n")
				items = removeEmptyStrings(items)
				l.SetText(fmt.Sprintf("%s (%d)", name, len(items)))
				reflect.ValueOf(active).MethodByName(setterFunc).Call(
					[]reflect.Value{reflect.ValueOf(items)},
				)
				start, end := list.CursorPosition()
				updateProperties()
				list.SetSelection(start, end)
				preview.Paint()
				markModified()
			}
		})
		update := func() {
			items := reflect.ValueOf(active).MethodByName(getterFunc).Call(nil)[0].Interface().([]string)
			l.SetText(fmt.Sprintf("%s (%d)", name, len(items)))
			newText := strings.Join(items, "\r\n") + "\r\n"
			if list.Text() != newText {
				list.SetText(newText)
			}
		}
		rightType := func(t reflect.Type) bool {
			return t.Kind() == reflect.Slice
		}
		return uiProp{
			panel:     p,
			setter:    setterFunc,
			update:    update,
			rightType: rightType,
		}
	}

	enumProp := func(name, getterFunc string, enumNames ...string) uiProp {
		setterFunc := "Set" + getterFunc
		c := wui.NewComboBox()
		for _, name := range enumNames {
			c.AddItem(name)
		}
		c.SetBounds(100, propMargin, 90, 22)
		l := wui.NewLabel()
		l.SetText(name)
		l.SetAlignment(wui.AlignRight)
		l.SetBounds(0, propMargin-1, 95, c.Height())
		p := wui.NewPanel()
		p.SetSize(195, c.Height()+2*propMargin)
		w.Add(p)
		p.Add(l)
		p.Add(c)
		c.SetOnChange(func(index int) {
			m, ok := reflect.TypeOf(active).MethodByName(setterFunc)
			if ok {
				reflect.ValueOf(active).MethodByName(setterFunc).Call(
					[]reflect.Value{reflect.ValueOf(index).Convert(m.Type.In(1))},
				)
				updateProperties()
				preview.Paint()
				markModified()
			}
		})
		update := func() {
			v := reflect.ValueOf(active).MethodByName(getterFunc).Call(nil)[0]
			index := int(v.Convert(reflect.TypeOf(0)).Int())
			if c.SelectedIndex() != index {
				c.SetSelectedIndex(index)
			}
		}
		rightType := func(t reflect.Type) bool {
			return true
		}
		return uiProp{
			panel:     p,
			setter:    setterFunc,
			update:    update,
			rightType: rightType,
		}
	}

	colorProp := func(name, getterFunc string) uiProp {
		setterFunc := "Set" + getterFunc
		l := wui.NewLabel()
		l.SetText(name)
		l.SetAlignment(wui.AlignRight)
		l.SetBounds(0, propMargin-1, 95, 22)
		swatch := wui.NewPaintBox()
		swatch.SetBounds(100, propMargin, 48, 22)
		reset := wui.NewButton()
		reset.SetText("Reset")
		reset.SetBounds(152, propMargin, 40, 22)
		p := wui.NewPanel()
		p.SetSize(195, 22+2*propMargin)
		w.Add(p)
		p.Add(l)
		p.Add(swatch)
		p.Add(reset)
		current := wui.Color(0)
		isSet := true
		swatch.SetOnPaint(func(c *wui.Canvas) {
			cw, ch := c.Size()
			c.FillRect(0, 0, cw, ch, wui.RGB(240, 240, 240))
			if isSet {
				c.FillRect(2, 2, cw-4, ch-4, current)
			} else {
				c.TextOut(6, 4, "auto", wui.RGB(90, 90, 90))
			}
			c.DrawRect(0, 0, cw, ch, wui.RGB(100, 100, 100))
		})
		swatch.SetOnMouseDown(func(x, y int, button wui.MouseButton) {
			if active == nil {
				return
			}
			m := reflect.ValueOf(active).MethodByName(setterFunc)
			if !m.IsValid() {
				return
			}
			dlg := wui.NewColorDialog()
			dlg.SetColor(current)
			if dlg.Execute(w) {
				m.Call([]reflect.Value{reflect.ValueOf(dlg.Color())})
				updateProperties()
				preview.Paint()
				markModified()
			}
		})
		reset.SetOnClick(func() {
			if active == nil {
				return
			}
			v := reflect.ValueOf(active)
			if m := v.MethodByName("Reset" + getterFunc); m.IsValid() {
				m.Call(nil)
			} else if getterFunc == "TransparentColor" {
				v.MethodByName("SetTransparent").Call([]reflect.Value{reflect.ValueOf(false)})
			} else {
				return
			}
			updateProperties()
			preview.Paint()
			markModified()
		})
		update := func() {
			v := reflect.ValueOf(active)
			current = wui.Color(v.MethodByName(getterFunc).Call(nil)[0].Uint())
			isSet = true
			canReset := false
			if m := v.MethodByName("Has" + getterFunc); m.IsValid() {
				isSet = m.Call(nil)[0].Bool()
			}
			if v.MethodByName("Reset" + getterFunc).IsValid() {
				canReset = true
			}
			if getterFunc == "TransparentColor" {
				isSet = v.MethodByName("Transparent").Call(nil)[0].Bool()
				canReset = true
			}
			reset.SetEnabled(isSet && canReset)
			swatch.Paint()
		}
		rightType := func(t reflect.Type) bool {
			return t.Kind() == reflect.Uint32
		}
		return uiProp{
			panel:     p,
			setter:    setterFunc,
			getter:    getterFunc,
			update:    update,
			rightType: rightType,
		}
	}

	// Editors for the extra properties, which the designer keeps itself, see
	// extras.go.
	hasExtra := func(name string) func(c interface{}) bool {
		return func(c interface{}) bool {
			_, ok := findExtra(c, name)
			return ok
		}
	}
	// extraEdited is called after the value of an extra property changed.
	extraEdited := func() {
		preview.Paint()
		markModified()
	}

	exBoolProp := func(label, name string) uiProp {
		c, p := boolPanel(w, label)
		c.SetOnChange(func(on bool) {
			if updatingProps || active == nil {
				return
			}
			if s, ok := findExtra(active, name); ok {
				setExtra(active, s, strconv.FormatBool(on))
				extraEdited()
			}
		})
		update := func() {
			s, ok := findExtra(active, name)
			if !ok {
				return
			}
			on := getExtra(active, s) == "true"
			if c.Checked() != on {
				updatingProps = true
				c.SetChecked(on)
				updatingProps = false
			}
		}
		return uiProp{panel: p, setter: "Set" + name, update: update, visibleFor: hasExtra(name), isExtra: true}
	}

	exStringProp := func(label, name string) uiProp {
		t, p := stringPanel(w, label)
		t.SetOnTextChange(func() {
			if updatingProps || active == nil {
				return
			}
			if s, ok := findExtra(active, name); ok {
				setExtra(active, s, t.Text())
				extraEdited()
			}
		})
		update := func() {
			s, ok := findExtra(active, name)
			if !ok {
				return
			}
			if v := getExtra(active, s); t.Text() != v {
				updatingProps = true
				t.SetText(v)
				updatingProps = false
			}
		}
		return uiProp{panel: p, setter: "Set" + name, update: update, visibleFor: hasExtra(name), isExtra: true}
	}

	exIntProp := func(label, name string) uiProp {
		spec, _ := anyExtraSpec(name)
		n, p := intPanel(w, label, spec.min, spec.max)
		n.SetOnValueChange(func(v int) {
			if updatingProps || active == nil {
				return
			}
			if s, ok := findExtra(active, name); ok {
				setExtra(active, s, strconv.Itoa(v))
				extraEdited()
			}
		})
		update := func() {
			if _, ok := findExtra(active, name); !ok {
				return
			}
			nums := extraInts(active, name)
			if len(nums) > 0 && n.Value() != nums[0] {
				updatingProps = true
				n.SetValue(nums[0])
				updatingProps = false
			}
		}
		return uiProp{panel: p, setter: "Set" + name, update: update, visibleFor: hasExtra(name), isExtra: true}
	}

	// exIntsProp edits two numbers, for example the smallest window size.
	exIntsProp := func(label, name string) uiProp {
		l := wui.NewLabel()
		l.SetText(label)
		l.SetAlignment(wui.AlignRight)
		a := wui.NewIntUpDown()
		a.SetMinMax(-1000000, 1000000)
		a.SetBounds(100, propMargin, 43, 22)
		b := wui.NewIntUpDown()
		b.SetMinMax(-1000000, 1000000)
		b.SetBounds(147, propMargin, 43, 22)
		l.SetBounds(0, propMargin-1, 95, a.Height())
		p := wui.NewPanel()
		p.SetSize(195, a.Height()+2*propMargin)
		w.Add(p)
		p.Add(l)
		p.Add(a)
		p.Add(b)
		changed := func(int) {
			if updatingProps || active == nil {
				return
			}
			if s, ok := findExtra(active, name); ok {
				setExtra(active, s, fmt.Sprintf("%d,%d", a.Value(), b.Value()))
				extraEdited()
			}
		}
		a.SetOnValueChange(changed)
		b.SetOnValueChange(changed)
		update := func() {
			if _, ok := findExtra(active, name); !ok {
				return
			}
			nums := extraInts(active, name)
			if len(nums) < 2 {
				return
			}
			updatingProps = true
			if a.Value() != nums[0] {
				a.SetValue(nums[0])
			}
			if b.Value() != nums[1] {
				b.SetValue(nums[1])
			}
			updatingProps = false
		}
		return uiProp{panel: p, setter: "Set" + name, update: update, visibleFor: hasExtra(name), isExtra: true}
	}

	exEnumProp := func(label, name string) uiProp {
		spec, _ := anyExtraSpec(name)
		c := wui.NewComboBox()
		for _, item := range spec.enum {
			c.AddItem(item)
		}
		c.SetBounds(100, propMargin, 90, 22)
		l := wui.NewLabel()
		l.SetText(label)
		l.SetAlignment(wui.AlignRight)
		l.SetBounds(0, propMargin-1, 95, c.Height())
		p := wui.NewPanel()
		p.SetSize(195, c.Height()+2*propMargin)
		w.Add(p)
		p.Add(l)
		p.Add(c)
		c.SetOnChange(func(index int) {
			if updatingProps || active == nil {
				return
			}
			if s, ok := findExtra(active, name); ok && index >= 0 && index < len(s.enum) {
				setExtra(active, s, s.enum[index])
				extraEdited()
			}
		})
		update := func() {
			s, ok := findExtra(active, name)
			if !ok {
				return
			}
			v := getExtra(active, s)
			for i, item := range s.enum {
				if item == v && c.SelectedIndex() != i {
					updatingProps = true
					c.SetSelectedIndex(i)
					updatingProps = false
				}
			}
		}
		return uiProp{panel: p, setter: "Set" + name, update: update, visibleFor: hasExtra(name), isExtra: true}
	}

	exColorProp := func(label, name string) uiProp {
		l := wui.NewLabel()
		l.SetText(label)
		l.SetAlignment(wui.AlignRight)
		l.SetBounds(0, propMargin-1, 95, 22)
		swatch := wui.NewPaintBox()
		swatch.SetBounds(100, propMargin, 48, 22)
		reset := wui.NewButton()
		reset.SetText("Reset")
		reset.SetBounds(152, propMargin, 40, 22)
		p := wui.NewPanel()
		p.SetSize(195, 22+2*propMargin)
		w.Add(p)
		p.Add(l)
		p.Add(swatch)
		p.Add(reset)
		var cr, cg, cb uint8
		isSet := false
		swatch.SetOnPaint(func(c *wui.Canvas) {
			sw, sh := c.Size()
			c.FillRect(0, 0, sw, sh, wui.RGB(240, 240, 240))
			if isSet {
				c.FillRect(2, 2, sw-4, sh-4, wui.RGB(cr, cg, cb))
			} else {
				c.TextOut(6, 4, "auto", wui.RGB(90, 90, 90))
			}
			c.DrawRect(0, 0, sw, sh, wui.RGB(100, 100, 100))
		})
		swatch.SetOnMouseDown(func(x, y int, button wui.MouseButton) {
			if active == nil {
				return
			}
			s, ok := findExtra(active, name)
			if !ok {
				return
			}
			dlg := wui.NewColorDialog()
			dlg.SetColor(wui.RGB(cr, cg, cb))
			if dlg.Execute(w) {
				c := dlg.Color()
				setExtra(active, s, fmt.Sprintf("#%02X%02X%02X", c.R(), c.G(), c.B()))
				updateProperties()
				extraEdited()
			}
		})
		reset.SetOnClick(func() {
			if active == nil {
				return
			}
			if s, ok := findExtra(active, name); ok {
				setExtra(active, s, s.def)
				updateProperties()
				extraEdited()
			}
		})
		update := func() {
			if _, ok := findExtra(active, name); !ok {
				return
			}
			cr, cg, cb, isSet = extraColor(active, name)
			reset.SetEnabled(isSet)
			swatch.Paint()
		}
		return uiProp{panel: p, setter: "Set" + name, update: update, visibleFor: hasExtra(name), isExtra: true}
	}

	// exListProp edits a list with one line per entry.
	exListProp := func(label, name string) uiProp {
		l := wui.NewLabel()
		l.SetBounds(10, 5, 180, 13)
		l.SetText(label)
		l.SetAlignment(wui.AlignCenter)
		list := wui.NewTextEdit()
		list.SetBounds(10, 20, 180, 80)
		list.SetWritesTabs(true)
		p := wui.NewPanel()
		p.SetSize(195, list.Height()+2*propMargin+18)
		w.Add(p)
		p.Add(l)
		p.Add(list)
		list.SetOnTextChange(func() {
			if updatingProps || active == nil {
				return
			}
			s, ok := findExtra(active, name)
			if !ok {
				return
			}
			var items []string
			for _, line := range strings.Split(strings.Replace(list.Text(), "\r", "", -1), "\n") {
				if strings.TrimSpace(line) != "" {
					items = append(items, line)
				}
			}
			l.SetText(fmt.Sprintf("%s (%d)", label, len(items)))
			setExtra(active, s, strings.Join(items, "\n"))
			extraEdited()
		})
		update := func() {
			if _, ok := findExtra(active, name); !ok {
				return
			}
			items := extraLines(active, name)
			l.SetText(fmt.Sprintf("%s (%d)", label, len(items)))
			text := strings.Join(items, "\r\n")
			if len(items) > 0 {
				text += "\r\n"
			}
			if list.Text() != text {
				updatingProps = true
				list.SetText(text)
				updatingProps = false
			}
		}
		return uiProp{panel: p, setter: "Set" + name, update: update, visibleFor: hasExtra(name), isExtra: true}
	}

	// exFileProp edits the path of a file with a button that opens the file
	// dialog.
	exFileProp := func(label, name string) uiProp {
		t, p := stringPanel(w, label)
		t.SetWidth(62)
		browse := wui.NewButton()
		browse.SetText("...")
		browse.SetBounds(164, propMargin, 26, 22)
		p.Add(browse)
		t.SetOnTextChange(func() {
			if updatingProps || active == nil {
				return
			}
			if s, ok := findExtra(active, name); ok {
				setExtra(active, s, t.Text())
				extraEdited()
			}
		})
		browse.SetOnClick(func() {
			if active == nil {
				return
			}
			open := wui.NewFileOpenDialog()
			open.SetTitle("Choose an image")
			open.AddFilter("Images", ".png", ".jpg", ".jpeg", ".gif")
			open.AddFilter("All files", "*")
			if accept, path := open.ExecuteSingleSelection(w); accept {
				t.SetText(path)
			}
		})
		update := func() {
			s, ok := findExtra(active, name)
			if !ok {
				return
			}
			if v := getExtra(active, s); t.Text() != v {
				updatingProps = true
				t.SetText(v)
				updatingProps = false
			}
		}
		return uiProp{panel: p, setter: "Set" + name, update: update, visibleFor: hasExtra(name), isExtra: true}
	}

	// only limits a property to the controls for which f is true.
	only := func(p uiProp, f func(c interface{}) bool) uiProp {
		p.visibleFor = f
		return p
	}
	inGroup := func(p uiProp, g string) uiProp {
		p.group = g
		return p
	}
	isWindow := func(c interface{}) bool { _, ok := c.(*wui.Window); return ok }
	isProgress := func(c interface{}) bool { _, ok := c.(*wui.ProgressBar); return ok }
	isDate := func(c interface{}) bool { _, ok := c.(*wui.DatePicker); return ok }
	isImage := func(c interface{}) bool { _, ok := c.(*wui.ImageView); return ok }
	isCombo := func(c interface{}) bool { _, ok := c.(*wui.ComboBox); return ok }
	isTabs := func(c interface{}) bool { _, ok := c.(*wui.TabControl); return ok }

	uiProps := []uiProp{
		stringProp("Title", "Title"),
		stringProp("Text", "Text"),
		only(enumProp("Window State", "State", "Normal", "Maximized", "Minimized"), isWindow),
		inGroup(only(enumProp("State", "State", "Normal", "Error", "Paused"), isProgress), "Appearance"),
		boolProp("Center On Show", "CenterOnShow"),
		exIntsProp("Min Size", "MinSize"),
		exIntsProp("Max Size", "MaxSize"),
		boolProp("Min Button", "HasMinButton"),
		boolProp("Max Button", "HasMaxButton"),
		boolProp("Close Button", "HasCloseButton"),
		boolProp("Has Border", "HasBorder"),
		boolProp("Resizable", "Resizable"),
		intProp("Alpha", "Alpha", 0, 255),
		colorProp("Background", "BackgroundColor"),
		boolProp("Top Most", "TopMost"),
		boolProp("Taskbar", "ShowInTaskbar"),
		boolProp("Drag Move", "DragByBackground"),
		boolProp("Accept Drop", "AcceptFiles"),
		boolProp("Transparent", "Transparent"),
		colorProp("Key Color", "TransparentColor"),
		intProp("Corner Radius", "CornerRadius", 0, 200),
		boolProp("Tray Icon", "TrayEnabled"),
		stringProp("Tray Tooltip", "TrayToolTip"),
		boolProp("Min To Tray", "MinimizeToTray"),
		boolProp("Close To Tray", "CloseToTray"),
		colorProp("Text Color", "TextColor"),
		boolProp("Enabled", "Enabled"),
		boolProp("Visible", "Visible"),
		enumProp("Horizontal Anchor", "HorizontalAnchor",
			"Left", "Right", "Center", "Left+Right", "Left+Center", "Right+Center"),
		enumProp("Vertical Anchor", "VerticalAnchor",
			"Top", "Bottom", "Center", "Top+Bottom", "Top+Center", "Bottom+Center"),
		intProp("X", "X"),
		intProp("Y", "Y"),
		intProp("Width", "Width"),
		intProp("Height", "Height"),
		intProp("Inner X", "InnerX"),
		intProp("Inner Y", "InnerY"),
		intProp("Inner Width", "InnerWidth"),
		intProp("Inner Height", "InnerHeight"),
		enumProp("Alignment", "Alignment", "Left", "Center", "Right"),
		boolProp("Checked", "Checked"),
		intProp("Arrow Increment", "ArrowIncrement"),
		intProp("Mouse Increment", "MouseIncrement"),
		intProp("Min", "Min"),
		intProp("Max", "Max"),
		intProp("Value", "Value"),
		floatProp("Min", "Min"),
		floatProp("Max", "Max"),
		floatProp("Value", "Value"),
		intProp("Cursor Position", "CursorPosition"),
		intProp("Precision", "Precision", 1, 6),
		enumProp("Orientation", "Orientation", "Horizontal", "Vertical"),
		enumProp("Tick Position", "TickPosition", "Right/Bottom", "Left/Top", "Both Sides"),
		intProp("Tick Frequency", "TickFrequency"),
		boolProp("Ticks Visible", "TicksVisible"),
		enumProp("Border Style", "BorderStyle", "None", "Single Line", "Sunken", "Sunken Thick", "Raised"),
		intProp("Character Limit", "CharacterLimit", 1, 0x7FFFFFFE),
		boolProp("Is Password", "IsPassword"),
		boolProp("Read Only", "ReadOnly"),
		boolProp("Writes Tabs", "WritesTabs"),
		only(stringListProp("Items", "Items"), isCombo),
		only(stringListProp("Tabs", "Tabs"), isTabs),
		only(enumProp("Date Mode", "Mode", "Short Date", "Long Date", "Time"), isDate),
		only(enumProp("Image Mode", "Mode", "Normal", "Center", "Stretch", "Fit", "Fill"), isImage),
		enumProp("List View", "View", "Details", "List", "Icons", "Small Icons", "Tiles"),
		boolProp("Multi Select", "MultiSelect"),
		boolProp("Three State", "ThreeState"),
		boolProp("Editable", "Editable"),
		boolProp("Tab Stop", "TabStop"),
		exEnumProp("Kind", "Kind"),
		exStringProp("Note", "Note"),
		exBoolProp("Default Button", "Default"),
		exBoolProp("Push Like", "PushLike"),
		exBoolProp("Numbers Only", "NumbersOnly"),
		exEnumProp("Text Align", "TextAlign"),
		exBoolProp("Header Visible", "HeaderVisible"),
		exBoolProp("Full Row Select", "FullRowSelect"),
		exBoolProp("Grid Lines", "GridLines"),
		exBoolProp("Check Boxes", "CheckBoxes"),
		exBoolProp("Editable", "Editable"),
		exBoolProp("Sortable", "Sortable"),
		exBoolProp("Word Wrap", "WordWrap"),
		exBoolProp("Detect Links", "AutoDetectLinks"),
		exBoolProp("Writes Tabs", "WritesTabs"),
		exBoolProp("Week Numbers", "ShowWeekNumbers"),
		exBoolProp("Vertical", "Vertical"),
		exIntsProp("Range", "Range"),
		exIntProp("Page", "Page"),
		exColorProp("Background", "BackgroundColor"),
		exColorProp("Back Color", "BackColor"),
		exFileProp("Image File", "ImageFile"),
		exListProp("Columns", "Columns"),
		exListProp("Rows", "Items"),
		exListProp("Nodes", "Nodes"),
		exStringProp("Tool Tip", "ToolTip"),
		intProp("Selected Index", "SelectedIndex", -1, math.MaxInt32),
		boolProp("Vertical", "Vertical"),
		boolProp("Moves Forever", "MovesForever"),
		boolProp("Word Wrap", "WordWrap"),
	}

	// Font properties panel
	fontProps := wui.NewPanel()
	w.Add(fontProps)
	useParentFont, useParentFontPanel := boolPanel(fontProps, "Use Parent Font")
	fontName, fontNamePanel := stringPanel(fontProps, "Name")
	fontName.SetCharacterLimit(31)
	fontHeight, fontHeightPanel := intPanel(fontProps, "Height")
	fontBold, fontBoldPanel := boolPanel(fontProps, "Bold")
	fontBold.SetFont(bold)
	fontItalic, fontItalicPanel := boolPanel(fontProps, "Italic")
	fontItalic.SetFont(italic)
	fontUnderlined, fontUnderlinedPanel := boolPanel(fontProps, "Underlined")
	fontUnderlined.SetFont(underlined)
	fontStrikedOut, fontStrikedOutPanel := boolPanel(fontProps, "StrikedOut")
	fontStrikedOut.SetFont(strikedOut)
	for _, p := range []*wui.Panel{
		useParentFontPanel, fontNamePanel, fontHeightPanel,
		fontBoldPanel, fontItalicPanel, fontUnderlinedPanel, fontStrikedOutPanel,
	} {
		p.SetX(p.X() - 40)
	}
	fontProps.SetBorderStyle(wui.PanelBorderSunken)
	{
		fontLabel := wui.NewLabel()
		fontLabel.SetText("Font")
		fontLabel.SetY(propMargin + 5)
		fontLabel.SetHeight(13)
		fontLabel.SetAlignment(wui.AlignCenter)
		fontLabel.SetFont(bold)
		fontProps.Add(fontLabel)

		y := fontLabel.Y() + fontLabel.Height() + 10
		for _, panel := range []*wui.Panel{
			useParentFontPanel, fontNamePanel, fontHeightPanel,
			fontBoldPanel, fontItalicPanel, fontUnderlinedPanel, fontStrikedOutPanel,
		} {
			panel.SetY(y)
			y += panel.Height()
		}
		fontProps.SetBounds(15, 0, 175, y+5)
		fontLabel.SetWidth(fontProps.InnerWidth())
	}
	updateFont := func() {
		f, ok := active.(fonter)
		if !ok {
			return
		}
		useParent := useParentFont.Checked()
		fontName.SetEnabled(!useParent)
		fontHeight.SetEnabled(!useParent)
		fontBold.SetEnabled(!useParent)
		fontItalic.SetEnabled(!useParent)
		fontUnderlined.SetEnabled(!useParent)
		fontStrikedOut.SetEnabled(!useParent)
		if useParent {
			f.SetFont(nil)
		} else {
			font, err := wui.NewFont(wui.FontDesc{
				Name:       fontName.Text(),
				Height:     fontHeight.Value(),
				Bold:       fontBold.Checked(),
				Italic:     fontItalic.Checked(),
				Underlined: fontUnderlined.Checked(),
				StrikedOut: fontStrikedOut.Checked(),
			})
			if err == nil {
				f.SetFont(font)
			}
		}
		preview.Paint()
		markModified()
	}
	useParentFont.SetOnChange(func(disable bool) { updateFont() })
	fontName.SetOnTextChange(func() { updateFont() })
	fontHeight.SetOnValueChange(func(int) { updateFont() })
	fontBold.SetOnChange(func(bool) { updateFont() })
	fontItalic.SetOnChange(func(bool) { updateFont() })
	fontUnderlined.SetOnChange(func(bool) { updateFont() })
	fontStrikedOut.SetOnChange(func(bool) { updateFont() })

	// App icon setup
	appIcon := w32.LoadIcon(0, w32.MakeIntResource(w32.IDI_APPLICATION))
	appIconWidth := w32.GetSystemMetrics(w32.SM_CXICON)
	appIconHeight := w32.GetSystemMetrics(w32.SM_CYICON)
	appIconWidth, appIconHeight = 17, 17

	defaultCursor := w.Cursor()

	// Sliders for resizing panels
	leftSlider := wui.NewPanel()
	leftSlider.SetBounds(195, -1, 5, 602)
	leftSlider.SetBorderStyle(wui.PanelBorderSingleLine)
	leftSlider.SetVerticalAnchor(wui.AnchorMinAndMax)
	w.Add(leftSlider)

	rightSlider := wui.NewPanel()
	rightSlider.SetBounds(600, -1, 5, 602)
	rightSlider.SetBorderStyle(wui.PanelBorderSingleLine)
	rightSlider.SetVerticalAnchor(wui.AnchorMinAndMax)
	rightSlider.SetHorizontalAnchor(wui.AnchorMax)
	w.Add(rightSlider)

	// Control templates
	panelTemplate := wui.NewPanel()
	panelTemplate.SetBounds(20, 10, 150, 50)
	panelTemplate.SetBorderStyle(wui.PanelBorderSingleLine)
	panelText := wui.NewLabel()
	panelText.SetText("Panel")
	panelText.SetAlignment(wui.AlignCenter)
	panelText.SetSize(panelTemplate.InnerWidth(), panelTemplate.InnerHeight())
	panelTemplate.Add(panelText)

	paintBoxTemplate := wui.NewPaintBox()
	paintBoxTemplate.SetBounds(20, 67, 150, 50)

	textEditTemplate := wui.NewTextEdit()
	textEditTemplate.SetBounds(20, 124, 150, 50)
	textEditTemplate.SetText("Text Edit")

	editLineTemplate := wui.NewEditLine()
	editLineTemplate.SetBounds(20, 181, 150, 20)
	editLineTemplate.SetText("Text Edit Line")

	comboTemplate := wui.NewComboBox()
	comboTemplate.SetBounds(20, 210, 150, 21)
	comboTemplate.AddItem("Combo Box")
	comboTemplate.SetSelectedIndex(0)

	sliderTemplate := wui.NewSlider()
	sliderTemplate.SetBounds(20, 245, 150, 45)

	progressTemplate := wui.NewProgressBar()
	progressTemplate.SetBounds(20, 295, 150, 25)
	progressTemplate.SetValue(0.5)

	buttonTemplate := wui.NewButton()
	buttonTemplate.SetText("Button")
	buttonTemplate.SetBounds(20, 329, 85, 25)

	intTemplate := wui.NewIntUpDown()
	intTemplate.SetBounds(20, 362, 80, 22)

	floatTemplate := wui.NewFloatUpDown()
	floatTemplate.SetBounds(20, 392, 80, 22)

	checkBoxTemplate := wui.NewCheckBox()
	checkBoxTemplate.SetText("Check Box")
	checkBoxTemplate.SetChecked(true)
	checkBoxTemplate.SetBounds(20, 423, 100, 17)

	radioButtonTemplate := wui.NewRadioButton()
	radioButtonTemplate.SetText("Radio Button")
	radioButtonTemplate.SetChecked(true)
	radioButtonTemplate.SetBounds(20, 448, 100, 17)

	labelTemplate := wui.NewLabel()
	labelTemplate.SetText("Text Label")
	labelTemplate.SetBounds(20, 473, 150, 13)

	groupBoxTemplate := wui.NewGroupBox()
	groupBoxTemplate.SetText("Group Box")
	groupBoxTemplate.SetBounds(20, 495, 150, 50)

	tabTemplate := wui.NewTabControl()
	tabTemplate.SetTabs([]string{"Tab 1", "Tab 2"})
	tabTemplate.SetBounds(20, 553, 150, 60)

	dateTemplate := wui.NewDatePicker()
	dateTemplate.SetBounds(20, 620, 150, 22)

	statusTemplate := wui.NewStatusBar()
	statusTemplate.SetText("Status")
	statusTemplate.SetBounds(20, 650, 150, 22)

	listViewTemplate := wui.NewListView()
	listViewTemplate.SetBounds(20, 680, 150, 80)
	setExtraByName(listViewTemplate, "Columns", "Name\nSize")
	setExtraByName(listViewTemplate, "Items", "Item 1\nItem 2")

	richEditTemplate := wui.NewRichEdit()
	richEditTemplate.SetText("Rich Edit")
	richEditTemplate.SetBounds(20, 770, 150, 50)

	linkLabelTemplate := wui.NewLinkLabel()
	linkLabelTemplate.SetText("<a href=\"https://example.com\">Link Label</a>")
	linkLabelTemplate.SetBounds(20, 830, 150, 17)

	monthCalendarTemplate := wui.NewMonthCalendar()
	monthCalendarTemplate.SetBounds(20, 855, 190, 160)

	hotKeyTemplate := wui.NewHotKeyEdit()
	hotKeyTemplate.SetBounds(20, 1020, 150, 22)

	ipTemplate := wui.NewIPAddressEdit()
	ipTemplate.SetBounds(20, 1050, 150, 22)

	imageViewTemplate := wui.NewImageView()
	imageViewTemplate.SetBounds(20, 1080, 100, 60)

	scrollPanelTemplate := wui.NewScrollPanel()
	scrollPanelTemplate.SetBounds(20, 1150, 150, 60)
	scrollPanelTemplate.SetBorderStyle(wui.PanelBorderSingleLine)

	scrollBarTemplate := wui.NewScrollBar(false)
	scrollBarTemplate.SetBounds(20, 1220, 150, 17)

	treeViewTemplate := wui.NewTreeView()
	treeViewTemplate.SetBounds(20, 1245, 150, 80)
	setExtraByName(treeViewTemplate, "Nodes", "Root\n  Child 1\n  Child 2")

	// The toolbox is split into collapsible groups so everything stays
	// reachable on small screens.
	type paletteGroup struct {
		title string
		items []wui.Control
		open  bool
	}
	paletteGroups := []*paletteGroup{
		{"Containers", []wui.Control{panelTemplate, groupBoxTemplate, tabTemplate, scrollPanelTemplate, statusTemplate}, true},
		{"Text", []wui.Control{textEditTemplate, editLineTemplate, richEditTemplate}, false},
		{"Input", []wui.Control{comboTemplate, intTemplate, floatTemplate, dateTemplate, monthCalendarTemplate, hotKeyTemplate, ipTemplate}, false},
		{"Buttons", []wui.Control{buttonTemplate, checkBoxTemplate, radioButtonTemplate, linkLabelTemplate}, false},
		{"Display", []wui.Control{labelTemplate, paintBoxTemplate, imageViewTemplate, progressTemplate, sliderTemplate, scrollBarTemplate}, false},
		{"Lists", []wui.Control{listViewTemplate, treeViewTemplate}, false},
	}
	var visibleTemplates []wui.Control
	var paletteHeaders []rectangle
	paletteScroll := 0

	var highlightedTemplate, controlToAdd wui.Control
	var templateDx, templateDy int

	palette := wui.NewPaintBox()
	palette.SetBounds(605, 0, 195, 600)
	palette.SetHorizontalAnchor(wui.AnchorMax)
	palette.SetVerticalAnchor(wui.AnchorMinAndMax)
	palette.SetOnPaint(func(c *wui.Canvas) {
		w, h := c.Size()
		c.FillRect(0, 0, w, h, wui.RGB(240, 240, 240))
		for i, g := range paletteGroups {
			if i >= len(paletteHeaders) {
				break
			}
			r := paletteHeaders[i]
			c.FillRect(r.x, r.y, r.w, r.h, wui.RGB(215, 225, 240))
			c.DrawRect(r.x, r.y, r.w, r.h, wui.RGB(150, 160, 180))
			sign := "+ "
			if g.open {
				sign = "- "
			}
			c.TextOut(r.x+8, r.y+4, sign+g.title, wui.RGB(0, 0, 0))
		}
		for _, template := range visibleTemplates {
			drawControl(template, c)
		}
		if highlightedTemplate != nil {
			x, y, w, h := highlightedTemplate.Bounds()
			c.DrawRect(x-1, y-1, w+2, h+2, wui.RGB(255, 0, 255))
			c.DrawRect(x-2, y-2, w+4, h+4, wui.RGB(255, 0, 255))
		}
	})
	palette.SetOnMouseMove(func(x, y int) {
		oldHighlight := highlightedTemplate
		highlightedTemplate = nil
		for _, c := range visibleTemplates {
			if contains(c, x, y) {
				highlightedTemplate = c
			}
		}
		if highlightedTemplate != oldHighlight {
			palette.Paint()
		}
	})
	w.Add(palette)

	layoutPalette := func() {
		total := 4
		for _, g := range paletteGroups {
			total += 26
			if g.open {
				for _, t := range g.items {
					_, _, _, th := t.Bounds()
					total += th + 6
				}
				total += 4
			}
		}
		paletteScroll = max(0, min(paletteScroll, total-palette.Height()))
		y := 4 - paletteScroll
		pw := palette.Width()
		visibleTemplates = visibleTemplates[:0]
		paletteHeaders = paletteHeaders[:0]
		for _, g := range paletteGroups {
			paletteHeaders = append(paletteHeaders, rect(4, y, pw-8, 22))
			y += 26
			if g.open {
				for _, t := range g.items {
					tx, _, tw, th := t.Bounds()
					t.SetBounds(tx, y, tw, th)
					visibleTemplates = append(visibleTemplates, t)
					y += th + 6
				}
				y += 4
			}
		}
		highlightedTemplate = nil
		palette.Paint()
	}

	// Variable name editor
	nameText := wui.NewLabel()
	nameText.SetText("Variable Name")
	nameText.SetAlignment(wui.AlignRight)
	nameText.SetBounds(10, 10, 85, 20)
	w.Add(nameText)
	name := wui.NewEditLine()
	name.SetBounds(100, 10, 90, 22)
	w.Add(name)

	// Preview area
	preview.SetBounds(200, 0, 400, 600)
	preview.SetHorizontalAnchor(wui.AnchorMinAndMax)
	preview.SetVerticalAnchor(wui.AnchorMinAndMax)
	white := wui.RGB(255, 255, 255)
	black := wui.RGB(0, 0, 0)

	// Events: a list of the events of the selected control and a button that
	// opens the code of the handler. Events that have code are marked with *.
	evCombo := wui.NewComboBox()
	evCombo.SetBounds(10, 4, 175, 22)
	evEdit := wui.NewButton()
	evEdit.SetText("Edit Code...")
	evEdit.SetBounds(10, 30, 175, 25)
	evPanel := wui.NewPanel()
	evPanel.SetBounds(0, 0, 195, 62)
	evPanel.Add(evCombo)
	evPanel.Add(evEdit)
	evPanel.SetVisible(false)
	w.Add(evPanel)
	var evNames []string
	var refreshEvents func()

	name.SetOnTextChange(func() {
		names[active] = name.Text()
		markModified()
	})
	

	// openEventEditor lets the user edit the Go code of an event handler.
	openEventEditor := func(evName, defaultCode string) {
		if active == nil {
			return
		}
		target := active
		dlg := wui.NewWindow()
		screenW := w32.GetSystemMetrics(w32.SM_CXSCREEN)
		screenH := w32.GetSystemMetrics(w32.SM_CYSCREEN)
		dlgW := min(max(preview.Width(), 480), screenW-40)
		dlgH := min(max(preview.Height(), 320), screenH-80)
		dlg.SetSize(dlgW, dlgH)
		dlg.SetMinSize(320, 220)
		dlg.SetCenterOnShow(true)
		dlg.SetTitle(evName + " - " + names[target])

		code := wui.NewTextEdit()
		font, _ := wui.NewFont(wui.FontDesc{Name: "Courier New", Height: -15})
		code.SetFont(font)
		code.SetWritesTabs(true)
		code.SetBounds(0, 0, dlg.InnerWidth(), dlg.InnerHeight()-30)
		code.SetAnchors(wui.AnchorMinAndMax, wui.AnchorMinAndMax)
		dlg.Add(code)

		ev := event{target, evName}
		current := events[ev]
		if current == "" {
			current = defaultCode
		}
		code.SetText(strings.Replace(current, "\n", "\r\n", -1))
		cursor := len(defaultCode) + 3
		if i := strings.Index(defaultCode, "\n"); i >= 0 {
			cursor = i + 3
		}
		code.SetCursorPosition(cursor)

		ok := wui.NewButton()
		ok.SetText("OK")
		ok.SetBounds(dlg.InnerWidth()/2-87, dlg.InnerHeight()-28, 85, 25)
		ok.SetAnchors(wui.AnchorCenter, wui.AnchorMax)
		ok.SetOnClick(func() {
			events[ev] = strings.Replace(code.Text(), "\r", "", -1)
			dlg.Close()
			markModified()
			if active == target {
				refreshEvents()
			}
		})
		dlg.Add(ok)

		cancel := wui.NewButton()
		cancel.SetText("Cancel")
		cancel.SetBounds(dlg.InnerWidth()/2+2, dlg.InnerHeight()-28, 85, 25)
		cancel.SetAnchors(wui.AnchorCenter, wui.AnchorMax)
		cancel.SetOnClick(dlg.Close)
		dlg.Add(cancel)

		dlg.SetOnShow(code.Focus)
		dlg.ShowModal()
	}

	refreshEvents = func() {
		evNames = evNames[:0]
		var items []string
		if active != nil {
			for _, ev := range eventsOf(active) {
				label := ev
				if !isEmptyHandler(events[event{control: active, name: ev}]) {
					label += " *"
				}
				evNames = append(evNames, ev)
				items = append(items, label)
			}
		}
		keep := evCombo.SelectedIndex()
		evCombo.SetItems(items)
		if len(items) > 0 {
			if keep < 0 || keep >= len(items) {
				keep = 0
			}
			evCombo.SetSelectedIndex(keep)
		}
	}
	evEdit.SetOnClick(func() {
		i := evCombo.SelectedIndex()
		if active == nil || i < 0 || i >= len(evNames) {
			return
		}
		openEventEditor(evNames[i], eventTemplate(active, evNames[i]))
	})

	updateProperties = func() {
		for _, prop := range uiProps {
			if prop.panel.Visible() {
				prop.update()
			}
		}
	}

	// Layout state of the side panels. The designer adapts to small screens:
	// the toolbox and the property panel can be hidden automatically or by
	// hand, both can be scrolled with the mouse wheel and the preview can be
	// panned.
	const (
		modeAuto = iota
		modeShown
		modeHidden
	)
	const sideWidth = 195
	var (
		propsMode, paletteMode   = modeAuto, modeAuto
		propsVisible             = true
		paletteVisible           = true
		propScroll               int
		propShown                = make([]bool, len(uiProps))
		fontShown                bool
		eventsShown              bool
		panX, panY               int
		layoutProps, layoutMain  func()
	)

	// Properties are shown in collapsible groups. Opening a group closes the
	// others, so every property can be reached on small screens.
	propGroups := []string{"General", "Layout", "Appearance", "Behavior", "Window Options", "Tray", "Font", "Events"}
	groupOf := func(setter string) string {
		switch setter {
		case "SetTitle", "SetText", "SetEnabled", "SetVisible", "SetChecked", "SetValue",
			"SetItems", "SetTabs", "SetSelectedIndex", "SetNote", "SetColumns", "SetNodes", "SetToolTip":
			return "General"
		case "SetMode", "SetView", "SetKind", "SetTextAlign", "SetBackColor", "SetGridLines",
			"SetHeaderVisible", "SetImageFile":
			return "Appearance"
		case "SetHorizontalAnchor", "SetVerticalAnchor", "SetX", "SetY", "SetWidth", "SetHeight",
			"SetInnerX", "SetInnerY", "SetInnerWidth", "SetInnerHeight":
			return "Layout"
		case "SetBackgroundColor", "SetTextColor", "SetAlignment", "SetBorderStyle", "SetVertical",
			"SetTickPosition", "SetOrientation", "SetTicksVisible", "SetTickFrequency", "SetAlpha":
			return "Appearance"
		case "SetState", "SetHasMinButton", "SetHasMaxButton", "SetHasCloseButton", "SetHasBorder",
			"SetResizable", "SetTopMost", "SetShowInTaskbar", "SetDragByBackground", "SetAcceptFiles",
			"SetTransparent", "SetTransparentColor", "SetCornerRadius", "SetCenterOnShow",
			"SetMinSize", "SetMaxSize":
			return "Window Options"
		case "SetTrayEnabled", "SetTrayToolTip", "SetMinimizeToTray", "SetCloseToTray":
			return "Tray"
		}
		return "Behavior"
	}
	propGroup := make([]string, len(uiProps))
	for i, prop := range uiProps {
		propGroup[i] = groupOf(prop.setter)
		if prop.group != "" {
			propGroup[i] = prop.group
		}
	}
	groupOpen := map[string]bool{"General": true}
	groupHeaders := make(map[string]*wui.Button)
	for _, g := range propGroups {
		g := g
		b := wui.NewButton()
		b.SetBounds(5, 0, 185, 22)
		b.SetText("+ " + g)
		b.SetVisible(false)
		b.SetOnClick(func() {
			wasOpen := groupOpen[g]
			for k := range groupOpen {
				groupOpen[k] = false
			}
			groupOpen[g] = !wasOpen
			propScroll = 0
			layoutProps()
			updateProperties()
		})
		w.Add(b)
		groupHeaders[g] = b
	}

	layoutProps = func() {
		has := make(map[string]bool)
		for i := range uiProps {
			if propShown[i] {
				has[propGroup[i]] = true
			}
		}
		has["Font"] = fontShown
		has["Events"] = eventsShown
		base := name.Y() + name.Height() + propMargin
		const headerH = 24

		// walk measures the content height and, if apply is set, places the
		// controls.
		walk := func(y int, apply bool) int {
			for _, g := range propGroups {
				if !has[g] {
					continue
				}
				if apply {
					hb := groupHeaders[g]
					hb.SetY(y)
					if groupOpen[g] {
						hb.SetText("- " + g)
					} else {
						hb.SetText("+ " + g)
					}
					hb.SetVisible(propsVisible)
				}
				y += headerH
				if !groupOpen[g] {
					continue
				}
				switch g {
				case "Font":
					if apply {
						fontProps.SetY(y)
						fontProps.SetVisible(propsVisible)
					}
					y += fontProps.Height()
				case "Events":
					if apply {
						evPanel.SetY(y + 2)
						evPanel.SetVisible(propsVisible)
					}
					y += evPanel.Height() + 4
				default:
					for i, prop := range uiProps {
						if propShown[i] && propGroup[i] == g {
							if apply {
								prop.panel.SetY(y)
								prop.panel.SetVisible(propsVisible)
							}
							y += prop.panel.Height()
						}
					}
				}
				y += 4
			}
			return y
		}

		total := walk(base, false)
		_, innerH := w.InnerSize()
		propScroll = max(0, min(propScroll, total+10-innerH))

		// Hide everything first, then show what belongs to the open groups.
		for _, prop := range uiProps {
			prop.panel.SetVisible(false)
		}
		fontProps.SetVisible(false)
		evPanel.SetVisible(false)
		for _, hb := range groupHeaders {
			hb.SetVisible(false)
		}
		nameText.SetVisible(propsVisible)
		name.SetVisible(propsVisible)
		walk(base-propScroll, true)
	}

	activate := func(newActive node) {
		active = newActive
		name.SetText(names[active])

		for i, prop := range uiProps {
			show := false
			if prop.visibleFor != nil {
				show = prop.visibleFor(active)
			} else {
				m, hasProp := reflect.TypeOf(active).MethodByName(prop.setter)
				show = hasProp && prop.rightType(m.Type.In(1))
			}
			// A property that is not in the property list of the control is
			// neither saved nor generated, so it is not offered.
			if show && !prop.isExtra {
				if !isListedProperty(active, strings.TrimPrefix(prop.setter, "Set")) {
					show = false
				}
			}
			propShown[i] = show
		}
		f, hasFont := active.(fonter)
		fontShown = hasFont
		eventsShown = len(eventsOf(active)) > 0
		refreshEvents()
		layoutProps()
		updateProperties()

		if hasFont {
			font := f.Font()
			if _, isWindow := active.(*wui.Window); isWindow {
				useParentFont.SetEnabled(false)
				useParentFont.SetChecked(false)
			} else {
				useParentFont.SetEnabled(true)
				useParentFont.SetChecked(font == nil)
			}
			if font != nil {
				fontName.SetText(font.Desc.Name)
				fontHeight.SetValue(font.Desc.Height)
				fontBold.SetChecked(font.Desc.Bold)
				fontItalic.SetChecked(font.Desc.Italic)
				fontUnderlined.SetChecked(font.Desc.Underlined)
				fontStrikedOut.SetChecked(font.Desc.StrikedOut)
			}
		}
	}

	// layoutMain places the property panel, the preview and the toolbox for the
	// current window size.
	layoutMain = func() {
		iw, ih := w.InnerSize()
		paletteVisible = paletteMode == modeShown || (paletteMode == modeAuto && iw >= 560)
		paletteW := 0
		if paletteVisible {
			paletteW = sideWidth
		}
		propsVisible = propsMode == modeShown || (propsMode == modeAuto && iw-paletteW >= 360)
		leftW := 0
		if propsVisible {
			leftW = sideWidth
		}

		leftSlider.SetVisible(propsVisible)
		leftSlider.SetBounds(leftW, -1, 5, ih+2)
		rightSlider.SetVisible(paletteVisible)
		rightSlider.SetBounds(iw-paletteW-5, -1, 5, ih+2)
		palette.SetVisible(paletteVisible)
		palette.SetBounds(iw-paletteW, 0, paletteW, ih)

		px := 0
		if propsVisible {
			px = leftW + 5
		}
		pw := iw - px - paletteW
		if paletteVisible {
			pw -= 5
		}
		preview.SetBounds(px, 0, max(pw, 40), ih)

		layoutProps()
		layoutPalette()
		if active != nil {
			updateProperties()
		}
		preview.Paint()
	}

	activate(theWindow)

	// Mouse modes
	const (
		idleMouse = iota
		addControl
		dragTopLeft
		dragTop
		dragTopRight
		dragRight
		dragBottomRight
		dragBottom
		dragBottomLeft
		dragLeft
		dragAll
	)
	var (
		mouseMode         = idleMouse
		nextDragMouseMode int
		nextToDrag        node
	)
	dragging := func() bool {
		return dragTopLeft <= mouseMode && mouseMode <= dragAll
	}

	var xOffset, yOffset int
	preview.SetOnPaint(func(c *wui.Canvas) {
		xOffset = 20 - panX - (theWindow.InnerX() - theWindow.X())
		yOffset = 40 - panY - (theWindow.InnerY() - theWindow.Y())
		width, height := theWindow.Size()
		innerWidth, innerHeight := theWindow.InnerSize()
		borderSize := (width - innerWidth) / 2
		topBorderSize := height - borderSize - innerHeight
		innerX = xOffset + borderSize
		innerY = yOffset + topBorderSize
		inner := makeOffsetDrawer(c, innerX, innerY)

		c.FillRect(0, 0, preview.Width(), preview.Height(), white)
		c.FillRect(innerX, innerY, innerWidth, innerHeight, wui.RGB(240, 240, 240))
		drawContainer(theWindow, inner)

		borderColor := wui.RGB(100, 200, 255)
		c.FillRect(xOffset, yOffset, width, topBorderSize, borderColor)
		c.FillRect(xOffset, yOffset, borderSize, height, borderColor)
		c.FillRect(xOffset, yOffset+height-borderSize, width, borderSize, borderColor)
		c.FillRect(xOffset+width-borderSize, yOffset, borderSize, height, borderColor)

		if theWindow.HasBorder() {
			_, textH := c.TextExtent(theWindow.Title())
			c.TextOut(
				xOffset+borderSize+appIconWidth+5,
				yOffset+(topBorderSize-textH)/2,
				theWindow.Title(),
				black,
			)

			w := topBorderSize
			h := w - 8
			y := yOffset + 4
			right := xOffset + width - borderSize
			x0 := right - 3*w - 2
			x1 := right - 2*w - 1
			x2 := right - 1*w - 0
			iconSize := h / 2
			if theWindow.HasMinButton() || theWindow.HasMaxButton() {
				{
					c.FillRect(x0, y, w, h, wui.RGB(240, 240, 240))
					cx := x0 + (w-iconSize)/2
					cy := y + h - 1 - (iconSize+1)/2
					color := black
					if !theWindow.HasMinButton() {
						color = wui.RGB(204, 204, 204)
					}
					c.Line(cx, cy, cx+iconSize, cy, color)
				}
				{
					c.FillRect(x1, y, w, h, wui.RGB(240, 240, 240))
					cx := x1 + (w-iconSize)/2
					cy := y + (h-iconSize)/2
					color := black
					if !theWindow.HasMaxButton() {
						color = wui.RGB(204, 204, 204)
					}
					c.DrawRect(cx, cy, iconSize, iconSize, color)
				}
			}
			color := black
			backColor := wui.RGB(255, 128, 128)
			if !theWindow.HasCloseButton() {
				color = wui.RGB(204, 204, 204)
				backColor = wui.RGB(240, 240, 240)
			}
			c.FillRect(x2, y, w, h, backColor)
			cx := x2 + (w-iconSize)/2
			cy := y + (h-iconSize)/2
			c.Line(cx, cy, cx+iconSize, cy+iconSize, color)
			c.Line(cx, cy+iconSize-1, cx+iconSize, cy-1, color)

			w32.DrawIconEx(
				w32.HDC(c.Handle()),
				xOffset+borderSize,
				yOffset+(topBorderSize-appIconHeight)/2,
				appIcon,
				appIconWidth, appIconHeight,
				0, 0, w32.DI_NORMAL,
			)
		}

		if !dragging() && active != nil && active != theWindow {
			x, y, w, h := active.Bounds()
			parent := active.Parent()
			for parent != theWindow {
				dx, dy, _, _ := parent.InnerBounds()
				x += dx
				y += dy
				parent = parent.Parent()
			}
			w = max(w, 0)
			h = max(h, 0)
			inner.DrawRect(x, y, w, h, wui.RGB(255, 0, 255))
			inner.DrawRect(x+1, y+1, w-2, h-2, wui.RGB(255, 0, 255))
		}

		if controlToAdd != nil {
			drawControl(controlToAdd, c)
		}
	})
	w.Add(preview)

	var (
		dragStartX, dragStartY                                  int
		preResizeX, preResizeY, preResizeWidth, preResizeHeight int
	)

	lastX, lastY := -999999, -999999
	w.SetOnMouseMove(func(x, y int) {
		if x == lastX && y == lastY {
			return
		}
		lastX, lastY = x, y

		// Un-highlight the template when the mouse has left the toolbox.
		if highlightedTemplate != nil && !contains(palette, x, y) {
			highlightedTemplate = nil
			palette.Paint()
		}

		if mouseMode == addControl {
			if contains(preview, x, y) {
				_, _, w, h := controlToAdd.Bounds()
				relX := x - preview.X()
				relY := y - preview.Y()
				relX += templateDx
				relY += templateDy
				controlToAdd.SetBounds(relX, relY, w, h)
			}
			preview.Paint()
		} else if mouseMode == idleMouse {
			x -= preview.X()
			y -= preview.Y()
			x -= xOffset
			y -= yOffset
			nextToDrag = active
			ax, ay, aw, ah := relativeBounds(active, theWindow)
			const margin = 6
			corner := func(x, y int) rectangle {
				return rect(x-margin, y-margin, 2*margin, 2*margin)
			}
			var (
				topLeft     = corner(ax, ay)
				top         = rect(ax, ay-margin, aw, 2*margin)
				topRight    = corner(ax+aw, ay)
				right       = rect(ax+aw-margin, ay, 2*margin, ah)
				bottomRight = corner(ax+aw, ay+ah)
				bottom      = rect(ax, ay+ah-margin, aw, 2*margin)
				bottomLeft  = corner(ax, ay+ah)
				left        = rect(ax-margin, ay, 2*margin, ah)
			)
			if active == theWindow {
				topLeft = rectangle{}
				top = rectangle{}
				topRight = rectangle{}
				bottomLeft = rectangle{}
				left = rectangle{}
			}
			var (
				winX, winY, winW, winH = relativeBounds(theWindow, theWindow)
				winRight               = rect(winX+winW-margin, winY, 2*margin, winH)
				winBottomRight         = corner(winX+winW, winY+winH)
				winBottom              = rect(winX, winY+winH-margin, winW, 2*margin)
			)
			if winBottomRight.contains(x, y) {
				nextDragMouseMode = dragBottomRight
				w.SetCursor(wui.CursorSizeNWSE)
				nextToDrag = theWindow
			} else if winRight.contains(x, y) {
				nextDragMouseMode = dragRight
				w.SetCursor(wui.CursorSizeWE)
				nextToDrag = theWindow
			} else if winBottom.contains(x, y) {
				nextDragMouseMode = dragBottom
				w.SetCursor(wui.CursorSizeNS)
				nextToDrag = theWindow
			} else if topLeft.contains(x, y) {
				nextDragMouseMode = dragTopLeft
				w.SetCursor(wui.CursorSizeNWSE)
			} else if topRight.contains(x, y) {
				nextDragMouseMode = dragTopRight
				w.SetCursor(wui.CursorSizeNESW)
			} else if bottomRight.contains(x, y) {
				nextDragMouseMode = dragBottomRight
				w.SetCursor(wui.CursorSizeNWSE)
			} else if bottomLeft.contains(x, y) {
				nextDragMouseMode = dragBottomLeft
				w.SetCursor(wui.CursorSizeNESW)
			} else if top.contains(x, y) {
				nextDragMouseMode = dragTop
				w.SetCursor(wui.CursorSizeNS)
			} else if right.contains(x, y) {
				nextDragMouseMode = dragRight
				w.SetCursor(wui.CursorSizeWE)
			} else if bottom.contains(x, y) {
				nextDragMouseMode = dragBottom
				w.SetCursor(wui.CursorSizeNS)
			} else if left.contains(x, y) {
				nextDragMouseMode = dragLeft
				w.SetCursor(wui.CursorSizeWE)
			} else {
				innerX, innerY, _, _ := theWindow.InnerBounds()
				outerX, outerY, _, _ := theWindow.Bounds()
				relX := x - (innerX - outerX)
				relY := y - (innerY - outerY)
				if theWindow != active && active == findControlAt(theWindow, relX, relY) {
					nextDragMouseMode = dragAll
					w.SetCursor(wui.CursorSizeAll)
				} else {
					nextDragMouseMode = idleMouse
					w.SetCursor(defaultCursor)
				}
			}
		} else {
			dx := x - dragStartX
			dy := y - dragStartY
			x, y, w, h := preResizeX, preResizeY, preResizeWidth, preResizeHeight
			switch mouseMode {
			case dragTopLeft:
				dx = min(dx, w)
				dy = min(dy, h)
				nextToDrag.SetBounds(x+dx, y+dy, w-dx, h-dy)
			case dragTop:
				dy = min(dy, h)
				nextToDrag.SetBounds(x, y+dy, w, h-dy)
			case dragTopRight:
				dx = max(dx, -w)
				dy = min(dy, h)
				nextToDrag.SetBounds(x, y+dy, w+dx, h-dy)
			case dragRight:
				dx = max(dx, -w)
				nextToDrag.SetBounds(x, y, w+dx, h)
			case dragBottomRight:
				dx = max(dx, -w)
				dy = max(dy, -h)
				nextToDrag.SetBounds(x, y, w+dx, h+dy)
			case dragBottom:
				dy = max(dy, -h)
				nextToDrag.SetBounds(x, y, w, h+dy)
			case dragBottomLeft:
				dx = min(dx, w)
				dy = max(dy, -h)
				nextToDrag.SetBounds(x+dx, y, w-dx, h+dy)
			case dragLeft:
				dx = min(dx, w)
				nextToDrag.SetBounds(x+dx, y, w-dx, h)
			case dragAll:
				nextToDrag.SetBounds(x+dx, y+dy, w, h)
			}
			updateProperties()
			preview.Paint()
		}
	})

	w.SetOnMouseDown(func(button wui.MouseButton, x, y int) {
		if button == wui.MouseButtonLeft {
			if paletteVisible && mouseMode == idleMouse && contains(palette, x, y) {
				lx, ly := x-palette.X(), y-palette.Y()
				for i, r := range paletteHeaders {
					if r.contains(lx, ly) {
						wasOpen := paletteGroups[i].open
						for _, g := range paletteGroups {
							g.open = false
						}
						paletteGroups[i].open = !wasOpen
						layoutPalette()
						return
					}
				}
			}
			if contains(palette, x, y) && highlightedTemplate != nil {
				controlToAdd = cloneControl(highlightedTemplate)
				hx, hy, _, _ := highlightedTemplate.Bounds()
				templateDx = hx - (x - palette.X())
				templateDy = hy - (y - palette.Y())
				mouseMode = addControl
				activate(theWindow)
				preview.Paint()
			} else if mouseMode == addControl {
				innerX, innerY, _, _ := theWindow.InnerBounds()
				outerX, outerY, _, _ := theWindow.Bounds()
				x, y, w, h := controlToAdd.Bounds()
				relX := x - (xOffset + innerX - outerX)
				relY := y - (yOffset + innerY - outerY)
				addToThis, x, y := findContainerAt(theWindow, relX+w/2, relY+h/2)
				controlToAdd.SetBounds(x-w/2, y-h/2, w, h)
				names[controlToAdd] = defaultName(controlToAdd)
				addToThis.Add(controlToAdd)
				activate(controlToAdd)
				controlToAdd = nil
				mouseMode = idleMouse
				name.Focus()
				name.SelectAll()
				preview.Paint()
				markModified()
			} else {
				dragStartX = x
				dragStartY = y
				preResizeX, preResizeY, preResizeWidth, preResizeHeight = nextToDrag.Bounds()
				mouseMode = nextDragMouseMode
				if mouseMode == idleMouse && contains(preview, x, y) {
					newActive := findControlAt(
						theWindow,
						x-preview.X()-innerX,
						y-preview.Y()-innerY,
					)
					if newActive != active {
						activate(newActive)
					}
				}
				preview.Paint()
			}
		}
	})

	w.SetOnMouseUp(func(button wui.MouseButton, x, y int) {
		if button == wui.MouseButtonLeft {
			if mouseMode != addControl {
				mouseMode = idleMouse
				markModified()
			}
		}
		preview.Paint()
	})


	// openPath loads the project file and shows it.
	openPath := func(path string) {
		newWindow, notice, err := loadProject(path)
		if err != nil {
			wui.MessageBoxError("Error", "Failed to load project:\n"+err.Error())
			return
		}
		theWindow = newWindow
		panX, panY = 0, 0
		activate(theWindow)
		layoutMain()
		setWorkingPath(path)
		markSaved()
		if notice != "" {
			wui.MessageBoxInfo("Project", notice)
		}
	}

	// Dropping a project file onto the designer opens it.
	w.SetOnDropFiles(func(files []string, x, y int) {
		for _, f := range files {
			if strings.HasSuffix(strings.ToLower(f), projectExt) {
				if confirmDiscard("Do you want to save changes to the current project?") {
					openPath(f)
				}
				return
			}
		}
	})

	// New project
	fileNewMenu.SetOnClick(func() {
		if !confirmDiscard("Do you want to save changes to the current project?") {
			return
		}
		theWindow = defaultWindow()
		names = make(map[interface{}]string)
		names[theWindow] = "window"
		events = make(map[event]string)
		extras = make(map[interface{}]map[string]string)
		panX, panY = 0, 0
		activate(theWindow)
		layoutMain()
		setWorkingPath("")
		markSaved()
	})

	// Open project
	fileOpenMenu.SetOnClick(func() {
		if !confirmDiscard("Do you want to save changes to the current project?") {
			return
		}
		open := wui.NewFileOpenDialog()
		open.SetTitle("Open wui Designer Project")
		open.AddFilter("WML project file", projectExt)
		open.AddFilter("All files", "*")
		if accept, path := open.ExecuteSingleSelection(w); accept {
			openPath(path)
		}
	})

	// Save project
	fileSaveMenu.SetOnClick(func() {
		saveCurrent()
	})

	// Save project as
	fileSaveAsMenu.SetOnClick(func() {
		saveAs()
	})

	// Export as Go
	fileExportGoMenu.SetOnClick(func() {
		save := wui.NewFileSaveDialog()
		save.SetAppendExt(true)
		save.AddFilter("Go file", ".go")
		if accept, path := save.Execute(w); accept {
			code, err := generateCode(theWindow, false)
			if err == nil {
				err = ioutil.WriteFile(path, code, 0666)
			}
			if err != nil {
				wui.MessageBoxError("Error", err.Error())
			} else {
				wui.MessageBoxInfo("Success", "Go code exported to "+path)
			}
		}
	})

	// Preview
	previewMenu.SetOnClick(func() {
		x, y := w32.ClientToScreen(w32.HWND(w.Handle()), preview.X(), preview.Y())
		showPreview(w, theWindow, x+xOffset, y+yOffset)
	})

	// Exit
	exitMenu.SetOnClick(func() {
		w.Close()
	})

	// Window close handler
	w.SetOnCanClose(func() bool {
		return confirmDiscard("Do you want to save changes before exiting?")
	})

	// Edit operations
	// A copied control is kept as a clone together with its position.
	var clipCtl wui.Control
	var clipX, clipY int

	copyActive := func() bool {
		c, ok := active.(wui.Control)
		if !ok || active == theWindow {
			return false
		}
		clipCtl = cloneTree(c)
		clipX, clipY, _, _ = c.Bounds()
		return true
	}

	pasteClip := func() {
		if clipCtl == nil {
			return
		}
		var target wui.Container
		if cont, ok := active.(wui.Container); ok {
			target = cont
		} else {
			target = realParent(theWindow, active)
		}
		if target == nil {
			target = theWindow
		}
		nc := cloneTree(clipCtl)
		_, _, cw, ch := clipCtl.Bounds()
		clipX += 10
		clipY += 10
		nc.SetBounds(clipX, clipY, cw, ch)
		names[nc] = defaultName(nc)
		if con, ok := nc.(wui.Container); ok {
			nameAll(con)
		}
		target.Add(nc)
		activate(nc)
		preview.Paint()
		markModified()
	}

	cutMenu.SetOnClick(func() {
		if copyActive() {
			c := active.(wui.Control)
			p := realParent(theWindow, active)
			activate(p)
			p.Remove(c)
			preview.Paint()
			markModified()
		}
	})

	copyMenu.SetOnClick(func() {
		copyActive()
	})

	pasteMenu.SetOnClick(pasteClip)

	duplicateMenu.SetOnClick(func() {
		if copyActive() {
			// Paste next to the original, not into the control itself.
			if p := realParent(theWindow, active); p != nil {
				activate(p)
			}
			pasteClip()
		}
	})

	deleteMenu.SetOnClick(func() {
		if active != nil && active != theWindow {
			result := wui.MessageBoxYesNo("Confirm Delete", 
				fmt.Sprintf("Delete '%s'?", names[active]))
			if result {
				c := active.(wui.Control)
				p := realParent(theWindow, active)
				activate(p)
				p.Remove(c)
				preview.Paint()
				markModified()
			}
		}
	})

	// View operations
	viewToolboxMenu.SetOnClick(func() {
		if paletteVisible {
			paletteMode = modeHidden
		} else {
			paletteMode = modeShown
		}
		layoutMain()
	})

	viewPropsMenu.SetOnClick(func() {
		if propsVisible {
			propsMode = modeHidden
		} else {
			propsMode = modeShown
		}
		layoutMain()
	})

	viewAutoLayoutMenu.SetOnClick(func() {
		paletteMode, propsMode = modeAuto, modeAuto
		layoutMain()
	})

	viewFullScreenMenu.SetOnClick(func() {
		if w.State() == wui.WindowMaximized {
			w.SetState(wui.WindowNormal)
		} else {
			w.SetState(wui.WindowMaximized)
		}
	})

	// Help operations
	helpAboutMenu.SetOnClick(func() {
		wui.MessageBoxInfo("About wui Designer", 
			"wui Designer\n\n"+
			"A visual designer for wui GUI applications.\n"+
			"Projects are saved as WML (.wml) files.\n\n"+
			"Created with wui framework.\n"+
			"https://github.com/2dprototype/wui")
	})

	helpShortcutsMenu.SetOnClick(func() {
		wui.MessageBoxInfo("Keyboard Shortcuts",
			"File Operations:\n"+
			"  Ctrl+N - New Project\n"+
			"  Ctrl+O - Open Project\n"+
			"  Ctrl+S - Save Project\n"+
			"  Ctrl+Shift+S - Save Project As\n"+
			"  Ctrl+E - Export as Go\n"+
			"  F5 - Run Preview\n"+
			"  Alt+F4 - Exit\n\n"+
			"Edit Operations:\n"+
			"  Ctrl+X - Cut\n"+
			"  Ctrl+C - Copy\n"+
			"  Ctrl+V - Paste\n"+
			"  Del - Delete\n"+
			"  Ctrl+D - Duplicate\n\n"+
			"View Operations:\n"+
			"  F11 - Maximize / restore window\n"+
			"  F2 - Toggle Toolbox\n"+
			"  F3 - Toggle Properties\n"+
			"  F4 - Auto Layout\n"+
			"  Mouse wheel - scroll panels / pan preview (Shift: sideways)\n\n"+
			"Help:\n"+
			"  F1 - About\n"+
			"  Ctrl+F1 - Keyboard Shortcuts")
	})

	// Set all keyboard shortcuts
	w.SetShortcut(fileNewMenu.OnClick(), wui.KeyControl, wui.KeyN)
	w.SetShortcut(fileOpenMenu.OnClick(), wui.KeyControl, wui.KeyO)
	w.SetShortcut(fileSaveMenu.OnClick(), wui.KeyControl, wui.KeyS)
	w.SetShortcut(fileSaveAsMenu.OnClick(), wui.KeyControl, wui.KeyShift, wui.KeyS)
	w.SetShortcut(fileExportGoMenu.OnClick(), wui.KeyControl, wui.KeyE)
	w.SetShortcut(previewMenu.OnClick(), wui.KeyF5)
	w.SetShortcut(exitMenu.OnClick(), wui.KeyAlt, wui.KeyF4)

	w.SetShortcut(cutMenu.OnClick(), wui.KeyControl, wui.KeyX)
	w.SetShortcut(copyMenu.OnClick(), wui.KeyControl, wui.KeyC)
	w.SetShortcut(pasteMenu.OnClick(), wui.KeyControl, wui.KeyV)
	w.SetShortcut(deleteMenu.OnClick(), wui.KeyDelete)
	w.SetShortcut(duplicateMenu.OnClick(), wui.KeyControl, wui.KeyD)
	w.SetShortcut(viewToolboxMenu.OnClick(), wui.KeyF2)
	w.SetShortcut(viewPropsMenu.OnClick(), wui.KeyF3)
	w.SetShortcut(viewAutoLayoutMenu.OnClick(), wui.KeyF4)

	w.SetShortcut(viewFullScreenMenu.OnClick(), wui.KeyF11)

	w.SetShortcut(helpAboutMenu.OnClick(), wui.KeyF1)
	w.SetShortcut(helpShortcutsMenu.OnClick(), wui.KeyControl, wui.KeyF1)

	w.SetMinSize(360, 300)
	w.SetOnResize(layoutMain)
	layoutMain()

	w.SetOnMouseWheel(func(sx, sy int, delta float64) {
		x, y, ok := w32.ScreenToClient(w32.HWND(w.Handle()), sx, sy)
		if !ok {
			return
		}
		step := int(-delta * 40)
		switch {
		case propsVisible && x < sideWidth:
			propScroll += step
			layoutProps()
			updateProperties()
		case paletteVisible && contains(palette, x, y):
			paletteScroll += step
			layoutPalette()
		case contains(preview, x, y):
			winW, winH := theWindow.Size()
			if w32.GetKeyState(w32.VK_SHIFT)&0x8000 != 0 {
				panX = max(0, min(panX+step, winW+40-preview.Width()))
			} else {
				panY = max(0, min(panY+step, winH+60-preview.Height()))
			}
			preview.Paint()
		}
	})
	
	w.SetMinSize(360, 300)
	w.SetOnResize(layoutMain)
	layoutMain()

	w.SetOnMouseWheel(func(sx, sy int, delta float64) {
		x, y, ok := w32.ScreenToClient(w32.HWND(w.Handle()), sx, sy)
		if !ok {
			return
		}
		step := int(-delta * 40)
		switch {
		case propsVisible && x < sideWidth:
			propScroll += step
			layoutProps()
			updateProperties()
		case paletteVisible && contains(palette, x, y):
			paletteScroll += step
			layoutPalette()
		case contains(preview, x, y):
			winW, winH := theWindow.Size()
			if w32.GetKeyState(w32.VK_SHIFT)&0x8000 != 0 {
				panX = max(0, min(panX+step, winW+40-preview.Width()))
			} else {
				panY = max(0, min(panY+step, winH+60-preview.Height()))
			}
			preview.Paint()
		}
	})

	// ------------------------------------------------------------------
	// FIX: the design window created at the top of main() ends up with
	// the wrong inner size because SetInnerSize relies on frame metrics
	// (border width, caption height) that the framework only learns once
	// a real window has been shown. The metrics are correct by the time
	// the designer window becomes visible, so we recreate the design
	// window in the first OnShow callback.
	//
	// This is also where a project file given on the command line is
	// loaded, so that it can replace the freshly created empty window.
	//
	// Usage: designer.exe <file.wml>
	// ------------------------------------------------------------------
	firstShow := true
	w.SetOnShow(func() {
		if !firstShow {
			return
		}
		firstShow = false

		// Recreate the empty design window with correct frame metrics.
		theWindow = defaultWindow()
		names = make(map[interface{}]string)
		names[theWindow] = "window"
		events = make(map[event]string)
		extras = make(map[interface{}]map[string]string)
		panX, panY = 0, 0
		activate(theWindow)
		layoutMain()
		setWorkingPath("")
		markSaved()

		// If a project file was passed on the command line, load it now.
		// This also handles .wml files dropped onto the executable in
		// Windows Explorer, since Explorer passes them as arguments.
		// "-flag" style arguments are skipped so future options do not get
		// confused with a project path.
		for _, arg := range os.Args[1:] {
			if strings.HasPrefix(arg, "-") {
				continue
			}
			openPath(arg)
			break
		}
	})

	w.SetState(wui.WindowMaximized)
	w.Show()
}


func rect(x, y, width, height int) rectangle {
	return rectangle{x: x, y: y, w: width, h: height}
}

type rectangle struct {
	x, y, w, h int
}

func (r rectangle) contains(x, y int) bool {
	return x >= r.x && y >= r.y && x < r.x+r.w && y < r.y+r.h
}

func defaultFont() *wui.Font {
	font, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -11})
	return font
}

// defaultWindow is the empty window of a new project. Its size is given as
// inner size: the area for controls is then always 600x400, whatever the
// borders and the title bar of the current Windows version and theme are.
func defaultWindow() *wui.Window {
	w := wui.NewWindow()
	w.SetFont(defaultFont())
	w.SetTitle("Window")
	w.SetInnerSize(600, 400)
	return w
}

func findControlAt(parent wui.Container, x, y int) node {
	for _, child := range parent.Children() {
		if contains(child, x, y) {
			if container, ok := child.(wui.Container); ok {
				dx, dy, _, _ := container.Bounds()
				return findControlAt(container, x-dx, y-dy)
			}
			return child
		}
	}
	return parent
}

func contains(b bounder, atX, atY int) bool {
	x, y, w, h := b.Bounds()
	return atX >= x && atY >= y && atX < x+w && atY < y+h
}

type bounder interface {
	Bounds() (x, y, width, height int)
}

func innerContains(b innerBounder, atX, atY int) bool {
	x, y, w, h := b.InnerBounds()
	return atX >= x && atY >= y && atX < x+w && atY < y+h
}

type innerBounder interface {
	InnerBounds() (x, y, width, height int)
}

type drawer interface {
	PushDrawRegion(x, y, width, height int)
	PopDrawRegion()
	Line(x1, y1, x2, y2 int, color wui.Color)
	DrawRect(x, y, w, h int, color wui.Color)
	FillRect(x, y, w, h int, color wui.Color)
	DrawEllipse(x, y, w, h int, color wui.Color)
	FillEllipse(x, y, w, h int, color wui.Color)
	TextRectFormat(x, y, w, h int, s string, format wui.Format, color wui.Color)
	TextExtent(s string) (width, height int)
	TextOut(x, y int, s string, color wui.Color)
	DrawImage(img *wui.Image, src wui.Rectangle, destX, destY int)
	DrawImageScaled(img *wui.Image, src, dest wui.Rectangle)
	Polygon(p []wui.Point, color wui.Color)
	SetFont(*wui.Font)
}

func makeOffsetDrawer(base drawer, dx, dy int) drawer {
	return &offsetDrawer{base: base, dx: dx, dy: dy}
}

type offsetDrawer struct {
	base   drawer
	dx, dy int
}

func (d *offsetDrawer) PushDrawRegion(x, y, width, height int) {
	d.base.PushDrawRegion(x+d.dx, y+d.dy, width, height)
}

func (d *offsetDrawer) PopDrawRegion() {
	d.base.PopDrawRegion()
}

func (d *offsetDrawer) DrawRect(x, y, w, h int, color wui.Color) {
	d.base.DrawRect(x+d.dx, y+d.dy, w, h, color)
}

func (d *offsetDrawer) FillRect(x, y, w, h int, color wui.Color) {
	d.base.FillRect(x+d.dx, y+d.dy, w, h, color)
}

func (d *offsetDrawer) DrawEllipse(x, y, w, h int, color wui.Color) {
	d.base.DrawEllipse(x+d.dx, y+d.dy, w, h, color)
}

func (d *offsetDrawer) FillEllipse(x, y, w, h int, color wui.Color) {
	d.base.FillEllipse(x+d.dx, y+d.dy, w, h, color)
}

func (d *offsetDrawer) TextRectFormat(
	x, y, w, h int, s string, format wui.Format, color wui.Color,
) {
	d.base.TextRectFormat(x+d.dx, y+d.dy, w, h, s, format, color)
}

func (d *offsetDrawer) TextExtent(s string) (width, height int) {
	return d.base.TextExtent(s)
}

func (d *offsetDrawer) TextOut(x, y int, s string, color wui.Color) {
	d.base.TextOut(x+d.dx, y+d.dy, s, color)
}

func (d *offsetDrawer) DrawImage(img *wui.Image, src wui.Rectangle, destX, destY int) {
	d.base.DrawImage(img, src, destX+d.dx, destY+d.dy)
}

func (d *offsetDrawer) DrawImageScaled(img *wui.Image, src, dest wui.Rectangle) {
	dest.X += d.dx
	dest.Y += d.dy
	d.base.DrawImageScaled(img, src, dest)
}

func (d *offsetDrawer) Polygon(p []wui.Point, color wui.Color) {
	for i := range p {
		p[i].X += int32(d.dx)
		p[i].Y += int32(d.dy)
	}
	d.base.Polygon(p, color)
}

func (d *offsetDrawer) Line(x1, y1, x2, y2 int, color wui.Color) {
	d.base.Line(x1+d.dx, y1+d.dy, x2+d.dx, y2+d.dy, color)
}

func (d *offsetDrawer) SetFont(f *wui.Font) {
	d.base.SetFont(f)
}

func drawContainer(container wui.Container, d drawer) {
	_, _, w, h := container.InnerBounds()
	d.PushDrawRegion(0, 0, w, h)
	for _, child := range container.Children() {
		if f, ok := child.(fontControl); ok {
			d.SetFont(getFont(f))
		}
		drawControl(child, d)
	}
	d.PopDrawRegion()
}

func drawControl(c wui.Control, d drawer) {
	switch x := c.(type) {
	case *wui.Button:
		drawButton(x, d)
	case *wui.RadioButton:
		drawRadioButton(x, d)
	case *wui.CheckBox:
		drawCheckBox(x, d)
	case *wui.Panel:
		drawPanel(x, d)
	case *wui.Slider:
		drawSlider(x, d)
	case *wui.Label:
		drawLabel(x, d)
	case *wui.PaintBox:
		drawPaintBox(x, d)
	case *wui.EditLine:
		drawEditLine(x, d)
	case *wui.IntUpDown:
		drawIntUpDown(x, d)
	case *wui.ComboBox:
		drawComboBox(x, d)
	case *wui.ProgressBar:
		drawProgressBar(x, d)
	case *wui.FloatUpDown:
		drawFloatUpDown(x, d)
	case *wui.TextEdit:
		drawTextEdit(x, d)
	case *wui.GroupBox:
		drawGroupBox(x, d)
	case *wui.StatusBar:
		drawStatusBar(x, d)
	case *wui.TabControl:
		drawTabControl(x, d)
	case *wui.DatePicker:
		drawDatePicker(x, d)
	case *wui.ListView:
		drawListView(x, d)
	case *wui.RichEdit:
		drawRichEdit(x, d)
	case *wui.LinkLabel:
		drawLinkLabel(x, d)
	case *wui.MonthCalendar:
		drawMonthCalendar(x, d)
	case *wui.HotKeyEdit:
		drawHotKeyEdit(x, d)
	case *wui.IPAddressEdit:
		drawIPAddressEdit(x, d)
	case *wui.ImageView:
		drawImageView(x, d)
	case *wui.ScrollPanel:
		drawScrollPanel(x, d)
	case *wui.ScrollBar:
		drawScrollBar(x, d)
	case *wui.TreeView:
		drawTreeView(x, d)
	default:
		drawUnknown(c, d)
	}
}

func drawButton(b *wui.Button, d drawer) {
	x, y, w, h := b.Bounds()
	if w > 0 && h > 0 {
		d.DrawRect(x, y, w, h, wui.RGB(240, 240, 240))
	}
	if w > 2 && h > 2 {
		d.FillRect(x+1, y+1, w-2, h-2, wui.RGB(173, 173, 173))
	}
	if w > 4 && h > 4 {
		d.FillRect(x+2, y+2, w-4, h-4, wui.RGB(225, 225, 225))
	}
	if w > 6 && h > 6 {
		d.SetFont(getFont(b))
		textW, textH := d.TextExtent(b.Text())
		d.PushDrawRegion(x+3, y+3, w-6, h-6)
		d.TextOut(x+(w-textW)/2, y+(h-textH)/2, b.Text(), wui.RGB(0, 0, 0))
		d.PopDrawRegion()
	}
}

func drawRadioButton(r *wui.RadioButton, d drawer) {
	x, y, w, h := r.Bounds()
	d.PushDrawRegion(x, y, w, h)
	d.FillRect(x, y, w, h, bgOf(r, wui.RGB(240, 240, 240)))
	d.FillEllipse(x, y+(h-13)/2, 13, 13, wui.RGB(255, 255, 255))
	d.DrawEllipse(x, y+(h-13)/2, 13, 13, wui.RGB(0, 0, 0))
	if r.Checked() {
		d.FillEllipse(x+3, y+(h-13)/2+3, 7, 7, wui.RGB(0, 0, 0))
	}
	_, textH := d.TextExtent(r.Text())
	d.TextOut(x+16, y+(h-textH)/2, r.Text(), fgOf(r, wui.RGB(0, 0, 0)))
	d.PopDrawRegion()
}

func drawCheckBox(c *wui.CheckBox, d drawer) {
	x, y, w, h := c.Bounds()
	d.PushDrawRegion(x, y, w, h)
	d.FillRect(x, y, w, h, bgOf(c, wui.RGB(240, 240, 240)))
	d.FillRect(x, y+(h-13)/2, 13, 13, wui.RGB(255, 255, 255))
	d.DrawRect(x, y+(h-13)/2, 13, 13, wui.RGB(0, 0, 0))
	if c.Checked() {
		// Draw two lines for the check mark. ✓
		startX := x + 2
		startY := y + (h-13)/2 + 6
		d.Line(startX, startY, startX+3, startY+3, wui.RGB(0, 0, 0))
		d.Line(startX+3, startY+2, startX+9, startY-4, wui.RGB(0, 0, 0))
	}
	_, textH := d.TextExtent(c.Text())
	d.TextOut(x+16, y+(h-textH)/2, c.Text(), fgOf(c, wui.RGB(0, 0, 0)))
	d.PopDrawRegion()
}

func drawPanel(p *wui.Panel, d drawer) {
	x, y, w, h := p.Bounds()
	if w <= 0 || h <= 0 {
		return
	}
	if p.HasBackgroundColor() {
		d.FillRect(x, y, w, h, p.BackgroundColor())
	}
	switch p.BorderStyle() {
	case wui.PanelBorderNone:
		d.DrawRect(x, y, w, h, wui.RGB(230, 230, 230))
	case wui.PanelBorderSingleLine:
		d.DrawRect(x, y, w, h, wui.RGB(100, 100, 100))
	case wui.PanelBorderRaised:
		d.Line(x, y, x+w, y, wui.RGB(227, 227, 227))
		d.Line(x, y, x, y+h, wui.RGB(227, 227, 227))
		d.Line(x+w-1, y, x+w-1, y+h, wui.RGB(105, 105, 105))
		d.Line(x, y+h-1, x+w, y+h-1, wui.RGB(105, 105, 105))
		d.Line(x+1, y+1, x+w-1, y+1, wui.RGB(255, 255, 255))
		d.Line(x+1, y+1, x+1, y+h-1, wui.RGB(255, 255, 255))
		d.Line(x+w-2, y+1, x+w-2, y+h-1, wui.RGB(160, 160, 160))
		d.Line(x+1, y+h-2, x+w-1, y+h-2, wui.RGB(160, 160, 160))
	case wui.PanelBorderSunken:
		d.Line(x, y, x+w, y, wui.RGB(160, 160, 160))
		d.Line(x, y, x, y+h, wui.RGB(160, 160, 160))
		d.Line(x+w-1, y, x+w-1, y+h, wui.RGB(255, 255, 255))
		d.Line(x, y+h-1, x+w, y+h-1, wui.RGB(255, 255, 255))
	case wui.PanelBorderSunkenThick:
		d.Line(x, y, x+w, y, wui.RGB(160, 160, 160))
		d.Line(x, y, x, y+h, wui.RGB(160, 160, 160))
		d.Line(x+w-1, y, x+w-1, y+h, wui.RGB(255, 255, 255))
		d.Line(x, y+h-1, x+w, y+h-1, wui.RGB(255, 255, 255))
		d.Line(x+1, y+1, x+w-1, y+1, wui.RGB(105, 105, 105))
		d.Line(x+1, y+1, x+1, y+h-1, wui.RGB(105, 105, 105))
		d.Line(x+w-2, y+1, x+w-2, y+h-1, wui.RGB(227, 227, 227))
		d.Line(x+1, y+h-2, x+w-1, y+h-2, wui.RGB(227, 227, 227))
	}
	innerX, innerY, _, _ := p.InnerBounds()
	drawContainer(p, makeOffsetDrawer(d, innerX, innerY))
}

func drawSlider(s *wui.Slider, d drawer) {
	var (
		drawSlideBar    func(offset int)
		drawCursorBody  func(offset, size int)
		drawCursorArrow func(offset int)
		// drawEndTicks and drawMiddleTicks are only assigned if ticks are
		// visible for this slider.
		drawEndTicks    = func(offset int) {}
		drawMiddleTicks = func(offset int) {}
	)

	cursorColor := wui.RGB(0, 120, 215)
	tickColor := wui.RGB(196, 196, 196)
	slideBarBorder := wui.RGB(214, 214, 214)
	slideBarBackground := wui.RGB(231, 231, 234)

	x, y, w, h := s.Bounds()
	if w <= 0 || h <= 0 {
		return
	}
	d.PushDrawRegion(x, y, w, h)
	defer d.PopDrawRegion()
	min, max := s.MinMax()
	innerTickCount := max - min - 1
	freq := s.TickFrequency()
	relCursor := s.CursorPosition() - min

	if s.Orientation() == wui.HorizontalSlider {
		xLeft := x + 13
		xRight := x + w - 14
		scale := 1.0 / float64(innerTickCount+1) * float64(xRight-xLeft)
		if xRight < xLeft {
			xRight = xLeft
			scale = 0
		}
		xOffset := int(float64(relCursor)*scale + 0.5)
		cursorCenter := xLeft + xOffset

		drawSlideBar = func(offset int) {
			if xLeft != xRight {
				d.DrawRect(x+8, y+offset, w-16, 4, slideBarBorder)
				d.FillRect(x+9, y+offset+1, w-18, 2, slideBarBackground)
			}
		}
		drawCursorBody = func(offset, size int) {
			d.FillRect(cursorCenter-5, y+offset, 11, size, cursorColor)
		}
		drawCursorArrow = func(offset int) {
			d.Polygon([]wui.Point{
				{int32(cursorCenter - 5), int32(y + 15)},
				{int32(cursorCenter), int32(y + 15 + offset)},
				{int32(cursorCenter + 5), int32(y + 15)},
			}, cursorColor)
		}

		if s.TicksVisible() {
			drawEndTicks = func(offset int) {
				d.Line(xLeft, y+offset, xLeft, y+offset+4, tickColor)
				d.Line(xRight, y+offset, xRight, y+offset+4, tickColor)
			}
			drawMiddleTicks = func(offset int) {
				for i := freq; i <= innerTickCount; i += freq {
					x := xLeft + int(float64(i)*scale+0.5)
					d.Line(x, y+offset, x, y+offset+3, tickColor)
				}
			}
		}
	} else {
		yTop := y + 13
		yBottom := y + h - 14
		scale := 1.0 / float64(innerTickCount+1) * float64(yBottom-yTop)
		if yBottom < yTop {
			yBottom = yTop
			scale = 0
		}
		yOffset := int(float64(relCursor)*scale + 0.5)
		cursorCenter := yTop + yOffset

		drawSlideBar = func(offset int) {
			if yTop != yBottom {
				d.DrawRect(x+offset, y+8, 4, h-16, slideBarBorder)
				d.FillRect(x+offset+1, y+9, 2, h-18, slideBarBackground)
			}
		}
		drawCursorBody = func(offset, size int) {
			d.FillRect(x+offset, cursorCenter-5, size, 11, cursorColor)
		}
		drawCursorArrow = func(offset int) {
			d.Polygon([]wui.Point{
				{int32(x + 15), int32(cursorCenter - 5)},
				{int32(x + 15 + offset), int32(cursorCenter)},
				{int32(x + 15), int32(cursorCenter + 5)},
			}, cursorColor)
		}

		if s.TicksVisible() {
			drawEndTicks = func(offset int) {
				d.Line(x+offset, yTop, x+offset+4, yTop, tickColor)
				d.Line(x+offset, yBottom, x+offset+4, yBottom, tickColor)
			}
			drawMiddleTicks = func(offset int) {
				for i := freq; i <= innerTickCount; i += freq {
					y := yTop + int(float64(i)*scale+0.5)
					d.Line(x+offset, y, x+offset+3, y, tickColor)
				}
			}
		}
	}

	switch s.TickPosition() {
	case wui.TicksBottomOrRight:
		drawSlideBar(8)
		drawCursorBody(2, 14)
		drawCursorArrow(5)
		drawEndTicks(22)
		drawMiddleTicks(22)
	case wui.TicksTopOrLeft:
		drawSlideBar(18)
		drawCursorBody(15, 14)
		drawCursorArrow(-5)
		drawEndTicks(5)
		drawMiddleTicks(6)
	case wui.TicksOnBothSides:
		drawSlideBar(19)
		drawCursorBody(10, 21)
		drawEndTicks(5)
		drawEndTicks(33)
		drawMiddleTicks(6)
		drawMiddleTicks(33)
	default:
		panic("unhandled tick position")
	}
}

func drawLabel(l *wui.Label, d drawer) {
	x, y, w, h := l.Bounds()
	textW, textH := d.TextExtent(l.Text())
	textX := x
	switch l.Alignment() {
	case wui.AlignCenter:
		textX = x + (w-textW)/2
	case wui.AlignRight:
		textX = x + w - textW
	}
	d.PushDrawRegion(x, y, w, h)
	if l.HasBackgroundColor() {
		d.FillRect(x, y, w, h, l.BackgroundColor())
	}
	d.TextOut(textX, y+(h-textH)/2, l.Text(), fgOf(l, wui.RGB(0, 0, 0)))
	d.PopDrawRegion()
}

func drawPaintBox(p *wui.PaintBox, d drawer) {
	x, y, w, h := p.Bounds()
	if w > 0 && h > 0 {
		d.DrawRect(x, y, w, h, wui.RGB(0, 0, 0))
		d.TextRectFormat(x, y, w, h, "Paint Box", wui.FormatCenter, wui.RGB(0, 0, 0))
	}
}

func drawIntUpDown(e *wui.IntUpDown, d drawer) {
	x, y, w, h := e.Bounds()
	if w > 0 && h > 0 {
		d.PushDrawRegion(x, y, w, h)
		d.DrawRect(x, y, w, h, wui.RGB(122, 122, 122))
		d.FillRect(x+1, y+1, w-2, h-2, wui.RGB(255, 255, 255))

		text := strconv.Itoa(e.Value())
		color := wui.RGB(0, 0, 0)
		d.TextOut(x+6, y+3, text, color)

		d.FillRect(x+w-19, y, 19, h, wui.RGB(231, 231, 231))
		d.DrawRect(x+w-19, y, 19, h, wui.RGB(172, 172, 172))
		d.DrawRect(x+w-19+2, y+2, 19-4, h-4, wui.RGB(172, 172, 172))
		d.DrawRect(x+w-19+2, y+h/2-1, 19-4, 2, wui.RGB(172, 172, 172))
		y1 := y + h/4
		d.Line(x+w-12, y1+2, x+w-12+5, y1+2, wui.RGB(0, 0, 0))
		d.Line(x+w-11, y1+1, x+w-11+3, y1+1, wui.RGB(0, 0, 0))
		d.Line(x+w-10, y1+0, x+w-10+1, y1+0, wui.RGB(0, 0, 0))
		y2 := y + 3*h/4 - 2
		d.Line(x+w-12, y2+0, x+w-12+5, y2+0, wui.RGB(0, 0, 0))
		d.Line(x+w-11, y2+1, x+w-11+3, y2+1, wui.RGB(0, 0, 0))
		d.Line(x+w-10, y2+2, x+w-10+1, y2+2, wui.RGB(0, 0, 0))
		d.PopDrawRegion()
	}
}

func drawFloatUpDown(e *wui.FloatUpDown, d drawer) {
	x, y, w, h := e.Bounds()
	if w > 0 && h > 0 {
		d.PushDrawRegion(x, y, w, h)
		d.DrawRect(x, y, w, h, wui.RGB(122, 122, 122))
		d.FillRect(x+1, y+1, w-2, h-2, wui.RGB(255, 255, 255))

		text := fmt.Sprintf("%."+strconv.Itoa(e.Precision())+"f", e.Value())
		color := wui.RGB(0, 0, 0)
		d.TextOut(x+6, y+3, text, color)

		d.FillRect(x+w-19, y, 19, h, wui.RGB(231, 231, 231))
		d.DrawRect(x+w-19, y, 19, h, wui.RGB(172, 172, 172))
		d.DrawRect(x+w-19+2, y+2, 19-4, h-4, wui.RGB(172, 172, 172))
		d.DrawRect(x+w-19+2, y+h/2-1, 19-4, 2, wui.RGB(172, 172, 172))
		y1 := y + h/4
		d.Line(x+w-12, y1+2, x+w-12+5, y1+2, wui.RGB(0, 0, 0))
		d.Line(x+w-11, y1+1, x+w-11+3, y1+1, wui.RGB(0, 0, 0))
		d.Line(x+w-10, y1+0, x+w-10+1, y1+0, wui.RGB(0, 0, 0))
		y2 := y + 3*h/4 - 2
		d.Line(x+w-12, y2+0, x+w-12+5, y2+0, wui.RGB(0, 0, 0))
		d.Line(x+w-11, y2+1, x+w-11+3, y2+1, wui.RGB(0, 0, 0))
		d.Line(x+w-10, y2+2, x+w-10+1, y2+2, wui.RGB(0, 0, 0))
		d.PopDrawRegion()
	}
}

func drawComboBox(c *wui.ComboBox, d drawer) {
	x, y, w, h := c.Bounds()
	if w > 0 && h > 0 {
		d.PushDrawRegion(x, y, w, h)
		d.DrawRect(x, y, w, h, wui.RGB(173, 173, 173))
		d.FillRect(x+1, y+1, w-2, h-2, bgOf(c, wui.RGB(225, 225, 225)))
		arrowX := x + w - 13
		arrowY := y + 9
		d.Line(arrowX, arrowY, arrowX+4, arrowY+4, wui.RGB(86, 86, 86))
		d.Line(arrowX+4, arrowY+3, arrowX+8, arrowY-1, wui.RGB(86, 86, 86))
		if w > 20 {
			i := c.SelectedIndex()
			items := c.Items()
			if 0 <= i && i < len(items) {
				text := items[i]
				d.PushDrawRegion(x, y, w-20, h)
				d.TextOut(x+4, y+4, text, fgOf(c, wui.RGB(0, 0, 0)))
				d.PopDrawRegion()
			}
		}
		d.PopDrawRegion()
	}
}

func drawProgressBar(p *wui.ProgressBar, d drawer) {
	x, y, w, h := p.Bounds()
	if w > 0 && h > 0 {
		d.PushDrawRegion(x, y, w, h)
		d.DrawRect(x, y, w, h, wui.RGB(188, 188, 188))
		d.FillRect(x+1, y+1, w-2, h-2, wui.RGB(230, 230, 230))
		if p.MovesForever() {
			if p.Vertical() {
				filledH := (h - 2) / 2
				d.FillRect(x+1, y+1+filledH/2, w-2, filledH, wui.RGB(0, 180, 40))
			} else {
				filledW := (w - 2) / 2
				d.FillRect(x+1+filledW/2, y+1, filledW, h-2, wui.RGB(0, 180, 40))
			}
		} else {
			if p.Vertical() {
				filledH := int(float64(h-2)*p.Value() + 0.5)
				d.FillRect(x+1, y+h-1-filledH, w-2, filledH, wui.RGB(0, 180, 40))
			} else {
				filledW := int(float64(w-2)*p.Value() + 0.5)
				d.FillRect(x+1, y+1, filledW, h-2, wui.RGB(0, 180, 40))
			}
		}
		d.PopDrawRegion()
	}
}

func drawEditLine(e *wui.EditLine, d drawer) {
	x, y, w, h := e.Bounds()
	if w > 0 && h > 0 {
		d.PushDrawRegion(x, y, w, h)
		if e.Enabled() {
			d.DrawRect(x, y, w, h, wui.RGB(122, 122, 122))
		} else {
			d.DrawRect(x, y, w, h, wui.RGB(204, 204, 204))
		}
		d.FillRect(x+1, y+1, w-2, h-2, bgOf(e, wui.RGB(255, 255, 255)))
		if !e.HasBackgroundColor() && (e.ReadOnly() || !e.Enabled()) {
			d.FillRect(x+2, y+2, w-4, h-4, wui.RGB(240, 240, 240))
		}
		text := e.Text()
		if e.IsPassword() {
			text = strings.Repeat("●", utf8.RuneCountInString(text))
		}
		color := fgOf(e, wui.RGB(0, 0, 0))
		if !e.Enabled() && !e.HasTextColor() {
			color = wui.RGB(109, 109, 109)
		}
		d.TextOut(x+6, y+3, text, color)
		d.PopDrawRegion()
	}
}

func drawTextEdit(t *wui.TextEdit, d drawer) {
	x, y, w, h := t.Bounds()
	if w > 0 && h > 0 {
		d.PushDrawRegion(x, y, w, h)
		if t.Enabled() {
			d.DrawRect(x, y, w, h, wui.RGB(122, 122, 122))
		} else {
			d.DrawRect(x, y, w, h, wui.RGB(204, 204, 204))
		}
		d.FillRect(x+1, y+1, w-2, h-2, bgOf(t, wui.RGB(255, 255, 255)))
		if !t.HasBackgroundColor() && !t.Enabled() {
			d.FillRect(x+2, y+2, w-4, h-4, wui.RGB(240, 240, 240))
		}
		color := fgOf(t, wui.RGB(0, 0, 0))
		if !t.Enabled() && !t.HasTextColor() {
			color = wui.RGB(109, 109, 109)
		}
		if t.WordWrap() {
			d.TextRectFormat(x+6, y+3, w-6, h-3, t.Text(), wui.FormatTopLeft, color)
		} else {
			d.TextOut(x+6, y+3, t.Text(), color)
		}
		d.PopDrawRegion()
	}
}

type colored interface {
	HasTextColor() bool
	TextColor() wui.Color
	HasBackgroundColor() bool
	BackgroundColor() wui.Color
}

// fgOf returns the custom text color of the control or the default.
func fgOf(c colored, def wui.Color) wui.Color {
	if c.HasTextColor() {
		return c.TextColor()
	}
	return def
}

// bgOf returns the custom background color of the control or the default.
func bgOf(c colored, def wui.Color) wui.Color {
	if c.HasBackgroundColor() {
		return c.BackgroundColor()
	}
	return def
}

func drawGroupBox(g *wui.GroupBox, d drawer) {
	x, y, w, h := g.Bounds()
	if w <= 4 || h <= 8 {
		return
	}
	d.PushDrawRegion(x, y, w, h)
	d.SetFont(getFont(g))
	textW, textH := d.TextExtent(g.Text())
	top := y + textH/2
	line := wui.RGB(220, 220, 220)
	d.Line(x, top, x+7, top, line)
	d.Line(x+10+textW, top, x+w-1, top, line)
	d.Line(x, top, x, y+h-1, line)
	d.Line(x, y+h-1, x+w-1, y+h-1, line)
	d.Line(x+w-1, top, x+w-1, y+h-1, line)
	d.TextOut(x+9, y, g.Text(), fgOf(g, wui.RGB(0, 0, 0)))
	d.PopDrawRegion()
}

func drawStatusBar(s *wui.StatusBar, d drawer) {
	x, y, w, h := s.Bounds()
	if w <= 0 || h <= 0 {
		return
	}
	d.PushDrawRegion(x, y, w, h)
	d.FillRect(x, y, w, h, wui.RGB(240, 240, 240))
	d.Line(x, y, x+w, y, wui.RGB(215, 215, 215))
	d.SetFont(getFont(s))
	_, textH := d.TextExtent(s.Text())
	d.TextOut(x+6, y+(h-textH)/2, s.Text(), wui.RGB(0, 0, 0))
	d.PopDrawRegion()
}

func drawTabControl(t *wui.TabControl, d drawer) {
	x, y, w, h := t.Bounds()
	if w <= 0 || h <= 0 {
		return
	}
	d.PushDrawRegion(x, y, w, h)
	d.SetFont(getFont(t))
	const headerH = 22
	line := wui.RGB(217, 217, 217)
	d.DrawRect(x, y+headerH-1, w, h-headerH+1, line)
	tabX := x + 2
	for i, title := range t.Tabs() {
		tw, th := d.TextExtent(title)
		tabW := tw + 16
		if i == t.SelectedIndex() {
			d.FillRect(tabX, y, tabW, headerH, wui.RGB(255, 255, 255))
		} else {
			d.FillRect(tabX, y+2, tabW, headerH-3, wui.RGB(240, 240, 240))
		}
		d.DrawRect(tabX, y, tabW, headerH, line)
		d.TextOut(tabX+8, y+(headerH-th)/2, title, wui.RGB(0, 0, 0))
		tabX += tabW
	}
	d.PopDrawRegion()
}

func drawDatePicker(p *wui.DatePicker, d drawer) {
	x, y, w, h := p.Bounds()
	if w <= 0 || h <= 0 {
		return
	}
	d.PushDrawRegion(x, y, w, h)
	d.DrawRect(x, y, w, h, wui.RGB(122, 122, 122))
	d.FillRect(x+1, y+1, w-2, h-2, wui.RGB(255, 255, 255))
	text := "10/31/2026"
	switch p.Mode() {
	case wui.DatePickerLongDate:
		text = "Saturday, October 31, 2026"
	case wui.DatePickerTime:
		text = "12:30:00"
	}
	_, th := d.TextExtent(text)
	d.PushDrawRegion(x, y, w-19, h)
	d.TextOut(x+4, y+(h-th)/2, text, wui.RGB(0, 0, 0))
	d.PopDrawRegion()
	if p.Mode() != wui.DatePickerTime {
		ax := x + w - 12
		ay := y + h/2 - 1
		d.Line(ax, ay, ax+4, ay+4, wui.RGB(86, 86, 86))
		d.Line(ax+4, ay+3, ax+8, ay-1, wui.RGB(86, 86, 86))
	}
	d.PopDrawRegion()
}

type node interface {
	Parent() wui.Container
	Bounds() (x, y, width, height int)
	SetBounds(x, y, width, height int)
}

func showPreview(parent, w *wui.Window, x, y int) {
	// Create a centered progress dialog that cannot be closed until the preview
	// is shown.
	canClose := make(chan bool, 1)
	progress := wui.NewWindow()
	progress.SetHasMinButton(false)
	progress.SetHasMaxButton(false)
	progress.SetHasCloseButton(false)
	progress.SetResizable(false)
	progress.SetTitle("Generating Preview...")
	progress.DisableAltF4()
	progress.SetOnCanClose(func() bool {
		return <-canClose
	})
	progress.SetInnerSize(420, 50)
	progress.SetX(parent.X() + (parent.Width()-progress.Width())/2)
	progress.SetY(parent.Y() + (parent.Height()-progress.Height())/2)
	p := wui.NewProgressBar()
	p.SetMovesForever(true)
	p.SetBounds(10, 10, 400, 30)
	progress.Add(p)

	// Generate the code in a different go routine while the progress bar is
	// showing.
	go func() {
		defer func() {
			canClose <- true
			progress.Close()
		}()

		// For the preview we set a temporary window position to align it with
		// the preview shown in the designer.
		oldX, oldY := w.Position()
		w.SetPosition(x, y)
		code, genErr := generateCode(w, true)
		w.SetPosition(oldX, oldY)
		if genErr != nil {
			wui.MessageBoxError("Error", genErr.Error())
			return
		}

		// Write the Go file to our temporary build dir.
		goFile := filepath.Join(buildDir, "wui_designer_temp_file.go")
		err := ioutil.WriteFile(goFile, code, 0666)
		if err != nil {
			wui.MessageBoxError("Error", err.Error())
			return
		}
		defer os.Remove(goFile)

		// Build the executable into our temporary build dir.
		exeFile := filepath.Join(buildDir, buildPrefix+strconv.Itoa(buildCount)+".exe")
		buildCount++

		// Do the build synchronously and report any build errors.
		var tryOutput []byte
		var tryErr error
		try := func(cmd string, args ...string) {
			if err != nil {
				return
			}
			command := exec.Command(cmd, args...)
			command.Dir = buildDir
			tryOutput, tryErr = command.CombinedOutput()
		}
		try("go", "mod", "init", "temp/wui/preview")
		try("go", "mod", "tidy")
		try("go", "build", "-o", exeFile, goFile)
		if tryErr != nil {
			wui.MessageBoxError("Error", tryErr.Error()+"\r\n"+string(tryOutput))
			return
		}

		// Start the program in parallel so we can have multiple previews open at
		// once.
		exec.Command(exeFile).Start()
	}()

	progress.ShowModal()
}

func generateCode(w *wui.Window, isPreview bool) ([]byte, error) {
	// TODO Remove the isPreview parameter once we can set window shortcuts
	// through the UI and generate them. Once we have that, temporarily add this
	// shortcut before generating the preview code and reset it afterwards, as
	// is done with the window position.
	var code bytes.Buffer
	var allCode []string
	for _, c := range events {
		allCode = append(allCode, c)
	}
	imports := goImports(strings.Join(allCode, "\n"))
	code.WriteString("package main\n\nimport (\n")
	for _, pkg := range imports {
		code.WriteString("\t" + strconv.Quote(pkg) + "\n")
	}
	code.WriteString("\t\"github.com/2dprototype/wui\"\n)\n\nfunc main() {")

	line := func(format string, a ...interface{}) {
		fmt.Fprint(&code, "\n")
		fmt.Fprintf(&code, format, a...)
	}

	name := names[w]
	if name == "" {
		name = defaultName(w)
	}
	writeControl(w, "", name, line)
	line("")
	if isPreview {
		line(name + ".SetShortcut(" + name + ".Close, wui.KeyEscape)")
	}
	line(name + ".Show()")
	code.WriteString("\n}")

	formatted, err := format.Source(code.Bytes())
	if err != nil {
		return nil, fmt.Errorf("the generated code is not valid Go, check the event code:\n%v", err)
	}
	return formatted, nil
}

func writeControl(c interface{}, parentName, name string, line func(format string, a ...interface{})) {
	do := func(format string, a ...interface{}) {
		line(name+format, a...)
	}

	var fontName string
	if f, ok := c.(fonter); ok {
		font := f.Font()
		if font != nil {
			fontName = name + "Font"
			line(fontName + ", _ := wui.NewFont(wui.FontDesc{")
			if font.Desc.Name != "" {
				line("Name: %q,", font.Desc.Name)
			}
			if font.Desc.Height != 0 {
				line("Height: %d,", font.Desc.Height)
			}
			if font.Desc.Bold {
				line("Bold: true,")
			}
			if font.Desc.Italic {
				line("Italic: true,")
			}
			if font.Desc.Underlined {
				line("Underlined: true,")
			}
			if font.Desc.StrikedOut {
				line("StrikedOut: true,")
			}
			line("})")
			line("")
		}
	}

	typeName := reflect.TypeOf(c).Elem().Name()
	ctorArgs := ""
	if typeName == "ScrollBar" {
		// NewScrollBar takes the orientation, which is set by the Vertical
		// property afterwards.
		ctorArgs = "false"
	}
	do(" := wui.New%s(%s)", typeName, ctorArgs)

	if fontName != "" {
		do(".SetFont(%s)", fontName)
	}

	setters := generateProperties(name, c)
	for _, setter := range setters {
		line("\t" + setter)
	}
	for _, setter := range extraGoLines(name, c) {
		line("\t" + setter)
	}
	if parentName != "" {
		line("%s.Add(%s)", parentName, name)
	}
	line("")

	for _, ev := range eventsOf(c) {
		if code := events[event{control: c, name: ev}]; !isEmptyHandler(code) {
			do(".Set"+ev+"(%s)", code)
		}
	}

	if con, ok := c.(wui.Container); ok {
		for _, child := range con.Children() {
			childName := names[child]
			if childName == "" {
				childName = defaultName(child)
			}
			writeControl(child, name, childName, line)
		}
	}
}

// cloneTree copies a control the way copy and paste needs it: including the
// colors, the font, the anchors, enabled and visible and all child controls.
func cloneTree(c wui.Control) wui.Control {
	n := cloneControl(c)
	h, v := c.Anchors()
	n.SetHorizontalAnchor(h)
	n.SetVerticalAnchor(v)
	if src, ok := c.(enabler); ok {
		if dst, ok := n.(enabler); ok {
			dst.SetEnabled(src.Enabled())
		}
	}
	if src, ok := c.(visibler); ok {
		if dst, ok := n.(visibler); ok {
			dst.SetVisible(src.Visible())
		}
	}
	if src, ok := c.(fonter); ok {
		if dst, ok := n.(fonter); ok && src.Font() != nil {
			dst.SetFont(src.Font())
		}
	}
	if src, ok := c.(wui.Container); ok {
		if dst, ok := n.(wui.Container); ok {
			for _, child := range src.Children() {
				cc := cloneTree(child)
				x, y, _, _ := child.Bounds()
				_, _, cw, ch := cc.Bounds()
				cc.SetBounds(x, y, cw, ch)
				dst.Add(cc)
			}
		}
	}
	return n
}

// cloneControl copies a control including its colors.
func cloneControl(c wui.Control) wui.Control {
	n := cloneControlBase(c)
	copyExtras(n, c)
	for _, ev := range eventsOf(c) {
		if code := events[event{control: c, name: ev}]; code != "" {
			events[event{control: n, name: ev}] = code
		}
	}
	if src, ok := c.(colored); ok {
		if dst, ok := n.(interface {
			SetTextColor(wui.Color)
			SetBackgroundColor(wui.Color)
		}); ok {
			if src.HasTextColor() {
				dst.SetTextColor(src.TextColor())
			}
			if src.HasBackgroundColor() {
				dst.SetBackgroundColor(src.BackgroundColor())
			}
		}
	}
	return n
}

func cloneControlBase(c wui.Control) wui.Control {
	// TODO Use the properties for this.
	switch x := c.(type) {
	case *wui.Button:
		b := wui.NewButton()
		b.SetText(x.Text())
		b.SetBounds(0, 0, x.Width(), x.Height())
		return b
	case *wui.CheckBox:
		c := wui.NewCheckBox()
		c.SetText(x.Text())
		c.SetChecked(x.Checked())
		c.SetBounds(0, 0, x.Width(), x.Height())
		return c
	case *wui.RadioButton:
		r := wui.NewRadioButton()
		r.SetText(x.Text())
		r.SetBounds(0, 0, x.Width(), x.Height())
		return r
	case *wui.Slider:
		s := wui.NewSlider()
		s.SetMinMax(x.MinMax())
		s.SetCursorPosition(x.CursorPosition())
		s.SetTickFrequency(x.TickFrequency())
		s.SetArrowIncrement(x.ArrowIncrement())
		s.SetMouseIncrement(x.MouseIncrement())
		s.SetTicksVisible(x.TicksVisible())
		s.SetOrientation(x.Orientation())
		s.SetTickPosition(x.TickPosition())
		s.SetBounds(0, 0, x.Width(), x.Height())
		return s
	case *wui.Panel:
		p := wui.NewPanel()
		p.SetBorderStyle(x.BorderStyle())
		p.SetBounds(0, 0, x.Width(), x.Height())
		return p
	case *wui.Label:
		l := wui.NewLabel()
		l.SetText(x.Text())
		l.SetAlignment(x.Alignment())
		l.SetBounds(0, 0, x.Width(), x.Height())
		return l
	case *wui.PaintBox:
		p := wui.NewPaintBox()
		p.SetBounds(0, 0, x.Width(), x.Height())
		return p
	case *wui.EditLine:
		e := wui.NewEditLine()
		e.SetBounds(0, 0, x.Width(), x.Height())
		e.SetText(x.Text())
		e.SetIsPassword(x.IsPassword())
		e.SetCharacterLimit(x.CharacterLimit())
		e.SetReadOnly(x.ReadOnly())
		return e
	case *wui.IntUpDown:
		n := wui.NewIntUpDown()
		n.SetBounds(0, 0, x.Width(), x.Height())
		n.SetMinMax(x.MinMax())
		n.SetValue(x.Value())
		return n
	case *wui.ComboBox:
		c := wui.NewComboBox()
		c.SetItems(x.Items())
		c.SetSelectedIndex(x.SelectedIndex())
		c.SetBounds(0, 0, x.Width(), x.Height())
		return c
	case *wui.ProgressBar:
		p := wui.NewProgressBar()
		p.SetValue(x.Value())
		p.SetVertical(x.Vertical())
		p.SetMovesForever(x.MovesForever())
		p.SetBounds(0, 0, x.Width(), x.Height())
		return p
	case *wui.FloatUpDown:
		f := wui.NewFloatUpDown()
		f.SetBounds(0, 0, x.Width(), x.Height())
		f.SetMinMax(x.MinMax())
		f.SetPrecision(x.Precision())
		f.SetValue(x.Value())
		return f
	case *wui.TextEdit:
		t := wui.NewTextEdit()
		t.SetBounds(0, 0, x.Width(), x.Height())
		t.SetCharacterLimit(x.CharacterLimit())
		t.SetWordWrap(x.WordWrap())
		t.SetText(x.Text())
		return t
	case *wui.GroupBox:
		g := wui.NewGroupBox()
		g.SetText(x.Text())
		g.SetBounds(0, 0, x.Width(), x.Height())
		return g
	case *wui.StatusBar:
		s := wui.NewStatusBar()
		s.SetText(x.Text())
		s.SetBounds(0, 0, x.Width(), x.Height())
		return s
	case *wui.TabControl:
		t := wui.NewTabControl()
		t.SetTabs(x.Tabs())
		t.SetSelectedIndex(x.SelectedIndex())
		t.SetBounds(0, 0, x.Width(), x.Height())
		return t
	case *wui.DatePicker:
		p := wui.NewDatePicker()
		p.SetMode(x.Mode())
		p.SetBounds(0, 0, x.Width(), x.Height())
		return p
	default:
		return cloneByProperties(c)
	}
}

// cloneByProperties copies the controls that have no case in cloneControlBase
// with the property list of their type. The extra properties are copied by
// cloneControl.
func cloneByProperties(c wui.Control) wui.Control {
	var n wui.Control
	switch c.(type) {
	case *wui.ListView:
		n = wui.NewListView()
	case *wui.RichEdit:
		n = wui.NewRichEdit()
	case *wui.LinkLabel:
		n = wui.NewLinkLabel()
	case *wui.MonthCalendar:
		n = wui.NewMonthCalendar()
	case *wui.HotKeyEdit:
		n = wui.NewHotKeyEdit()
	case *wui.IPAddressEdit:
		n = wui.NewIPAddressEdit()
	case *wui.ImageView:
		n = wui.NewImageView()
	case *wui.ScrollPanel:
		n = wui.NewScrollPanel()
	case *wui.ScrollBar:
		n = wui.NewScrollBar(false)
	case *wui.TreeView:
		n = wui.NewTreeView()
	default:
		panic("unhandled control type in cloneControl: " + typeNameOf(c))
	}
	_, _, w, h := c.Bounds()
	n.SetBounds(0, 0, w, h)
	copyProperties(n, c)
	return n
}

// copyProperties copies the properties of the property list that differ from
// their default. Position, anchors, colors, enabled and visible are copied by
// cloneTree and cloneControl.
func copyProperties(dst, src interface{}) {
	var def interface{}
	var list []property
	for d, props := range properties {
		if reflect.TypeOf(src) == reflect.TypeOf(d) {
			def, list = d, props
			break
		}
	}
	for _, p := range list {
		if len(p.combines) > 0 || p.skipDefault || savedByBounds[p.name] {
			continue
		}
		switch p.name {
		case "Enabled", "Visible", "HorizontalAnchor", "VerticalAnchor":
			continue
		}
		if hasDefaultValue(src, def, p.name) {
			continue
		}
		get := reflect.ValueOf(src).MethodByName(p.name)
		set := reflect.ValueOf(dst).MethodByName("Set" + p.name)
		if !get.IsValid() || !set.IsValid() || get.Type().NumOut() != 1 || set.Type().NumIn() != 1 {
			continue
		}
		out := get.Call(nil)[0]
		if !out.Type().AssignableTo(set.Type().In(0)) {
			continue
		}
		set.Call([]reflect.Value{out})
	}
}

// realParent returns the container that holds the control in the design. A
// control inside of a ScrollPanel reports the Panel that is embedded in the
// ScrollPanel as its parent, this returns the ScrollPanel itself.
func realParent(root wui.Container, c node) wui.Container {
	p := c.Parent()
	if pp, ok := p.(*wui.Panel); ok {
		if sp := findScrollPanel(root, pp); sp != nil {
			return sp
		}
	}
	return p
}

func findScrollPanel(c wui.Container, pp *wui.Panel) *wui.ScrollPanel {
	for _, child := range c.Children() {
		if sp, ok := child.(*wui.ScrollPanel); ok && &sp.Panel == pp {
			return sp
		}
		if con, ok := child.(wui.Container); ok {
			if r := findScrollPanel(con, pp); r != nil {
				return r
			}
		}
	}
	return nil
}

type enabler interface {
	Enabled() bool
	SetEnabled(bool)
}

type visibler interface {
	Visible() bool
	SetVisible(bool)
}

type fonter interface {
	Font() *wui.Font
	SetFont(*wui.Font)
}

func findContainerAt(c wui.Container, x, y int) (innerMost wui.Container, atX, atY int) {
	for _, child := range c.Children() {
		if container, ok := child.(wui.Container); ok {
			if innerContains(container, x, y) {
				dx, dy, _, _ := container.InnerBounds()
				return findContainerAt(container, x-dx, y-dy)
			}
		}
	}
	return c, x, y
}

// removeEmptyStrings changes the given input slice.
func removeEmptyStrings(items []string) []string {
	n := 0
	for i := range items {
		if items[i] != "" {
			items[n] = items[i]
			n++
		}
	}
	return items[:n]
}

func defaultName(of interface{}) string {
	typ := strings.TrimPrefix(reflect.TypeOf(of).String(), "*wui.")
	prefix := decapitalize(typ)
	i := 1
	for {
		name := prefix + strconv.Itoa(i)
		if !nameUsed(name) {
			return name
		}
		i++
	}
}

func decapitalize(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(r)) + s[size:]
}

func nameUsed(name string) bool {
	for _, n := range names {
		if name == n {
			return true
		}
	}
	return false
}

type fontControl interface {
	Font() *wui.Font
	Parent() wui.Container
}

func getFont(f fontControl) *wui.Font {
	if f == nil {
		return nil
	}
	font := f.Font()
	if font != nil {
		return font
	}
	return getFont(f.Parent())
}

// relativeBounds returns a control's outer bounds relative to the outer bounds
// of the given container. If the control is the same as the container this will
// result in x and y being 0.
func relativeBounds(of node, in wui.Container) (x, y, width, height int) {
	x, y, width, height = of.Bounds()
	parent := of.Parent()
	for parent != nil {
		innerX, innerY, _, _ := parent.InnerBounds()
		x += innerX
		y += innerY
		parent = parent.Parent()
	}
	dx, dy, _, _ := in.Bounds()
	x -= dx
	y -= dy
	return
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
