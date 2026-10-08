package main

// Extra properties.
//
// Most properties are read from the controls with their getter, see
// properties.go. Some properties of wui cannot be read back: Button.SetKind
// has no Kind(), ListView.SetGridLines has no GridLines() and the columns and
// rows of a ListView cannot be listed or replaced. Other properties are
// only known to the designer, for example the tool tip text that is saved
// next to the WML file.
//
// The designer therefore keeps the values of these properties itself, in the
// extras map: control -> property name -> value. Values are stored in the
// syntax that WML uses for attributes ("true", "Split", "#FF0000", "0,100"),
// lists are stored as lines separated by "\n". A value that equals the
// default is not stored at all.
//
// The extras are
//   - written to the WML file as attributes (or <Item>/<Column> children)
//     when the WML schema knows the property, otherwise to the designer
//     comment block of the file (see wmlproject.go),
//   - written to the generated Go code with the setter of the same name,
//   - copied when a control is copied.

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/2dprototype/wui/wml"
)

type exKind int

const (
	exBool exKind = iota
	exString
	exInt
	exEnum
	exColor
	exInts // exactly two whole numbers, "a,b"
	exList // lines of text
)

type extraSpec struct {
	name  string
	label string
	kind  exKind
	def   string   // default value in WML syntax, "" if there is none
	enum  []string // exEnum: the WML names, the index is the numeric value
	min   int      // exInt
	max   int      // exInt
}

var extras = make(map[interface{}]map[string]string)

var textAlignNames = []string{"Left", "Center", "Right"}

var toolTipSpec = extraSpec{name: "ToolTip", label: "Tool Tip", kind: exString}

// extraTable lists the extra properties by type name. Every control except
// the Window also has a tool tip, see extrasOf. The order is the order in
// which the setters are written to the generated code.
var extraTable = map[string][]extraSpec{
	"Window": {
		{name: "MinSize", label: "Min Size", kind: exInts, def: "0,0"},
		{name: "MaxSize", label: "Max Size", kind: exInts, def: "0,0"},
	},
	"Button": {
		{name: "Kind", label: "Kind", kind: exEnum, def: "Normal", enum: []string{"Normal", "Split", "CommandLink"}},
		{name: "Note", label: "Note", kind: exString},
		{name: "Default", label: "Default Button", kind: exBool, def: "false"},
	},
	"CheckBox": {
		{name: "PushLike", label: "Push Like", kind: exBool, def: "false"},
	},
	"EditLine": {
		{name: "NumbersOnly", label: "Numbers Only", kind: exBool, def: "false"},
		{name: "TextAlign", label: "Text Align", kind: exEnum, def: "Left", enum: textAlignNames},
	},
	"TextEdit": {
		{name: "NumbersOnly", label: "Numbers Only", kind: exBool, def: "false"},
		{name: "TextAlign", label: "Text Align", kind: exEnum, def: "Left", enum: textAlignNames},
	},
	"ListView": {
		{name: "HeaderVisible", label: "Header Visible", kind: exBool, def: "true"},
		{name: "FullRowSelect", label: "Full Row Select", kind: exBool, def: "true"},
		{name: "GridLines", label: "Grid Lines", kind: exBool, def: "false"},
		{name: "CheckBoxes", label: "Check Boxes", kind: exBool, def: "false"},
		{name: "Editable", label: "Editable", kind: exBool, def: "false"},
		{name: "Sortable", label: "Sortable", kind: exBool, def: "false"},
		{name: "Columns", label: "Columns", kind: exList},
		{name: "Items", label: "Rows", kind: exList},
	},
	"RichEdit": {
		{name: "WordWrap", label: "Word Wrap", kind: exBool, def: "true"},
		{name: "AutoDetectLinks", label: "Detect Links", kind: exBool, def: "true"},
		{name: "WritesTabs", label: "Writes Tabs", kind: exBool, def: "false"},
		{name: "BackgroundColor", label: "Background", kind: exColor},
	},
	"MonthCalendar": {
		{name: "ShowWeekNumbers", label: "Week Numbers", kind: exBool, def: "false"},
	},
	"ImageView": {
		{name: "BackColor", label: "Back Color", kind: exColor},
		// The picture is loaded from this file. WML has no file properties,
		// so the path is only kept in the designer comment block.
		{name: "ImageFile", label: "Image File", kind: exString},
	},
	"ScrollBar": {
		{name: "Vertical", label: "Vertical", kind: exBool, def: "false"},
		{name: "Range", label: "Range", kind: exInts, def: "0,100"},
		{name: "Page", label: "Page", kind: exInt, def: "10", min: 0, max: 1 << 30},
	},
	"TreeView": {
		{name: "CheckBoxes", label: "Check Boxes", kind: exBool, def: "false"},
		{name: "Editable", label: "Editable", kind: exBool, def: "false"},
		{name: "Nodes", label: "Nodes", kind: exList},
	},
}

