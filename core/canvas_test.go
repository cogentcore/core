// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/paint"
)

func TestCanvas(t *testing.T) {
	b := NewBody()
	NewCanvas(b).SetDraw(func(pc *paint.Context) {
		pc.MoveTo(0.15, 0.3)
		pc.LineTo(0.3, 0.15)
		pc.StrokeStyle.Color = colors.C(colors.Blue)
		pc.Stroke()

		pc.FillBox(mat32.V2(0.7, 0.3), mat32.V2(0.2, 0.5), colors.C(colors.Scheme.Success.Container))

		pc.FillStyle.Color = colors.C(colors.Orange)
		pc.DrawCircle(0.4, 0.5, 0.15)
		pc.Fill()
	})
	b.AssertRender(t, "canvas/basic")
}
