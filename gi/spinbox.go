// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"strconv"

	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
	"goki.dev/gi/v2/gist"
	"goki.dev/gi/v2/icons"
	"goki.dev/gi/v2/oswin"
	"goki.dev/gi/v2/oswin/key"
	"goki.dev/gi/v2/oswin/mouse"
)

////////////////////////////////////////////////////////////////////////////////////////
// SpinBox

// SpinBox combines a TextField with up / down buttons for incrementing /
// decrementing values -- all configured within the Parts of the widget
type SpinBox struct {
	PartsWidgetBase

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
	SpinBoxSig ki.Signal `copy:"-" json:"-" xml:"-" view:"-" desc:"signal for spin box -- has no signal types, just emitted when the value changes"`
}

var TypeSpinBox = kit.Types.AddType(&SpinBox{}, SpinBoxProps)

// AddNewSpinBox adds a new spinbox to given parent node, with given name.
func AddNewSpinBox(parent ki.Ki, name string) *SpinBox {
	return parent.AddNewChild(TypeSpinBox, name).(*SpinBox)
}

func (sb *SpinBox) OnInit() {
	sb.Step = 0.1
	sb.PageStep = 0.2
	sb.Max = 1.0
	sb.Prec = 6
}

func (sb *SpinBox) OnChildAdded(child ki.Ki) {
	if w := KiAsWidget(child); w != nil {
		switch w.Name() {
		case "Parts":
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.AlignV = gist.AlignMiddle
			})
		case "text-field":
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.MinWidth.SetEm(6)
			})
		case "space":
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Width.SetCh(0.1)
			})
		case "buttons":
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.AlignV = gist.AlignMiddle
			})
		case "up", "down", "but0", "but1": // TODO: maybe fix this? (OnChildAdded is called with SetNChildren, so before actual names)
			act := child.(*Action)
			act.Type = ActionParts
			act.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Font.Size.SetPx(18)
			})
		}
	}

}

