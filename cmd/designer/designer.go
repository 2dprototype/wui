package main

import (
	"bytes"
	"encoding/json"
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

// TODO Have color property for window background.
// TODO Have icon for window.
// TODO Have cursor properties for all controls, first let all controls have changeable cursors.
// TODO Edit main menu.
// TODO Have a way to edit shortcuts.
// TODO Make edit lines select the whole text when they receive focus.
// TODO Un-highlight the template when the mouse leaves the palette area, even when it leaves it fast.
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

// ProjectData represents the serializable project state
type ProjectData struct {
	Windows []WindowData `json:"windows"`
	Events  []EventData  `json:"events"`
}

type WindowData struct {
	Name     string        `json:"name"`
	Type     string        `json:"type"`
	Props    []PropData    `json:"props"`
	Font     *FontData     `json:"font,omitempty"`
	Children []ControlData `json:"children,omitempty"`
}

type ControlData struct {
	Name     string        `json:"name"`
	Type     string        `json:"type"`
	Props    []PropData    `json:"props"`
	Font     *FontData     `json:"font,omitempty"`
	Children []ControlData `json:"children,omitempty"`
}

type PropData struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

type FontData struct {
	Name       string `json:"name"`
	Height     int    `json:"height"`
	Bold       bool   `json:"bold"`
	Italic     bool   `json:"italic"`
	Underlined bool   `json:"underlined"`
	StrikedOut bool   `json:"striked_out"`
}

type EventData struct {
	ControlName string `json:"control_name"`
	EventName   string `json:"event_name"`
	Code        string `json:"code"`
}

func saveProject(w *wui.Window, filePath string) error {
	data := ProjectData{}
	windowData := serializeWindow(w)
	data.Windows = append(data.Windows, windowData)
	
	for evt, code := range events {
		controlName := names[evt.control]
		if controlName == "" {
			continue
		}
		data.Events = append(data.Events, EventData{
			ControlName: controlName,
			EventName:   evt.name,
			Code:        code,
		})
	}
	
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	
	return ioutil.WriteFile(filePath, jsonData, 0666)
}

func serializeWindow(w *wui.Window) WindowData {
	data := WindowData{
		Name: names[w],
		Type: reflect.TypeOf(w).Elem().Name(),
	}
	data.Props = serializeProperties(w)
	if font := w.Font(); font != nil {
		data.Font = serializeFont(font)
	}
	for _, child := range w.Children() {
		data.Children = append(data.Children, serializeControl(child))
	}
	return data
}

func serializeControl(c wui.Control) ControlData {
	data := ControlData{
		Name: names[c],
		Type: reflect.TypeOf(c).Elem().Name(),
	}
	data.Props = serializeProperties(c)
	if f, ok := c.(fonter); ok {
		if font := f.Font(); font != nil {
			data.Font = serializeFont(font)
		}
	}
	if container, ok := c.(wui.Container); ok {
		for _, child := range container.Children() {
			data.Children = append(data.Children, serializeControl(child))
		}
	}
	return data
}

func serializeProperties(c interface{}) []PropData {
	var props []PropData
	for def, propList := range properties {
		if reflect.TypeOf(c) == reflect.TypeOf(def) {
			for _, prop := range propList {
				if len(prop.combines) > 0 {
					continue
				}
				method := reflect.ValueOf(c).MethodByName(prop.name)
				if !method.IsValid() {
					continue
				}
				results := method.Call(nil)
				if len(results) > 0 {
					props = append(props, PropData{
						Name:  prop.name,
						Value: results[0].Interface(),
					})
				}
			}
			break
		}
	}
	return props
}

func serializeFont(font *wui.Font) *FontData {
	return &FontData{
		Name:       font.Desc.Name,
		Height:     font.Desc.Height,
		Bold:       font.Desc.Bold,
		Italic:     font.Desc.Italic,
		Underlined: font.Desc.Underlined,
		StrikedOut: font.Desc.StrikedOut,
	}
}

func loadProject(filePath string) (*wui.Window, error) {
	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	var data ProjectData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, err
	}
	
	if len(data.Windows) == 0 {
		return nil, fmt.Errorf("no windows found in project file")
	}
	
	names = make(map[interface{}]string)
	events = make(map[event]string)
	
	window := deserializeWindow(data.Windows[0])
	
	for _, evt := range data.Events {
		var control interface{}
		findControlByName(window, evt.ControlName, &control)
		if control != nil {
			events[event{control: control, name: evt.EventName}] = evt.Code
		}
	}
	
	return window, nil
}

