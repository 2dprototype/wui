package wui

import (
	"syscall"
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

// NewTabControl creates a row of tabs. It only shows the tab headers; to
// switch content, use SetOnChange and show/hide Panels. Add the TabControl to
// its parent before the panels that are displayed in its content area.
func NewTabControl() *TabControl {
	return &TabControl{}
}

type TabControl struct {
	textControl
	tabs      []string
	tabImages []int
	pages     []Control
	images    *ImageList
	selected  int
	onChange  func(index int)
}

var _ Control = (*TabControl)(nil)

func (*TabControl) canFocus() bool { return true }

func (*TabControl) eatsTabs() bool { return false }

func (t *TabControl) create(id int) {
	t.textControl.create(id, 0, w32.TAB_CLASS, w32.WS_TABSTOP|w32.WS_CLIPSIBLINGS)
	if t.images != nil {
		w32.SendMessage(t.handle, w32.TCM_SETIMAGELIST, 0, t.images.handle)
	}
	for i, title := range t.tabs {
		t.insertTab(i, title)
	}
	if len(t.tabs) > 0 {
		w32.SendMessage(t.handle, w32.TCM_SETCURSEL, uintptr(t.selected), 0)
	}
	t.showPages()
}

func (t *TabControl) insertTab(index int, title string) {
	item := w32.TCITEM{
		Mask:    w32.TCIF_TEXT,
		PszText: syscall.StringToUTF16Ptr(title),
	}
	if img := t.imageFor(index); img >= 0 && t.images != nil {
		item.Mask |= 2 // TCIF_IMAGE
		item.IImage = int32(img)
	}
	w32.SendMessage(
		t.handle, w32.TCM_INSERTITEMW, uintptr(index), uintptr(unsafe.Pointer(&item)),
	)
}

// AddTab appends a tab with the given title.
func (t *TabControl) AddTab(title string) {
	t.appendTab(title, -1, nil)
}

// Tabs returns the titles of all tabs.
func (t *TabControl) Tabs() []string {
	return append([]string(nil), t.tabs...)
}

// SetTabs replaces all tabs.
func (t *TabControl) SetTabs(titles []string) {
	t.tabs = append([]string(nil), titles...)
	t.tabImages = make([]int, len(titles))
	for i := range t.tabImages {
		t.tabImages[i] = -1
	}
	t.pages = make([]Control, len(titles))
	if t.selected >= len(t.tabs) {
		t.selected = 0
	}
	if t.handle != 0 {
		w32.SendMessage(t.handle, w32.TCM_DELETEALLITEMS, 0, 0)
		for i, title := range t.tabs {
			t.insertTab(i, title)
		}
		if len(t.tabs) > 0 {
			w32.SendMessage(t.handle, w32.TCM_SETCURSEL, uintptr(t.selected), 0)
		}
	}
}

// SelectedIndex returns the index of the selected tab or -1 if there are none.
func (t *TabControl) SelectedIndex() int {
	if len(t.tabs) == 0 {
		return -1
	}
	return t.selected
}

// SetSelectedIndex selects the tab at the given index and calls the change
// handler if the selection changed.
func (t *TabControl) SetSelectedIndex(i int) {
	if i < 0 || i >= len(t.tabs) || i == t.selected {
		return
	}
	t.selected = i
	if t.handle != 0 {
		w32.SendMessage(t.handle, w32.TCM_SETCURSEL, uintptr(i), 0)
	}
	t.showPages()
	if t.onChange != nil {
		t.onChange(i)
	}
}

// SetOnChange sets the function called when the selected tab changes.
func (t *TabControl) SetOnChange(f func(index int)) {
	t.onChange = f
}

func (t *TabControl) OnChange() func(index int) {
	return t.onChange
}

// ContentBounds returns the area below the tab headers in the coordinates of
// the TabControl's parent. Use it to place a Panel for the tab's content.
func (t *TabControl) ContentBounds() (x, y, width, height int) {
	if t.handle == 0 {
		return t.x, t.y, t.width, t.height
	}
	r := w32.RECT{Left: 0, Top: 0, Right: int32(t.width), Bottom: int32(t.height)}
	w32.SendMessage(t.handle, w32.TCM_ADJUSTRECT, 0, uintptr(unsafe.Pointer(&r)))
	return t.x + int(r.Left), t.y + int(r.Top), int(r.Right - r.Left), int(r.Bottom - r.Top)
}

func (t *TabControl) notify(code uint32) {
	if code != w32.TCN_SELCHANGE || t.handle == 0 {
		return
	}
	sel := int(int32(w32.SendMessage(t.handle, w32.TCM_GETCURSEL, 0, 0)))
	if sel >= 0 && sel != t.selected {
		t.selected = sel
		t.showPages()
		if t.onChange != nil {
			t.onChange(sel)
		}
	}
}
