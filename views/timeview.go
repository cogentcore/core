// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"strconv"
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

// TimeView is a view for selecting a time
type TimeView struct {
	core.Frame

	// Time is the time that we are viewing
	Time time.Time

	// the raw input hour
	Hour int `set:"-"`

	// whether we are in PM mode (so we have to add 12h to everything)
	PM bool `set:"-"`
}

func (tv *TimeView) WidgetValue() any { return &tv.Time }

func (tv *TimeView) Init() {
	tv.Frame.Init()
	core.AddChild(tv, func(w *core.Spinner) {
		w.SetStep(1).SetEnforceStep(true)
		if core.SystemSettings.Clock24 {
			tv.Hour = tv.Time.Hour()
			w.SetMax(24).SetMin(0)
		} else {
			tv.Hour = tv.Time.Hour() % 12
			if tv.Hour == 0 {
				tv.Hour = 12
			}
			w.SetMax(12).SetMin(1)
		}
		w.SetValue(float32(tv.Hour))
		w.Style(func(s *styles.Style) {
			s.Font.Size.Dp(57)
			s.Min.X.Dp(96)
		})
		w.OnChange(func(e events.Event) {
			hr := int(w.Value)
			if hr == 12 && !core.SystemSettings.Clock24 {
				hr = 0
			}
			tv.Hour = hr
			if tv.PM {
				// only add to local variable
				hr += 12
			}
			// we set our hour and keep everything else
			tt := tv.Time
			tv.Time = time.Date(tt.Year(), tt.Month(), tt.Day(), hr, tt.Minute(), tt.Second(), tt.Nanosecond(), tt.Location())
			tv.SendChange()
		})
	})
	core.AddChild(tv, func(w *core.Text) {
		w.SetType(core.TextDisplayLarge).SetText(":")
		w.Style(func(s *styles.Style) {
			s.SetTextWrap(false)
			s.Min.X.Ch(1)
		})
	})
	core.AddChild(tv, func(w *core.Spinner) {
		w.SetStep(1).SetEnforceStep(true).
			SetMin(0).SetMax(60).SetFormat("%02d").
			SetValue(float32(tv.Time.Minute()))
		w.Style(func(s *styles.Style) {
			s.Font.Size.Dp(57)
			s.Min.X.Dp(96)
		})
		w.OnChange(func(e events.Event) {
			// we set our minute and keep everything else
			// TODO(config)
			// tt := tv.Time
			// tv.Time = time.Date(tt.Year(), tt.Month(), tt.Day(), tt.Hour(), int(minute.Value), tt.Second(), tt.Nanosecond(), tt.Location())
			tv.SendChange()
		})
	})
	tv.Maker(func(p *core.Plan) {
		if !core.SystemSettings.Clock24 {
			core.Add(p, func(w *core.Switches) {
				w.SetMutex(true).SetType(core.SwitchSegmentedButton).SetItems(core.SwitchItem{Value: "AM"}, core.SwitchItem{Value: "PM"})
				tv.PM = tv.Time.Hour() >= 12
				w.Style(func(s *styles.Style) {
					s.Direction = styles.Column
				})
				w.Updater(func() {
					if tv.PM {
						w.SelectValue("PM")
					} else {
						w.SelectValue("AM")
					}
				})

				w.OnChange(func(e events.Event) {
					si := w.SelectedItem()
					tt := tv.Time
					if tv.Hour == 12 {
						tv.Hour = 0
					}
					switch si.Value {
					case "AM":
						tv.PM = false
						tv.Time = time.Date(tt.Year(), tt.Month(), tt.Day(), tv.Hour, tt.Minute(), tt.Second(), tt.Nanosecond(), tt.Location())
					case "PM":
						tv.PM = true
						tv.Time = time.Date(tt.Year(), tt.Month(), tt.Day(), tv.Hour+12, tt.Minute(), tt.Second(), tt.Nanosecond(), tt.Location())
					default:
						// must always have something valid selected
						tv.PM = false
						tv.Time = time.Date(tt.Year(), tt.Month(), tt.Day(), tv.Hour, tt.Minute(), tt.Second(), tt.Nanosecond(), tt.Location())
					}
				})
			})
		}
	})
}

