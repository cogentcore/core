// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"

	"cogentcore.org/core/styles"
	"github.com/stretchr/testify/assert"
)

func TestMeter(t *testing.T) {
	b := NewBody()
	m := NewMeter(b)
	tt, _ := m.WidgetTooltip(image.Point{})
	assert.Equal(t, "(value: 0.5, minimum: 0, maximum: 1)", tt)
	m.SetTooltip("Rating")
	tt, _ = m.WidgetTooltip(image.Point{})
	assert.Equal(t, "Rating (value: 0.5, minimum: 0, maximum: 1)", tt)
	AssertRender(t, b, "meter/basic")
}

func TestMeterValue(t *testing.T) {
	b := NewBody()
	m := NewMeter(b).SetValue(0.7)
	tt, _ := m.WidgetTooltip(image.Point{})
	assert.Equal(t, "(value: 0.7, minimum: 0, maximum: 1)", tt)
	AssertRender(t, b, "meter/value")
}

func TestMeterBounds(t *testing.T) {
	b := NewBody()
	m := NewMeter(b).SetMin(5.7).SetMax(18).SetValue(10.2)
	tt, _ := m.WidgetTooltip(image.Point{})
	assert.Equal(t, "(value: 10.2, minimum: 5.7, maximum: 18)", tt)
	AssertRender(t, b, "meter/bounds")
}

func TestMeterColumn(t *testing.T) {
	b := NewBody()
	NewMeter(b).Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	AssertRender(t, b, "meter/column")
}

func TestMeterCircle(t *testing.T) {
	b := NewBody()
	NewMeter(b).SetType(MeterCircle)
	AssertRender(t, b, "meter/circle")
}

func TestMeterSemicircle(t *testing.T) {
	b := NewBody()
	NewMeter(b).SetType(MeterSemicircle)
	AssertRender(t, b, "meter/semicircle")
}

func TestMeterCircleText(t *testing.T) {
	b := NewBody()
	NewMeter(b).SetType(MeterCircle).SetText("50%")
	AssertRender(t, b, "meter/circle-text")
}

func TestMeterSemicircleText(t *testing.T) {
	b := NewBody()
	NewMeter(b).SetType(MeterSemicircle).SetText("50%")
	AssertRender(t, b, "meter/semicircle-text")
}
