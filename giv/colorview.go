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

	"goki.dev/cam/hct"
	"goki.dev/cam/hsl"
	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/laser"
	"golang.org/x/image/colornames"
)

/////////////////////////////////////////////////////////////////////////////
//  ColorView

// ColorView shows a color, using sliders or numbers to set values.
type ColorView struct {
	gi.Frame

	// the color that we view
	Color hct.HCT `set:"-"`

	// // the color that we view
	// Color color.RGBA `set:"-"`

	// // the color that we view, in HSLA form
	// ColorHSLA hsl.HSL `edit:"-"`

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave Value `json:"-" xml:"-"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string
}

// SetColor sets the source color
func (cv *ColorView) SetColor(clr color.Color) *ColorView {
	return cv.SetHCT(hct.FromColor(clr))
}

// SetHCT sets the source color in terms of HCT
func (cv *ColorView) SetHCT(hct hct.HCT) *ColorView {
	updt := cv.UpdateStart()
	cv.Color = hct
	if cv.TmpSave != nil {
		cv.TmpSave.SetValue(cv.Color.AsRGBA())
	}
	cv.Update()
	cv.UpdateEndRender(updt)
	cv.SendChange()
	return cv
}

// Config configures a standard setup of entire view
func (cv *ColorView) ConfigWidget() {
	if cv.HasChildren() {
		return
	}
	updt := cv.UpdateStart()
	cv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	hue := gi.NewSlider(cv, "hue").SetMin(0).SetMax(360).SetValue(cv.Color.Hue)
	hue.OnInput(func(e events.Event) {
		cv.Color.SetHue(hue.Value)
		cv.SetHCT(cv.Color)
	})
	hue.Style(func(s *styles.Style) {
		hue.ValueColor.SetSolid(colors.Transparent)
		hue.ThumbColor.SetSolid(cv.Color)
		s.Min.Y.Em(2)
		s.Min.X.Em(40)
		s.StateLayer = 0 // we don't want any state layer interfering with the way the color looks
		s.BackgroundColor.Gradient = colors.LinearGradient()
		for h := float32(0); h <= 360; h += 5 {
			gc := cv.Color.WithHue(h)
			s.BackgroundColor.Gradient.AddStop(gc.AsRGBA(), h/360, 1)
		}
	})

	chroma := gi.NewSlider(cv, "chroma").SetMin(0).SetMax(150).SetValue(cv.Color.Chroma)
	chroma.OnInput(func(e events.Event) {
		cv.Color.SetChroma(chroma.Value)
		cv.SetHCT(cv.Color)
	})
	chroma.Style(func(s *styles.Style) {
		chroma.ValueColor.SetSolid(colors.Transparent)
		chroma.ThumbColor.SetSolid(cv.Color)
		s.Min.Y.Em(2)
		s.Min.X.Em(40)
		s.StateLayer = 0 // we don't want any state layer interfering with the way the color looks
		s.BackgroundColor.Gradient = colors.LinearGradient()
		for c := float32(0); c <= 150; c += 5 {
			gc := cv.Color.WithChroma(c)
			s.BackgroundColor.Gradient.AddStop(gc.AsRGBA(), c/150, 1)
		}
	})

	tone := gi.NewSlider(cv, "tone").SetMin(0).SetMax(100).SetValue(cv.Color.Tone)
	tone.OnInput(func(e events.Event) {
		cv.Color.SetTone(tone.Value)
		cv.SetHCT(cv.Color)
	})
	tone.Style(func(s *styles.Style) {
		tone.ValueColor.SetSolid(colors.Transparent)
		tone.ThumbColor.SetSolid(cv.Color)
		s.Min.Y.Em(2)
		s.Min.X.Em(40)
		s.StateLayer = 0 // we don't want any state layer interfering with the way the color looks
		s.BackgroundColor.Gradient = colors.LinearGradient()
		for c := float32(0); c <= 100; c += 5 {
			gc := cv.Color.WithTone(c)
			s.BackgroundColor.Gradient.AddStop(gc.AsRGBA(), c/100, 1)
		}
	})

	cv.UpdateEnd(updt)
}

/*
func (cv *ColorView) OnInit() {
	cv.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(cv) {
		case "value":
			w.Style(func(s *styles.Style) {
				s.Min.X.Em(6)
				s.Min.Y.Em(6)
				s.Border.Radius = styles.BorderRadiusFull
				s.BackgroundColor.SetSolid(cv.Color)
			})
		case "slider-grid":
			w.Style(func(s *styles.Style) {
				s.Columns = 4
			})
		case "hexlbl":
			w.Style(func(s *styles.Style) {
				s.Align.Y = styles.Center
			})
		case "palette":
			w.Style(func(s *styles.Style) {
				s.Columns = 25
			})
		case "nums-hex":
			w.Style(func(s *styles.Style) {
				s.Min.X.Ch(20)
			})
		}
		if sl, ok := w.(*gi.Slider); ok {
			sl.Style(func(s *styles.Style) {
				s.Min.X.Ch(20)
				s.Min.Y.Em(1)
				s.Margin.Set(units.Dp(6))
			})
		}
		if w.Parent().Name() == "palette" {
			if cbt, ok := w.(*gi.Button); ok {
				cbt.Style(func(s *styles.Style) {
					c := colornames.Map[cbt.Name()]

					s.BackgroundColor.SetSolid(c)
					s.Max.Set(units.Em(1.3))
					s.Margin.Zero()
				})
			}
		}
	})
}

// SetColor sets the source color
func (cv *ColorView) SetColor(clr color.Color) *ColorView {
	updt := cv.UpdateStart()
	cv.Color = colors.AsRGBA(clr)
	cv.ColorHSLA = hsl.FromColor(clr)
	cv.ColorHSLA.Round()
	cv.UpdateEndRender(updt)
	cv.SendChange()
	return cv
}

// Config configures a standard setup of entire view
func (cv *ColorView) ConfigWidget(sc *gi.Scene) {
	if cv.HasChildren() {
		return
	}
	updt := cv.UpdateStart()
	vl := gi.NewLayout(cv, "slider-lay")
	nl := gi.NewLayout(cv, "num-lay")

	rgbalay := gi.NewLayout(nl, "nums-rgba-lay")

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

	hslalay := gi.NewLayout(nl, "nums-hsla-lay")

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

	hexlay := gi.NewLayout(nl, "nums-hex-lay")

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

	gi.NewFrame(vl, "value")
	sg := gi.NewLayout(vl, "slider-grid").SetDisplay(styles.Grid)

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
	pg := gi.NewLayout(cv, "palette").SetDisplay(styles.Grid)

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

*/

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
	vv.CreateTempIfNotPtr()
	bt := vv.Widget.(*gi.Button)
	bt.SetNeedsRender(true)
}

func (vv *ColorValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	// need TmpSave
	if vv.TmpSave == nil {
		tt, _ := vv.Color()
		vv.TmpSave = NewSoloValue(tt)
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	vv.CreateTempIfNotPtr() // we need our value to be a ptr to a struct -- if not make a tmp

	bt.SetText("Edit color")
	bt.SetIcon(icons.Colors)
	bt.Tooltip = "Open color picker dialog"
	bt.OnClick(func(e events.Event) {
		if !vv.IsReadOnly() {
			vv.OpenDialog(vv.Widget, nil)
		}
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
	bt.Config()
	vv.UpdateWidget()
}

func (vv *ColorValue) HasDialog() bool                      { return true }
func (vv *ColorValue) OpenDialog(ctx gi.Widget, fun func()) { OpenValueDialog(vv, ctx, fun) }

func (vv *ColorValue) ConfigDialog(d *gi.Body) (bool, func()) {
	dclr := color.RGBA{}
	clr, ok := vv.Color()
	if ok && clr != nil {
		dclr = *clr
	}
	NewColorView(d).SetColor(dclr).SetTmpSave(vv.TmpSave)
	return true, func() {
		cclr := vv.TmpSave.Val().Interface().(*color.RGBA)
		vv.SetColor(*cclr)
		vv.UpdateWidget()
	}
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

func (vv *ColorNameValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config()
	bt.OnClick(func(e events.Event) {
		if !vv.IsReadOnly() {
			vv.OpenDialog(vv.Widget, nil)
		}
	})
	vv.UpdateWidget()
}

func (vv *ColorNameValue) HasDialog() bool                      { return true }
func (vv *ColorNameValue) OpenDialog(ctx gi.Widget, fun func()) { OpenValueDialog(vv, ctx, fun) }

func (vv *ColorNameValue) ConfigDialog(d *gi.Body) (bool, func()) {
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
	si := 0
	NewTableView(d).SetSlice(&sl).SetSelIdx(curRow).BindSelectDialog(&si)
	return true, func() {
		if si >= 0 {
			vv.SetValue(sl[si].Name)
			vv.UpdateWidget()
		}
	}
}
