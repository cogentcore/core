// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"image"
	"os"
	"testing"

	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/grows/images"
	"goki.dev/grr"
	"goki.dev/mat32/v2"
)

func TestBackgroundColor(t *testing.T) {
	imgsz := image.Point{320, 240}
	szrec := image.Rectangle{Max: imgsz}
	img := image.NewRGBA(szrec)

	rs := &State{}
	pc := &Paint{}

	pc.Defaults()               // zeros are not good defaults for paint
	pc.SetUnitContextExt(imgsz) // initialize units

	rs.Init(imgsz.X, imgsz.Y, img)
	rs.PushBounds(szrec)
	rs.Lock()

	st := &styles.Style{}
	st.Defaults()
	st.BackgroundColor.SetSolid(colors.Blue)

	st.ToDots()

	sbg := &colors.Full{Solid: colors.White}

	sz := st.BoxSpace().Size().Add(mat32.Vec2{200, 100})
	pc.DrawStdBox(rs, st, mat32.Vec2{50, 75}, sz, sbg, 0)

	rs.Unlock()

	images.Assert(t, img, "background_color")
}

func TestBackgroundImage(t *testing.T) {
	imgsz := image.Point{320, 240}
	szrec := image.Rectangle{Max: imgsz}
	img := image.NewRGBA(szrec)

	rs := &State{}
	pc := &Paint{}

	pc.Defaults()               // zeros are not good defaults for paint
	pc.SetUnitContextExt(imgsz) // initialize units

	rs.Init(imgsz.X, imgsz.Y, img)
	rs.PushBounds(szrec)
	rs.Lock()

	st := &styles.Style{}
	st.Defaults()
	f, err := os.Open("test.png")
	if grr.Test(t, err) == nil {
		defer f.Close()
		st.BackgroundImage = f
	}

	st.ToDots()

	sbg := &colors.Full{Solid: colors.White}

	sz := st.BoxSpace().Size().Add(mat32.Vec2{200, 100})
	pc.DrawStdBox(rs, st, mat32.Vec2{50, 75}, sz, sbg, 0)

	rs.Unlock()

	images.Assert(t, img, "background_image")
}
