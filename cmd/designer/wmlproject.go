package main

// Project files of the designer are WML files (see package wml). The file is
// a normal WML document, so it can be loaded by an application with
// wml.ParseFile and Document.Build.
//
// WML never contains code. The Go code of event handlers that is typed into
// the designer is therefore stored in one XML comment at the end of the file.
// The On... attributes of the elements name the handlers, the comment holds
// their code. Parsers skip comments, the designer reads the code back.

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/2dprototype/wui"
	"github.com/2dprototype/wui/wml"
)

// projectExt is the file extension of designer projects.
const projectExt = ".wml"

const (
	eventBlockStart = "<!-- wui-designer:events"
	propsBlockStart = "<!-- wui-designer:props"
)

// customTypes are the controls that WML does not know. They are saved as a
// Panel at the same place, and the real type is noted in the props block under
// the key "<name>:Type". Loading the file in the designer brings the real
// control back, an application that loads it with package wml sees a Panel.
var customTypes = map[string]func() wui.Control{
	"TreeView": func() wui.Control { return wui.NewTreeView() },
}

func isCustomControl(c interface{}) bool {
	_, ok := customTypes[typeNameOf(c)]
	return ok
}

// savedByBounds are the properties that are not written one by one: controls
// are saved with Bounds and the window with InnerSize.
var savedByBounds = map[string]bool{
	"X": true, "Y": true, "Width": true, "Height": true,
	"InnerX": true, "InnerY": true, "InnerWidth": true, "InnerHeight": true,
}

// saveProject writes the window with all its controls and event code as a WML
// file.
func saveProject(w *wui.Window, filePath string) error {
	doc, codes, props, err := buildDocument(w)
	if err != nil {
		return err
	}
	if err := doc.Validate(); err != nil {
		return fmt.Errorf("the project cannot be written as WML:\n%s", describeError(err))
	}
	data := appendBlock(doc.Marshal(), propsBlockStart,
		"Properties that WML cannot store. Lines starting with @ name a key, lines starting with | are its value.", props)
	data = appendEventBlock(data, codes)
	return ioutil.WriteFile(filePath, data, 0666)
}

// loadProject reads a WML project. The notice is not empty when something was
// left out that the user should know about.
func loadProject(filePath string) (window *wui.Window, notice string, err error) {
	raw, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, "", err
	}
	doc, err := wml.Parse(bytes.NewReader(raw), wml.WithFileName(filepath.Base(filePath)))
	if err != nil {
		return nil, "", fmt.Errorf("%s", describeError(err))
	}

	// The designer edits one window. Further windows would be lost when the
	// project is saved again, so say so.
	first := doc.Windows[0]
	if len(doc.Windows) > 1 {
		notice = fmt.Sprintf("The file contains %d windows. The designer edits the first one; "+
			"the others are dropped when you save.", len(doc.Windows))
	}
	single := wml.NewDocument(first)
	single.File = doc.File

	view, err := single.BuildWith(wml.BuildOptions{AllowMissingHandlers: true}, nil, nil)
	if err != nil {
		return nil, "", fmt.Errorf("%s", describeError(err))
	}
	window = view.Window()
	if window == nil {
		return nil, "", fmt.Errorf("the file contains no window")
	}

	// Only now, when everything worked, replace the state of the designer.
	names = make(map[interface{}]string)
	events = make(map[event]string)
	extras = make(map[interface{}]map[string]string)
	codes := parseEventBlock(raw)
	props := parseBlock(raw, propsBlockStart)
	registerNode(first, view, codes, props)
	upgradeCustom(window, codes, props)
	if names[window] == "" {
		names[window] = "window"
	}
	nameAll(window)
	if window.Font() == nil {
		window.SetFont(defaultFont())
	}
	return window, notice, nil
}

func describeError(err error) string {
	if list, ok := err.(wml.ErrorList); ok {
		return list.Details()
	}
	return err.Error()
}

// registerNode records the names, the extra properties and the event code of
// the built objects.
func registerNode(n *wml.Node, view *wml.View, codes, props map[string]string) {
	if name, ok := n.Attr("Name"); ok {
		if obj, found := view.Lookup(name); found {
			names[obj] = name
			if spec := wml.Lookup(n.Type); spec != nil {
				for _, ev := range spec.Events {
					handler, has := n.Attr(ev)
					if !has {
						continue
					}
					if code, ok := codes[handler]; ok && strings.TrimSpace(code) != "" {
						events[event{control: obj, name: ev}] = code
					}
				}
			}
			loadExtras(obj, n, name, props)
		}
	}
	for _, child := range n.Children {
		registerNode(child, view, codes, props)
	}
}

