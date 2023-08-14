// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"image/color"
	"log"
	"reflect"
	"sort"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/icons"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
	"golang.org/x/image/colornames"
)

/////////////////////////////////////////////////////////////////////////////
//  ColorView

// ColorView shows a color, using sliders or numbers to set values.
type ColorView struct {
	gi.Frame

	// the color that we view
	Color gist.Color `desc:"the color that we view"`

	// the color that we view, in HSLA form
	ColorHSLA gist.HSLA `desc:"the color that we view, in HSLA form"`

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave ValueView `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`

	// signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update
	ViewSig ki.Signal `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`

	// manipulating signal -- this is sent when sliders are being manipulated -- ViewSig is only sent at end for final selected value
	ManipSig ki.Signal `json:"-" xml:"-" desc:"manipulating signal -- this is sent when sliders are being manipulated -- ViewSig is only sent at end for final selected value"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string `desc:"a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows"`
}

var TypeColorView = kit.Types.AddType(&ColorView{}, ColorViewProps)

// AddNewColorView adds a new colorview to given parent node, with given name.
func AddNewColorView(parent ki.Ki, name string) *ColorView {
	return parent.AddNewChild(TypeColorView, name).(*ColorView)
}

func (cv *ColorView) OnInit() {
	cv.Lay = gi.LayoutVert
	cv.AddStyler(gi.StylerDefault, func() {
		cv.Spacing = gi.StdDialogVSpaceUnits

		s := &cv.Style
		s.BackgroundColor = cv.ParentBackgroundColor()
	})
}

func (cv *ColorView) OnChildAdded(child ki.Ki) {
	switch child.Name() {
	case "value":
		v := child.(*gi.Frame)
		v.AddStyler(gi.StylerParent(cv), func() {
			s := &v.Style
			s.MinWidth.SetEm(6)
			s.MinHeight.SetEm(6)
			s.Border.Radius = gist.BorderRadiusFull
			s.BackgroundColor.SetColor(cv.Color)
		})
	case "slider-grid":
		sg := child.(*gi.Layout)
		sg.AddStyler(gi.StylerParent(cv), func() {
			s := &sg.Style
			s.Columns = 4
		})
	case "hexlbl":
		hexlbl := child.(*gi.Label)
		hexlbl.AddStyler(gi.StylerParent(cv), func() {
			s := &hexlbl.Style
			s.AlignV = gist.AlignMiddle
		})
	case "palette":
		pg := child.(*gi.Layout)
		pg.AddStyler(gi.StylerParent(pg), func() {
			s := &pg.Style
			s.Columns = 25
		})
	case "nums-hex":
		hex := child.(*gi.TextField)
		hex.AddStyler(gi.StylerParent(cv), func() {
			s := &hex.Style
			s.MinWidth.SetCh(20)
		})
	case "num-lay":
		vl := child.(*gi.Layout)
		vl.AddStyler(gi.StylerParent(cv), func() {
			vl.Spacing = gi.StdDialogVSpaceUnits
		})
	}
	if sl, ok := child.(*gi.Slider); ok {
		sl.AddStyler(gi.StylerParent(cv), func() {
			s := &sl.Style
			s.MinWidth.SetCh(20)
			s.Width.SetCh(20)
			s.MinHeight.SetEm(1)
			s.Height.SetEm(1)
			s.Margin.Set(units.Px(6 * gi.Prefs.DensityMul()))
		})
	}
	if child.Parent().Name() == "palette" {
		if cbt, ok := child.(*gi.Button); ok {
			cbt.AddStyler(gi.StylerParent(cv), func() {
				c := colornames.Map[cbt.Name()]
				s := &cbt.Style
				s.BackgroundColor.SetColor(c)
				s.MaxHeight.SetEm(1.3)
				s.MaxWidth.SetEm(1.3)
				s.Margin.Set()
			})
		}
	}
}

func (cv *ColorView) Disconnect() {
	cv.Frame.Disconnect()
	cv.ViewSig.DisconnectAll()
	cv.ManipSig.DisconnectAll()
}

var ColorViewProps = ki.Props{
	ki.EnumTypeFlag: gi.TypeNodeFlags,
}

// SetColor sets the source color
func (cv *ColorView) SetColor(clr color.Color) {
	cv.Color.SetColor(clr)
	cv.ColorHSLA = gist.HSLAModel.Convert(clr).(gist.HSLA)
	cv.ColorHSLA.Round()
	cv.Config()
	cv.Update()
}

// Config configures a standard setup of entire view
func (cv *ColorView) Config() {
	if cv.HasChildren() {
		return
	}
	updt := cv.UpdateStart()
	vl := gi.AddNewLayout(cv, "slider-lay", gi.LayoutHoriz)
	nl := gi.AddNewLayout(cv, "num-lay", gi.LayoutVert)

	// cv.NumView = ToValueView(&cv.Color, "")
	// cv.NumView.SetSoloValue(reflect.ValueOf(&cv.Color))
	// vtyp := cv.NumView.WidgetType()
	// widg := nl.AddNewChild(vtyp, "nums").(gi.Node2D)
	// cv.NumView.ConfigWidget(widg)
	// vvb := cv.NumView.AsValueViewBase()
	// vvb.ViewSig.ConnectOnly(cv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	cvv, _ := recv.Embed(TypeColorView).(*ColorView)
	// 	cvv.ColorHSLA = gist.HSLAModel.Convert(cvv.Color).(gist.HSLA)
	// 	cvv.UpdateSliderGrid()
	// 	cvv.ViewSig.Emit(cvv.This(), 0, nil)
	// })

	rgbalay := gi.AddNewLayout(nl, "nums-rgba-lay", gi.LayoutHoriz)

	nrgba := AddNewStructViewInline(rgbalay, "nums-rgba")
	nrgba.SetStruct(&cv.Color)
	nrgba.ViewSig.ConnectOnly(cv.This(), func(recv, send ki.Ki, sig int64, data any) {
		cvv, _ := recv.Embed(TypeColorView).(*ColorView)
		updt := cvv.UpdateStart()
		cvv.ColorHSLA = gist.HSLAModel.Convert(cv.Color).(gist.HSLA)
		cvv.ColorHSLA.Round()
		cvv.ViewSig.Emit(cvv.This(), 0, nil)
		cvv.UpdateEnd(updt)
	})

	rgbacopy := gi.AddNewButton(rgbalay, "rgbacopy")
	rgbacopy.Icon = icons.ContentCopy
	rgbacopy.Tooltip = "Copy RGBA Color"
	rgbacopy.Menu.AddAction(gi.ActOpts{Label: "gist.ColorFromRGB(r, g, b)"},
		cv.This(), func(recv, send ki.Ki, sig int64, data any) {
			cvv := recv.(*ColorView)
			text := fmt.Sprintf("gist.ColorFromRGB(%d, %d, %d)", cvv.Color.R, cvv.Color.G, cvv.Color.B)
			oswin.TheApp.ClipBoard(cv.ParentWindow().OSWin).Write(mimedata.NewText(text))
		})
	rgbacopy.Menu.AddAction(gi.ActOpts{Label: "gist.ColorFromRGBA(r, g, b, a)"},
		cv.This(), func(recv, send ki.Ki, sig int64, data any) {
			cvv := recv.(*ColorView)
			text := fmt.Sprintf("gist.ColorFromRGBA(%d, %d, %d, %d)", cvv.Color.R, cvv.Color.G, cvv.Color.B, cvv.Color.A)
			oswin.TheApp.ClipBoard(cv.ParentWindow().OSWin).Write(mimedata.NewText(text))
		})
	rgbacopy.Menu.AddAction(gi.ActOpts{Label: "rgb(r, g, b)"},
		cv.This(), func(recv, send ki.Ki, sig int64, data any) {
			cvv := recv.(*ColorView)
			text := fmt.Sprintf("rgb(%d, %d, %d)", cvv.Color.R, cvv.Color.G, cvv.Color.B)
			oswin.TheApp.ClipBoard(cv.ParentWindow().OSWin).Write(mimedata.NewText(text))
		})
	rgbacopy.Menu.AddAction(gi.ActOpts{Label: "rgba(r, g, b, a)"},
		cv.This(), func(recv, send ki.Ki, sig int64, data any) {
			cvv := recv.(*ColorView)
			text := fmt.Sprintf("rgba(%d, %d, %d, %d)", cvv.Color.R, cvv.Color.G, cvv.Color.B, cvv.Color.A)
			oswin.TheApp.ClipBoard(cv.ParentWindow().OSWin).Write(mimedata.NewText(text))
		})

	hslalay := gi.AddNewLayout(nl, "nums-hsla-lay", gi.LayoutHoriz)

	nhsla := AddNewStructViewInline(hslalay, "nums-hsla")
	nhsla.SetStruct(&cv.ColorHSLA)
	nhsla.ViewSig.ConnectOnly(cv.This(), func(recv, send ki.Ki, sig int64, data any) {
		cvv, _ := recv.Embed(TypeColorView).(*ColorView)
		updt := cvv.UpdateStart()
		cvv.Color = gist.ColorModel.Convert(cv.ColorHSLA).(gist.Color)
		cvv.ViewSig.Emit(cvv.This(), 0, nil)
		cvv.UpdateEnd(updt)
	})

	hslacopy := gi.AddNewButton(hslalay, "hslacopy")
	hslacopy.Icon = icons.ContentCopy
	hslacopy.Tooltip = "Copy HSLA Color"
	hslacopy.Menu.AddAction(gi.ActOpts{Label: "gist.ColorFromHSL(h, s, l)"},
		cv.This(), func(recv, send ki.Ki, sig int64, data any) {
			cvv := recv.(*ColorView)
			text := fmt.Sprintf("gist.ColorFromHSL(%g, %g, %g)", cvv.ColorHSLA.H, cvv.ColorHSLA.S, cvv.ColorHSLA.L)
			oswin.TheApp.ClipBoard(cv.ParentWindow().OSWin).Write(mimedata.NewText(text))
		})
	hslacopy.Menu.AddAction(gi.ActOpts{Label: "gist.ColorFromHSLA(h, s, l, a)"},
		cv.This(), func(recv, send ki.Ki, sig int64, data any) {
			cvv := recv.(*ColorView)
			text := fmt.Sprintf("gist.ColorFromHSLA(%g, %g, %g, %g)", cvv.ColorHSLA.H, cvv.ColorHSLA.S, cvv.ColorHSLA.L, cvv.ColorHSLA.A)
			oswin.TheApp.ClipBoard(cv.ParentWindow().OSWin).Write(mimedata.NewText(text))
		})
	hslacopy.Menu.AddAction(gi.ActOpts{Label: "hsl(h, s, l)"},
		cv.This(), func(recv, send ki.Ki, sig int64, data any) {
			cvv := recv.(*ColorView)
			text := fmt.Sprintf("hsl(%g, %g, %g)", cvv.ColorHSLA.H, cvv.ColorHSLA.S, cvv.ColorHSLA.L)
			oswin.TheApp.ClipBoard(cv.ParentWindow().OSWin).Write(mimedata.NewText(text))
		})
	hslacopy.Menu.AddAction(gi.ActOpts{Label: "hsla(h, s, l, a)"},
		cv.This(), func(recv, send ki.Ki, sig int64, data any) {
			cvv := recv.(*ColorView)
			text := fmt.Sprintf("hsla(%g, %g, %g, %g)", cvv.ColorHSLA.H, cvv.ColorHSLA.S, cvv.ColorHSLA.L, cvv.ColorHSLA.A)
			oswin.TheApp.ClipBoard(cv.ParentWindow().OSWin).Write(mimedata.NewText(text))
		})

	hexlay := gi.AddNewLayout(nl, "nums-hex-lay", gi.LayoutHoriz)

	gi.AddNewLabel(hexlay, "hexlbl", "Hex")

	hex := gi.AddNewTextField(hexlay, "nums-hex")
	hex.Tooltip = "The color in hexadecimal form"
	hex.TextFieldSig.ConnectOnly(cv.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(gi.TextFieldDone) || sig == int64(gi.TextFieldDeFocused) {
			cvv, _ := recv.Embed(TypeColorView).(*ColorView)
			updt := cvv.UpdateStart()
			err := cvv.Color.ParseHex(hex.Text())
			if err != nil {
				log.Println("color view: error parsing hex '"+hex.Text()+"':", err)
			}
			cvv.ColorHSLA = gist.HSLAModel.Convert(cv.Color).(gist.HSLA)
			cvv.ColorHSLA.Round()
			cvv.ViewSig.Emit(cvv.This(), 0, nil)
			cvv.UpdateEnd(updt)
		}
	})

	hexcopy := gi.AddNewButton(hexlay, "hexcopy")
	hexcopy.Icon = icons.ContentCopy
	hexcopy.Tooltip = "Copy Hex Color"
	hexcopy.Menu.AddAction(gi.ActOpts{Label: `gist.ColorFromHex("#RRGGBB")`},
		cv.This(), func(recv, send ki.Ki, sig int64, data any) {
			cvv := recv.(*ColorView)
			hs := cvv.Color.HexString()
			// get rid of transparency because this is just RRGGBB
			text := fmt.Sprintf(`gist.ColorFromHex("%s")`, hs[:len(hs)-2])
			oswin.TheApp.ClipBoard(cv.ParentWindow().OSWin).Write(mimedata.NewText(text))
		})
	hexcopy.Menu.AddAction(gi.ActOpts{Label: `gist.ColorFromHex("#RRGGBBAA")`},
		cv.This(), func(recv, send ki.Ki, sig int64, data any) {
			cvv := recv.(*ColorView)
			text := fmt.Sprintf(`gist.ColorFromHex("%s")`, cvv.Color.HexString())
			oswin.TheApp.ClipBoard(cv.ParentWindow().OSWin).Write(mimedata.NewText(text))
		})
	hexcopy.Menu.AddAction(gi.ActOpts{Label: "#RRGGBB"},
		cv.This(), func(recv, send ki.Ki, sig int64, data any) {
			cvv := recv.(*ColorView)
			hs := cvv.Color.HexString()
			text := hs[:len(hs)-2]
			oswin.TheApp.ClipBoard(cv.ParentWindow().OSWin).Write(mimedata.NewText(text))
		})
	hexcopy.Menu.AddAction(gi.ActOpts{Label: "#RRGGBBAA"},
		cv.This(), func(recv, send ki.Ki, sig int64, data any) {
			cvv := recv.(*ColorView)
			text := cvv.Color.HexString()
			oswin.TheApp.ClipBoard(cv.ParentWindow().OSWin).Write(mimedata.NewText(text))
		})

	gi.AddNewFrame(vl, "value", gi.LayoutHoriz)
	sg := gi.AddNewLayout(vl, "slider-grid", gi.LayoutGrid)

	rl := gi.AddNewLabel(sg, "rlab", "Red:")
	rs := gi.AddNewSlider(sg, "red")
	hl := gi.AddNewLabel(sg, "hlab", "Hue:")
	hs := gi.AddNewSlider(sg, "hue")
	gl := gi.AddNewLabel(sg, "glab", "Green:")
	gs := gi.AddNewSlider(sg, "green")
	sl := gi.AddNewLabel(sg, "slab", "Sat:")
	ss := gi.AddNewSlider(sg, "sat")
	bl := gi.AddNewLabel(sg, "blab", "Blue:")
	bs := gi.AddNewSlider(sg, "blue")
	ll := gi.AddNewLabel(sg, "llab", "Light:")
	ls := gi.AddNewSlider(sg, "light")
	al := gi.AddNewLabel(sg, "alab", "Alpha:")
	as := gi.AddNewSlider(sg, "alpha")

	rl.Redrawable = true
	gl.Redrawable = true
	bl.Redrawable = true
	hl.Redrawable = true
	sl.Redrawable = true
	ll.Redrawable = true
	al.Redrawable = true

	cv.ConfigRGBSlider(rs, 0)
	cv.ConfigRGBSlider(gs, 1)
	cv.ConfigRGBSlider(bs, 2)
	cv.ConfigRGBSlider(as, 3)
	cv.ConfigHSLSlider(hs, 0)
	cv.ConfigHSLSlider(ss, 1)
	cv.ConfigHSLSlider(ls, 2)

	cv.ConfigPalette()

	cv.UpdateEnd(updt)
}

// IsConfiged returns true if widget is fully configured
func (cv *ColorView) IsConfiged() bool {
	if !cv.HasChildren() {
		return false
	}
	sl := cv.SliderLay()
	return sl.HasChildren()
}

func (cv *ColorView) NumLay() *gi.Layout {
	return cv.ChildByName("num-lay", 1).(*gi.Layout)
}

func (cv *ColorView) SliderLay() *gi.Layout {
	return cv.ChildByName("slider-lay", 0).(*gi.Layout)
}

func (cv *ColorView) Value() *gi.Frame {
	return cv.SliderLay().ChildByName("value", 0).(*gi.Frame)
}

func (cv *ColorView) SliderGrid() *gi.Layout {
	return cv.SliderLay().ChildByName("slider-grid", 0).(*gi.Layout)
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
	cv.ColorHSLA = gist.HSLAModel.Convert(cv.Color).(gist.HSLA)
	cv.ColorHSLA.Round()
	if cv.TmpSave != nil {
		cv.TmpSave.SaveTmp()
	}
}

func (cv *ColorView) ConfigRGBSlider(sl *gi.Slider, rgb int) {
	sl.Max = 255
	sl.Step = 1
	sl.PageStep = 16
	sl.Prec = 3
	sl.Dim = mat32.X
	sl.Tracking = true
	sl.TrackThr = 1
	sl.SliderSig.ConnectOnly(cv.This(), func(recv, send ki.Ki, sig int64, data any) {
		cvv, _ := recv.Embed(TypeColorView).(*ColorView)
		slv := send.Embed(gi.TypeSlider).(*gi.Slider)
		if sig == int64(gi.SliderReleased) {
			updt := cvv.UpdateStart()
			cvv.SetRGBValue(slv.Value, rgb)
			cvv.ViewSig.Emit(cvv.This(), 0, nil)
			cvv.UpdateEnd(updt)
		} else if sig == int64(gi.SliderValueChanged) {
			updt := cvv.UpdateStart()
			cvv.SetRGBValue(slv.Value, rgb)
			cvv.ManipSig.Emit(cvv.This(), 0, nil)
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
		cv.ColorHSLA.H = h
	case 1:
		s = val / 360.0
		cv.ColorHSLA.S = s
	case 2:
		l = val / 360.0
		cv.ColorHSLA.L = l
	}
	cv.ColorHSLA.Round()
	cv.Color.SetHSL(h, s, l)
	if cv.TmpSave != nil {
		cv.TmpSave.SaveTmp()
	}
}

func (cv *ColorView) ConfigHSLSlider(sl *gi.Slider, hsl int) {
	sl.Max = 360
	sl.Step = 1
	sl.PageStep = 15
	sl.Prec = 3
	sl.Dim = mat32.X
	sl.Tracking = true
	sl.TrackThr = 1
	sl.SliderSig.ConnectOnly(cv.This(), func(recv, send ki.Ki, sig int64, data any) {
		cvv, _ := recv.Embed(TypeColorView).(*ColorView)
		slv := send.Embed(gi.TypeSlider).(*gi.Slider)
		if sig == int64(gi.SliderReleased) {
			updt := cvv.UpdateStart()
			cvv.SetHSLValue(slv.Value, hsl)
			cvv.ViewSig.Emit(cvv.This(), 0, nil)
			cvv.UpdateEnd(updt)
		} else if sig == int64(gi.SliderValueChanged) {
			updt := cvv.UpdateStart()
			cvv.SetHSLValue(slv.Value, hsl)
			cvv.ManipSig.Emit(cvv.This(), 0, nil)
			cvv.UpdateEnd(updt)
		}
	})
}

func (cv *ColorView) UpdateHSLSlider(sl *gi.Slider, hsl int) {
	switch hsl {
	case 0:
		sl.SetValue(cv.ColorHSLA.H)
	case 1:
		sl.SetValue(cv.ColorHSLA.S * 360.0)
	case 2:
		sl.SetValue(cv.ColorHSLA.L * 360.0)
	}
}

func (cv *ColorView) UpdateSliderGrid() {
	sg := cv.SliderGrid()
	updt := sg.UpdateStart()
	cv.UpdateRGBSlider(sg.ChildByName("red", 0).(*gi.Slider), 0)
	cv.UpdateRGBSlider(sg.ChildByName("green", 0).(*gi.Slider), 1)
	cv.UpdateRGBSlider(sg.ChildByName("blue", 0).(*gi.Slider), 2)
	cv.UpdateRGBSlider(sg.ChildByName("alpha", 0).(*gi.Slider), 3)
	cv.UpdateHSLSlider(sg.ChildByName("hue", 0).(*gi.Slider), 0)
	cv.UpdateHSLSlider(sg.ChildByName("sat", 0).(*gi.Slider), 1)
	cv.UpdateHSLSlider(sg.ChildByName("light", 0).(*gi.Slider), 2)
	sg.UpdateEnd(updt)
}

func (cv *ColorView) ConfigPalette() {
	pg := gi.AddNewLayout(cv, "palette", gi.LayoutGrid)

	nms := gist.HSLSortedColorNames()

	for _, cn := range nms {
		cbt := gi.AddNewButton(pg, cn)
		cbt.Tooltip = cn
		cbt.SetText("  ")
		cbt.ButtonSig.Connect(cv.This(), func(recv, send ki.Ki, sig int64, data any) {
			cvv, _ := recv.Embed(TypeColorView).(*ColorView)
			if sig == int64(gi.ButtonPressed) {
				but := send.Embed(gi.TypeButton).(*gi.Button)
				cvv.Color.SetName(but.Nm)
				cvv.ColorHSLA = gist.HSLAModel.Convert(cvv.Color).(gist.HSLA)
				cvv.ColorHSLA.Round()
				cvv.ViewSig.Emit(cvv.This(), 0, nil)
				cvv.Update()
			}
		})
	}
}

func (cv *ColorView) Update() {
	updt := cv.UpdateStart()
	cv.UpdateImpl()
	cv.UpdateEnd(updt)
}

// UpdateImpl does the raw updates based on current value,
// without UpdateStart / End wrapper
func (cv *ColorView) UpdateImpl() {
	cv.UpdateSliderGrid()
	cv.UpdateNums()
	cv.UpdateValueFrame()
	// cv.NumView.UpdateWidget()
	// v := cv.Value()
	// v.Style.BackgroundColor.Color = cv.Color // direct copy
}

// UpdateValueFrame updates the value frame of the color view
// that displays the color.
func (cv *ColorView) UpdateValueFrame() {
	cv.SliderLay().ChildByName("value", 0).(*gi.Frame).Style.BackgroundColor.Color = cv.Color // direct copy
}

// UpdateNums updates the values of the number inputs
// in the color view to reflect the latest values
func (cv *ColorView) UpdateNums() {
	cv.NumLay().ChildByName("nums-rgba-lay", 0).ChildByName("nums-rgba", 0).(*StructViewInline).UpdateFields()
	cv.NumLay().ChildByName("nums-hsla-lay", 1).ChildByName("nums-hsla", 0).(*StructViewInline).UpdateFields()
	hs := cv.Color.HexString()
	// if we are fully opaque, which is typical,
	// then we can skip displaying transparency in hex
	if cv.Color.A == 255 {
		hs = hs[:len(hs)-2]
	}
	cv.NumLay().ChildByName("nums-hex-lay", 2).ChildByName("nums-hex", 1).(*gi.TextField).SetText(hs)
}

func (cv *ColorView) Render2D() {
	if cv.FullReRenderIfNeeded() {
		return
	}
	if cv.PushBounds() {
		updt := cv.UpdateStart()
		cv.UpdateImpl()
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
	TmpColor gist.Color
}

var TypeColorValueView = kit.Types.AddType(&ColorValueView{}, nil)

// AddNewColorValueView adds a new color value view to given parent node, with given name.
func AddNewColorValueView(parent ki.Ki, name string) *ColorValueView {
	return parent.AddNewChild(TypeColorValueView, name).(*ColorValueView)
}

// Color returns a standardized color value from whatever value is represented
// internally
func (vv *ColorValueView) Color() (*gist.Color, bool) {
	ok := true
	clri := vv.Value.Interface()
	clr := &vv.TmpColor
	switch c := clri.(type) {
	case gist.Color:
		vv.TmpColor = c
	case *gist.Color:
		clr = c
	case **gist.Color:
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
func (vv *ColorValueView) SetColor(clr gist.Color) {
	clri := vv.Value.Interface()
	switch c := clri.(type) {
	case gist.Color:
		vv.SetValue(clr)
	case *gist.Color:
		vv.SetValue(clr)
	case **gist.Color:
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
	vv.WidgetTyp = TypeStructViewInline
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
			edac.AddStyler(gi.StylerParent(vv), func() {
				// we need to display button as non-transparent
				// so that it can be seen
				dclr := *clr
				dclr.A = 255
				edac.Style.BackgroundColor.SetColor(dclr)
				edac.Style.Color = (dclr).ContrastColor()
			})
			edac.SetFullReRender()
		}
	}
	sv.UpdateFields()
}

func (vv *ColorValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	sv := vv.Widget.(*StructViewInline)
	// fmt.Println(reflect.TypeOf(sv.Parent()))
	// sv.AddAction = reflect.TypeOf(sv.Parent().Parent()) != TypeColorView
	sv.AddAction = true
	sv.ViewPath = vv.ViewPath
	sv.TmpSave = vv.TmpSave
	vv.CreateTempIfNotPtr() // we need our value to be a ptr to a struct -- if not make a tmp
	sv.SetStruct(vv.Value.Interface())

	edack, err := sv.Parts.Children().ElemFromEndTry(0) // action at end, from AddAction above
	if err == nil {
		edac := edack.(*gi.Action)
		edac.SetIcon(icons.Colors)
		edac.Tooltip = "color selection dialog"
		edac.ActionSig.ConnectOnly(sv.This(), func(recv, send ki.Ki, sig int64, data any) {
			svv, _ := recv.Embed(TypeStructViewInline).(*StructViewInline)
			vv.Activate(svv.ViewportSafe(), nil, nil)
		})
	}
	sv.ViewSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data any) {
		vvv, _ := recv.Embed(TypeColorValueView).(*ColorValueView)
		vvv.UpdateWidget() // necessary in this case!
		vvv.ViewSig.Emit(vvv.This(), 0, nil)
	})
	vv.UpdateWidget()
}

func (vv *ColorValueView) HasAction() bool {
	return true
}

func (vv *ColorValueView) Activate(vp *gi.Viewport2D, dlgRecv ki.Ki, dlgFunc ki.RecvFunc) {
	if kit.ValueIsZero(vv.Value) || kit.ValueIsZero(kit.NonPtrValue(vv.Value)) {
		return
	}
	if vv.IsInactive() {
		return
	}
	desc, _ := vv.Tag("desc")
	dclr := gist.Color{}
	clr, ok := vv.Color()
	if ok && clr != nil {
		dclr = *clr
	}
	ColorViewDialog(vp, dclr, DlgOpts{Title: "Color Value View", Prompt: desc, TmpSave: vv.TmpSave},
		vv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.DialogAccepted) {
				ddlg := send.Embed(gi.TypeDialog).(*gi.Dialog)
				cclr := ColorViewDialogValue(ddlg)
				vv.SetColor(cclr)
				vv.UpdateWidget()
			}
			if dlgRecv != nil && dlgFunc != nil {
				dlgFunc(dlgRecv, send, sig, data)
			}
		})
}

////////////////////////////////////////////////////////////////////////////////////////
//  ColorNameValueView

// ColorNameValueView presents an action for displaying a ColorNameName and selecting
// meshes from a ChooserDialog
type ColorNameValueView struct {
	ValueViewBase
}

var TypeColorNameValueView = kit.Types.AddType(&ColorNameValueView{}, nil)

func (vv *ColorNameValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.TypeAction
	return vv.WidgetTyp
}

func (vv *ColorNameValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Action)
	txt := kit.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none, click to select)"
	}
	ac.SetText(txt)
}

func (vv *ColorNameValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	ac := vv.Widget.(*gi.Action)
	ac.AddStyler(gi.StylerParent(vv), func() {
		ac.Style.Border.Radius = gist.BorderRadiusFull
	})
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data any) {
		vvv, _ := recv.Embed(TypeColorNameValueView).(*ColorNameValueView)
		ac := vvv.Widget.(*gi.Action)
		vvv.Activate(ac.ViewportSafe(), nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *ColorNameValueView) HasAction() bool {
	return true
}

func (vv *ColorNameValueView) Activate(vp *gi.Viewport2D, dlgRecv ki.Ki, dlgFunc ki.RecvFunc) {
	if vv.IsInactive() {
		return
	}
	cur := kit.ToString(vv.Value.Interface())
	sl := make([]struct {
		Name  string
		Color gist.Color
	}, len(colornames.Map))
	ctr := 0
	for k, v := range colornames.Map {
		sl[ctr].Name = k
		sl[ctr].Color.SetColor(v)
		ctr++
	}
	sort.Slice(sl, func(i, j int) bool {
		return sl[i].Name < sl[j].Name
	})
	curRow := -1
	for i := range sl {
		if sl[i].Name == cur {
			curRow = i
		}
	}
	desc, _ := vv.Tag("desc")
	TableViewSelectDialog(vp, &sl, DlgOpts{Title: "Select a Color Name", Prompt: desc}, curRow, nil,
		vv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.DialogAccepted) {
				ddlg := send.Embed(gi.TypeDialog).(*gi.Dialog)
				si := TableViewSelectDialogValue(ddlg)
				if si >= 0 {
					vv.SetValue(sl[si].Name)
					vv.UpdateWidget()
				}
			}
			if dlgRecv != nil && dlgFunc != nil {
				dlgFunc(dlgRecv, send, sig, data)
			}
		})
}
