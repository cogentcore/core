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

// SpinBox combines a TextField with up / down buttons for incrementing /
// decrementing values -- all configured within the Parts of the widget
//
//goki:embedder
type SpinBox struct {
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

func (sb *SpinBox) CopyFieldsFrom(frm any) {
	fr := frm.(*SpinBox)
	sb.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	sb.Value = fr.Value
	sb.HasMin = fr.HasMin
	sb.Min = fr.Min
	sb.HasMax = fr.HasMax
	sb.Max = fr.Max
	sb.Step = fr.Step
	sb.PageStep = fr.PageStep
	sb.Prec = fr.Prec
	sb.UpIcon = fr.UpIcon
	sb.DownIcon = fr.DownIcon
}

func (sb *SpinBox) OnInit() {
	sb.SpinBoxHandlers()
	sb.SpinBoxStyles()
}

func (sb *SpinBox) SpinBoxStyles() {
	sb.Step = 0.1
	sb.PageStep = 0.2
	sb.Max = 1.0
	sb.Prec = 6
	sb.AddStyles(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Focusable)
	})
}

func (sb *SpinBox) OnChildAdded(child ki.Ki) {
	w, _ := AsWidget(child)
	switch w.Name() {
	case "parts":
		w.AddStyles(func(s *styles.Style) {
			s.AlignV = styles.AlignMiddle
		})
	case "text-field":
		tf := w.(*TextField)
		tf.SetText(sb.ValToString(sb.Value))
		tf.SetState(sb.IsDisabled(), states.Disabled)
		sb.TextFieldHandlers(tf)
		tf.AddStyles(func(s *styles.Style) {
			s.SetMinPrefWidth(units.Em(3))
		})
	case "up":
		up := w.(*Button)
		up.Type = ButtonAction
		if sb.UpIcon.IsNil() {
			sb.UpIcon = icons.Add
		}
		up.SetIcon(sb.UpIcon)
		w.SetState(sb.IsDisabled(), states.Disabled)
		up.OnClick(func(e events.Event) {
			sb.IncrValue(1)
		})
		up.AddStyles(func(s *styles.Style) {
			s.SetAbilities(false, abilities.Focusable)
			s.Font.Size.SetDp(18)
		})
	case "down":
		down := w.(*Button)
		down.Type = ButtonAction
		if sb.DownIcon.IsNil() {
			sb.DownIcon = icons.Remove
		}
		down.SetIcon(sb.DownIcon)
		w.SetState(sb.IsDisabled(), states.Disabled)
		down.OnClick(func(e events.Event) {
			sb.IncrValue(-1)
		})
		down.AddStyles(func(s *styles.Style) {
			s.SetAbilities(false, abilities.Focusable)
			s.Font.Size.SetDp(18)
		})
	}
}

// SetMin sets the min limits on the value
func (sb *SpinBox) SetMin(min float32) *SpinBox {
	sb.HasMin = true
	sb.Min = min
	return sb
}

// SetMax sets the max limits on the value
func (sb *SpinBox) SetMax(max float32) *SpinBox {
	sb.HasMax = true
	sb.Max = max
	return sb
}

// SetMax sets the format of the spin box
func (sb *SpinBox) SetFormat(format string) *SpinBox {
	sb.Format = format
	return sb
}

// SetMinMax sets the min and max limits on the value
func (sb *SpinBox) SetMinMax(hasMin bool, min float32, hasMax bool, max float32) *SpinBox {
	sb.HasMin = hasMin
	sb.Min = min
	sb.HasMax = hasMax
	sb.Max = max
	if sb.Max < sb.Min {
		slog.Warn("gi.SpinBox.SetMinMax: max was less than min; disabling limits")
		sb.HasMax = false
		sb.HasMin = false
	}
	return sb
}

// SetStep sets the step (increment) value of the spinbox
func (sb *SpinBox) SetStep(step float32) *SpinBox {
	sb.Step = step
	return sb
}

// SetValue sets the value, enforcing any limits, and updates the display
func (sb *SpinBox) SetValue(val float32) *SpinBox {
	updt := sb.UpdateStart()
	defer sb.UpdateEndRender(updt)
	sb.Value = val
	if sb.HasMax {
		sb.Value = mat32.Min(sb.Value, sb.Max)
	}
	if sb.HasMin {
		sb.Value = mat32.Max(sb.Value, sb.Min)
	}
	sb.Value = mat32.Truncate(sb.Value, sb.Prec)
	tf := sb.TextField()
	if tf != nil {
		tf.SetText(sb.ValToString(sb.Value))
	}
	return sb
}

// SetValueAction calls SetValue and also emits the signal
func (sb *SpinBox) SetValueAction(val float32) *SpinBox {
	sb.SetValue(val)
	sb.Send(events.Change, nil)
	return sb
}

// IncrValue increments the value by given number of steps (+ or -),
// and enforces it to be an even multiple of the step size (snap-to-value),
// and emits the signal
func (sb *SpinBox) IncrValue(steps float32) *SpinBox {
	val := sb.Value + steps*sb.Step
	val = mat32.IntMultiple(val, sb.Step)
	return sb.SetValueAction(val)
}

