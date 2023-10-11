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
//
//goki:embedder
type Spinner struct {
	WidgetBase

	// current value
	Value float32 `xml:"value" desc:"current value"`

	// is there a minimum value to enforce
	HasMin bool `xml:"has-min" desc:"is there a minimum value to enforce"`

	// minimum value in range
	Min float32 `xml:"min" desc:"minimum value in range"`

	// is there a maximumvalue to enforce
	HasMax bool `xml:"has-max" desc:"is there a maximumvalue to enforce"`

	// maximum value in range
	Max float32 `xml:"max" desc:"maximum value in range"`

	// smallest step size to increment
	Step float32 `xml:"step" desc:"smallest step size to increment"`

	// larger PageUp / Dn step size
	PageStep float32 `xml:"pagestep" desc:"larger PageUp / Dn step size"`

	// specifies the precision of decimal places (total, not after the decimal point) to use in representing the number -- this helps to truncate small weird floating point values in the nether regions
	Prec int `desc:"specifies the precision of decimal places (total, not after the decimal point) to use in representing the number -- this helps to truncate small weird floating point values in the nether regions"`

	// prop = format -- format string for printing the value -- blank defaults to %g.  If decimal based (ends in d, b, c, o, O, q, x, X, or U) then value is converted to decimal prior to printing
	Format string `xml:"format" desc:"prop = format -- format string for printing the value -- blank defaults to %g.  If decimal based (ends in d, b, c, o, O, q, x, X, or U) then value is converted to decimal prior to printing"`

	// [view: show-name] icon to use for up button -- defaults to [icons.Add]
	UpIcon icons.Icon `view:"show-name" desc:"icon to use for up button -- defaults to [icons.Add]"`

	// [view: show-name] icon to use for down button -- defaults to [icons.Remove]
	DownIcon icons.Icon `view:"show-name" desc:"icon to use for down button -- defaults to [icons.Remove]"`
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
	sp.SpinnerHandlers()
	sp.SpinnerStyles()
}

func (sp *Spinner) SpinnerStyles() {
	sp.Step = 0.1
	sp.PageStep = 0.2
	sp.Max = 1.0
	sp.Prec = 6
	sp.AddStyles(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Focusable)
	})
}

func (sp *Spinner) OnChildAdded(child ki.Ki) {
	w, _ := AsWidget(child)
	switch w.Name() {
	case "parts":
		w.AddStyles(func(s *styles.Style) {
			s.AlignV = styles.AlignMiddle
		})
	case "text-field":
		tf := w.(*TextField)
		tf.SetText(sp.ValToString(sp.Value))
		tf.SetState(sp.IsDisabled(), states.Disabled)
		sp.TextFieldHandlers(tf)
		tf.AddStyles(func(s *styles.Style) {
			s.SetMinPrefWidth(units.Em(3))
		})
	case "up":
		up := w.(*Button)
		up.Type = ButtonAction
		if sp.UpIcon.IsNil() {
			sp.UpIcon = icons.Add
		}
		up.SetIcon(sp.UpIcon)
		w.SetState(sp.IsDisabled(), states.Disabled)
		up.OnClick(func(e events.Event) {
			sp.IncrValue(1)
		})
		up.AddStyles(func(s *styles.Style) {
			s.SetAbilities(false, abilities.Focusable)
			s.Font.Size.SetDp(18)
		})
	case "down":
		down := w.(*Button)
		down.Type = ButtonAction
		if sp.DownIcon.IsNil() {
			sp.DownIcon = icons.Remove
		}
		down.SetIcon(sp.DownIcon)
		w.SetState(sp.IsDisabled(), states.Disabled)
		down.OnClick(func(e events.Event) {
			sp.IncrValue(-1)
		})
		down.AddStyles(func(s *styles.Style) {
			s.SetAbilities(false, abilities.Focusable)
			s.Font.Size.SetDp(18)
		})
	}
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

// SetMax sets the format of the spin box
func (sp *Spinner) SetFormat(format string) *Spinner {
	sp.Format = format
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

// SetStep sets the step (increment) value of the Spinner
func (sp *Spinner) SetStep(step float32) *Spinner {
	sp.Step = step
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
	val := sp.Value + steps*sp.Step
	val = mat32.IntMultiple(val, sp.Step)
	return sp.SetValueAction(val)
}

// PageIncrValue increments the value by given number of page steps (+ or -),
// and enforces it to be an even multiple of the step size (snap-to-value),
// and emits the signal
func (sp *Spinner) PageIncrValue(steps float32) *Spinner {
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

func (sp *Spinner) SpinnerHandlers() {
	sp.WidgetHandlers()
	sp.SpinnerScroll()
}

func (sp *Spinner) SpinnerScroll() {
	sp.On(events.Scroll, func(e events.Event) {
		if sp.StateIs(states.Disabled) || !sp.StateIs(states.Focused) {
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
		if sp.IsDisabled() {
			return
		}
		sp.SetSelected(!sp.StateIs(states.Selected))
		sp.Send(events.Select, e)
	})
	// TODO(kai): improve spin box focus handling
	// tf.OnClick(func(e events.Event) {
	// 	if sp.IsDisabled() {
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
		if sp.StateIs(states.Disabled) {
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

func (sp *Spinner) GetSize(sc *Scene, iter int) {
	sp.GetSizeParts(sc, iter)
}

func (sp *Spinner) DoLayout(sc *Scene, parBBox image.Rectangle, iter int) bool {
	sp.DoLayoutBase(sc, parBBox, iter)
	sp.DoLayoutParts(sc, parBBox, iter)
	return sp.DoLayoutChildren(sc, iter)
}

func (sp *Spinner) Render(sc *Scene) {
	if sp.PushBounds(sc) {
		tf := sp.Parts.ChildByName("text-field", 2).(*TextField)
		tf.SetSelected(sp.StateIs(states.Selected))
		sp.RenderChildren(sc)
		sp.RenderParts(sc)
		sp.PopBounds(sc)
	}
}
