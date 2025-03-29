// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"log/slog"
	"reflect"
	"strconv"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
)

// Spinner is a [TextField] for editing numerical values. It comes with
// fields, methods, buttons, and shortcuts to enhance numerical value editing.
type Spinner struct {
	TextField

	// Value is the current value.
	Value float32 `set:"-"`

	// HasMin is whether there is a minimum value to enforce.
	// It should be set using [Spinner.SetMin].
	HasMin bool `set:"-"`

	// Min, if [Spinner.HasMin] is true, is the the minimum value in range.
	// It should be set using [Spinner.SetMin].
	Min float32 `set:"-"`

	// HaxMax is whether there is a maximum value to enforce.
	// It should be set using [Spinner.SetMax].
	HasMax bool `set:"-"`

	// Max, if [Spinner.HasMax] is true, is the maximum value in range.
	// It should be set using [Spinner.SetMax].
	Max float32 `set:"-"`

	// Step is the amount that the up and down buttons and arrow keys
	// increment/decrement the value by. It defaults to 0.1.
	Step float32

	// EnforceStep is whether to ensure that the value of the spinner
	// is always a multiple of [Spinner.Step].
	EnforceStep bool

	// PageStep is the amount that the PageUp and PageDown keys
	// increment/decrement the value by.
	// It defaults to 0.2, and will be at least as big as [Spinner.Step].
	PageStep float32

	// Precision specifies the precision of decimal places
	// (total, not after the decimal point) to use in
	// representing the number. This helps to truncate
	// small weird floating point values.
	Precision int

	// Format is the format string to use for printing the value.
	// If it unset, %g is used. If it is decimal based
	// (ends in d, b, c, o, O, q, x, X, or U) then the value is
	// converted to decimal prior to printing.
	Format string
}

func (sp *Spinner) WidgetValue() any { return &sp.Value }

func (sp *Spinner) SetWidgetValue(value any) error {
	f, err := reflectx.ToFloat32(value)
	if err != nil {
		return err
	}
	sp.SetValue(f)
	return nil
}

func (sp *Spinner) OnBind(value any, tags reflect.StructTag) {
	kind := reflectx.NonPointerType(reflect.TypeOf(value)).Kind()
	if kind >= reflect.Int && kind <= reflect.Uintptr {
		sp.SetStep(1).SetEnforceStep(true)
		if kind >= reflect.Uint {
			sp.SetMin(0)
		}
	}
	if f, ok := tags.Lookup("format"); ok {
		sp.SetFormat(f)
	}
	setFromTag(tags, "min", func(v float32) { sp.SetMin(v) })
	setFromTag(tags, "max", func(v float32) { sp.SetMax(v) })
	setFromTag(tags, "step", func(v float32) { sp.SetStep(v) })
}

func (sp *Spinner) Init() {
	sp.TextField.Init()
	sp.SetStep(0.1).SetPageStep(0.2).SetPrecision(6).SetFormat("%g")
	sp.SetLeadingIcon(icons.Remove, func(e events.Event) {
		sp.incrementValue(-1)
	}).SetTrailingIcon(icons.Add, func(e events.Event) {
		sp.incrementValue(1)
	})
	sp.Updater(sp.setTextToValue)
	sp.Styler(func(s *styles.Style) {
		s.SetTextWrap(false)
		s.VirtualKeyboard = styles.KeyboardNumber
		if sp.IsReadOnly() {
			s.Min.X.Ch(6)
			s.Max.X.Ch(14)
		} else {
			s.Min.X.Ch(14)
			s.Max.X.Ch(22)
		}
		// s.Text.Align = styles.End // this doesn't work
	})

	sp.On(events.Scroll, func(e events.Event) {
		if sp.IsReadOnly() || !sp.StateIs(states.Focused) {
			return
		}
		se := e.(*events.MouseScroll)
		se.SetHandled()
		sp.incrementValue(float32(se.Delta.Y))
	})
	sp.SetValidator(func() error {
		text := sp.Text()
		val, err := sp.stringToValue(text)
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
		kf := keymap.Of(e.KeyChord())
		if DebugSettings.KeyEventTrace {
			slog.Info("Spinner KeyChordEvent", "widget", sp, "keyFunction", kf)
		}
		switch {
		case kf == keymap.MoveUp:
			e.SetHandled()
			sp.incrementValue(1)
		case kf == keymap.MoveDown:
			e.SetHandled()
			sp.incrementValue(-1)
		case kf == keymap.PageUp:
			e.SetHandled()
			sp.pageIncrementValue(1)
		case kf == keymap.PageDown:
			e.SetHandled()
			sp.pageIncrementValue(-1)
		}
	})

	i := func(w *Button) {
		w.Styler(func(s *styles.Style) {
			// icons do not get separate focus, as people can
			// use the arrow keys to get the same effect
			s.SetAbilities(false, abilities.Focusable)
			s.SetAbilities(true, abilities.RepeatClickable)
		})
	}
	sp.Maker(func(p *tree.Plan) {
		if sp.IsReadOnly() {
			return
		}
		tree.AddInit(p, "lead-icon", i)
		tree.AddInit(p, "trail-icon", i)
	})
}

