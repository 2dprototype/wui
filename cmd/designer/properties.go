package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/2dprototype/wui"
)

type property struct {
	name     string
	combines []string

	// requires is the name of a bool getter. The property is only written to
	// the generated Go code if that getter returns true.
	requires string

	// skipDefault leaves the property out of project files when it has its
	// default value. Used for properties where setting a value, even the
	// default, changes the behavior (colors).
	skipDefault bool
}

func prop(name string, combines ...string) property {
	return property{name: name, combines: combines}
}

// propIf is a property that is only generated if the bool getter requires
// returns true.
func propIf(name, requires string) property {
	return property{name: name, requires: requires}
}

// propColor is a color property. Colors are only saved and generated when they
// differ from the default.
func propColor(name string) property {
	return property{name: name, skipDefault: true}
}

func commonPropertiesPlus(plus ...property) []property {
	return append(
		[]property{
			prop("Enabled"),
			prop("Visible"),
			prop("HorizontalAnchor"),
			prop("VerticalAnchor"),
			prop("Anchors", "HorizontalAnchor", "VerticalAnchor"),
			prop("X"),
			prop("Y"),
			prop("Position", "X", "Y"),
			prop("Width"),
			prop("Height"),
			prop("Size", "Width", "Height"),
			prop("Bounds", "Position", "Size"),
			prop("TabStop"),
		},
		plus...)
}

// scrollBarProperties are the common properties without Position: a ScrollBar
// has its own Position (the scroll position) that replaces the one of the
// other controls.
func scrollBarProperties(plus ...property) []property {
	return append(
		[]property{
			prop("Enabled"),
			prop("Visible"),
			prop("HorizontalAnchor"),
			prop("VerticalAnchor"),
			prop("Anchors", "HorizontalAnchor", "VerticalAnchor"),
			prop("X"),
			prop("Y"),
			prop("Width"),
			prop("Height"),
			prop("Size", "Width", "Height"),
			prop("Bounds", "X", "Y", "Size"),
			prop("TabStop"),
		},
		plus...)
}

var properties = map[interface{}][]property{
	wui.NewWindow(): []property{
		// We do not use the outer bounds for a Window since that is usually not
		// what the user wants. The inner size determines the layout of the
		// controls, also different Windows versions will have differently sized
		// bounds. Keeping the inner size constant makes the most sense.
		prop("InnerX"),
		prop("InnerY"),
		prop("InnerPosition", "InnerX", "InnerY"),
		prop("InnerWidth"),
		prop("InnerHeight"),
		prop("InnerSize", "InnerWidth", "InnerHeight"),
		prop("InnerBounds", "InnerPosition", "InnerSize"),
		prop("Title"),
		prop("Alpha"),
		prop("HasMinButton"),
		prop("HasMaxButton"),
		prop("HasCloseButton"),
		prop("HasBorder"),
		prop("Resizable"),
		prop("State"),
		propColor("BackgroundColor"),
		prop("TopMost"),
		prop("ShowInTaskbar"),
		prop("DragByBackground"),
		prop("CornerRadius"),
		prop("AcceptFiles"),
		// The color comes before the switch: setting a color turns the
		// transparency on, SetTransparent(false) turns it off again when a
		// project is loaded.
		propIf("TransparentColor", "Transparent"),
		prop("Transparent"),
		prop("TrayEnabled"),
		prop("TrayToolTip"),
		prop("MinimizeToTray"),
		prop("CloseToTray"),
		prop("CenterOnShow"),
	},

	wui.NewButton(): commonPropertiesPlus(
		prop("Text"),
	),

	wui.NewLabel(): commonPropertiesPlus(
		prop("Text"),
		prop("Alignment"),
		propColor("TextColor"),
		propColor("BackgroundColor"),
	),

	wui.NewCheckBox(): commonPropertiesPlus(
		prop("Text"),
		prop("Checked"),
		prop("ThreeState"),
		propColor("TextColor"),
		propColor("BackgroundColor"),
	),

	wui.NewRadioButton(): commonPropertiesPlus(
		prop("Text"),
		prop("Checked"),
		propColor("TextColor"),
		propColor("BackgroundColor"),
	),

	wui.NewGroupBox(): commonPropertiesPlus(
		prop("Text"),
		propColor("TextColor"),
	),

	wui.NewStatusBar(): commonPropertiesPlus(
		prop("Text"),
	),

	wui.NewTabControl(): commonPropertiesPlus(
		prop("Tabs"),
		prop("SelectedIndex"),
	),

	wui.NewDatePicker(): commonPropertiesPlus(
		prop("Mode"),
	),

	wui.NewSlider(): commonPropertiesPlus(
		prop("ArrowIncrement"),
		prop("MouseIncrement"),
		// The range comes before the cursor position, which is clamped to
		// the range when it is set.
		prop("Min"),
		prop("Max"),
		prop("MinMax", "Min", "Max"),
		prop("CursorPosition"),
		prop("Orientation"),
		prop("TickFrequency"),
		prop("TickPosition"),
		prop("TicksVisible"),
	),

	wui.NewPanel(): commonPropertiesPlus(
		prop("BorderStyle"),
		propColor("BackgroundColor"),
	),

	wui.NewPaintBox(): commonPropertiesPlus(),

	wui.NewEditLine(): commonPropertiesPlus(
		prop("Text"),
		prop("CharacterLimit"),
		prop("IsPassword"),
		prop("ReadOnly"),
		propColor("TextColor"),
		propColor("BackgroundColor"),
	),

	wui.NewIntUpDown(): commonPropertiesPlus(
		prop("Min"),
		prop("Max"),
		prop("MinMax", "Min", "Max"),
		prop("Value"),
	),

	wui.NewComboBox(): commonPropertiesPlus(
		prop("Editable"),
		prop("Items"),
		prop("SelectedIndex"),
		propColor("TextColor"),
		propColor("BackgroundColor"),
	),

	wui.NewProgressBar(): commonPropertiesPlus(
		prop("Vertical"),
		prop("MovesForever"),
		prop("State"),
		prop("Value"),
	),

	wui.NewFloatUpDown(): commonPropertiesPlus(
		prop("Min"),
		prop("Max"),
		prop("MinMax", "Min", "Max"),
		prop("Precision"),
		prop("Value"),
	),

	wui.NewTextEdit(): commonPropertiesPlus(
		prop("Text"),
		prop("WordWrap"),
		prop("CharacterLimit"),
		prop("WritesTabs"),
		prop("ReadOnly"),
		propColor("TextColor"),
		propColor("BackgroundColor"),
	),

	wui.NewListView(): commonPropertiesPlus(
		prop("View"),
		prop("MultiSelect"),
	),

	wui.NewRichEdit(): commonPropertiesPlus(
		prop("Text"),
		prop("ReadOnly"),
	),

	wui.NewLinkLabel(): commonPropertiesPlus(
		prop("Text"),
	),

	wui.NewMonthCalendar(): commonPropertiesPlus(),

	wui.NewHotKeyEdit(): commonPropertiesPlus(),

	wui.NewIPAddressEdit(): commonPropertiesPlus(),

	wui.NewImageView(): commonPropertiesPlus(
		prop("Mode"),
	),

	wui.NewScrollPanel(): commonPropertiesPlus(
		prop("BorderStyle"),
		propColor("BackgroundColor"),
	),

	wui.NewScrollBar(false): scrollBarProperties(),

	// The TreeView is not known to WML. The designer saves it as a Panel with
	// a note in the designer comment block, see wmlproject.go.
	wui.NewTreeView(): commonPropertiesPlus(),
}

