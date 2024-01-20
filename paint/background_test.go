// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/grows/images"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/styles"
)

func TestBackgroundColor(t *testing.T) {
	RunTest(t, "background_color", 300, 300, func(pc *Context) {
		pabg := colors.C(colors.White)
		st := styles.NewStyle()
		st.Background = colors.C(colors.Blue)
		st.ComputeActualBackground(pabg)
		st.ToDots()

		sz := st.BoxSpace().Size().Add(mat32.V2(200, 100))
		pc.DrawStdBox(st, mat32.V2(50, 100), sz, pabg)
	})
}

func TestBackgroundImage(t *testing.T) {
	img, _, err := images.Open("test.png")
	grr.Test(t, err)
	RunTest(t, "background_image", 750, 400, func(pc *Context) {
		pabg := colors.C(colors.White)
		st := styles.NewStyle()
		st.Background = img
		st.ComputeActualBackground(pabg)
		st.ToDots()

		sz := st.BoxSpace().Size().Add(mat32.V2(200, 100))

		test := func(of styles.ObjectFits, pos mat32.Vec2) {
			st.ObjectFit = of
			pc.DrawStdBox(st, pos, sz, pabg)
		}

		test(styles.FitFill, mat32.V2(0, 0))
		test(styles.FitContain, mat32.V2(0, 120))
		test(styles.FitCover, mat32.V2(250, 0))
		test(styles.FitNone, mat32.V2(250, 120))
		test(styles.FitScaleDown, mat32.V2(500, 0))
	})
}