func (sp *Spinner) setTextToValue() {
	sp.SetText(sp.valueToString(sp.Value))
}

// SetMin sets the minimum bound on the value.
func (sp *Spinner) SetMin(min float32) *Spinner {
	sp.HasMin = true
	sp.Min = min
	return sp
}

// SetMax sets the maximum bound on the value.
func (sp *Spinner) SetMax(max float32) *Spinner {
	sp.HasMax = true
	sp.Max = max
	return sp
}

// SetValue sets the value, enforcing any limits, and updates the display.
func (sp *Spinner) SetValue(val float32) *Spinner {
	sp.Value = val
	if sp.HasMax && sp.Value > sp.Max {
		sp.Value = sp.Max
	} else if sp.HasMin && sp.Value < sp.Min {
		sp.Value = sp.Min
	}
	sp.Value = math32.Truncate(sp.Value, sp.Precision)
	if sp.EnforceStep {
		// round to the nearest step
		sp.Value = sp.Step * math32.Round(sp.Value/sp.Step)
	}
	sp.setTextToValue()
	sp.NeedsRender()
	return sp
}

// setValueEvent calls SetValue and also sends a change event.
func (sp *Spinner) setValueEvent(val float32) *Spinner {
	sp.SetValue(val)
	sp.SendChange()
	return sp
}

// incrementValue increments the value by given number of steps (+ or -),
// and enforces it to be an even multiple of the step size (snap-to-value),
// and sends a change event.
func (sp *Spinner) incrementValue(steps float32) *Spinner {
	if sp.IsReadOnly() {
		return sp
	}
	val := sp.Value + steps*sp.Step
	val = sp.wrapAround(val)
	return sp.setValueEvent(val)
}

// pageIncrementValue increments the value by given number of page steps (+ or -),
// and enforces it to be an even multiple of the step size (snap-to-value),
// and sends a change event.
func (sp *Spinner) pageIncrementValue(steps float32) *Spinner {
	if sp.IsReadOnly() {
		return sp
	}
	if sp.PageStep < sp.Step {
		sp.PageStep = 2 * sp.Step
	}
	val := sp.Value + steps*sp.PageStep
	val = sp.wrapAround(val)
	return sp.setValueEvent(val)
}

// wrapAround, if the spinner has a min and a max, converts values less
// than min to max and values greater than max to min.
func (sp *Spinner) wrapAround(val float32) float32 {
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

// formatIsInt returns true if the format string requires an integer value
func (sp *Spinner) formatIsInt() bool {
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

// valueToString converts the value to the string representation thereof
func (sp *Spinner) valueToString(val float32) string {
	if sp.formatIsInt() {
		return fmt.Sprintf(sp.Format, int64(val))
	}
	return fmt.Sprintf(sp.Format, val)
}

// stringToValue converts the string field back to float value
func (sp *Spinner) stringToValue(str string) (float32, error) {
	if sp.Format == "" {
		f64, err := strconv.ParseFloat(str, 32)
		return float32(f64), err
	}

	var err error
	if sp.formatIsInt() {
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

func (sp *Spinner) WidgetTooltip(pos image.Point) (string, image.Point) {
	res, rpos := sp.TextField.WidgetTooltip(pos)
	if sp.error != nil {
		return res, rpos
	}
	if sp.HasMin {
		if res != "" {
			res += " "
		}
		res += "(minimum: " + sp.valueToString(sp.Min)
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
		res += "maximum: " + sp.valueToString(sp.Max) + ")"
	}
	return res, rpos
}
