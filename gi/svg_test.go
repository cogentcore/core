// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"testing"

	"cogentcore.org/core/mat32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/units"
)

func TestSVG(t *testing.T) {
	b := NewBody()
	sv := NewSVG(b)
	sv.SVG.Root.ViewBox.Size.SetScalar(10)
	svg.NewCircle(&sv.SVG.Root).SetPos(mat32.V2(5, 5)).SetRadius(5)
	b.AssertRender(t, "svg/basic-circle")
}

// For https://github.com/cogentcore/core/issues/729
func TestSVGZoom(t *testing.T) {
	b := NewBody()
	sv := NewSVG(b)
	sv.Style(func(s *styles.Style) {
		s.Min.Set(units.Dp(1024))
	})
	sv.SVG.Root.ViewBox.Size.SetScalar(1000)
	sv.SVG.Scale = 100
	svg.NewText(&sv.SVG.Root).SetText("Hello, world!").SetPos(mat32.V2(0, 10))
	b.AssertRender(t, "svg/zoom")
}
