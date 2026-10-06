package wml

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/2dprototype/wui"
)

// Handlers maps the handler names used in OnXxx attributes to Go functions.
// Every function must have exactly the signature of the event's setter, for
// example func() for Button.OnClick or func(bool) for CheckBox.OnChange.
type Handlers map[string]interface{}

// BuildOptions change how a Document is built.
type BuildOptions struct {
	// AllowMissingHandlers skips events whose handler is not in Handlers
	// instead of reporting an error.
	AllowMissingHandlers bool
}

// View is the result of building a Document: the live windows and all named
// controls.
type View struct {
	// Windows are the built windows in file order. They are not shown yet.
	Windows []*wui.Window

	file   string
	byName map[string]interface{}
}

// Window returns the first window of the document.
func (v *View) Window() *wui.Window {
	if len(v.Windows) == 0 {
		return nil
	}
	return v.Windows[0]
}

// Lookup returns the window or control with the given Name.
func (v *View) Lookup(name string) (interface{}, bool) {
	obj, ok := v.byName[name]
	return obj, ok
}

// Control returns the control with the given Name, or nil if there is none or
// it is a window.
func (v *View) Control(name string) wui.Control {
	if c, ok := v.byName[name].(wui.Control); ok {
		return c
	}
	return nil
}

// factories create the wui object for every element type.
var factories = map[string]func() interface{}{
	"Window":      func() interface{} { return wui.NewWindow() },
	"Button":      func() interface{} { return wui.NewButton() },
	"Label":       func() interface{} { return wui.NewLabel() },
	"CheckBox":    func() interface{} { return wui.NewCheckBox() },
	"RadioButton": func() interface{} { return wui.NewRadioButton() },
	"GroupBox":    func() interface{} { return wui.NewGroupBox() },
	"StatusBar":   func() interface{} { return wui.NewStatusBar() },
	"TabControl":  func() interface{} { return wui.NewTabControl() },
	"DatePicker":  func() interface{} { return wui.NewDatePicker() },
	"Slider":      func() interface{} { return wui.NewSlider() },
	"Panel":       func() interface{} { return wui.NewPanel() },
	"PaintBox":    func() interface{} { return wui.NewPaintBox() },
	"EditLine":    func() interface{} { return wui.NewEditLine() },
	"TextEdit":    func() interface{} { return wui.NewTextEdit() },
	"IntUpDown":   func() interface{} { return wui.NewIntUpDown() },
	"FloatUpDown": func() interface{} { return wui.NewFloatUpDown() },
	"ComboBox":    func() interface{} { return wui.NewComboBox() },
	"ProgressBar": func() interface{} { return wui.NewProgressBar() },
}

// methodKey and methodIndex cache reflection lookups. The method set of a
// type never changes, so the lookup is done once per (type, name).
type methodKey struct {
	typ  reflect.Type
	name string
}

var methodIndex sync.Map // methodKey -> int, -1 if the method does not exist

func lookupMethod(v reflect.Value, name string) (reflect.Value, bool) {
	key := methodKey{v.Type(), name}
	if cached, ok := methodIndex.Load(key); ok {
		idx := cached.(int)
		if idx < 0 {
			return reflect.Value{}, false
		}
		return v.Method(idx), true
	}
	m, ok := v.Type().MethodByName(name)
	if !ok {
		methodIndex.Store(key, -1)
		return reflect.Value{}, false
	}
	methodIndex.Store(key, m.Index)
	return v.Method(m.Index), true
}

type builder struct {
	doc      *Document
	opts     BuildOptions
	handlers Handlers
	view     *View
	errs     []*Error
}

func (b *builder) fail(line, col int, format string, args ...interface{}) {
	if len(b.errs) >= maxErrors {
		return
	}
	b.errs = append(b.errs, &Error{
		File: b.doc.File,
		Line: line,
		Col:  col,
		Msg:  fmt.Sprintf(format, args...),
	})
}

// Build creates the windows and controls of the document.
//
// into is optional. If it is a pointer to a struct, every exported field with
// a `wml:"Name"` tag is set to the control with that Name (see Inject).
// handlers binds the OnXxx attributes to functions.
//
// Build must run on the GUI thread. It either returns a complete View or an
// error; it never panics.
func (d *Document) Build(into interface{}, handlers Handlers) (*View, error) {
	return d.BuildWith(BuildOptions{}, into, handlers)
}

// BuildWith is Build with options.
func (d *Document) BuildWith(opts BuildOptions, into interface{}, handlers Handlers) (view *View, err error) {
	defer func() {
		if r := recover(); r != nil {
			view = nil
			err = fmt.Errorf("wml: building %q failed: %v", d.File, r)
		}
	}()

	b := &builder{
		doc:      d,
		opts:     opts,
		handlers: handlers,
		view:     &View{file: d.File, byName: make(map[string]interface{})},
	}

	// Pass 1: check every handler before anything is created, so that a
	// missing or wrongly typed handler is reported without a half built UI.
	for _, w := range d.Windows {
		b.checkHandlers(w)
	}
	if len(b.errs) > 0 {
		return nil, ErrorList(b.errs)
	}

	// Pass 2: build.
	for _, w := range d.Windows {
		if win, ok := b.buildNode(w, nil).(*wui.Window); ok {
			b.view.Windows = append(b.view.Windows, win)
		}
	}
	if len(b.errs) > 0 {
		return nil, ErrorList(b.errs)
	}

	if into != nil {
		if err := b.view.Inject(into); err != nil {
			return nil, err
		}
	}
	return b.view, nil
}

