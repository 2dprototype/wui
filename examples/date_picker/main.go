// DatePicker example: a date picker and a time picker.
package main

import (
	"time"

	"github.com/2dprototype/wui"
)

func main() {
	w := wui.NewWindow()
	w.SetTitle("DatePicker")
	w.SetInnerSize(300, 140)
	w.SetCenterOnShow(true)

	result := wui.NewLabel()
	result.SetBounds(10, 90, 280, 25)

	date := wui.NewDatePicker()
	date.SetBounds(10, 10, 280, 25)
	date.SetOnChange(func(t time.Time) {
		result.SetText("Date: " + t.Format("2006-01-02"))
	})
	w.Add(date)

	clock := wui.NewDatePicker()
	clock.SetMode(wui.DatePickerTime)
	clock.SetBounds(10, 50, 280, 25)
	clock.SetOnChange(func(t time.Time) {
		result.SetText("Time: " + t.Format("15:04:05"))
	})
	w.Add(clock)

	w.Add(result)
	w.Show()
}
