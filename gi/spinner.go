// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"log/slog"
	"strconv"

	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/grr"
	"goki.dev/icons"
	"goki.dev/mat32/v2"
)

// Spinner combines a TextField with up / down buttons for incrementing /
// decrementing values -- all configured within the Parts of the widget
type Spinner struct { //goki:embedder
	TextField

	// current value
	Value float32 `xml:"value" set:"-"`

	// is there a minimum value to enforce
	HasMin bool `xml:"has-min" set:"-"`

	// minimum value in range
	Min float32 `xml:"min" set:"-"`

	// is there a maximumvalue to enforce
	HasMax bool `xml:"has-max" set:"-"`

	// maximum value in range
	Max float32 `xml:"max" set:"-"`

	// smallest step size to increment
	Step float32 `xml:"step"`

	// larger PageUp / Dn step size
	PageStep float32 `xml:"pagestep"`

	// specifies the precision of decimal places (total, not after the decimal point) to use in representing the number -- this helps to truncate small weird floating point values in the nether regions
	Prec int

	// prop = format -- format string for printing the value -- blank defaults to %g.  If decimal based (ends in d, b, c, o, O, q, x, X, or U) then value is converted to decimal prior to printing
	Format string `xml:"format"`
}

func (sp *Spinner) CopyFieldsFrom(frm any) {
	fr := frm.(*Spinner)
	sp.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	sp.Value = fr.Value
	sp.HasMin = fr.HasMin
	sp.Min = fr.Min
	sp.HasMax = fr.HasMax
	sp.Max = fr.Max
	sp.Step = fr.Step
	sp.PageStep = fr.PageStep
	sp.Prec = fr.Prec
}

func (sp *Spinner) OnInit() {
	sp.Step = 0.1
	sp.PageStep = 0.2
	sp.Max = 1.0
	sp.Prec = 6
	sp.SetLeadingIcon(icons.Remove, func(e events.Event) {
		sp.IncrValue(-1)
	}).SetTrailingIcon(icons.Add, func(e events.Event) {
		sp.IncrValue(1)
	})
	sp.HandleSpinnerEvents()
	sp.SpinnerStyles()
}

func (sp *Spinner) SpinnerStyles() {
	sp.TextFieldStyles()
	sp.Style(func(s *styles.Style) {
		s.SetMinPrefWidth(units.Em(6))
	})
}

// SetMin sets the min limits on the value
func (sp *Spinner) SetMin(min float32) *Spinner {
	sp.HasMin = true
	sp.Min = min
	return sp
}

// SetMax sets the max limits on the value
func (sp *Spinner) SetMax(max float32) *Spinner {
	sp.HasMax = true
	sp.Max = max
	return sp
}

// SetMinMax sets the min and max limits on the value
func (sp *Spinner) SetMinMax(hasMin bool, min float32, hasMax bool, max float32) *Spinner {
	sp.HasMin = hasMin
	sp.Min = min
	sp.HasMax = hasMax
	sp.Max = max
	if sp.Max < sp.Min {
		slog.Warn("gi.Spinner.SetMinMax: max was less than min; disabling limits")
		sp.HasMax = false
		sp.HasMin = false
	}
	return sp
}

// SetValue sets the value, enforcing any limits, and updates the display
func (sp *Spinner) SetValue(val float32) *Spinner {
	updt := sp.UpdateStart()
	defer sp.UpdateEndRender(updt)
	sp.Value = val
	if sp.HasMax {
		sp.Value = mat32.Min(sp.Value, sp.Max)
	}
	if sp.HasMin {
		sp.Value = mat32.Max(sp.Value, sp.Min)
	}
	sp.Value = mat32.Truncate(sp.Value, sp.Prec)
	sp.SetText(sp.ValToString(sp.Value))
	return sp
}

// SetValueAction calls SetValue and also emits the signal
func (sp *Spinner) SetValueAction(val float32) *Spinner {
	sp.SetValue(val)
	sp.SendChange()
	return sp
}

