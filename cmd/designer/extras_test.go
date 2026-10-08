package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/2dprototype/wui"
	"github.com/2dprototype/wui/check"
)

func TestParseTreeLines(t *testing.T) {
	items := parseTreeLines([]string{"A", "  B", "  C", "    D", "", "E"})
	want := []treeItem{
		{"A", 0, true},
		{"B", 1, false},
		{"C", 1, true},
		{"D", 2, false},
		{"E", 0, false},
	}
	if len(items) != len(want) {
		t.Fatalf("got %d items, want %d", len(items), len(want))
	}
	for i := range want {
		if items[i] != want[i] {
			t.Errorf("item %d is %+v, want %+v", i, items[i], want[i])
		}
	}
}

func TestTreeGoLines(t *testing.T) {
	got := treeGoLines("tree", []string{"Root", "  Child", "Other"})
	want := []string{
		`treeNode0 := tree.Add("Root")`,
		`treeNode0.Add("Child")`,
		`tree.Add("Other")`,
	}
	check.Eq(t, got, want)
}

func TestExtraGoLines(t *testing.T) {
	b := wui.NewButton()
	check.Eq(t, len(extraGoLines("b", b)), 0)

	setExtraByName(b, "Kind", "Split")
	setExtraByName(b, "Note", "hello")
	setExtraByName(b, "Default", "true")
	check.Eq(t, extraGoLines("b", b), []string{
		`b.SetKind(1)`,
		`b.SetNote("hello")`,
		`b.SetDefault(true)`,
	})

	// A value that equals the default is not stored.
	setExtraByName(b, "Kind", "Normal")
	if _, ok := extras[b]["Kind"]; ok {
		t.Error("the default value must not be stored")
	}

	lv := wui.NewListView()
	setExtraByName(lv, "Columns", "A\nB")
	setExtraByName(lv, "HeaderVisible", "false")
	check.Eq(t, extraGoLines("lv", lv), []string{
		`lv.SetHeaderVisible(false)`,
		`lv.SetColumns([]string{"A", "B"})`,
	})

	sb := wui.NewScrollBar(false)
	setExtraByName(sb, "Range", "0,50")
	check.Eq(t, extraGoLines("sb", sb), []string{`sb.SetRange(0, 50)`})
}

func TestEventTemplate(t *testing.T) {
	check.Eq(t, eventTemplate(wui.NewButton(), "OnClick"), "func() {\n\t\n}")
	check.Eq(t, eventTemplate(wui.NewButton(), "OnMouseDown"),
		"func(x int, y int, button wui.MouseButton) {\n\t\n}")
	check.Eq(t, eventTemplate(wui.NewWindow(), "OnCanClose"),
		"func() bool {\n\treturn true\n}")
	check.Eq(t, eventTemplate(wui.NewButton(), "OnNoSuchEvent"), "func() {\n\t\n}")
}

func TestEventsOfEveryType(t *testing.T) {
	controls := []interface{}{
		wui.NewWindow(), wui.NewButton(), wui.NewListView(), wui.NewRichEdit(),
		wui.NewScrollBar(false), wui.NewTreeView(), wui.NewImageView(),
	}
	for _, c := range controls {
		for _, ev := range eventsOf(c) {
			if _, ok := reflectMethod(c, "Set"+ev); !ok {
				t.Errorf("%s has the event %s but no Set%s", typeNameOf(c), ev, ev)
			}
		}
	}
}

func TestGoImports(t *testing.T) {
	check.Eq(t, goImports("func(date time.Time) { fmt.Println(date) }"), []string{"fmt", "time"})
	check.Eq(t, len(goImports("myfmt.Println(1)")), 0)
}

