// Copyright (c) 2018, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"log/slog"
	"strconv"

	"goki.dev/events"
	"goki.dev/grr"
	"goki.dev/icons"
	"goki.dev/keyfun"
	"goki.dev/mat32"
	"goki.dev/states"
	"goki.dev/styles"
)

// Spinner combines a TextField with up / down buttons for incrementing /
// decrementing values -- all configured within the Parts of the widget
type Spinner struct { //goki:embedder
	TextField

	// Value is the current value
	Value float32 `set:"-"`

	// HasMin is whether there is a minimum value to enforce
	HasMin bool `set:"-"`

	// If HasMin is true, Min is the the minimum value in range
	Min float32 `set:"-"`

	// HaxMax is whether there is a maximum value to enforce
	HasMax bool `set:"-"`

	// If HasMax is true, Max is the maximum value in range
	Max float32 `set:"-"`

	// Step is the smallest step size to increment
	Step float32

	// PageStep is a larger step size used for PageUp and PageDown
	PageStep float32

	// Prec specifies the precision of decimal places
	// (total, not after the decimal point) to use in
	// representing the number. This helps to truncate
	// small weird floating point values.
	Prec int

	// Format is the format string to use for printing the value.
	// If it unset, %g is used. If it is decimal based
	// (ends in d, b, c, o, O, q, x, X, or U) then the value is
	// converted to decimal prior to printing.
	Format string
}

func (sp *Spinner) CopyFieldsFrom(frm any) {
	fr := frm.(*Spinner)
	sp.TextField.CopyFieldsFrom(&fr.TextField)
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
	sp.WidgetBase.OnInit()
	sp.HandleEvents()
	sp.SetStyles()
}

func (sp *Spinner) SetStyles() {
	sp.Step = 0.1
	sp.PageStep = 0.2
	sp.Max = 1.0
	sp.Prec = 6
	sp.SetLeadingIcon(icons.Remove, func(e events.Event) {
		sp.IncrValue(-1)
	}).SetTrailingIcon(icons.Add, func(e events.Event) {
		sp.IncrValue(1)
	})
	sp.TextField.SetStyles()
	sp.Style(func(s *styles.Style) {
		if sp.IsReadOnly() {
			s.Min.X.Ch(4)
			s.Max.X.Ch(8)
		} else {
			s.Min.X.Ch(14)
			s.Max.X.Ch(18)
		}
	})
}

func (sp *Spinner) SetTextToValue() {
	sp.SetTextUpdate(sp.ValToString(sp.Value))
}

func (sp *Spinner) SizeUp() {
	sp.SetTextToValue()
	sp.TextField.SizeUp()
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

// SetValue sets the value, enforcing any limits, and updates the display
func (sp *Spinner) SetValue(val float32) *Spinner {
	updt := sp.UpdateStart()
	defer sp.UpdateEndRender(updt)
	sp.Value = val
	if sp.HasMax && sp.Value > sp.Max {
		if sp.HasMin {
			sp.Value = sp.Min // wrap-around
		} else {
			sp.Value = sp.Max
		}
	} else if sp.HasMin && sp.Value < sp.Min {
		if sp.HasMax {
			sp.Value = sp.Max // wrap-around
		} else {
			sp.Value = sp.Min
		}
	}
	sp.Value = mat32.Truncate(sp.Value, sp.Prec)
	sp.SetTextToValue()
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
		return float32(f64), grr.Log(err)
	}
	if sp.FormatIsInt() {
		var ival int
		_, err := fmt.Sscanf(str, sp.Format, &ival)
		return float32(ival), grr.Log(err)
	}
	var fval float32
	_, err := fmt.Sscanf(str, sp.Format, &fval)
	return fval, grr.Log(err)
}

func (sp *Spinner) WidgetTooltip() string {
	res := sp.Tooltip
	if sp.HasMin {
		if res != "" {
			res += " "
		}
		res += fmt.Sprintf("(minimum: %g", sp.Min)
		if !sp.HasMax {
			res += ")"
		}
	}
	if sp.HasMax {
		if sp.HasMin {
			res += ", "
		} else if res != "" {
			res += " ("
		} else {
			res += "("
		}
		res += fmt.Sprintf("maximum: %g)", sp.Max)
	}
	return res
}

func (sp *Spinner) HandleEvents() {
	sp.TextField.HandleEvents()
	sp.On(events.Scroll, func(e events.Event) {
		if sp.IsReadOnly() || !sp.StateIs(states.Focused) {
			return
		}
		se := e.(*events.MouseScroll)
		se.SetHandled()
		sp.IncrValue(float32(se.Delta.Y))
	})
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
	sp.HandleKeys()
}

func (sp *Spinner) HandleKeys() {
	sp.OnKeyChord(func(e events.Event) {
		if sp.IsReadOnly() {
			return
		}
		if DebugSettings.KeyEventTrace {
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
