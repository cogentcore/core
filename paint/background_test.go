// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"testing"

	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/grows/images"
	"goki.dev/grr"
	"goki.dev/mat32/v2"
)

func TestBackgroundColor(t *testing.T) {
	RunTest(t, "background_color", 300, 300, func(pc *Context) {
		st := styles.NewStyle()
		st.Background = colors.C(colors.Blue)
		st.ToDots()

		sz := st.BoxSpace().Size().Add(mat32.Vec2{200, 100})
		pc.DrawStdBox(st, mat32.Vec2{50, 100}, sz, colors.C(colors.White))
	})
}

func TestBackgroundImage(t *testing.T) {
	img, _, err := images.Open("test.png")
	grr.Test(t, err)
	RunTest(t, "background_image", 750, 400, func(pc *Context) {
		st := styles.NewStyle()
		st.Background = img
		st.ToDots()

		sz := st.BoxSpace().Size().Add(mat32.Vec2{200, 100})

		test := func(of styles.ObjectFits, pos mat32.Vec2) {
			st.ObjectFit = of
			pc.DrawStdBox(st, pos, sz, colors.C(colors.White))
		}

		test(styles.FitFill, mat32.Vec2{0, 0})
		test(styles.FitContain, mat32.Vec2{0, 120})
		test(styles.FitCover, mat32.Vec2{250, 0})
		test(styles.FitNone, mat32.Vec2{250, 120})
		test(styles.FitScaleDown, mat32.Vec2{500, 0})
	})
}
