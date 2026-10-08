package wui

import (
	"sync"
	"sync/atomic"

	"github.com/2dprototype/wui/w32"
)

// wmInvoke is the private message that wakes the GUI thread up to run the
// functions queued by Invoke.
const wmInvoke = w32.WM_APP + 0x31

type invokeState struct {
	mu   sync.Mutex
	fns  []func()
	hwnd uintptr // atomic, 0 while the window does not exist
	tid  uint32  // atomic, the thread running the window's message loop
}

// Invoke queues f to run on the window's GUI thread and returns immediately.
// It is safe to call from any goroutine and is the way to update controls
// from background work. If the window is not shown yet, f runs as soon as it
// is. The functions run in the order they were queued.
func (w *Window) Invoke(f func()) {
	if f == nil {
		return
	}
	w.ext.inv.mu.Lock()
	w.ext.inv.fns = append(w.ext.inv.fns, f)
	w.ext.inv.mu.Unlock()
	if h := atomic.LoadUintptr(&w.ext.inv.hwnd); h != 0 {
		w32.PostMessage(w32.HWND(h), wmInvoke, 0, 0)
	}
}

// InvokeSync runs f on the GUI thread and waits until it is done. When it is
// called from the GUI thread itself, or when the window does not exist (not
// shown yet, or already closed), f runs directly so this can never deadlock
// in those cases.
func (w *Window) InvokeSync(f func()) {
	if f == nil {
		return
	}
	if atomic.LoadUintptr(&w.ext.inv.hwnd) == 0 || w.OnGUIThread() {
		f()
		return
	}
	done := make(chan struct{})
	w.Invoke(func() {
		defer close(done)
		f()
	})
	<-done
}

// OnGUIThread tells whether the caller runs on the window's GUI thread.
func (w *Window) OnGUIThread() bool {
	tid := atomic.LoadUint32(&w.ext.inv.tid)
	return tid != 0 && tid == currentThreadID()
}

func (w *Window) drainInvoked() {
	for {
		w.ext.inv.mu.Lock()
		fns := w.ext.inv.fns
		w.ext.inv.fns = nil
		w.ext.inv.mu.Unlock()
		if len(fns) == 0 {
			return
		}
		for _, f := range fns {
			f()
		}
	}
}

func (w *Window) invokeStarted() {
	atomic.StoreUint32(&w.ext.inv.tid, currentThreadID())
	atomic.StoreUintptr(&w.ext.inv.hwnd, uintptr(w.handle))
	w.ext.inv.mu.Lock()
	pending := len(w.ext.inv.fns) > 0
	w.ext.inv.mu.Unlock()
	if pending {
		w32.PostMessage(w.handle, wmInvoke, 0, 0)
	}
}

func (w *Window) invokeStopped() {
	atomic.StoreUintptr(&w.ext.inv.hwnd, 0)
	atomic.StoreUint32(&w.ext.inv.tid, 0)
}
