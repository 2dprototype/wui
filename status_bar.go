package wui

import (
	"syscall"
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

// NewStatusBar creates a status bar. Place it at the bottom of a window and
// anchor it with SetAnchors(AnchorMinAndMax, AnchorMax). SetText sets the text
// of the first part.
func NewStatusBar() *StatusBar {
	return &StatusBar{}
}

type StatusBar struct {
	textControl
	parts []int
	texts map[int]string
}

var _ Control = (*StatusBar)(nil)

func (*StatusBar) canFocus() bool { return false }

func (*StatusBar) eatsTabs() bool { return false }

func (s *StatusBar) create(id int) {
	s.textControl.create(id, 0, w32.STATUS_CLASS, 0)
	s.applyParts()
	for i, t := range s.texts {
		s.setPartText(i, t)
	}
}

// SetParts splits the status bar into parts with the given widths in pixels.
// A width of -1 for the last part makes it fill the remaining space.
func (s *StatusBar) SetParts(widths ...int) {
	s.parts = append([]int(nil), widths...)
	s.applyParts()
}

func (s *StatusBar) applyParts() {
	if s.handle == 0 || len(s.parts) == 0 {
		return
	}
	edges := make([]int32, len(s.parts))
	right := 0
	for i, w := range s.parts {
		if w < 0 {
			edges[i] = -1
			continue
		}
		right += w
		edges[i] = int32(right)
	}
	w32.SendMessage(
		s.handle, w32.SB_SETPARTS_MSG, uintptr(len(edges)),
		uintptr(unsafe.Pointer(&edges[0])),
	)
}

// SetPartText sets the text of the part with the given index.
func (s *StatusBar) SetPartText(index int, text string) {
	if index < 0 {
		return
	}
	if s.texts == nil {
		s.texts = make(map[int]string)
	}
	s.texts[index] = text
	if index == 0 {
		s.text = text
	}
	s.setPartText(index, text)
}

func (s *StatusBar) setPartText(index int, text string) {
	if s.handle != 0 {
		w32.SendMessage(
			s.handle, w32.SB_SETTEXTW_MSG, uintptr(index),
			uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))),
		)
	}
}
