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

// TimePicker is a widget for picking a time.
type TimePicker struct {
	core.Frame

	// Time is the time that we are viewing
	Time time.Time

	// the raw input hour
	Hour int `set:"-"`

	// whether we are in PM mode (so we have to add 12h to everything)
	PM bool `set:"-"`
}

func (tp *TimePicker) WidgetValue() any { return &tp.Time }

func (tp *TimePicker) Init() {
	tp.Frame.Init()
	core.AddChild(tp, func(w *core.Spinner) {
		w.SetStep(1).SetEnforceStep(true)
		if core.SystemSettings.Clock24 {
			tp.Hour = tp.Time.Hour()
			w.SetMax(24).SetMin(0)
		} else {
			tp.Hour = tp.Time.Hour() % 12
			if tp.Hour == 0 {
				tp.Hour = 12
			}
			w.SetMax(12).SetMin(1)
		}
		w.SetValue(float32(tp.Hour))
		w.Styler(func(s *styles.Style) {
			s.Font.Size.Dp(57)
			s.Min.X.Dp(96)
		})
		w.OnChange(func(e events.Event) {
			hr := int(w.Value)
			if hr == 12 && !core.SystemSettings.Clock24 {
				hr = 0
			}
			tp.Hour = hr
			if tp.PM {
				// only add to local variable
				hr += 12
			}
			// we set our hour and keep everything else
			tt := tp.Time
			tp.Time = time.Date(tt.Year(), tt.Month(), tt.Day(), hr, tt.Minute(), tt.Second(), tt.Nanosecond(), tt.Location())
			tp.SendChange()
		})
	})
	core.AddChild(tp, func(w *core.Text) {
		w.SetType(core.TextDisplayLarge).SetText(":")
		w.Styler(func(s *styles.Style) {
			s.SetTextWrap(false)
			s.Min.X.Ch(1)
		})
	})
	core.AddChild(tp, func(w *core.Spinner) {
		w.SetStep(1).SetEnforceStep(true).
			SetMin(0).SetMax(60).SetFormat("%02d").
			SetValue(float32(tp.Time.Minute()))
		w.Styler(func(s *styles.Style) {
			s.Font.Size.Dp(57)
			s.Min.X.Dp(96)
		})
		w.OnChange(func(e events.Event) {
			// we set our minute and keep everything else
			// TODO(config)
			// tt := tp.Time
			// tp.Time = time.Date(tt.Year(), tt.Month(), tt.Day(), tt.Hour(), int(minute.Value), tt.Second(), tt.Nanosecond(), tt.Location())
			tp.SendChange()
		})
	})
	tp.Maker(func(p *core.Plan) {
		if !core.SystemSettings.Clock24 {
			core.Add(p, func(w *core.Switches) {
				w.SetMutex(true).SetType(core.SwitchSegmentedButton).SetItems(core.SwitchItem{Value: "AM"}, core.SwitchItem{Value: "PM"})
				tp.PM = tp.Time.Hour() >= 12
				w.Styler(func(s *styles.Style) {
					s.Direction = styles.Column
				})
				w.Updater(func() {
					if tp.PM {
						w.SelectValue("PM")
					} else {
						w.SelectValue("AM")
					}
				})

				w.OnChange(func(e events.Event) {
					si := w.SelectedItem()
					tt := tp.Time
					if tp.Hour == 12 {
						tp.Hour = 0
					}
					switch si.Value {
					case "AM":
						tp.PM = false
						tp.Time = time.Date(tt.Year(), tt.Month(), tt.Day(), tp.Hour, tt.Minute(), tt.Second(), tt.Nanosecond(), tt.Location())
					case "PM":
						tp.PM = true
						tp.Time = time.Date(tt.Year(), tt.Month(), tt.Day(), tp.Hour+12, tt.Minute(), tt.Second(), tt.Nanosecond(), tt.Location())
					default:
						// must always have something valid selected
						tp.PM = false
						tp.Time = time.Date(tt.Year(), tt.Month(), tt.Day(), tp.Hour, tt.Minute(), tt.Second(), tt.Nanosecond(), tt.Location())
					}
				})
			})
		}
	})
}

