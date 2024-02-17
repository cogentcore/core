// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"log/slog"
	"strconv"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
)

// Spinner combines a TextField with up / down buttons for incrementing /
// decrementing values -- all configured within the Parts of the widget
type Spinner struct { //core:embedder
	TextField

	// Value is the current value.
	Value float32 `set:"-"`

	// HasMin is whether there is a minimum value to enforce.
	HasMin bool `set:"-"`

	// Min, if HasMin is true, is the the minimum value in range.
	Min float32 `set:"-"`

	// HaxMax is whether there is a maximum value to enforce.
	HasMax bool `set:"-"`

	// Max, if HasMax is true, is the maximum value in range.
	Max float32 `set:"-"`

	// Step is the smallest step size to increment when using the
	// up and down buttons and arrow keys.
	Step float32

	// EnforceStep is whether to ensure that the value of the spinner
	// is always a multiple of [Spinner.Step].
	EnforceStep bool

	// PageStep is a larger step size used for PageUp and PageDown events.
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
		s.Grow.Set(0, 0) // TODO: remove
		if sp.IsReadOnly() {
			s.Min.X.Ch(4)
			s.Max.X.Ch(8)
		} else {
			s.Min.X.Ch(14)
			s.Max.X.Ch(18)
		}
		// s.Text.Align = styles.End // this doesn't work
	})
	sp.OnWidgetAdded(func(w Widget) {
		switch w.PathFrom(sp) {
		case "parts/lead-icon", "parts/trail-icon":
			w.Style(func(s *styles.Style) {
				// icons do not get separate focus, as people can
				// use the arrow keys to get the same effect
				s.SetAbilities(false, abilities.Focusable)
			})
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
		sp.Value = sp.Max
	} else if sp.HasMin && sp.Value < sp.Min {
		sp.Value = sp.Min
	}
	sp.Value = mat32.Truncate(sp.Value, sp.Prec)
	if sp.EnforceStep {
		sp.Value -= mat32.Mod(sp.Value, sp.Step)
	}
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
	val = sp.WrapAround(val)
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
	val = sp.WrapAround(val)
	return sp.SetValueAction(val)
}

// WrapAround, if the spinner has a min and a max, converts values less
// than min to max and values greater than max to min.
func (sp *Spinner) WrapAround(val float32) float32 {
	if !sp.HasMin || !sp.HasMax {
		return val
	}
	if val < sp.Min {
		return sp.Max
	}
	if val > sp.Max {
		return sp.Min
	}
	return val
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
	if sp.Format == "" {
		f64, err := strconv.ParseFloat(str, 32)
		return float32(f64), err
	}

	var err error
	if sp.FormatIsInt() {
		var ival int
		_, err = fmt.Sscanf(str, sp.Format, &ival)
		if err == nil {
			return float32(ival), nil
		}
	} else {
		var fval float32
		_, err = fmt.Sscanf(str, sp.Format, &fval)
		if err == nil {
			return fval, nil
		}
	}
	// if we have an error using the formatted version,
	// we try using a pure parse
	f64, ferr := strconv.ParseFloat(str, 32)
	if ferr == nil {
		return float32(f64), nil
	}
	// if everything fails, we return the error for the
	// formatted version
	return 0, err
}

func (sp *Spinner) WidgetTooltip() string {
	res := sp.TextField.WidgetTooltip()
	if sp.Error != nil {
		return res
	}
	if sp.HasMin {
		if res != "" {
			res += " "
		}
		res += "(minimum: " + sp.ValToString(sp.Min)
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
		res += "maximum: " + sp.ValToString(sp.Max) + ")"
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
		sp.IncrValue(se.Delta.Y)
	})
	sp.SetValidator(func() error {
		text := sp.Text()
		val, err := sp.StringToVal(text)
		if err != nil {
			return err
		}
		sp.SetValue(val)
		return nil
	})
	sp.OnKeyChord(func(e events.Event) {
		if sp.IsReadOnly() {
			return
		}
		kf := keyfun.Of(e.KeyChord())
		if DebugSettings.KeyEventTrace {
			slog.Info("Spinner KeyChordEvent", "widget", sp, "keyfun", kf)
		}
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
