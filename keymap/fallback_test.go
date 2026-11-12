// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keymap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestOfFallback tests that Shift with the right and left arrows is bound
// to NextTab and PrevTab in all of the standard key maps, which only
// [OfFallback] returns, while [Of] returns the navigation (selecting), as
// for text fields.
func TestOfFallback(t *testing.T) {
	prev := ActiveMapName
	defer SetActiveMapName(prev)
	for _, km := range StandardMaps {
		SetActiveMapName(MapName(km.Name))
		assert.Equal(t, NextTab, OfFallback("Shift+RightArrow"), km.Name)
		assert.Equal(t, PrevTab, OfFallback("Shift+LeftArrow"), km.Name)
		assert.Equal(t, MoveRight, Of("Shift+RightArrow"), km.Name)
		assert.Equal(t, MoveLeft, Of("Shift+LeftArrow"), km.Name)
		assert.Equal(t, MoveRight, OfFallback("RightArrow"), km.Name)
	}
}
