package wui

import (
	"bytes"
	"sync"
	"syscall"
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

// RichEdit is a multi line text editor with formatting: fonts, sizes, bold,
// italic, underline, colors, paragraph alignment, bullets, clickable web
// links, undo and redo, and RTF import and export. It also has everything
// TextEdit has (selection, clipboard, cue...).
//
// Formatting methods named SetSelection... change the selected text, or the
// text typed next when nothing is selected.
type RichEdit struct {
	textEditControl
	readOnly     bool
	wrap         bool
	autoURL      bool
	writesTabs   bool
	back         Color
	hasBack      bool
	onTextChange func()
	onSelection  func()
	onLink       func(url string)
}

var _ Control = (*RichEdit)(nil)

// NewRichEdit creates an empty rich text editor with word wrap and automatic
// recognition of web addresses.
func NewRichEdit() *RichEdit {
	return &RichEdit{wrap: true, autoURL: true}
}

func (*RichEdit) canFocus() bool                 { return true }
func (r *RichEdit) eatsTabs() bool               { return r.writesTabs }
func (r *RichEdit) OnTabFocus() func()           { return r.onTabFocus }
func (r *RichEdit) SetOnTabFocus(f func())       { r.onTabFocus = f }
func (r *RichEdit) closing()                     { r.Text() }
func (r *RichEdit) SetWritesTabs(on bool)        { r.writesTabs = on }

const (
	emSetBkgndColor  = 0x0443
	emSetCharFormat  = 0x0444
	emSetEventMask   = 0x0445
	emSetParaFormat  = 0x0447
	emSetTargetDev   = 0x0448
	emStreamOut      = 0x044A
	emGetTextRange   = 0x044B
	emRedo           = 0x0454
	emCanRedo        = 0x0455
	emAutoURLDetect  = 0x045B
	emSetTextEx      = 0x0461
	emGetSelText     = 0x043E
	emExLimitText    = 0x0435
	emSetReadOnly    = 0x00CF

	enmChange    = 0x00000001
	enmSelChange = 0x00080000
	enmLink      = 0x04000000

	enChange    = 0x0300
	enLink      uint32 = 0x070B
	enSelChange uint32 = 0x0702

	scfSelection = 0x0001
	scfAll       = 0x0004

	cfmBold      = 0x00000001
	cfmItalic    = 0x00000002
	cfmUnderline = 0x00000004
	cfmStrikeout = 0x00000008
	cfmColor     = 0x40000000
	cfmBackColor = 0x04000000
	cfmFace      = 0x20000000
	cfmSize      = 0x80000000

	cfeBold      = 0x00000001
	cfeItalic    = 0x00000002
	cfeUnderline = 0x00000004
	cfeStrikeout = 0x00000008

	pfmStartIndent = 0x00000001
	pfmOffset      = 0x00000004
	pfmAlignment   = 0x00000008
	pfmNumbering   = 0x00000020
)

type charFormat2 struct {
	cbSize          uint32
	dwMask          uint32
	dwEffects       uint32
	yHeight         int32
	yOffset         int32
	crTextColor     uint32
	bCharSet        byte
	bPitchAndFamily byte
	szFaceName      [32]uint16
	wWeight         uint16
	sSpacing        int16
	crBackColor     uint32
	lcid            uint32
	dwReserved      uint32
	sStyle          int16
	wKerning        uint16
	bUnderlineType  byte
	bAnimation      byte
	bRevAuthor      byte
	bUnderlineColor byte
}

type paraFormat struct {
	cbSize        uint32
	dwMask        uint32
	wNumbering    uint16
	wEffects      uint16
	dxStartIndent int32
	dxRightIndent int32
	dxOffset      int32
	wAlignment    uint16
	cTabCount     int16
	rgxTabs       [32]int32
}

var richEditOnce sync.Once
var richEditClass = "RICHEDIT50W"

func loadRichEdit() string {
	richEditOnce.Do(func() {
		r, _, _ := nxLoadLibrary.Call(uintptr(unsafe.Pointer(utf16Ptr("Msftedit.dll"))))
		if r == 0 {
			nxLoadLibrary.Call(uintptr(unsafe.Pointer(utf16Ptr("Riched20.dll"))))
			richEditClass = "RichEdit20W"
		}
	})
	return richEditClass
}

func (r *RichEdit) create(id int) {
	r.textEditControl.create(
		id, w32.WS_EX_CLIENTEDGE, loadRichEdit(),
		w32.WS_TABSTOP|w32.WS_VSCROLL|w32.WS_HSCROLL|w32.ES_MULTILINE|w32.ES_AUTOVSCROLL|
			w32.ES_AUTOHSCROLL|w32.ES_WANTRETURN|w32.ES_NOHIDESEL,
	)
	w32.SendMessage(r.handle, emExLimitText, 0, 0x10000000)
	w32.SendMessage(r.handle, emSetEventMask, 0, enmChange|enmSelChange|enmLink)
	w32.SendMessage(r.handle, emAutoURLDetect, boolToUintptr(r.autoURL), 0)
	w32.SendMessage(r.handle, emSetReadOnly, boolToUintptr(r.readOnly), 0)
	r.applyWrap()
	if r.hasBack {
		w32.SendMessage(r.handle, emSetBkgndColor, 0, uintptr(r.back))
	}
}

func (r *RichEdit) applyWrap() {
	if r.handle != 0 {
		// A target device width of 1 turns wrapping off, 0 wraps at the
		// window border.
		w32.SendMessage(r.handle, emSetTargetDev, 0, boolToUintptr(!r.wrap))
	}
}

// SetWordWrap chooses whether long lines wrap (the default) or scroll
// horizontally.
func (r *RichEdit) SetWordWrap(wrap bool) {
	r.wrap = wrap
	r.applyWrap()
}

// SetReadOnly makes the text unchangeable by the user.
func (r *RichEdit) SetReadOnly(ro bool) {
	r.readOnly = ro
	if r.handle != 0 {
		w32.SendMessage(r.handle, emSetReadOnly, boolToUintptr(ro), 0)
	}
}

// ReadOnly tells whether the user cannot change the text.
func (r *RichEdit) ReadOnly() bool { return r.readOnly }

// SetAutoDetectLinks turns the automatic recognition of web addresses on or
// off. Recognized addresses are underlined and report clicks to the function
// set with SetOnLinkClick.
func (r *RichEdit) SetAutoDetectLinks(on bool) {
	r.autoURL = on
	if r.handle != 0 {
		w32.SendMessage(r.handle, emAutoURLDetect, boolToUintptr(on), 0)
	}
}

// SetBackgroundColor sets the color behind the text.
func (r *RichEdit) SetBackgroundColor(c Color) {
	r.back, r.hasBack = c, true
	if r.handle != 0 {
		w32.SendMessage(r.handle, emSetBkgndColor, 0, uintptr(c))
	}
}

// SetOnTextChange sets the function called after the text changed.
func (r *RichEdit) SetOnTextChange(f func()) { r.onTextChange = f }

// SetOnSelectionChange sets the function called when the cursor or selection
// moved.
func (r *RichEdit) SetOnSelectionChange(f func()) { r.onSelection = f }

// SetOnLinkClick sets the function called with the address when a link is
// clicked. Without it clicks are ignored.
func (r *RichEdit) SetOnLinkClick(f func(url string)) { r.onLink = f }

func (r *RichEdit) handleNotification(cmd uintptr) {
	if cmd == enChange && r.onTextChange != nil {
		r.onTextChange()
	}
}

func (r *RichEdit) handleNotify(code uint32, lParam uintptr) bool {
	switch code {
	case enSelChange:
		if r.onSelection != nil {
			r.onSelection()
		}
		return true
	case enLink:
		// ENLINK is packed to 4 bytes, so its fields are read by offset.
		hdr := unsafe.Sizeof(w32.NMHDR{})
		ptr := unsafe.Sizeof(uintptr(0))
		msg := *(*uint32)(unsafe.Pointer(lParam + hdr))
		if msg != w32.WM_LBUTTONUP || r.onLink == nil {
			return true
		}
		rng := lParam + hdr + 4 + 2*ptr
		min := int(*(*int32)(unsafe.Pointer(rng)))
		max := int(*(*int32)(unsafe.Pointer(rng + 4)))
		if max > min {
			r.onLink(r.textRange(min, max))
		}
		return true
	}
	return false
}

func (r *RichEdit) textRange(min, max int) string {
	buf := make([]uint16, max-min+1)
	tr := struct {
		min, max int32
		text     *uint16
	}{int32(min), int32(max), &buf[0]}
	n := int(w32.SendMessage(r.handle, emGetTextRange, 0, uintptr(unsafe.Pointer(&tr))))
	if n < 0 || n > len(buf) {
		n = 0
	}
	return utf16ToString(buf[:n])
}

// SelectedText returns the selected text.
func (r *RichEdit) SelectedText() string {
	if r.handle == 0 {
		return ""
	}
	s, e := r.utf16Selection()
	if e <= s {
		return ""
	}
	buf := make([]uint16, e-s+2)
	n := int(w32.SendMessage(r.handle, emGetSelText, 0, uintptr(unsafe.Pointer(&buf[0]))))
	if n < 0 || n > len(buf) {
		n = 0
	}
	return utf16ToString(buf[:n])
}

// Redo repeats what Undo reverted.
func (r *RichEdit) Redo() bool {
	return r.handle != 0 && w32.SendMessage(r.handle, emRedo, 0, 0) != 0
}

// CanRedo tells whether Redo has something to repeat.
func (r *RichEdit) CanRedo() bool {
	return r.handle != 0 && w32.SendMessage(r.handle, emCanRedo, 0, 0) != 0
}

func (r *RichEdit) format(scope uintptr, cf *charFormat2) {
	if r.handle == 0 {
		return
	}
	cf.cbSize = uint32(unsafe.Sizeof(*cf))
	w32.SendMessage(r.handle, emSetCharFormat, scope, uintptr(unsafe.Pointer(cf)))
}

func effectFormat(mask, effect uint32, on bool) charFormat2 {
	cf := charFormat2{dwMask: mask}
	if on {
		cf.dwEffects = effect
	}
	return cf
}

// SetSelectionBold makes the selection bold or normal.
func (r *RichEdit) SetSelectionBold(on bool) {
	cf := effectFormat(cfmBold, cfeBold, on)
	r.format(scfSelection, &cf)
}

// SetSelectionItalic makes the selection italic or upright.
func (r *RichEdit) SetSelectionItalic(on bool) {
	cf := effectFormat(cfmItalic, cfeItalic, on)
	r.format(scfSelection, &cf)
}

// SetSelectionUnderline underlines the selection or removes it.
func (r *RichEdit) SetSelectionUnderline(on bool) {
	cf := effectFormat(cfmUnderline, cfeUnderline, on)
	r.format(scfSelection, &cf)
}

// SetSelectionStrikeout strikes the selection through or removes it.
func (r *RichEdit) SetSelectionStrikeout(on bool) {
	cf := effectFormat(cfmStrikeout, cfeStrikeout, on)
	r.format(scfSelection, &cf)
}

// SetSelectionColor sets the text color of the selection.
func (r *RichEdit) SetSelectionColor(c Color) {
	cf := charFormat2{dwMask: cfmColor, crTextColor: uint32(c)}
	r.format(scfSelection, &cf)
}

// SetSelectionBackColor sets the highlight color behind the selection.
func (r *RichEdit) SetSelectionBackColor(c Color) {
	cf := charFormat2{dwMask: cfmBackColor, crBackColor: uint32(c)}
	r.format(scfSelection, &cf)
}

func fontFormat(name string, points int) charFormat2 {
	var cf charFormat2
	if name != "" {
		cf.dwMask |= cfmFace
		face := utf16Slice(name)
		if len(face) > 31 {
			face = face[:31]
		}
		copy(cf.szFaceName[:], face)
	}
	if points > 0 {
		cf.dwMask |= cfmSize
		cf.yHeight = int32(points * 20) // twips
	}
	return cf
}

func utf16Slice(s string) []uint16 {
	u := stringToUTF16(s)
	return u[:len(u)-1]
}

// SetSelectionFont sets the font family and the size in points of the
// selection. An empty name or a size of 0 keeps that part.
func (r *RichEdit) SetSelectionFont(name string, points int) {
	cf := fontFormat(name, points)
	r.format(scfSelection, &cf)
}

// SetAllFont sets the font family and size in points of the whole text.
func (r *RichEdit) SetAllFont(name string, points int) {
	cf := fontFormat(name, points)
	r.format(scfAll, &cf)
}

func (r *RichEdit) paragraph(pf *paraFormat) {
	if r.handle == 0 {
		return
	}
	pf.cbSize = uint32(unsafe.Sizeof(*pf))
	w32.SendMessage(r.handle, emSetParaFormat, 0, uintptr(unsafe.Pointer(pf)))
}

// SetSelectionAlignment aligns the paragraphs of the selection.
func (r *RichEdit) SetSelectionAlignment(a EditAlign) {
	pf := paraFormat{dwMask: pfmAlignment, wAlignment: uint16(a + 1)} // PFA_LEFT 1, RIGHT 2, CENTER 3
	switch a {
	case EditAlignRight:
		pf.wAlignment = 2
	case EditAlignCenter:
		pf.wAlignment = 3
	}
	r.paragraph(&pf)
}

// SetSelectionBullets turns bullet points on or off for the selected
// paragraphs.
func (r *RichEdit) SetSelectionBullets(on bool) {
	pf := paraFormat{dwMask: pfmNumbering | pfmOffset | pfmStartIndent}
	if on {
		pf.wNumbering = 1 // PFN_BULLET
		pf.dxOffset = 360
		pf.dxStartIndent = 0
	}
	r.paragraph(&pf)
}

// SetRTF replaces the whole text with RTF formatted text. Characters outside
// ASCII must be written as \uN escapes as RTF requires.
func (r *RichEdit) SetRTF(rtf string) {
	if r.handle == 0 {
		r.text = rtf
		return
	}
	st := struct{ flags, codepage uint32 }{0, 0} // ST_DEFAULT, CP_ACP
	b := append([]byte(rtf), 0)
	w32.SendMessage(r.handle, emSetTextEx, uintptr(unsafe.Pointer(&st)), uintptr(unsafe.Pointer(&b[0])))
}

var rtfStreams = struct {
	sync.Mutex
	m    map[uintptr]*bytes.Buffer
	next uintptr
}{m: make(map[uintptr]*bytes.Buffer)}

var rtfOutCallback = syscall.NewCallback(func(cookie, pbBuff, cb, pcb uintptr) uintptr {
	rtfStreams.Lock()
	buf := rtfStreams.m[cookie]
	rtfStreams.Unlock()
	n := int(int32(cb))
	if buf != nil && n > 0 {
		buf.Write((*[1 << 28]byte)(unsafe.Pointer(pbBuff))[:n:n])
		*(*int32)(unsafe.Pointer(pcb)) = int32(n)
	}
	return 0
})

// RTF returns the whole text with its formatting as RTF.
func (r *RichEdit) RTF() string {
	if r.handle == 0 {
		return r.text
	}
	buf := new(bytes.Buffer)
	rtfStreams.Lock()
	rtfStreams.next++
	cookie := rtfStreams.next
	rtfStreams.m[cookie] = buf
	rtfStreams.Unlock()
	defer func() {
		rtfStreams.Lock()
		delete(rtfStreams.m, cookie)
		rtfStreams.Unlock()
	}()
	// EDITSTREAM is packed to 4 bytes: cookie, error (uint32), callback.
	ptr := unsafe.Sizeof(uintptr(0))
	es := make([]byte, 2*ptr+4)
	*(*uintptr)(unsafe.Pointer(&es[0])) = cookie
	*(*uintptr)(unsafe.Pointer(&es[ptr+4])) = rtfOutCallback
	w32.SendMessage(r.handle, emStreamOut, 2, uintptr(unsafe.Pointer(&es[0]))) // SF_RTF
	return buf.String()
}
