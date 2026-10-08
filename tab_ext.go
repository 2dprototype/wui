package wui

import "github.com/2dprototype/wui/w32"

func (t *TabControl) imageFor(i int) int {
	if i >= 0 && i < len(t.tabImages) {
		return t.tabImages[i]
	}
	return -1
}

func (t *TabControl) appendTab(title string, image int, page Control) {
	t.tabs = append(t.tabs, title)
	t.tabImages = append(t.tabImages, image)
	t.pages = append(t.pages, page)
	if t.handle != 0 {
		t.insertTab(len(t.tabs)-1, title)
		t.showPages()
	}
}

// SetImages gives the tab headers icons, see AddTabWithImage.
func (t *TabControl) SetImages(il *ImageList) {
	t.images = il
	if t.handle != 0 && il != nil {
		w32.SendMessage(t.handle, w32.TCM_SETIMAGELIST, 0, il.handle)
	}
}

// AddTabWithImage appends a tab with an icon (an index into the image list
// from SetImages).
func (t *TabControl) AddTabWithImage(title string, image int) {
	t.appendTab(title, image, nil)
}

// AddPage appends a tab whose content is the given control, usually a Panel.
// The TabControl shows only the page of the selected tab, hidden pages are
// invisible, and it sizes the shown page to the content area whenever the
// TabControl is resized. The page must be added to the same container as the
// TabControl, after it. image is an index into the image list or -1.
//
//	tabs := wui.NewTabControl()
//	win.Add(tabs)
//	general, advanced := wui.NewPanel(), wui.NewPanel()
//	win.Add(general)
//	win.Add(advanced)
//	tabs.AddPage("General", general, -1)
//	tabs.AddPage("Advanced", advanced, -1)
func (t *TabControl) AddPage(title string, page Control, image int) {
	t.appendTab(title, image, page)
}

// Page returns the content control of a tab or nil.
func (t *TabControl) Page(index int) Control {
	if index < 0 || index >= len(t.pages) {
		return nil
	}
	return t.pages[index]
}

// SetBounds moves or resizes the tab control and its selected page.
func (t *TabControl) SetBounds(x, y, width, height int) {
	t.control.SetBounds(x, y, width, height)
	t.showPages()
}

func (t *TabControl) showPages() {
	for i, p := range t.pages {
		if p == nil {
			continue
		}
		v, _ := p.(interface{ SetVisible(bool) })
		if i == t.selected {
			x, y, w, h := t.ContentBounds()
			p.SetBounds(x, y, w, h)
			if v != nil {
				v.SetVisible(true)
			}
		} else if v != nil {
			v.SetVisible(false)
		}
	}
}
