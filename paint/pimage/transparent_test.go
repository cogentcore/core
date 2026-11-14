// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pimage

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/image/draw"
)

// TestDrawTransparent tests that drawing a transparent color over an
// image changes nothing, and that drawing other colors does.
func TestDrawTransparent(t *testing.T) {
	dest := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(dest, dest.Bounds(), image.NewUniform(color.RGBA{10, 20, 30, 255}), image.Point{}, draw.Src)
	NewDraw(dest.Bounds(), image.NewUniform(color.Transparent), nil, image.Point{}, draw.Over).Render(dest)
	assert.Equal(t, color.RGBA{10, 20, 30, 255}, dest.RGBAAt(1, 1))
	NewDraw(dest.Bounds(), image.NewUniform(color.Transparent), nil, image.Point{}, draw.Src).Render(dest)
	assert.Equal(t, color.RGBA{}, dest.RGBAAt(1, 1), "src")
	NewDraw(dest.Bounds(), image.NewUniform(color.RGBA{0, 0, 128, 128}), nil, image.Point{}, draw.Over).Render(dest)
	assert.Equal(t, color.RGBA{0, 0, 128, 128}, dest.RGBAAt(1, 1), "over")
}
