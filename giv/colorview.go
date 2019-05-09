// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"image/color"
	"log"
	"reflect"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

/////////////////////////////////////////////////////////////////////////////
//  ColorView

// ColorView shows a color, using sliders to set values,
type ColorView struct {
	gi.Frame
	Color   gi.Color  `desc:"the color that we view"`
	NumView ValueView `desc:"inline struct view of the numbers"`
	TmpSave ValueView `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	ViewSig ki.Signal `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`
}

var KiT_ColorView = kit.Types.AddType(&ColorView{}, ColorViewProps)

// AddNewColorView adds a new colorview to given parent node, with given name.
func AddNewColorView(parent ki.Ki, name string) *ColorView {
	return parent.AddNewChild(KiT_ColorView, name).(*ColorView)
}

var ColorViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
}

// SetColor sets the source color
func (cv *ColorView) SetColor(clr color.Color, tmpSave ValueView) {
	cv.Color.SetColor(clr)
	cv.Config()
	cv.TmpSave = tmpSave
	cv.Update()
}

// Config configures a standard setup of entire view
func (cv *ColorView) Config() {
	cv.Lay = gi.LayoutVert
	cv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := cv.StdFrameConfig()
	mods, updt := cv.ConfigChildren(config, false)
	if mods {
		cv.SliderLayConfig()
		cv.NumLayConfig()
	} else {
		updt = cv.UpdateStart()
	}
	cv.UpdateEnd(updt)
}

// SliderLayConfig configures the sliders layout
func (cv *ColorView) SliderLayConfig() {
	vl, _ := cv.SliderLay()
	if vl == nil {
		return
	}
	vl.Lay = gi.LayoutHoriz
	vl.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := cv.StdSliderLayConfig()
	mods, updt := vl.ConfigChildren(config, false)
	v, _ := cv.Value()
	if mods {
		cv.ConfigSliderGrid()
		v.SetProp("min-width", units.NewEm(6))
		v.SetProp("min-height", units.NewEm(6))
	} else {
		updt = vl.UpdateStart()
	}
	vl.UpdateEnd(updt)
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (cv *ColorView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Layout, "slider-lay")
	config.Add(gi.KiT_Layout, "num-lay")
	return config
}

func (cv *ColorView) StdSliderLayConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Frame, "value")
	config.Add(gi.KiT_Layout, "slider-grid")
	return config
}

func (cv *ColorView) NumLay() (*gi.Layout, int) {
	idx, ok := cv.Children().IndexByName("num-lay", 1)
	if !ok {
		return nil, -1
	}
	return cv.Child(idx).(*gi.Layout), idx
}

func (cv *ColorView) SliderLay() (*gi.Layout, int) {
	idx, ok := cv.Children().IndexByName("slider-lay", 0)
	if !ok {
		return nil, -1
	}
	return cv.Child(idx).(*gi.Layout), idx
}

func (cv *ColorView) Value() (*gi.Frame, int) {
	vl, _ := cv.SliderLay()
	if vl == nil {
		return nil, -1
	}
	idx, ok := vl.Children().IndexByName("value", 0)
	if !ok {
		return nil, -1
	}
	return vl.Child(idx).(*gi.Frame), idx
}

func (cv *ColorView) SliderGrid() (*gi.Layout, int) {
	vl, _ := cv.SliderLay()
	if vl == nil {
		return nil, -1
	}
	idx, ok := vl.Children().IndexByName("slider-grid", 0)
	if !ok {
		return nil, -1
	}
	return vl.Child(idx).(*gi.Layout), idx
}

// StdSliderConfig returns a TypeAndNameList for configuring a standard sliders
func (cv *ColorView) StdSliderConfig() kit.TypeAndNameList {
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
	config.Add(gi.KiT_Label, "alab")
	config.Add(gi.KiT_Slider, "alpha")
	return config
}

func (cv *ColorView) SetRGBValue(val float32, rgb int) {
	if val > 0 && cv.Color.IsNil() { // starting out with dummy color
		cv.Color.A = 255
	}
	switch rgb {
	case 0:
		cv.Color.R = uint8(val)
	case 1:
		cv.Color.G = uint8(val)
	case 2:
		cv.Color.B = uint8(val)
	case 3:
		cv.Color.A = uint8(val)
	}
	if cv.TmpSave != nil {
		cv.TmpSave.SaveTmp()
	}
}

func (cv *ColorView) ConfigRGBSlider(sl *gi.Slider, rgb int) {
	sl.Defaults()
	sl.Max = 255
	sl.Step = 1
	sl.PageStep = 16
	sl.Prec = 3
	sl.Dim = gi.X
	sl.Tracking = true
	sl.TrackThr = 1
	sl.SetMinPrefWidth(units.NewCh(20))
	sl.SetMinPrefHeight(units.NewEm(2))
	sl.SliderSig.ConnectOnly(cv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.SliderValueChanged) {
			cvv, _ := recv.Embed(KiT_ColorView).(*ColorView)
			slv := send.Embed(gi.KiT_Slider).(*gi.Slider)
			updt := cvv.UpdateStart()
			cvv.SetRGBValue(slv.Value, rgb)
			cvv.ViewSig.Emit(cvv.This(), 0, nil)
			cvv.UpdateEnd(updt)
		}
	})
}

func (cv *ColorView) UpdateRGBSlider(sl *gi.Slider, rgb int) {
	switch rgb {
	case 0:
		sl.SetValue(float32(cv.Color.R))
	case 1:
		sl.SetValue(float32(cv.Color.G))
	case 2:
		sl.SetValue(float32(cv.Color.B))
	case 3:
		sl.SetValue(float32(cv.Color.A))
	}
}

func (cv *ColorView) SetHSLValue(val float32, hsl int) {
	h, s, l, _ := cv.Color.ToHSLA()
	switch hsl {
	case 0:
		h = val
	case 1:
		s = val / 360.0
	case 2:
		l = val / 360.0
	}
	cv.Color.SetHSL(h, s, l)
	if cv.TmpSave != nil {
		cv.TmpSave.SaveTmp()
	}
}

func (cv *ColorView) ConfigHSLSlider(sl *gi.Slider, hsl int) {
	sl.Defaults()
	sl.Max = 360
	sl.Step = 1
	sl.PageStep = 15
	sl.Prec = 3
	sl.Dim = gi.X
	sl.Tracking = true
	sl.TrackThr = 1
	sl.SetMinPrefWidth(units.NewCh(20))
	sl.SetMinPrefHeight(units.NewEm(2))
	sl.SliderSig.ConnectOnly(cv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.SliderValueChanged) {
			cvv, _ := recv.Embed(KiT_ColorView).(*ColorView)
			slv := send.Embed(gi.KiT_Slider).(*gi.Slider)
			updt := cvv.UpdateStart()
			cvv.SetHSLValue(slv.Value, hsl)
			cvv.ViewSig.Emit(cvv.This(), 0, nil)
			cvv.UpdateEnd(updt)
		}
	})
}

func (cv *ColorView) UpdateHSLSlider(sl *gi.Slider, hsl int) {
	h, s, l, _ := cv.Color.ToHSLA()
	switch hsl {
	case 0:
		sl.SetValue(h)
	case 1:
		sl.SetValue(s * 360.0)
	case 2:
		sl.SetValue(l * 360.0)
	}
}

func (cv *ColorView) ConfigLabel(lab *gi.Label, txt string) {
	lab.Text = txt
	lab.Redrawable = true
}

// ConfigSliderGrid configures the SliderGrid
func (cv *ColorView) ConfigSliderGrid() {
	sg, _ := cv.SliderGrid()
	if sg == nil {
		return
	}
	sg.Lay = gi.LayoutGrid
	sg.SetProp("columns", 4)
	config := cv.StdSliderConfig()
	mods, updt := sg.ConfigChildren(config, false)
	if mods {
		cv.ConfigLabel(sg.ChildByName("rlab", 0).Embed(gi.KiT_Label).(*gi.Label), "Red:")
		cv.ConfigLabel(sg.ChildByName("blab", 0).Embed(gi.KiT_Label).(*gi.Label), "Blue")
		cv.ConfigLabel(sg.ChildByName("glab", 0).Embed(gi.KiT_Label).(*gi.Label), "Green:")
		cv.ConfigLabel(sg.ChildByName("hlab", 0).Embed(gi.KiT_Label).(*gi.Label), "Hue:")
		cv.ConfigLabel(sg.ChildByName("slab", 0).Embed(gi.KiT_Label).(*gi.Label), "Sat:")
		cv.ConfigLabel(sg.ChildByName("llab", 0).Embed(gi.KiT_Label).(*gi.Label), "Light:")
		cv.ConfigLabel(sg.ChildByName("alab", 0).Embed(gi.KiT_Label).(*gi.Label), "Alpha:")

		cv.ConfigRGBSlider(sg.ChildByName("red", 0).Embed(gi.KiT_Slider).(*gi.Slider), 0)
		cv.ConfigRGBSlider(sg.ChildByName("green", 0).Embed(gi.KiT_Slider).(*gi.Slider), 1)
		cv.ConfigRGBSlider(sg.ChildByName("blue", 0).Embed(gi.KiT_Slider).(*gi.Slider), 2)
		cv.ConfigRGBSlider(sg.ChildByName("alpha", 0).Embed(gi.KiT_Slider).(*gi.Slider), 3)
		cv.ConfigHSLSlider(sg.ChildByName("hue", 0).Embed(gi.KiT_Slider).(*gi.Slider), 0)
		cv.ConfigHSLSlider(sg.ChildByName("sat", 0).Embed(gi.KiT_Slider).(*gi.Slider), 1)
		cv.ConfigHSLSlider(sg.ChildByName("light", 0).Embed(gi.KiT_Slider).(*gi.Slider), 2)
	} else {
		updt = sg.UpdateStart()
	}
	sg.UpdateEnd(updt)
}

func (cv *ColorView) UpdateSliderGrid() {
	sg, _ := cv.SliderGrid()
	if sg == nil {
		return
	}
	updt := sg.UpdateStart()
	cv.UpdateRGBSlider(sg.ChildByName("red", 0).Embed(gi.KiT_Slider).(*gi.Slider), 0)
	cv.UpdateRGBSlider(sg.ChildByName("green", 0).Embed(gi.KiT_Slider).(*gi.Slider), 1)
	cv.UpdateRGBSlider(sg.ChildByName("blue", 0).Embed(gi.KiT_Slider).(*gi.Slider), 2)
	cv.UpdateRGBSlider(sg.ChildByName("alpha", 0).Embed(gi.KiT_Slider).(*gi.Slider), 3)
	cv.UpdateHSLSlider(sg.ChildByName("hue", 0).Embed(gi.KiT_Slider).(*gi.Slider), 0)
	cv.UpdateHSLSlider(sg.ChildByName("sat", 0).Embed(gi.KiT_Slider).(*gi.Slider), 1)
	cv.UpdateHSLSlider(sg.ChildByName("light", 0).Embed(gi.KiT_Slider).(*gi.Slider), 2)
	sg.UpdateEnd(updt)
}

// NumLayConfig configures the numerical layout
func (cv *ColorView) NumLayConfig() {
	nl, _ := cv.NumLay()
	updt := nl.UpdateStart()
	cv.NumView = ToValueView(&cv.Color, "")
	cv.NumView.SetStandaloneValue(reflect.ValueOf(&cv.Color))
	vtyp := cv.NumView.WidgetType()
	widg := nl.AddNewChild(vtyp, "nums").(gi.Node2D)
	cv.NumView.ConfigWidget(widg)
	vvb := cv.NumView.AsValueViewBase()
	vvb.ViewSig.ConnectOnly(cv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		cvv, _ := recv.Embed(KiT_ColorView).(*ColorView)
		cvv.UpdateSliderGrid()
		cvv.ViewSig.Emit(cvv.This(), 0, nil)
	})
	nl.UpdateEnd(updt)
}

func (cv *ColorView) Update() {
	updt := cv.UpdateStart()
	cv.UpdateSliderGrid()
	cv.NumView.UpdateWidget()
	v, _ := cv.Value()
	v.Sty.Font.BgColor.Color = cv.Color // direct copy
	cv.UpdateEnd(updt)
}

func (cv *ColorView) Render2D() {
	if cv.FullReRenderIfNeeded() {
		return
	}
	if cv.PushBounds() {
		updt := cv.UpdateStart()
		cv.Update()
		cv.UpdateEndNoSig(updt)
		cv.PopBounds()
	}
	cv.Frame.Render2D()
}

////////////////////////////////////////////////////////////////////////////////////////
//  ColorValueView

// ColorValueView presents a StructViewInline for a struct plus a ColorView button..
type ColorValueView struct {
	ValueViewBase
	TmpColor gi.Color
}

var KiT_ColorValueView = kit.Types.AddType(&ColorValueView{}, nil)

// Color returns a standardized color value from whatever value is represented
// internally
func (vv *ColorValueView) Color() (*gi.Color, bool) {
	ok := true
	clri := vv.Value.Interface()
	clr := &vv.TmpColor
	switch c := clri.(type) {
	case gi.Color:
		vv.TmpColor = c
	case *gi.Color:
		clr = c
	case **gi.Color:
		if c != nil {
			// todo: not clear this ever works
			clr = *c
		}
	case color.Color:
		vv.TmpColor.SetColor(c)
	case *color.Color:
		if c != nil {
			vv.TmpColor.SetColor(*c)
		}
	default:
		ok = false
		log.Printf("ColorValueView: could not get color value from type: %T val: %+v\n", c, c)
	}
	return clr, ok
}

// SetColor sets color value from a standard color value -- more robust than
// plain SetValue
func (vv *ColorValueView) SetColor(clr gi.Color) {
	clri := vv.Value.Interface()
	switch c := clri.(type) {
	case gi.Color:
		vv.SetValue(clr)
	case *gi.Color:
		vv.SetValue(clr)
	case **gi.Color:
		vv.SetValue(clr)
	case color.Color:
		vv.SetValue((color.Color)(clr))
	case *color.Color:
		if c != nil {
			vv.SetValue((color.Color)(clr))
		}
	default:
		log.Printf("ColorValueView: could not set color value from type: %T val: %+v\n", c, c)
	}
}

func (vv *ColorValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = KiT_StructViewInline
	return vv.WidgetTyp
}

func (vv *ColorValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sv := vv.Widget.(*StructViewInline)
	clr, ok := vv.Color()
	if ok && clr != nil {
		edack, err := sv.Parts.Children().ElemFromEndTry(0) // action at end, from AddAction above
		if err == nil {
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

	edack, err := sv.Parts.Children().ElemFromEndTry(0) // action at end, from AddAction above
	if err == nil {
		edac := edack.(*gi.Action)
		edac.SetIcon("color")
		edac.Tooltip = "color selection dialog"
		edac.ActionSig.ConnectOnly(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			svv, _ := recv.Embed(KiT_StructViewInline).(*StructViewInline)
			vv.Activate(svv.Viewport, nil, nil)
		})
	}
	sv.ViewSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_ColorValueView).(*ColorValueView)
		vvv.UpdateWidget() // necessary in this case!
		vvv.ViewSig.Emit(vvv.This(), 0, nil)
	})
	vv.UpdateWidget()
}

func (vv *ColorValueView) HasAction() bool {
	return true
}

func (vv *ColorValueView) Activate(vp *gi.Viewport2D, dlgRecv ki.Ki, dlgFunc ki.RecvFunc) {
	if vv.IsInactive() {
		return
	}
	desc, _ := vv.Tag("desc")
	dclr := gi.Color{}
	clr, ok := vv.Color()
	if ok && clr != nil {
		dclr = *clr
	}
	ColorViewDialog(vp, dclr, DlgOpts{Title: "Color Value View", Prompt: desc, TmpSave: vv.TmpSave},
		vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				ddlg := send.Embed(gi.KiT_Dialog).(*gi.Dialog)
				cclr := ColorViewDialogValue(ddlg)
				vv.SetColor(cclr)
				vv.UpdateWidget()
			}
			if dlgRecv != nil && dlgFunc != nil {
				dlgFunc(dlgRecv, send, sig, data)
			}
		})
}