// loadExtras reads the extra properties of one element.
func loadExtras(obj interface{}, n *wml.Node, name string, props map[string]string) {
	for _, spec := range extrasOf(typeNameOf(obj)) {
		if ps := wmlPropFor(n.Type, spec); ps != nil {
			if spec.kind == exList {
				var items []string
				for _, c := range n.Children {
					if c.Type == ps.Elem {
						items = append(items, strings.Replace(c.Text, "\n", " ", -1))
					}
				}
				if len(items) > 0 {
					setExtra(obj, spec, strings.Join(items, "\n"))
				}
			} else if v, ok := n.Attr(spec.name); ok {
				setExtra(obj, spec, v)
			}
			continue
		}
		if v, ok := props[name+"."+spec.name]; ok {
			setExtra(obj, spec, v)
		}
	}
}

// upgradeCustom replaces the Panels that stand for controls WML does not
// know by the real controls.
func upgradeCustom(c wui.Container, codes, props map[string]string) {
	for _, child := range append([]wui.Control(nil), c.Children()...) {
		if p, ok := child.(*wui.Panel); ok {
			name := names[p]
			if newCtl := customNew(props[name+":Type"]); name != "" && newCtl != nil {
				x, y, w, h := p.Bounds()
				newCtl.SetBounds(x, y, w, h)
				ha, va := p.Anchors()
				newCtl.SetHorizontalAnchor(ha)
				newCtl.SetVerticalAnchor(va)
				if dst, ok := newCtl.(enabler); ok {
					dst.SetEnabled(p.Enabled())
				}
				if dst, ok := newCtl.(visibler); ok {
					dst.SetVisible(p.Visible())
				}
				if dst, ok := newCtl.(fonter); ok && p.Font() != nil {
					dst.SetFont(p.Font())
				}
				delete(names, p)
				names[newCtl] = name
				copyExtras(newCtl, p)
				delete(extras, p)
				// Extras that were stored for a Panel are replaced by the
				// ones of the real type.
				for _, spec := range extrasOf(typeNameOf(newCtl)) {
					if v, ok := props[name+"."+spec.name]; ok {
						setExtra(newCtl, spec, v)
					}
				}
				for _, ev := range eventsOf(newCtl) {
					if code, ok := codes[name+"_"+ev]; ok && strings.TrimSpace(code) != "" {
						events[event{control: newCtl, name: ev}] = code
					}
				}
				replaceChild(c, p, newCtl)
				continue
			}
		}
		if con, ok := child.(wui.Container); ok {
			upgradeCustom(con, codes, props)
		}
	}
}

func customNew(typeName string) wui.Control {
	if f, ok := customTypes[typeName]; ok {
		return f()
	}
	return nil
}

