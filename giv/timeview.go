// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"log/slog"
	"strconv"
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/fi"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// TimeView is a view for selecting a time
type TimeView struct {
	gi.Frame

	// the time that we are viewing
	Time time.Time `set:"-"`

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave Value `json:"-" xml:"-"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string

	// the raw input hour
	Hour int `set:"-"`

	// whether we are in PM mode (so we have to add 12h to everything)
	PM bool `set:"-"`
}

// SetTime sets the source time and updates the view
func (tv *TimeView) SetTime(tim time.Time) *TimeView {
	tv.Time = tim
	if tv.TmpSave != nil {
		tv.TmpSave.SetValue(tv.Time)
	}
	tv.SendChange()
	tv.NeedsLayout()
	return tv
}

func (tv *TimeView) Config() {
	if tv.HasChildren() {
		return
	}

	hour := gi.NewTextField(tv, "hour")
	if gi.SystemSettings.Clock24 {
		tv.Hour = tv.Time.Hour()
		hour.SetText(strconv.Itoa(tv.Hour))
	} else {
		tv.Hour = tv.Time.Hour() % 12
		hour.SetText(strconv.Itoa(tv.Hour))
	}
	hour.Style(func(s *styles.Style) {
		s.Font.Size.Dp(57)
		s.Min.X.Dp(96)
	})
	hour.OnChange(func(e events.Event) {
		hr, err := strconv.Atoi(hour.Text())
		// TODO(kai/snack)
		if err != nil {
			slog.Error(err.Error())
		}
		tv.Hour = hr
		if tv.PM {
			// only add to local
			hr += 12
		}
		// we set our hour and keep everything else
		tt := tv.Time
		tv.Time = time.Date(tt.Year(), tt.Month(), tt.Day(), hr, tt.Minute(), tt.Second(), tt.Nanosecond(), tt.Location())
		if tv.TmpSave != nil {
			tv.TmpSave.SetValue(tv.Time)
		}
		tv.SendChange()
	})

	gi.NewLabel(tv, "colon").SetType(gi.LabelDisplayLarge).SetText(":").
		Style(func(s *styles.Style) {
			s.SetTextWrap(false)
			s.Min.X.Ch(1)
		})

	minute := gi.NewTextField(tv, "minute").
		SetText(strconv.Itoa(tv.Time.Minute()))
	minute.Style(func(s *styles.Style) {
		s.Font.Size.Dp(57)
		s.Min.X.Dp(96)
	})
	minute.OnChange(func(e events.Event) {
		minute, err := strconv.Atoi(minute.Text())
		// TODO(kai/snack)
		if err != nil {
			slog.Error(err.Error())
		}
		// we set our minute and keep everything else
		tt := tv.Time
		tv.Time = time.Date(tt.Year(), tt.Month(), tt.Day(), tt.Hour(), minute, tt.Second(), tt.Nanosecond(), tt.Location())
		if tv.TmpSave != nil {
			tv.TmpSave.SetValue(tv.Time)
		}
		tv.SendChange()
	})

	if !gi.SystemSettings.Clock24 {
		sw := gi.NewSwitches(tv, "am-pm").SetMutex(true).SetType(gi.SwitchSegmentedButton).SetItems("AM", "PM")
		if tv.Time.Hour() < 12 {
			tv.PM = false
			sw.SelectItemAction(0)
		} else {
			tv.PM = true
			sw.SelectItemAction(1)
		}
		sw.Style(func(s *styles.Style) {
			s.Direction = styles.Column
		})
		sw.OnChange(func(e events.Event) {
			si := sw.SelectedItem()
			tt := tv.Time
			switch si {
			case "AM":
				tv.PM = false
				tv.Time = time.Date(tt.Year(), tt.Month(), tt.Day(), tv.Hour, tt.Minute(), tt.Second(), tt.Nanosecond(), tt.Location())
			case "PM":
				tv.PM = true
				tv.Time = time.Date(tt.Year(), tt.Month(), tt.Day(), tv.Hour+12, tt.Minute(), tt.Second(), tt.Nanosecond(), tt.Location())
			default:
				// must always have something valid selected
				tv.PM = false
				sw.SelectItem(0)
				tv.Time = time.Date(tt.Year(), tt.Month(), tt.Day(), tv.Hour, tt.Minute(), tt.Second(), tt.Nanosecond(), tt.Location())
			}
			if tv.TmpSave != nil {
				tv.TmpSave.SetValue(tv.Time)
			}
		})
	}
}

var shortMonths = []string{"Jan", "Feb", "Apr", "Mar", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

// DateView is a view for selecting a date
type DateView struct {
	gi.Frame

	// the time that we are viewing
	Time time.Time `set:"-"`

	// value view that needs to have SaveTmp called on it whenever a change
	// is made to one of the underlying values.
	// pass this down to any sub-views created from a parent
	TmpSave Value `json:"-" xml:"-"`

	// a record of parent View names that have led up to this view
	// displayed as extra contextual information in view dialog windows
	ViewPath string

	// ConfigTime is the time that was configured
	ConfigTime time.Time `set:"-" view:"-"`
}

// SetTime sets the source time and updates the view
func (dv *DateView) SetTime(tim time.Time) *DateView {
	dv.Time = tim
	if dv.TmpSave != nil {
		dv.TmpSave.SetValue(dv.Time)
	}
	dv.SendChange()
	dv.Update()
	return dv
}

func (dv *DateView) Config() {
	if dv.HasChildren() {
		dv.DeleteChildren()
	} else {
		dv.Style(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Grow.Set(0, 0)
		})
	}

	trow := gi.NewLayout(dv)
	trow.Style(func(s *styles.Style) {
		s.Gap.Zero()
	})

	arrowStyle := func(s *styles.Style) {
		s.Padding.SetHoriz(units.Dp(12))
		s.Color = colors.C(colors.Scheme.OnSurfaceVariant)
	}

	gi.NewButton(trow).SetType(gi.ButtonAction).SetIcon(icons.NavigateBefore).OnClick(func(e events.Event) {
		dv.SetTime(dv.Time.AddDate(0, -1, 0))
	}).Style(arrowStyle)

	sms := make([]gi.ChooserItem, len(shortMonths))
	for i, sm := range shortMonths {
		sms[i] = gi.ChooserItem{Value: sm}
	}
	month := gi.NewChooser(trow, "month").SetItems(sms...)
	month.SetCurrentIndex(int(dv.Time.Month() - 1))
	month.OnChange(func(e events.Event) {
		// set our month
		dv.SetTime(dv.Time.AddDate(0, month.CurrentIndex+1-int(dv.Time.Month()), 0))
	})

	gi.NewButton(trow).SetType(gi.ButtonAction).SetIcon(icons.NavigateNext).OnClick(func(e events.Event) {
		dv.SetTime(dv.Time.AddDate(0, 1, 0))
	}).Style(arrowStyle)

	gi.NewButton(trow).SetType(gi.ButtonAction).SetIcon(icons.NavigateBefore).OnClick(func(e events.Event) {
		dv.SetTime(dv.Time.AddDate(-1, 0, 0))
	}).Style(arrowStyle)

	yr := dv.Time.Year()
	var yrs []gi.ChooserItem
	// we go 100 in each direction from the current year
	for i := yr - 100; i <= yr+100; i++ {
		yrs = append(yrs, gi.ChooserItem{Value: i})
	}
	year := gi.NewChooser(trow, "year").SetItems(yrs...)
	year.SetCurrentValue(yr)
	year.OnChange(func(e events.Event) {
		// we are centered at current year with 100 in each direction
		nyr := year.CurrentIndex + yr - 100
		// set our year
		dv.SetTime(dv.Time.AddDate(nyr-dv.Time.Year(), 0, 0))
	})

	gi.NewButton(trow).SetType(gi.ButtonAction).SetIcon(icons.NavigateNext).OnClick(func(e events.Event) {
		dv.SetTime(dv.Time.AddDate(1, 0, 0))
	}).Style(arrowStyle)

	dv.ConfigDateGrid()
	dv.NeedsLayout()
}

func (dv *DateView) ConfigDateGrid() {
	grid := gi.NewLayout(dv, "grid")
	grid.Style(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = 7
	})

	// start of the month
	som := dv.Time.AddDate(0, 0, -dv.Time.Day()+1)
	// end of the month
	eom := dv.Time.AddDate(0, 1, -dv.Time.Day())
	// start of the week containing the start of the month
	somw := som.AddDate(0, 0, -int(som.Weekday()))
	// year day of the start of the week containing the start of the month
	somwyd := somw.YearDay()
	// end of the week containing the end of the month
	eomw := eom.AddDate(0, 0, int(6-eom.Weekday()))
	// year day of the end of the week containing the end of the month
	eomwyd := eomw.YearDay()
	// if we have moved up a year (happens in December),
	// we add the number of days in this year
	if eomw.Year() > somw.Year() {
		eomwyd += time.Date(somw.Year(), 13, -1, 0, 0, 0, 0, somw.Location()).YearDay()
	}

	for yd := somwyd; yd <= eomwyd; yd++ {
		yds := strconv.Itoa(yd)
		// actual time of this date
		dt := somw.AddDate(0, 0, yd-somwyd)
		ds := strconv.Itoa(dt.Day())
		bt := gi.NewButton(grid, "day-"+yds).SetType(gi.ButtonAction).SetText(ds)
		bt.OnClick(func(e events.Event) {
			dv.SetTime(dt)
		})
		bt.Style(func(s *styles.Style) {
			s.Min.X.Dp(32)
			s.Min.Y.Dp(32)
			s.Padding.Set(units.Dp(6))
			if dt.Month() != som.Month() {
				s.Color = colors.C(colors.Scheme.OnSurfaceVariant)
			}
			if dt.Year() == time.Now().Year() && dt.YearDay() == time.Now().YearDay() {
				s.Border.Width.Set(units.Dp(1))
				s.Border.Color.Set(colors.C(colors.Scheme.Primary.Base))
				s.Color = colors.C(colors.Scheme.Primary.Base)
			}
			if dt.Year() == dv.Time.Year() && dt.YearDay() == dv.Time.YearDay() {
				s.Background = colors.C(colors.Scheme.Primary.Base)
				s.Color = colors.C(colors.Scheme.Primary.On)
			}
		})
		bt.OnWidgetAdded(func(w gi.Widget) {
			switch w.PathFrom(bt) {
			case "parts":
				w.Style(func(s *styles.Style) {
					s.Justify.Content = styles.Center
					s.Justify.Items = styles.Center
				})
			case "parts/label":
				lb := w.(*gi.Label)
				lb.Type = gi.LabelBodyLarge
			}
		})
	}
}

// TimeValue presents two text fields for editing a date and time,
// both of which can pull up corresponding picker view dialogs.
type TimeValue struct {
	ValueBase[*gi.Layout]
}

func (v *TimeValue) Config() {
	v.Widget.Style(func(s *styles.Style) {
		s.Grow.Set(0, 0)
	})

	dt := gi.NewTextField(v.Widget, "date").SetTooltip("The date")
	dt.SetLeadingIcon(icons.CalendarToday, func(e events.Event) {
		d := gi.NewBody().AddTitle("Select date")
		dv := NewDateView(d).SetTime(*v.TimeValue())
		d.AddBottomBar(func(pw gi.Widget) {
			d.AddCancel(pw)
			d.AddOk(pw).OnClick(func(e events.Event) {
				v.SetValue(dv.Time)
				v.Update()
			})
		})
		d.NewDialog(dt).Run()
	})
	dt.Style(func(s *styles.Style) {
		s.Min.X.Em(8)
		s.Max.X.Em(10)
	})
	dt.SetReadOnly(v.IsReadOnly())
	dt.SetValidator(func() error {
		d, err := time.Parse("01/02/2006", dt.Text())
		if err != nil {
			return err
		}
		tv := v.TimeValue()
		// new date and old time
		*tv = time.Date(d.Year(), d.Month(), d.Day(), tv.Hour(), tv.Minute(), tv.Second(), tv.Nanosecond(), tv.Location())
		return nil
	})

	tm := gi.NewTextField(v.Widget, "time").SetTooltip("The time")
	tm.SetLeadingIcon(icons.Schedule, func(e events.Event) {
		d := gi.NewBody().AddTitle("Edit time")
		tv := NewTimeView(d).SetTime(*v.TimeValue())
		d.AddBottomBar(func(pw gi.Widget) {
			d.AddCancel(pw)
			d.AddOk(pw).OnClick(func(e events.Event) {
				v.SetValue(tv.Time)
				v.Update()
			})
		})
		d.NewDialog(tm).Run()
	})
	tm.Style(func(s *styles.Style) {
		s.Min.X.Em(8)
		s.Max.X.Em(10)
	})
	tm.SetReadOnly(v.IsReadOnly())
	tm.SetValidator(func() error {
		t, err := time.Parse(gi.SystemSettings.TimeFormat(), tm.Text())
		if err != nil {
			return err
		}
		tv := v.TimeValue()
		// old date and new time
		*tv = time.Date(tv.Year(), tv.Month(), tv.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), tv.Location())
		return nil
	})
}

func (v *TimeValue) Update() {
	tm := v.TimeValue()
	v.Widget.ChildByName("date").(*gi.TextField).SetText(tm.Format("1/2/2006"))
	v.Widget.ChildByName("time").(*gi.TextField).SetText(tm.Format(gi.SystemSettings.TimeFormat()))
}

// TimeValue decodes the value into a *time.Time value, also handling the [fi.FileTime] case.
func (v *TimeValue) TimeValue() *time.Time {
	tmi := laser.PtrValue(v.Value).Interface()
	switch v := tmi.(type) {
	case *time.Time:
		return v
	case *fi.FileTime:
		return (*time.Time)(v)
	}
	return nil
}

// DurationValue represents a [time.Duration] value with a spinner and unit chooser.
type DurationValue struct {
	ValueBase[*gi.Layout]
}

func (v *DurationValue) Config() {
	v.Widget.Style(func(s *styles.Style) {
		s.Grow.Set(0, 0)
	})

	var ch *gi.Chooser

	sp := gi.NewSpinner(v.Widget, "value").SetTooltip("The value of time").SetStep(1).SetPageStep(10)
	sp.OnChange(func(e events.Event) {
		v.SetValue(sp.Value * float32(durationUnitsMap[ch.CurrentItem.Value.(string)]))
	})

	units := make([]gi.ChooserItem, len(durationUnits))
	for i, u := range durationUnits {
		units[i] = gi.ChooserItem{Value: u}
	}

	ch = gi.NewChooser(v.Widget, "unit").SetTooltip("The unit of time").SetItems(units...)
	ch.OnChange(func(e events.Event) {
		// we update the value to fit the unit
		npv := laser.NonPtrValue(v.Value)
		dur := npv.Interface().(time.Duration)
		sp.SetValue(float32(dur) / float32(durationUnitsMap[ch.CurrentItem.Value.(string)]))
	})
}

func (v *DurationValue) Update() {
	npv := laser.NonPtrValue(v.Value)
	dur := npv.Interface().(time.Duration)
	un := "seconds"
	undur := time.Duration(0)
	for _, u := range durationUnits {
		v := durationUnitsMap[u]
		if v > dur {
			break
		}
		un = u
		undur = v
	}
	adur := float32(dur)
	if undur != 0 {
		adur /= float32(undur)
	}

	v.Widget.ChildByName("value").(*gi.Spinner).SetValue(adur)
	v.Widget.ChildByName("unit").(*gi.Chooser).SetCurrentValue(un)
}

var durationUnits = []string{
	"nanoseconds",
	"microseconds",
	"milliseconds",
	"seconds",
	"minutes",
	"hours",
	"days",
	"weeks",
	"months",
	"years",
}

var durationUnitsMap = map[string]time.Duration{
	"nanoseconds":  time.Nanosecond,
	"microseconds": time.Microsecond,
	"milliseconds": time.Millisecond,
	"seconds":      time.Second,
	"minutes":      time.Minute,
	"hours":        time.Hour,
	"days":         24 * time.Hour,
	"weeks":        7 * 24 * time.Hour,
	"months":       30 * 24 * time.Hour,
	"years":        365 * 24 * time.Hour,
}
