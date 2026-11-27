// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"image/color"
	"path/filepath"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/draw"
)

var testImagePath = Filename(filepath.Join("..", "docs", "image.png"))

func TestImage(t *testing.T) {
	b := NewBody()
	img := NewImage(b)
	assert.NoError(t, img.Open(testImagePath))
	AssertRender(t, b, "image/basic")
}

func TestImageSize(t *testing.T) {
	b := NewBody()
	img := NewImage(b)
	assert.NoError(t, img.Open(testImagePath))
	img.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(256))
	})
	AssertRender(t, b, "image/size")
}

func TestImageDirect(t *testing.T) {
	b := NewBody()
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	draw.Draw(img, image.Rect(10, 5, 100, 90), colors.Scheme.Warn.Container, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(20, 20, 60, 50), colors.Scheme.Success.Base, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(60, 70, 80, 100), colors.Scheme.Error.Base, image.Point{}, draw.Src)
	NewImage(b).SetImage(img)
	AssertRender(t, b, "image/direct")
}

func TestImageCropped(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Max.Set(units.Dp(75))
		s.Overflow.Set(styles.OverflowAuto)
	})
	img := NewImage(b)
	assert.NoError(t, img.Open(testImagePath))
	AssertRender(t, b, "image/cropped")
}

func TestImageScrolled(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Max.Set(units.Dp(75))
		s.Overflow.Set(styles.OverflowAuto)
	})
	img := NewImage(b)
	assert.NoError(t, img.Open(testImagePath))
	AssertRender(t, b, "image/scrolled", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		b.SystemEvents().Scroll(image.Pt(10, 10), math32.Vec2(2, 3))
	})
}

// TestImageAspectRatio tests that the box of an image has the aspect ratio
// of its style, if set, instead of that of the image, which fits in it.
func TestImageAspectRatio(t *testing.T) {
	b := NewBody()
	img := NewImage(b)
	img.SetImage(image.NewRGBA(image.Rect(0, 0, 100, 100)))
	img.Styler(func(s *styles.Style) {
		s.Min.X = units.Dot(200)
		s.AspectRatio = 2
	})
	RenderImage(b, func() {}, Locked(b, func() {
		assert.Equal(t, math32.Vec2(200, 100), img.Geom.Size.Actual.Content)
	}))
}

// TestImageContainCentered tests that an image that ObjectFit contain
// makes smaller than its box is centered in it, as in CSS.
func TestImageContainCentered(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Padding.Zero()
	})
	src := image.NewRGBA(image.Rect(0, 0, 100, 50))
	draw.Draw(src, src.Bounds(), image.NewUniform(color.RGBA{255, 0, 0, 255}), image.Point{}, draw.Src)
	fr := NewFrame(b)
	fr.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dot(100), units.Dot(100))
		s.Max.Set(units.Dot(100), units.Dot(100))
		s.Padding.Zero()
	})
	im := NewImage(fr).SetImage(src)
	im.Styler(func(s *styles.Style) { // filling the frame
		s.Grow.Set(1, 1)
		s.Min.Zero()
		s.ObjectFit = styles.FitContain
	})
	img := RenderImage(b)
	require.NotNil(t, img)
	bb := im.Geom.ContentBBox
	red := func(x, y int) bool {
		c := img.RGBAAt(bb.Min.X+x, bb.Min.Y+y)
		return c.R > 200 && c.G < 50 && c.B < 50
	}
	require.Equal(t, 100, bb.Dy(), "the image fills the frame")
	assert.False(t, red(50, 10), "above the image")
	assert.True(t, red(50, 50), "the middle")
	assert.False(t, red(50, 90), "below the image")
}
