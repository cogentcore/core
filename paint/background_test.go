// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/iox/imagex"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"github.com/stretchr/testify/assert"
)

func TestBackgroundColor(t *testing.T) {
	RunTest(t, "background-color", 300, 300, func(pc *Context) {
		pabg := colors.C(colors.White)
		st := styles.NewStyle()
		st.Background = colors.C(colors.Blue)
		st.ComputeActualBackground(pabg)
		st.ToDots()

		sz := st.BoxSpace().Size().Add(math32.Vec2(200, 100))
		pc.DrawStandardBox(st, math32.Vec2(50, 100), sz, pabg)
	})
}

func TestBackgroundImage(t *testing.T) {
	img, _, err := imagex.Open("test.png")
	assert.NoError(t, err)
	RunTest(t, "background-image", 1260, 200, func(pc *Context) {
		pabg := colors.C(colors.White)
		st := styles.NewStyle()
		st.Background = img
		st.ComputeActualBackground(pabg)
		st.ToDots()

		sz := st.BoxSpace().Size().Add(math32.Vec2(200, 100))

		test := func(of styles.ObjectFits, pos math32.Vector2) {
			st.ObjectFit = of
			pc.DrawStandardBox(st, pos, sz, pabg)
		}

		test(styles.FitFill, math32.Vec2(0, 0))
		test(styles.FitContain, math32.Vec2(220, 0))
		test(styles.FitCover, math32.Vec2(440, 0))
		test(styles.FitScaleDown, math32.Vec2(660, 0))
		test(styles.FitNone, math32.Vec2(880, 0))
	})
}

func TestObjectFit(t *testing.T) {
	img, _, err := imagex.Open("test.png")
	// obj := math32.Vector2FromPoint(img.Bounds().Size())
	assert.NoError(t, err)
	RunTest(t, "object-fit", 1260, 300, func(pc *Context) {
		st := styles.NewStyle()
		st.ToDots()
		box := math32.Vec2(200, 100)

		test := func(of styles.ObjectFits, pos math32.Vector2) {
			st.ObjectFit = of
			fitimg := st.ResizeImage(img, box)
			pc.DrawImage(fitimg, pos.X, pos.Y)
			// trgsz := styles.ObjectSizeFromFit(of, obj, box)
			// fmt.Println(of, trgsz)
		}

		test(styles.FitFill, math32.Vec2(0, 0))
		test(styles.FitContain, math32.Vec2(220, 0))
		test(styles.FitCover, math32.Vec2(440, 0))
		test(styles.FitScaleDown, math32.Vec2(660, 0))
		test(styles.FitNone, math32.Vec2(880, 0))
	})
}
