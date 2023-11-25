// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"fmt"
	"image"
	"testing"

	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/grows/images"
	"goki.dev/mat32/v2"
)

func TestBoxModel(t *testing.T) {
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
	st.Color = colors.Black
	st.BackgroundColor.SetSolid(colors.Lightblue)
	st.Border.Style.Set(styles.BorderSolid)
	st.Border.Width.Set(units.Dp(5))
	st.Border.Color.Set(colors.Red)
	st.Border.Radius = styles.BorderRadiusFull

	st.ToDots()

	sbg := &colors.Full{Solid: colors.White}

	sz := st.BoxSpace().Size().Add(mat32.Vec2{200, 100})
	pc.DrawStdBox(rs, st, mat32.Vec2{50, 75}, sz, sbg, 0)

	rs.Unlock()

	images.Assert(t, img, "boxmodel")
}

func TestBoxShadow(t *testing.T) {
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
	st.Color = colors.Black
	st.BackgroundColor.SetSolid(colors.Lightblue)
	st.Border.Style.Set(styles.BorderSolid)
	st.Border.Width.Set(units.Dp(0))
	st.Border.Color.Set(colors.Red)
	st.Border.Radius = styles.BorderRadiusFull
	st.BoxShadow = styles.BoxShadow5()

	st.ToDots()

	sbg := &colors.Full{Solid: colors.White}

	spc := st.BoxSpace().Size()
	sz := spc.Add(mat32.Vec2{200, 100})
	fmt.Println("spc:", spc)

	pc.DrawStdBox(rs, st, mat32.Vec2{50, 75}, sz, sbg, 0)

	rs.Unlock()

	images.Assert(t, img, "boxshadow")
}
