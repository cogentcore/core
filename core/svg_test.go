// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"path/filepath"
	"testing"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/svg"
)

var testSVGPath = Filename(filepath.Join("..", "icon.svg"))

func TestSVG(t *testing.T) {
	b := NewBody()
	errors.Log(NewSVG(b).Open(testSVGPath))
	b.AssertRender(t, "svg/basic")
}

func TestSVGSize(t *testing.T) {
	b := NewBody()
	svg := NewSVG(b)
	errors.Log(svg.Open(testSVGPath))
	svg.Style(func(s *styles.Style) {
		s.Min.Set(units.Dp(128))
	})
	b.AssertRender(t, "svg/size")
}

func TestSVGString(t *testing.T) {
	b := NewBody()
	errors.Log(NewSVG(b).ReadString(`<rect width="100" height="100" fill="red"/>`))
	b.AssertRender(t, "svg/string")
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
	svg.NewText(sv.SVG.Root).SetText("Hello, world!").SetPos(math32.Vec2(0, 10))
	b.AssertRender(t, "svg/zoom")
}
