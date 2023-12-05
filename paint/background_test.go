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
	"goki.dev/grr"
	"goki.dev/mat32/v2"
)

func TestBackgroundColor(t *testing.T) {
	RunTest(t, "background_color", image.Pt(300, 300), func(rs *State, pc *Paint) {
		st := &styles.Style{}
		st.Defaults()
		st.BackgroundColor.SetSolid(colors.Blue)

		st.ToDots()

		sbg := &colors.Full{Solid: colors.White}

		sz := st.BoxSpace().Size().Add(mat32.Vec2{200, 100})
		pc.DrawStdBox(rs, st, mat32.Vec2{50, 100}, sz, sbg, 0)
	})
}

func TestBackgroundImage(t *testing.T) {
	RunTest(t, "background_image", image.Pt(750, 400), func(rs *State, pc *Paint) {
		st := &styles.Style{}
		st.Defaults()
		st.ToDots()

		sbg := &colors.Full{Solid: colors.White}
		sz := st.BoxSpace().Size().Add(mat32.Vec2{200, 100})

		test := func(of styles.ObjectFits, pos mat32.Vec2) {
			st.ObjectFit = of
			f, err := os.Open("test.png")
			if grr.Test(t, err) == nil {
				defer f.Close()
				st.BackgroundImage = f
			}
			pc.DrawStdBox(rs, st, pos, sz, sbg, 0)
		}

		test(styles.FitFill, mat32.Vec2{0, 0})
		test(styles.FitContain, mat32.Vec2{0, 120})
		test(styles.FitCover, mat32.Vec2{250, 0})
		test(styles.FitNone, mat32.Vec2{250, 120})
		test(styles.FitScaleDown, mat32.Vec2{500, 0})
	})
}
