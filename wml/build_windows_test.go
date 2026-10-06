package wml

import (
	"reflect"
	"strings"
	"testing"

	"github.com/2dprototype/wui"
)

// TestSchemaMatchesWui makes sure that every property and event in the schema
// really exists on the wui type. If wui renames a setter, this test fails
// instead of a user's layout.
func TestSchemaMatchesWui(t *testing.T) {
	for name, ts := range types {
		newObj := factories[name]
		if newObj == nil {
			t.Errorf("%s: no factory", name)
			continue
		}
		rv := reflect.ValueOf(newObj())
		for _, p := range ts.Props {
			_, ok := lookupMethod(rv, p.setterName())
			if !ok && p.Alt != "" {
				_, ok = lookupMethod(rv, p.Alt)
			}
			if !ok {
				t.Errorf("%s.%s: wui has no method %s", name, p.Name, p.setterName())
			}
		}
		for _, e := range ts.Events {
			if _, ok := lookupMethod(rv, "Set"+e); !ok {
				t.Errorf("%s.%s: wui has no method Set%s", name, e, e)
			}
		}
	}
	for name := range factories {
		if types[name] == nil {
			t.Errorf("factory %s has no schema", name)
		}
	}
}

type loginForm struct {
	Window  *wui.Window   `wml:"main"`
	User    *wui.EditLine `wml:"user"`
	Combo   *wui.ComboBox `wml:"lang"`
	OK      *wui.Button   `wml:"ok"`
	Any     wui.Control   `wml:"frame"`
	Missing *wui.Label    `wml:"nope,optional"`
}

func TestBuildAndInject(t *testing.T) {
	doc, err := parseString(loginDoc)
	if err != nil {
		t.Fatal(err)
	}
	var f loginForm
	clicked := false
	view, err := doc.Build(&f, Handlers{"doLogin": func() { clicked = true }})
	if err != nil {
		t.Fatal(err)
	}
	_ = clicked

	if f.Window == nil || f.User == nil || f.Combo == nil || f.OK == nil || f.Any == nil {
		t.Fatalf("fields were not filled: %+v", f)
	}
	if f.Missing != nil {
		t.Error("optional field must stay nil")
	}
	if view.Window() != f.Window || len(view.Windows) != 1 {
		t.Error("View.Window does not match")
	}
	if f.OK.Text() != "Login" {
		t.Errorf("button text = %q", f.OK.Text())
	}
	if x, y, w, h := f.OK.Bounds(); x != 10 || y != 10 || w != 90 || h != 26 {
		t.Errorf("button bounds = %d,%d,%d,%d", x, y, w, h)
	}
	if got := len(f.Combo.Items()); got != 2 {
		t.Errorf("combo has %d items, want 2", got)
	}
	if f.Window.Title() != `a < b & "c"` {
		t.Errorf("title = %q", f.Window.Title())
	}
	if view.Control("ok") == nil || view.Control("main") != nil {
		t.Error("View.Control must return controls but not windows")
	}
}

func TestBuildHandlerErrors(t *testing.T) {
	doc, err := parseString(loginDoc)
	if err != nil {
		t.Fatal(err)
	}

	_, err = doc.Build(nil, Handlers{})
	if err == nil || !strings.Contains(err.Error(), `no handler named "doLogin"`) {
		t.Errorf("missing handler: got %v", err)
	}

	_, err = doc.Build(nil, Handlers{"doLogin": func(int) {}})
	if err == nil || !strings.Contains(err.Error(), "has type func(int)") {
		t.Errorf("wrong handler type: got %v", err)
	}

	_, err = doc.Build(nil, Handlers{"doLogin": "not a function"})
	if err == nil || !strings.Contains(err.Error(), "is not a function") {
		t.Errorf("non function handler: got %v", err)
	}

	if _, err = doc.BuildWith(BuildOptions{AllowMissingHandlers: true}, nil, nil); err != nil {
		t.Errorf("AllowMissingHandlers: %v", err)
	}
}

func TestInjectErrors(t *testing.T) {
	doc, err := parseString(`<wml><Window Name="w"><Label Name="l"/></Window></wml>`)
	if err != nil {
		t.Fatal(err)
	}
	var wrongType struct {
		L *wui.Button `wml:"l"`
	}
	if _, err = doc.Build(&wrongType, nil); err == nil || !strings.Contains(err.Error(), `"l" is a *wui.Label`) {
		t.Errorf("wrong field type: got %v", err)
	}
	var missing struct {
		X *wui.Label `wml:"x"`
	}
	if _, err = doc.Build(&missing, nil); err == nil || !strings.Contains(err.Error(), `no element with Name "x"`) {
		t.Errorf("missing element: got %v", err)
	}
	if _, err = doc.Build(missing, nil); err == nil {
		t.Error("Build accepted a non-pointer")
	}
}

func TestEveryEnumValueIsAccepted(t *testing.T) {
	src := `<wml><Window Title="enums">
  <Label Alignment="Center" HorizontalAnchor="MinAndMax" TextColor="#102030"/>
  <Slider Orientation="Vertical" TickPosition="BothSides" MinMax="0,50" CursorPosition="25"/>
  <Panel BorderStyle="Raised" BackgroundColor="#fff"/>
  <DatePicker Mode="Time"/>
  <IntUpDown MinMax="-5,5" Value="3" Position="1,2"/>
  <FloatUpDown MinMax="0,1.5" Precision="2" Value="0.25"/>
  <ProgressBar Value="0.5"/>
  <TabControl SelectedIndex="1"><Tab>A</Tab><Tab>B</Tab></TabControl>
</Window></wml>`
	doc, err := parseString(src)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doc.Build(nil, nil); err != nil {
		t.Fatal(err)
	}
}