// typeNameOf returns the wui type name of a control, "Button" for *wui.Button.
func typeNameOf(c interface{}) string {
	return strings.TrimPrefix(reflect.TypeOf(c).String(), "*wui.")
}

// extrasOf returns the extra properties of a type.
func extrasOf(typeName string) []extraSpec {
	list := extraTable[typeName]
	if typeName == "Window" {
		return list
	}
	// The full slice expression makes append copy instead of changing the table.
	return append(list[:len(list):len(list)], toolTipSpec)
}

// findExtra returns the extra property with the given name of a control.
func findExtra(c interface{}, name string) (extraSpec, bool) {
	for _, spec := range extrasOf(typeNameOf(c)) {
		if spec.name == name {
			return spec, true
		}
	}
	return extraSpec{}, false
}

// anyExtraSpec returns the first extra property with the given name of any
// type. The property panel uses it to find labels and enum names.
func anyExtraSpec(name string) (extraSpec, bool) {
	for _, list := range extraTable {
		for _, spec := range list {
			if spec.name == name {
				return spec, true
			}
		}
	}
	if name == toolTipSpec.name {
		return toolTipSpec, true
	}
	return extraSpec{}, false
}

func getExtra(c interface{}, spec extraSpec) string {
	if m := extras[c]; m != nil {
		if v, ok := m[spec.name]; ok {
			return v
		}
	}
	return spec.def
}

func setExtra(c interface{}, spec extraSpec, value string) {
	m := extras[c]
	if value == spec.def {
		if m != nil {
			delete(m, spec.name)
		}
		return
	}
	if m == nil {
		m = make(map[string]string)
		extras[c] = m
	}
	m[spec.name] = value
}

// setExtraByName sets the value of an extra property if the control has it.
func setExtraByName(c interface{}, name, value string) {
	if spec, ok := findExtra(c, name); ok {
		setExtra(c, spec, value)
	}
}

func extraValue(c interface{}, name string) string {
	if spec, ok := findExtra(c, name); ok {
		return getExtra(c, spec)
	}
	return ""
}

func extraBool(c interface{}, name string) bool {
	return extraValue(c, name) == "true"
}

func extraLines(c interface{}, name string) []string {
	v := extraValue(c, name)
	if v == "" {
		return nil
	}
	return strings.Split(v, "\n")
}

// extraInts returns the numbers of an exInts or exInt property.
func extraInts(c interface{}, name string) []int {
	var out []int
	for _, part := range strings.Split(extraValue(c, name), ",") {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			n = 0
		}
		out = append(out, n)
	}
	return out
}

// extraColor returns the color of an exColor property and whether it is set.
func extraColor(c interface{}, name string) (r, g, b uint8, ok bool) {
	return parseHexColor(extraValue(c, name))
}

// parseHexColor reads "#RRGGBB".
func parseHexColor(s string) (r, g, b uint8, ok bool) {
	if len(s) != 7 || s[0] != '#' {
		return 0, 0, 0, false
	}
	n, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return uint8(n >> 16), uint8(n >> 8), uint8(n), true
}

// copyExtras gives dst the same extra properties as src.
func copyExtras(dst, src interface{}) {
	m := extras[src]
	if m == nil {
		return
	}
	n := make(map[string]string, len(m))
	for k, v := range m {
		n[k] = v
	}
	extras[dst] = n
}

// wmlProp returns the schema of a property of a WML type, or nil.
func wmlProp(typeName, name string) *wml.PropSpec {
	spec := wml.Lookup(typeName)
	if spec == nil {
		return nil
	}
	for i := range spec.Props {
		if spec.Props[i].Name == name {
			return &spec.Props[i]
		}
	}
	return nil
}