var shortMonths = []string{"Jan", "Feb", "Apr", "Mar", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

// DatePicker is a widget for picking a date.
type DatePicker struct {
	core.Frame

	// Time is the time that we are viewing
	Time time.Time `set:"-"`
}

// SetTime sets the source time and updates the view
func (dp *DatePicker) SetTime(tim time.Time) *DatePicker { // TODO(config)
	dp.Time = tim
	dp.SendChange()
	dp.Update()
	return dp
}

func (dp *DatePicker) Init() {
	dp.Frame.Init()
	dp.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	core.AddChild(dp, func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Gap.Zero()
		})
		arrowStyle := func(s *styles.Style) {
			s.Padding.SetHorizontal(units.Dp(12))
			s.Color = colors.C(colors.Scheme.OnSurfaceVariant)
		}
		core.AddChild(w, func(w *core.Button) {
			w.SetType(core.ButtonAction).SetIcon(icons.NavigateBefore)
			w.OnClick(func(e events.Event) {
				dp.SetTime(dp.Time.AddDate(0, -1, 0))
			})
			w.Styler(arrowStyle)
		})
		core.AddChild(w, func(w *core.Chooser) {
			sms := make([]core.ChooserItem, len(shortMonths))
			for i, sm := range shortMonths {
				sms[i] = core.ChooserItem{Value: sm}
			}
			w.SetItems(sms...)
			w.SetCurrentIndex(int(dp.Time.Month() - 1))
			w.OnChange(func(e events.Event) {
				// set our month
				dp.SetTime(dp.Time.AddDate(0, w.CurrentIndex+1-int(dp.Time.Month()), 0))
			})
		})
		core.AddChild(w, func(w *core.Button) {
			w.SetType(core.ButtonAction).SetIcon(icons.NavigateNext)
			w.OnClick(func(e events.Event) {
				dp.SetTime(dp.Time.AddDate(0, 1, 0))
			})
			w.Styler(arrowStyle)
		})
		core.AddChild(w, func(w *core.Button) {
			w.SetType(core.ButtonAction).SetIcon(icons.NavigateBefore)
			w.OnClick(func(e events.Event) {
				dp.SetTime(dp.Time.AddDate(-1, 0, 0))
			})
			w.Styler(arrowStyle)
		})
		core.AddChild(w, func(w *core.Chooser) {
			yr := dp.Time.Year()
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
				dp.SetTime(dp.Time.AddDate(nyr-dp.Time.Year(), 0, 0))
			})
		})
		core.AddChild(w, func(w *core.Button) {
			w.SetType(core.ButtonAction).SetIcon(icons.NavigateNext)
			w.OnClick(func(e events.Event) {
				dp.SetTime(dp.Time.AddDate(1, 0, 0))
			})
			w.Styler(arrowStyle)
		})
	})
	core.AddChild(dp, func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Display = styles.Grid
			s.Columns = 7
		})
		w.Maker(func(p *core.Plan) {
			// start of the month
			som := dp.Time.AddDate(0, 0, -dp.Time.Day()+1)
			// end of the month
			eom := dp.Time.AddDate(0, 1, -dp.Time.Day())
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
						dp.SetTime(dt)
					})
					w.Styler(func(s *styles.Style) {
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
						if dt.Year() == dp.Time.Year() && dt.YearDay() == dp.Time.YearDay() {
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

// TimeInput presents two text fields for editing a date and time,
// both of which can pull up corresponding picker dialogs.
type TimeInput struct {
	core.Frame
	Time time.Time
}

func (ti *TimeInput) WidgetValue() any { return &ti.Time }

func (ti *TimeInput) Init() {
	ti.Frame.Init()

	style := func(s *styles.Style) {
		s.Min.X.Em(8)
		s.Max.X.Em(10)
		if ti.IsReadOnly() { // must inherit abilities when read only for table
			s.Abilities = ti.Styles.Abilities
		}
	}

	core.AddChild(ti, func(w *core.TextField) {
		w.SetTooltip("The date")
		w.SetLeadingIcon(icons.CalendarToday, func(e events.Event) {
			d := core.NewBody().AddTitle("Select date")
			dp := NewDatePicker(d).SetTime(ti.Time)
			d.AddBottomBar(func(parent core.Widget) {
				d.AddCancel(parent)
				d.AddOK(parent).OnClick(func(e events.Event) {
					ti.Time = dp.Time
					ti.SendChange()
					ti.Update()
				})
			})
			d.RunDialog(w)
		})
		w.Styler(style)
		w.Updater(func() {
			w.SetReadOnly(ti.IsReadOnly())
			w.SetText(ti.Time.Format("1/2/2006"))
		})
		w.SetValidator(func() error {
			d, err := time.Parse("1/2/2006", w.Text())
			if err != nil {
				return err
			}
			// new date and old time
			ti.Time = time.Date(d.Year(), d.Month(), d.Day(), ti.Time.Hour(), ti.Time.Minute(), ti.Time.Second(), ti.Time.Nanosecond(), ti.Time.Location())
			ti.SendChange()
			return nil
		})
	})

	core.AddChild(ti, func(w *core.TextField) {
		w.SetTooltip("The time")
		w.SetLeadingIcon(icons.Schedule, func(e events.Event) {
			d := core.NewBody().AddTitle("Edit time")
			tp := NewTimePicker(d).SetTime(ti.Time)
			d.AddBottomBar(func(parent core.Widget) {
				d.AddCancel(parent)
				d.AddOK(parent).OnClick(func(e events.Event) {
					ti.Time = tp.Time
					ti.SendChange()
					ti.Update()
				})
			})
			d.RunDialog(w)
		})
		w.Styler(style)
		w.Updater(func() {
			w.SetReadOnly(ti.IsReadOnly())
			w.SetText(ti.Time.Format(core.SystemSettings.TimeFormat()))
		})
		w.SetValidator(func() error {
			t, err := time.Parse(core.SystemSettings.TimeFormat(), w.Text())
			if err != nil {
				return err
			}
			// old date and new time
			ti.Time = time.Date(ti.Time.Year(), ti.Time.Month(), ti.Time.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), ti.Time.Location())
			ti.SendChange()
			return nil
		})
	})
}

// DurationInput represents a [time.Duration] value with a spinner and unit chooser.
type DurationInput struct {
	core.Frame

	Duration time.Duration

	// Unit is the unit of time.
	Unit string
}

func (di *DurationInput) WidgetValue() any { return &di.Duration }

func (di *DurationInput) Init() {
	di.Frame.Init()
	core.AddChild(di, func(w *core.Spinner) {
		w.SetStep(1).SetPageStep(10)
		w.SetTooltip("The value of time")
		w.Updater(func() {
			w.SetValue(float32(di.Duration) / float32(durationUnitsMap[di.Unit]))
		})
		w.OnChange(func(e events.Event) {
			di.Duration = time.Duration(w.Value * float32(durationUnitsMap[di.Unit]))
			di.SendChange()
		})
	})
	core.AddChild(di, func(w *core.Chooser) {
		core.Bind(&di.Unit, w)

		units := make([]core.ChooserItem, len(durationUnits))
		for i, u := range durationUnits {
			units[i] = core.ChooserItem{Value: u}
		}

		w.SetItems(units...)
		w.SetTooltip("The unit of time")
		w.Updater(func() {
			if di.Unit == "" {
				di.SetAutoUnit()
			}
		})
		w.OnChange(func(e events.Event) {
			di.Update()
		})
	})
}

// SetAutoUnit sets the [DurationInput.Unit] automatically based on the current duration.
func (di *DurationInput) SetAutoUnit() {
	di.Unit = durationUnits[0]
	for _, u := range durationUnits {
		if durationUnitsMap[u] > di.Duration {
			break
		}
		di.Unit = u
	}
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
