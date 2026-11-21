// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterColors(t *testing.T) {
	grey := color.NRGBA{95, 99, 104, 255}
	white := ParseFilter("brightness(0) saturate(1) invert(1)").Color(grey)
	assert.Equal(t, color.NRGBA{255, 255, 255, 255}, white, "the wedges of pkg.go.dev")
	assert.Equal(t, color.NRGBA{160, 156, 151, 255}, ParseFilter("invert(100%)").Color(grey))
	half := ParseFilter("opacity(50%)").Color(grey)
	assert.Equal(t, uint8(128), half.A)
	g := ParseFilter("grayscale(1)").Color(color.NRGBA{255, 0, 0, 255})
	assert.Equal(t, g.R, g.G, "gray")
	assert.Nil(t, ParseFilter("none"))
	assert.Nil(t, ParseFilter("blur(2px)"), "not a color function")
}
