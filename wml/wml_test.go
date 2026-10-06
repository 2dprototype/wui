package wml

import (
	"bytes"
	"strings"
	"testing"
)

const loginDoc = `<?xml version="1.0" encoding="UTF-8"?>
<wml version="1">
  <!-- a comment -->
  <Window Name="main" Title="a &lt; b &amp; &quot;c&quot;" InnerSize="400,300">
    <Font Name="Tahoma" Height="-11"/>
    <GroupBox Name="frame" Text="Account" Bounds="10,10,380,120"/>
    <Label Text="User" Position="20,35"/>
    <EditLine Name="user" Bounds="80,32,280,22"/>
    <ComboBox Name="lang" Bounds="10,140,150,22" SelectedIndex="0">
      <Item>English</Item>
      <Item>Bangla &amp; more</Item>
    </ComboBox>
    <Panel Name="box" Bounds="10,170,380,80" BorderStyle="Sunken">
      <Button Name="ok" Text="Login" Bounds="10,10,90,26" OnClick="doLogin"/>
    </Panel>
  </Window>
</wml>`

func parseString(src string, opts ...ParseOption) (*Document, error) {
	opts = append([]ParseOption{WithFileName("t.xml")}, opts...)
	return Parse(strings.NewReader(src), opts...)
}

func allErrors(err error) []*Error {
	switch e := err.(type) {
	case ErrorList:
		return e
	case *Error:
		return []*Error{e}
	}
	return nil
}

func TestParseValid(t *testing.T) {
	doc, err := parseString(loginDoc)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Windows) != 1 {
		t.Fatalf("got %d windows, want 1", len(doc.Windows))
	}
	w := doc.Windows[0]
	if got, _ := w.Attr("Title"); got != `a < b & "c"` {
		t.Errorf("Title = %q", got)
	}
	if len(w.Children) != 6 {
		t.Fatalf("window has %d children, want 6", len(w.Children))
	}
	combo := w.Children[4]
	if combo.Type != "ComboBox" || len(combo.Children) != 2 {
		t.Fatalf("unexpected combo box node: %+v", combo)
	}
	if combo.Children[1].Text != "Bangla & more" {
		t.Errorf("item text = %q", combo.Children[1].Text)
	}
	if combo.Line != 9 {
		t.Errorf("combo line = %d, want 9", combo.Line)
	}
}

