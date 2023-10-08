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

	"goki.dev/girl/states"
	"goki.dev/girl/styles"
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

	// [view: show-name] icon to use for up button -- defaults to icons.KeyboardArrowUp
	UpIcon icons.Icon `view:"show-name" desc:"icon to use for up button -- defaults to icons.KeyboardArrowUp"`

	// [view: show-name] icon to use for down button -- defaults to icons.KeyboardArrowDown
	DownIcon icons.Icon `view:"show-name" desc:"icon to use for down button -- defaults to icons.KeyboardArrowDown"`
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
}

func (sb *SpinBox) OnChildAdded(child ki.Ki) {
	w, _ := AsWidget(child)
	switch w.Name() {
	case "parts":
		w.AddStyles(func(s *styles.Style) {
			s.AlignV = styles.AlignMiddle
		})
	case "text-field":
		w.AddStyles(func(s *styles.Style) {
			s.MinWidth.SetEm(6)
		})
	case "space":
		w.AddStyles(func(s *styles.Style) {
			s.Width.SetCh(0.1)
		})
	case "buttons":
		w.AddStyles(func(s *styles.Style) {
			s.AlignV = styles.AlignMiddle
		})
	case "up", "down", "but0", "but1": // TODO: maybe fix this? (OnChildAdded is called with SetNChildren, so before actual names)
		act := w.(*Action)
		act.Type = ActionParts
		act.AddStyles(func(s *styles.Style) {
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
	parts := sb.NewParts(LayoutHoriz)
	if sb.UpIcon.IsNil() {
		sb.UpIcon = icons.KeyboardArrowUp
	}
	if sb.DownIcon.IsNil() {
		sb.DownIcon = icons.KeyboardArrowDown
	}
	config := ki.Config{}
	config.Add(TextFieldType, "text-field")
	if !sb.IsDisabled() {
		config.Add(SpaceType, "space")
		config.Add(LayoutType, "buttons")
	}
	mods, updt := parts.ConfigChildren(config)
	if !mods && !sb.NeedsRebuild() {
		parts.UpdateEnd(updt)
		return
	}
	if !sb.IsDisabled() {
		// STYTODO: maybe do some of this config in OnChildAdded?
		buts := parts.ChildByName("buttons", 1).(*Layout)
		buts.Lay = LayoutVert
		buts.SetNChildren(2, ActionType, "but")
		// up
		up := buts.Child(0).(*Action)
		up.SetName("up")
		up.SetProp("no-focus", true) // note: cannot be in compiled props b/c
		// not compiled into style prop
		// up.SetFlagState(sb.IsInactive(), int(Inactive))
		up.SetIcon(sb.UpIcon)
		// todo:
		up.On(events.Click, func(e events.Event) {
			sb.IncrValue(1.0)
		})
		// dn
		dn := buts.Child(1).(*Action)
		// dn.SetFlagState(sb.IsInactive(), int(Inactive))
		dn.SetName("down")
		dn.SetProp("no-focus", true)
		dn.Icon = sb.DownIcon
		dn.On(events.Click, func(e events.Event) {
			sb.IncrValue(-1.0)
		})
		// space
		// sp := parts.ChildByName("space", 2).(*Space)
	}
	// text-field
	tf := parts.ChildByName("text-field", 0).(*TextField)
	tf.SetState(sb.IsDisabled(), states.Disabled)
	tf.Txt = sb.ValToString(sb.Value)
	if !sb.IsDisabled() {
		sb.SpinBoxTextFieldHandlers(tf)
	}
	parts.UpdateEnd(updt)
	sb.SetNeedsLayout(sc, updt)
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
	sb.SpinBoxKeys()
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

// todo: how to deal with this??
func (sb *SpinBox) SpinBoxTextFieldHandlers(tf *TextField) {
	tf.On(events.Select, func(e events.Event) {
		sb.SetSelected(!sb.StateIs(states.Selected))
		sb.Send(events.Select, nil)
	})
	tf.On(events.Click, func(e events.Event) {
		fmt.Println("sb tf click")
		sb.SetState(true, states.Focused)
		sb.HandleEvent(e)
	})
}

func (sb *SpinBox) SpinBoxKeys() {
	sb.On(events.KeyChord, func(e events.Event) {
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
