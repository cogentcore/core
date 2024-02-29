// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hct

import (
	"fmt"
	"image/color"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransform(t *testing.T) {
	assert.Equal(t, color.RGBA{131, 140, 255, 255}, Lighten(color.RGBA{0, 0, 255, 255}, 30))
	assert.Equal(t, color.RGBA{0, 0, 52, 255}, Darken(color.RGBA{0, 0, 255, 255}, 30))
	assert.Equal(t, color.RGBA{131, 140, 255, 255}, Highlight(color.RGBA{0, 0, 255, 255}, 30))
	assert.Equal(t, color.RGBA{0, 54, 92, 255}, Highlight(color.RGBA{18, 127, 205, 255}, 30))

	c := Blend(50, color.RGBA{255, 255, 255, 255}, color.RGBA{0, 0, 0, 255})
	assert.Equal(t, color.RGBA{119, 119, 119, 255}, c)
}

func TestMinHueDistance(t *testing.T) {
	t.Skip("informational confirmation")
	for i := 0; i < 50; i++ {
		a := rand.Intn(360)
		b := rand.Intn(360)
		d := MinHueDistance(float32(a), float32(b))
		fmt.Println(a, b, d)
	}
}