func (sb *SpinBox) CopyFieldsFrom(frm any) {
	fr := frm.(*SpinBox)
	sb.PartsWidgetBase.CopyFieldsFrom(&fr.PartsWidgetBase)
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

func (sb *SpinBox) Disconnect() {
	sb.PartsWidgetBase.Disconnect()
	sb.SpinBoxSig.DisconnectAll()
}

var SpinBoxProps = ki.Props{
	ki.EnumTypeFlag: TypeNodeFlags,
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
	sb.SpinBoxSig.Emit(sb.This(), 0, sb.Value)
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

func (sb *SpinBox) ConfigParts() {
	if sb.UpIcon.IsNil() {
		sb.UpIcon = icons.KeyboardArrowUp
	}
	if sb.DownIcon.IsNil() {
		sb.DownIcon = icons.KeyboardArrowDown
	}
	sb.Parts.Lay = LayoutHoriz
	if sb.Style.Template != "" {
		sb.Parts.Style.Template = sb.Style.Template + ".Parts"
	}
	config := kit.TypeAndNameList{}
	config.Add(TypeTextField, "text-field")
	if !sb.IsDisabled() {
		config.Add(TypeSpace, "space")
		config.Add(TypeLayout, "buttons")
	}
	mods, updt := sb.Parts.ConfigChildren(config)
	if mods || gist.RebuildDefaultStyles {
		if !sb.IsDisabled() {
			// STYTODO: maybe do some of this config in OnChildAdded?
			buts := sb.Parts.ChildByName("buttons", 1).(*Layout)
			buts.Lay = LayoutVert
			buts.SetNChildren(2, TypeAction, "but")
			// up
			up := buts.Child(0).(*Action)
			up.SetName("up")
			up.SetProp("no-focus", true) // note: cannot be in compiled props b/c
			// not compiled into style prop
			// up.SetFlagState(sb.IsInactive(), int(Inactive))
			up.Icon = sb.UpIcon
			if sb.Style.Template != "" {
				up.Style.Template = sb.Style.Template + ".up"
			}
			up.ActionSig.ConnectOnly(sb.This(), func(recv, send ki.Ki, sig int64, data any) {
				sbb := recv.Embed(TypeSpinBox).(*SpinBox)
				sbb.IncrValue(1.0)
			})
			// dn
			dn := buts.Child(1).(*Action)
			// dn.SetFlagState(sb.IsInactive(), int(Inactive))
			dn.SetName("down")
			dn.SetProp("no-focus", true)
			dn.Icon = sb.DownIcon
			if sb.Style.Template != "" {
				dn.Style.Template = sb.Style.Template + ".dn"
			}
			dn.ActionSig.ConnectOnly(sb.This(), func(recv, send ki.Ki, sig int64, data any) {
				sbb := recv.Embed(TypeSpinBox).(*SpinBox)
				sbb.IncrValue(-1.0)
			})
			// space
			sp := sb.Parts.ChildByName("space", 2).(*Space)
			if sb.Style.Template != "" {
				sp.Style.Template = sb.Style.Template + ".space"
			}
		}
		// text-field
		tf := sb.Parts.ChildByName("text-field", 0).(*TextField)
		tf.SetFlagState(sb.IsDisabled(), int(Disabled))
		// todo: see TreeView for extra steps needed to generally support styling of parts..
		// doing it manually for now..
		if sb.Style.Template != "" {
			tf.Style.Template = sb.Style.Template + ".text"
		}
		tf.Txt = sb.ValToString(sb.Value)
		if !sb.IsDisabled() {
			tf.TextFieldSig.ConnectOnly(sb.This(), func(recv, send ki.Ki, sig int64, data any) {
				if sig == int64(TextFieldDone) || sig == int64(TextFieldDeFocused) {
					sbb := recv.Embed(TypeSpinBox).(*SpinBox)
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

func (sb *SpinBox) ConfigPartsIfNeeded() {
	if !sb.Parts.HasChildren() {
		sb.ConfigParts()
	}
	tf := sb.Parts.ChildByName("text-field", 0).(*TextField)
	txt := sb.ValToString(sb.Value)
	if tf.Txt != txt {
		tf.SetText(txt)
	}
}

func (sb *SpinBox) MouseScrollEvent() {
	sb.ConnectEvent(oswin.MouseScrollEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		sbb := recv.Embed(TypeSpinBox).(*SpinBox)
		if sbb.IsDisabled() || !sbb.HasFocus2D() {
			return
		}
		me := d.(*mouse.ScrollEvent)
		me.SetProcessed()
		sbb.IncrValue(float32(me.NonZeroDelta(false)))
	})
}

func (sb *SpinBox) TextFieldEvent() {
	tf := sb.Parts.ChildByName("text-field", 0).(*TextField)
	tf.WidgetSig.ConnectOnly(sb.This(), func(recv, send ki.Ki, sig int64, data any) {
		sbb := recv.Embed(TypeSpinBox).(*SpinBox)
		if sig == int64(WidgetSelected) {
			sbb.SetSelectedState(!sbb.IsSelected())
		}
		sbb.WidgetSig.Emit(sbb.This(), sig, data) // passthrough
	})
}

func (sb *SpinBox) KeyChordEvent() {
	sb.ConnectEvent(oswin.KeyChordEvent, HiPri, func(recv, send ki.Ki, sig int64, d any) {
		sbb := recv.(*SpinBox)
		if sbb.IsDisabled() {
			return
		}
		kt := d.(*key.ChordEvent)
		if KeyEventTrace {
			fmt.Printf("SpinBox KeyChordEvent: %v\n", sbb.Path())
		}
		kf := KeyFun(kt.Chord())
		switch {
		case kf == KeyFunMoveUp:
			kt.SetProcessed()
			sb.IncrValue(1)
		case kf == KeyFunMoveDown:
			kt.SetProcessed()
			sb.IncrValue(-1)
		case kf == KeyFunPageUp:
			kt.SetProcessed()
			sb.PageIncrValue(1)
		case kf == KeyFunPageDown:
			kt.SetProcessed()
			sb.PageIncrValue(-1)
		}
	})
}

func (sb *SpinBox) SpinBoxEvents() {
	sb.HoverTooltipEvent()
	sb.MouseScrollEvent()
	sb.TextFieldEvent()
	sb.KeyChordEvent()
}

func (sb *SpinBox) Init2D() {
	sb.Init2DWidget()
	sb.ConfigParts()
}

// StyleFromProps styles SpinBox-specific fields from ki.Prop properties
// doesn't support inherit or default
func (sb *SpinBox) StyleFromProps(props ki.Props, vp *Viewport2D) {
	for key, val := range props {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		switch key {
		case "value":
			if iv, ok := kit.ToFloat32(val); ok {
				sb.Value = iv
			}
		case "min":
			if iv, ok := kit.ToFloat32(val); ok {
				sb.Min = iv
			}
		case "max":
			if iv, ok := kit.ToFloat32(val); ok {
				sb.Max = iv
			}
		case "step":
			if iv, ok := kit.ToFloat32(val); ok {
				sb.Step = iv
			}
		case "pagestep":
			if iv, ok := kit.ToFloat32(val); ok {
				sb.PageStep = iv
			}
		case "prec":
			if iv, ok := kit.ToInt(val); ok {
				sb.Prec = int(iv)
			}
		case "has-min":
			if bv, ok := kit.ToBool(val); ok {
				sb.HasMin = bv
			}
		case "has-max":
			if bv, ok := kit.ToBool(val); ok {
				sb.HasMax = bv
			}
		case "format":
			sb.Format = kit.ToString(val)
		}
	}
	if sb.PageStep < sb.Step { // often forget to set this..
		sb.PageStep = 10 * sb.Step
	}
}

// StyleSpinBox does spinbox styling -- sets StyMu Lock
func (sb *SpinBox) StyleSpinBox() {
	sb.StyMu.Lock()
	defer sb.StyMu.Unlock()

	sb.Style2DWidget()
}

func (sb *SpinBox) Style2D() {
	sb.StyleSpinBox()
	sb.StyMu.Lock()
	sb.LayState.SetFromStyle(&sb.Style) // also does reset
	sb.StyMu.Unlock()
	sb.ConfigParts()
}

func (sb *SpinBox) Size2D(iter int) {
	sb.Size2DParts(iter)
}

func (sb *SpinBox) Layout2D(parBBox image.Rectangle, iter int) bool {
	sb.ConfigPartsIfNeeded()
	sb.Layout2DBase(parBBox, true, iter) // init style
	sb.Layout2DParts(parBBox, iter)
	return sb.Layout2DChildren(iter)
}

func (sb *SpinBox) Render2D() {
	if sb.FullReRenderIfNeeded() {
		return
	}
	if sb.PushBounds() {
		sb.This().(Node2D).ConnectEvents2D()
		tf := sb.Parts.ChildByName("text-field", 2).(*TextField)
		tf.SetSelectedState(sb.IsSelected())
		sb.ConfigPartsIfNeeded()
		sb.Render2DChildren()
		sb.Render2DParts()
		sb.PopBounds()
	} else {
		sb.DisconnectAllEvents(RegPri)
	}
}

func (sb *SpinBox) ConnectEvents2D() {
	sb.SpinBoxEvents()
}

func (sb *SpinBox) HasFocus2D() bool {
	if sb.IsDisabled() {
		return false
	}
	return sb.ContainsFocus() // needed for getting key events
}
