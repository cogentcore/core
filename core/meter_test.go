// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMeter(t *testing.T) {
	b := NewBody()
	m := NewMeter(b)
	assert.Equal(t, "(value: 0.5, minimum: 0, maximum: 1)", m.WidgetTooltip())
	m.SetTooltip("Rating")
	assert.Equal(t, "Rating (value: 0.5, minimum: 0, maximum: 1)", m.WidgetTooltip())
	b.AssertRender(t, "meter/basic")
}

func TestMeterValue(t *testing.T) {
	b := NewBody()
	m := NewMeter(b).SetValue(0.7)
	assert.Equal(t, "(value: 0.7, minimum: 0, maximum: 1)", m.WidgetTooltip())
	b.AssertRender(t, "meter/value")
}

func TestMeterBounds(t *testing.T) {
	b := NewBody()
	m := NewMeter(b).SetMin(5.7).SetMax(18).SetValue(10.2)
	assert.Equal(t, "(value: 10.2, minimum: 5.7, maximum: 18)", m.WidgetTooltip())
	b.AssertRender(t, "meter/bounds")
}
