package wui

import (
	"image"
	"image/color"
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

const (
	cfDIB   = 8
	cfHDROP = 15
)

// ClipboardFiles returns the paths of the files that were copied in the file
// explorer (Ctrl+C on files). ok is false if the clipboard holds no files.
func ClipboardFiles() (files []string, ok bool) {
	if !w32.IsClipboardFormatAvailable(cfHDROP) || !w32.OpenClipboard(0) {
		return nil, false
	}
	defer w32.CloseClipboard()
	h := w32.GetClipboardData(cfHDROP)
	if h == 0 {
		return nil, false
	}
	n := dropFileCount(w32.HDROP(h))
	for i := 0; i < n; i++ {
		size, _, _ := procDragQueryFile.Call(uintptr(h), uintptr(i), 0, 0)
		buf := make([]uint16, int(size)+1)
		procDragQueryFile.Call(uintptr(h), uintptr(i), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		files = append(files, utf16Z(buf))
	}
	return files, len(files) > 0
}

// ClipboardImage returns the picture on the clipboard (24 and 32 bit bitmaps,
// as put there by screenshots and most programs). ok is false if there is
// none.
func ClipboardImage() (img image.Image, ok bool) {
	if !w32.IsClipboardFormatAvailable(cfDIB) || !w32.OpenClipboard(0) {
		return nil, false
	}
	defer w32.CloseClipboard()
	h := w32.GetClipboardData(cfDIB)
	if h == 0 {
		return nil, false
	}
	p := w32.GlobalLock(w32.HGLOBAL(h))
	if p == nil {
		return nil, false
	}
	defer w32.GlobalUnlock(w32.HGLOBAL(h))
	total, _, _ := nxGlobalSize.Call(uintptr(h))
	data := (*[1 << 30]byte)(p)[:int(total):int(total)]
	if len(data) < 40 {
		return nil, false
	}
	le32 := func(o int) int32 {
		return int32(uint32(data[o]) | uint32(data[o+1])<<8 | uint32(data[o+2])<<16 | uint32(data[o+3])<<24)
	}
	headerSize := int(le32(0))
	width, height := int(le32(4)), int(le32(8))
	bits := int(uint16(data[14]) | uint16(data[15])<<8)
	compression := int(le32(16))
	if width <= 0 || height == 0 || (bits != 24 && bits != 32) || (compression != 0 && compression != 3) {
		return nil, false
	}
	topDown := height < 0
	if topDown {
		height = -height
	}
	offset := headerSize
	if compression == 3 && headerSize == 40 {
		offset += 12 // the three color masks
	}
	stride := ((width*bits + 31) / 32) * 4
	if offset+stride*height > len(data) {
		return nil, false
	}
	out := image.NewNRGBA(image.Rect(0, 0, width, height))
	hasAlpha := false
	if bits == 32 {
		for y := 0; y < height && !hasAlpha; y++ {
			row := data[offset+y*stride:]
			for x := 0; x < width; x++ {
				if row[x*4+3] != 0 {
					hasAlpha = true
					break
				}
			}
		}
	}
	for y := 0; y < height; y++ {
		sy := y
		if !topDown {
			sy = height - 1 - y
		}
		row := data[offset+sy*stride:]
		for x := 0; x < width; x++ {
			px := row[x*(bits/8):]
			a := uint8(255)
			if bits == 32 && hasAlpha {
				a = px[3]
			}
			out.SetNRGBA(x, y, color.NRGBA{R: px[2], G: px[1], B: px[0], A: a})
		}
	}
	return out, true
}

// SetClipboardImage replaces the clipboard contents with the picture, as a
// 32 bit bitmap.
func SetClipboardImage(img image.Image) bool {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return false
	}
	size := 40 + w*h*4
	if !w32.OpenClipboard(0) {
		return false
	}
	defer w32.CloseClipboard()
	w32.EmptyClipboard()
	mem := w32.GlobalAlloc(w32.GMEM_MOVEABLE, uint32(size))
	if mem == 0 {
		return false
	}
	p := w32.GlobalLock(mem)
	if p == nil {
		w32.GlobalFree(mem)
		return false
	}
	data := (*[1 << 30]byte)(p)[:size:size]
	put32 := func(o int, v uint32) {
		data[o], data[o+1], data[o+2], data[o+3] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
	}
	for i := 0; i < 40; i++ {
		data[i] = 0
	}
	put32(0, 40)
	put32(4, uint32(w))
	put32(8, uint32(h)) // positive: rows run bottom to top
	data[12], data[14] = 1, 32
	put32(20, uint32(w*h*4))
	for y := 0; y < h; y++ {
		dst := data[40+(h-1-y)*w*4:]
		for x := 0; x < w; x++ {
			c := color.NRGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			dst[x*4], dst[x*4+1], dst[x*4+2], dst[x*4+3] = c.B, c.G, c.R, c.A
		}
	}
	w32.GlobalUnlock(mem)
	if w32.SetClipboardData(cfDIB, w32.HANDLE(mem)) == 0 {
		w32.GlobalFree(mem)
		return false
	}
	return true
}
