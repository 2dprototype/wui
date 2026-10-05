package wui

import (
	"github.com/2dprototype/wui/w32"
)

const (
	tpmRightButton = 0x0002
	tpmReturnCmd   = 0x0100
)

// NewPopupMenu creates a context menu. Fill it with the same items as normal
// menus: NewMenuString, NewMenuSeparator and sub-menus created with NewMenu.
//
// Show it with PopupMenu.Show or attach it to a window with
// Window.SetContextMenu or to any control with SetContextMenu, then it opens on
// right click.
func NewPopupMenu() *PopupMenu {
	return &PopupMenu{}
}

type PopupMenu struct {
	items []MenuItem
}

// Add appends an item to the menu.
func (m *PopupMenu) Add(item MenuItem) *PopupMenu {
	m.items = append(m.items, item)
	return m
}

func buildPopupMenu(items []MenuItem, entries *[]*MenuString) w32.HMENU {
	h := w32.CreatePopupMenu()
	for _, item := range items {
		switch it := item.(type) {
		case *MenuString:
			*entries = append(*entries, it)
			flags := uint(w32.MF_STRING)
			if it.checked {
				flags |= w32.MF_CHECKED
			}
			w32.AppendMenu(h, flags, uintptr(len(*entries)), it.text)
		case *Menu:
			sub := buildPopupMenu(it.items, entries)
			w32.AppendMenu(h, w32.MF_POPUP, uintptr(sub), it.name)
		default:
			w32.AppendMenu(h, w32.MF_SEPARATOR, 0, "")
		}
	}
	return h
}

// ShowAtScreen opens the menu at the given screen position and runs the
// OnClick function of the chosen item. It returns after the menu is closed.
func (m *PopupMenu) ShowAtScreen(parent *Window, screenX, screenY int) {
	if parent == nil || parent.handle == 0 || len(m.items) == 0 {
		return
	}
	var entries []*MenuString
	h := buildPopupMenu(m.items, &entries)
	w32.SetForegroundWindow(parent.handle)
	cmd := w32.TrackPopupMenu(
		h, tpmReturnCmd|tpmRightButton, screenX, screenY, parent.handle, nil,
	)
	w32.DestroyMenu(h)
	if cmd > 0 && cmd <= len(entries) {
		if f := entries[cmd-1].onClick; f != nil {
			f()
		}
	}
}

// Show opens the menu at a position in the client coordinates of the window.
func (m *PopupMenu) Show(parent *Window, x, y int) {
	if parent == nil || parent.handle == 0 {
		return
	}
	sx, sy := w32.ClientToScreen(parent.handle, x, y)
	m.ShowAtScreen(parent, sx, sy)
}

// ShowAtCursor opens the menu at the current mouse position.
func (m *PopupMenu) ShowAtCursor(parent *Window) {
	if x, y, ok := w32.GetCursorPos(); ok {
		m.ShowAtScreen(parent, x, y)
	}
}

// SetContextMenu sets the menu that opens when the user right-clicks the
// window's background. Controls can have their own menu with SetContextMenu.
func (w *Window) SetContextMenu(m *PopupMenu) {
	w.contextMenu = m
}

func (w *Window) ContextMenu() *PopupMenu {
	return w.contextMenu
}

// SetContextMenu sets the menu that opens when the user right-clicks the
// control. Not all native controls (e.g. edit boxes) forward right clicks.
func (c *control) SetContextMenu(m *PopupMenu) {
	c.popup = m
}

func (c *control) ContextMenu() *PopupMenu {
	return c.popup
}

func (c *control) popupMenu() *PopupMenu {
	return c.popup
}

func (w *Window) handleContextMenu(wParam, lParam uintptr) bool {
	x := int(int16(lParam & 0xFFFF))
	y := int(int16((lParam >> 16) & 0xFFFF))
	if lParam == 0xFFFFFFFF || lParam == ^uintptr(0) {
		cx, cy, ok := w32.GetCursorPos()
		if !ok {
			return false
		}
		x, y = cx, cy
	}
	menu := w.contextMenu
	if h := uintptr(wParam); h != 0 && w32.HWND(h) != w.handle {
		if ctl := findControlByHandle(w.children, h); ctl != nil {
			if pm, ok := ctl.(interface{ popupMenu() *PopupMenu }); ok && pm.popupMenu() != nil {
				menu = pm.popupMenu()
			}
		}
	}
	if menu == nil {
		return false
	}
	menu.ShowAtScreen(w, x, y)
	return true
}