// checkHandlers resolves all events of n and its children against a throw-away
// object of the right type.
func (b *builder) checkHandlers(n *Node) {
	ts := types[n.Type]
	newObj := factories[n.Type]
	if ts == nil || newObj == nil {
		return
	}
	proto := reflect.ValueOf(newObj())
	for _, a := range n.Attrs {
		if ts.hasEvent(a.Name) {
			b.resolveHandler(n, a, proto)
		}
	}
	for _, c := range n.Children {
		b.checkHandlers(c)
	}
}

// resolveHandler finds the setter and the handler for one event attribute and
// checks that their types fit. ok is false if the event must be skipped.
func (b *builder) resolveHandler(n *Node, a Attr, obj reflect.Value) (setter, handler reflect.Value, ok bool) {
	line, col := a.pos(n)
	h, found := b.handlers[a.Value]
	if !found {
		if !b.opts.AllowMissingHandlers {
			b.fail(line, col, "%s.%s: no handler named %q was bound", n.Type, a.Name, a.Value)
		}
		return
	}
	hv := reflect.ValueOf(h)
	if !hv.IsValid() || hv.Kind() != reflect.Func || hv.IsNil() {
		b.fail(line, col, "%s.%s: handler %q is not a function", n.Type, a.Name, a.Value)
		return
	}
	setter, exists := lookupMethod(obj, "Set"+a.Name)
	if !exists || setter.Type().NumIn() != 1 {
		b.fail(line, col, "%s does not support the event %s", n.Type, a.Name)
		return
	}
	want := setter.Type().In(0)
	if !hv.Type().AssignableTo(want) {
		b.fail(line, col, "%s.%s: handler %q has type %s but the event needs %s",
			n.Type, a.Name, a.Value, hv.Type(), want)
		return
	}
	return setter, hv, true
}

// buildNode creates the object for n, applies its properties, adds it to
// parent and builds its children. The order is the same as in the code that
// the designer generates: font, properties, Add, events, children.
func (b *builder) buildNode(n *Node, parent wui.Container) interface{} {
	ts := types[n.Type]
	newObj := factories[n.Type]
	if ts == nil || newObj == nil {
		b.fail(n.Line, n.Col, "unknown element <%s>", n.Type)
		return nil
	}
	obj := newObj()
	rv := reflect.ValueOf(obj)

	for _, c := range n.Children {
		if c.Type == fontElement {
			b.applyFont(n, c, obj)
			break
		}
	}

	for i := range ts.Props {
		p := &ts.Props[i]
		if p.Kind == KindList {
			b.applyList(n, ts, p, rv)
			continue
		}
		a := n.lookup(p.Name)
		if a == nil {
			continue
		}
		val, err := parseValue(p, a.Value)
		if err == nil {
			err = applyValue(rv, p, val)
		}
		if err != nil {
			line, col := a.pos(n)
			b.fail(line, col, "%s.%s: %v", n.Type, p.Name, err)
		}
	}

	if a := n.lookup("Name"); a != nil {
		b.view.byName[a.Value] = obj
	}

	if parent != nil {
		ctrl, ok := obj.(wui.Control)
		if !ok {
			b.fail(n.Line, n.Col, "<%s> is not a control and cannot be added to a parent", n.Type)
			return obj
		}
		parent.Add(ctrl)
	}

	for _, a := range n.Attrs {
		if !ts.hasEvent(a.Name) {
			continue
		}
		if setter, handler, ok := b.resolveHandler(n, a, rv); ok {
			setter.Call([]reflect.Value{handler})
		}
	}

	if con, ok := obj.(wui.Container); ok {
		for _, c := range n.Children {
			if c.Type == fontElement || ts.listProp(c.Type) != nil {
				continue
			}
			b.buildNode(c, con)
		}
	}
	return obj
}

func (b *builder) applyFont(owner, fn *Node, obj interface{}) {
	target, ok := obj.(interface{ SetFont(*wui.Font) })
	if !ok {
		b.fail(fn.Line, fn.Col, "%s does not support <Font>", owner.Type)
		return
	}
	var desc wui.FontDesc
	for _, a := range fn.Attrs {
		p := fontProp(a.Name)
		if p == nil {
			continue
		}
		val, err := parseValue(p, a.Value)
		if err != nil {
			continue // the validator already reported it
		}
		switch a.Name {
		case "Name":
			desc.Name = val.(string)
		case "Height":
			desc.Height = int(val.(int64))
		case "Bold":
			desc.Bold = val.(bool)
		case "Italic":
			desc.Italic = val.(bool)
		case "Underlined":
			desc.Underlined = val.(bool)
		case "StrikedOut":
			desc.StrikedOut = val.(bool)
		}
	}
	font, err := wui.NewFont(desc)
	if err != nil {
		b.fail(fn.Line, fn.Col, "cannot create the font: %v", err)
		return
	}
	target.SetFont(font)
}

