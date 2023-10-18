// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"image/color"
	"log"
	"log/slog"
	"sort"

	"goki.dev/cam/hsl"
	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/goosi/mimedata"
	"goki.dev/grr"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/laser"
	"goki.dev/mat32/v2"
	"golang.org/x/image/colornames"
)

/////////////////////////////////////////////////////////////////////////////
//  ColorView

// ColorView shows a color, using sliders or numbers to set values.
type ColorView struct {
	gi.Frame

	// the color that we view
	Color color.RGBA

	// the color that we view, in HSLA form
	ColorHSLA hsl.HSL

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave Value `json:"-" xml:"-"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string
}

func (cv *ColorView) OnInit() {
	cv.Lay = gi.LayoutVert
	cv.Style(func(s *styles.Style) {
		cv.Spacing = gi.StdDialogVSpaceUnits
	})
	cv.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(cv.This()) {
		case "value":
			w.Style(func(s *styles.Style) {
				s.MinWidth.SetEm(6)
				s.MinHeight.SetEm(6)
				s.Border.Radius = styles.BorderRadiusFull
				s.BackgroundColor.SetSolid(cv.Color)
			})
		case "slider-grid":
			w.Style(func(s *styles.Style) {
				s.Columns = 4
			})
		case "hexlbl":
			w.Style(func(s *styles.Style) {
				s.AlignV = styles.AlignMiddle
			})
		case "palette":
			w.Style(func(s *styles.Style) {
				s.Columns = 25
			})
		case "nums-hex":
			w.Style(func(s *styles.Style) {
				s.MinWidth.SetCh(20)
			})
		case "num-lay":
			vl := w.(*gi.Layout)
			vl.Style(func(s *styles.Style) {
				vl.Spacing = gi.StdDialogVSpaceUnits
			})
		}
		if sl, ok := w.(*gi.Slider); ok {
			sl.Style(func(s *styles.Style) {
				s.MinWidth.SetCh(20)
				s.Width.SetCh(20)
				s.MinHeight.SetEm(1)
				s.Height.SetEm(1)
				s.Margin.Set(units.Dp(6))
			})
		}
		if w.Parent().Name() == "palette" {
			if cbt, ok := w.(*gi.Button); ok {
				cbt.Style(func(s *styles.Style) {
					c := colornames.Map[cbt.Name()]

					s.BackgroundColor.SetSolid(c)
					s.MaxHeight.SetEm(1.3)
					s.MaxWidth.SetEm(1.3)
					s.Margin.Set()
				})
			}
		}
	})
}

// SetColor sets the source color
func (cv *ColorView) SetColor(clr color.Color) {
	updt := cv.UpdateStart()
	cv.Color = colors.AsRGBA(clr)
	cv.ColorHSLA = hsl.FromColor(clr)
	cv.ColorHSLA.Round()
	cv.UpdateEndRender(updt)
	cv.SendChange()
}

// Config configures a standard setup of entire view
func (cv *ColorView) ConfigWidget(sc *gi.Scene) {
	if cv.HasChildren() {
		return
	}
	updt := cv.UpdateStart()
	vl := gi.NewLayout(cv, "slider-lay").SetLayout(gi.LayoutHoriz)
	nl := gi.NewLayout(cv, "num-lay").SetLayout(gi.LayoutVert)

	rgbalay := gi.NewLayout(nl, "nums-rgba-lay").SetLayout(gi.LayoutHoriz)

	nrgba := NewStructViewInline(rgbalay, "nums-rgba")
	nrgba.SetStruct(&cv.Color)
	nrgba.OnChange(func(e events.Event) {
		cv.SetColor(cv.Color)
	})

	rgbacopy := gi.NewButton(rgbalay, "rgbacopy")
	rgbacopy.Icon = icons.ContentCopy
	rgbacopy.Tooltip = "Copy RGBA Color"
	rgbacopy.Menu = func(m *gi.Scene) {
		gi.NewButton(m).SetText("styles.ColorFromRGB(r, g, b)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("styles.ColorFromRGB(%d, %d, %d)", cv.Color.R, cv.Color.G, cv.Color.B)
			cv.EventMgr().ClipBoard().Write(mimedata.NewText(text))
		})
		gi.NewButton(m).SetText("styles.ColorFromRGBA(r, g, b, a)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("styles.ColorFromRGBA(%d, %d, %d, %d)", cv.Color.R, cv.Color.G, cv.Color.B, cv.Color.A)
			cv.EventMgr().ClipBoard().Write(mimedata.NewText(text))
		})
		gi.NewButton(m).SetText("rgb(r, g, b)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("rgb(%d, %d, %d)", cv.Color.R, cv.Color.G, cv.Color.B)
			cv.EventMgr().ClipBoard().Write(mimedata.NewText(text))
		})
		gi.NewButton(m).SetText("rgba(r, g, b, a)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("rgba(%d, %d, %d, %d)", cv.Color.R, cv.Color.G, cv.Color.B, cv.Color.A)
			cv.EventMgr().ClipBoard().Write(mimedata.NewText(text))
		})
	}

	hslalay := gi.NewLayout(nl, "nums-hsla-lay").SetLayout(gi.LayoutHoriz)

	nhsla := NewStructViewInline(hslalay, "nums-hsla")
	nhsla.SetStruct(&cv.ColorHSLA)
	nhsla.OnChange(func(e events.Event) {
		cv.SetColor(cv.ColorHSLA)
	})

	hslacopy := gi.NewButton(hslalay, "hslacopy")
	hslacopy.Icon = icons.ContentCopy
	hslacopy.Tooltip = "Copy HSLA Color"
	hslacopy.Menu = func(m *gi.Scene) {
		gi.NewButton(m).SetText("styles.ColorFromHSL(h, s, l)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("styles.ColorFromHSL(%g, %g, %g)", cv.ColorHSLA.H, cv.ColorHSLA.S, cv.ColorHSLA.L)
			cv.EventMgr().ClipBoard().Write(mimedata.NewText(text))
		})
		gi.NewButton(m).SetText("styles.ColorFromHSLA(h, s, l, a)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("styles.ColorFromHSLA(%g, %g, %g, %g)", cv.ColorHSLA.H, cv.ColorHSLA.S, cv.ColorHSLA.L, cv.ColorHSLA.A)
			cv.EventMgr().ClipBoard().Write(mimedata.NewText(text))
		})
		gi.NewButton(m).SetText("hsl(h, s, l)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("hsl(%g, %g, %g)", cv.ColorHSLA.H, cv.ColorHSLA.S, cv.ColorHSLA.L)
			cv.EventMgr().ClipBoard().Write(mimedata.NewText(text))
		})
		gi.NewButton(m).SetText("hsla(h, s, l, a)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("hsla(%g, %g, %g, %g)", cv.ColorHSLA.H, cv.ColorHSLA.S, cv.ColorHSLA.L, cv.ColorHSLA.A)
			cv.EventMgr().ClipBoard().Write(mimedata.NewText(text))
		})
	}

	hexlay := gi.NewLayout(nl, "nums-hex-lay").SetLayout(gi.LayoutHoriz)

	gi.NewLabel(hexlay, "hexlbl").SetText("Hex")

	hex := gi.NewTextField(hexlay, "nums-hex")
	hex.Tooltip = "The color in hexadecimal form"
	hex.OnChange(func(e events.Event) {
		cv.SetColor(grr.Log(colors.FromHex(hex.Text())))
	})

	hexcopy := gi.NewButton(hexlay, "hexcopy")
	hexcopy.Icon = icons.ContentCopy
	hexcopy.Tooltip = "Copy Hex Color"
	hexcopy.Menu = func(m *gi.Scene) {
		gi.NewButton(m).SetText(`styles.ColorFromHex("#RRGGBB")`).OnClick(func(e events.Event) {
			hs := colors.AsHex(cv.Color)
			// get rid of transparency because this is just RRGGBB
			text := fmt.Sprintf(`styles.ColorFromHex("%s")`, hs[:len(hs)-2])
			cv.EventMgr().ClipBoard().Write(mimedata.NewText(text))
		})
		gi.NewButton(m).SetText(`styles.ColorFromHex("#RRGGBBAA")`).OnClick(func(e events.Event) {
			text := fmt.Sprintf(`styles.ColorFromHex("%s")`, colors.AsHex(cv.Color))
			cv.EventMgr().ClipBoard().Write(mimedata.NewText(text))
		})
		gi.NewButton(m).SetText("#RRGGBB").OnClick(func(e events.Event) {
			hs := colors.AsHex(cv.Color)
			text := hs[:len(hs)-2]
			cv.EventMgr().ClipBoard().Write(mimedata.NewText(text))
		})
		gi.NewButton(m).SetText("#RRGGBBAA").OnClick(func(e events.Event) {
			text := colors.AsHex(cv.Color)
			cv.EventMgr().ClipBoard().Write(mimedata.NewText(text))
		})
	}

	gi.NewFrame(vl, "value").SetLayout(gi.LayoutHoriz)
	sg := gi.NewLayout(vl, "slider-grid").SetLayout(gi.LayoutGrid)

	gi.NewLabel(sg, "rlab").SetText("Red:")
	rs := gi.NewSlider(sg, "red")
	gi.NewLabel(sg, "hlab").SetText("Hue:")
	hs := gi.NewSlider(sg, "hue")
	gi.NewLabel(sg, "glab").SetText("Green:")
	gs := gi.NewSlider(sg, "green")
	gi.NewLabel(sg, "slab").SetText("Sat:")
	ss := gi.NewSlider(sg, "sat")
	gi.NewLabel(sg, "blab").SetText("Blue:")
	bs := gi.NewSlider(sg, "blue")
	gi.NewLabel(sg, "llab").SetText("Light:")
	ls := gi.NewSlider(sg, "light")
	gi.NewLabel(sg, "alab").SetText("Alpha:")
	as := gi.NewSlider(sg, "alpha")

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
	if val > 0 && colors.IsNil(cv.Color) { // starting out with dummy color
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
}

func (cv *ColorView) ConfigRGBSlider(sl *gi.Slider, rgb int) {
	sl.Max = 255
	sl.Step = 1
	sl.PageStep = 16
	sl.Prec = 3
	sl.Dim = mat32.X
	sl.Tracking = true
	sl.TrackThr = 1
	cv.UpdateRGBSlider(sl, rgb)
	sl.OnChange(func(e events.Event) {
		cv.SetRGBValue(sl.Value, rgb)
		cv.SetColor(cv.Color)
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

func (cv *ColorView) SetHSLValue(val float32, hsln int) {
	switch hsln {
	case 0:
		cv.ColorHSLA.H = val
	case 1:
		cv.ColorHSLA.S = val / 360.0
	case 2:
		cv.ColorHSLA.L = val / 360.0
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
	sl.OnChange(func(e events.Event) {
		cv.SetHSLValue(sl.Value, hsl)
		cv.SetColor(cv.ColorHSLA)
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
	sg.UpdateEndRender(updt)
}

func (cv *ColorView) ConfigPalette() {
	pg := gi.NewLayout(cv, "palette").SetLayout(gi.LayoutGrid)

	// STYTOOD: use hct sorted names here (see https://github.com/goki/gi/issues/619)
	nms := colors.Names

	for _, cn := range nms {
		cbt := gi.NewButton(pg, cn)
		cbt.Tooltip = cn
		cbt.SetText("  ")
		cbt.OnChange(func(e events.Event) {
			cv.SetColor(grr.Log(colors.FromName(cbt.Name())))
		})
	}
}

func (cv *ColorView) Update() {
	updt := cv.UpdateStart()
	cv.UpdateImpl()
	cv.UpdateEndRender(updt)
}

// UpdateImpl does the raw updates based on current value,
// without UpdateStart / End wrapper
func (cv *ColorView) UpdateImpl() {
	cv.UpdateSliderGrid()
	cv.UpdateNums()
	cv.UpdateValueFrame()
	// cv.NumView.UpdateWidget()
	// v := cv.Value()
	// v.Style.BackgroundColor.Solid = cv.Color // direct copy
}

// UpdateValueFrame updates the value frame of the color view
// that displays the color.
func (cv *ColorView) UpdateValueFrame() {
	cv.SliderLay().ChildByName("value", 0).(*gi.Frame).Styles.BackgroundColor.Solid = cv.Color // direct copy
}

// UpdateNums updates the values of the number inputs
// in the color view to reflect the latest values
func (cv *ColorView) UpdateNums() {
	cv.NumLay().ChildByName("nums-rgba-lay", 0).ChildByName("nums-rgba", 0).(*StructViewInline).UpdateFields()
	cv.NumLay().ChildByName("nums-hsla-lay", 1).ChildByName("nums-hsla", 0).(*StructViewInline).UpdateFields()
	hs := colors.AsHex(cv.Color)
	// if we are fully opaque, which is typical,
	// then we can skip displaying transparency in hex
	if cv.Color.A == 255 {
		hs = hs[:len(hs)-2]
	}
	cv.NumLay().ChildByName("nums-hex-lay", 2).ChildByName("nums-hex", 1).(*gi.TextField).SetText(hs)
}

// func (cv *ColorView) Render(sc *gi.Scene) {
// 	if cv.PushBounds(sc) {
// 		cv.FrameStdRender(sc)
// 		cv.RenderChildren(sc)
// 		cv.PopBounds(sc)
// 	}
// }

////////////////////////////////////////////////////////////////////////////////////////
//  ColorValue

// ColorValue presents a StructViewInline for a struct plus a ColorView button..
type ColorValue struct {
	ValueBase
	TmpColor color.RGBA
}

// Color returns a standardized color value from whatever value is represented
// internally
func (vv *ColorValue) Color() (*color.RGBA, bool) {
	ok := true
	clri := vv.Value.Interface()
	clr := &vv.TmpColor
	switch c := clri.(type) {
	case color.RGBA:
		vv.TmpColor = c
	case *color.RGBA:
		clr = c
	case **color.RGBA:
		if c != nil {
			// todo: not clear this ever works
			clr = *c
		}
	case color.Color:
		vv.TmpColor = colors.AsRGBA(c)
	case *color.Color:
		if c != nil {
			vv.TmpColor = colors.AsRGBA(*c)
		}
	default:
		ok = false
		// todo: validation
		slog.Error(fmt.Sprintf("ColorValue: could not get color value from type: %T val: %+v\n", c, c))
	}
	return clr, ok
}

// SetColor sets color value from a standard color value -- more robust than
// plain SetValue
func (vv *ColorValue) SetColor(clr color.RGBA) {
	clri := vv.Value.Interface()
	switch c := clri.(type) {
	case color.RGBA:
		vv.SetValue(clr)
	case *color.RGBA:
		vv.SetValue(clr)
	case **color.RGBA:
		vv.SetValue(clr)
	case color.Color:
		vv.SetValue((color.Color)(clr))
	case *color.Color:
		if c != nil {
			vv.SetValue((color.Color)(clr))
		}
	default:
		log.Printf("ColorValue: could not set color value from type: %T val: %+v\n", c, c)
	}
}

func (vv *ColorValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *ColorValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	bt.SetNeedsRender()
}

func (vv *ColorValue) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	vv.CreateTempIfNotPtr() // we need our value to be a ptr to a struct -- if not make a tmp

	bt.SetText("Edit Color")
	bt.SetIcon(icons.Colors)
	bt.Tooltip = "Open color picker dialog"
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(bt, nil)
	})
	bt.Style(func(s *styles.Style) {
		clr, _ := vv.Color()
		// we need to display button as non-transparent
		// so that it can be seen
		dclr := colors.SetAF32(clr, 1)
		s.BackgroundColor.SetSolid(dclr)
		// TODO: use hct contrast color
		s.Color = colors.AsRGBA(hsl.ContrastColor(dclr))
	})
	vv.UpdateWidget()
}

func (vv *ColorValue) HasButton() bool {
	return true
}

func (vv *ColorValue) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if laser.ValueIsZero(vv.Value) || laser.ValueIsZero(laser.NonPtrValue(vv.Value)) {
		return
	}
	if vv.IsInactive() {
		return
	}
	desc, _ := vv.Desc()
	dclr := color.RGBA{}
	clr, ok := vv.Color()
	if ok && clr != nil {
		dclr = *clr
	}
	ColorViewDialog(ctx, DlgOpts{Title: "Color Value View", Prompt: desc, TmpSave: vv.TmpSave}, dclr, func(dlg *gi.Dialog) {
		if dlg.Accepted {
			cclr := dlg.Data.(color.RGBA)
			vv.SetColor(cclr)
			vv.UpdateWidget()
		}
		if fun != nil {
			fun(dlg)
		}
	}).Run()
}

////////////////////////////////////////////////////////////////////////////////////////
//  ColorNameValue

// ColorNameValue presents an button for displaying a ColorNameName and selecting
// meshes from a ChooserDialog
type ColorNameValue struct {
	ValueBase
}

func (vv *ColorNameValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *ColorNameValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none, click to select)"
	}
	bt.SetText(txt)
}

func (vv *ColorNameValue) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(bt, nil)
	})
	vv.UpdateWidget()
}

func (vv *ColorNameValue) HasButton() bool {
	return true
}

func (vv *ColorNameValue) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsInactive() {
		return
	}
	cur := laser.ToString(vv.Value.Interface())
	sl := make([]struct {
		Name  string
		Color color.RGBA
	}, len(colornames.Map))
	ctr := 0
	for k, v := range colornames.Map {
		sl[ctr].Name = k
		sl[ctr].Color = colors.AsRGBA(v)
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
	desc, _ := vv.Desc()
	TableViewSelectDialog(ctx, DlgOpts{Title: "Select a Color Name", Prompt: desc}, &sl, curRow, nil, func(dlg *gi.Dialog) {
		if dlg.Accepted {
			si := dlg.Data.(int)
			if si >= 0 {
				vv.SetValue(sl[si].Name)
				vv.UpdateWidget()
			}
		}
		if fun != nil {
			fun(dlg)
		}
	}).Run()
}
