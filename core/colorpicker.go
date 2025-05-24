// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image/color"

	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/cam/hct"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// ColorPicker represents a color value with an interactive color picker
// composed of history buttons, a hex input, three HCT sliders, and standard
// named color buttons.
type ColorPicker struct {
	Frame

	// Color is the current color.
	Color hct.HCT `set:"-"`
}

func (cp *ColorPicker) WidgetValue() any { return &cp.Color }

// SetColor sets the color of the color picker.
func (cp *ColorPicker) SetColor(c color.Color) *ColorPicker {
	cp.Color = hct.FromColor(c)
	return cp
}

var namedColors = []string{"red", "orange", "yellow", "green", "blue", "violet", "sienna"}

func (cp *ColorPicker) Init() {
	cp.Frame.Init()
	cp.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 0)
	})

	colorButton := func(w *Button, c color.Color) {
		w.Styler(func(s *styles.Style) {
			s.Background = colors.Uniform(c)
			s.Padding.Set(units.Dp(ConstantSpacing(16)))
		})
		w.OnClick(func(e events.Event) {
			cp.SetColor(c).UpdateChange()
		})
	}

	tree.AddChild(cp, func(w *Frame) {
		tree.AddChild(w, func(w *Button) {
			w.SetTooltip("Current color")
			colorButton(w, &cp.Color) // a pointer so it updates
		})
		tree.AddChild(w, func(w *Button) {
			w.SetTooltip("Previous color")
			colorButton(w, cp.Color) // not a pointer so it does not update
		})
		tree.AddChild(w, func(w *TextField) {
			w.SetTooltip("Hex color")
			w.Styler(func(s *styles.Style) {
				s.Min.X.Em(6)
				s.Max.X.Em(6)
			})
			w.Updater(func() {
				w.SetText(colors.AsHex(cp.Color))
			})
			w.SetValidator(func() error {
				c, err := colors.FromHex(w.Text())
				if err != nil {
					return err
				}
				cp.SetColor(c).UpdateChange()
				return nil
			})
		})
	})

	sf := func(s *styles.Style) {
		s.Min.Y.Em(2)
		s.Min.X.Em(6)
		s.Max.X.Em(40)
		s.Grow.Set(1, 0)
	}
	tree.AddChild(cp, func(w *Slider) {
		Bind(&cp.Color.Hue, w)
		w.SetMin(0).SetMax(360)
		w.SetTooltip("The hue, which is the spectral identity of the color (red, green, blue, etc) in degrees")
		w.OnInput(func(e events.Event) {
			cp.Color.SetHue(w.Value)
			cp.UpdateInput()
		})
		w.OnChange(func(e events.Event) {
			cp.Color.SetHue(w.Value)
			cp.UpdateChange()
		})
		w.Styler(func(s *styles.Style) {
			w.ValueColor = nil
			w.ThumbColor = colors.Uniform(cp.Color)
			g := gradient.NewLinear()
			for h := float32(0); h <= 360; h += 5 {
				gc := cp.Color.WithHue(h)
				g.AddStop(gc.AsRGBA(), h/360)
			}
			s.Background = g
		})
		w.FinalStyler(sf)
	})
	tree.AddChild(cp, func(w *Slider) {
		Bind(&cp.Color.Chroma, w)
		w.SetMin(0).SetMax(120)
		w.SetTooltip("The chroma, which is the colorfulness/saturation of the color")
		w.Updater(func() {
			w.SetMax(cp.Color.MaximumChroma())
		})
		w.OnInput(func(e events.Event) {
			cp.Color.SetChroma(w.Value)
			cp.UpdateInput()
		})
		w.OnChange(func(e events.Event) {
			cp.Color.SetChroma(w.Value)
			cp.UpdateChange()
		})
		w.Styler(func(s *styles.Style) {
			w.ValueColor = nil
			w.ThumbColor = colors.Uniform(cp.Color)
			g := gradient.NewLinear()
			for c := float32(0); c <= w.Max; c += 5 {
				gc := cp.Color.WithChroma(c)
				g.AddStop(gc.AsRGBA(), c/w.Max)
			}
			s.Background = g
		})
		w.FinalStyler(sf)
	})
	tree.AddChild(cp, func(w *Slider) {
		Bind(&cp.Color.Tone, w)
		w.SetMin(0).SetMax(100)
		w.SetTooltip("The tone, which is the lightness of the color")
		w.OnInput(func(e events.Event) {
			cp.Color.SetTone(w.Value)
			cp.UpdateInput()
		})
		w.OnChange(func(e events.Event) {
			cp.Color.SetTone(w.Value)
			cp.UpdateChange()
		})
		w.Styler(func(s *styles.Style) {
			w.ValueColor = nil
			w.ThumbColor = colors.Uniform(cp.Color)
			g := gradient.NewLinear()
			for c := float32(0); c <= 100; c += 5 {
				gc := cp.Color.WithTone(c)
				g.AddStop(gc.AsRGBA(), c/100)
			}
			s.Background = g
		})
		w.FinalStyler(sf)
	})
	tree.AddChild(cp, func(w *Slider) {
		Bind(&cp.Color.A, w)
		w.SetMin(0).SetMax(1)
		w.SetTooltip("The opacity of the color")
		w.OnInput(func(e events.Event) {
			cp.Color.SetColor(colors.WithAF32(cp.Color, w.Value))
			cp.UpdateInput()
		})
		w.OnChange(func(e events.Event) {
			cp.Color.SetColor(colors.WithAF32(cp.Color, w.Value))
			cp.UpdateChange()
		})
		w.Styler(func(s *styles.Style) {
			w.ValueColor = nil
			w.ThumbColor = colors.Uniform(cp.Color)
			g := gradient.NewLinear()
			for c := float32(0); c <= 1; c += 0.05 {
				gc := colors.WithAF32(cp.Color, c)
				g.AddStop(gc, c)
			}
			s.Background = g
		})
		w.FinalStyler(sf)
	})

	tree.AddChild(cp, func(w *Frame) {
		for _, name := range namedColors {
			c := colors.Map[name]
			tree.AddChildAt(w, name, func(w *Button) {
				w.SetTooltip(strcase.ToSentence(name))
				colorButton(w, c)
			})
		}
	})
}

// ColorButton represents a color value with a button that opens a [ColorPicker].
type ColorButton struct {
	Button
	Color color.RGBA
}

func (cb *ColorButton) WidgetValue() any { return &cb.Color }

func (cb *ColorButton) Init() {
	cb.Button.Init()
	cb.SetType(ButtonTonal).SetText("Edit color").SetIcon(icons.Colors)
	cb.Styler(func(s *styles.Style) {
		// we need to display button as non-transparent
		// so that it can be seen
		dclr := colors.WithAF32(cb.Color, 1)
		s.Background = colors.Uniform(dclr)
		s.Color = colors.Uniform(hct.ContrastColor(dclr, hct.ContrastAAA))
	})
	InitValueButton(cb, false, func(d *Body) {
		d.SetTitle("Edit color")
		cp := NewColorPicker(d).SetColor(cb.Color)
		cp.OnChange(func(e events.Event) {
			cb.Color = cp.Color.AsRGBA()
		})
	})
}