// PageIncrValue increments the value by given number of page steps (+ or -),
// and enforces it to be an even multiple of the step size (snap-to-value),
// and emits the signal
func (sb *SpinBox) PageIncrValue(steps float32) *SpinBox {
	val := sb.Value + steps*sb.PageStep
	val = mat32.IntMultiple(val, sb.PageStep)
	return sb.SetValueAction(val)
}

func (sb *SpinBox) ConfigParts(sc *Scene) {
	config := ki.Config{}
	config.Add(ButtonType, "down")
	config.Add(TextFieldType, "text-field")
	config.Add(ButtonType, "up")
	sb.ConfigPartsImpl(sc, config, LayoutHoriz)
}

func (sb *SpinBox) TextField() *TextField {
	tf, ok := sb.Parts.ChildByName("text-field", 1).(*TextField)
	if !ok {
		return nil
	}
	return tf
}

// FormatIsInt returns true if the format string requires an integer value
func (sb *SpinBox) FormatIsInt() bool {
	if sb.Format == "" {
		return false
	}
	fc := sb.Format[len(sb.Format)-1]
	switch fc {
	case 'd', 'b', 'c', 'o', 'O', 'q', 'x', 'X', 'U':
		return true
	}
	return false
}

// ValToString converts the value to the string representation thereof
func (sb *SpinBox) ValToString(val float32) string {
	if sb.Format == "" {
		return fmt.Sprintf("%g", val)
	}
	if sb.FormatIsInt() {
		return fmt.Sprintf(sb.Format, int64(val))
	}
	return fmt.Sprintf(sb.Format, val)
}

// StringToVal converts the string field back to float value
func (sb *SpinBox) StringToVal(str string) (float32, error) {
	var fval float32
	var err error
	if sb.FormatIsInt() {
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

func (sb *SpinBox) SpinBoxHandlers() {
	sb.WidgetHandlers()
	sb.SpinBoxScroll()
}

func (sb *SpinBox) SpinBoxScroll() {
	sb.On(events.Scroll, func(e events.Event) {
		if sb.StateIs(states.Disabled) || !sb.StateIs(states.Focused) {
			return
		}
		se := e.(*events.MouseScroll)
		se.SetHandled()
		sb.IncrValue(float32(se.DimDelta(mat32.Y)))
	})
}

// TextFieldHandlers adds the spinbox textfield handlers for the given textfield
func (sb *SpinBox) TextFieldHandlers(tf *TextField) {
	tf.On(events.Select, func(e events.Event) {
		if sb.IsDisabled() {
			return
		}
		sb.SetSelected(!sb.StateIs(states.Selected))
		sb.Send(events.Select, e)
	})
	// TODO(kai): improve spin box focus handling
	// tf.OnClick(func(e events.Event) {
	// 	if sb.IsDisabled() {
	// 		return
	// 	}
	// 	sb.SetState(true, states.Focused)
	// 	sb.HandleEvent(e)
	// })
	tf.OnChange(func(e events.Event) {
		text := tf.Text()
		val, err := sb.StringToVal(text)
		if err != nil {
			// TODO: use validation
			slog.Error("invalid spinbox value", "value", text, "err", err)
			return
		}
		sb.SetValueAction(val)
	})
	tf.OnKeyChord(func(e events.Event) {
		if sb.StateIs(states.Disabled) {
			return
		}
		if KeyEventTrace {
			fmt.Printf("SpinBox KeyChordEvent: %v\n", sb.Path())
		}
		kf := KeyFun(e.KeyChord())
		switch {
		case kf == KeyFunMoveUp:
			e.SetHandled()
			sb.IncrValue(1)
		case kf == KeyFunMoveDown:
			e.SetHandled()
			sb.IncrValue(-1)
		case kf == KeyFunPageUp:
			e.SetHandled()
			sb.PageIncrValue(1)
		case kf == KeyFunPageDown:
			e.SetHandled()
			sb.PageIncrValue(-1)
		}
	})
	// spinbox always gives its focus to textfield
	sb.On(events.Focus, func(e events.Event) {
		tf.GrabFocus()
		tf.Send(events.Focus, e) // sets focused flag
	})
}

func (sb *SpinBox) ConfigWidget(sc *Scene) {
	sb.ConfigParts(sc)
}

func (sb *SpinBox) GetSize(sc *Scene, iter int) {
	sb.GetSizeParts(sc, iter)
}

func (sb *SpinBox) DoLayout(sc *Scene, parBBox image.Rectangle, iter int) bool {
	sb.DoLayoutBase(sc, parBBox, iter)
	sb.DoLayoutParts(sc, parBBox, iter)
	return sb.DoLayoutChildren(sc, iter)
}

func (sb *SpinBox) Render(sc *Scene) {
	if sb.PushBounds(sc) {
		tf := sb.Parts.ChildByName("text-field", 2).(*TextField)
		tf.SetSelected(sb.StateIs(states.Selected))
		sb.RenderChildren(sc)
		sb.RenderParts(sc)
		sb.PopBounds(sc)
	}
}