func TestMinimalDocuments(t *testing.T) {
	for _, src := range []string{
		`<wml><Window/></wml>`,
		`<wml version="1"><Window/><Window Name="second"/></wml>`,
		"\xef\xbb\xbf<wml><Window/></wml>",
	} {
		if _, err := parseString(src); err != nil {
			t.Errorf("%q: %v", src, err)
		}
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
		pos  string // expected "line:col" prefix of the message, empty to skip
	}{
		{
			name: "unknown property with suggestion",
			src:  "<wml version=\"1\">\n  <Window>\n    <Button Txet=\"x\"/>\n  </Window>\n</wml>",
			want: `unknown property "Txet" on Button; did you mean "Text"?`,
			pos:  "t.xml:3:13:",
		},
		{
			name: "not a number",
			src:  `<wml><Window><Label Width="abc"/></Window></wml>`,
			want: `Label.Width: "abc" is not a whole number`,
		},
		{
			name: "out of range",
			src:  `<wml><Window><Label Width="-5"/></Window></wml>`,
			want: `Label.Width: -5 is out of range`,
		},
		{
			name: "negative size inside bounds",
			src:  `<wml><Window><Label Bounds="1,2,-3,4"/></Window></wml>`,
			want: `must not be negative`,
		},
		{
			name: "group box is not a container",
			src:  `<wml><Window><GroupBox><Label/></GroupBox></Window></wml>`,
			want: `<GroupBox> cannot contain <Label>`,
		},
		{
			name: "button is not a container",
			src:  `<wml><Window><Button><Label/></Button></Window></wml>`,
			want: `only Window and Panel can contain controls`,
		},
		{
			name: "window nested",
			src:  `<wml><Window><Panel><Window/></Panel></Window></wml>`,
			want: `a <Window> cannot be nested inside <Panel>`,
		},
		{
			name: "doctype",
			src:  `<!DOCTYPE wml><wml><Window/></wml>`,
			want: `DOCTYPE and other directives are not allowed`,
		},
		{
			name: "processing instruction",
			src:  `<wml><?php echo 1 ?><Window/></wml>`,
			want: `processing instructions are not allowed`,
		},
		{
			name: "namespace",
			src:  `<wml xmlns="http://example.com"><Window/></wml>`,
			want: `XML namespaces are not supported`,
		},
		{
			name: "wrong root",
			src:  `<ui><Window/></ui>`,
			want: `the root element must be <wml>, found <ui>`,
		},
		{
			name: "unsupported version",
			src:  `<wml version="2"><Window/></wml>`,
			want: `unsupported version "2"`,
		},
		{
			name: "syntax error",
			src:  `<wml><Window></wml>`,
			want: `XML syntax error`,
		},
		{
			name: "empty document",
			src:  ``,
			want: `the document is empty`,
		},
		{
			name: "no window",
			src:  `<wml/>`,
			want: `contains no <Window>`,
		},
		{
			name: "duplicate name",
			src:  `<wml><Window><Label Name="a"/><Label Name="a"/></Window></wml>`,
			want: `duplicate Name "a"`,
		},
		{
			name: "invalid name",
			src:  `<wml><Window><Label Name="1bad"/></Window></wml>`,
			want: `Name "1bad" is not valid`,
		},
		{
			name: "duplicate attribute",
			src:  `<wml><Window><Label Text="a" Text="b"/></Window></wml>`,
			want: `duplicate attribute "Text"`,
		},
		{
			name: "conflicting composites",
			src:  `<wml><Window><Button Bounds="1,2,3,4" X="1"/></Window></wml>`,
			want: `Button.Bounds cannot be combined with X`,
		},
		{
			name: "invalid handler name",
			src:  `<wml><Window><Button OnClick="do login"/></Window></wml>`,
			want: `is not a valid handler name`,
		},
		{
			name: "event of another control",
			src:  `<wml><Window><Label OnClick="x"/></Window></wml>`,
			want: `unknown property "OnClick" on Label`,
		},
		{
			name: "list as attribute",
			src:  `<wml><Window><ComboBox Items="a,b"/></Window></wml>`,
			want: `written as <Item> child elements`,
		},
		{
			name: "bad enum",
			src:  `<wml><Window><Slider Orientation="Sideways"/></Window></wml>`,
			want: `use one of: Horizontal, Vertical`,
		},
		{
			name: "bad color",
			src:  `<wml><Window><Panel BackgroundColor="red"/></Window></wml>`,
			want: `is not a color`,
		},
		{
			name: "bad bool",
			src:  `<wml><Window><Button Enabled="yes"/></Window></wml>`,
			want: `is not a bool`,
		},
		{
			name: "text in button",
			src:  `<wml><Window><Button>hello</Button></Window></wml>`,
			want: `<Button> cannot contain text`,
		},
		{
			name: "unknown element with suggestion",
			src:  `<wml><Window><Buton/></Window></wml>`,
			want: `unknown element <Buton>; did you mean <Button>?`,
		},
		{
			name: "item in wrong control",
			src:  `<wml><Window><TabControl><Item>a</Item></TabControl></Window></wml>`,
			want: `this control takes <Tab> elements`,
		},
		{
			name: "two fonts",
			src:  `<wml><Window><Font Name="a"/><Font Name="b"/></Window></wml>`,
			want: `only one <Font> is allowed`,
		},
		{
			name: "bad font attribute",
			src:  `<wml><Window><Font Size="3"/></Window></wml>`,
			want: `unknown property "Size" on Font`,
		},
		{
			name: "window only at top level",
			src:  `<wml><Button/></wml>`,
			want: `<wml> may only contain <Window> elements`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := parseString(tc.src)
			if err == nil {
				t.Fatalf("expected an error containing %q, got a document: %+v", tc.want, doc)
			}
			if !strings.Contains(err.Error(), tc.want) && !strings.Contains(detailsOf(err), tc.want) {
				t.Fatalf("error %q does not contain %q", detailsOf(err), tc.want)
			}
			if tc.pos != "" && !strings.HasPrefix(allErrors(err)[0].Error(), tc.pos) {
				t.Fatalf("error %q does not start with %q", allErrors(err)[0].Error(), tc.pos)
			}
		})
	}
}

