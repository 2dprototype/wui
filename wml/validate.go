package wml

import "fmt"

// validator checks a Document against the schema. It never stops at the first
// problem: everything it finds is added to the collector.
type validator struct {
	c       *collector
	lenient bool
	warns   []*Error
	names   map[string]*Node
}

func (v *validator) warn(line, col int, format string, args ...interface{}) {
	if len(v.warns) >= maxErrors {
		return
	}
	v.warns = append(v.warns, &Error{
		File: v.c.file,
		Line: line,
		Col:  col,
		Msg:  fmt.Sprintf(format, args...),
	})
}

func (a Attr) pos(n *Node) (line, col int) {
	if a.Line > 0 {
		return a.Line, a.Col
	}
	return n.Line, n.Col
}

func (v *validator) document(d *Document) {
	v.names = make(map[string]*Node)
	if d.Version != 0 && d.Version != CurrentVersion {
		v.c.add(0, 0, "unsupported version %d, this package understands version %d", d.Version, CurrentVersion)
	}
	if len(d.Windows) == 0 {
		v.c.add(0, 0, "the document contains no <Window>")
	}
	for _, w := range d.Windows {
		if w.Type != "Window" {
			v.c.add(w.Line, w.Col, "<wml> may only contain <Window> elements, found <%s>", w.Type)
			continue
		}
		v.node(w)
	}
}

// node validates a control element (or a Window) and everything below it.
func (v *validator) node(n *Node) {
	ts := types[n.Type]
	if ts == nil {
		msg := fmt.Sprintf("unknown element <%s>", n.Type)
		if s := closest(n.Type, TypeNames()); s != "" {
			msg += fmt.Sprintf("; did you mean <%s>?", s)
		}
		v.c.add(n.Line, n.Col, "%s", msg)
		return
	}
	v.attrs(n, ts)
	v.children(n, ts)
}

func (v *validator) attrs(n *Node, ts *TypeSpec) {
	seen := make(map[string]bool, len(n.Attrs))
	present := make(map[string]bool, len(n.Attrs))

	for _, a := range n.Attrs {
		line, col := a.pos(n)
		if seen[a.Name] {
			v.c.add(line, col, "duplicate attribute %q on %s", a.Name, n.Type)
			continue
		}
		seen[a.Name] = true

		switch {
		case a.Name == "Name":
			v.name(n, a)
		case ts.hasEvent(a.Name):
			if !isIdent(a.Value, true) {
				v.c.add(line, col, "%s.%s: %q is not a valid handler name", n.Type, a.Name, a.Value)
			}
		default:
			p := ts.prop(a.Name)
			if p == nil {
				v.unknownAttr(n, ts, a)
				continue
			}
			if p.Kind == KindList {
				v.c.add(line, col, "%s.%s is written as <%s> child elements, not as an attribute", n.Type, p.Name, p.Elem)
				continue
			}
			if _, err := parseValue(p, a.Value); err != nil {
				v.c.add(line, col, "%s.%s: %v", n.Type, p.Name, err)
			}
			present[p.Name] = true
		}
	}

	for _, p := range ts.Props {
		if !present[p.Name] {
			continue
		}
		for _, other := range p.Covers {
			if present[other] {
				v.c.add(n.Line, n.Col, "%s.%s cannot be combined with %s, both set the same values", n.Type, p.Name, other)
			}
		}
	}
}

func (v *validator) name(n *Node, a Attr) {
	line, col := a.pos(n)
	if !isIdent(a.Value, false) {
		v.c.add(line, col, "Name %q is not valid, use letters, digits and underscores, starting with a letter or underscore (at most 64 characters)", a.Value)
		return
	}
	if first, dup := v.names[a.Value]; dup {
		v.c.add(line, col, "duplicate Name %q, first used on line %d", a.Value, first.Line)
		return
	}
	v.names[a.Value] = n
}

