// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scan

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/image/math/fixed"
)

// fp returns the fixed point of the given coordinates.
func fp(x, y float64) fixed.Point26_6 {
	return fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6(y * 64)}
}

// polygon adds the closed polygon of the given points to the scanner.
func polygon(s *Scanner, pts ...[2]float64) {
	s.Start(fp(pts[0][0], pts[0][1]))
	for _, p := range pts[1:] {
		s.Line(fp(p[0], p[1]))
	}
	s.Line(fp(pts[0][0], pts[0][1]))
}

// square adds the square with the given corner and size.
func square(s *Scanner, x, y, size float64) {
	polygon(s, [2]float64{x, y}, [2]float64{x + size, y}, [2]float64{x + size, y + size}, [2]float64{x, y + size})
}

var red = color.RGBA{255, 0, 0, 255}

// newImgScanner returns a new scanner of an [ImgSpanner] on a new image.
func newImgScanner(w, h int) (*Scanner, *image.RGBA) {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	sc := NewScanner(NewImgSpanner(img), w, h)
	sc.SetColor(image.NewUniform(red))
	return sc, img
}

// TestImgSpannerSquare tests filling a square, which covers its pixels,
// with partial coverage at its fractional edges.
func TestImgSpannerSquare(t *testing.T) {
	sc, img := newImgScanner(20, 20)
	square(sc, 2.5, 2, 10)
	sc.Draw()
	assert.Equal(t, red, img.RGBAAt(7, 7), "inside")
	assert.Equal(t, color.RGBA{}, img.RGBAAt(15, 15), "outside")
	edge := img.RGBAAt(2, 7)
	assert.InDelta(t, 128, edge.A, 2, "half covered at x=2.5")
	assert.InDelta(t, 128, img.RGBAAt(12, 7).A, 2)
	assert.Equal(t, uint8(255), img.RGBAAt(3, 2).A, "the top edge is on a pixel boundary")

	ext := sc.GetPathExtent()
	assert.Equal(t, fp(2.5, 2), ext.Min)
	assert.Equal(t, fp(12.5, 12), ext.Max)
	sc.Clear()
	assert.Empty(t, sc.Cell)
}

// TestWinding tests the nonzero and even-odd winding rules with two
// overlapping squares, whose overlap is filled only with nonzero.
func TestWinding(t *testing.T) {
	for _, nonZero := range []bool{true, false} {
		sc, img := newImgScanner(20, 20)
		sc.SetWinding(nonZero)
		square(sc, 0, 0, 10)
		square(sc, 5, 5, 10)
		sc.Draw()
		assert.Equal(t, red, img.RGBAAt(2, 2))
		assert.Equal(t, red, img.RGBAAt(12, 12))
		if nonZero {
			assert.Equal(t, red, img.RGBAAt(7, 7), "the overlap is filled")
		} else {
			assert.Equal(t, uint8(0), img.RGBAAt(7, 7).A, "the overlap is a hole")
		}
	}
}

// TestClipAndBounds tests that drawing is clipped, and that paths outside
// of the bounds are ignored.
func TestClipAndBounds(t *testing.T) {
	sc, img := newImgScanner(20, 20)
	sc.SetClip(image.Rect(0, 0, 5, 20))
	square(sc, -5, -5, 30)
	sc.Draw()
	assert.Equal(t, red, img.RGBAAt(2, 10))
	assert.Equal(t, uint8(0), img.RGBAAt(10, 10).A, "clipped")

	sc.SetBounds(-1, -1)
	assert.Equal(t, 0, sc.Width)
	sc.SetBounds(20, 20)
	assert.Len(t, sc.CellIndex, 20)
}

// TestColorImage tests filling with a non-uniform color image, which is
// sampled for each pixel.
func TestColorImage(t *testing.T) {
	sc, img := newImgScanner(20, 20)
	src := image.NewRGBA(image.Rect(0, 0, 20, 20))
	draw.Draw(src, image.Rect(0, 0, 10, 20), image.NewUniform(color.RGBA{0, 0, 255, 255}), image.Point{}, draw.Src)
	draw.Draw(src, image.Rect(10, 0, 20, 20), image.NewUniform(color.RGBA{0, 255, 0, 255}), image.Point{}, draw.Src)
	sc.SetColor(src)
	square(sc, 0, 0, 20)
	sc.Draw()
	assert.Equal(t, color.RGBA{0, 0, 255, 255}, img.RGBAAt(3, 3))
	assert.Equal(t, color.RGBA{0, 255, 0, 255}, img.RGBAAt(15, 3))
}

// TestOpOverSrc tests the draw operations of the image spanner: over
// blends a translucent color with what is under it, and src replaces it.
func TestOpOverSrc(t *testing.T) {
	for _, op := range []draw.Op{draw.Over, draw.Src} {
		img := image.NewRGBA(image.Rect(0, 0, 10, 10))
		draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{0, 0, 255, 255}), image.Point{}, draw.Src)
		sp := NewImgSpanner(img)
		sp.Op = op
		sc := NewScanner(sp, 10, 10)
		sc.SetColor(image.NewUniform(color.RGBA{128, 0, 0, 128})) // premultiplied half red
		square(sc, 0, 0, 10)
		sc.Draw()
		c := img.RGBAAt(5, 5)
		if op == draw.Over {
			assert.InDelta(t, 128, c.R, 2)
			assert.InDelta(t, 127, c.B, 2, "the blue under it shows through")
			assert.Equal(t, uint8(255), c.A)
		} else {
			assert.Equal(t, color.RGBA{128, 0, 0, 128}, c, "replaced")
		}
	}
}

// TestLinkListSpanner tests that the linked list spanner, which
// accumulates spans and then draws them, draws the same image as the
// image spanner, for opaque and translucent colors.
func TestLinkListSpanner(t *testing.T) {
	for _, clr := range []color.RGBA{red, {0, 100, 0, 128}} {
		sc, img := newImgScanner(30, 30)
		sc.SetColor(image.NewUniform(clr))
		square(sc, 3.5, 4.25, 15)
		polygon(sc, [2]float64{10, 10}, [2]float64{28, 12}, [2]float64{16, 28})
		sc.Draw()

		ll := &LinkListSpanner{}
		ll.SetBounds(image.Rect(0, 0, 30, 30))
		ll.SetBgColor(image.NewUniform(color.RGBA{}))
		lsc := NewScanner(ll, 30, 30)
		lsc.SetColor(image.NewUniform(clr))
		square(lsc, 3.5, 4.25, 15)
		polygon(lsc, [2]float64{10, 10}, [2]float64{28, 12}, [2]float64{16, 28})
		lsc.Draw()
		limg := image.NewRGBA(image.Rect(0, 0, 30, 30))
		ll.DrawToImage(limg)

		diff := 0
		for i := range img.Pix {
			d := int(img.Pix[i]) - int(limg.Pix[i])
			diff = max(diff, d, -d)
		}
		assert.LessOrEqual(t, diff, 2, "color %v", clr)

		ll.Clear()
		pix := make([]uint8, len(limg.Pix))
		ll.SpansToPix(pix, limg.Stride)
		assert.Equal(t, make([]uint8, len(pix)), pix, "nothing after clear")
	}
}