func generateProperties(variable string, control interface{}) []string {
	for def, props := range properties {
		if reflect.TypeOf(control) == reflect.TypeOf(def) {
			return genProps(variable, control, def, props)
		}
	}
	panic("no properties found for type " + reflect.TypeOf(control).String())

}

func genProps(variable string, c, def interface{}, props []property) []string {
	var s []string
	control := reflect.ValueOf(c)
	wasSet := make(map[string]bool)
	for _, p := range props {
		if len(p.combines) > 0 {
			if !containsAll(wasSet, p.combines) {
				continue
			}
		}
		if _, ok := control.Type().MethodByName(p.name); !ok {
			panic(fmt.Sprintf("%v does not have method %v", control.Type(), p.name))
		}
		if p.requires != "" {
			if !control.MethodByName(p.requires).Call(nil)[0].Bool() {
				continue
			}
		}
		ours := control.MethodByName(p.name).Call(nil)
		defaults := reflect.ValueOf(def).MethodByName(p.name).Call(nil)
		if !equal(ours, defaults) {
			s = append(s, variable+".Set"+p.name+"("+toGo(ours)+")")
			wasSet[p.name] = true
			for _, previous := range p.combines {
				s = removeFirstThatContains(s, variable+".Set"+previous+"(")
			}
		}
	}
	return s
}

func containsAll(set map[string]bool, list []string) bool {
	for _, s := range list {
		if !set[s] {
			return false
		}
	}
	return true
}

func removeFirstThatContains(from []string, pattern string) []string {
	for i := range from {
		if strings.Contains(from[i], pattern) {
			return append(from[:i], from[i+1:]...)
		}
	}
	return from
}

func equal(a, b []reflect.Value) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !reflect.DeepEqual(a[i].Interface(), b[i].Interface()) {
			return false
		}
	}
	return true
}

func toGo(args []reflect.Value) string {
	var asGo []string
	for _, arg := range args {
		var s string
		switch arg.Kind() {
		case reflect.Slice:
			if list, ok := arg.Interface().([]string); ok {
				switch len(list) {
				case 0:
					s = "[]string{}"
				case 1:
					s = fmt.Sprintf("[]string{%q}", list[0])
				default:
					s = "[]string{\n"
					for i := range list {
						s += fmt.Sprintf("%q,\n", list[i])
					}
					s += "}"
				}
			} else {
				panic("slices of types other than string not handled")
			}
		case reflect.String:
			s = fmt.Sprintf("%q", arg.String())
		case reflect.Uint32:
			if arg.Type() == reflect.TypeOf(wui.Color(0)) {
				c := wui.Color(arg.Uint())
				s = fmt.Sprintf("wui.RGB(%d, %d, %d)", c.R(), c.G(), c.B())
			} else {
				s = fmt.Sprint(arg)
			}
		default:
			s = fmt.Sprint(arg)
		}
		asGo = append(asGo, s)
	}
	return strings.Join(asGo, ", ")
}

// hasDefaultValue reports whether the property has the same value on the
// control and on the default instance of its type.
func hasDefaultValue(control, def interface{}, name string) bool {
	m := reflect.ValueOf(control).MethodByName(name)
	d := reflect.ValueOf(def).MethodByName(name)
	if !m.IsValid() || !d.IsValid() {
		return false
	}
	return equal(m.Call(nil), d.Call(nil))
}

// isListedProperty returns true if the property is part of the property list
// of the control's type.
func isListedProperty(control interface{}, name string) bool {
	for def, list := range properties {
		if reflect.TypeOf(control) == reflect.TypeOf(def) {
			for _, p := range list {
				if p.name == name {
					return true
				}
			}
			return false
		}
	}
	return false
}
