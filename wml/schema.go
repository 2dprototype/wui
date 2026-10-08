package wml

import (
	"fmt"
	"sort"
	"strings"
)

// Kind is the value kind of a property.
type Kind int

// Property kinds. How each one is written in XML:
//
//	KindString  any text
//	KindBool    true | false
//	KindInt     a whole number, e.g. 25
//	KindFloat   a decimal number, e.g. 0.5
//	KindColor   #RRGGBB or #RGB
//	KindEnum    one of a fixed list of names, case-insensitive
//	KindInts    comma separated whole numbers, e.g. Bounds="10,10,100,25"
//	KindList    not an attribute: repeated child elements, e.g. <Item>
//	KindFloats  comma separated decimal numbers, e.g. MinMax="0,1.5"
const (
	KindString Kind = iota
	KindBool
	KindInt
	KindFloat
	KindColor
	KindEnum
	KindInts
	KindList
	KindFloats
)

func (k Kind) String() string {
	switch k {
	case KindString:
		return "string"
	case KindBool:
		return "bool"
	case KindInt:
		return "int"
	case KindFloat:
		return "float"
	case KindColor:
		return "color"
	case KindEnum:
		return "enum"
	case KindInts:
		return "ints"
	case KindList:
		return "list"
	case KindFloats:
		return "floats"
	}
	return "unknown"
}

// maxStringLen is the longest string value (in bytes) a document may contain.
const maxStringLen = 1 << 16

// Coordinate and size limits. Win32 child windows use 16 bit coordinates in
// many places, so anything outside this range is a mistake.
const (
	coordMin = -32768
	coordMax = 32767
	sizeMax  = 32767
	int32Min = -1 << 31
	int32Max = 1<<31 - 1
)

// PropSpec describes one property of a control type.
type PropSpec struct {
	Name       string
	Kind       Kind
	Setter     string   // method name on the wui type, default "Set"+Name
	Alt        string   // method name to try if Setter does not exist
	Enum       []string // KindEnum: the names, the index is the numeric value
	N          int      // KindInts: how many numbers
	Min, Max   int64    // KindInt and KindInts: inclusive range
	FMin, FMax float64  // KindFloat: inclusive range
	NonNegTail int      // KindInts: the last NonNegTail numbers must be >= 0
	Elem       string   // KindList: the child element name
	Covers     []string // properties that this one sets as well, they conflict
}

func (p PropSpec) setterName() string {
	if p.Setter != "" {
		return p.Setter
	}
	return "Set" + p.Name
}

func (p PropSpec) withAlt(method string) PropSpec {
	p.Alt = method
	return p
}

func (p PropSpec) withSetter(method string) PropSpec {
	p.Setter = method
	return p
}

// Describe returns a one-line description such as "Width  int  0..32767".
func (p PropSpec) Describe() string {
	switch p.Kind {
	case KindInt:
		return fmt.Sprintf("%s  int  %d..%d", p.Name, p.Min, p.Max)
	case KindFloat:
		return fmt.Sprintf("%s  float  %g..%g", p.Name, p.FMin, p.FMax)
	case KindEnum:
		return fmt.Sprintf("%s  enum  %s", p.Name, strings.Join(p.Enum, " | "))
	case KindInts:
		return fmt.Sprintf("%s  %d ints  %d..%d", p.Name, p.N, p.Min, p.Max)
	case KindList:
		return fmt.Sprintf("%s  list of <%s> elements", p.Name, p.Elem)
	case KindFloats:
		return fmt.Sprintf("%s  %d floats  %g..%g", p.Name, p.N, p.FMin, p.FMax)
	}
	return fmt.Sprintf("%s  %s", p.Name, p.Kind)
}

// TypeSpec describes one element type.
type TypeSpec struct {
	Name      string
	Container bool       // may contain other controls (Window and Panel)
	Props     []PropSpec // in the order in which they are applied
	Events    []string
}

func (t *TypeSpec) prop(name string) *PropSpec {
	for i := range t.Props {
		if t.Props[i].Name == name {
			return &t.Props[i]
		}
	}
	return nil
}

func (t *TypeSpec) hasEvent(name string) bool {
	for _, e := range t.Events {
		if e == name {
			return true
		}
	}
	return false
}

// listProp returns the list property whose child element is called elem.
func (t *TypeSpec) listProp(elem string) *PropSpec {
	for i := range t.Props {
		if t.Props[i].Kind == KindList && t.Props[i].Elem == elem {
			return &t.Props[i]
		}
	}
	return nil
}

// Lookup returns the schema of an element type, or nil.
func Lookup(typeName string) *TypeSpec {
	return types[typeName]
}

