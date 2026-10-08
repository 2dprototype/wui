package main

// Preview drawing of the controls that were added after the first version of
// the designer. The designer never shows the real controls, it paints a
// picture of each one, so every control type needs a function here.

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/2dprototype/wui"
)

var (
	colBorder   = wui.RGB(122, 122, 122)
	colDisabled = wui.RGB(204, 204, 204)
	colText     = wui.RGB(0, 0, 0)
	colGrayText = wui.RGB(109, 109, 109)
	colWhite    = wui.RGB(255, 255, 255)
	colFace     = wui.RGB(240, 240, 240)
	colLine     = wui.RGB(229, 229, 229)
	colAccent   = wui.RGB(0, 120, 215)
)

// drawUnknown draws a control that has no drawing function: a labeled box.
func drawUnknown(c wui.Control, d drawer) {
	x, y, w, h := c.Bounds()
	if w <= 0 || h <= 0 {
		return
	}
	d.PushDrawRegion(x, y, w, h)
	d.FillRect(x, y, w, h, wui.RGB(250, 250, 250))
	d.DrawRect(x, y, w, h, wui.RGB(160, 160, 160))
	d.TextRectFormat(x, y, w, h, typeNameOf(c), wui.FormatCenter, wui.RGB(90, 90, 90))
	d.PopDrawRegion()
}

func borderOf(enabled bool) wui.Color {
	if enabled {
		return colBorder
	}
	return colDisabled
}

func drawListView(l *wui.ListView, d drawer) {
	x, y, w, h := l.Bounds()
	if w <= 2 || h <= 2 {
		return
	}
	d.PushDrawRegion(x, y, w, h)
	defer d.PopDrawRegion()

	d.FillRect(x, y, w, h, colWhite)
	d.DrawRect(x, y, w, h, wui.RGB(130, 135, 144))

	cols := extraLines(l, "Columns")
	rows := extraLines(l, "Items")
	checks := extraBool(l, "CheckBoxes")
	grid := extraBool(l, "GridLines")
	const rowH = 17

	if l.View() == wui.ListViewDetails {
		top := y + 1
		if extraBool(l, "HeaderVisible") && len(cols) > 0 {
			d.Line(x+1, top+19, x+w-1, top+19, colLine)
			cx := x + 1
			for _, title := range cols {
				d.TextOut(cx+6, top+4, title, colText)
				d.Line(cx+99, top, cx+99, top+19, colLine)
				cx += 100
			}
			top += 20
		}
		for i, row := range rows {
			ry := top + i*rowH
			if ry+rowH > y+h {
				break
			}
			cx := x + 1
			if checks {
				d.DrawRect(cx+3, ry+2, 13, 13, wui.RGB(100, 100, 100))
				cx += 20
			}
			d.TextOut(cx+4, ry+1, row, colText)
			if grid {
				d.Line(x+1, ry+rowH-1, x+w-1, ry+rowH-1, colLine)
			}
		}
		if grid {
			cx := x + 100
			for range cols {
				if cx < x+w {
					d.Line(cx, top, cx, y+h-1, colLine)
				}
				cx += 100
			}
		}
		return
	}

	cw, ch := 110, 20
	switch l.View() {
	case wui.ListViewIcons:
		cw, ch = 80, 64
	case wui.ListViewTiles:
		cw, ch = 160, 44
	case wui.ListViewList:
		ch = rowH
	}
	perRow := max(1, (w-2)/cw)
	perCol := max(1, (h-2)/ch)
	for i, row := range rows {
		var cx, cy int
		if l.View() == wui.ListViewList {
			cx = x + 1 + (i/perCol)*cw
			cy = y + 1 + (i%perCol)*ch
		} else {
			cx = x + 1 + (i%perRow)*cw
			cy = y + 1 + (i/perRow)*ch
		}
		if cx >= x+w || cy+ch > y+h {
			continue
		}
		iconBack := wui.RGB(190, 205, 230)
		iconLine := wui.RGB(120, 140, 180)
		switch l.View() {
		case wui.ListViewIcons:
			d.FillRect(cx+(cw-32)/2, cy+2, 32, 32, iconBack)
			d.DrawRect(cx+(cw-32)/2, cy+2, 32, 32, iconLine)
			tw, _ := d.TextExtent(row)
			d.TextOut(cx+(cw-tw)/2, cy+38, row, colText)
		case wui.ListViewTiles:
			d.FillRect(cx+4, cy+4, 32, 32, iconBack)
			d.DrawRect(cx+4, cy+4, 32, 32, iconLine)
			d.TextOut(cx+42, cy+14, row, colText)
		default:
			d.FillRect(cx+2, cy+1, 16, 15, iconBack)
			d.DrawRect(cx+2, cy+1, 16, 15, iconLine)
			d.TextOut(cx+22, cy+2, row, colText)
		}
	}
}

