package wui

import (
	"time"
	"unsafe"

	"github.com/2dprototype/wui/w32"
)

// DatePickerMode selects what a DatePicker shows.
type DatePickerMode int

const (
	DatePickerShortDate DatePickerMode = iota
	DatePickerLongDate
	DatePickerTime
)

func (m DatePickerMode) String() string {
	// NOTE that these strings are used in the designer to get their
	// representations as Go code so they must always correspond to their
	// constant names and be prefixed with the package name.
	switch m {
	case DatePickerShortDate:
		return "wui.DatePickerShortDate"
	case DatePickerLongDate:
		return "wui.DatePickerLongDate"
	case DatePickerTime:
		return "wui.DatePickerTime"
	default:
		return "unknown DatePickerMode"
	}
}

func NewDatePicker() *DatePicker {
	return &DatePicker{value: time.Now()}
}

// DatePicker is a date (or time) chooser with a drop-down calendar.
type DatePicker struct {
	control
	mode     DatePickerMode
	value    time.Time
	onChange func(time.Time)
}

var _ Control = (*DatePicker)(nil)

func (*DatePicker) canFocus() bool { return true }

func (*DatePicker) eatsTabs() bool { return false }

func (d *DatePicker) create(id int) {
	var style uint = w32.WS_TABSTOP
	switch d.mode {
	case DatePickerLongDate:
		style |= w32.DTS_LONGDATEFORMAT
	case DatePickerTime:
		style |= w32.DTS_TIMEFORMAT
	default:
		style |= w32.DTS_SHORTDATEFORMAT
	}
	d.control.create(id, 0, w32.DATETIMEPICK_CLASS, style)
	d.pushValue()
}

func (d *DatePicker) pushValue() {
	if d.handle == 0 {
		return
	}
	st := w32.SYSTEMTIME{
		Year:   uint16(d.value.Year()),
		Month:  uint16(d.value.Month()),
		Day:    uint16(d.value.Day()),
		Hour:   uint16(d.value.Hour()),
		Minute: uint16(d.value.Minute()),
		Second: uint16(d.value.Second()),
	}
	w32.SendMessage(d.handle, w32.DTM_SETSYSTEMTIME_MSG, w32.GDT_VALID, uintptr(unsafe.Pointer(&st)))
}

// Value returns the selected date and time (in the local time zone).
func (d *DatePicker) Value() time.Time {
	if d.handle != 0 {
		var st w32.SYSTEMTIME
		w32.SendMessage(d.handle, w32.DTM_GETSYSTEMTIME_MSG, 0, uintptr(unsafe.Pointer(&st)))
		d.value = time.Date(
			int(st.Year), time.Month(st.Month), int(st.Day),
			int(st.Hour), int(st.Minute), int(st.Second), 0, time.Local,
		)
	}
	return d.value
}

func (d *DatePicker) SetValue(t time.Time) {
	d.value = t
	d.pushValue()
}

func (d *DatePicker) Mode() DatePickerMode { return d.mode }

// SetMode must be called before the window is shown.
func (d *DatePicker) SetMode(m DatePickerMode) { d.mode = m }

func (d *DatePicker) SetOnChange(f func(time.Time)) { d.onChange = f }

func (d *DatePicker) OnChange() func(time.Time) { return d.onChange }

func (d *DatePicker) notify() {
	if d.onChange != nil {
		d.onChange(d.Value())
	}
}