var shortMonths = []string{"Jan", "Feb", "Apr", "Mar", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

// DateView is a view for selecting a date
type DateView struct {
	core.Frame

	// Time is the time that we are viewing
	Time time.Time `set:"-"`
}

// SetTime sets the source time and updates the view
func (dv *DateView) SetTime(tim time.Time) *DateView { // TODO(config)
	dv.Time = tim
	dv.SendChange()
	dv.Update()
	return dv
}

func (dv *DateView) Init() {
	dv.Frame.Init()
	dv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	core.AddChild(dv, func(w *core.Frame) {
		w.Style(func(s *styles.Style) {
			s.Gap.Zero()
		})
		arrowStyle := func(s *styles.Style) {
			s.Padding.SetHorizontal(units.Dp(12))
			s.Color = colors.C(colors.Scheme.OnSurfaceVariant)
		}
		core.AddChild(w, func(w *core.Button) {
			w.SetType(core.ButtonAction).SetIcon(icons.NavigateBefore)
			w.OnClick(func(e events.Event) {
				dv.SetTime(dv.Time.AddDate(0, -1, 0))
			})
			w.Style(arrowStyle)
		})
		core.AddChild(w, func(w *core.Chooser) {
			sms := make([]core.ChooserItem, len(shortMonths))
			for i, sm := range shortMonths {
				sms[i] = core.ChooserItem{Value: sm}
			}
			w.SetItems(sms...)
			w.SetCurrentIndex(int(dv.Time.Month() - 1))
			w.OnChange(func(e events.Event) {
				// set our month
				dv.SetTime(dv.Time.AddDate(0, w.CurrentIndex+1-int(dv.Time.Month()), 0))
			})
		})
		core.AddChild(w, func(w *core.Button) {
			w.SetType(core.ButtonAction).SetIcon(icons.NavigateNext)
			w.OnClick(func(e events.Event) {
				dv.SetTime(dv.Time.AddDate(0, 1, 0))
			})
			w.Style(arrowStyle)
		})
		core.AddChild(w, func(w *core.Button) {
			w.SetType(core.ButtonAction).SetIcon(icons.NavigateBefore)
			w.OnClick(func(e events.Event) {
				dv.SetTime(dv.Time.AddDate(-1, 0, 0))
			})
			w.Style(arrowStyle)
		})
		core.AddChild(w, func(w *core.Chooser) {
			yr := dv.Time.Year()
			var yrs []core.ChooserItem
			// we go 100 in each direction from the current year
			for i := yr - 100; i <= yr+100; i++ {
				yrs = append(yrs, core.ChooserItem{Value: i})
			}
			w.SetItems(yrs...)
			w.SetCurrentValue(yr)
			w.OnChange(func(e events.Event) {
				// we are centered at current year with 100 in each direction
				nyr := w.CurrentIndex + yr - 100
				// set our year
				dv.SetTime(dv.Time.AddDate(nyr-dv.Time.Year(), 0, 0))
			})
		})
		core.AddChild(w, func(w *core.Button) {
			w.SetType(core.ButtonAction).SetIcon(icons.NavigateNext)
			w.OnClick(func(e events.Event) {
				dv.SetTime(dv.Time.AddDate(1, 0, 0))
			})
			w.Style(arrowStyle)
		})
	})
	core.AddChild(dv, func(w *core.Frame) {
		w.Style(func(s *styles.Style) {
			s.Display = styles.Grid
			s.Columns = 7
		})
		w.Maker(func(p *core.Plan) {
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
				core.AddAt(p, strconv.Itoa(yd), func(w *core.Button) { // TODO(config)
					// actual time of this date
					dt := somw.AddDate(0, 0, yd-somwyd)
					ds := strconv.Itoa(dt.Day())
					w.SetType(core.ButtonAction).SetText(ds)
					w.OnClick(func(e events.Event) {
						dv.SetTime(dt)
					})
					w.Style(func(s *styles.Style) {
						s.CenterAll()
						s.Min.Set(units.Dp(32))
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
					w.Maker(func(p *core.Plan) {
						core.AddInit(p, "text", func(w *core.Text) {
							w.SetType(core.TextBodyLarge)
						})
					})
				})
			}
		})
	})
}

/* TODO(config)
// TimeValue presents two text fields for editing a date and time,
// both of which can pull up corresponding picker view dialogs.
type TimeValue struct {
	ValueBase[*core.Frame]
}

func (v *TimeValue) Config() {
	dt := core.NewTextField(v.Widget).SetTooltip("The date")
	dt.SetLeadingIcon(icons.CalendarToday, func(e events.Event) {
		d := core.NewBody().AddTitle("Select date")
		dv := NewDateView(d).SetTime(*v.TimeValue())
		d.AddBottomBar(func(parent core.Widget) {
			d.AddCancel(parent)
			d.AddOK(parent).OnClick(func(e events.Event) {
				v.SetValue(dv.Time)
				v.Update()
			})
		})
		d.RunDialog(dt)
	})
	dt.Style(func(s *styles.Style) {
		s.Min.X.Em(8)
		s.Max.X.Em(10)
	})
	dt.SetReadOnly(v.IsReadOnly())
	dt.SetValidator(func() error {
		d, err := time.Parse("1/2/2006", dt.Text())
		if err != nil {
			return err
		}
		tv := v.TimeValue()
		// new date and old time
		v.SetValue(time.Date(d.Year(), d.Month(), d.Day(), tv.Hour(), tv.Minute(), tv.Second(), tv.Nanosecond(), tv.Location()))
		return nil
	})

	tm := core.NewTextField(v.Widget).SetTooltip("The time")
	tm.SetLeadingIcon(icons.Schedule, func(e events.Event) {
		d := core.NewBody().AddTitle("Edit time")
		tv := NewTimeView(d).SetTime(*v.TimeValue())
		d.AddBottomBar(func(parent core.Widget) {
			d.AddCancel(parent)
			d.AddOK(parent).OnClick(func(e events.Event) {
				v.SetValue(tv.Time)
				v.Update()
			})
		})
		d.RunDialog(tm)
	})
	tm.Style(func(s *styles.Style) {
		s.Min.X.Em(8)
		s.Max.X.Em(10)
	})
	tm.SetReadOnly(v.IsReadOnly())
	tm.SetValidator(func() error {
		t, err := time.Parse(core.SystemSettings.TimeFormat(), tm.Text())
		if err != nil {
			return err
		}
		tv := v.TimeValue()
		// old date and new time
		v.SetValue(time.Date(tv.Year(), tv.Month(), tv.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), tv.Location()))
		return nil
	})
}

func (v *TimeValue) Update() {
	tm := v.TimeValue()
	v.Widget.Child(0).(*core.TextField).SetText(tm.Format("1/2/2006"))
	v.Widget.Child(1).(*core.TextField).SetText(tm.Format(core.SystemSettings.TimeFormat()))
}

// TimeValue decodes the value into a *time.Time value, also handling the [fileinfo.FileTime] case.
func (v *TimeValue) TimeValue() *time.Time {
	tmi := reflectx.PointerValue(v.Value).Interface()
	switch v := tmi.(type) {
	case *time.Time:
		return v
	case *fileinfo.FileTime:
		return (*time.Time)(v)
	}
	return nil
}

*/

// DurationInput represents a [time.Duration] value with a spinner and unit chooser.
type DurationInput struct {
	core.Frame
	Duration time.Duration
}

func (di *DurationInput) WidgetValue() any { return &di.Duration }

func (di *DurationInput) Init() {
	di.Frame.Init()

	var ch *core.Chooser // TODO(config)

	sp := core.NewSpinner(di).SetStep(1).SetPageStep(10)
	sp.SetTooltip("The value of time")
	sp.OnChange(func(e events.Event) {
		di.Duration = time.Duration(sp.Value * float32(durationUnitsMap[ch.CurrentItem.Value.(string)]))
	})

	units := make([]core.ChooserItem, len(durationUnits))
	for i, u := range durationUnits {
		units[i] = core.ChooserItem{Value: u}
	}

	ch = core.NewChooser(di).SetItems(units...)
	ch.SetTooltip("The unit of time")
	ch.OnChange(func(e events.Event) {
		// we update the value to fit the unit
		sp.SetValue(float32(di.Duration) / float32(durationUnitsMap[ch.CurrentItem.Value.(string)]))
	})
}

// TODO(config)
/*
func (v *DurationInput) Update() {
	npv := reflectx.NonPointerValue(v.Value)
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

	v.Widget.Child(0).(*core.Spinner).SetValue(adur)
	v.Widget.Child(1).(*core.Chooser).SetCurrentValue(un)
}
*/

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
