// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/cam/hct"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
)

// ColorPicker shows a color, using sliders or numbers to set values.
type ColorPicker struct {
	core.Frame

	// the color that we view
	Color hct.HCT `set:"-"`
}

func (cp *ColorPicker) WidgetValue() any { return &cp.Color }

// SetColor sets the source color.
func (cp *ColorPicker) SetColor(clr color.Color) *ColorPicker {
	return cp.SetHCT(hct.FromColor(clr))
}

// SetHCT sets the source color in terms of HCT
func (cp *ColorPicker) SetHCT(hct hct.HCT) *ColorPicker {
	cp.Color = hct
	cp.Update()
	cp.SendChange()
	return cp
}

func (cp *ColorPicker) Init() {
	cp.Frame.Init()
	cp.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	cp.Maker(func(p *core.Plan) {
		if cp.HasChildren() { // TODO(config)
			return
		}

		cp.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
		})

		sf := func(s *styles.Style) {
			s.Min.Y.Em(2)
			s.Min.X.Em(6)
			s.Max.X.Em(40)
			s.Grow.Set(1, 0)
		}

		hue := core.NewSlider(cp).SetMin(0).SetMax(360).SetValue(cp.Color.Hue)
		hue.SetTooltip("The hue, which is the spectral identity of the color (red, green, blue, etc) in degrees")
		hue.OnInput(func(e events.Event) {
			cp.Color.SetHue(hue.Value)
			cp.SetHCT(cp.Color)
		})
		hue.Styler(func(s *styles.Style) {
			hue.ValueColor = nil
			hue.ThumbColor = colors.C(cp.Color)
			g := gradient.NewLinear()
			for h := float32(0); h <= 360; h += 5 {
				gc := cp.Color.WithHue(h)
				g.AddStop(gc.AsRGBA(), h/360)
			}
			s.Background = g
		})
		hue.FinalStyler(sf)

		chroma := core.NewSlider(cp).SetMin(0).SetMax(150).SetValue(cp.Color.Chroma)
		chroma.SetTooltip("The chroma, which is the colorfulness/saturation of the color")
		chroma.OnInput(func(e events.Event) {
			cp.Color.SetChroma(chroma.Value)
			cp.SetHCT(cp.Color)
		})
		chroma.Styler(func(s *styles.Style) {
			chroma.ValueColor = nil
			chroma.ThumbColor = colors.C(cp.Color)
			g := gradient.NewLinear()
			for c := float32(0); c <= 150; c += 5 {
				gc := cp.Color.WithChroma(c)
				g.AddStop(gc.AsRGBA(), c/150)
			}
			s.Background = g
		})
		chroma.FinalStyler(sf)

		tone := core.NewSlider(cp).SetMin(0).SetMax(100).SetValue(cp.Color.Tone)
		tone.SetTooltip("The tone, which is the lightness of the color")
		tone.OnInput(func(e events.Event) {
			cp.Color.SetTone(tone.Value)
			cp.SetHCT(cp.Color)
		})
		tone.Styler(func(s *styles.Style) {
			tone.ValueColor = nil
			tone.ThumbColor = colors.C(cp.Color)
			g := gradient.NewLinear()
			for c := float32(0); c <= 100; c += 5 {
				gc := cp.Color.WithTone(c)
				g.AddStop(gc.AsRGBA(), c/100)
			}
			s.Background = g
		})
		tone.FinalStyler(sf)
	})
}