// replaceChild puts repl at the place of old in the children of parent.
func replaceChild(parent wui.Container, old, repl wui.Control) {
	children := append([]wui.Control(nil), parent.Children()...)
	idx := -1
	for i, ch := range children {
		if ch == old {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	for _, ch := range children[idx:] {
		parent.Remove(ch)
	}
	parent.Add(repl)
	for _, ch := range children[idx+1:] {
		parent.Add(ch)
	}
}

// nameAll gives every control without a name a unique default name.
func nameAll(c wui.Container) {
	for _, child := range c.Children() {
		if names[child] == "" {
			names[child] = defaultName(child)
		}
		if sub, ok := child.(wui.Container); ok {
			nameAll(sub)
		}
	}
}

// ----- writing ---------------------------------------------------------------

type docBuilder struct {
	codes    map[string]string // handler name -> Go code
	props    map[string]string // properties for the designer comment block
	seen     map[string]bool
	problems []string
}

func buildDocument(w *wui.Window) (*wml.Document, map[string]string, map[string]string, error) {
	if names[w] == "" {
		names[w] = "window"
	}
	b := &docBuilder{codes: make(map[string]string), props: make(map[string]string), seen: make(map[string]bool)}
	root := b.node(w, "Window")
	if len(b.problems) > 0 {
		return nil, nil, nil, fmt.Errorf("the project cannot be saved:\n%s", strings.Join(b.problems, "\n"))
	}
	return wml.NewDocument(root), b.codes, b.props, nil
}

func (b *docBuilder) problem(format string, args ...interface{}) {
	if len(b.problems) < 20 {
		b.problems = append(b.problems, fmt.Sprintf(format, args...))
	}
}

func (b *docBuilder) node(c interface{}, typeName string) *wml.Node {
	wmlType := typeName
	custom := isCustomControl(c)
	if custom {
		wmlType = "Panel"
	}
	n := &wml.Node{Type: wmlType}
	spec := wml.Lookup(wmlType)
	if spec == nil {
		b.problem("%s is not supported by WML", typeName)
		return n
	}

	name := names[c]
	if name == "" {
		name = defaultName(c)
		names[c] = name
	}
	if !validName(name) {
		b.problem("The name %q is not valid. Use letters, digits and underscores, "+
			"start with a letter or underscore, at most 64 characters.", name)
	} else if b.seen[name] {
		b.problem("The name %q is used more than once.", name)
	}
	b.seen[name] = true
	n.SetAttr("Name", name)
	if custom {
		b.props[name+":Type"] = typeName
	}

	// Size and position first. A window is saved by its inner size only: the
	// position on the screen does not belong into a layout.
	if win, ok := c.(*wui.Window); ok {
		iw, ih := win.InnerSize()
		n.SetAttr("InnerSize", fmt.Sprintf("%d,%d", iw, ih))
	} else if ctl, ok := c.(wui.Control); ok {
		x, y, cw, ch := ctl.Bounds()
		n.SetAttr("Bounds", fmt.Sprintf("%d,%d,%d,%d", x, y, cw, ch))
	}

	var def interface{}
	var list []property
	for d, props := range properties {
		if reflect.TypeOf(c) == reflect.TypeOf(d) {
			def, list = d, props
			break
		}
	}

	var lists []*wml.Node
	for _, p := range list {
		if len(p.combines) > 0 || savedByBounds[p.name] {
			continue
		}
		var ps *wml.PropSpec
		for i := range spec.Props {
			if spec.Props[i].Name == p.name {
				ps = &spec.Props[i]
				break
			}
		}
		if ps == nil {
			continue
		}
		if p.requires != "" {
			if on, ok := getValue(c, p.requires); !ok || on.Kind() != reflect.Bool || !on.Bool() {
				continue
			}
		}
		if hasDefaultValue(c, def, p.name) {
			continue
		}
		v, ok := getValue(c, p.name)
		if !ok {
			continue
		}
		if ps.Kind == wml.KindList {
			if items, ok := v.Interface().([]string); ok {
				for _, item := range items {
					lists = append(lists, &wml.Node{Type: ps.Elem, Text: item})
				}
			}
			continue
		}
		if s, ok := encodeValue(ps, v); ok {
			n.SetAttr(p.name, s)
		}
	}

	// Events: the attribute names the handler, the code goes into the comment.
	evList := spec.Events
	if custom {
		evList = eventsOf(c)
	}
	for _, ev := range evList {
		code := events[event{control: c, name: ev}]
		if isEmptyHandler(code) {
			continue
		}
		handler := name + "_" + ev
		if !custom {
			n.SetAttr(ev, handler)
		}
		b.codes[handler] = code
	}

	// Extra properties: attributes where WML has the property, otherwise the
	// designer comment block.
	for _, ex := range extrasOf(typeName) {
		v := getExtra(c, ex)
		if v == ex.def {
			continue
		}
		ps := wmlPropFor(wmlType, ex)
		switch {
		case ps == nil:
			b.props[name+"."+ex.name] = v
		case ex.kind == exList:
			for _, item := range strings.Split(v, "\n") {
				lists = append(lists, &wml.Node{Type: ps.Elem, Text: item})
			}
		default:
			n.SetAttr(ex.name, v)
		}
	}

	if f, ok := c.(fonter); ok {
		if font := f.Font(); font != nil {
			n.Add(fontNode(font))
		}
	}
	for _, item := range lists {
		n.Add(item)
	}
	if con, ok := c.(wui.Container); ok {
		for _, child := range con.Children() {
			n.Add(b.node(child, reflect.TypeOf(child).Elem().Name()))
		}
	}
	return n
}

func fontNode(font *wui.Font) *wml.Node {
	f := &wml.Node{Type: "Font"}
	if font.Desc.Name != "" {
		f.SetAttr("Name", font.Desc.Name)
	}
	f.SetAttr("Height", strconv.Itoa(font.Desc.Height))
	if font.Desc.Bold {
		f.SetAttr("Bold", "true")
	}
	if font.Desc.Italic {
		f.SetAttr("Italic", "true")
	}
	if font.Desc.Underlined {
		f.SetAttr("Underlined", "true")
	}
	if font.Desc.StrikedOut {
		f.SetAttr("StrikedOut", "true")
	}
	return f
}

// getValue calls the getter with the given name.
func getValue(c interface{}, name string) (reflect.Value, bool) {
	m := reflect.ValueOf(c).MethodByName(name)
	if !m.IsValid() || m.Type().NumIn() != 0 || m.Type().NumOut() < 1 {
		return reflect.Value{}, false
	}
	return m.Call(nil)[0], true
}

func asInt64(v reflect.Value) (int64, bool) {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(v.Uint()), true
	}
	return 0, false
}

// encodeValue writes a property value in the syntax of its WML kind.
func encodeValue(ps *wml.PropSpec, v reflect.Value) (string, bool) {
	switch ps.Kind {
	case wml.KindString:
		if v.Kind() == reflect.String {
			return v.String(), true
		}
	case wml.KindBool:
		if v.Kind() == reflect.Bool {
			return strconv.FormatBool(v.Bool()), true
		}
	case wml.KindInt:
		if n, ok := asInt64(v); ok {
			return strconv.FormatInt(n, 10), true
		}
	case wml.KindFloat:
		switch v.Kind() {
		case reflect.Float32:
			return strconv.FormatFloat(v.Float(), 'g', -1, 32), true
		case reflect.Float64:
			return strconv.FormatFloat(v.Float(), 'g', -1, 64), true
		}
	case wml.KindColor:
		if v.Type() == reflect.TypeOf(wui.Color(0)) {
			c := wui.Color(v.Uint())
			return fmt.Sprintf("#%02X%02X%02X", c.R(), c.G(), c.B()), true
		}
	case wml.KindEnum:
		if n, ok := asInt64(v); ok && n >= 0 && int(n) < len(ps.Enum) {
			return ps.Enum[n], true
		}
	}
	return "", false
}

// validName reports whether s can be used as the Name of an element.
func validName(s string) bool {
	if len(s) == 0 || len(s) > 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		letter := c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		digit := c >= '0' && c <= '9'
		if !letter && !(digit && i > 0) {
			return false
		}
	}
	return true
}

