// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"reflect"

	"github.com/goki/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

/////////////////////////////////////////////////////////////////////////////
//  ColorView

// ColorView represents a color, using sliders to set values
type ColorView struct {
	gi.Frame
	Color   *gi.Color `desc:"the color that we view"`
	Title   string    `desc:"title / prompt to show above the editor fields"`
	TmpSave ValueView `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	ViewSig ki.Signal `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`
}

var KiT_ColorView = kit.Types.AddType(&ColorView{}, ColorViewProps)

var ColorViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"#title": ki.Props{
		"max-width":      units.NewValue(-1, units.Px),
		"text-align":     gi.AlignCenter,
		"vertical-align": gi.AlignTop,
	},
}

// SetColor sets the source color
func (sv *ColorView) SetColor(color *gi.Color, tmpSave ValueView) {
	if sv.Color != color {
		sv.Color = color
		sv.Config()
	}
	sv.TmpSave = tmpSave
	sv.Update()
}

// Config configures a standard setup of entire view
func (sv *ColorView) Config() {
	sv.Lay = gi.LayoutVert
	config := sv.StdFrameConfig()
	mods, updt := sv.ConfigChildren(config, false)
	if mods {
		sv.ValueLayConfig()
	} else {
		updt = sv.UpdateStart()
	}
	sv.UpdateEnd(updt)
}

func (sv *ColorView) ValueLayConfig() {
	vl, _ := sv.ValueLay()
	if vl == nil {
		return
	}
	config := sv.StdValueLayConfig()
	mods, updt := vl.ConfigChildren(config, false)
	v, _ := sv.Value()
	if mods {
		sv.ConfigSliderGrid()
		v.SetProp("min-width", units.NewValue(6, units.Em))
		v.SetProp("min-height", units.NewValue(6, units.Em))
	} else {
		updt = vl.UpdateStart()
	}
	vl.UpdateEnd(updt)
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (sv *ColorView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Label, "title")
	config.Add(gi.KiT_Space, "title-space")
	config.Add(gi.KiT_Layout, "value-lay")
	config.Add(gi.KiT_Space, "slider-space")
	// config.Add(gi.KiT_Layout, "buttons")
	return config
}

func (sv *ColorView) StdValueLayConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Frame, "value")
	config.Add(gi.KiT_Layout, "slider-grid")
	return config
}

// SetTitle sets the title and updates the Title label
func (sv *ColorView) SetTitle(title string) {
	sv.Title = title
	lab, _ := sv.TitleWidget()
	if lab != nil {
		lab.Text = title
	}
}

// Title returns the title label widget, and its index, within frame -- nil, -1 if not found
func (sv *ColorView) TitleWidget() (*gi.Label, int) {
	idx, ok := sv.Children().IndexByName("title", 0)
	if !ok {
		return nil, -1
	}
	return sv.KnownChild(idx).(*gi.Label), idx
}

func (sv *ColorView) ValueLay() (*gi.Layout, int) {
	idx, ok := sv.Children().IndexByName("value-lay", 3)
	if !ok {
		return nil, -1
	}
	return sv.KnownChild(idx).(*gi.Layout), idx
}

func (sv *ColorView) Value() (*gi.Frame, int) {
	vl, _ := sv.ValueLay()
	if vl == nil {
		return nil, -1
	}
	idx, ok := vl.Children().IndexByName("value", 3)
	if !ok {
		return nil, -1
	}
	return vl.KnownChild(idx).(*gi.Frame), idx
}

func (sv *ColorView) SliderGrid() (*gi.Layout, int) {
	vl, _ := sv.ValueLay()
	if vl == nil {
		return nil, -1
	}
	idx, ok := vl.Children().IndexByName("slider-grid", 0)
	if !ok {
		return nil, -1
	}
	return vl.KnownChild(idx).(*gi.Layout), idx
}

// ButtonBox returns the ButtonBox layout widget, and its index, within frame -- nil, -1 if not found
func (sv *ColorView) ButtonBox() (*gi.Layout, int) {
	idx, ok := sv.Children().IndexByName("buttons", 0)
	if !ok {
		return nil, -1
	}
	return sv.KnownChild(idx).(*gi.Layout), idx
}

// StdSliderConfig returns a TypeAndNameList for configuring a standard sliders
func (sv *ColorView) StdSliderConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Label, "rlab")
	config.Add(gi.KiT_Slider, "red")
	config.Add(gi.KiT_Label, "hlab")
	config.Add(gi.KiT_Slider, "hue")
	config.Add(gi.KiT_Label, "glab")
	config.Add(gi.KiT_Slider, "green")
	config.Add(gi.KiT_Label, "slab")
	config.Add(gi.KiT_Slider, "sat")
	config.Add(gi.KiT_Label, "blab")
	config.Add(gi.KiT_Slider, "blue")
	config.Add(gi.KiT_Label, "llab")
	config.Add(gi.KiT_Slider, "light")
	return config
}

func (sv *ColorView) SetRGBValue(val float32, rgb int) {
	if sv.Color == nil {
		return
	}
	switch rgb {
	case 0:
		sv.Color.R = uint8(val)
	case 1:
		sv.Color.G = uint8(val)
	case 2:
		sv.Color.B = uint8(val)
	}
	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}
}

func (sv *ColorView) ConfigRGBSlider(sl *gi.Slider, rgb int) {
	sl.Defaults()
	sl.Max = 255
	sl.Step = 1
	sl.PageStep = 16
	sl.Prec = 3
	sl.Dim = gi.X
	sl.Tracking = true
	sl.TrackThr = 1
	sl.SetMinPrefWidth(units.NewValue(20, units.Ch))
	sl.SetMinPrefHeight(units.NewValue(2, units.Em))
	sl.SliderSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.SliderValueChanged) {
			svv, _ := recv.Embed(KiT_ColorView).(*ColorView)
			slv := send.Embed(gi.KiT_Slider).(*gi.Slider)
			updt := svv.UpdateStart()
			svv.SetRGBValue(slv.Value, rgb)
			svv.ViewSig.Emit(svv.This, 0, nil)
			svv.UpdateEnd(updt)
		}
	})
}

func (sv *ColorView) UpdateRGBSlider(sl *gi.Slider, rgb int) {
	if sv.Color == nil {
		return
	}
	switch rgb {
	case 0:
		sl.SetValue(float32(sv.Color.R))
	case 1:
		sl.SetValue(float32(sv.Color.G))
	case 2:
		sl.SetValue(float32(sv.Color.B))
	}
}

func (sv *ColorView) SetHSLValue(val float32, hsl int) {
	if sv.Color == nil {
		return
	}
	h, s, l, _ := sv.Color.ToHSLA()
	switch hsl {
	case 0:
		h = val
	case 1:
		s = val / 360.0
	case 2:
		l = val / 360.0
	}
	sv.Color.SetHSL(h, s, l)
	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}
}

func (sv *ColorView) ConfigHSLSlider(sl *gi.Slider, hsl int) {
	sl.Defaults()
	sl.Max = 360
	sl.Step = 1
	sl.PageStep = 15
	sl.Prec = 3
	sl.Dim = gi.X
	sl.Tracking = true
	sl.TrackThr = 1
	sl.SetMinPrefWidth(units.NewValue(20, units.Ch))
	sl.SetMinPrefHeight(units.NewValue(2, units.Em))
	sl.SliderSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.SliderValueChanged) {
			svv, _ := recv.Embed(KiT_ColorView).(*ColorView)
			slv := send.Embed(gi.KiT_Slider).(*gi.Slider)
			updt := svv.UpdateStart()
			svv.SetHSLValue(slv.Value, hsl)
			svv.ViewSig.Emit(svv.This, 0, nil)
			svv.UpdateEnd(updt)
		}
	})
}

func (sv *ColorView) UpdateHSLSlider(sl *gi.Slider, hsl int) {
	if sv.Color == nil {
		return
	}
	h, s, l, _ := sv.Color.ToHSLA()
	switch hsl {
	case 0:
		sl.SetValue(h)
	case 1:
		sl.SetValue(s * 360.0)
	case 2:
		sl.SetValue(l * 360.0)
	}
}

func (sv *ColorView) ConfigLabel(lab *gi.Label, txt string) {
	lab.Text = txt

}

// ConfigSliderGrid configures the SliderGrid
func (sv *ColorView) ConfigSliderGrid() {
	sg, _ := sv.SliderGrid()
	if sg == nil {
		return
	}
	sg.Lay = gi.LayoutGrid
	sg.SetProp("columns", 4)
	config := sv.StdSliderConfig()
	mods, updt := sg.ConfigChildren(config, false)
	if mods {
		sv.ConfigLabel(sg.KnownChildByName("rlab", 0).Embed(gi.KiT_Label).(*gi.Label), "Red:")
		sv.ConfigLabel(sg.KnownChildByName("blab", 0).Embed(gi.KiT_Label).(*gi.Label), "Blue")
		sv.ConfigLabel(sg.KnownChildByName("glab", 0).Embed(gi.KiT_Label).(*gi.Label), "Green:")
		sv.ConfigLabel(sg.KnownChildByName("hlab", 0).Embed(gi.KiT_Label).(*gi.Label), "Hue:")
		sv.ConfigLabel(sg.KnownChildByName("slab", 0).Embed(gi.KiT_Label).(*gi.Label), "Sat:")
		sv.ConfigLabel(sg.KnownChildByName("llab", 0).Embed(gi.KiT_Label).(*gi.Label), "Light:")

		sv.ConfigRGBSlider(sg.KnownChildByName("red", 0).Embed(gi.KiT_Slider).(*gi.Slider), 0)
		sv.ConfigRGBSlider(sg.KnownChildByName("green", 0).Embed(gi.KiT_Slider).(*gi.Slider), 1)
		sv.ConfigRGBSlider(sg.KnownChildByName("blue", 0).Embed(gi.KiT_Slider).(*gi.Slider), 2)
		sv.ConfigHSLSlider(sg.KnownChildByName("hue", 0).Embed(gi.KiT_Slider).(*gi.Slider), 0)
		sv.ConfigHSLSlider(sg.KnownChildByName("sat", 0).Embed(gi.KiT_Slider).(*gi.Slider), 1)
		sv.ConfigHSLSlider(sg.KnownChildByName("light", 0).Embed(gi.KiT_Slider).(*gi.Slider), 2)
	} else {
		updt = sg.UpdateStart()
	}
	sg.UpdateEnd(updt)
}

func (sv *ColorView) UpdateSliderGrid() {
	sg, _ := sv.SliderGrid()
	if sg == nil {
		return
	}
	updt := sg.UpdateStart()
	sv.UpdateRGBSlider(sg.KnownChildByName("red", 0).Embed(gi.KiT_Slider).(*gi.Slider), 0)
	sv.UpdateRGBSlider(sg.KnownChildByName("green", 0).Embed(gi.KiT_Slider).(*gi.Slider), 1)
	sv.UpdateRGBSlider(sg.KnownChildByName("blue", 0).Embed(gi.KiT_Slider).(*gi.Slider), 2)
	sv.UpdateHSLSlider(sg.KnownChildByName("hue", 0).Embed(gi.KiT_Slider).(*gi.Slider), 0)
	sv.UpdateHSLSlider(sg.KnownChildByName("sat", 0).Embed(gi.KiT_Slider).(*gi.Slider), 1)
	sv.UpdateHSLSlider(sg.KnownChildByName("light", 0).Embed(gi.KiT_Slider).(*gi.Slider), 2)
	sg.UpdateEnd(updt)
}

func (sv *ColorView) Update() {
	updt := sv.UpdateStart()
	sv.UpdateSliderGrid()
	if sv.Color != nil {
		v, _ := sv.Value()
		v.Sty.Font.BgColor.Color = *sv.Color // direct copy
	}
	sv.UpdateEnd(updt)
}

func (sv *ColorView) Render2D() {
	if sv.FullReRenderIfNeeded() {
		return
	}
	if sv.PushBounds() {
		updt := sv.UpdateStart()
		sv.Update()
		sv.UpdateEndNoSig(updt)
		sv.PopBounds()
	}
	sv.Frame.Render2D()
}

////////////////////////////////////////////////////////////////////////////////////////
//  ColorValueView

// ColorValueView presents a StructViewInline for a struct plus a ColorView button..
type ColorValueView struct {
	ValueViewBase
}

var KiT_ColorValueView = kit.Types.AddType(&ColorValueView{}, nil)

func (vv *ColorValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = KiT_StructViewInline
	return vv.WidgetTyp
}

func (vv *ColorValueView) UpdateWidget() {
	sv := vv.Widget.(*StructViewInline)
	clr, ok := vv.Value.Interface().(*gi.Color)
	if !ok {
		clr, ok = vv.Value.Elem().Interface().(*gi.Color)
		// fmt.Printf("clr is ** in vv: %v\n", vv.PathUnique())
	}
	if clr != nil {
		edack, ok := sv.Parts.Children().ElemFromEnd(0) // action at end, from AddAction above
		if ok {
			edac := edack.(*gi.Action)
			edac.SetProp("background-color", *clr)
			edac.SetFullReRender()
		}
	}
	sv.UpdateFields()
}

func (vv *ColorValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg

	sv := vv.Widget.(*StructViewInline)
	sv.AddAction = true
	vv.CreateTempIfNotPtr() // we need our value to be a ptr to a struct -- if not make a tmp
	sv.SetStruct(vv.Value.Interface(), vv.TmpSave)

	edack, ok := sv.Parts.Children().ElemFromEnd(0) // action at end, from AddAction above
	if ok {
		edac := edack.(*gi.Action)
		edac.SetIcon("color")
		edac.Tooltip = "color selection dialog"
		edac.ActionSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			svv, _ := recv.Embed(KiT_StructViewInline).(*StructViewInline)
			clr, ok := svv.Struct.(*gi.Color)
			if !ok {
				clrp, ok := svv.Struct.(**gi.Color)
				if !ok {
					return
				}
				clr = *clrp
			}
			dlg := ColorViewDialog(svv.Viewport, clr, svv.TmpSave, "Color Value View", "", nil, nil, nil)
			cvvvk, ok := dlg.Frame().Children().ElemByType(KiT_ColorView, true, 2)
			if ok {
				cvvv := cvvvk.(*ColorView)
				cvvv.ViewSig.ConnectOnly(svv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					cvvvv, _ := recv.Embed(KiT_StructViewInline).(*StructViewInline)
					cvvvv.ViewSig.Emit(cvvvv.This, 0, nil)
				})
			}
		})
	}

	vv.UpdateWidget()

	sv.ViewSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_ColorValueView).(*ColorValueView)
		vvv.UpdateWidget() // necessary in this case!
		vvv.ViewSig.Emit(vvv.This, 0, nil)
	})
}
