// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"image"
	"image/color"

	"cogentcore.org/core/cam/hct"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/styles"
)

/////////////////////////////////////////////////////////////////////////////
//  ColorView

// ColorView shows a color, using sliders or numbers to set values.
type ColorView struct {
	gi.Frame

	// the color that we view
	Color hct.HCT `set:"-"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string
}

// SetColor sets the source color
func (cv *ColorView) SetColor(clr color.Color) *ColorView {
	return cv.SetHCT(hct.FromColor(clr))
}

// SetHCT sets the source color in terms of HCT
func (cv *ColorView) SetHCT(hct hct.HCT) *ColorView {
	cv.Color = hct
	cv.Update()
	cv.SendChange()
	return cv
}

func (cv *ColorView) OnInit() {
	cv.Frame.OnInit()
}

// Config configures a standard setup of entire view
func (cv *ColorView) Config() {
	if cv.HasChildren() {
		return
	}

	cv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	sf := func(s *styles.Style) {
		s.Min.Y.Em(2)
		s.Min.X.Em(1)
		s.Max.X.Em(40)
		s.Grow.Set(1, 0)
	}

	hue := gi.NewSlider(cv, "hue").SetMin(0).SetMax(360).SetValue(cv.Color.Hue).
		SetTooltip("The hue, which is the spectral identity of the color (red, green, blue, etc) in degrees")
	hue.OnInput(func(e events.Event) {
		cv.Color.SetHue(hue.Value)
		cv.SetHCT(cv.Color)
	})
	hue.Style(func(s *styles.Style) {
		hue.ValueColor = nil
		hue.ThumbColor = colors.C(cv.Color)
		g := gradient.NewLinear()
		for h := float32(0); h <= 360; h += 5 {
			gc := cv.Color.WithHue(h)
			g.AddStop(gc.AsRGBA(), h/360)
		}
		s.Background = g
	})
	hue.StyleFinal(sf)

	chroma := gi.NewSlider(cv, "chroma").SetMin(0).SetMax(150).SetValue(cv.Color.Chroma).
		SetTooltip("The chroma, which is the colorfulness/saturation of the color")
	chroma.OnInput(func(e events.Event) {
		cv.Color.SetChroma(chroma.Value)
		cv.SetHCT(cv.Color)
	})
	chroma.Style(func(s *styles.Style) {
		chroma.ValueColor = nil
		chroma.ThumbColor = colors.C(cv.Color)
		g := gradient.NewLinear()
		for c := float32(0); c <= 150; c += 5 {
			gc := cv.Color.WithChroma(c)
			g.AddStop(gc.AsRGBA(), c/150)
		}
		s.Background = g
	})
	chroma.StyleFinal(sf)

	tone := gi.NewSlider(cv, "tone").SetMin(0).SetMax(100).SetValue(cv.Color.Tone).
		SetTooltip("The tone, which is the lightness of the color")
	tone.OnInput(func(e events.Event) {
		cv.Color.SetTone(tone.Value)
		cv.SetHCT(cv.Color)
	})
	tone.Style(func(s *styles.Style) {
		tone.ValueColor = nil
		tone.ThumbColor = colors.C(cv.Color)
		g := gradient.NewLinear()
		for c := float32(0); c <= 100; c += 5 {
			gc := cv.Color.WithTone(c)
			g.AddStop(gc.AsRGBA(), c/100)
		}
		s.Background = g
	})
	tone.StyleFinal(sf)
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
func (cv *ColorView) Config(sc *gi.Scene) {
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
			cv.Clipboard().Write(mimedata.NewText(text))
		})
		gi.NewButton(m).SetText("styles.ColorFromRGBA(r, g, b, a)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("styles.ColorFromRGBA(%d, %d, %d, %d)", cv.Color.R, cv.Color.G, cv.Color.B, cv.Color.A)
			cv.Clipboard().Write(mimedata.NewText(text))
		})
		gi.NewButton(m).SetText("rgb(r, g, b)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("rgb(%d, %d, %d)", cv.Color.R, cv.Color.G, cv.Color.B)
			cv.Clipboard().Write(mimedata.NewText(text))
		})
		gi.NewButton(m).SetText("rgba(r, g, b, a)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("rgba(%d, %d, %d, %d)", cv.Color.R, cv.Color.G, cv.Color.B, cv.Color.A)
			cv.Clipboard().Write(mimedata.NewText(text))
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
			cv.Clipboard().Write(mimedata.NewText(text))
		})
		gi.NewButton(m).SetText("styles.ColorFromHSLA(h, s, l, a)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("styles.ColorFromHSLA(%g, %g, %g, %g)", cv.ColorHSLA.H, cv.ColorHSLA.S, cv.ColorHSLA.L, cv.ColorHSLA.A)
			cv.Clipboard().Write(mimedata.NewText(text))
		})
		gi.NewButton(m).SetText("hsl(h, s, l)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("hsl(%g, %g, %g)", cv.ColorHSLA.H, cv.ColorHSLA.S, cv.ColorHSLA.L)
			cv.Clipboard().Write(mimedata.NewText(text))
		})
		gi.NewButton(m).SetText("hsla(h, s, l, a)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("hsla(%g, %g, %g, %g)", cv.ColorHSLA.H, cv.ColorHSLA.S, cv.ColorHSLA.L, cv.ColorHSLA.A)
			cv.Clipboard().Write(mimedata.NewText(text))
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
			cv.Clipboard().Write(mimedata.NewText(text))
		})
		gi.NewButton(m).SetText(`styles.ColorFromHex("#RRGGBBAA")`).OnClick(func(e events.Event) {
			text := fmt.Sprintf(`styles.ColorFromHex("%s")`, colors.AsHex(cv.Color))
			cv.Clipboard().Write(mimedata.NewText(text))
		})
		gi.NewButton(m).SetText("#RRGGBB").OnClick(func(e events.Event) {
			hs := colors.AsHex(cv.Color)
			text := hs[:len(hs)-2]
			cv.Clipboard().Write(mimedata.NewText(text))
		})
		gi.NewButton(m).SetText("#RRGGBBAA").OnClick(func(e events.Event) {
			text := colors.AsHex(cv.Color)
			cv.Clipboard().Write(mimedata.NewText(text))
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
// 		cv.RenderFrame(sc)
// 		cv.RenderChildren(sc)
// 		cv.PopBounds(sc)
// 	}
// }

*/

////////////////////////////////////////////////////////////////////////////////////////
//  ColorValue

// ColorValue represents a color value with a button.
type ColorValue struct {
	ValueBase[*gi.Button]
}

func (v *ColorValue) Config() {
	v.Widget.SetType(gi.ButtonTonal).SetText("Edit color").SetIcon(icons.Colors)
	ConfigDialogWidget(v, false)
	v.Widget.Style(func(s *styles.Style) {
		// we need to display button as non-transparent
		// so that it can be seen
		dclr := colors.WithAF32(v.ColorValue(), 1)
		s.Background = colors.C(dclr)
		s.Color = colors.C(hct.ContrastColor(dclr, hct.ContrastAAA))
	})
}

func (v *ColorValue) Update() {
	v.Widget.Update()
}

func (v *ColorValue) ConfigDialog(d *gi.Body) (bool, func()) {
	d.SetTitle("Edit color")
	cv := NewColorView(d).SetColor(v.ColorValue())
	return true, func() {
		if u, ok := laser.OnePtrUnderlyingValue(v.Value).Interface().(*image.Uniform); ok {
			u.C = cv.Color.AsRGBA()
		} else {
			v.SetValue(cv.Color.AsRGBA())
		}
		v.Update()
	}
}

// ColorValue returns a standardized color value from whatever value is represented
// internally, or nil.
func (v *ColorValue) ColorValue() color.RGBA {
	c := laser.NonPtrValue(v.Value).Interface().(color.Color)
	return colors.AsRGBA(c)
}
