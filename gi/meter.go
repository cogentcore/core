// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"goki.dev/colors"
	"goki.dev/styles"
)

// Meter is a widget that renders a current value on as a filled bar
// relative to a minimum and maximum potential value.
// The [styles.Style.Direction] determines the direction in which the
// meter goes.
type Meter struct {
	WidgetBase

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

func (m *Meter) OnInit() {
	m.WidgetBase.OnInit()
	m.SetStyles()
}

func (m *Meter) SetStyles() {
	m.Max = 1
	m.Style(func(s *styles.Style) {
		m.ValueColor = colors.C(colors.Scheme.Primary.Base)
		s.Background = colors.C(colors.Scheme.Primary.Container)
		s.Border.Radius = styles.BorderRadiusFull
	})
	m.StyleFinal(func(s *styles.Style) {
		if s.Direction == styles.Row {
			s.Min.X.Em(20)
			s.Min.Y.Em(0.5)
		} else {
			s.Min.Y.Em(20)
			s.Min.X.Em(0.5)
		}
	})
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
	m.RenderStdBox(st)

	if m.ValueColor != nil {
		dim := m.Styles.Direction.Dim()
		prop := (m.Value - m.Min) / (m.Max - m.Min)
		size := m.Geom.Size.Actual.Content.MulDim(dim, prop)
		pc.FillStyle.Color = m.ValueColor
		m.RenderBoxImpl(m.Geom.Pos.Content, size, st.Border)
	}
}
