// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"strconv"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
// SpinBox

// SpinBox combines a TextField with up / down buttons for incrementing /
// decrementing values -- all configured within the Parts of the widget
type SpinBox struct {
	PartsWidgetBase
	Value      float32   `xml:"value" desc:"current value"`
	HasMin     bool      `xml:"has-min" desc:"is there a minimum value to enforce"`
	Min        float32   `xml:"min" desc:"minimum value in range"`
	HasMax     bool      `xml:"has-max" desc:"is there a maximumvalue to enforce"`
	Max        float32   `xml:"max" desc:"maximum value in range"`
	Step       float32   `xml:"step" desc:"smallest step size to increment"`
	PageStep   float32   `xml:"pagestep" desc:"larger PageUp / Dn step size"`
	Prec       int       `desc:"specifies the precision of decimal places (total, not after the decimal point) to use in representing the number -- this helps to truncate small weird floating point values in the nether regions"`
	UpIcon     IconName  `view:"show-name" desc:"icon to use for up button -- defaults to widget-wedge-up"`
	DownIcon   IconName  `view:"show-name" desc:"icon to use for down button -- defaults to widget-wedge-down"`
	SpinBoxSig ki.Signal `json:"-" xml:"-" view:"-" desc:"signal for spin box -- has no signal types, just emitted when the value changes"`
}

var KiT_SpinBox = kit.Types.AddType(&SpinBox{}, SpinBoxProps)

var SpinBoxProps = ki.Props{
	"#buttons": ki.Props{
		"vertical-align": AlignMiddle,
	},
	"#up": ki.Props{
		"max-width":  units.NewValue(1.5, units.Ex),
		"max-height": units.NewValue(1.5, units.Ex),
		"margin":     units.NewValue(1, units.Px),
		"padding":    units.NewValue(0, units.Px),
		"fill":       &Prefs.Colors.Icon,
		"stroke":     &Prefs.Colors.Font,
	},
	"#down": ki.Props{
		"max-width":  units.NewValue(1.5, units.Ex),
		"max-height": units.NewValue(1.5, units.Ex),
		"margin":     units.NewValue(1, units.Px),
		"padding":    units.NewValue(0, units.Px),
		"fill":       &Prefs.Colors.Icon,
		"stroke":     &Prefs.Colors.Font,
	},
	"#space": ki.Props{
		"width": units.NewValue(.1, units.Ch),
	},
	"#text-field": ki.Props{
		"min-width": units.NewValue(4, units.Ch),
		"width":     units.NewValue(8, units.Ch),
		"margin":    units.NewValue(2, units.Px),
		"padding":   units.NewValue(2, units.Px),
		"clear-act": false,
	},
}

func (sb *SpinBox) Defaults() { // todo: should just get these from props
	sb.Step = 0.1
	sb.PageStep = 0.2
	sb.Max = 1.0
	sb.Prec = 6
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
	if sb.Prec == 0 {
		sb.Defaults()
	}
	sb.Value = val
	if sb.HasMax {
		sb.Value = Min32(sb.Value, sb.Max)
	}
	if sb.HasMin {
		sb.Value = Max32(sb.Value, sb.Min)
	}
	sb.Value = Truncate32(sb.Value, sb.Prec)
}

// SetValueAction calls SetValue and also emits the signal
func (sb *SpinBox) SetValueAction(val float32) {
	sb.SetValue(val)
	sb.SpinBoxSig.Emit(sb.This(), 0, sb.Value)
}

// IncrValue increments the value by given number of steps (+ or -), and enforces it to be an even multiple of the step size (snap-to-value), and emits the signal
func (sb *SpinBox) IncrValue(steps float32) {
	val := sb.Value + steps*sb.Step
	val = FloatMod32(val, sb.Step)
	sb.SetValueAction(val)
}

// internal indexes for accessing elements of the widget
const (
	sbTextFieldIdx = iota
	sbSpaceIdx
	sbButtonsIdx
)

