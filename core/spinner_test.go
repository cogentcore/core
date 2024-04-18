// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpinner(t *testing.T) {
	b := NewBody()
	sp := NewSpinner(b)
	assert.Equal(t, "", sp.WidgetTooltip())
	b.AssertRender(t, "spinner/basic")
}

func TestSpinnerValue(t *testing.T) {
	b := NewBody()
	sp := NewSpinner(b).SetValue(12.7)
	assert.Equal(t, "", sp.WidgetTooltip())
	b.AssertRender(t, "spinner/value")
}

func TestSpinnerBounds(t *testing.T) {
	b := NewBody()
	sp := NewSpinner(b).SetMin(-0.5).SetMax(2.7)
	assert.Equal(t, "(minimum: -0.5, maximum: 2.7)", sp.WidgetTooltip())
	sp.SetTooltip("Rating")
	assert.Equal(t, "Rating (minimum: -0.5, maximum: 2.7)", sp.WidgetTooltip())
	b.AssertRender(t, "spinner/bounds")
}

func TestSpinnerEnforceStep(t *testing.T) {
	b := NewBody()
	NewSpinner(b).SetStep(10).SetEnforceStep(true).SetValue(43)
	b.AssertRender(t, "spinner/enforce-step")
}
