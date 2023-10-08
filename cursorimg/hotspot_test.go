// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cursorimg

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"goki.dev/cursors"
	"goki.dev/grows/images"
)

func TestHotspot(t *testing.T) {
	const size = 64
	for _, c := range cursors.CursorValues() {
		if c == cursors.None {
			continue
		}
		cur, err := Get(c, size)
		if err != nil {
			t.Fatal(err)
		}
		red := color.NRGBA{R: 255, A: 200}
		hs := cur.ImageHotspot(size)
		draw.Draw(cur.Image.(draw.Image), image.Rect(hs.X-3, hs.Y-3, hs.X+3, hs.Y+3), image.NewUniform(red), hs.Sub(image.Pt(3, 3)), draw.Over)
		err = images.Save(cur.Image, "testdata/"+c.String()+".png")
		if err != nil {
			t.Fatal(err)
		}
	}
}