// kindMatches reports whether the WML property can hold the extra property.
func kindMatches(spec extraSpec, ps *wml.PropSpec) bool {
	switch spec.kind {
	case exBool:
		return ps.Kind == wml.KindBool
	case exString:
		return ps.Kind == wml.KindString
	case exInt:
		return ps.Kind == wml.KindInt
	case exEnum:
		return ps.Kind == wml.KindEnum
	case exColor:
		return ps.Kind == wml.KindColor
	case exInts:
		return ps.Kind == wml.KindInts
	case exList:
		return ps.Kind == wml.KindList
	}
	return false
}

// wmlPropFor returns the WML schema of an extra property if a WML element of
// the given type stores it. Otherwise the value goes to the designer comment
// block.
func wmlPropFor(wmlType string, spec extraSpec) *wml.PropSpec {
	ps := wmlProp(wmlType, spec.name)
	if ps != nil && kindMatches(spec, ps) {
		return ps
	}
	return nil
}

// ----- Go code ---------------------------------------------------------------

// extraGoLines returns the setter calls for the extra properties of a control
// that do not have their default value.
func extraGoLines(variable string, c interface{}) []string {
	var out []string
	for _, spec := range extrasOf(typeNameOf(c)) {
		v := getExtra(c, spec)
		if v == spec.def {
			continue
		}
		if spec.name == "Nodes" {
			out = append(out, treeGoLines(variable, strings.Split(v, "\n"))...)
			continue
		}
		if spec.name == "ImageFile" {
			out = append(out, variable+".SetImageFromFile("+strconv.Quote(v)+")")
			continue
		}
		args := extraGoArgs(spec, v)
		if args == "" && spec.kind != exString && spec.kind != exList {
			continue
		}
		out = append(out, variable+".Set"+spec.name+"("+args+")")
	}
	return out
}

// extraGoArgs writes a value as Go arguments.
func extraGoArgs(spec extraSpec, v string) string {
	switch spec.kind {
	case exBool:
		return v
	case exString:
		return strconv.Quote(v)
	case exInt:
		return v
	case exEnum:
		for i, name := range spec.enum {
			if name == v {
				return strconv.Itoa(i)
			}
		}
		return ""
	case exColor:
		if r, g, b, ok := parseHexColor(v); ok {
			return fmt.Sprintf("wui.RGB(%d, %d, %d)", r, g, b)
		}
		return ""
	case exInts:
		parts := strings.Split(v, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return strings.Join(parts, ", ")
	case exList:
		items := strings.Split(v, "\n")
		quoted := make([]string, len(items))
		for i, item := range items {
			quoted[i] = strconv.Quote(item)
		}
		return "[]string{" + strings.Join(quoted, ", ") + "}"
	}
	return ""
}

// ----- tree nodes ----------------------------------------------------------------

type treeItem struct {
	text        string
	depth       int
	hasChildren bool
}

// parseTreeLines reads the nodes of a TreeView. Every line is a node, the
// indentation of a line (spaces or tabs) tells whose child it is.
func parseTreeLines(lines []string) []treeItem {
	var items []treeItem
	var stack []int
	for _, raw := range lines {
		text := strings.TrimSpace(raw)
		if text == "" {
			continue
		}
		indent := 0
		for _, r := range raw {
			if r == ' ' {
				indent++
			} else if r == '\t' {
				indent += 4
			} else {
				break
			}
		}
		for len(stack) > 0 && stack[len(stack)-1] >= indent {
			stack = stack[:len(stack)-1]
		}
		items = append(items, treeItem{text: text, depth: len(stack)})
		stack = append(stack, indent)
	}
	for i := range items {
		if i+1 < len(items) && items[i+1].depth > items[i].depth {
			items[i].hasChildren = true
		}
	}
	return items
}

// treeGoLines writes the Add calls that create the nodes.
func treeGoLines(variable string, lines []string) []string {
	var out []string
	vars := make(map[int]string)
	for i, it := range parseTreeLines(lines) {
		parent := variable
		if it.depth > 0 {
			parent = vars[it.depth-1]
		}
		if it.hasChildren {
			v := fmt.Sprintf("%sNode%d", variable, i)
			out = append(out, fmt.Sprintf("%s := %s.Add(%q)", v, parent, it.text))
			vars[it.depth] = v
		} else {
			out = append(out, fmt.Sprintf("%s.Add(%q)", parent, it.text))
		}
	}
	return out
}
