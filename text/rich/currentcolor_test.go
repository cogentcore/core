// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rich

import (
	"image/color"
	"testing"

	"cogentcore.org/core/colors"
	"github.com/stretchr/testify/assert"
)

// TestCurrentColor tests that a color of currentColor is the color that
// the style inherits (as the links of github.com, whose text was
// transparent).
func TestCurrentColor(t *testing.T) {
	parent := NewStyle()
	parent.SetFillColor(color.RGBA{255, 0, 0, 255})
	s := NewStyle()
	s.FromProperty(parent, "color", "currentColor", colors.BaseContext(colors.Black))
	assert.Equal(t, color.Color(color.RGBA{255, 0, 0, 255}), s.FillColor())
	s = NewStyle()
	s.FromProperty(nil, "color", "currentcolor", colors.BaseContext(colors.Black))
	assert.Nil(t, s.FillColor(), "the default color")
}