// applyList sets a list property (Items, Tabs) from its child elements.
func (b *builder) applyList(n *Node, ts *TypeSpec, p *PropSpec, rv reflect.Value) {
	var items []string
	found := false
	for _, c := range n.Children {
		if c.Type == p.Elem {
			found = true
			items = append(items, c.Text)
		}
	}
	if !found {
		return
	}
	if err := applyValue(rv, p, items); err != nil {
		b.fail(n.Line, n.Col, "%s.%s: %v", n.Type, p.Name, err)
	}
}

// applyValue calls the setter of p on rv with the parsed value.
func applyValue(rv reflect.Value, p *PropSpec, val interface{}) error {
	m, ok := lookupMethod(rv, p.setterName())
	if !ok && p.Alt != "" {
		m, ok = lookupMethod(rv, p.Alt)
	}
	if !ok {
		return fmt.Errorf("this control has no setter for it (%s)", p.setterName())
	}
	args, err := makeArgs(m.Type(), p, val)
	if err != nil {
		return err
	}
	m.Call(args)
	return nil
}

var colorType = reflect.TypeOf(wui.Color(0))

// makeArgs converts a parsed value to the argument types of the setter.
func makeArgs(mt reflect.Type, p *PropSpec, val interface{}) ([]reflect.Value, error) {
	want := 1
	if p.Kind == KindInts || p.Kind == KindFloats {
		want = p.N
	}
	if mt.NumIn() != want || mt.IsVariadic() {
		return nil, fmt.Errorf("setter %s takes %d arguments, expected %d", p.setterName(), mt.NumIn(), want)
	}

	switch p.Kind {
	case KindBool:
		v := reflect.New(mt.In(0)).Elem()
		if v.Kind() != reflect.Bool {
			return nil, fmt.Errorf("setter expects %s, not a bool", mt.In(0))
		}
		v.SetBool(val.(bool))
		return []reflect.Value{v}, nil

	case KindString:
		v := reflect.New(mt.In(0)).Elem()
		if v.Kind() != reflect.String {
			return nil, fmt.Errorf("setter expects %s, not a string", mt.In(0))
		}
		v.SetString(val.(string))
		return []reflect.Value{v}, nil

	case KindInt:
		v, err := intArg(mt.In(0), val.(int64))
		if err != nil {
			return nil, err
		}
		return []reflect.Value{v}, nil

	case KindEnum:
		v, err := intArg(mt.In(0), int64(val.(int)))
		if err != nil {
			return nil, err
		}
		return []reflect.Value{v}, nil

	case KindFloat:
		v, err := floatArg(mt.In(0), val.(float64))
		if err != nil {
			return nil, err
		}
		return []reflect.Value{v}, nil

	case KindColor:
		if mt.In(0) != colorType {
			return nil, fmt.Errorf("setter expects %s, not a color", mt.In(0))
		}
		c := val.(rgb)
		return []reflect.Value{reflect.ValueOf(wui.RGB(c.R, c.G, c.B))}, nil

	case KindInts:
		nums := val.([]int64)
		args := make([]reflect.Value, len(nums))
		for i, n := range nums {
			v, err := intArg(mt.In(i), n)
			if err != nil {
				return nil, err
			}
			args[i] = v
		}
		return args, nil

	case KindFloats:
		nums := val.([]float64)
		args := make([]reflect.Value, len(nums))
		for i, n := range nums {
			v, err := floatArg(mt.In(i), n)
			if err != nil {
				return nil, err
			}
			args[i] = v
		}
		return args, nil

	case KindList:
		t := mt.In(0)
		if t.Kind() != reflect.Slice || t.Elem().Kind() != reflect.String {
			return nil, fmt.Errorf("setter expects %s, not a list of strings", t)
		}
		return []reflect.Value{reflect.ValueOf(val.([]string)).Convert(t)}, nil
	}
	return nil, fmt.Errorf("unsupported property kind %s", p.Kind)
}

func intArg(t reflect.Type, n int64) (reflect.Value, error) {
	v := reflect.New(t).Elem()
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.OverflowInt(n) {
			return reflect.Value{}, fmt.Errorf("%d does not fit into %s", n, t)
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if n < 0 || v.OverflowUint(uint64(n)) {
			return reflect.Value{}, fmt.Errorf("%d does not fit into %s", n, t)
		}
		v.SetUint(uint64(n))
	default:
		return reflect.Value{}, fmt.Errorf("setter expects %s, not a whole number", t)
	}
	return v, nil
}

func floatArg(t reflect.Type, f float64) (reflect.Value, error) {
	v := reflect.New(t).Elem()
	switch t.Kind() {
	case reflect.Float32, reflect.Float64:
		if v.OverflowFloat(f) {
			return reflect.Value{}, fmt.Errorf("%g does not fit into %s", f, t)
		}
		v.SetFloat(f)
	default:
		return reflect.Value{}, fmt.Errorf("setter expects %s, not a number", t)
	}
	return v, nil
}
