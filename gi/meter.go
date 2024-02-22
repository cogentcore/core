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
}

// MeterTypes are the different styling types of [Meter]s.
type MeterTypes int32 //enums:enum -trim-prefix Meter

const (
	// MeterLinear indicates to render a meter that goes in a straight,
	// linear direction, either horizontal or vertical, as specified by
	// [styles.Style.Direction].
	MeterLinear MeterTypes = iota

	// MeterConic indicates to render a meter shaped in a conic/circular
	// way, like a semicircle.
	MeterConic
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
		if m.Type == MeterLinear {
			if s.Direction == styles.Row {
				s.Min.Set(units.Em(20), units.Em(0.5))
			} else {
				s.Min.Set(units.Em(0.5), units.Em(20))
			}
		} else {
			s.Min.Set(units.Em(9), units.Em(4))
		}
	})
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

	pc.StrokeStyle.Width = units.Dp(20)
	pc.StrokeStyle.Width.ToDots(&st.UnitContext)

	sw := pc.StrokeWidth() / 2
	pos := m.Geom.Pos.Content.AddScalar(sw)
	size := m.Geom.Size.Actual.Content.SubScalar(sw)

	cx := (pos.X + size.X) / 2
	cy := pos.Y + size.Y
	rx := cx - pos.X

	pc.DrawEllipticalArc(cx, cy, rx, size.Y, mat32.Pi, 2*mat32.Pi)
	pc.StrokeStyle.Color = st.Background
	pc.Stroke()

	pc.DrawEllipticalArc(cx, cy, rx, size.Y, mat32.Pi, (1+prop)*mat32.Pi)
	pc.StrokeStyle.Color = m.ValueColor
	pc.Stroke()
}