// TypeNames returns all element type names in alphabetical order.
func TypeNames() []string {
	names := make([]string, 0, len(types))
	for name := range types {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Names of the special child elements.
const (
	fontElement = "Font"
	itemElement = "Item"
	tabElement  = "Tab"
	columnElement = "Column"
)

// fontProps are the attributes of a <Font> element. They mirror wui.FontDesc.
var fontProps = []PropSpec{
	pStr("Name"),
	pInt("Height", -400, 400),
	pBool("Bold"),
	pBool("Italic"),
	pBool("Underlined"),
	pBool("StrikedOut"),
}

func fontProp(name string) *PropSpec {
	for i := range fontProps {
		if fontProps[i].Name == name {
			return &fontProps[i]
		}
	}
	return nil
}

// ----- property constructors ------------------------------------------------

func pStr(name string) PropSpec  { return PropSpec{Name: name, Kind: KindString} }
func pBool(name string) PropSpec { return PropSpec{Name: name, Kind: KindBool} }

func pColor(name string) PropSpec { return PropSpec{Name: name, Kind: KindColor} }

func pInt(name string, min, max int64) PropSpec {
	return PropSpec{Name: name, Kind: KindInt, Min: min, Max: max}
}

func pFloat(name string, min, max float64) PropSpec {
	return PropSpec{Name: name, Kind: KindFloat, FMin: min, FMax: max}
}

func pEnum(name string, names ...string) PropSpec {
	return PropSpec{Name: name, Kind: KindEnum, Enum: names}
}

func pInts(name string, n int, min, max int64, nonNegTail int, covers ...string) PropSpec {
	return PropSpec{
		Name: name, Kind: KindInts, N: n, Min: min, Max: max,
		NonNegTail: nonNegTail, Covers: covers,
	}
}

func pList(name, elem string) PropSpec {
	return PropSpec{Name: name, Kind: KindList, Elem: elem}
}

// ----- enums (the index must match the constant's value in package wui) ----

var anchorNames = []string{"Min", "Max", "Center", "MinAndMax", "MinAndCenter", "MaxAndCenter"}

// ----- the schema ----------------------------------------------------------

// commonProps are the properties of every control. The order is the order in
// which the designer generates them.
func commonProps(plus ...PropSpec) []PropSpec {
	props := []PropSpec{
		pBool("Enabled"),
		pBool("Visible"),
		pEnum("HorizontalAnchor", anchorNames...),
		pEnum("VerticalAnchor", anchorNames...),
		pInt("X", coordMin, coordMax),
		pInt("Y", coordMin, coordMax),
		pInts("Position", 2, coordMin, coordMax, 0, "X", "Y").withAlt("SetPos"),
		pInt("Width", 0, sizeMax),
		pInt("Height", 0, sizeMax),
		pInts("Size", 2, 0, sizeMax, 0, "Width", "Height"),
		pInts("Bounds", 4, coordMin, coordMax, 2, "X", "Y", "Position", "Width", "Height", "Size"),
		pBool("TabStop"),
	}
	return append(props, plus...)
}

// controlEvents are the events that every control has.
func controlEvents(plus ...string) []string {
	all := []string{
		"OnResize", "OnDropFiles",
		"OnMouseDown", "OnMouseUp", "OnDoubleClick", "OnMouseMove", "OnMouseWheel",
		"OnMouseEnter", "OnMouseLeave", "OnKeyDown", "OnKeyUp", "OnChar", "OnFocus", "OnBlur",
	}
	seen := make(map[string]bool, len(all))
	for _, e := range all {
		seen[e] = true
	}
	for _, e := range plus {
		if !seen[e] {
			seen[e] = true
			all = append(all, e)
		}
	}
	return all
}

func textProp() PropSpec { return pStr("Text").withSetter("SetText") }

func colors() []PropSpec {
	return []PropSpec{pColor("TextColor"), pColor("BackgroundColor")}
}

var types = buildTypes()

func buildTypes() map[string]*TypeSpec {
	all := []*TypeSpec{
		{
			Name:      "Window",
			Container: true,
			Props: []PropSpec{
				pInt("InnerX", coordMin, coordMax),
				pInt("InnerY", coordMin, coordMax),
				pInts("InnerPosition", 2, coordMin, coordMax, 0, "InnerX", "InnerY"),
				pInt("InnerWidth", 0, sizeMax),
				pInt("InnerHeight", 0, sizeMax),
				pInts("InnerSize", 2, 0, sizeMax, 0, "InnerWidth", "InnerHeight"),
				pInts("InnerBounds", 4, coordMin, coordMax, 2,
					"InnerX", "InnerY", "InnerPosition", "InnerWidth", "InnerHeight", "InnerSize"),
				pStr("Title"),
				pInt("Alpha", 0, 255),
				pBool("HasMinButton"),
				pBool("HasMaxButton"),
				pBool("HasCloseButton"),
				pBool("HasBorder"),
				pBool("Resizable"),
				pEnum("State", "Normal", "Maximized", "Minimized"),
				pColor("BackgroundColor"),
				pBool("TopMost"),
				pBool("ShowInTaskbar"),
				pBool("DragByBackground"),
				pInt("CornerRadius", 0, 1000),
				pBool("AcceptFiles"),
				// The color comes before the switch: setting a color turns the
				// transparency on, SetTransparent(false) turns it off again.
				pColor("TransparentColor"),
				pBool("Transparent"),
				pBool("TrayEnabled"),
				pStr("TrayToolTip"),
				pBool("MinimizeToTray"),
				pBool("CloseToTray"),
				pBool("CenterOnShow"),
				pInts("MinSize", 2, 0, sizeMax, 0),
				pInts("MaxSize", 2, 0, sizeMax, 0),
			},
			Events: []string{
				"OnShow", "OnClose", "OnCanClose", "OnResize", "OnDropFiles",
				"OnMouseMove", "OnMouseWheel", "OnMouseDown", "OnMouseUp",
				"OnKeyDown", "OnKeyUp", "OnChar",
				"OnMove", "OnActivate", "OnStateChange", "OnDPIChanged",
			},
		},
		{
			Name: "Button",
			Props: commonProps(
				textProp(),
				pEnum("Kind", "Normal", "Split", "CommandLink"),
				pStr("Note"),
				pBool("Default"),
			),
			Events: controlEvents("OnClick", "OnTabFocus"),
		},
		{
			Name: "Label",
			Props: commonProps(append([]PropSpec{
				textProp(),
				pEnum("Alignment", "Left", "Center", "Right"),
			}, colors()...)...),
			Events: controlEvents(),
		},
		{
			Name: "CheckBox",
			Props: commonProps(append([]PropSpec{
				textProp(),
				pBool("Checked"),
				pBool("ThreeState"),
				pBool("PushLike"),
			}, colors()...)...),
			Events: controlEvents("OnChange", "OnTabFocus"),
		},
		{
			Name: "RadioButton",
			Props: commonProps(append([]PropSpec{
				textProp(),
				pBool("Checked"),
			}, colors()...)...),
			Events: controlEvents("OnCheck", "OnTabFocus"),
		},
		{
			Name:   "GroupBox",
			Props:  commonProps(textProp(), pColor("TextColor")),
			Events: controlEvents(),
		},
		{
			Name:   "StatusBar",
			Props:  commonProps(textProp()),
			Events: controlEvents(),
		},
		{
			Name: "TabControl",
			Props: commonProps(
				pList("Tabs", tabElement),
				pInt("SelectedIndex", -1, 65535),
			),
			Events: controlEvents("OnChange"),
		},
		{
			Name: "DatePicker",
			Props: commonProps(
				pEnum("Mode", "ShortDate", "LongDate", "Time"),
			),
			Events: controlEvents("OnChange"),
		},
		{
			Name: "Slider",
			Props: commonProps(
				pInt("ArrowIncrement", 0, int32Max),
				pInt("MouseIncrement", 0, int32Max),
				// The range comes before the cursor position, which is clamped
				// to the range when it is set.
				pInt("Min", int32Min, int32Max),
				pInt("Max", int32Min, int32Max),
				pInts("MinMax", 2, int32Min, int32Max, 0, "Min", "Max"),
				pInt("CursorPosition", int32Min, int32Max),
				pEnum("Orientation", "Horizontal", "Vertical"),
				pInt("TickFrequency", 0, int32Max),
				pEnum("TickPosition", "BottomOrRight", "TopOrLeft", "BothSides"),
				pBool("TicksVisible"),
			),
			Events: controlEvents("OnChange", "OnTabFocus"),
		},
		{
			Name:      "Panel",
			Container: true,
			Props: commonProps(
				pEnum("BorderStyle", "None", "SingleLine", "Sunken", "SunkenThick", "Raised"),
				pColor("BackgroundColor"),
			),
			Events: controlEvents(),
		},
		{
			Name:  "PaintBox",
			Props: commonProps(),
			Events: controlEvents(
				"OnPaint", "OnMouseDown", "OnMouseUp", "OnDoubleClick", "OnMouseMove"),
		},
		{
			Name: "EditLine",
			Props: commonProps(append([]PropSpec{
				textProp(),
				pInt("CharacterLimit", 0, int32Max),
				pBool("IsPassword"),
				pBool("ReadOnly"),
				pBool("NumbersOnly"),
				pEnum("TextAlign", "Left", "Center", "Right"),
			}, colors()...)...),
			Events: controlEvents("OnTextChange", "OnTabFocus"),
		},
		{
			Name: "TextEdit",
			Props: commonProps(append([]PropSpec{
				textProp(),
				pBool("WordWrap"),
				pInt("CharacterLimit", 0, int32Max),
				pBool("WritesTabs"),
				pBool("ReadOnly"),
				pBool("NumbersOnly"),
				pEnum("TextAlign", "Left", "Center", "Right"),
			}, colors()...)...),
			Events: controlEvents("OnTextChange", "OnTabFocus"),
		},
		{
			Name: "IntUpDown",
			Props: commonProps(
				// The range comes before the value, see Slider.
				pInt("Min", int32Min, int32Max),
				pInt("Max", int32Min, int32Max),
				pInts("MinMax", 2, int32Min, int32Max, 0, "Min", "Max"),
				pInt("Value", int32Min, int32Max),
			),
			Events: controlEvents("OnValueChange", "OnTabFocus"),
		},
		{
			Name: "FloatUpDown",
			Props: commonProps(
				pFloat("Min", -1e12, 1e12),
				pFloat("Max", -1e12, 1e12),
				pFloatPair("MinMax", -1e12, 1e12, "Min", "Max"),
				pInt("Precision", 0, 15),
				pFloat("Value", -1e12, 1e12),
			),
			Events: controlEvents("OnValueChange", "OnTabFocus"),
		},
		{
			Name: "ComboBox",
			Props: commonProps(append([]PropSpec{
				pBool("Editable"),
				pList("Items", itemElement),
				pInt("SelectedIndex", -1, 65535),
			}, colors()...)...),
			Events: controlEvents("OnChange", "OnTextChange", "OnTabFocus"),
		},
		{
			Name: "ProgressBar",
			Props: commonProps(
				pBool("Vertical"),
				pBool("MovesForever"),
				pEnum("State", "Normal", "Error", "Paused"),
				pFloat("Value", 0, 1),
			),
			Events: controlEvents(),
		},
		{
			Name: "ListView",
			Props: commonProps(
				pEnum("View", "Details", "List", "Icons", "SmallIcons", "Tiles"),
				pBool("MultiSelect"),
				pBool("Editable"),
				pBool("HeaderVisible"),
				pBool("CheckBoxes"),
				pBool("FullRowSelect"),
				pBool("GridLines"),
				pBool("Sortable"),
				pList("Columns", columnElement),
				pList("Items", itemElement),
			),
			Events: controlEvents("OnSelect", "OnActivate", "OnColumnClick", "OnTabFocus"),
		},
		{
			Name: "RichEdit",
			Props: commonProps(
				textProp(),
				pBool("WordWrap"),
				pBool("ReadOnly"),
				pBool("AutoDetectLinks"),
				pBool("WritesTabs"),
				pColor("BackgroundColor"),
			),
			Events: controlEvents("OnTextChange", "OnSelectionChange", "OnLinkClick", "OnTabFocus"),
		},
		{
			Name:   "LinkLabel",
			Props:  commonProps(textProp()),
			Events: controlEvents("OnClick", "OnTabFocus"),
		},
		{
			Name:   "MonthCalendar",
			Props:  commonProps(pBool("ShowWeekNumbers")),
			Events: controlEvents("OnChange", "OnTabFocus"),
		},
		{
			Name:   "HotKeyEdit",
			Props:  commonProps(),
			Events: controlEvents("OnChange", "OnTabFocus"),
		},
		{
			Name:   "IPAddressEdit",
			Props:  commonProps(),
			Events: controlEvents("OnChange", "OnTabFocus"),
		},
		{
			Name: "ImageView",
			Props: commonProps(
				pEnum("Mode", "Normal", "Center", "Stretch", "Fit", "Fill"),
				pColor("BackColor"),
			),
			Events: controlEvents(),
		},
		{
			Name:      "ScrollPanel",
			Container: true,
			Props: commonProps(
				pEnum("BorderStyle", "None", "SingleLine", "Sunken", "SunkenThick", "Raised"),
				pColor("BackgroundColor"),
			),
			Events: controlEvents(),
		},
		{
			Name: "ScrollBar",
			Props: commonProps(
				pBool("Vertical"),
				pInts("Range", 2, int32Min, int32Max, 0),
				pInt("Page", 0, int32Max),
				pInt("Position", int32Min, int32Max),
			),
			Events: controlEvents("OnChange"),
		},
	}

	m := make(map[string]*TypeSpec, len(all))
	for _, t := range all {
		m[t.Name] = t
	}
	return m
}

// pFloatPair is MinMax for float properties: two floats set by one call.
func pFloatPair(name string, min, max float64, covers ...string) PropSpec {
	return PropSpec{
		Name: name, Kind: KindFloats, N: 2, FMin: min, FMax: max, Covers: covers,
	}
}
