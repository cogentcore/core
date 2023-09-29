// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"strconv"

	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/goosi"
	"goki.dev/goosi/key"
	"goki.dev/goosi/mouse"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/laser"
	"goki.dev/mat32/v2"
)

type SpinBoxEmbedder interface {
	AsSpinBox() *SpinBox
}

func AsSpinBox(k ki.Ki) *SpinBox {
	if k == nil || k.This() == nil {
		return nil
	}
	if ac, ok := k.(SpinBoxEmbedder); ok {
		return ac.AsSpinBox()
	}
	return nil
}

func (ac *SpinBox) AsSpinBox() *SpinBox {
	return ac
}

// SpinBox combines a TextField with up / down buttons for incrementing /
// decrementing values -- all configured within the Parts of the widget
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

	// [view: -] signal for spin box -- has no signal types, just emitted when the value changes
	// SpinBoxSig ki.Signal `copy:"-" json:"-" xml:"-" view:"-" desc:"signal for spin box -- has no signal types, just emitted when the value changes"`
}

// event functions for this type
var SpinBoxEventFuncs WidgetEvents

func (sb *SpinBox) OnInit() {
	sb.AddEvents(&SpinBoxEventFuncs)
	sb.Step = 0.1
	sb.PageStep = 0.2
	sb.Max = 1.0
	sb.Prec = 6
}

func (sb *SpinBox) OnChildAdded(child ki.Ki) {
	if _, wb := AsWidget(child); wb != nil {
		switch wb.Name() {
		case "Parts":
			wb.AddStyler(func(w *WidgetBase, s *styles.Style) {
				s.AlignV = styles.AlignMiddle
			})
		case "text-field":
			wb.AddStyler(func(w *WidgetBase, s *styles.Style) {
				s.MinWidth.SetEm(6)
			})
		case "space":
			wb.AddStyler(func(w *WidgetBase, s *styles.Style) {
				s.Width.SetCh(0.1)
			})
		case "buttons":
			wb.AddStyler(func(w *WidgetBase, s *styles.Style) {
				s.AlignV = styles.AlignMiddle
			})
		case "up", "down", "but0", "but1": // TODO: maybe fix this? (OnChildAdded is called with SetNChildren, so before actual names)
			act := child.(*Action)
			act.Type = ActionParts
			act.AddStyler(func(w *WidgetBase, s *styles.Style) {
				s.Font.Size.SetPx(18)
			})
		}
	}

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

// SetMin sets the min limits on the value
func (sb *SpinBox) SetMin(min float32) {
	sb.HasMin = true
	sb.Min = min
}

// SetMax sets the max limits on the value
func (sb *SpinBox) SetMax(max float32) {
	sb.HasMax = true
	sb.Max = max
}

// SetMinMax sets the min and max limits on the value
func (sb *SpinBox) SetMinMax(hasMin bool, min float32, hasMax bool, max float32) {
	sb.HasMin = hasMin
	sb.Min = min
	sb.HasMax = hasMax
	sb.Max = max
	if sb.Max < sb.Min {
		log.Printf("gi.SpinBox SetMinMax: max was less than min -- disabling limits\n")
		sb.HasMax = false
		sb.HasMin = false
	}
}

// SetValue sets the value, enforcing any limits, and updates the display
func (sb *SpinBox) SetValue(val float32) {
	updt := sb.UpdateStart()
	defer sb.UpdateEnd(updt)
	sb.Value = val
	if sb.HasMax {
		sb.Value = mat32.Min(sb.Value, sb.Max)
	}
	if sb.HasMin {
		sb.Value = mat32.Max(sb.Value, sb.Min)
	}
	sb.Value = mat32.Truncate(sb.Value, sb.Prec)
}

// SetValueAction calls SetValue and also emits the signal
func (sb *SpinBox) SetValueAction(val float32) {
	sb.SetValue(val)
	// sb.SpinBoxSig.Emit(sb.This(), 0, sb.Value)
}

// IncrValue increments the value by given number of steps (+ or -),
// and enforces it to be an even multiple of the step size (snap-to-value),
// and emits the signal
func (sb *SpinBox) IncrValue(steps float32) {
	val := sb.Value + steps*sb.Step
	val = mat32.IntMultiple(val, sb.Step)
	sb.SetValueAction(val)
}

// PageIncrValue increments the value by given number of page steps (+ or -),
// and enforces it to be an even multiple of the step size (snap-to-value),
// and emits the signal
func (sb *SpinBox) PageIncrValue(steps float32) {
	val := sb.Value + steps*sb.PageStep
	val = mat32.IntMultiple(val, sb.PageStep)
	sb.SetValueAction(val)
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
	if !mods && !styles.RebuildDefaultStyles {
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
		up.ActionSig.ConnectOnly(sb.This(), func(recv, send ki.Ki, sig int64, data any) {
			sbb := AsSpinBox(recv)
			sbb.IncrValue(1.0)
		})
		// dn
		dn := buts.Child(1).(*Action)
		// dn.SetFlagState(sb.IsInactive(), int(Inactive))
		dn.SetName("down")
		dn.SetProp("no-focus", true)
		dn.Icon = sb.DownIcon
		dn.ActionSig.ConnectOnly(sb.This(), func(recv, send ki.Ki, sig int64, data any) {
			sbb := AsSpinBox(recv)
			sbb.IncrValue(-1.0)
		})
		// space
		// sp := parts.ChildByName("space", 2).(*Space)
	}
	// text-field
	tf := parts.ChildByName("text-field", 0).(*TextField)
	tf.SetFlag(sb.IsDisabled(), Disabled)
	tf.Txt = sb.ValToString(sb.Value)
	if !sb.IsDisabled() {
		tf.TextFieldSig.ConnectOnly(sb.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(TextFieldDone) || sig == int64(TextFieldDeFocused) {
				sbb := AsSpinBox(recv)
				tf := send.(*TextField)
				vl, err := sb.StringToVal(tf.Text())
				if err == nil {
					sbb.SetValueAction(vl)
				}
			}
		})
	}
	sb.UpdateEnd(updt)

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

func (sb *SpinBox) AddEvents(we *WidgetEvents) {
	if we.HasFuncs() {
		return
	}
	sb.SpinBoxEvents(we)
}

func (sb *SpinBox) FilterEvents() {
	sb.Events.CopyFrom(&SpinBoxEventFuncs)
}

func (sb *SpinBox) SpinBoxEvents(we *WidgetEvents) {
	sb.HoverTooltipEvent(we)
	sb.MouseScrollEvent(we)
	// sb.TextFieldEvent(we)
	sb.KeyChordEvent(we)
}

func (sb *SpinBox) MouseScrollEvent(we *WidgetEvents) {
	we.AddFunc(goosi.MouseScrollEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		sbb := AsSpinBox(recv)
		if sbb.IsDisabled() || !sbb.StateIs(states.Focused) {
			return
		}
		me := d.(*mouse.ScrollEvent)
		me.SetHandled()
		sbb.IncrValue(float32(me.NonZeroDelta(false)))
	})
}

