// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/paint"
)

func TestCanvas(t *testing.T) {
	b := NewBody()
	NewCanvas(b).SetDraw(func(pc *paint.Context) {
		pc.MoveTo(0.25, 0.5)
		pc.LineTo(0.5, 0.25)
		pc.StrokeStyle.Color = colors.C(colors.Blue)
		pc.Stroke()

		pc.FillBoxColor(mat32.V2(0.3, 0.4), mat32.V2(0.5, 0.8), colors.Scheme.Success.Container)

		pc.FillStyle.Color = colors.C(colors.Orange)
		pc.DrawCircle(0.4, 0.6, 0.5)
		pc.Fill()
	})
	b.AssertRender(t, "canvas/basic")
}
