// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/base/tolassert"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/stretchr/testify/assert"
)

func TestSpinner(t *testing.T) {
	b := NewBody()
	sp := NewSpinner(b)
	tt, _ := sp.WidgetTooltip()
	assert.Equal(t, "", tt)
	b.AssertRender(t, "spinner/basic")
}

func TestSpinnerValue(t *testing.T) {
	b := NewBody()
	sp := NewSpinner(b).SetValue(12.7)
	tt, _ := sp.WidgetTooltip()
	assert.Equal(t, "", tt)
	b.AssertRender(t, "spinner/value")
}

func TestSpinnerBounds(t *testing.T) {
	b := NewBody()
	sp := NewSpinner(b).SetMin(-0.5).SetMax(2.7)
	tt, _ := sp.WidgetTooltip()
	assert.Equal(t, "(minimum: -0.5, maximum: 2.7)", tt)
	sp.SetTooltip("Rating")
	tt, _ = sp.WidgetTooltip()
	assert.Equal(t, "Rating (minimum: -0.5, maximum: 2.7)", tt)
	sp.SetValue(-2.1)
	assert.Equal(t, float32(-0.5), sp.Value)
	sp.SetValue(18)
	assert.Equal(t, float32(2.7), sp.Value)
	b.AssertRender(t, "spinner/bounds")
}

func TestSpinnerButtons(t *testing.T) {
	b := NewBody()
	sp := NewSpinner(b)
	b.AssertRender(t, "spinner/buttons", func() {
		sp.LeadingIconButton().Send(events.Click)
		assert.Equal(t, float32(-0.1), sp.Value)
		sp.TrailingIconButton().Send(events.Click)
		assert.Equal(t, float32(0), sp.Value)
		sp.TrailingIconButton().Send(events.Click)
		assert.Equal(t, float32(0.1), sp.Value)
	})
}

func TestSpinnerArrowKeys(t *testing.T) {
	b := NewBody()
	sp := NewSpinner(b)
	b.AssertRender(t, "spinner/arrow-keys", func() {
		sp.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeDownArrow, 0))
		assert.Equal(t, float32(-0.1), sp.Value)
		sp.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeUpArrow, 0))
		assert.Equal(t, float32(0), sp.Value)
		sp.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodePageDown, 0))
		assert.Equal(t, float32(-0.2), sp.Value)
		sp.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodePageUp, 0))
		assert.Equal(t, float32(0), sp.Value)
		sp.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodePageUp, 0))
		assert.Equal(t, float32(0.2), sp.Value)
	})
}

func TestSpinnerStep(t *testing.T) {
	b := NewBody()
	sp := NewSpinner(b).SetStep(0.3)
	b.AssertRender(t, "spinner/step", func() {
		sp.LeadingIconButton().Send(events.Click)
		assert.Equal(t, float32(-0.3), sp.Value)
		sp.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodePageUp, 0))
		assert.Equal(t, float32(0.3), sp.Value)
	})
}

func TestSpinnerEnforceStep(t *testing.T) {
	b := NewBody()
	sp := NewSpinner(b).SetStep(10).SetEnforceStep(true).SetValue(47)
	assert.Equal(t, float32(50), sp.Value)
	b.AssertRender(t, "spinner/enforce-step")
}

func TestSpinnerOutlined(t *testing.T) {
	b := NewBody()
	NewSpinner(b).SetType(TextFieldOutlined)
	b.AssertRender(t, "spinner/outlined")
}

func TestSpinnerFormat(t *testing.T) {
	b := NewBody()
	sp := NewSpinner(b).SetFormat("%X").SetStep(1).SetValue(44)
	assert.Equal(t, "2C", sp.Txt)
	b.AssertRender(t, "spinner/format")
}

func TestSpinnerChangeButton(t *testing.T) {
	b := NewBody()
	sp := NewSpinner(b)
	value := float32(-1)
	sp.OnChange(func(e events.Event) {
		value = sp.Value
	})
	b.AssertRender(t, "spinner/change-button", func() {
		sp.TrailingIconButton().Send(events.Click)
		tolassert.Equal(t, 0.1, value)
	})
}

func TestSpinnerChangeArrowKey(t *testing.T) {
	b := NewBody()
	sp := NewSpinner(b)
	value := float32(-1)
	sp.OnChange(func(e events.Event) {
		value = sp.Value
	})
	b.AssertRender(t, "spinner/change-arrow-key", func() {
		sp.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeDownArrow, 0))
		tolassert.Equal(t, -0.1, value)
	})
}

func TestSpinnerChangeNumberKey(t *testing.T) {
	b := NewBody()
	sp := NewSpinner(b)
	value := float32(-1)
	sp.OnChange(func(e events.Event) {
		value = sp.Value
	})
	b.AssertRender(t, "spinner/change-number-key", func() {
		sp.HandleEvent(events.NewKey(events.KeyChord, '5', 0, 0))
		tolassert.Equal(t, -1, value)
		sp.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeReturnEnter, 0))
		tolassert.Equal(t, 50, value)
	})
}