func TestEveryPropertyListIsValid(t *testing.T) {
	// generateProperties panics for a listed property that has no getter.
	controls := []interface{}{
		wui.NewButton(), wui.NewLabel(), wui.NewCheckBox(), wui.NewRadioButton(),
		wui.NewGroupBox(), wui.NewStatusBar(), wui.NewTabControl(), wui.NewDatePicker(), wui.NewSlider(),
		wui.NewPanel(), wui.NewPaintBox(), wui.NewEditLine(), wui.NewTextEdit(), wui.NewIntUpDown(),
		wui.NewFloatUpDown(), wui.NewComboBox(), wui.NewProgressBar(), wui.NewListView(), wui.NewRichEdit(),
		wui.NewLinkLabel(), wui.NewMonthCalendar(), wui.NewHotKeyEdit(), wui.NewIPAddressEdit(),
		wui.NewImageView(), wui.NewScrollPanel(), wui.NewScrollBar(false), wui.NewTreeView(),
	}
	for _, c := range controls {
		if got := generateProperties("x", c); len(got) != 0 {
			t.Errorf("%s: a new control should have no non-default properties, got %v", typeNameOf(c), got)
		}
	}
}

func TestSaveLoadExtras(t *testing.T) {
	win := defaultWindow()
	names = map[interface{}]string{win: "window"}
	events = make(map[event]string)
	extras = make(map[interface{}]map[string]string)
	setExtraByName(win, "MinSize", "300,200")

	btn := wui.NewButton()
	btn.SetBounds(5, 5, 100, 25)
	names[btn] = "button1"
	win.Add(btn)
	setExtraByName(btn, "Kind", "Split")
	setExtraByName(btn, "ToolTip", "Hello <tip>")

	scroll := wui.NewScrollPanel()
	scroll.SetBounds(10, 40, 200, 100)
	names[scroll] = "scroll1"
	win.Add(scroll)

	lv := wui.NewListView()
	lv.SetBounds(0, 0, 150, 80)
	names[lv] = "list1"
	scroll.Add(lv)
	setExtraByName(lv, "Columns", "Name\nSize")
	setExtraByName(lv, "Items", "a\nb\nc")
	setExtraByName(lv, "GridLines", "true")

	tree := wui.NewTreeView()
	tree.SetBounds(220, 40, 150, 80)
	names[tree] = "tree1"
	win.Add(tree)
	setExtraByName(tree, "Nodes", "Root\n  Child")
	events[event{control: tree, name: "OnSelect"}] = "func(node *wui.TreeNode) {\n\tprintln(1)\n}"

	path := filepath.Join(os.TempDir(), "wui_designer_extras"+projectExt)
	defer os.Remove(path)
	if err := saveProject(win, path); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := loadProject(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := extraValue(loaded, "MinSize"); got != "300,200" {
		t.Errorf("MinSize is %q", got)
	}
	children := loaded.Children()
	if len(children) != 3 {
		t.Fatalf("window has %d children, want 3", len(children))
	}
	b, ok := children[0].(*wui.Button)
	if !ok {
		t.Fatalf("first child is %T", children[0])
	}
	if extraValue(b, "Kind") != "Split" || extraValue(b, "ToolTip") != "Hello <tip>" {
		t.Errorf("button extras are %v", extras[b])
	}
	sp, ok := children[1].(*wui.ScrollPanel)
	if !ok {
		t.Fatalf("second child is %T", children[1])
	}
	inner := sp.Children()
	if len(inner) != 1 {
		t.Fatalf("scroll panel has %d children", len(inner))
	}
	l, ok := inner[0].(*wui.ListView)
	if !ok {
		t.Fatalf("scroll panel child is %T", inner[0])
	}
	if extraValue(l, "Columns") != "Name\nSize" || extraValue(l, "Items") != "a\nb\nc" || !extraBool(l, "GridLines") {
		t.Errorf("list view extras are %v", extras[l])
	}
	tv, ok := children[2].(*wui.TreeView)
	if !ok {
		t.Fatalf("third child is %T, want *wui.TreeView", children[2])
	}
	if extraValue(tv, "Nodes") != "Root\n  Child" {
		t.Errorf("tree nodes are %q", extraValue(tv, "Nodes"))
	}
	if names[tv] != "tree1" {
		t.Errorf("tree name is %q", names[tv])
	}
	if code := events[event{control: tv, name: "OnSelect"}]; !strings.Contains(code, "println(1)") {
		t.Errorf("tree event code is %q", code)
	}
}