// todo: how to deal with this??
func (sb *SpinBox) TextFieldEvent() {
	tf := sb.Parts.ChildByName("text-field", 0).(*TextField)
	tf.WidgetSig.ConnectOnly(sb.This(), func(recv, send ki.Ki, sig int64, data any) {
		sbb := AsSpinBox(recv)
		if sig == int64(WidgetSelected) {
			sbb.SetSelected(!sbb.StateIs(states.Selected))
		}
		sbb.WidgetSig.Emit(sbb.This(), sig, data) // passthrough
	})
}

func (sb *SpinBox) KeyChordEvent(we *WidgetEvents) {
	we.AddFunc(goosi.KeyChordEvent, HiPri, func(recv, send ki.Ki, sig int64, d any) {
		sbb := recv.(*SpinBox)
		if sbb.IsDisabled() {
			return
		}
		kt := d.(*key.Event)
		if KeyEventTrace {
			fmt.Printf("SpinBox KeyChordEvent: %v\n", sbb.Path())
		}
		kf := KeyFun(kt.Chord())
		switch {
		case kf == KeyFunMoveUp:
			kt.SetHandled()
			sb.IncrValue(1)
		case kf == KeyFunMoveDown:
			kt.SetHandled()
			sb.IncrValue(-1)
		case kf == KeyFunPageUp:
			kt.SetHandled()
			sb.PageIncrValue(1)
		case kf == KeyFunPageDown:
			kt.SetHandled()
			sb.PageIncrValue(-1)
		}
	})
}

func (sb *SpinBox) ConfigWidget(sc *Scene) {
	sb.ConfigParts(sc)
}

// StyleFromProps styles SpinBox-specific fields from ki.Prop properties
// doesn't support inherit or default
func (sb *SpinBox) StyleFromProps(props ki.Props, sc *Scene) {
	for key, val := range props {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		switch key {
		case "value":
			if iv, ok := laser.ToFloat32(val); ok {
				sb.Value = iv
			}
		case "min":
			if iv, ok := laser.ToFloat32(val); ok {
				sb.Min = iv
			}
		case "max":
			if iv, ok := laser.ToFloat32(val); ok {
				sb.Max = iv
			}
		case "step":
			if iv, ok := laser.ToFloat32(val); ok {
				sb.Step = iv
			}
		case "pagestep":
			if iv, ok := laser.ToFloat32(val); ok {
				sb.PageStep = iv
			}
		case "prec":
			if iv, ok := laser.ToInt(val); ok {
				sb.Prec = int(iv)
			}
		case "has-min":
			if bv, ok := laser.ToBool(val); ok {
				sb.HasMin = bv
			}
		case "has-max":
			if bv, ok := laser.ToBool(val); ok {
				sb.HasMax = bv
			}
		case "format":
			sb.Format = laser.ToString(val)
		}
	}
	if sb.PageStep < sb.Step { // often forget to set this..
		sb.PageStep = 10 * sb.Step
	}
}

// StyleSpinBox does spinbox styling -- sets StyMu Lock
func (sb *SpinBox) StyleSpinBox(sc *Scene) {
	sb.StyMu.Lock()
	defer sb.StyMu.Unlock()

	sb.ApplyStyleWidget(sc)
}

func (sb *SpinBox) ApplyStyle(sc *Scene) {
	sb.StyleSpinBox(sc)
	sb.ConfigParts(sc)
}

func (sb *SpinBox) GetSize(sc *Scene, iter int) {
	sb.GetSizeParts(sc, iter)
}

func (sb *SpinBox) DoLayout(sc *Scene, parBBox image.Rectangle, iter int) bool {
	sb.DoLayoutBase(sc, parBBox, true, iter) // init style
	sb.DoLayoutParts(sc, parBBox, iter)
	return sb.DoLayoutChildren(sc, iter)
}

func (sb *SpinBox) Render(sc *Scene) {
	wi := sb.This().(Widget)
	if sb.PushBounds(sc) {
		wi.FilterEvents()
		tf := sb.Parts.ChildByName("text-field", 2).(*TextField)
		tf.SetSelected(sb.StateIs(states.Selected))
		sb.RenderChildren(sc)
		sb.RenderParts(sc)
		sb.PopBounds(sc)
	}
}

// func (sb *SpinBox) StateIs(states.Focused) bool {
// 	if sb.IsDisabled() {
// 		return false
// 	}
// 	return sb.ContainsFocus() // needed for getting key events
// }
