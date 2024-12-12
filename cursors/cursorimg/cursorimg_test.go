// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cursorimg

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"github.com/stretchr/testify/require"
)

func TestCursors(t *testing.T) {
	const size = 64
	img := image.NewRGBA(image.Rect(0, 0, 1000, 800))

	do := func(yOffset int) {
		draw.Draw(img, image.Rect(0, yOffset, 1000, 400+yOffset), colors.Scheme.Surface, image.Point{}, draw.Src)
		for i, c := range cursors.CursorValues() {
			if c == cursors.None {
				continue
			}
			cur, err := Get(c, size)
			require.NoError(t, err)
			x := ((i - 1) % 10) * 100
			y := ((i - 1) / 10) * 100
			draw.Draw(img, cur.Image.Bounds().Add(image.Pt(x, y+yOffset)), cur.Image, image.Point{}, draw.Over)
		}
		imagex.Assert(t, img, "cursors")
	}

	do(0)
	colors.SetScheme(true)
	do(400)
	colors.SetScheme(false)
}

func TestHotspot(t *testing.T) {
	const size = 64
	for _, c := range cursors.CursorValues() {
		if c == cursors.None {
			continue
		}
		cur, err := Get(c, size)
		require.NoError(t, err)
		red := color.RGBA{R: 255, A: 255}
		hs := cur.Hotspot
		cur.Image.(draw.Image).Set(hs.X, hs.Y, red)
		imagex.Assert(t, cur.Image, "hotspot/"+c.String())
	}
}
