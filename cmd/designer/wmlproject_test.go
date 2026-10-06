package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/2dprototype/wui"
)

func TestEventBlockRoundTrip(t *testing.T) {
	codes := map[string]string{
		"button1_OnClick":    "func() {\n\ti := 5\n\ti--\n\tprintln(\"100% done -->\")\n}",
		"window_OnDropFiles": "func(files []string, x, y int) {\n\tprintln(len(files))\n\n}",
	}
	data := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<wml version=\"1\">\n</wml>\n")
	out := appendEventBlock(data, codes)

	// The comment must be valid XML: no two hyphens in a row inside of it.
	s := string(out)
	start := strings.Index(s, eventBlockStart)
	if start < 0 {
		t.Fatal("event block not written")
	}
	rest := s[start+len(eventBlockStart):]
	end := strings.Index(rest, "-->")
	if end < 0 {
		t.Fatal("event block is not closed")
	}
	if strings.Contains(rest[:end], "--") {
		t.Errorf("the comment contains two hyphens in a row:\n%s", rest[:end])
	}
	if !strings.HasSuffix(s, "</wml>\n") {
		t.Errorf("the closing tag must stay at the end:\n%s", s)
	}

	got := parseEventBlock(out)
	if len(got) != len(codes) {
		t.Fatalf("got %d handlers, want %d", len(got), len(codes))
	}
	for name, want := range codes {
		if got[name] != want {
			t.Errorf("handler %s:\n got %q\nwant %q", name, got[name], want)
		}
	}
}

func TestEventBlockAbsent(t *testing.T) {
	data := []byte("<wml version=\"1\"></wml>")
	if got := appendEventBlock(data, nil); string(got) != string(data) {
		t.Errorf("a file without code must not change, got %q", got)
	}
	if got := parseEventBlock(data); len(got) != 0 {
		t.Errorf("expected no handlers, got %v", got)
	}
}

func TestIsEmptyHandler(t *testing.T) {
	empty := []string{"", "  \n", "func() {\n\t\n}", "func(canvas *wui.Canvas) {\n\t\n}"}
	for _, code := range empty {
		if !isEmptyHandler(code) {
			t.Errorf("%q should count as empty", code)
		}
	}
	full := []string{"func() {\n\tprintln(1)\n}", "func() { x := T{}; _ = x }"}
	for _, code := range full {
		if isEmptyHandler(code) {
			t.Errorf("%q should not count as empty", code)
		}
	}
}

func TestValidName(t *testing.T) {
	for _, ok := range []string{"a", "button1", "_x", "My_Name_2"} {
		if !validName(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "1abc", "my button", "a-b", "ü", strings.Repeat("a", 65)} {
		if validName(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	win := defaultWindow()
	names = map[interface{}]string{win: "window"}
	events = make(map[event]string)

	panel := wui.NewPanel()
	panel.SetBounds(10, 20, 300, 200)
	names[panel] = "panel1"
	win.Add(panel)

	btn := wui.NewButton()
	btn.SetText(`Go <now> & "here"`)
	btn.SetBounds(5, 6, 80, 25)
	names[btn] = "button1"
	panel.Add(btn)
	events[event{control: btn, name: "OnClick"}] = "func() {\n\tprintln(\"hi\")\n}"

	combo := wui.NewComboBox()
	combo.SetItems([]string{"one", "two"})
	combo.SetSelectedIndex(1)
	combo.SetBounds(0, 0, 100, 21)
	names[combo] = "combo1"
	win.Add(combo)

	path := filepath.Join(os.TempDir(), "wui_designer_roundtrip"+projectExt)
	defer os.Remove(path)
	if err := saveProject(win, path); err != nil {
		t.Fatal(err)
	}

	loaded, notice, err := loadProject(path)
	if err != nil {
		t.Fatal(err)
	}
	if notice != "" {
		t.Errorf("unexpected notice %q", notice)
	}
	if iw, ih := loaded.InnerSize(); iw != 600 || ih != 400 {
		t.Errorf("inner size is %dx%d, want 600x400", iw, ih)
	}
	if names[loaded] != "window" {
		t.Errorf("window name is %q", names[loaded])
	}

	children := loaded.Children()
	if len(children) != 2 {
		t.Fatalf("window has %d children, want 2", len(children))
	}
	p, ok := children[0].(*wui.Panel)
	if !ok {
		t.Fatalf("first child is %T, want *wui.Panel", children[0])
	}
	if x, y, w, h := p.Bounds(); x != 10 || y != 20 || w != 300 || h != 200 {
		t.Errorf("panel bounds are %d,%d,%d,%d", x, y, w, h)
	}
	inner := p.Children()
	if len(inner) != 1 {
		t.Fatalf("panel has %d children, want 1", len(inner))
	}
	b, ok := inner[0].(*wui.Button)
	if !ok {
		t.Fatalf("panel child is %T, want *wui.Button", inner[0])
	}
	if b.Text() != `Go <now> & "here"` {
		t.Errorf("button text is %q", b.Text())
	}
	if names[b] != "button1" {
		t.Errorf("button name is %q", names[b])
	}
	if code := events[event{control: b, name: "OnClick"}]; !strings.Contains(code, `println("hi")`) {
		t.Errorf("button code is %q", code)
	}
	c, ok := children[1].(*wui.ComboBox)
	if !ok {
		t.Fatalf("second child is %T, want *wui.ComboBox", children[1])
	}
	if items := c.Items(); len(items) != 2 || items[1] != "two" || c.SelectedIndex() != 1 {
		t.Errorf("combo box has items %v, selected %d", items, c.SelectedIndex())
	}
}

func TestSaveRejectsBadNames(t *testing.T) {
	win := defaultWindow()
	names = map[interface{}]string{win: "window"}
	events = make(map[event]string)

	a := wui.NewButton()
	names[a] = "same"
	win.Add(a)
	b := wui.NewButton()
	names[b] = "same"
	win.Add(b)
	c := wui.NewButton()
	names[c] = "not valid"
	win.Add(c)

	path := filepath.Join(os.TempDir(), "wui_designer_badnames"+projectExt)
	defer os.Remove(path)
	err := saveProject(win, path)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "same") || !strings.Contains(err.Error(), "not valid") {
		t.Errorf("the error should name both problems: %v", err)
	}
}
