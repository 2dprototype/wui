package wui

import (
	"syscall"

	"github.com/2dprototype/wui/w32"
)

// DropFilesFunc is called when the user drops files from the file explorer
// onto a window or control. x and y are relative to the top left corner of the
// window's (or control's) client area.
type DropFilesFunc func(files []string, x, y int)

var procDragQueryFile = syscall.NewLazyDLL("shell32.dll").NewProc("DragQueryFileW")

func dropFileCount(h w32.HDROP) int {
	n, _, _ := procDragQueryFile.Call(uintptr(h), 0xFFFFFFFF, 0, 0)
	return int(uint32(n))
}

// SetOnDropFiles makes the window accept files dragged from the file explorer.
// The handler receives the dropped file paths. Dropping onto a control that has
// its own handler calls the control's handler instead.
func (w *Window) SetOnDropFiles(f DropFilesFunc) {
	w.ext.onDropFiles = f
	w.applyDropTarget()
}

// OnDropFiles returns the handler set with SetOnDropFiles.
func (w *Window) OnDropFiles() DropFilesFunc {
	return w.ext.onDropFiles
}

// SetAcceptFiles enables or disables drag and drop of files onto the window.
// It is implied by SetOnDropFiles.
func (w *Window) SetAcceptFiles(accept bool) {
	w.ext.acceptFiles = accept
	w.applyDropTarget()
}

// AcceptFiles returns true if the window accepts dropped files.
func (w *Window) AcceptFiles() bool {
	return w.ext.acceptFiles
}

type dropHandler interface {
	OnDropFiles() DropFilesFunc
}

func hasDropHandlers(children []Control) bool {
	for _, c := range children {
		if h, ok := c.(dropHandler); ok && h.OnDropFiles() != nil {
			return true
		}
		if cont, ok := c.(Container); ok && hasDropHandlers(cont.Children()) {
			return true
		}
	}
	return false
}

func (w *Window) applyDropTarget() {
	if w.handle == 0 {
		return
	}
	accept := w.ext.acceptFiles || w.ext.onDropFiles != nil || hasDropHandlers(w.children)
	w32.DragAcceptFiles(w.handle, accept)
}

// SetOnDropFiles makes the control a drop target for files dragged from the
// file explorer.
func (c *control) SetOnDropFiles(f DropFilesFunc) {
	c.onDropFiles = f
	if f != nil {
		var cont Container = c.parent
		for cont != nil {
			if win, ok := cont.(*Window); ok {
				win.applyDropTarget()
				return
			}
			cont = cont.Parent()
		}
	}
}

// OnDropFiles returns the handler set with SetOnDropFiles.
func (c *control) OnDropFiles() DropFilesFunc {
	return c.onDropFiles
}

// dropTarget finds the handler that is responsible for the client position.
func (w *Window) dropTarget(x, y int) (f DropFilesFunc, rx, ry int) {
	f, rx, ry = w.ext.onDropFiles, x, y
	var walk func(c Container, x, y int)
	walk = func(c Container, x, y int) {
		children := c.Children()
		for i := len(children) - 1; i >= 0; i-- {
			ch := children[i]
			if !ch.Visible() {
				continue
			}
			bx, by, bw, bh := ch.Bounds()
			if x < bx || y < by || x >= bx+bw || y >= by+bh {
				continue
			}
			if h, ok := ch.(dropHandler); ok && h.OnDropFiles() != nil {
				f, rx, ry = h.OnDropFiles(), x-bx, y-by
			}
			if sub, ok := ch.(Container); ok {
				ix, iy, _, _ := sub.InnerBounds()
				walk(sub, x-ix, y-iy)
			}
			return
		}
	}
	walk(w, x, y)
	return
}

func (w *Window) onWM_DROPFILES(wParam uintptr) {
	drop := w32.HDROP(wParam)
	defer w32.DragFinish(drop)
	x, y, _ := w32.DragQueryPoint(drop)
	n := dropFileCount(drop)
	files := make([]string, 0, n)
	for i := 0; i < n; i++ {
		files = append(files, w32.DragQueryFile(drop, uint(i)))
	}
	if f, rx, ry := w.dropTarget(x, y); f != nil {
		f(files, rx, ry)
	}
}