/*
func (cv *ColorPicker) Init() {
	cv.OnWidgetAdded(func(w core.Widget) {
		switch w.PathFrom(cv) {
		case "value":
			w.Styler(func(s *styles.Style) {
				s.Min.X.Em(6)
				s.Min.Y.Em(6)
				s.Border.Radius = styles.BorderRadiusFull
				s.BackgroundColor.SetSolid(cv.Color)
			})
		case "slider-grid":
			w.Styler(func(s *styles.Style) {
				s.Columns = 4
			})
		case "hexlbl":
			w.Styler(func(s *styles.Style) {
				s.Align.Y = styles.Center
			})
		case "palette":
			w.Styler(func(s *styles.Style) {
				s.Columns = 25
			})
		case "nums-hex":
			w.Styler(func(s *styles.Style) {
				s.Min.X.Ch(20)
			})
		}
		if sl, ok := w.(*core.Slider); ok {
			sl.Styler(func(s *styles.Style) {
				s.Min.X.Ch(20)
				s.Min.Y.Em(1)
				s.Margin.Set(units.Dp(6))
			})
		}
		if w.Parent.Name == "palette" {
			if cbt, ok := w.(*core.Button); ok {
				cbt.Styler(func(s *styles.Style) {
					c := colornames.Map[cbt.Name]

					s.BackgroundColor.SetSolid(c)
					s.Max.Set(units.Em(1.3))
					s.Margin.Zero()
				})
			}
		}
	})
}

// SetColor sets the source color
func (cv *ColorPicker) SetColor(clr color.Color) *ColorPicker {
	update := cv.UpdateStart()
	cv.Color = colors.AsRGBA(clr)
	cv.ColorHSLA = hsl.FromColor(clr)
	cv.ColorHSLA.Round()
	cv.UpdateEndRender(update)
	cv.SendChange()
	return cv
}

// Config configures a standard setup of entire view
func (cv *ColorPicker) Config(sc *core.Scene) {
	if cv.HasChildren() {
		return
	}
	update := cv.UpdateStart()
	vl := core.NewFrame(cv, "slider-lay")
	nl := core.NewFrame(cv, "num-lay")

	rgbalay := core.NewFrame(nl, "nums-rgba-lay")

	nrgba := NewStructViewInline(rgbalay, "nums-rgba")
	nrgba.SetStruct(&cv.Color)
	nrgba.OnChange(func(e events.Event) {
		cv.SetColor(cv.Color)
	})

	rgbacopy := core.NewButton(rgbalay, "rgbacopy")
	rgbacopy.Icon = icons.ContentCopy
	rgbacopy.Tooltip = "Copy RGBA Color"
	rgbacopy.Menu = func(m *core.Scene) {
		core.NewButton(m).SetText("styles.ColorFromRGB(r, g, b)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("styles.ColorFromRGB(%d, %d, %d)", cv.Color.R, cv.Color.G, cv.Color.B)
			cv.Clipboard().Write(mimedata.NewText(text))
		})
		core.NewButton(m).SetText("styles.ColorFromRGBA(r, g, b, a)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("styles.ColorFromRGBA(%d, %d, %d, %d)", cv.Color.R, cv.Color.G, cv.Color.B, cv.Color.A)
			cv.Clipboard().Write(mimedata.NewText(text))
		})
		core.NewButton(m).SetText("rgb(r, g, b)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("rgb(%d, %d, %d)", cv.Color.R, cv.Color.G, cv.Color.B)
			cv.Clipboard().Write(mimedata.NewText(text))
		})
		core.NewButton(m).SetText("rgba(r, g, b, a)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("rgba(%d, %d, %d, %d)", cv.Color.R, cv.Color.G, cv.Color.B, cv.Color.A)
			cv.Clipboard().Write(mimedata.NewText(text))
		})
	}

	hslalay := core.NewFrame(nl, "nums-hsla-lay")

	nhsla := NewStructViewInline(hslalay, "nums-hsla")
	nhsla.SetStruct(&cv.ColorHSLA)
	nhsla.OnChange(func(e events.Event) {
		cv.SetColor(cv.ColorHSLA)
	})

	hslacopy := core.NewButton(hslalay, "hslacopy")
	hslacopy.Icon = icons.ContentCopy
	hslacopy.Tooltip = "Copy HSLA Color"
	hslacopy.Menu = func(m *core.Scene) {
		core.NewButton(m).SetText("styles.ColorFromHSL(h, s, l)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("styles.ColorFromHSL(%g, %g, %g)", cv.ColorHSLA.H, cv.ColorHSLA.S, cv.ColorHSLA.L)
			cv.Clipboard().Write(mimedata.NewText(text))
		})
		core.NewButton(m).SetText("styles.ColorFromHSLA(h, s, l, a)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("styles.ColorFromHSLA(%g, %g, %g, %g)", cv.ColorHSLA.H, cv.ColorHSLA.S, cv.ColorHSLA.L, cv.ColorHSLA.A)
			cv.Clipboard().Write(mimedata.NewText(text))
		})
		core.NewButton(m).SetText("hsl(h, s, l)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("hsl(%g, %g, %g)", cv.ColorHSLA.H, cv.ColorHSLA.S, cv.ColorHSLA.L)
			cv.Clipboard().Write(mimedata.NewText(text))
		})
		core.NewButton(m).SetText("hsla(h, s, l, a)").OnClick(func(e events.Event) {
			text := fmt.Sprintf("hsla(%g, %g, %g, %g)", cv.ColorHSLA.H, cv.ColorHSLA.S, cv.ColorHSLA.L, cv.ColorHSLA.A)
			cv.Clipboard().Write(mimedata.NewText(text))
		})
	}

	hexlay := core.NewFrame(nl, "nums-hex-lay")

	core.NewText(hexlay, "hexlbl").SetText("Hex")

	hex := core.NewTextField(hexlay, "nums-hex")
	hex.Tooltip = "The color in hexadecimal form"
	hex.OnChange(func(e events.Event) {
		cv.SetColor(errors.Log(colors.FromHex(hex.Text())))
	})

	hexcopy := core.NewButton(hexlay, "hexcopy")
	hexcopy.Icon = icons.ContentCopy
	hexcopy.Tooltip = "Copy Hex Color"
	hexcopy.Menu = func(m *core.Scene) {
		core.NewButton(m).SetText(`styles.ColorFromHex("#RRGGBB")`).OnClick(func(e events.Event) {
			hs := colors.AsHex(cv.Color)
			// get rid of transparency because this is just RRGGBB
			text := fmt.Sprintf(`styles.ColorFromHex("%s")`, hs[:len(hs)-2])
			cv.Clipboard().Write(mimedata.NewText(text))
		})
		core.NewButton(m).SetText(`styles.ColorFromHex("#RRGGBBAA")`).OnClick(func(e events.Event) {
			text := fmt.Sprintf(`styles.ColorFromHex("%s")`, colors.AsHex(cv.Color))
			cv.Clipboard().Write(mimedata.NewText(text))
		})
		core.NewButton(m).SetText("#RRGGBB").OnClick(func(e events.Event) {
			hs := colors.AsHex(cv.Color)
			text := hs[:len(hs)-2]
			cv.Clipboard().Write(mimedata.NewText(text))
		})
		core.NewButton(m).SetText("#RRGGBBAA").OnClick(func(e events.Event) {
			text := colors.AsHex(cv.Color)
			cv.Clipboard().Write(mimedata.NewText(text))
		})
	}

	core.NewFrame(vl, "value")
	sg := core.NewFrame(vl, "slider-grid").SetDisplay(styles.Grid)

	core.NewText(sg, "rlab").SetText("Red:")
	rs := core.NewSlider(sg, "red")
	core.NewText(sg, "hlab").SetText("Hue:")
	hs := core.NewSlider(sg, "hue")
	core.NewText(sg, "glab").SetText("Green:")
	gs := core.NewSlider(sg, "green")
	core.NewText(sg, "slab").SetText("Sat:")
	ss := core.NewSlider(sg, "sat")
	core.NewText(sg, "blab").SetText("Blue:")
	bs := core.NewSlider(sg, "blue")
	core.NewText(sg, "llab").SetText("Light:")
	ls := core.NewSlider(sg, "light")
	core.NewText(sg, "alab").SetText("Alpha:")
	as := core.NewSlider(sg, "alpha")

	cv.ConfigRGBSlider(rs, 0)
	cv.ConfigRGBSlider(gs, 1)
	cv.ConfigRGBSlider(bs, 2)
	cv.ConfigRGBSlider(as, 3)
	cv.ConfigHSLSlider(hs, 0)
	cv.ConfigHSLSlider(ss, 1)
	cv.ConfigHSLSlider(ls, 2)

	cv.ConfigPalette()

	cv.UpdateEnd(update)
}

func (cv *ColorPicker) NumLay() *core.Frame {
	return cv.ChildByName("num-lay", 1).(*core.Frame)
}

func (cv *ColorPicker) SliderLay() *core.Frame {
	return cv.ChildByName("slider-lay", 0).(*core.Frame)
}

func (cv *ColorPicker) Value() *core.Frame {
	return cv.SliderLay().ChildByName("value", 0).(*core.Frame)
}

func (cv *ColorPicker) SliderGrid() *core.Frame {
	return cv.SliderLay().ChildByName("slider-grid", 0).(*core.Frame)
}

func (cv *ColorPicker) SetRGBValue(val float32, rgb int) {
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

func (cv *ColorPicker) ConfigRGBSlider(sl *core.Slider, rgb int) {
	sl.Max = 255
	sl.Step = 1
	sl.PageStep = 16
	sl.Prec = 3
	sl.Dim = math32.X
	sl.Tracking = true
	sl.TrackThr = 1
	cv.UpdateRGBSlider(sl, rgb)
	sl.OnChange(func(e events.Event) {
		cv.SetRGBValue(sl.Value, rgb)
		cv.SetColor(cv.Color)
	})
}

func (cv *ColorPicker) UpdateRGBSlider(sl *core.Slider, rgb int) {
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

func (cv *ColorPicker) SetHSLValue(val float32, hsln int) {
	switch hsln {
	case 0:
		cv.ColorHSLA.H = val
	case 1:
		cv.ColorHSLA.S = val / 360.0
	case 2:
		cv.ColorHSLA.L = val / 360.0
	}
}

func (cv *ColorPicker) ConfigHSLSlider(sl *core.Slider, hsl int) {
	sl.Max = 360
	sl.Step = 1
	sl.PageStep = 15
	sl.Prec = 3
	sl.Dim = math32.X
	sl.Tracking = true
	sl.TrackThr = 1
	sl.OnChange(func(e events.Event) {
		cv.SetHSLValue(sl.Value, hsl)
		cv.SetColor(cv.ColorHSLA)
	})
}

func (cv *ColorPicker) UpdateHSLSlider(sl *core.Slider, hsl int) {
	switch hsl {
	case 0:
		sl.SetValue(cv.ColorHSLA.H)
	case 1:
		sl.SetValue(cv.ColorHSLA.S * 360.0)
	case 2:
		sl.SetValue(cv.ColorHSLA.L * 360.0)
	}
}

func (cv *ColorPicker) UpdateSliderGrid() {
	sg := cv.SliderGrid()
	update := sg.UpdateStart()
	cv.UpdateRGBSlider(sg.ChildByName("red", 0).(*core.Slider), 0)
	cv.UpdateRGBSlider(sg.ChildByName("green", 0).(*core.Slider), 1)
	cv.UpdateRGBSlider(sg.ChildByName("blue", 0).(*core.Slider), 2)
	cv.UpdateRGBSlider(sg.ChildByName("alpha", 0).(*core.Slider), 3)
	cv.UpdateHSLSlider(sg.ChildByName("hue", 0).(*core.Slider), 0)
	cv.UpdateHSLSlider(sg.ChildByName("sat", 0).(*core.Slider), 1)
	cv.UpdateHSLSlider(sg.ChildByName("light", 0).(*core.Slider), 2)
	sg.UpdateEndRender(update)
}

func (cv *ColorPicker) ConfigPalette() {
	pg := core.NewFrame(cv, "palette").SetDisplay(styles.Grid)

	// STYTOOD: use hct sorted names here (see https://github.com/cogentcore/core/issues/619)
	nms := colors.Names

	for _, cn := range nms {
		cbt := core.NewButton(pg, cn)
		cbt.Tooltip = cn
		cbt.SetText("  ")
		cbt.OnChange(func(e events.Event) {
			cv.SetColor(errors.Log(colors.FromName(cbt.Name)))
		})
	}
}

func (cv *ColorPicker) Update() {
	update := cv.UpdateStart()
	cv.UpdateImpl()
	cv.UpdateEndRender(update)
}

// UpdateImpl does the raw updates based on current value,
// without UpdateStart / End wrapper
func (cv *ColorPicker) UpdateImpl() {
	cv.UpdateSliderGrid()
	cv.UpdateNums()
	cv.UpdateValueFrame()
	// cv.NumView.UpdateWidget()
	// v := cv.Value()
	// v.Style.BackgroundColor.Solid = cv.Color // direct copy
}

// UpdateValueFrame updates the value frame of the color picker
// that displays the color.
func (cv *ColorPicker) UpdateValueFrame() {
	cv.SliderLay().ChildByName("value", 0).(*core.Frame).Styles.BackgroundColor.Solid = cv.Color // direct copy
}

// UpdateNums updates the values of the number inputs
// in the color picker to reflect the latest values
func (cv *ColorPicker) UpdateNums() {
	cv.NumLay().ChildByName("nums-rgba-lay", 0).ChildByName("nums-rgba", 0).(*StructViewInline).UpdateFields()
	cv.NumLay().ChildByName("nums-hsla-lay", 1).ChildByName("nums-hsla", 0).(*StructViewInline).UpdateFields()
	hs := colors.AsHex(cv.Color)
	// if we are fully opaque, which is typical,
	// then we can skip displaying transparency in hex
	if cv.Color.A == 255 {
		hs = hs[:len(hs)-2]
	}
	cv.NumLay().ChildByName("nums-hex-lay", 2).ChildByName("nums-hex", 1).(*core.TextField).SetText(hs)
}

// func (cv *ColorPicker) Render(sc *core.Scene) {
// 	if cv.PushBounds(sc) {
// 		cv.RenderFrame(sc)
// 		cv.RenderChildren(sc)
// 		cv.PopBounds(sc)
// 	}
// }

*/

// ColorButton represents a color value with a button.
type ColorButton struct {
	core.Button
	Color color.RGBA
}

func (cb *ColorButton) WidgetValue() any { return &cb.Color }

func (cb *ColorButton) Init() {
	cb.Button.Init()
	cb.SetType(core.ButtonTonal).SetText("Edit color").SetIcon(icons.Colors)
	cb.Styler(func(s *styles.Style) {
		// we need to display button as non-transparent
		// so that it can be seen
		dclr := colors.WithAF32(cb.Color, 1)
		s.Background = colors.C(dclr)
		s.Color = colors.C(hct.ContrastColor(dclr, hct.ContrastAAA))
	})
	core.InitValueButton(cb, false, func(d *core.Body) {
		d.SetTitle("Edit color")
		cp := NewColorPicker(d).SetColor(cb.Color)
		cp.OnChange(func(e events.Event) {
			cb.Color = cp.Color.AsRGBA()
		})
	})
}
