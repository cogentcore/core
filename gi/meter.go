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
	m.Style(func(s *styles.Style) {
		m.ValueColor = colors.C(colors.Scheme.Primary.Base)
		s.Background = colors.C(colors.Scheme.SurfaceVariant)
	})
}