func detailsOf(err error) string {
	if l, ok := err.(ErrorList); ok {
		return l.Details()
	}
	return err.Error()
}

func TestAllErrorsAreReported(t *testing.T) {
	_, err := parseString(`<wml><Window><Label Width="x"/><Label Height="y"/><Buton/></Window></wml>`)
	if got := len(allErrors(err)); got != 3 {
		t.Fatalf("got %d errors, want 3:\n%v", got, detailsOf(err))
	}
}

func TestLimits(t *testing.T) {
	deep := `<wml><Window><Panel><Panel/></Panel></Window></wml>`
	if _, err := parseString(deep, WithLimits(Limits{MaxDepth: 3})); err == nil ||
		!strings.Contains(err.Error(), "nested deeper than 3 levels") {
		t.Errorf("depth limit: got %v", err)
	}
	if _, err := parseString(deep, WithLimits(Limits{MaxBytes: 10})); err == nil ||
		!strings.Contains(err.Error(), "larger than 10 bytes") {
		t.Errorf("size limit: got %v", err)
	}
	if _, err := parseString(deep, WithLimits(Limits{MaxNodes: 2})); err == nil ||
		!strings.Contains(err.Error(), "more than 2 elements") {
		t.Errorf("node limit: got %v", err)
	}
	if _, err := parseString(deep); err != nil {
		t.Errorf("default limits rejected a small document: %v", err)
	}
}

func TestLenient(t *testing.T) {
	src := `<wml><Window><Button Txet="x"/></Window></wml>`
	if _, err := parseString(src); err == nil {
		t.Fatal("strict mode accepted an unknown property")
	}
	doc, err := parseString(src, Lenient())
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Warnings) != 1 || !strings.Contains(doc.Warnings[0].Msg, `unknown property "Txet"`) {
		t.Errorf("warnings = %v", doc.Warnings)
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	doc, err := parseString(loginDoc)
	if err != nil {
		t.Fatal(err)
	}
	out := doc.Marshal()
	doc2, err := Parse(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("marshaled document does not parse: %v\n%s", err, out)
	}
	if got, _ := doc2.Windows[0].Attr("Title"); got != `a < b & "c"` {
		t.Errorf("Title after round trip = %q", got)
	}
	if out2 := doc2.Marshal(); !bytes.Equal(out, out2) {
		t.Errorf("Marshal is not stable:\n%s\n---\n%s", out, out2)
	}
}

func TestProgrammaticDocument(t *testing.T) {
	w := &Node{Type: "Window"}
	w.SetAttr("Title", "Hello")
	b := w.Add(&Node{Type: "Button"})
	b.SetAttr("Text", "OK")
	b.SetAttr("Text", "Okay")

	doc := NewDocument(w)
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
	if got, _ := b.Attr("Text"); got != "Okay" {
		t.Errorf("SetAttr did not replace: %q", got)
	}

	b.SetAttr("Bogus", "1")
	if err := doc.Validate(); err == nil {
		t.Error("Validate accepted an unknown property")
	}
}

