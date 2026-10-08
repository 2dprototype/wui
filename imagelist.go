package wui

import (
	"image"
	"image/color"
	"image/draw"
)

// ImageList is a collection of equally sized images that controls (ListView,
// TreeView, TabControl, NativeToolBar) use for their icons. Create one, add
// images, and give it to a control with its SetImages or SetSmallImages method.
// An image is referred to by the index AddImage and AddIcon returned.
type ImageList struct {
	handle        uintptr
	width, height int
}

const (
	ilcMask    = 0x0001
	ilcColor32 = 0x0020
)

// NewImageList creates an empty list for images of the given size in pixels.
// Use 16x16 for small icons and 32x32 for large ones.
func NewImageList(width, height int) *ImageList {
	l := &ImageList{width: width, height: height}
	initCommonControls()
	r, _, _ := nxImageListCreate.Call(
		uintptr(width), uintptr(height), ilcColor32|ilcMask, 8, 8,
	)
	l.handle = r
	return l
}

// Handle returns the HIMAGELIST.
func (l *ImageList) Handle() uintptr { return l.handle }

// Size returns the size of each image.
func (l *ImageList) Size() (width, height int) { return l.width, l.height }

// Count returns the number of images in the list.
func (l *ImageList) Count() int {
	if l.handle == 0 {
		return 0
	}
	r, _, _ := nxImageListGetImageCount.Call(l.handle)
	return int(r)
}

// AddIcon appends an icon and returns its index, or -1 on failure. The icon is
// copied into the list.
func (l *ImageList) AddIcon(icon *Icon) int {
	if l.handle == 0 || icon == nil {
		return -1
	}
	r, _, _ := nxImageListReplaceIcon.Call(l.handle, ^uintptr(0), uintptr(icon.handle))
	return int(int32(r))
}

// AddImage appends a Go image, scaled to the size of the list if it has a
// different size, and returns its index. Transparency is kept.
func (l *ImageList) AddImage(img image.Image) (int, error) {
	icon, err := NewIconFromImage(scaleImage(img, l.width, l.height))
	if err != nil {
		return -1, err
	}
	return l.AddIcon(icon), nil
}

// Destroy frees the list. Do not use it, or controls showing it, afterwards.
func (l *ImageList) Destroy() {
	if l.handle != 0 {
		nxImageListDestroy.Call(l.handle)
		l.handle = 0
	}
}

// scaleImage returns img resized to w by h with bilinear filtering, or img
// itself as an NRGBA if it already has that size.
func scaleImage(img image.Image, w, h int) image.Image {
	b := img.Bounds()
	if b.Dx() == w && b.Dy() == h {
		out := image.NewNRGBA(image.Rect(0, 0, w, h))
		draw.Draw(out, out.Bounds(), img, b.Min, draw.Src)
		return out
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	if b.Dx() == 0 || b.Dy() == 0 || w <= 0 || h <= 0 {
		return out
	}
	sx := float64(b.Dx()) / float64(w)
	sy := float64(b.Dy()) / float64(h)
	for y := 0; y < h; y++ {
		fy := (float64(y)+0.5)*sy - 0.5
		y0 := int(fy)
		if fy < 0 {
			fy, y0 = 0, 0
		}
		ty := fy - float64(y0)
		y1 := y0 + 1
		if y1 >= b.Dy() {
			y1 = b.Dy() - 1
		}
		for x := 0; x < w; x++ {
			fx := (float64(x)+0.5)*sx - 0.5
			x0 := int(fx)
			if fx < 0 {
				fx, x0 = 0, 0
			}
			tx := fx - float64(x0)
			x1 := x0 + 1
			if x1 >= b.Dx() {
				x1 = b.Dx() - 1
			}
			c00 := nrgbaAt(img, b.Min.X+x0, b.Min.Y+y0)
			c10 := nrgbaAt(img, b.Min.X+x1, b.Min.Y+y0)
			c01 := nrgbaAt(img, b.Min.X+x0, b.Min.Y+y1)
			c11 := nrgbaAt(img, b.Min.X+x1, b.Min.Y+y1)
			mix := func(a, b, c, d float64) uint8 {
				top := a*(1-tx) + b*tx
				bot := c*(1-tx) + d*tx
				return uint8(top*(1-ty) + bot*ty + 0.5)
			}
			out.SetNRGBA(x, y, color.NRGBA{
				R: mix(float64(c00.R), float64(c10.R), float64(c01.R), float64(c11.R)),
				G: mix(float64(c00.G), float64(c10.G), float64(c01.G), float64(c11.G)),
				B: mix(float64(c00.B), float64(c10.B), float64(c01.B), float64(c11.B)),
				A: mix(float64(c00.A), float64(c10.A), float64(c01.A), float64(c11.A)),
			})
		}
	}
	return out
}

func nrgbaAt(img image.Image, x, y int) color.NRGBA {
	return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
}
