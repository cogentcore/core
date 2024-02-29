// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hsl

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransform(t *testing.T) {
	assert.Equal(t, color.RGBA{153, 153, 255, 255}, Lighten(color.RGBA{0, 0, 255, 255}, 30))
	assert.Equal(t, color.RGBA{0, 0, 102, 255}, Darken(color.RGBA{0, 0, 255, 255}, 30))
	assert.Equal(t, color.RGBA{0, 0, 179, 255}, Highlight(color.RGBA{0, 0, 255, 255}, 15))
	assert.Equal(t, color.RGBA{76, 171, 239, 255}, Highlight(color.RGBA{18, 127, 205, 255}, 18))
	assert.Equal(t, color.RGBA{13, 21, 79, 255}, Samelight(color.RGBA{29, 46, 171, 255}, 21))
	assert.Equal(t, color.RGBA{119, 8, 211, 255}, Samelight(color.RGBA{130, 9, 231, 255}, 4))

	assert.Equal(t, color.RGBA{203, 0, 144, 255}, Saturate(color.RGBA{201, 2, 143, 255}, 16))
	assert.Equal(t, color.RGBA{117, 87, 154, 255}, Desaturate(color.RGBA{112, 35, 206, 255}, 43))

	assert.Equal(t, color.RGBA{105, 30, 116, 255}, Spin(color.RGBA{30, 85, 116, 255}, 91))

	assert.False(t, IsLight(color.RGBA{17, 38, 91, 255}))
	assert.True(t, IsLight(color.RGBA{178, 129, 203, 255}))
	assert.True(t, IsDark(color.RGBA{17, 38, 91, 255}))
	assert.False(t, IsDark(color.RGBA{178, 129, 203, 255}))

	assert.Equal(t, color.RGBA{255, 255, 255, 255}, ContrastColor(color.RGBA{87, 32, 65, 255}))
	assert.Equal(t, color.RGBA{0, 0, 0, 255}, ContrastColor(color.RGBA{232, 146, 133, 255}))
}