func TestValues(t *testing.T) {
	if c, err := parseColor("#fff"); err != nil || c != (rgb{255, 255, 255}) {
		t.Errorf("#fff = %v, %v", c, err)
	}
	if c, err := parseColor("#102030"); err != nil || c != (rgb{0x10, 0x20, 0x30}) {
		t.Errorf("#102030 = %v, %v", c, err)
	}
	for _, bad := range []string{"", "fff", "#ff", "#ggg", "#1234567", "red"} {
		if _, err := parseColor(bad); err == nil {
			t.Errorf("color %q was accepted", bad)
		}
	}

	if v, err := parseInts("10, 20,30,40", 4, -100, 100, 2); err != nil || len(v) != 4 || v[1] != 20 {
		t.Errorf("parseInts = %v, %v", v, err)
	}
	if _, err := parseInts("1,2,3", 4, 0, 10, 0); err == nil {
		t.Error("wrong number of ints accepted")
	}
	if _, err := parseInts("1,2,-3,4", 4, -10, 10, 2); err == nil {
		t.Error("negative size accepted")
	}

	if i, err := parseEnum("minandmax", anchorNames); err != nil || i != 3 {
		t.Errorf("parseEnum = %d, %v", i, err)
	}
	if _, err := parseEnum("middle", anchorNames); err == nil {
		t.Error("unknown enum name accepted")
	}

	for _, bad := range []string{"NaN", "Inf", "x", ""} {
		if _, err := parseFloat(bad, -10, 10); err == nil {
			t.Errorf("float %q was accepted", bad)
		}
	}
	if _, err := parseFloat("11", -10, 10); err == nil {
		t.Error("out of range float accepted")
	}

	for s, want := range map[string]bool{
		"ok": true, "_x1": true, "A_b": true, "1a": false, "a-b": false, "": false, "a.b": false,
	} {
		if got := isIdent(s, false); got != want {
			t.Errorf("isIdent(%q) = %v", s, got)
		}
	}
	if !isIdent("App.Save", true) || isIdent(".x", true) {
		t.Error("isIdent with dots is wrong")
	}

	if d := editDistance("kitten", "sitting"); d != 3 {
		t.Errorf("editDistance = %d, want 3", d)
	}
}

// TestSchemaIsConsistent catches typos in the schema table itself.
func TestSchemaIsConsistent(t *testing.T) {
	for _, name := range TypeNames() {
		ts := Lookup(name)
		seen := map[string]bool{}
		for _, p := range ts.Props {
			if seen[p.Name] {
				t.Errorf("%s: property %s is defined twice", name, p.Name)
			}
			seen[p.Name] = true
		}
		for _, p := range ts.Props {
			switch p.Kind {
			case KindEnum:
				if len(p.Enum) == 0 {
					t.Errorf("%s.%s: enum without names", name, p.Name)
				}
			case KindInt, KindInts:
				if p.Min > p.Max {
					t.Errorf("%s.%s: Min > Max", name, p.Name)
				}
			case KindFloat, KindFloats:
				if p.FMin > p.FMax {
					t.Errorf("%s.%s: FMin > FMax", name, p.Name)
				}
			case KindList:
				if p.Elem == "" {
					t.Errorf("%s.%s: list without element name", name, p.Name)
				}
			}
			if p.Kind == KindInts || p.Kind == KindFloats {
				if p.N < 2 {
					t.Errorf("%s.%s: composite with N=%d", name, p.Name, p.N)
				}
			}
			for _, c := range p.Covers {
				if !seen[c] {
					t.Errorf("%s.%s covers unknown property %s", name, p.Name, c)
				}
			}
		}
		evs := map[string]bool{}
		for _, e := range ts.Events {
			if !strings.HasPrefix(e, "On") {
				t.Errorf("%s: event %s does not start with On", name, e)
			}
			if evs[e] {
				t.Errorf("%s: event %s is listed twice", name, e)
			}
			evs[e] = true
			if seen[e] {
				t.Errorf("%s: %s is both a property and an event", name, e)
			}
		}
	}
	if Lookup("Window") == nil || !Lookup("Window").Container || !Lookup("Panel").Container {
		t.Error("Window and Panel must be containers")
	}
	if Lookup("GroupBox").Container {
		t.Error("GroupBox is not a container in wui")
	}
}
