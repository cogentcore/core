// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// Meter is a widget that renders a current value on as a filled
// bar/semicircle relative to a minimum and maximum potential value.
// The [Meter.Type] determines the shape of the meter, and the
// [styles.Style.Direction] determines the direction in which the
// meter goes.
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
type MeterTypes int32 //enums:enum

const (
	// MeterLinear indicates to render a meter that goes in a straight,
	// linear direction, either horizontal or vertical.
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
		if s.Direction == styles.Row {
			if m.Type == MeterLinear {
				s.Min.Set(units.Em(20), units.Em(0.5))
			} else {
				s.Min.Set(units.Em(8), units.Em(4))
			}
		} else {
			s.Min.Set(units.Em(0.5), units.Em(20))
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

	if m.Type == MeterLinear {
		m.RenderStdBox(st)
		if m.ValueColor != nil {
			dim := m.Styles.Direction.Dim()
			prop := (m.Value - m.Min) / (m.Max - m.Min)
			size := m.Geom.Size.Actual.Content.MulDim(dim, prop)
			pc.FillStyle.Color = m.ValueColor
			m.RenderBoxImpl(m.Geom.Pos.Content, size, st.Border)
		}
		return
	}

	pc.FillStyle.Color = nil
	pc.StrokeStyle.Color = st.Background
	pc.StrokeStyle.Width = units.Dp(20)
	pc.StrokeStyle.Width.ToDots(&st.UnitContext)

	pos := m.Geom.Pos.Content.AddScalar(pc.StrokeWidth())
	size := m.Geom.Size.Actual.Content.SubScalar(pc.StrokeWidth())

	cx := (pos.X + size.X) / 2
	cy := pos.Y + size.Y
	pc.DrawEllipticalArc(cx, cy, cx-pos.X, size.Y, 0, 180)
	pc.FillStrokeClear()
}
