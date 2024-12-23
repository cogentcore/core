// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/base/tolassert"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"github.com/stretchr/testify/assert"
)

func TestSlider(t *testing.T) {
	b := NewBody()
	sr := NewSlider(b)
	tt, _ := sr.WidgetTooltip(image.Point{})
	assert.Equal(t, "(value: 0.5, minimum: 0, maximum: 1)", tt)
	sr.SetTooltip("Rating")
	tt, _ = sr.WidgetTooltip(image.Point{})
	assert.Equal(t, "Rating (value: 0.5, minimum: 0, maximum: 1)", tt)
	b.AssertRender(t, "slider/basic")
}

func TestSliderValue(t *testing.T) {
	b := NewBody()
	sr := NewSlider(b).SetValue(0.7)
	tt, _ := sr.WidgetTooltip(image.Point{})
	assert.Equal(t, "(value: 0.7, minimum: 0, maximum: 1)", tt)
	b.AssertRender(t, "slider/value")
}

func TestSliderBounds(t *testing.T) {
	b := NewBody()
	sr := NewSlider(b).SetMin(5.7).SetMax(18).SetValue(10.2)
	tt, _ := sr.WidgetTooltip(image.Point{})
	assert.Equal(t, "(value: 10.2, minimum: 5.7, maximum: 18)", tt)
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

func TestSliderEnforceStep(t *testing.T) {
	b := NewBody()
	sr := NewSlider(b).SetStep(0.2).SetEnforceStep(true)
	b.AssertRender(t, "slider/enforce-step", func() {
		sr.SetValue(0.7)
		tolassert.Equal(t, float32(0.8), sr.Value)
	})
}

func TestSliderIcon(t *testing.T) {
	b := NewBody()
	NewSlider(b).SetIcon(icons.DeployedCodeFill)
	b.AssertRender(t, "slider/icon")
}

func TestSliderStart(t *testing.T) {
	b := NewBody()
	sr := NewSlider(b)
	b.AssertRender(t, "slider/start", func() {
		sr.SystemEvents().MouseButton(events.MouseDown, events.Left, image.Pt(60, 20), 0)
	})
}

func TestSliderChange(t *testing.T) {
	b := NewBody()
	sr := NewSlider(b)
	n := 0
	value := float32(-1)
	sr.OnChange(func(e events.Event) {
		n++
		value = sr.Value
	})
	b.AssertRender(t, "slider/change", func() {
		sr.SystemEvents().MouseButton(events.MouseDown, events.Left, image.Pt(60, 20), 0)
		for x := 70; x < 200; x += 10 {
			sr.SystemEvents().MouseMove(image.Pt(x, 20))
			time.Sleep(5 * time.Millisecond)
		}
		sr.SystemEvents().MouseButton(events.MouseUp, events.Left, image.Pt(200, 20), 0)
	}, func() {
		assert.Equal(t, 1, n)
		tolassert.Equal(t, 0.5756579, value)
	})
}

func TestSliderChangeClick(t *testing.T) {
	b := NewBody()
	sr := NewSlider(b)
	n := 0
	value := float32(-1)
	sr.OnChange(func(e events.Event) {
		n++
		value = sr.Value
	})
	b.AssertRender(t, "slider/change-click", func() {
		sr.SystemEvents().MouseButton(events.MouseDown, events.Left, image.Pt(200, 20), 0)
		sr.SystemEvents().MouseButton(events.MouseUp, events.Left, image.Pt(200, 20), 0)
	}, func() {
		assert.Equal(t, 1, n)
		tolassert.Equal(t, 0.5756579, value)
	})
}

func TestSliderInput(t *testing.T) {
	b := NewBody()
	sr := NewSlider(b)
	n := 0
	value := float32(-1)
	sr.OnInput(func(e events.Event) {
		n++
		assert.Greater(t, sr.Value, value)
		value = sr.Value
	})
	b.AssertRender(t, "slider/input", func() {
		sr.SystemEvents().MouseButton(events.MouseDown, events.Left, image.Pt(60, 20), 0)
		for x := 70; x < 200; x += 10 {
			sr.SystemEvents().MouseMove(image.Pt(x, 20))
			time.Sleep(5 * time.Millisecond)
		}
		sr.SystemEvents().MouseButton(events.MouseUp, events.Left, image.Pt(200, 20), 0)
	}, func() {
		tolassert.EqualTol(t, 5, float32(n), 4)
		tolassert.Equal(t, 0.5756579, value)
	})
}
