package wui

import "github.com/2dprototype/wui/w32"

// Timer calls a function periodically on the GUI thread. Create one with
// Window.AddTimer. Timers can be created before the window is shown.
type Timer struct {
	window   *Window
	id       uintptr
	interval int
	f        func()
	running  bool
}

// AddTimer creates and starts a timer that calls f every intervalMs
// milliseconds while the window is open.
func (w *Window) AddTimer(intervalMs int, f func()) *Timer {
	if intervalMs < 1 {
		intervalMs = 1
	}
	t := &Timer{window: w, id: uintptr(len(w.timers) + 1), interval: intervalMs, f: f}
	w.timers = append(w.timers, t)
	t.Start()
	return t
}

// Start starts (or restarts) the timer.
func (t *Timer) Start() {
	t.running = true
	if t.window.handle != 0 {
		w32.SetTimer(t.window.handle, t.id, uint(t.interval), 0)
	}
}

// Stop stops the timer. It can be started again with Start.
func (t *Timer) Stop() {
	t.running = false
	if t.window.handle != 0 {
		w32.KillTimer(t.window.handle, t.id)
	}
}

// Running returns true if the timer is active.
func (t *Timer) Running() bool { return t.running }

// SetInterval changes the interval in milliseconds.
func (t *Timer) SetInterval(ms int) {
	if ms < 1 {
		ms = 1
	}
	t.interval = ms
	if t.running {
		t.Start()
	}
}

func (w *Window) startTimers() {
	for _, t := range w.timers {
		if t.running {
			w32.SetTimer(w.handle, t.id, uint(t.interval), 0)
		}
	}
}

func (w *Window) onTimer(id uintptr) bool {
	for _, t := range w.timers {
		if t.id == id {
			if t.running && t.f != nil {
				t.f()
			}
			return true
		}
	}
	return false
}

// findControlByHandle searches children (recursively through containers) for
// the control with the given window handle.
func findControlByHandle(children []Control, h uintptr) Control {
	for _, c := range children {
		if c.Handle() == h {
			return c
		}
		if cont, ok := c.(Container); ok {
			if found := findControlByHandle(cont.Children(), h); found != nil {
				return found
			}
		}
	}
	return nil
}
