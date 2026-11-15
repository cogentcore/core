// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"github.com/stretchr/testify/assert"
)

// datePickerDays returns the day buttons of the given date picker.
func datePickerDays(dp *DatePicker) []*Button {
	var days []*Button
	for _, c := range dp.Child(1).AsTree().Children {
		days = append(days, c.(*Button))
	}
	return days
}

// TestDatePickerGrid tests that the grid of days of a date picker is
// whole weeks, from the start of the week of the first day of the month
// to the end of the week of the last, including months whose weeks cross
// into another year.
func TestDatePickerGrid(t *testing.T) {
	for _, tm := range []time.Time{
		time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC),  // starts the week before, in 2020
		time.Date(2021, 6, 15, 0, 0, 0, 0, time.UTC), // within one year
		time.Date(2021, 12, 4, 0, 0, 0, 0, time.UTC), // ends the week after, in 2022
		time.Date(2024, 12, 4, 0, 0, 0, 0, time.UTC), // a leap year
		time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),  // exactly four weeks
	} {
		b := NewBody()
		dp := NewDatePicker(b).SetTime(tm)
		RenderImage(b, Locked(b, func() {
			days := datePickerDays(dp)
			n := len(days)
			assert.Equal(t, 0, n%7, "%v: %d days is not whole weeks", tm, n)
			first, last := dp.getTime(0), dp.getTime(n-1)
			assert.Equal(t, time.Sunday, first.Weekday(), "%v", tm)
			assert.Equal(t, time.Saturday, last.Weekday(), "%v", tm)
			assert.False(t, first.After(time.Date(tm.Year(), tm.Month(), 1, 0, 0, 0, 0, time.UTC)), "%v", tm)
			assert.Equal(t, tm.Month(), last.AddDate(0, 0, -6).Month(), "%v: the last week has a day of the month", tm)
		}))
	}
}

// TestDatePickerNavigate tests changing the month and year of a date
// picker with its buttons, and picking a day.
func TestDatePickerNavigate(t *testing.T) {
	b := NewBody()
	dp := NewDatePicker(b).SetTime(time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC))
	changes := 0
	dp.OnChange(func(e events.Event) { changes++ })
	AssertRender(t, b, "date-picker/navigate", Locked(b, func() {
		bar := dp.Child(0).AsTree()
		prevMonth, nextMonth := bar.Child(0).(*Button), bar.Child(2).(*Button)
		prevYear, nextYear := bar.Child(3).(*Button), bar.Child(5).(*Button)
		nextMonth.Send(events.Click)
		assert.Equal(t, time.March, dp.Time.Month(), "Jan 31 + 1 month normalizes")
		prevMonth.Send(events.Click)
		assert.Equal(t, time.February, dp.Time.Month())
		nextYear.Send(events.Click)
		assert.Equal(t, 2022, dp.Time.Year())
		prevYear.Send(events.Click)
		prevYear.Send(events.Click)
		assert.Equal(t, 2020, dp.Time.Year())

		dp.SetTime(time.Date(2021, 6, 15, 0, 0, 0, 0, time.UTC)).Update()
		days := datePickerDays(dp)
		days[len(days)-1].Send(events.Click)
		assert.Equal(t, time.Date(2021, 7, 3, 0, 0, 0, 0, time.UTC), dp.Time, "the last day shown")
		assert.Equal(t, 6, changes)
	}))
}

// TestTimePicker tests changing the hour, minute, and AM/PM of a time
// picker, with a 12 hour clock.
func TestTimePicker(t *testing.T) {
	clock24 := SystemSettings.Clock24
	SystemSettings.Clock24 = false
	defer func() { SystemSettings.Clock24 = clock24 }()

	b := NewBody()
	tp := NewTimePicker(b).SetTime(time.Date(2021, 1, 4, 11, 16, 0, 0, time.UTC))
	AssertRender(t, b, "time-picker/basic", Locked(b, func() {
		hour, minute := tp.Child(0).(*Spinner), tp.Child(2).(*Spinner)
		ampm := tp.Child(3).(*Switches)
		assert.Equal(t, float32(11), hour.Value)

		hour.SetValue(12).SendChange()
		assert.Equal(t, 0, tp.Time.Hour(), "12 AM is midnight")
		minute.SetValue(45).SendChange()
		assert.Equal(t, 45, tp.Time.Minute())

		ampm.SelectValue("PM")
		ampm.SendChange()
		assert.Equal(t, 12, tp.Time.Hour(), "12 PM is noon")
		hour.SetValue(3).SendChange()
		assert.Equal(t, 15, tp.Time.Hour())
		ampm.SelectValue("AM")
		ampm.SendChange()
		assert.Equal(t, 3, tp.Time.Hour())
		assert.Equal(t, 4, tp.Time.Day(), "the date is kept")
	}))
}

// TestKeyChordButton tests that a key chord button records the key
// chord typed while it has the focus, and clears it from its menu.
func TestKeyChordButton(t *testing.T) {
	b := NewBody()
	kc := NewKeyChordButton(b)
	AssertRender(t, b, "key-chord-button/basic", Locked(b, func() {
		SendKey(kc, 'a', key.CodeA, key.Control)
		assert.Equal(t, "", string(kc.Chord), "not recorded without the focus")
		kc.SetFocus()
		SendKey(kc, 'a', key.CodeA, key.Control)
		assert.Equal(t, "Control+A", string(kc.Chord))
		assert.Equal(t, kc.Chord.Label(), kc.Text)

		m := NewScene()
		for _, f := range kc.ContextMenus {
			f(m, image.Point{})
		}
		clear := m.Child(0).(*Button)
		assert.Equal(t, "Clear", clear.Text)
		clear.Send(events.Click)
		assert.Equal(t, "", string(kc.Chord))
	}))
}

// TestValueButtons tests the buttons that represent values that are
// edited in dialogs, in a form.
func TestValueButtons(t *testing.T) {
	type values struct {
		KeyMap       keymap.MapName
		Chord        key.Chord
		Font         FontName
		Highlighting HighlightingName
		Image        image.Image
	}
	v := &values{KeyMap: "MacStandard", Chord: "Control+S", Highlighting: "emacs",
		Image: image.NewRGBA(image.Rect(0, 0, 3, 2))}
	b := NewBody()
	NewForm(b).SetStruct(v)
	AssertRender(t, b, "form/value-buttons")
}
