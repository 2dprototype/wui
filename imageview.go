package wui

import (
	"image"
	_ "image/gif"  // register decoders for SetImageFromFile
	_ "image/jpeg" //
	_ "image/png"  //
	"os"
)

// ImageMode says how an ImageView fits its image into the available space.
type ImageMode int

const (
	// ImageNormal draws the image in its original size at the top left.
	ImageNormal ImageMode = iota
	// ImageCenter draws the image in its original size in the middle.
	ImageCenter
	// ImageStretch stretches the image to fill the view, ignoring the aspect
	// ratio.
	ImageStretch
	// ImageFit scales the image to the largest size that fits completely,
	// keeping the aspect ratio.
	ImageFit
	// ImageFill scales the image to cover the whole view, keeping the aspect
	// ratio and cutting what does not fit.
	ImageFill
)

// ImageView shows a picture (PNG, JPEG, GIF or any Go image.Image).
type ImageView struct {
	*PaintBox
	img     *Image
	mode    ImageMode
	back    Color
	hasBack bool
}

var _ Control = (*ImageView)(nil)

// NewImageView creates an empty image view.
func NewImageView() *ImageView {
	v := &ImageView{PaintBox: NewPaintBox(), mode: ImageFit}
	v.PaintBox.SetOnPaint(v.paint)
	return v
}

// SetImage shows a Go image, nil clears the view.
func (v *ImageView) SetImage(img image.Image) {
	if img == nil {
		v.img = nil
	} else {
		v.img = NewImage(img)
	}
	v.Paint()
}

// SetImageFromFile loads a PNG, JPEG or GIF file and shows it.
func (v *ImageView) SetImageFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}
	v.SetImage(img)
	return nil
}

// Image returns the displayed image or nil.
func (v *ImageView) Image() *Image { return v.img }

// SetMode chooses how the image is fitted, ImageFit by default.
func (v *ImageView) SetMode(m ImageMode) {
	v.mode = m
	v.Paint()
}

// Mode returns the fit mode.
func (v *ImageView) Mode() ImageMode { return v.mode }

// SetBackColor sets the color behind (and around) the image. Without it the
// view is filled with the standard control background.
func (v *ImageView) SetBackColor(c Color) {
	v.back, v.hasBack = c, true
	v.Paint()
}

func (v *ImageView) paint(c *Canvas) {
	cw, ch := c.Size()
	if v.hasBack {
		c.Clear(v.back)
	} else {
		c.Clear(ColorButtonFace)
	}
	if v.img == nil || cw <= 0 || ch <= 0 {
		return
	}
	iw, ih := v.img.Size()
	if iw <= 0 || ih <= 0 {
		return
	}
	switch v.mode {
	case ImageNormal:
		c.DrawImage(v.img, v.img.Bounds(), 0, 0)
	case ImageCenter:
		c.DrawImage(v.img, v.img.Bounds(), (cw-iw)/2, (ch-ih)/2)
	case ImageStretch:
		c.DrawImageScaled(v.img, v.img.Bounds(), Rect(0, 0, cw, ch))
	case ImageFit:
		scale := float64(cw) / float64(iw)
		if s := float64(ch) / float64(ih); s < scale {
			scale = s
		}
		w, h := int(float64(iw)*scale), int(float64(ih)*scale)
		c.DrawImageScaled(v.img, v.img.Bounds(), Rect((cw-w)/2, (ch-h)/2, w, h))
	case ImageFill:
		scale := float64(cw) / float64(iw)
		if s := float64(ch) / float64(ih); s > scale {
			scale = s
		}
		// The part of the source that is visible.
		sw, sh := int(float64(cw)/scale), int(float64(ch)/scale)
		c.DrawImageScaled(v.img, Rect((iw-sw)/2, (ih-sh)/2, sw, sh), Rect(0, 0, cw, ch))
	}
}
