package main

// Events of the controls.
//
// The events of every type that WML knows come from the WML schema, so the
// designer and WML never disagree about them. Types that only the designer
// knows (the TreeView) list their events here.
//
// The template of a handler is built from the setter of the event with
// reflection, so it always has the signature that the library expects.

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/2dprototype/wui/wml"
)

// customEvents are the events of the types that WML does not know.
var customEvents = map[string][]string{
	"TreeView": {
		"OnResize", "OnDropFiles",
		"OnMouseDown", "OnMouseUp", "OnMouseMove", "OnMouseWheel",
		"OnMouseEnter", "OnMouseLeave", "OnKeyDown", "OnKeyUp", "OnChar", "OnFocus", "OnBlur",
		"OnSelect", "OnDoubleClick", "OnExpand", "OnCollapse", "OnCheck", "OnLabelEdit",
	},
}

// eventsOf returns the names of the events of a control or window.
func eventsOf(c interface{}) []string {
	name := typeNameOf(c)
	if list, ok := customEvents[name]; ok {
		return list
	}
	if spec := wml.Lookup(name); spec != nil {
		return spec.Events
	}
	return nil
}

// eventTemplate returns the Go code of an empty handler for the event.
func eventTemplate(c interface{}, ev string) string {
	m, ok := reflect.TypeOf(c).MethodByName("Set" + ev)
	if !ok || m.Type.NumIn() != 2 || m.Type.In(1).Kind() != reflect.Func {
		return "func() {\n\t\n}"
	}
	ft := m.Type.In(1)

	var params []string
	used := make(map[string]bool)
	ints := 0
	for i := 0; i < ft.NumIn(); i++ {
		t := ft.In(i).String()
		name := paramName(ev, t, &ints)
		if used[name] {
			name = fmt.Sprintf("%s%d", name, i)
		}
		used[name] = true
		params = append(params, name+" "+t)
	}

	result := ""
	body := "\t\n"
	if ft.NumOut() == 1 {
		result = " " + ft.Out(0).String()
		if ft.Out(0).Kind() == reflect.Bool {
			body = "\treturn true\n"
		}
	}
	return "func(" + strings.Join(params, ", ") + ")" + result + " {\n" + body + "}"
}

// paramName chooses a readable name for a parameter of an event handler.
func paramName(ev, typ string, ints *int) string {
	switch typ {
	case "wui.MouseButton":
		return "button"
	case "[]string":
		return "files"
	case "time.Time":
		return "date"
	case "*wui.TreeNode":
		return "node"
	case "*wui.ListViewItem":
		return "item"
	case "wui.WindowState":
		return "state"
	case "rune":
		return "r"
	case "float64":
		if ev == "OnMouseWheel" {
			return "delta"
		}
		return "value"
	case "string":
		if ev == "OnLinkClick" {
			return "url"
		}
		return "text"
	case "bool":
		if ev == "OnActivate" {
			return "active"
		}
		return "on"
	case "int":
		switch ev {
		case "OnKeyDown", "OnKeyUp":
			return "key"
		case "OnDPIChanged":
			return "dpi"
		case "OnColumnClick":
			return "column"
		case "OnSelect", "OnActivate":
			return "index"
		case "OnChange", "OnValueChange":
			return "value"
		}
		*ints++
		if *ints == 1 {
			return "x"
		}
		if *ints == 2 {
			return "y"
		}
		return fmt.Sprintf("n%d", *ints)
	}
	return "arg"
}

// goImports returns the standard packages that the event code uses, so that
// handlers can use them without the user having to add imports.
func goImports(code string) []string {
	candidates := []string{"fmt", "time", "os", "strings", "strconv", "math", "sort", "errors", "log"}
	var out []string
	for _, pkg := range candidates {
		if usesPackage(code, pkg) {
			out = append(out, pkg)
		}
	}
	return out
}

// usesPackage reports whether the code contains pkg followed by a dot that is
// not the end of a longer name, like "fmt.Println" but not "myfmt.Println".
func usesPackage(code, pkg string) bool {
	start := 0
	for {
		i := strings.Index(code[start:], pkg+".")
		if i < 0 {
			return false
		}
		i += start
		if i == 0 {
			return true
		}
		prev := code[i-1]
		isName := prev == '_' || prev == '.' || (prev >= 'a' && prev <= 'z') ||
			(prev >= 'A' && prev <= 'Z') || (prev >= '0' && prev <= '9')
		if !isName {
			return true
		}
		start = i + len(pkg) + 1
		if start >= len(code) {
			return false
		}
	}
}

// reflectMethod looks for a method of a control by name.
func reflectMethod(c interface{}, name string) (reflect.Method, bool) {
	return reflect.TypeOf(c).MethodByName(name)
}
