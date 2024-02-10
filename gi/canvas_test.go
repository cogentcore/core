// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/paint"
)

func TestCanvas(t *testing.T) {
	b := NewBody()
	c := NewCanvas(b)
	c.Draw(func(pc *paint.Context) {
		pc.MoveTo(20, 50)
		pc.LineTo(50, 20)
		pc.StrokeStyle.Color = colors.C(colors.Blue)
		pc.Stroke()
	})
	b.AssertRender(t, "canvas/basic")
}
