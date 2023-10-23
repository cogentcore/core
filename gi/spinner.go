// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"log/slog"
	"strconv"

	"goki.dev/girl/abilities"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// Spinner combines a TextField with up / down buttons for incrementing /
// decrementing values -- all configured within the Parts of the widget
type Spinner struct { //goki:embedder
	WidgetBase

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

	// icon to use for up button -- defaults to
	UpIcon icons.Icon `view:"show-name"`

	// icon to use for down button -- defaults to
	DownIcon icons.Icon `view:"show-name"`
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
	sp.UpIcon = fr.UpIcon
	sp.DownIcon = fr.DownIcon
}

func (sp *Spinner) OnInit() {
	sp.Step = 0.1
	sp.PageStep = 0.2
	sp.Max = 1.0
	sp.Prec = 6
	sp.HandleSpinnerEvents()
	sp.SpinnerStyles()
}

func (sp *Spinner) SpinnerStyles() {
	sp.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Focusable)
		// our parts take responsibility for their own state layers
		s.StateLayer = 0
	})
	sp.OnWidgetAdded(func(w Widget) {
		switch w.PathFrom(sp.This()) {
		case "parts/parts":
			w.Style(func(s *styles.Style) {
				s.AlignV = styles.AlignMiddle
			})
		case "parts/text-field":
			tf := w.(*TextField)
			tf.SetText(sp.ValToString(sp.Value))
			sp.TextFieldHandlers(tf)
			tf.Style(func(s *styles.Style) {
				tf.SetState(sp.IsReadOnly(), states.ReadOnly)
				s.SetMinPrefWidth(units.Em(3))
			})
			tf.OnSelect(func(e events.Event) {
				sp.HandleEvent(e) // pass up
			})
		case "parts/up":
			up := w.(*Button)
			up.Type = ButtonAction
			if sp.UpIcon.IsNil() {
				sp.UpIcon = icons.KeyboardArrowUp
			}
			up.SetIcon(sp.UpIcon)
			w.SetState(sp.IsReadOnly(), states.Disabled)
			up.OnClick(func(e events.Event) {
				sp.IncrValue(1)
			})
			up.OnSelect(func(e events.Event) {
				sp.HandleEvent(e) // pass up
			})
			up.Style(func(s *styles.Style) {
				s.SetAbilities(false, abilities.Focusable)
				s.Font.Size.SetDp(18)
				s.Padding.Set(units.Dp(1))
			})
		case "parts/down":
			down := w.(*Button)
			down.Type = ButtonAction
			if sp.DownIcon.IsNil() {
				sp.DownIcon = icons.KeyboardArrowDown
			}
			down.SetIcon(sp.DownIcon)
			w.SetState(sp.IsReadOnly(), states.Disabled)
			down.OnClick(func(e events.Event) {
				sp.IncrValue(-1)
			})
			down.OnSelect(func(e events.Event) {
				sp.HandleEvent(e) // pass up
			})
			down.Style(func(s *styles.Style) {
				s.SetAbilities(false, abilities.Focusable)
				s.Font.Size.SetDp(18)
				s.Padding.Set(units.Dp(1))
			})
		}
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
	tf := sp.TextField()
	if tf != nil {
		tf.SetText(sp.ValToString(sp.Value))
	}
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

func (sp *Spinner) ConfigParts(sc *Scene) {
	config := ki.Config{}
	config.Add(ButtonType, "down")
	config.Add(TextFieldType, "text-field")
	config.Add(ButtonType, "up")
	sp.ConfigPartsImpl(sc, config, LayoutHoriz)
}

func (sp *Spinner) TextField() *TextField {
	if sp.Parts == nil {
		return nil
	}
	tf, ok := sp.Parts.ChildByName("text-field", 1).(*TextField)
	if !ok {
		return nil
	}
	return tf
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
	var fval float32
	var err error
	if sp.FormatIsInt() {
		var iv int64
		iv, err = strconv.ParseInt(str, 0, 64)
		fval = float32(iv)
	} else {
		var fv float64
		fv, err = strconv.ParseFloat(str, 32)
		fval = float32(fv)
	}
	if err != nil {
		log.Println(err)
	}
	return fval, err
}

func (sp *Spinner) HandleSpinnerEvents() {
	sp.HandleWidgetEvents()
	sp.HandleSelectToggle()
	sp.HandleSpinnerScroll()
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

// TextFieldHandlers adds the Spinner textfield handlers for the given textfield
func (sp *Spinner) TextFieldHandlers(tf *TextField) {
	tf.On(events.Select, func(e events.Event) {
		if sp.IsReadOnly() {
			return
		}
		sp.SetSelected(!sp.StateIs(states.Selected))
		sp.Send(events.Select, e)
	})
	// TODO(kai): improve spin box focus handling
	// tf.OnClick(func(e events.Event) {
	// 	if sp.IsReadOnly() {
	// 		return
	// 	}
	// 	sp.SetState(true, states.Focused)
	// 	sp.HandleEvent(e)
	// })
	tf.OnChange(func(e events.Event) {
		text := tf.Text()
		val, err := sp.StringToVal(text)
		if err != nil {
			// TODO: use validation
			slog.Error("invalid Spinner value", "value", text, "err", err)
			return
		}
		sp.SetValueAction(val)
	})
	tf.OnKeyChord(func(e events.Event) {
		if sp.IsReadOnly() {
			return
		}
		if KeyEventTrace {
			fmt.Printf("Spinner KeyChordEvent: %v\n", sp.Path())
		}
		kf := KeyFun(e.KeyChord())
		switch {
		case kf == KeyFunMoveUp:
			e.SetHandled()
			sp.IncrValue(1)
		case kf == KeyFunMoveDown:
			e.SetHandled()
			sp.IncrValue(-1)
		case kf == KeyFunPageUp:
			e.SetHandled()
			sp.PageIncrValue(1)
		case kf == KeyFunPageDown:
			e.SetHandled()
			sp.PageIncrValue(-1)
		}
	})
	// Spinner always gives its focus to textfield
	sp.OnFocus(func(e events.Event) {
		tf.GrabFocus()
		tf.Send(events.Focus, e) // sets focused flag
	})
}

func (sp *Spinner) ConfigWidget(sc *Scene) {
	sp.ConfigParts(sc)
}

func (sp *Spinner) ApplyStyle(sc *Scene) {
	sp.WidgetBase.ApplyStyle(sc)
}

func (sp *Spinner) DoLayout(sc *Scene, parBBox image.Rectangle, iter int) bool {
	sp.DoLayoutBase(sc, parBBox, iter)
	sp.DoLayoutParts(sc, parBBox, iter)
	return sp.DoLayoutChildren(sc, iter)
}

func (sp *Spinner) Render(sc *Scene) {
	if sp.PushBounds(sc) {
		tf := sp.TextField()
		if tf != nil {
			tf.SetSelected(sp.StateIs(states.Selected))
		}
		sp.RenderChildren(sc)
		sp.RenderParts(sc)
		sp.PopBounds(sc)
	}
}
