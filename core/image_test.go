// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"path/filepath"
	"testing"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
	"github.com/stretchr/testify/assert"
)

var testImagePath = Filename(filepath.Join("..", "icon.png"))

func TestImage(t *testing.T) {
	b := NewBody()
	img := NewImage(b)
	assert.NoError(t, img.Open(testImagePath))
	b.AssertRender(t, "image/basic")
}

func TestImageCropped(t *testing.T) {
	b := NewBody()
	b.Style(func(s *styles.Style) {
		s.Max.Set(units.Dp(75))
		s.Overflow.Set(styles.OverflowAuto)
	})
	img := NewImage(b)
	assert.NoError(t, img.Open(testImagePath))
	b.AssertRender(t, "image/cropped")
}

func TestImageScrolled(t *testing.T) {
	b := NewBody()
	b.Style(func(s *styles.Style) {
		s.Max.Set(units.Dp(75))
		s.Overflow.Set(styles.OverflowAuto)
	})
	img := NewImage(b)
	assert.NoError(t, img.Open(testImagePath))
	b.AssertRender(t, "image/scrolled", func() {
		b.SystemEvents().Scroll(image.Pt(10, 10), math32.Vec2(2, 3))
	})
}
