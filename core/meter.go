// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

// Meter is a widget that renders a current value on as a filled
// bar/semicircle relative to a minimum and maximum potential value.
type Meter struct {
	WidgetBase

	// Type is the styling type of the meter.
	Type MeterTypes

	// Value is the current value of the meter.
	// It defaults to 0.5.
	Value float32

	// Min is the minimum possible value of the meter.
	// It defaults to 0.
	Min float32

	// Max is the maximum possible value of the meter.
	// It defaults to 1.
	Max float32

	// Text, for [MeterCircle] and [MeterSemicircle], is the
	// text to render inside of the circle/semicircle.
	Text string

	// ValueColor is the image color that will be used to
	// render the filled value bar. It should be set in Style.
	ValueColor image.Image

	// Width, for [MeterCircle] and [MeterSemicircle], is the
	// width of the circle/semicircle. It should be set in Style.
	Width units.Value
}

// MeterTypes are the different styling types of [Meter]s.
type MeterTypes int32 //enums:enum -trim-prefix Meter

const (
	// MeterLinear indicates to render a meter that goes in a straight,
	// linear direction, either horizontal or vertical, as specified by
	// [styles.Style.Direction].
	MeterLinear MeterTypes = iota

	// MeterCircle indicates to render the meter as a circle.
	MeterCircle

	// MeterSemicircle indicates to render the meter as a semicircle.
	MeterSemicircle
)

func (m *Meter) WidgetValue() any { return &m.Value }

func (m *Meter) OnInit() {
	m.WidgetBase.OnInit()
	m.Value = 0.5
	m.Max = 1
	m.Style(func(s *styles.Style) {
		m.ValueColor = colors.C(colors.Scheme.Primary.Base)
		s.Background = colors.C(colors.Scheme.SurfaceVariant)
		s.Border.Radius = styles.BorderRadiusFull
	})
	m.StyleFinal(func(s *styles.Style) {
		switch m.Type {
		case MeterLinear:
			if s.Direction == styles.Row {
				s.Min.Set(units.Dp(320), units.Dp(8))
			} else {
				s.Min.Set(units.Dp(8), units.Dp(320))
			}
		case MeterCircle:
			s.Min.Set(units.Dp(128))
			m.Width.Dp(8)
			s.Font.Size.Dp(32)
			s.Text.LineHeight.Dp(40)
			s.Text.Align = styles.Center
			s.Text.AlignV = styles.Center
		case MeterSemicircle:
			s.Min.Set(units.Dp(112), units.Dp(64))
			m.Width.Dp(16)
			s.Font.Size.Dp(22)
			s.Text.LineHeight.Dp(28)
			s.Text.Align = styles.Center
			s.Text.AlignV = styles.Center
		}
	})
}

func (m *Meter) ApplyStyle() {
	m.ApplyStyleWidget()
	m.Width.ToDots(&m.Styles.UnitContext)
}

func (m *Meter) WidgetTooltip(pos image.Point) (string, image.Point) {
	res := m.Tooltip
	if res != "" {
		res += " "
	}
	res += fmt.Sprintf("(value: %.4g, minimum: %g, maximum: %g)", m.Value, m.Min, m.Max)
	return res, m.DefaultTooltipPos()
}

func (m *Meter) Render() {
	pc := &m.Scene.PaintContext
	st := &m.Styles

	prop := (m.Value - m.Min) / (m.Max - m.Min)

	if m.Type == MeterLinear {
		m.RenderStandardBox()
		if m.ValueColor != nil {
			dim := m.Styles.Direction.Dim()
			size := m.Geom.Size.Actual.Content.MulDim(dim, prop)
			pc.FillStyle.Color = m.ValueColor
			m.RenderBoxImpl(m.Geom.Pos.Content, size, st.Border)
		}
		return
	}

	pc.StrokeStyle.Width = m.Width
	sw := pc.StrokeWidth()
	pos := m.Geom.Pos.Content.AddScalar(sw / 2)
	size := m.Geom.Size.Actual.Content.SubScalar(sw)

	var txt *paint.Text
	var toff math32.Vector2
	if m.Text != "" {
		txt = &paint.Text{}
		txt.SetHTML(m.Text, st.FontRender(), &st.Text, &st.UnitContext, nil)
		tsz := txt.Layout(&st.Text, st.FontRender(), &st.UnitContext, size)
		toff = tsz.DivScalar(2)
	}

	if m.Type == MeterCircle {
		r := size.DivScalar(2)
		c := pos.Add(r)

		pc.DrawEllipticalArc(c.X, c.Y, r.X, r.Y, 0, 2*math32.Pi)
		pc.StrokeStyle.Color = st.Background
		pc.Stroke()

		if m.ValueColor != nil {
			pc.DrawEllipticalArc(c.X, c.Y, r.X, r.Y, -math32.Pi/2, prop*2*math32.Pi-math32.Pi/2)
			pc.StrokeStyle.Color = m.ValueColor
			pc.Stroke()
		}
		if txt != nil {
			txt.Render(pc, c.Sub(toff))
		}
		return
	}

	r := size.Mul(math32.Vec2(0.5, 1))
	c := pos.Add(r)

	pc.DrawEllipticalArc(c.X, c.Y, r.X, r.Y, math32.Pi, 2*math32.Pi)
	pc.StrokeStyle.Color = st.Background
	pc.Stroke()

	if m.ValueColor != nil {
		pc.DrawEllipticalArc(c.X, c.Y, r.X, r.Y, math32.Pi, (1+prop)*math32.Pi)
		pc.StrokeStyle.Color = m.ValueColor
		pc.Stroke()
	}
	if txt != nil {
		txt.Render(pc, c.Sub(size.Mul(math32.Vec2(0, 0.3))).Sub(toff))
	}
}
