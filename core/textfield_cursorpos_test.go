// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cursorColumns returns the pixel columns of the given image in the given
// rows that are mostly of the given color, as those of a text cursor.
func cursorColumns(img *image.RGBA, y0, y1 int, clr image.Image) []int {
	r, g, b, _ := colors.ToUniform(clr).RGBA()
	near := func(a uint8, b uint32) bool {
		d := int(a)<<8 - int(b)
		return d > -0x3000 && d < 0x3000
	}
	var cols []int
	for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
		n := 0
		for y := y0; y < y1; y++ {
			c := img.RGBAAt(x, y)
			if near(c.R, r) && near(c.G, g) && near(c.B, b) {
				n++
			}
		}
		if n > (y1-y0)/2 {
			cols = append(cols, x)
		}
	}
	return cols
}

// inkColumns returns whether each pixel column of the given image in the
// given rows has dark (text) pixels in it.
func inkColumns(img *image.RGBA, y0, y1 int) []bool {
	ink := make([]bool, img.Bounds().Max.X)
	for x := range ink {
		for y := y0; y < y1; y++ {
			c := img.RGBAAt(x, y)
			if int(c.R)+int(c.G)+int(c.B) < 3*160 {
				ink[x] = true
				break
			}
		}
	}
	return ink
}

// TestTextFieldCursorPosition tests that the cursor of a text field, at normal
// zoom, is drawn where [TextCursorPoint] puts it, and that it is centered in
// the space between the ink of the characters before and after it, on
// average over the positions between the letters of a text: with the Floor
// of the exact position, it was about a pixel to the left of that space,
// touching the character before it.
func TestTextFieldCursorPosition(t *testing.T) {
	text := "Hello, World! The quick brown fox jumps over lazy dogs"

	// the exact positions between the characters, and the ink of the text
	b := NewBody()
	tf := NewTextField(b).SetText(text)
	tf.Styler(func(s *styles.Style) { s.Min.X.Dp(700) })
	var xs []float32
	var y0, y1 int
	img := RenderImage(b, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		for i := 0; i <= len(text); i++ {
			p := tf.charRenderPos(i).Sub(math32.FromPoint(tf.Scene.SceneGeom.Pos))
			xs = append(xs, p.X)
			y0, y1 = int(p.Y)+2, int(p.Y+tf.lineHeight)-2
		}
	})
	ink := inkColumns(img, y0, y1)

	// the cursor is drawn at the columns of TextCursorPoint (its line is
	// 2 pixels wide, centered on it)
	const pos = 1 // between H and e
	cb := NewBody()
	ctf := NewTextField(cb).SetText(text)
	ctf.Styler(func(s *styles.Style) { s.Min.X.Dp(700) })
	cimg := RenderImage(cb, func() {
		cb.AsyncLock()
		ctf.SetFocus()
		cb.AsyncUnlock()
	}, func() {
		cb.AsyncLock()
		ctf.cursorPos = pos
		ctf.updateCursorPosition()
		cb.AsyncUnlock()
	})
	c := TextCursorPoint(math32.Vec2(xs[pos], 0)).X
	assert.Equal(t, []int{c - 1, c}, cursorColumns(cimg, y0, y1, ctf.CursorColor), "the cursor columns, at %.2f", xs[pos])

	// the gaps between the cursor and the ink before and after it
	sumBefore, sumAfter, n := 0, 0, 0
	for i := 1; i < len(text); i++ {
		if text[i-1] == ' ' || text[i] == ' ' {
			continue // only between the ink of two characters
		}
		x := xs[i]
		prev, next := -1, -1 // the last ink column left of x, the first right of it
		for c := int(x); c >= 0; c-- {
			if float32(c)+0.5 < x && ink[c] {
				prev = c
				break
			}
		}
		for c := int(x); c < len(ink); c++ {
			if float32(c)+0.5 > x && ink[c] {
				next = c
				break
			}
		}
		require.True(t, prev >= 0 && next >= 0, "ink around %d", i)
		c := TextCursorPoint(math32.Vec2(x, 0)).X
		sumBefore += (c - 1) - prev - 1 // negative: it overlaps the ink
		sumAfter += next - c - 1
		n++
	}
	before, after := float32(sumBefore)/float32(n), float32(sumAfter)/float32(n)
	t.Logf("%d positions: mean gap to the ink before %.2f, after %.2f", n, before, after)
	assert.InDelta(t, before, after, 0.5, "the cursor is centered between the characters")
}
