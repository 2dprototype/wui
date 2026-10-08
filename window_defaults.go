package wui

import "github.com/2dprototype/wui/w32"

// SetDefaultButton makes b the default button: it is drawn with a heavier
// frame and is clicked when the user presses Enter while no multi line text
// field, list or other button has the focus. Pass nil to remove it.
func (w *Window) SetDefaultButton(b *Button) {
	if old := w.ext.feat.defaultButton; old != nil && old != b {
		old.SetDefault(false)
	}
	w.ext.feat.defaultButton = b
	if b != nil {
		b.SetDefault(true)
	}
}

// SetCancelButton makes b the button that is clicked when the user presses
// Escape. Pass nil to remove it.
func (w *Window) SetCancelButton(b *Button) {
	w.ext.feat.cancelButton = b
}

// SetTabOrder sets the order in which the Tab key moves the focus. Controls
// that are not listed are skipped by Tab. Call it without arguments to go
// back to the order the controls were added in.
func (w *Window) SetTabOrder(controls ...Control) {
	w.ext.feat.tabOrder = append([]Control(nil), controls...)
}

func (w *Window) tabList() []Control {
	if len(w.ext.feat.tabOrder) > 0 {
		return w.ext.feat.tabOrder
	}
	return w.controls
}

func clickButton(b *Button) bool {
	if b == nil || b.onClick == nil || b.Parent() == nil || !Visible(b) || !Enabled(b) {
		return false
	}
	b.onClick()
	return true
}

// handleDefaultKeys implements the Enter and Escape keys for the default and
// cancel buttons. It returns true if the key was used.
func (w *Window) handleDefaultKeys(vk uintptr) bool {
	focus := w32.GetFocus()
	fc := findControlByHandle(w.children, uintptr(focus))
	if fc == nil && focus != 0 && focus != w.handle {
		// The focus is in something inside a control, e.g. the edit box of
		// a combo box or a label being edited. It needs the key itself.
		return false
	}
	if vk == w32.VK_RETURN {
		switch c := fc.(type) {
		case *TextEdit, *RichEdit, *ListView, *StringList, *StringTable, *TreeView, *ComboBox, *LinkLabel:
			return false
		case *Button:
			return clickButton(c)
		}
		return clickButton(w.ext.feat.defaultButton)
	}
	if _, ok := fc.(*ComboBox); ok {
		return false
	}
	return clickButton(w.ext.feat.cancelButton)
}