// isEmptyHandler is true for code that has no content, for example the
// template "func() {\n\t\n}" of an event that was opened but not filled.
func isEmptyHandler(code string) bool {
	compact := strings.Join(strings.Fields(code), "")
	return compact == "" || strings.HasSuffix(compact, "{}")
}

// ----- event code in the comment -----------------------------------------------

// A comment must not contain two hyphens in a row, so '-' and '%' are written
// as %2D and %25.
func escapeComment(s string) string {
	s = strings.Replace(s, "%", "%25", -1)
	return strings.Replace(s, "-", "%2D", -1)
}

func unescapeComment(s string) string {
	s = strings.Replace(s, "%2D", "-", -1)
	return strings.Replace(s, "%25", "%", -1)
}

// appendEventBlock inserts the event code before the closing </wml>.
//
//	<!-- wui-designer:events
//	@ button1_OnClick
//	| func() {
//	| 	println("hello")
//	| }
//	-->
func appendEventBlock(data []byte, codes map[string]string) []byte {
	return appendBlock(data, eventBlockStart,
		"Go code of the event handlers. Lines starting with @ name a handler, lines starting with | are its code.", codes)
}

// parseEventBlock reads the event code written by appendEventBlock.
func parseEventBlock(raw []byte) map[string]string {
	return parseBlock(raw, eventBlockStart)
}

// appendBlock inserts a comment block with the given values before the
// closing </wml>.
func appendBlock(data []byte, start, help string, values map[string]string) []byte {
	if len(values) == 0 {
		return data
	}
	end := bytes.LastIndex(data, []byte("</wml>"))
	if end < 0 {
		return data
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b bytes.Buffer
	b.WriteString("  " + start + "\n")
	b.WriteString("  " + help + "\n")
	for _, k := range keys {
		b.WriteString("@ " + k + "\n")
		for _, line := range strings.Split(strings.Replace(values[k], "\r", "", -1), "\n") {
			b.WriteString("| " + escapeComment(line) + "\n")
		}
	}
	b.WriteString("  -->\n")

	out := make([]byte, 0, len(data)+b.Len())
	out = append(out, data[:end]...)
	out = append(out, b.Bytes()...)
	out = append(out, data[end:]...)
	return out
}

// parseBlock reads the values of the comment block that starts with start.
func parseBlock(raw []byte, start string) map[string]string {
	codes := make(map[string]string)
	s := string(raw)
	i := strings.Index(s, start)
	if i < 0 {
		return codes
	}
	s = s[i+len(start):]
	j := strings.Index(s, "-->")
	if j < 0 {
		return codes
	}

	current := ""
	var lines []string
	flush := func() {
		if current != "" {
			codes[current] = strings.Join(lines, "\n")
		}
		current = ""
		lines = nil
	}
	for _, line := range strings.Split(s[:j], "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "@ "):
			flush()
			current = strings.TrimSpace(line[2:])
		case strings.HasPrefix(line, "| ") && current != "":
			lines = append(lines, unescapeComment(line[2:]))
		case line == "|" && current != "":
			lines = append(lines, "")
		}
	}
	flush()
	return codes
}