func (sb *SpinBox) ConfigParts() {
	if sb.UpIcon.IsNil() {
		sb.UpIcon = IconName("widget-wedge-up")
	}
	if sb.DownIcon.IsNil() {
		sb.DownIcon = IconName("widget-wedge-down")
	}
	sb.Parts.Lay = LayoutHoriz
	sb.Parts.SetProp("vertical-align", AlignMiddle)
	config := kit.TypeAndNameList{}
	config.Add(KiT_TextField, "text-field")
	config.Add(KiT_Space, "space")
	config.Add(KiT_Layout, "buttons")
	mods, updt := sb.Parts.ConfigChildren(config, false) // not unique names
	if mods || RebuildDefaultStyles {
		buts := sb.Parts.Child(sbButtonsIdx).(*Layout)
		buts.Lay = LayoutVert
		sb.StylePart(Node2D(buts))
		buts.SetNChildren(2, KiT_Action, "but")
		// up
		up := buts.Child(0).(*Action)
		up.SetName("up")
		up.SetProp("no-focus", true) // note: cannot be in compiled props b/c
		// not compiled into style prop
		up.SetFlagState(sb.IsInactive(), int(Inactive))
		up.Icon = sb.UpIcon
		sb.StylePart(Node2D(up))
		if !sb.IsInactive() {
			up.ActionSig.ConnectOnly(sb.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				sbb := recv.Embed(KiT_SpinBox).(*SpinBox)
				sbb.IncrValue(1.0)
			})
		}
		// dn
		dn := buts.Child(1).(*Action)
		dn.SetFlagState(sb.IsInactive(), int(Inactive))
		dn.SetName("down")
		dn.SetProp("no-focus", true)
		dn.Icon = sb.DownIcon
		sb.StylePart(Node2D(dn))
		if !sb.IsInactive() {
			dn.ActionSig.ConnectOnly(sb.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				sbb := recv.Embed(KiT_SpinBox).(*SpinBox)
				sbb.IncrValue(-1.0)
			})
		}
		// space
		sb.StylePart(sb.Parts.Child(sbSpaceIdx).(Node2D)) // also get the space
		// text-field
		tf := sb.Parts.Child(sbTextFieldIdx).(*TextField)
		sb.SetFlagState(sb.IsInactive(), int(Inactive))
		// todo: see TreeView for extra steps needed to generally support styling of parts..
		// doing it manually for now..
		tf.SetProp("clear-act", false)
		sb.StylePart(Node2D(tf))
		tf.Txt = fmt.Sprintf("%g", sb.Value)
		if !sb.IsInactive() {
			tf.TextFieldSig.ConnectOnly(sb.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(TextFieldDone) || sig == int64(TextFieldDeFocused) {
					sbb := recv.Embed(KiT_SpinBox).(*SpinBox)
					tf := send.(*TextField)
					vl, err := strconv.ParseFloat(tf.Text(), 32)
					if err == nil {
						sbb.SetValueAction(float32(vl))
					}
				}
			})
		}
		sb.UpdateEnd(updt)
	}
}

func (sb *SpinBox) ConfigPartsIfNeeded() {
	if !sb.Parts.HasChildren() {
		sb.ConfigParts()
	}
	tf := sb.Parts.Child(sbTextFieldIdx).(*TextField)
	txt := fmt.Sprintf("%g", sb.Value)
	if tf.Txt != txt {
		tf.SetText(txt)
	}
}

func (sb *SpinBox) MouseScrollEvent() {
	sb.ConnectEvent(oswin.MouseScrollEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		sbb := recv.Embed(KiT_SpinBox).(*SpinBox)
		if sbb.IsInactive() || !sbb.HasFocus2D() {
			return
		}
		me := d.(*mouse.ScrollEvent)
		me.SetProcessed()
		sbb.IncrValue(float32(me.NonZeroDelta(false)))
	})
}

func (sb *SpinBox) TextFieldEvent() {
	tf := sb.Parts.Child(sbTextFieldIdx).(*TextField)
	tf.WidgetSig.ConnectOnly(sb.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		sbb := recv.Embed(KiT_SpinBox).(*SpinBox)
		if sig == int64(WidgetSelected) {
			sbb.SetSelectedState(!sbb.IsSelected())
		}
		sbb.WidgetSig.Emit(sbb.This(), sig, data) // passthrough
	})
}

func (sb *SpinBox) SpinBoxEvents() {
	sb.HoverTooltipEvent()
	sb.MouseScrollEvent()
	sb.TextFieldEvent()
}

func (sb *SpinBox) Init2D() {
	sb.Init2DWidget()
	sb.ConfigParts()
}

func (sb *SpinBox) StyleSpinBox() {
	if sb.Step == 0 {
		sb.Defaults()
	}
	sb.Style2DWidget()
}

func (sb *SpinBox) Style2D() {
	sb.StyleSpinBox()
	sb.LayData.SetFromStyle(&sb.Sty.Layout) // also does reset
	sb.ConfigParts()
}

func (sb *SpinBox) Size2D(iter int) {
	sb.Size2DParts(iter)
	sb.ConfigParts()
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
		// sb.Sty = sb.StateStyles[sb.State] // get current styles
		tf := sb.Parts.Child(sbTextFieldIdx).(*TextField)
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
	if sb.IsInactive() {
		return false
	}
	return sb.ContainsFocus() // needed for getting key events
}
