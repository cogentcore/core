// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"image"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/styles"
)

func TestText(t *testing.T) {
	size := image.Point{100, 40}
	sizef := mat32.V2FromPoint(size)
	RunTest(t, "text", size.X, size.Y, func(pc *Context) {
		pc.BlitBoxColor(mat32.Vec2{}, sizef, colors.White)
		tsty := &styles.Text{}
		tsty.Defaults()
		fsty := &styles.FontRender{}
		fsty.Defaults()
		fsty.Size.Dp(60)

		txt := &Text{}
		txt.SetHTML("This is <a>HTML</a> <b>formatted</b> <i>text</i>", fsty, tsty, &pc.UnContext, nil)

		tsz := txt.Layout(tsty, fsty, &pc.UnContext, sizef)
		_ = tsz
		// if tsz.X != 100 || tsz.Y != 40 {
		// 	t.Errorf("unexpected text size: %v", tsz)
		// }
		txt.HasOverflow = true
		txt.Render(pc, mat32.Vec2{})
	})
}
