// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"path/filepath"
	"testing"

	"cogentcore.org/core/grr"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

var testImagePath = Filename(filepath.Join("..", ".core", "icon.png"))

func TestImageBasic(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	img := NewImage(fr)
	grr.Test(t, img.OpenImage(testImagePath))
	b.AssertRender(t, "image/basic")
}

func TestImageCropped(t *testing.T) {
	b := NewBody()
	b.Style(func(s *styles.Style) {
		s.Max.Set(units.Dp(75))
	})
	fr := NewFrame(b).Style(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowAuto)
	})
	img := NewImage(fr)
	grr.Test(t, img.OpenImage(testImagePath))
	b.AssertRender(t, "image/cropped")
}

func TestImageScrolled(t *testing.T) {
	b := NewBody()
	b.Style(func(s *styles.Style) {
		s.Max.Set(units.Dp(75))
	})
	fr := NewFrame(b).Style(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowAuto)
	})
	img := NewImage(fr)
	grr.Test(t, img.OpenImage(testImagePath))
	b.AssertRender(t, "image/scrolled", func() {
		b.GoosiEventMgr().Scroll(image.Pt(10, 10), mat32.V2(2, 3))
	})
}
