// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlider(t *testing.T) {
	b := NewBody()
	sr := NewSlider(b)
	assert.Equal(t, "(value: 0.5, minimum: 0, maximum: 1)", sr.WidgetTooltip())
	sr.SetTooltip("Rating")
	assert.Equal(t, "Rating (value: 0.5, minimum: 0, maximum: 1)", sr.WidgetTooltip())
	b.AssertRender(t, "slider/basic")
}

func TestSliderValue(t *testing.T) {
	b := NewBody()
	sr := NewSlider(b).SetValue(0.7)
	assert.Equal(t, "(value: 0.7, minimum: 0, maximum: 1)", sr.WidgetTooltip())
	b.AssertRender(t, "slider/value")
}

func TestSliderBounds(t *testing.T) {
	b := NewBody()
	sr := NewSlider(b).SetMin(5.7).SetMax(18).SetValue(10.2)
	assert.Equal(t, "(value: 10.2, minimum: 5.7, maximum: 18)", sr.WidgetTooltip())
	b.AssertRender(t, "slider/bounds")
}
