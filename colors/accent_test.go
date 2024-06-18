// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image/color"
	"testing"

	"cogentcore.org/core/colors/matcolor"
	"github.com/stretchr/testify/assert"
)

func TestToBase(t *testing.T) {
	c := color.RGBA{255, 0, 0, 255}
	assert.Equal(t, color.RGBA{192, 3, 0, 255}, ToBase(c))

	c = color.RGBA{0, 255, 0, 255}
	assert.Equal(t, color.RGBA{3, 110, 0, 255}, ToBase(c))

	matcolor.SchemeIsDark = true

	c = color.RGBA{255, 0, 0, 255}
	assert.Equal(t, color.RGBA{255, 180, 168, 255}, ToBase(c))

	c = color.RGBA{0, 255, 0, 255}
	assert.Equal(t, color.RGBA{15, 230, 0, 255}, ToBase(c))

	matcolor.SchemeIsDark = false
}

func TestToOn(t *testing.T) {
	c := color.RGBA{255, 0, 0, 255}
	assert.Equal(t, color.RGBA{255, 255, 255, 255}, ToOn(c))

	c = color.RGBA{0, 255, 0, 255}
	assert.Equal(t, color.RGBA{255, 255, 255, 255}, ToOn(c))

	matcolor.SchemeIsDark = true

	c = color.RGBA{255, 0, 0, 255}
	assert.Equal(t, color.RGBA{104, 1, 0, 255}, ToOn(c))

	c = color.RGBA{0, 255, 0, 255}
	assert.Equal(t, color.RGBA{1, 58, 0, 255}, ToOn(c))

	matcolor.SchemeIsDark = false
}

func TestToContainer(t *testing.T) {
	c := color.RGBA{255, 0, 0, 255}
	assert.Equal(t, color.RGBA{255, 218, 212, 255}, ToContainer(c))

	c = color.RGBA{0, 255, 0, 255}
	assert.Equal(t, color.RGBA{119, 255, 96, 255}, ToContainer(c))

	matcolor.SchemeIsDark = true

	c = color.RGBA{255, 0, 0, 255}
	assert.Equal(t, color.RGBA{147, 2, 0, 255}, ToContainer(c))

	c = color.RGBA{0, 255, 0, 255}
	assert.Equal(t, color.RGBA{2, 83, 0, 255}, ToContainer(c))

	matcolor.SchemeIsDark = false
}

func TestToOnContainer(t *testing.T) {
	c := color.RGBA{255, 0, 0, 255}
	assert.Equal(t, color.RGBA{65, 0, 0, 255}, ToOnContainer(c))

	c = color.RGBA{0, 255, 0, 255}
	assert.Equal(t, color.RGBA{0, 34, 0, 255}, ToOnContainer(c))

	matcolor.SchemeIsDark = true

	c = color.RGBA{255, 0, 0, 255}
	assert.Equal(t, color.RGBA{255, 218, 212, 255}, ToOnContainer(c))

	c = color.RGBA{0, 255, 0, 255}
	assert.Equal(t, color.RGBA{119, 255, 96, 255}, ToOnContainer(c))

	matcolor.SchemeIsDark = false
}
