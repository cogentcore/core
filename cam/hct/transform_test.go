// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hct

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransform(t *testing.T) {
	assert.Equal(t, color.RGBA{131, 140, 255, 255}, Lighten(color.RGBA{0, 0, 255, 255}, 30))
	assert.Equal(t, color.RGBA{0, 0, 52, 255}, Darken(color.RGBA{0, 0, 255, 255}, 30))
	assert.Equal(t, color.RGBA{80, 90, 255, 255}, Highlight(color.RGBA{0, 0, 255, 255}, 15))
	assert.Equal(t, color.RGBA{0, 82, 136, 255}, Highlight(color.RGBA{18, 127, 205, 255}, 18))

	assert.Equal(t, color.RGBA{201, 0, 143, 255}, Saturate(color.RGBA{201, 2, 143, 255}, 16))
	assert.Equal(t, color.RGBA{96, 76, 125, 255}, Desaturate(color.RGBA{112, 35, 206, 255}, 43))

	assert.Equal(t, color.RGBA{107, 66, 106, 255}, Spin(color.RGBA{30, 85, 116, 255}, 91))

	assert.Equal(t, float32(80), MinHueDistance(240, 320))
	assert.Equal(t, float32(-80), MinHueDistance(320, 240))
	assert.Equal(t, float32(46), MinHueDistance(320, 6))
	assert.Equal(t, float32(-46), MinHueDistance(6, 320))

	c := Blend(50, color.RGBA{255, 255, 255, 255}, color.RGBA{0, 0, 0, 255})
	assert.Equal(t, color.RGBA{119, 119, 119, 255}, c)

	assert.False(t, IsLight(color.RGBA{17, 38, 91, 255}))
	assert.True(t, IsLight(color.RGBA{178, 89, 203, 255}))
	assert.True(t, IsDark(color.RGBA{17, 38, 91, 255}))
	assert.False(t, IsDark(color.RGBA{178, 89, 203, 255}))
}
