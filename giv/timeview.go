// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"time"

	"goki.dev/gi/v2/gi"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/laser"
)

var timeUnits = map[string]time.Duration{
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

// DurationValue presents a spinner and unit chooser for a [time.Duration]
type DurationValue struct {
	ValueBase
}

func (vv *DurationValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.FrameType
	return vv.WidgetTyp
}

func (vv *DurationValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	npv := laser.NonPtrValue(vv.Value)
	dur := npv.Interface().(time.Duration)
	un := "seconds"
	undur := time.Duration(0)
	for k, v := range timeUnits {
		if v > dur {
			break
		}
		un = k
		undur = v
	}
	adur := dur
	if undur != 0 {
		adur = dur / undur
	}

	fr := vv.Widget.(*gi.Frame)
	fr.ChildByName("value").(*gi.Spinner).SetValue(float32(adur))
	fr.ChildByName("unit").(*gi.Chooser).SetCurVal(un)
}

func (vv *DurationValue) ConfigWidget(widg gi.Widget, sc *gi.Scene) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	fr := vv.Widget.(*gi.Frame)
	fr.SetLayout(gi.LayoutHoriz)

	if len(fr.Kids) > 0 {
		return
	}

	var ch *gi.Chooser

	sp := gi.NewSpinner(fr, "value").SetTooltip("The value of time").SetStep(1).SetPageStep(10)
	sp.OnChange(func(e events.Event) {
		vv.SetValue(sp.Value * float32(timeUnits[ch.CurLabel]))
	})
	sp.Config(sc)

	units := []any{}
	for k := range timeUnits {
		units = append(units, k)
	}

	ch = gi.NewChooser(fr, "unit").SetTooltip("The unit of time").SetItems(units)
	ch.OnChange(func(e events.Event) {
		vv.SetValue(sp.Value * float32(timeUnits[ch.CurLabel]))
	})

	vv.UpdateWidget()
}
