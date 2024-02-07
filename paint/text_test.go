// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/styles"
)

func TestText(t *testing.T) {
	RunTest(t, "text", 300, 300, func(pc *Context) {
		pc.BlitBoxColor(mat32.Vec2{}, mat32.V2(300, 300), colors.White)
		tsty := &styles.Text{}
		tsty.Defaults()
		fsty := &styles.FontRender{}
		fsty.Defaults()
		fsty.Size.Dp(60)

		txt := &Text{}
		txt.SetHTML("This is <a>HTML</a> <b>formatted</b> <i>text</i>", fsty, tsty, &pc.UnContext, nil)

		tsz := txt.LayoutStdLR(tsty, fsty, &pc.UnContext, mat32.V2(100, 40))
		_ = tsz
		// if tsz.X != 100 || tsz.Y != 40 {
		// 	t.Errorf("unexpected text size: %v", tsz)
		// }

		txt.Render(pc, mat32.V2(85, 80))
	})
}
