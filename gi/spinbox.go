// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"strconv"

	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
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
	Format     string    `xml:"format" desc:"prop = format -- format string for printing the value -- blank defaults to %g.  If decimal based (ends in d, b, c, o, O, q, x, X, or U) then value is converted to decimal prior to printing"`
	UpIcon     IconName  `view:"show-name" desc:"icon to use for up button -- defaults to wedge-up"`
	DownIcon   IconName  `view:"show-name" desc:"icon to use for down button -- defaults to wedge-down"`
	SpinBoxSig ki.Signal `copy:"-" json:"-" xml:"-" view:"-" desc:"signal for spin box -- has no signal types, just emitted when the value changes"`
}

var KiT_SpinBox = kit.Types.AddType(&SpinBox{}, SpinBoxProps)

// AddNewSpinBox adds a new spinbox to given parent node, with given name.
func AddNewSpinBox(parent ki.Ki, name string) *SpinBox {
	return parent.AddNewChild(KiT_SpinBox, name).(*SpinBox)
}

func (sb *SpinBox) CopyFieldsFrom(frm interface{}) {
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
	"EnumType:Flag": KiT_NodeFlags,
	"#buttons": ki.Props{
		"vertical-align": gist.AlignMiddle,
	},
	"#up": ki.Props{
		"max-width":  units.NewEx(1.5),
		"max-height": units.NewEx(1.5),
		"margin":     units.NewPx(1),
		"padding":    units.NewPx(0),
		"fill":       &Prefs.Colors.Icon,
		"stroke":     &Prefs.Colors.Font,
	},
	"#down": ki.Props{
		"max-width":  units.NewEx(1.5),
		"max-height": units.NewEx(1.5),
		"margin":     units.NewPx(1),
		"padding":    units.NewPx(0),
		"fill":       &Prefs.Colors.Icon,
		"stroke":     &Prefs.Colors.Font,
	},
	"#space": ki.Props{
		"width": units.NewCh(.1),
	},
	"#text-field": ki.Props{
		"min-width": units.NewCh(4),
		"width":     units.NewCh(8),
		"margin":    units.NewPx(2),
		"padding":   units.NewPx(2),
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

// IncrValue increments the value by given number of steps (+ or -), and enforces it to be an even multiple of the step size (snap-to-value), and emits the signal
func (sb *SpinBox) IncrValue(steps float32) {
	val := sb.Value + steps*sb.Step
	val = mat32.IntMultiple(val, sb.Step)
	sb.SetValueAction(val)
}

func (sb *SpinBox) ConfigParts() {
	if sb.UpIcon.IsNil() {
		sb.UpIcon = IconName("wedge-up")
	}
	if sb.DownIcon.IsNil() {
		sb.DownIcon = IconName("wedge-down")
	}
	sb.Parts.Lay = LayoutHoriz
	sb.Parts.SetProp("vertical-align", gist.AlignMiddle)
	if sb.Sty.Template != "" {
		sb.Parts.Sty.Template = sb.Sty.Template + ".Parts"
	}
	config := kit.TypeAndNameList{}
	config.Add(KiT_TextField, "text-field")
	if !sb.IsInactive() {
		config.Add(KiT_Space, "space")
		config.Add(KiT_Layout, "buttons")
	}
	mods, updt := sb.Parts.ConfigChildren(config)
	if mods || gist.RebuildDefaultStyles {
		if !sb.IsInactive() {
			buts := sb.Parts.ChildByName("buttons", 1).(*Layout)
			buts.Lay = LayoutVert
			sb.StylePart(Node2D(buts))
			buts.SetNChildren(2, KiT_Action, "but")
			// up
			up := buts.Child(0).(*Action)
			up.SetName("up")
			up.SetProp("no-focus", true) // note: cannot be in compiled props b/c
			// not compiled into style prop
			// up.SetFlagState(sb.IsInactive(), int(Inactive))
			up.Icon = sb.UpIcon
			if sb.Sty.Template != "" {
				up.Sty.Template = sb.Sty.Template + ".up"
			}
			sb.StylePart(Node2D(up))
			up.ActionSig.ConnectOnly(sb.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				sbb := recv.Embed(KiT_SpinBox).(*SpinBox)
				sbb.IncrValue(1.0)
			})
			// dn
			dn := buts.Child(1).(*Action)
			// dn.SetFlagState(sb.IsInactive(), int(Inactive))
			dn.SetName("down")
			dn.SetProp("no-focus", true)
			dn.Icon = sb.DownIcon
			sb.StylePart(Node2D(dn))
			if sb.Sty.Template != "" {
				dn.Sty.Template = sb.Sty.Template + ".dn"
			}
			dn.ActionSig.ConnectOnly(sb.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				sbb := recv.Embed(KiT_SpinBox).(*SpinBox)
				sbb.IncrValue(-1.0)
			})
			// space
			sp := sb.Parts.ChildByName("space", 2).(*Space)
			if sb.Sty.Template != "" {
				sp.Sty.Template = sb.Sty.Template + ".space"
			}
			sb.StylePart(sp) // also get the space
		}
		// text-field
		tf := sb.Parts.ChildByName("text-field", 0).(*TextField)
		tf.SetFlagState(sb.IsInactive(), int(Inactive))
		// todo: see TreeView for extra steps needed to generally support styling of parts..
		// doing it manually for now..
		tf.SetProp("clear-act", false)
		if sb.Sty.Template != "" {
			tf.Sty.Template = sb.Sty.Template + ".text"
		}
		sb.StylePart(Node2D(tf))
		tf.Txt = sb.ValToString(sb.Value)
		if !sb.IsInactive() {
			tf.TextFieldSig.ConnectOnly(sb.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(TextFieldDone) || sig == int64(TextFieldDeFocused) {
					sbb := recv.Embed(KiT_SpinBox).(*SpinBox)
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
	tf := sb.Parts.ChildByName("text-field", 0).(*TextField)
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
}

// StyleSpinBox does spinbox styling -- sets StyMu Lock
func (sb *SpinBox) StyleSpinBox() {
	sb.StyMu.Lock()
	defer sb.StyMu.Unlock()

	if sb.Step == 0 {
		sb.Defaults()
	}
	hasTempl, saveTempl := sb.Sty.FromTemplate()
	if !hasTempl || saveTempl {
		sb.Style2DWidget()
	} else {
		SetUnitContext(&sb.Sty, sb.Viewport, mat32.Vec2Zero)
	}
	if hasTempl && saveTempl {
		sb.Sty.SaveTemplate()
	}
	sb.StyleFromProps(sb.Props, sb.Viewport)
}

func (sb *SpinBox) Style2D() {
	sb.StyleSpinBox()
	sb.StyMu.Lock()
	sb.LayState.SetFromStyle(&sb.Sty.Layout) // also does reset
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
	if sb.IsInactive() {
		return false
	}
	return sb.ContainsFocus() // needed for getting key events
}
