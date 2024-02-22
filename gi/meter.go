// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// Meter is a widget that renders a current value on as a filled
// bar/semicircle relative to a minimum and maximum potential value.
type Meter struct {
	WidgetBase

	// Type is the styling type of the meter
	Type MeterTypes

	// Value is the current value of the meter
	Value float32

	// Min is the minimum possible value of the meter
	Min float32

	// Max is the maximum possible value of the meter
	Max float32

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

func (m *Meter) OnInit() {
	m.WidgetBase.OnInit()
	m.SetStyles()
}

func (m *Meter) SetStyles() {
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
				s.Min.Set(units.Em(20), units.Em(0.5))
			} else {
				s.Min.Set(units.Em(0.5), units.Em(20))
			}
		case MeterCircle:
			s.Min.Set(units.Em(8))
			m.Width.Em(0.5)
		case MeterSemicircle:
			s.Min.Set(units.Em(8), units.Em(4))
			m.Width.Em(1)
		}
	})
}

func (m *Meter) ApplyStyle() {
	m.ApplyStyleWidget()
	m.Width.ToDots(&m.Styles.UnitContext)
}

func (m *Meter) WidgetTooltip() string {
	res := m.Tooltip
	if res != "" {
		res += " "
	}
	res += fmt.Sprintf("(value: %9.4g, minimum: %g, maximum: %g)", m.Value, m.Min, m.Max)
	return res
}

func (m *Meter) Render() {
	if m.PushBounds() {
		m.RenderMeter()
		m.PopBounds()
	}
}

func (m *Meter) RenderMeter() {
	pc, st := m.RenderLock()
	defer m.RenderUnlock()

	prop := (m.Value - m.Min) / (m.Max - m.Min)

	if m.Type == MeterLinear {
		m.RenderStdBox(st)
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

	if m.Type == MeterCircle {
		r := size.DivScalar(2)
		c := pos.Add(r)

		pc.DrawEllipticalArc(c.X, c.Y, r.X, r.Y, 0, 2*mat32.Pi)
		pc.StrokeStyle.Color = st.Background
		pc.Stroke()

		if m.ValueColor != nil {
			pc.DrawEllipticalArc(c.X, c.Y, r.X, r.Y, -mat32.Pi/2, prop*2*mat32.Pi-mat32.Pi/2)
			pc.StrokeStyle.Color = m.ValueColor
			pc.Stroke()
		}
		return
	}

	r := size.Mul(mat32.V2(0.5, 1))
	c := pos.Add(r)

	pc.DrawEllipticalArc(c.X, c.Y, r.X, r.Y, mat32.Pi, 2*mat32.Pi)
	pc.StrokeStyle.Color = st.Background
	pc.Stroke()

	if m.ValueColor != nil {
		pc.DrawEllipticalArc(c.X, c.Y, r.X, r.Y, mat32.Pi, (1+prop)*mat32.Pi)
		pc.StrokeStyle.Color = m.ValueColor
		pc.Stroke()
	}
}
