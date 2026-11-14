// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"image/color"
	"testing"

	"cogentcore.org/core/colors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDragImage tests the images of a button and a tree node for the
// sprite of dragging them (see [Events.DragStart]), on all platforms,
// including the web (see goscript_test.go): they are the size of the
// widget, opaque, with the background of its parent at the corner, and
// with its text (#1552: on the web, the text was missing).
func TestDragImage(t *testing.T) {
	b := NewBody()
	bt := NewButton(b).SetText("Drag this text")
	tr := NewTree(b).SetText("Drag this node") // (as on the tree page of the docs)
	NewTree(tr).SetText("Child")
	b.runAndShowNewWindow()
	defer func() {
		rc := b.Scene.renderContext()
		rc.Lock()
		b.Close()
		rc.Unlock()
	}()
	b.waitNoEvents(b.Scene.RenderWindow())
	testDragImage(t, b, bt, bt.Text)
	testDragImage(t, b, tr, tr.Text)
}

// testDragImage tests the drag image of the given widget, with its text.
func testDragImage(t *testing.T, b *Body, w Widget, label string) {
	b.AsyncLock()
	img := grabRenderFrom(w)
	wb := w.AsWidget()
	size := wb.Geom.TotalBBox.Size()
	parent := colors.AsRGBA(colors.ToUniform(wb.parentActualBackground()))
	b.AsyncUnlock()
	require.NotNil(t, img, label)
	assert.Equal(t, size, img.Bounds().Size(), label)

	opaque := true
	for y := range size.Y {
		for x := range size.X {
			if img.RGBAAt(x, y).A != 255 {
				opaque = false
			}
		}
	}
	assert.True(t, opaque, "%s: opaque, as the widget is shown", label)
	assert.True(t, near(parent, img.RGBAAt(0, 0)), "%s: the background of the parent at the corner: %v, not %v", label, img.RGBAAt(0, 0), parent)

	// the text: pixels in the middle that are not the color of the background
	mid := image.Rect(size.X/4, size.Y/3, 3*size.X/4, 2*size.Y/3)
	bg := img.RGBAAt(size.X-2, size.Y/2)
	text := 0
	for y := mid.Min.Y; y < mid.Max.Y; y++ {
		for x := mid.Min.X; x < mid.Max.X; x++ {
			if !near(bg, img.RGBAAt(x, y)) {
				text++
			}
		}
	}
	assert.Greater(t, text, mid.Dx()*mid.Dy()/40, "%s: the text is drawn", label)
}

// near returns whether the given colors are nearly the same.
func near(a, b color.RGBA) bool {
	d := func(x, y uint8) int { return max(int(x)-int(y), int(y)-int(x)) }
	return d(a.R, b.R)+d(a.G, b.G)+d(a.B, b.B)+d(a.A, b.A) < 40
}
