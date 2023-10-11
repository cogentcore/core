// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"image"
	"image/png"
	"os"
	"testing"

	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/mat32/v2"
)

func TestBoxModel(t *testing.T) {
	FontLibrary.InitFontPaths(FontPaths...)

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
	st.BackgroundColor.SetSolid(colors.Gray)

	sbg := &colors.Full{Solid: colors.White}

	pc.DrawStdBox(rs, st, mat32.Vec2{50, 75}, mat32.Vec2{200, 100}, sbg)

	rs.Unlock()

	file, err := os.Create("boxmodel_test.png")
	if err != nil {
		t.Error(err)
	}
	defer file.Close()
	png.Encode(file, img)
}