func drawRichEdit(r *wui.RichEdit, d drawer) {
	x, y, w, h := r.Bounds()
	if w <= 2 || h <= 2 {
		return
	}
	d.PushDrawRegion(x, y, w, h)
	d.DrawRect(x, y, w, h, borderOf(r.Enabled()))
	bg := colWhite
	if cr, cg, cb, ok := extraColor(r, "BackgroundColor"); ok {
		bg = wui.RGB(cr, cg, cb)
	}
	d.FillRect(x+1, y+1, w-2, h-2, bg)
	color := colText
	if !r.Enabled() {
		color = colGrayText
	}
	if extraBool(r, "WordWrap") {
		d.TextRectFormat(x+6, y+3, w-6, h-3, r.Text(), wui.FormatTopLeft, color)
	} else {
		d.TextOut(x+6, y+3, r.Text(), color)
	}
	d.PopDrawRegion()
}

// stripLinkMarkup removes the <a ...> tags of a LinkLabel text.
func stripLinkMarkup(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>' && inTag:
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func drawLinkLabel(l *wui.LinkLabel, d drawer) {
	x, y, w, h := l.Bounds()
	if w <= 0 || h <= 0 {
		return
	}
	d.PushDrawRegion(x, y, w, h)
	text := stripLinkMarkup(l.Text())
	tw, th := d.TextExtent(text)
	blue := wui.RGB(0, 102, 204)
	d.TextOut(x, y+(h-th)/2, text, blue)
	d.Line(x, y+(h+th)/2, x+tw, y+(h+th)/2, blue)
	d.PopDrawRegion()
}

func drawMonthCalendar(m *wui.MonthCalendar, d drawer) {
	x, y, w, h := m.Bounds()
	if w <= 40 || h <= 60 {
		return
	}
	d.PushDrawRegion(x, y, w, h)
	defer d.PopDrawRegion()

	d.FillRect(x, y, w, h, colWhite)
	d.DrawRect(x, y, w, h, wui.RGB(160, 160, 160))

	date := m.Date()
	title := date.Format("January 2006")
	d.FillRect(x+1, y+1, w-2, 24, colAccent)
	tw, th := d.TextExtent(title)
	d.TextOut(x+(w-tw)/2, y+1+(24-th)/2, title, colWhite)

	weeks := extraBool(m, "ShowWeekNumbers")
	left := x + 4
	if weeks {
		left += 22
	}
	cw := (x + w - 4 - left) / 7
	if cw < 6 {
		return
	}
	for i, s := range []string{"S", "M", "T", "W", "T", "F", "S"} {
		d.TextOut(left+i*cw+cw/2-3, y+28, s, wui.RGB(90, 90, 90))
	}

	first := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
	offset := int(first.Weekday())
	total := time.Date(date.Year(), date.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	rowH := max(10, (h-48)/6)
	for day := 1; day <= total; day++ {
		idx := offset + day - 1
		cx := left + (idx%7)*cw
		cy := y + 44 + (idx/7)*rowH
		if cy+rowH > y+h {
			break
		}
		s := strconv.Itoa(day)
		dw, dh := d.TextExtent(s)
		if day == date.Day() {
			d.FillRect(cx+1, cy, cw-2, rowH-1, colAccent)
			d.TextOut(cx+(cw-dw)/2, cy+(rowH-dh)/2, s, colWhite)
		} else {
			d.TextOut(cx+(cw-dw)/2, cy+(rowH-dh)/2, s, colText)
		}
	}
	if weeks {
		for row := 0; row < 6; row++ {
			cy := y + 44 + row*rowH
			if cy+rowH > y+h {
				break
			}
			_, week := first.AddDate(0, 0, 7*row).ISOWeek()
			d.TextOut(x+6, cy+2, strconv.Itoa(week), wui.RGB(120, 120, 120))
		}
		d.Line(left-2, y+44, left-2, y+h-4, colLine)
	}
}

func drawHotKeyEdit(e *wui.HotKeyEdit, d drawer) {
	x, y, w, h := e.Bounds()
	if w <= 2 || h <= 2 {
		return
	}
	d.PushDrawRegion(x, y, w, h)
	d.DrawRect(x, y, w, h, borderOf(e.Enabled()))
	d.FillRect(x+1, y+1, w-2, h-2, colWhite)
	d.TextOut(x+6, y+3, "None", colGrayText)
	d.PopDrawRegion()
}

func drawIPAddressEdit(e *wui.IPAddressEdit, d drawer) {
	x, y, w, h := e.Bounds()
	if w <= 2 || h <= 2 {
		return
	}
	d.PushDrawRegion(x, y, w, h)
	d.DrawRect(x, y, w, h, borderOf(e.Enabled()))
	d.FillRect(x+1, y+1, w-2, h-2, colWhite)
	part := (w - 6) / 4
	for i := 0; i < 4; i++ {
		d.TextOut(x+4+i*part+part/2-3, y+3, "0", colText)
		if i < 3 {
			d.TextOut(x+4+(i+1)*part-2, y+3, ".", colText)
		}
	}
	d.PopDrawRegion()
}

func drawImageView(v *wui.ImageView, d drawer) {
	x, y, w, h := v.Bounds()
	if w <= 2 || h <= 2 {
		return
	}
	d.PushDrawRegion(x, y, w, h)
	bg := colFace
	if cr, cg, cb, ok := extraColor(v, "BackColor"); ok {
		bg = wui.RGB(cr, cg, cb)
	}
	d.FillRect(x, y, w, h, bg)
	if img := loadDesignImage(extraValue(v, "ImageFile")); img != nil {
		drawImageInView(d, img, v.Mode(), x, y, w, h)
		d.PopDrawRegion()
		return
	}
	d.DrawRect(x, y, w, h, wui.RGB(160, 160, 160))
	d.Line(x, y, x+w-1, y+h-1, wui.RGB(200, 200, 200))
	d.Line(x, y+h-1, x+w-1, y, wui.RGB(200, 200, 200))
	d.TextRectFormat(x, y, w, h, "Image", wui.FormatCenter, wui.RGB(90, 90, 90))
	d.PopDrawRegion()
}

func drawScrollPanel(s *wui.ScrollPanel, d drawer) {
	drawPanel(&s.Panel, d)
	x, y, w, h := s.Bounds()
	if w > 70 && h > 24 {
		d.PushDrawRegion(x, y, w, h)
		tw, th := d.TextExtent("scroll")
		d.TextOut(x+w-tw-4, y+h-th-2, "scroll", wui.RGB(150, 150, 150))
		d.PopDrawRegion()
	}
}

func drawScrollBar(s *wui.ScrollBar, d drawer) {
	x, y, w, h := s.Bounds()
	if w <= 4 || h <= 4 {
		return
	}
	d.PushDrawRegion(x, y, w, h)
	defer d.PopDrawRegion()

	vertical := extraBool(s, "Vertical")
	d.FillRect(x, y, w, h, wui.RGB(240, 240, 240))

	rng := extraInts(s, "Range")
	page := 10
	if p := extraInts(s, "Page"); len(p) > 0 {
		page = p[0]
	}
	total := 101
	if len(rng) >= 2 {
		total = rng[1] - rng[0] + 1
	}
	if total < 1 {
		total = 1
	}
	frac := float64(page) / float64(total)
	if frac > 1 {
		frac = 1
	}
	if frac < 0 {
		frac = 0
	}

	arrow := wui.RGB(96, 96, 96)
	thumb := wui.RGB(205, 205, 205)
	const btn = 17
	if vertical {
		track := h - 2*btn
		if track < 4 {
			return
		}
		size := max(8, int(float64(track)*frac))
		d.FillRect(x+1, y+btn, w-2, size, thumb)
		cx := x + w/2
		d.Polygon([]wui.Point{{X: int32(cx - 3), Y: int32(y + 10)}, {X: int32(cx), Y: int32(y + 6)}, {X: int32(cx + 3), Y: int32(y + 10)}}, arrow)
		d.Polygon([]wui.Point{{X: int32(cx - 3), Y: int32(y + h - 10)}, {X: int32(cx), Y: int32(y + h - 6)}, {X: int32(cx + 3), Y: int32(y + h - 10)}}, arrow)
	} else {
		track := w - 2*btn
		if track < 4 {
			return
		}
		size := max(8, int(float64(track)*frac))
		d.FillRect(x+btn, y+1, size, h-2, thumb)
		cy := y + h/2
		d.Polygon([]wui.Point{{X: int32(x + 10), Y: int32(cy - 3)}, {X: int32(x + 6), Y: int32(cy)}, {X: int32(x + 10), Y: int32(cy + 3)}}, arrow)
		d.Polygon([]wui.Point{{X: int32(x + w - 10), Y: int32(cy - 3)}, {X: int32(x + w - 6), Y: int32(cy)}, {X: int32(x + w - 10), Y: int32(cy + 3)}}, arrow)
	}
}

func drawTreeView(t *wui.TreeView, d drawer) {
	x, y, w, h := t.Bounds()
	if w <= 2 || h <= 2 {
		return
	}
	d.PushDrawRegion(x, y, w, h)
	defer d.PopDrawRegion()

	d.FillRect(x, y, w, h, colWhite)
	d.DrawRect(x, y, w, h, wui.RGB(130, 135, 144))
	checks := extraBool(t, "CheckBoxes")
	const rowH = 17
	for i, it := range parseTreeLines(extraLines(t, "Nodes")) {
		ry := y + 2 + i*rowH
		if ry+rowH > y+h {
			break
		}
		ix := x + 4 + it.depth*16
		if it.hasChildren {
			d.DrawRect(ix, ry+4, 9, 9, wui.RGB(130, 130, 130))
			d.Line(ix+2, ry+8, ix+6, ry+8, colText)
		}
		tx := ix + 14
		if checks {
			d.DrawRect(tx, ry+2, 13, 13, wui.RGB(100, 100, 100))
			tx += 18
		}
		d.TextOut(tx, ry+1, it.text, colText)
	}
}

// designImages caches the pictures of ImageViews by path. A picture that
// cannot be loaded is cached as nil.
var designImages = make(map[string]*wui.Image)

func loadDesignImage(path string) *wui.Image {
	if path == "" {
		return nil
	}
	if img, ok := designImages[path]; ok {
		return img
	}
	var result *wui.Image
	if f, err := os.Open(path); err == nil {
		if decoded, _, err := image.Decode(f); err == nil {
			result = wui.NewImage(decoded)
		}
		f.Close()
	}
	designImages[path] = result
	return result
}

// drawImageInView draws the picture the way ImageView paints it for the mode.
func drawImageInView(d drawer, img *wui.Image, mode wui.ImageMode, x, y, cw, ch int) {
	iw, ih := img.Size()
	if iw <= 0 || ih <= 0 || cw <= 0 || ch <= 0 {
		return
	}
	switch mode {
	case wui.ImageNormal:
		d.DrawImage(img, img.Bounds(), x, y)
	case wui.ImageCenter:
		d.DrawImage(img, img.Bounds(), x+(cw-iw)/2, y+(ch-ih)/2)
	case wui.ImageStretch:
		d.DrawImageScaled(img, img.Bounds(), wui.Rect(x, y, cw, ch))
	case wui.ImageFit:
		scale := float64(cw) / float64(iw)
		if s := float64(ch) / float64(ih); s < scale {
			scale = s
		}
		w, h := int(float64(iw)*scale), int(float64(ih)*scale)
		d.DrawImageScaled(img, img.Bounds(), wui.Rect(x+(cw-w)/2, y+(ch-h)/2, w, h))
	case wui.ImageFill:
		scale := float64(cw) / float64(iw)
		if s := float64(ch) / float64(ih); s > scale {
			scale = s
		}
		sw, sh := int(float64(cw)/scale), int(float64(ch)/scale)
		d.DrawImageScaled(img, wui.Rect((iw-sw)/2, (ih-sh)/2, sw, sh), wui.Rect(x, y, cw, ch))
	}
}
