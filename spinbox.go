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
	"github.com/goki/ki/bitflag"
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
	SpinBoxSig ki.Signal `json:"-" xml:"-" desc:"signal for spin box -- has no signal types, just emitted when the value changes"`
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
		"fill":       &Prefs.IconColor,
		"stroke":     &Prefs.FontColor,
	},
	"#down": ki.Props{
		"max-width":  units.NewValue(1.5, units.Ex),
		"max-height": units.NewValue(1.5, units.Ex),
		"margin":     units.NewValue(1, units.Px),
		"padding":    units.NewValue(0, units.Px),
		"fill":       &Prefs.IconColor,
		"stroke":     &Prefs.FontColor,
	},
	"#space": ki.Props{
		"width": units.NewValue(.1, units.Ex),
	},
	"#text-field": ki.Props{
		"min-width": units.NewValue(4, units.Ex),
		"width":     units.NewValue(8, units.Ex),
		"margin":    units.NewValue(2, units.Px),
		"padding":   units.NewValue(2, units.Px),
	},
}

func (g *SpinBox) Defaults() { // todo: should just get these from props
	g.Step = 0.1
	g.PageStep = 0.2
	g.Max = 1.0
	g.Prec = 6
}

// SetMin sets the min limits on the value
func (g *SpinBox) SetMin(min float32) {
	g.HasMin = true
	g.Min = min
}

// SetMax sets the max limits on the value
func (g *SpinBox) SetMax(max float32) {
	g.HasMax = true
	g.Max = max
}

// SetMinMax sets the min and max limits on the value
func (g *SpinBox) SetMinMax(hasMin bool, min float32, hasMax bool, max float32) {
	g.HasMin = hasMin
	g.Min = min
	g.HasMax = hasMax
	g.Max = max
	if g.Max < g.Min {
		log.Printf("gi.SpinBox SetMinMax: max was less than min -- disabling limits\n")
		g.HasMax = false
		g.HasMin = false
	}
}

// SetValue sets the value, enforcing any limits, and updates the display
func (g *SpinBox) SetValue(val float32) {
	updt := g.UpdateStart()
	defer g.UpdateEnd(updt)
	if g.Prec == 0 {
		g.Defaults()
	}
	g.Value = val
	if g.HasMax {
		g.Value = Min32(g.Value, g.Max)
	}
	if g.HasMin {
		g.Value = Max32(g.Value, g.Min)
	}
	g.Value = Truncate32(g.Value, g.Prec)
}

// SetValueAction calls SetValue and also emits the signal
func (g *SpinBox) SetValueAction(val float32) {
	g.SetValue(val)
	g.SpinBoxSig.Emit(g.This, 0, g.Value)
}

// IncrValue increments the value by given number of steps (+ or -), and enforces it to be an even multiple of the step size (snap-to-value), and emits the signal
func (g *SpinBox) IncrValue(steps float32) {
	val := g.Value + steps*g.Step
	val = FloatMod32(val, g.Step)
	g.SetValueAction(val)
}

// internal indexes for accessing elements of the widget
const (
	sbTextFieldIdx = iota
	sbSpaceIdx
	sbButtonsIdx
)

