// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/gox/tolassert"
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

func TestSliderArrowKeys(t *testing.T) {
	b := NewBody()
	sr := NewSlider(b)
	b.AssertRender(t, "slider/arrow-keys", func() {
		sr.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeLeftArrow, 0))
		assert.Equal(t, float32(0.4), sr.Value)
		sr.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeDownArrow, 0))
		assert.Equal(t, float32(0.5), sr.Value)
		sr.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeUpArrow, 0))
		assert.Equal(t, float32(0.4), sr.Value)
		sr.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeRightArrow, 0))
		assert.Equal(t, float32(0.5), sr.Value)
		sr.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodePageUp, 0))
		assert.Equal(t, float32(0.3), sr.Value)
		sr.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodePageDown, 0))
		assert.Equal(t, float32(0.5), sr.Value)
		sr.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodePageDown, 0))
		assert.Equal(t, float32(0.7), sr.Value)
	})
}

func TestSliderStep(t *testing.T) {
	b := NewBody()
	sr := NewSlider(b).SetStep(0.3)
	b.AssertRender(t, "slider/step", func() {
		sr.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeRightArrow, 0))
		tolassert.Equal(t, float32(0.8), sr.Value)
		sr.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodePageUp, 0))
		tolassert.Equal(t, float32(0.2), sr.Value)
	})
}