func (v *validator) unknownAttr(n *Node, ts *TypeSpec, a Attr) {
	line, col := a.pos(n)
	candidates := []string{"Name"}
	for _, p := range ts.Props {
		candidates = append(candidates, p.Name)
	}
	candidates = append(candidates, ts.Events...)

	msg := fmt.Sprintf("unknown property %q on %s", a.Name, n.Type)
	if s := closest(a.Name, candidates); s != "" {
		msg += fmt.Sprintf("; did you mean %q?", s)
	}
	if v.lenient {
		v.warn(line, col, "%s", msg)
		return
	}
	v.c.add(line, col, "%s", msg)
}

func (v *validator) children(n *Node, ts *TypeSpec) {
	if n.Text != "" {
		v.c.add(n.Line, n.Col, "<%s> cannot contain text", n.Type)
	}
	fonts := 0
	for _, c := range n.Children {
		switch {
		case c.Type == fontElement:
			fonts++
			if fonts > 1 {
				v.c.add(c.Line, c.Col, "only one <Font> is allowed in <%s>", n.Type)
			}
			v.font(c)
		case ts.listProp(c.Type) != nil:
			v.listItem(c)
		default:
			v.child(n, ts, c)
		}
	}
}

// child validates a nested control.
func (v *validator) child(parent *Node, ts *TypeSpec, c *Node) {
	cts := types[c.Type]
	switch {
	case !ts.Container:
		msg := fmt.Sprintf("<%s> cannot contain <%s>", parent.Type, c.Type)
		switch {
		case cts != nil && parent.Type == "GroupBox":
			msg += "; a GroupBox only draws a frame, put the controls after it in the same parent or use a Panel"
		case cts != nil:
			msg += "; only Window and Panel can contain controls"
		case c.Type == itemElement || c.Type == tabElement:
			for _, p := range ts.Props {
				if p.Kind == KindList {
					msg += fmt.Sprintf("; this control takes <%s> elements", p.Elem)
				}
			}
		}
		v.c.add(c.Line, c.Col, "%s", msg)
	case c.Type == "Window":
		v.c.add(c.Line, c.Col, "a <Window> cannot be nested inside <%s>", parent.Type)
	case cts == nil:
		msg := fmt.Sprintf("unknown element <%s>", c.Type)
		if s := closest(c.Type, TypeNames()); s != "" {
			msg += fmt.Sprintf("; did you mean <%s>?", s)
		}
		v.c.add(c.Line, c.Col, "%s", msg)
	default:
		v.node(c)
	}
}

// listItem validates an <Item> or <Tab> element.
func (v *validator) listItem(c *Node) {
	if len(c.Attrs) > 0 {
		v.c.add(c.Line, c.Col, "<%s> takes no attributes, put the text between the tags", c.Type)
	}
	if len(c.Children) > 0 {
		v.c.add(c.Line, c.Col, "<%s> cannot contain elements", c.Type)
	}
	if len(c.Text) > maxStringLen {
		v.c.add(c.Line, c.Col, "<%s> text is longer than %d bytes", c.Type, maxStringLen)
	}
}

// font validates a <Font> element.
func (v *validator) font(c *Node) {
	if len(c.Children) > 0 || c.Text != "" {
		v.c.add(c.Line, c.Col, "<Font> cannot contain anything, set its values as attributes")
	}
	seen := make(map[string]bool, len(c.Attrs))
	for _, a := range c.Attrs {
		line, col := a.pos(c)
		if seen[a.Name] {
			v.c.add(line, col, "duplicate attribute %q on Font", a.Name)
			continue
		}
		seen[a.Name] = true

		p := fontProp(a.Name)
		if p == nil {
			names := make([]string, len(fontProps))
			for i := range fontProps {
				names[i] = fontProps[i].Name
			}
			msg := fmt.Sprintf("unknown property %q on Font", a.Name)
			if s := closest(a.Name, names); s != "" {
				msg += fmt.Sprintf("; did you mean %q?", s)
			}
			if v.lenient {
				v.warn(line, col, "%s", msg)
			} else {
				v.c.add(line, col, "%s", msg)
			}
			continue
		}
		if _, err := parseValue(p, a.Value); err != nil {
			v.c.add(line, col, "Font.%s: %v", p.Name, err)
		}
	}
}