func (g *SpinBox) ConfigParts() {
	if g.UpIcon.IsNil() {
		g.UpIcon = IconName("widget-wedge-up")
	}
	if g.DownIcon.IsNil() {
		g.DownIcon = IconName("widget-wedge-down")
	}
	g.Parts.Lay = LayoutRow
	g.Parts.SetProp("vertical-align", AlignMiddle)
	config := kit.TypeAndNameList{}
	config.Add(KiT_TextField, "text-field")
	config.Add(KiT_Space, "space")
	config.Add(KiT_Layout, "buttons")
	mods, updt := g.Parts.ConfigChildren(config, false) // not unique names
	if mods {
		buts := g.Parts.KnownChild(sbButtonsIdx).(*Layout)
		buts.Lay = LayoutCol
		g.StylePart(Node2D(buts))
		buts.SetNChildren(2, KiT_Action, "but")
		// up
		up := buts.KnownChild(0).(*Action)
		up.SetName("up")
		bitflag.SetState(up.Flags(), g.IsInactive(), int(Inactive))
		up.Icon = g.UpIcon
		g.StylePart(Node2D(up))
		if !g.IsInactive() {
			up.ActionSig.ConnectOnly(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				sb := recv.EmbeddedStruct(KiT_SpinBox).(*SpinBox)
				sb.IncrValue(1.0)
			})
		}
		// dn
		dn := buts.KnownChild(1).(*Action)
		bitflag.SetState(dn.Flags(), g.IsInactive(), int(Inactive))
		dn.SetName("down")
		dn.Icon = g.DownIcon
		g.StylePart(Node2D(dn))
		if !g.IsInactive() {
			dn.ActionSig.ConnectOnly(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				sb := recv.EmbeddedStruct(KiT_SpinBox).(*SpinBox)
				sb.IncrValue(-1.0)
			})
		}
		// space
		g.StylePart(g.Parts.KnownChild(sbSpaceIdx).(Node2D)) // also get the space
		// text-field
		tf := g.Parts.KnownChild(sbTextFieldIdx).(*TextField)
		bitflag.SetState(tf.Flags(), g.IsInactive(), int(Inactive))
		g.StylePart(Node2D(tf))
		tf.Txt = fmt.Sprintf("%g", g.Value)
		if !g.IsInactive() {
			tf.TextFieldSig.ConnectOnly(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(TextFieldDone) {
					sb := recv.EmbeddedStruct(KiT_SpinBox).(*SpinBox)
					tf := send.(*TextField)
					vl, err := strconv.ParseFloat(tf.Text(), 32)
					if err == nil {
						sb.SetValueAction(float32(vl))
					}
				}
			})
		}
		g.UpdateEnd(updt)
	}
}

func (g *SpinBox) ConfigPartsIfNeeded() {
	if !g.Parts.HasChildren() {
		g.ConfigParts()
	}
	tf := g.Parts.KnownChild(sbTextFieldIdx).(*TextField)
	txt := fmt.Sprintf("%g", g.Value)
	if tf.Txt != txt {
		tf.SetText(txt)
	}
}

func (g *SpinBox) SpinBoxEvents() {
	g.ConnectEventType(oswin.MouseScrollEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		sb := recv.EmbeddedStruct(KiT_SpinBox).(*SpinBox)
		if sb.IsInactive() {
			return
		}
		me := d.(*mouse.ScrollEvent)
		me.SetProcessed()
		sb.IncrValue(float32(me.NonZeroDelta(false)))
	})
	tf := g.Parts.KnownChild(sbTextFieldIdx).(*TextField)
	tf.WidgetSig.ConnectOnly(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		sb := recv.EmbeddedStruct(KiT_SpinBox).(*SpinBox)
		if sig == int64(WidgetSelected) {
			sb.SetSelectedState(!sb.IsSelected())
		}
		sb.WidgetSig.Emit(sb.This, sig, data) // passthrough
	})
}

func (g *SpinBox) Init2D() {
	g.Init2DWidget()
	g.ConfigParts()
}

func (g *SpinBox) Style2D() {
	if g.Step == 0 {
		g.Defaults()
	}
	g.Style2DWidget()
	g.ConfigParts()
}

func (g *SpinBox) Size2D() {
	g.Size2DParts()
	g.ConfigParts()
}

func (g *SpinBox) Layout2D(parBBox image.Rectangle) {
	g.ConfigPartsIfNeeded()
	g.Layout2DBase(parBBox, true) // init style
	g.Layout2DParts(parBBox)
	g.Layout2DChildren()
}

func (g *SpinBox) Render2D() {
	if g.FullReRenderIfNeeded() {
		return
	}
	if g.PushBounds() {
		g.SpinBoxEvents()
		// g.Sty = g.StateStyles[g.State] // get current styles
		tf := g.Parts.KnownChild(sbTextFieldIdx).(*TextField)
		tf.SetSelectedState(g.IsSelected())
		g.ConfigPartsIfNeeded()
		g.Render2DChildren()
		g.Render2DParts()
		g.PopBounds()
	} else {
		g.DisconnectAllEvents(RegPri)
	}
}