func deserializeWindow(data WindowData) *wui.Window {
	window := defaultWindow()
	names[window] = data.Name
	applyProperties(window, data.Props)
	if data.Font != nil {
		if font, err := deserializeFont(data.Font); err == nil {
			window.SetFont(font)
		}
	}
	for _, childData := range data.Children {
		child := deserializeControl(childData)
		window.Add(child)
	}
	return window
}

func deserializeControl(data ControlData) wui.Control {
	var control wui.Control
	switch data.Type {
	case "Button":
		control = wui.NewButton()
	case "CheckBox":
		control = wui.NewCheckBox()
	case "RadioButton":
		control = wui.NewRadioButton()
	case "Slider":
		control = wui.NewSlider()
	case "Panel":
		control = wui.NewPanel()
	case "Label":
		control = wui.NewLabel()
	case "PaintBox":
		control = wui.NewPaintBox()
	case "EditLine":
		control = wui.NewEditLine()
	case "IntUpDown":
		control = wui.NewIntUpDown()
	case "ComboBox":
		control = wui.NewComboBox()
	case "ProgressBar":
		control = wui.NewProgressBar()
	case "FloatUpDown":
		control = wui.NewFloatUpDown()
	case "TextEdit":
		control = wui.NewTextEdit()
	default:
		panic("unknown control type: " + data.Type)
	}
	
	names[control] = data.Name
	applyProperties(control, data.Props)
	
	if data.Font != nil {
		if f, ok := control.(fonter); ok {
			if font, err := deserializeFont(data.Font); err == nil {
				f.SetFont(font)
			}
		}
	}
	
	if container, ok := control.(wui.Container); ok {
		for _, childData := range data.Children {
			child := deserializeControl(childData)
			container.Add(child)
		}
	}
	
	return control
}

func applyProperties(c interface{}, props []PropData) {
	for _, prop := range props {
		setterName := "Set" + prop.Name
		method := reflect.ValueOf(c).MethodByName(setterName)
		if !method.IsValid() {
			continue
		}
		value := reflect.ValueOf(prop.Value)
		if method.Type().NumIn() > 0 {
			paramType := method.Type().In(0)
			if value.Type().ConvertibleTo(paramType) {
				value = value.Convert(paramType)
			}
		}
		method.Call([]reflect.Value{value})
	}
}

func deserializeFont(data *FontData) (*wui.Font, error) {
	return wui.NewFont(wui.FontDesc{
		Name:       data.Name,
		Height:     data.Height,
		Bold:       data.Bold,
		Italic:     data.Italic,
		Underlined: data.Underlined,
		StrikedOut: data.StrikedOut,
	})
}