// IncrValue increments the value by given number of steps (+ or -),
// and enforces it to be an even multiple of the step size (snap-to-value),
// and emits the signal
func (sp *Spinner) IncrValue(steps float32) *Spinner {
	if sp.IsReadOnly() {
		return sp
	}
	val := sp.Value + steps*sp.Step
	val = mat32.IntMultiple(val, sp.Step)
	return sp.SetValueAction(val)
}

// PageIncrValue increments the value by given number of page steps (+ or -),
// and enforces it to be an even multiple of the step size (snap-to-value),
// and emits the signal
func (sp *Spinner) PageIncrValue(steps float32) *Spinner {
	if sp.IsReadOnly() {
		return sp
	}
	val := sp.Value + steps*sp.PageStep
	val = mat32.IntMultiple(val, sp.PageStep)
	return sp.SetValueAction(val)
}

// FormatIsInt returns true if the format string requires an integer value
func (sp *Spinner) FormatIsInt() bool {
	if sp.Format == "" {
		return false
	}
	fc := sp.Format[len(sp.Format)-1]
	switch fc {
	case 'd', 'b', 'c', 'o', 'O', 'q', 'x', 'X', 'U':
		return true
	}
	return false
}

// ValToString converts the value to the string representation thereof
func (sp *Spinner) ValToString(val float32) string {
	if sp.Format == "" {
		return fmt.Sprintf("%g", val)
	}
	if sp.FormatIsInt() {
		return fmt.Sprintf(sp.Format, int64(val))
	}
	return fmt.Sprintf(sp.Format, val)
}

// StringToVal converts the string field back to float value
func (sp *Spinner) StringToVal(str string) (float32, error) {
	// TODO(kai/snack)
	if sp.Format == "" {
		f64, err := strconv.ParseFloat(str, 32)
		return float32(f64), grr.Log0(err)
	}
	if sp.FormatIsInt() {
		var ival int
		_, err := fmt.Sscanf(str, sp.Format, &ival)
		return float32(ival), grr.Log0(err)
	}
	var fval float32
	_, err := fmt.Sscanf(str, sp.Format, &fval)
	return fval, grr.Log0(err)
}

func (sp *Spinner) HandleSpinnerEvents() {
	sp.HandleTextFieldEvents()
	sp.HandleSpinnerScroll()
	sp.HandleSpinnerKeys()
}

func (sp *Spinner) HandleSpinnerScroll() {
	sp.On(events.Scroll, func(e events.Event) {
		if sp.IsReadOnly() || !sp.StateIs(states.Focused) {
			return
		}
		se := e.(*events.MouseScroll)
		se.SetHandled()
		sp.IncrValue(float32(se.DimDelta(mat32.Y)))
	})
}

func (sp *Spinner) HandleSpinnerKeys() {
	sp.OnChange(func(e events.Event) {
		text := sp.Text()
		val, err := sp.StringToVal(text)
		if err != nil {
			// TODO: use validation
			slog.Error("invalid Spinner value", "value", text, "err", err)
			return
		}
		sp.SetValue(val)
	})
	sp.OnKeyChord(func(e events.Event) {
		if sp.IsReadOnly() {
			return
		}
		if KeyEventTrace {
			fmt.Printf("Spinner KeyChordEvent: %v\n", sp.Path())
		}
		kf := keyfun.Of(e.KeyChord())
		switch {
		case kf == keyfun.MoveUp:
			e.SetHandled()
			sp.IncrValue(1)
		case kf == keyfun.MoveDown:
			e.SetHandled()
			sp.IncrValue(-1)
		case kf == keyfun.PageUp:
			e.SetHandled()
			sp.PageIncrValue(1)
		case kf == keyfun.PageDown:
			e.SetHandled()
			sp.PageIncrValue(-1)
		}
	})
}
