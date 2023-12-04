// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"fmt"
	"image"
	"image/draw"
	"testing"

	"goki.dev/grows/images"
)

func TestAlphaBlend(t *testing.T) {
	alphas := []float32{0.1, 0.5, 0.9}

	for _, a := range alphas {
		dst := Lightblue
		src := WithAF32(Darkblue, a)

		isz := image.Rectangle{Max: image.Point{800, 200}}
		i0 := image.Rect(0, 0, 200, 200)
		i1 := image.Rect(200, 0, 400, 200)
		i2 := image.Rect(400, 0, 600, 200)
		i3 := image.Rect(600, 0, 800, 200)
		img := image.NewRGBA(isz)

		b := AlphaBlend(dst, src)

		draw.Draw(img, i0, &image.Uniform{dst}, image.Point{}, draw.Src)
		draw.Draw(img, i1, &image.Uniform{src}, image.Point{}, draw.Src)
		draw.Draw(img, i2, &image.Uniform{b}, image.Point{}, draw.Src)
		draw.Draw(img, i3, &image.Uniform{dst}, image.Point{}, draw.Src)
		draw.Draw(img, i3, &image.Uniform{src}, image.Point{}, draw.Over)

		fnm := fmt.Sprintf("alpha_blend_%2d", int(a*100))
		images.Assert(t, img, fnm)
	}
}