func findControlByName(container wui.Container, name string, result *interface{}) {
	for _, child := range container.Children() {
		if names[child] == name {
			*result = child
			return
		}
		if subContainer, ok := child.(wui.Container); ok {
			findControlByName(subContainer, name, result)
			if *result != nil {
				return
			}
		}
	}
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

	theWindow := defaultWindow()
	names[theWindow] = "window"

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

	// Menu setup
	menu := wui.NewMainMenu()
	fileMenu := wui.NewMenu("&File")
	editMenu := wui.NewMenu("&Edit")
	viewMenu := wui.NewMenu("&View")
	projectMenu := wui.NewMenu("&Project")
	helpMenu := wui.NewMenu("&Help")

	// File menu items
	fileNewMenu := wui.NewMenuString("&New Project\tCtrl+N")
	fileOpenMenu := wui.NewMenuString("&Open Project...\tCtrl+O")
	fileSaveMenu := wui.NewMenuString("&Save Project\tCtrl+S")
	fileSaveAsMenu := wui.NewMenuString("Save Project &As...\tCtrl+Shift+S")
	fileExportGoMenu := wui.NewMenuString("&Export as Go...\tCtrl+E")
	fileExportJSONMenu := wui.NewMenuString("Export as &JSON...\tCtrl+J")
	previewMenu := wui.NewMenuString("&Run Preview\tF5")
	exitMenu := wui.NewMenuString("E&xit\tAlt+F4")

	// Edit menu items
	undoMenu := wui.NewMenuString("&Undo\tCtrl+Z")
	redoMenu := wui.NewMenuString("&Redo\tCtrl+Y")
	cutMenu := wui.NewMenuString("Cu&t\tCtrl+X")
	copyMenu := wui.NewMenuString("&Copy\tCtrl+C")
	pasteMenu := wui.NewMenuString("&Paste\tCtrl+V")
	deleteMenu := wui.NewMenuString("&Delete\tDel")
	selectAllMenu := wui.NewMenuString("Select &All\tCtrl+A")

	// View menu items
	viewZoomInMenu := wui.NewMenuString("Zoom &In\tCtrl++")
	viewZoomOutMenu := wui.NewMenuString("Zoom &Out\tCtrl+-")
	viewZoomResetMenu := wui.NewMenuString("&Reset Zoom\tCtrl+0")
	viewFullScreenMenu := wui.NewMenuString("&Full Screen\tF11")

	// Project menu items
	projectSettingsMenu := wui.NewMenuString("Project &Settings...\tAlt+Enter")
	projectPreviewMenu := wui.NewMenuString("&Preview Window\tF5")
	projectBuildMenu := wui.NewMenuString("&Build Preview\tCtrl+F5")

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
	fileMenu.Add(fileExportJSONMenu)
	fileMenu.Add(wui.NewMenuSeparator())
	fileMenu.Add(previewMenu)
	fileMenu.Add(wui.NewMenuSeparator())
	fileMenu.Add(exitMenu)

	// Build Edit menu
	editMenu.Add(undoMenu)
	editMenu.Add(redoMenu)
	editMenu.Add(wui.NewMenuSeparator())
	editMenu.Add(cutMenu)
	editMenu.Add(copyMenu)
	editMenu.Add(pasteMenu)
	editMenu.Add(wui.NewMenuSeparator())
	editMenu.Add(deleteMenu)
	editMenu.Add(wui.NewMenuSeparator())
	editMenu.Add(selectAllMenu)

	// Build View menu
	viewMenu.Add(viewZoomInMenu)
	viewMenu.Add(viewZoomOutMenu)
	viewMenu.Add(viewZoomResetMenu)
	viewMenu.Add(wui.NewMenuSeparator())
	viewMenu.Add(viewFullScreenMenu)

	// Build Project menu
	projectMenu.Add(projectSettingsMenu)
	projectMenu.Add(wui.NewMenuSeparator())
	projectMenu.Add(projectPreviewMenu)
	projectMenu.Add(projectBuildMenu)

	// Build Help menu
	helpMenu.Add(helpAboutMenu)
	helpMenu.Add(helpShortcutsMenu)

	// Add menus to main menu
	menu.Add(fileMenu)
	menu.Add(editMenu)
	menu.Add(viewMenu)
	menu.Add(projectMenu)
	menu.Add(helpMenu)
	w.SetMenu(menu)

	type uiProp struct {
		panel     *wui.Panel
		setter    string
		update    func()
		rightType func(t reflect.Type) bool
	}
	var updateProperties func()

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

	uiProps := []uiProp{
		stringProp("Title", "Title"),
		stringProp("Text", "Text"),
		enumProp("Window State", "State", "Normal", "Maximized", "Minimized"),
		boolProp("Min Button", "HasMinButton"),
		boolProp("Max Button", "HasMaxButton"),
		boolProp("Close Button", "HasCloseButton"),
		boolProp("Has Border", "HasBorder"),
		boolProp("Resizable", "Resizable"),
		intProp("Alpha", "Alpha", 0, 255),
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
		stringListProp("Items", "Items"),
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

	allTemplates := []wui.Control{
		panelTemplate, paintBoxTemplate, textEditTemplate,
		editLineTemplate, comboTemplate, sliderTemplate,
		progressTemplate, buttonTemplate, intTemplate,
		floatTemplate, checkBoxTemplate, radioButtonTemplate,
		labelTemplate,
	}

	var highlightedTemplate, controlToAdd wui.Control
	var templateDx, templateDy int

	palette := wui.NewPaintBox()
	palette.SetBounds(605, 0, 195, 600)
	palette.SetHorizontalAnchor(wui.AnchorMax)
	palette.SetVerticalAnchor(wui.AnchorMinAndMax)
	palette.SetOnPaint(func(c *wui.Canvas) {
		w, h := c.Size()
		c.FillRect(0, 0, w, h, wui.RGB(240, 240, 240))
		for _, template := range allTemplates {
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
		for _, c := range allTemplates {
			if contains(c, x, y) {
				highlightedTemplate = c
			}
		}
		if highlightedTemplate != oldHighlight {
			palette.Paint()
		}
	})
	w.Add(palette)

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

	editOnPaint := wui.NewButton()
	editOnPaint.SetText("OnPaint")
	editOnPaint.SetBounds(105, 500, 85, 25)
	editOnPaint.SetVisible(false)
	w.Add(editOnPaint)

	name.SetOnTextChange(func() {
		names[active] = name.Text()
		markModified()
	})
	
	editOnPaint.SetOnClick(func() {
		p, valid := active.(*wui.PaintBox)
		if !valid {
			panic("OnPaint only valid for paint boxes")
		}

		dlg := wui.NewWindow()
		dlg.SetPosition(w32.ClientToScreen(w32.HWND(preview.Handle()), 0, 0))
		dlg.SetSize(preview.Size())

		code := wui.NewTextEdit()
		font, _ := wui.NewFont(wui.FontDesc{Name: "Courier New", Height: -15})
		code.SetFont(font)
		code.SetWritesTabs(true)
		code.SetBounds(0, 0, dlg.InnerWidth(), dlg.InnerHeight()-30)
		code.SetAnchors(wui.AnchorMinAndMax, wui.AnchorMinAndMax)
		dlg.Add(code)

		onPaint := event{p, "OnPaint"}
		if events[onPaint] == "" {
			events[onPaint] = "func(canvas *wui.Canvas) {\n\t\n}"
		}
		code.SetText(strings.Replace(events[onPaint], "\n", "\r\n", -1))
		code.SetCursorPosition(29)

		ok := wui.NewButton()
		ok.SetText("OK")
		ok.SetBounds(dlg.InnerWidth()/2-87, dlg.InnerHeight()-28, 85, 25)
		ok.SetAnchors(wui.AnchorCenter, wui.AnchorMax)
		ok.SetOnClick(func() {
			events[onPaint] = strings.Replace(code.Text(), "\r", "", -1)
			dlg.Close()
			markModified()
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
	})

	updateProperties = func() {
		for _, prop := range uiProps {
			if prop.panel.Visible() {
				prop.update()
			}
		}
	}

	activate := func(newActive node) {
		active = newActive
		name.SetText(names[active])
		y := name.Y() + name.Height() + propMargin

		for _, prop := range uiProps {
			m, hasProp := reflect.TypeOf(active).MethodByName(prop.setter)
			show := hasProp && prop.rightType(m.Type.In(1))
			prop.panel.SetVisible(show)
			if show {
				prop.panel.SetY(y)
				y += prop.panel.Height()
			}
		}
		updateProperties()

		f, hasFont := active.(fonter)
		fontProps.SetVisible(hasFont)
		if hasFont {
			fontProps.SetY(y)
			y += fontProps.Height()
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
		xOffset = 20 - (theWindow.InnerX() - theWindow.X())
		yOffset = 40 - (theWindow.InnerY() - theWindow.Y())
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


	// New project
	fileNewMenu.SetOnClick(func() {
		if projectModified {
			result := wui.MessageBoxYesNo("Unsaved Changes", 
				"Do you want to save changes to the current project?")
			if result { // Yes
				fileSaveMenu.OnClick()()
			} 
		}
		
		theWindow = defaultWindow()
		names = make(map[interface{}]string)
		names[theWindow] = "window"
		events = make(map[event]string)
		activate(theWindow)
		preview.Paint()
		setWorkingPath("")
		markSaved()
	})

	// Open project
	fileOpenMenu.SetOnClick(func() {
		if projectModified {
			result := wui.MessageBoxYesNo("Unsaved Changes", 
				"Do you want to save changes to the current project?")
			if result {
				fileSaveMenu.OnClick()()
			} 
		}
		
		open := wui.NewFileOpenDialog()
		open.SetTitle("Open wui Designer Project")
		open.AddFilter("JSON project file", ".json")
		open.AddFilter("All files", "*")
		if accept, path := open.ExecuteSingleSelection(w); accept {
			newWindow, err := loadProject(path)
			if err != nil {
				wui.MessageBoxError("Error", "Failed to load project: "+err.Error())
				return
			}
			
			theWindow = newWindow
			activate(theWindow)
			preview.Paint()
			setWorkingPath(path)
			markSaved()
		}
	})

	// Save project
	fileSaveMenu.SetOnClick(func() {
		if workingPath != "" {
			saveProjectFile(workingPath)
		} else {
			fileSaveAsMenu.OnClick()()
		}
	})

	// Save project as
	fileSaveAsMenu.SetOnClick(func() {
		save := wui.NewFileSaveDialog()
		save.SetAppendExt(true)
		save.AddFilter("JSON project file", ".json")
		if accept, path := save.Execute(w); accept {
			saveProjectFile(path)
		}
	})

	// Export as Go
	fileExportGoMenu.SetOnClick(func() {
		save := wui.NewFileSaveDialog()
		save.SetAppendExt(true)
		save.AddFilter("Go file", ".go")
		if accept, path := save.Execute(w); accept {
			code := generateCode(theWindow, false)
			err := ioutil.WriteFile(path, code, 0666)
			if err != nil {
				wui.MessageBoxError("Error", err.Error())
			} else {
				wui.MessageBoxInfo("Success", "Go code exported to "+path)
			}
		}
	})

	// Export as JSON
	fileExportJSONMenu.SetOnClick(func() {
		save := wui.NewFileSaveDialog()
		save.SetAppendExt(true)
		save.AddFilter("JSON file", ".json")
		if accept, path := save.Execute(w); accept {
			if err := saveProject(theWindow, path); err != nil {
				wui.MessageBoxError("Error", "Failed to export project: "+err.Error())
			} else {
				wui.MessageBoxInfo("Success", "Project exported to "+path)
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
		if projectModified {
			result := wui.MessageBoxYesNo("Unsaved Changes", 
				"Do you want to save changes before exiting?")
			if result {
				if workingPath != "" {
					return saveProjectFile(workingPath)
				} else {
					save := wui.NewFileSaveDialog()
					save.SetAppendExt(true)
					save.AddFilter("JSON project file", ".json")
					if accept, path := save.Execute(w); accept {
						return saveProjectFile(path)
					}
					return false
				}
			} else {
				return true
			}
		}
		return true
	})

	// Edit operations
	undoMenu.SetOnClick(func() {
		wui.MessageBoxInfo("TODO", "Undo not yet implemented")
	})

	redoMenu.SetOnClick(func() {
		wui.MessageBoxInfo("TODO", "Redo not yet implemented")
	})

	cutMenu.SetOnClick(func() {
		wui.MessageBoxInfo("TODO", "Cut not yet implemented")
	})

	copyMenu.SetOnClick(func() {
		wui.MessageBoxInfo("TODO", "Copy not yet implemented")
	})

	pasteMenu.SetOnClick(func() {
		wui.MessageBoxInfo("TODO", "Paste not yet implemented")
	})

	deleteMenu.SetOnClick(func() {
		if active != nil && active != theWindow {
			result := wui.MessageBoxYesNo("Confirm Delete", 
				fmt.Sprintf("Delete '%s'?", names[active]))
			if result {
				c := active.(wui.Control)
				p := active.Parent()
				activate(p)
				p.Remove(c)
				preview.Paint()
				markModified()
			}
		}
	})

	selectAllMenu.SetOnClick(func() {
		wui.MessageBoxInfo("TODO", "Select All not yet implemented")
	})

	// View operations
	viewZoomInMenu.SetOnClick(func() {
		wui.MessageBoxInfo("TODO", "Zoom In not yet implemented")
	})

	viewZoomOutMenu.SetOnClick(func() {
		wui.MessageBoxInfo("TODO", "Zoom Out not yet implemented")
	})

	viewZoomResetMenu.SetOnClick(func() {
		wui.MessageBoxInfo("TODO", "Reset Zoom not yet implemented")
	})

	viewFullScreenMenu.SetOnClick(func() {
		if w.State() == wui.WindowMaximized {
			w.SetState(wui.WindowNormal)
		} else {
			w.SetState(wui.WindowMaximized)
		}
	})

	// Project operations
	projectSettingsMenu.SetOnClick(func() {
		wui.MessageBoxInfo("TODO", "Project Settings not yet implemented")
	})

	projectPreviewMenu.SetOnClick(func() {
		previewMenu.OnClick()()
	})

	projectBuildMenu.SetOnClick(func() {
		wui.MessageBoxInfo("TODO", "Build Preview not yet implemented")
	})

	// Help operations
	helpAboutMenu.SetOnClick(func() {
		wui.MessageBoxInfo("About wui Designer", 
			"wui Designer\n\n"+
			"A visual designer for wui GUI applications.\n\n"+
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
			"  Ctrl+J - Export as JSON\n"+
			"  F5 - Run Preview\n"+
			"  Alt+F4 - Exit\n\n"+
			"Edit Operations:\n"+
			"  Ctrl+Z - Undo\n"+
			"  Ctrl+Y - Redo\n"+
			"  Ctrl+X - Cut\n"+
			"  Ctrl+C - Copy\n"+
			"  Ctrl+V - Paste\n"+
			"  Del - Delete\n"+
			"  Ctrl+A - Select All\n\n"+
			"View Operations:\n"+
			"  Ctrl++ - Zoom In\n"+
			"  Ctrl+- - Zoom Out\n"+
			"  Ctrl+0 - Reset Zoom\n"+
			"  F11 - Full Screen\n\n"+
			"Project Operations:\n"+
			"  Alt+Enter - Project Settings\n"+
			"  F5 - Preview Window\n"+
			"  Ctrl+F5 - Build Preview\n\n"+
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
	w.SetShortcut(fileExportJSONMenu.OnClick(), wui.KeyControl, wui.KeyJ)
	w.SetShortcut(previewMenu.OnClick(), wui.KeyF5)
	w.SetShortcut(exitMenu.OnClick(), wui.KeyAlt, wui.KeyF4)

	w.SetShortcut(undoMenu.OnClick(), wui.KeyControl, wui.KeyZ)
	w.SetShortcut(redoMenu.OnClick(), wui.KeyControl, wui.KeyY)
	w.SetShortcut(cutMenu.OnClick(), wui.KeyControl, wui.KeyX)
	w.SetShortcut(copyMenu.OnClick(), wui.KeyControl, wui.KeyC)
	w.SetShortcut(pasteMenu.OnClick(), wui.KeyControl, wui.KeyV)
	w.SetShortcut(deleteMenu.OnClick(), wui.KeyDelete)
	w.SetShortcut(selectAllMenu.OnClick(), wui.KeyControl, wui.KeyA)

	// w.SetShortcut(viewZoomInMenu.OnClick(), wui.KeyControl, wui.KeyPlus)
	// w.SetShortcut(viewZoomOutMenu.OnClick(), wui.KeyControl, wui.KeyMinus)
	// w.SetShortcut(viewZoomResetMenu.OnClick(), wui.KeyControl, wui.Key0)
	// w.SetShortcut(viewFullScreenMenu.OnClick(), wui.KeyF11)

	// w.SetShortcut(projectSettingsMenu.OnClick(), wui.KeyAlt, wui.KeyEnter)
	w.SetShortcut(projectPreviewMenu.OnClick(), wui.KeyF5)
	w.SetShortcut(projectBuildMenu.OnClick(), wui.KeyControl, wui.KeyF5)

	w.SetShortcut(helpAboutMenu.OnClick(), wui.KeyF1)
	w.SetShortcut(helpShortcutsMenu.OnClick(), wui.KeyControl, wui.KeyF1)

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

func defaultWindow() *wui.Window {
	font, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -11})
	w := wui.NewWindow()
	w.SetFont(font)
	w.SetTitle("Window")
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
	default:
		panic("unhandled control type")
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
	d.FillRect(x, y, w, h, wui.RGB(240, 240, 240))
	d.FillEllipse(x, y+(h-13)/2, 13, 13, wui.RGB(255, 255, 255))
	d.DrawEllipse(x, y+(h-13)/2, 13, 13, wui.RGB(0, 0, 0))
	if r.Checked() {
		d.FillEllipse(x+3, y+(h-13)/2+3, 7, 7, wui.RGB(0, 0, 0))
	}
	_, textH := d.TextExtent(r.Text())
	d.TextOut(x+16, y+(h-textH)/2, r.Text(), wui.RGB(0, 0, 0))
	d.PopDrawRegion()
}

func drawCheckBox(c *wui.CheckBox, d drawer) {
	x, y, w, h := c.Bounds()
	d.PushDrawRegion(x, y, w, h)
	d.FillRect(x, y, w, h, wui.RGB(240, 240, 240))
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
	d.TextOut(x+16, y+(h-textH)/2, c.Text(), wui.RGB(0, 0, 0))
	d.PopDrawRegion()
}

func drawPanel(p *wui.Panel, d drawer) {
	x, y, w, h := p.Bounds()
	if w <= 0 || h <= 0 {
		return
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
	d.TextOut(textX, y+(h-textH)/2, l.Text(), wui.RGB(0, 0, 0))
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
		d.FillRect(x+1, y+1, w-2, h-2, wui.RGB(225, 225, 225))
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
				d.TextOut(x+4, y+4, text, wui.RGB(0, 0, 0))
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
		d.FillRect(x+1, y+1, w-2, h-2, wui.RGB(255, 255, 255))
		if e.ReadOnly() || !e.Enabled() {
			d.FillRect(x+2, y+2, w-4, h-4, wui.RGB(240, 240, 240))
		}
		text := e.Text()
		if e.IsPassword() {
			text = strings.Repeat("●", utf8.RuneCountInString(text))
		}
		color := wui.RGB(0, 0, 0)
		if !e.Enabled() {
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
		d.FillRect(x+1, y+1, w-2, h-2, wui.RGB(255, 255, 255))
		if !t.Enabled() {
			d.FillRect(x+2, y+2, w-4, h-4, wui.RGB(240, 240, 240))
		}
		color := wui.RGB(0, 0, 0)
		if !t.Enabled() {
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
		code := generateCode(w, true)
		w.SetPosition(oldX, oldY)

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

func generateCode(w *wui.Window, isPreview bool) []byte {
	// TODO Remove the isPreview parameter once we can set window shortcuts
	// through the UI and generate them. Once we have that, temporarily add this
	// shortcut before generating the preview code and reset it afterwards, as
	// is done with the window position.
	var code bytes.Buffer
	code.WriteString(`package main

import "github.com/2dprototype/wui"

func main() {`)

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
		panic("We generated wrong code: " + err.Error())
	}
	return formatted
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
	do(" := wui.New%s()", typeName)

	if fontName != "" {
		do(".SetFont(%s)", fontName)
	}

	setters := generateProperties(name, c)
	for _, setter := range setters {
		line("\t" + setter)
	}
	if parentName != "" {
		line("%s.Add(%s)", parentName, name)
	}
	line("")

	// TODO Generate ALL events.
	if p, ok := c.(*wui.PaintBox); ok {
		onPaint := event{p, "OnPaint"}
		if events[onPaint] != "" {
			do(".SetOnPaint(%s)", events[onPaint])
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

func cloneControl(c wui.Control) wui.Control {
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
	default:
		panic("unhandled control type in cloneControl")
	}
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
