// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"image"
)

// Geom2DInt defines a geometry in 2D dots units (int) -- this is just a more
// convenient format than image.Rectangle for cases where the size and
// position are independently updated (e.g., Viewport)
type Geom2DInt struct {
	Pos  image.Point
	Size image.Point
}

// Bounds converts geom to equivalent image.Rectangle
func (gm *Geom2DInt) Bounds() image.Rectangle {
	return image.Rect(gm.Pos.X, gm.Pos.Y, gm.Pos.X+gm.Size.X, gm.Pos.Y+gm.Size.Y)
}

// Box2 returns bounds as a [Box2].
func (gm *Geom2DInt) Box2() Box2 {
	return B2FromRect(gm.Bounds())
}

// SizeRect converts geom to rect version of size at 0 pos.
func (gm *Geom2DInt) SizeRect() image.Rectangle {
	return image.Rect(0, 0, gm.Size.X, gm.Size.Y)
}

// SetRect sets values from image.Rectangle.
func (gm *Geom2DInt) SetRect(r image.Rectangle) {
	gm.Pos = r.Min
	gm.Size = r.Size()
}

// FitGeomInWindow returns a position and size for a region (sub-window)
// within a larger window geom (pos and size) that fits entirely
// within that window to the extent possible,
// given an initial starting position and size.
// The position is first adjusted to try to fit the size, and then the size
// is adjusted to make it fit if it is still too big.
func FitGeomInWindow(stPos, stSz, winPos, winSz int) (pos, sz int) {
	pos = stPos
	sz = stSz
	// we go through two iterations: one to fix our position and one to fix
	// our size. this ensures that we adjust position and not size if we can,
	// but we still always end up with valid dimensions by using size as a fallback.
	if pos < winPos {
		pos = winPos
	}
	if pos+sz > winPos+winSz { // our max > window max
		pos = winPos + winSz - sz // window max - our size
	}
	if pos < winPos {
		pos = winPos
	}
	if pos+sz > winPos+winSz { // our max > window max
		sz = winSz + winPos - pos // window max - our min
	}
	return
}

// FitInWindow returns a position and size for a region (sub-window)
// within a larger window geom that fits entirely within that window to the
// extent possible, for the initial "ideal" starting position and size.
// The position is first adjusted to try to fit the size, and then the size
// is adjusted to make it fit if it is still too big.
func (gm *Geom2DInt) FitInWindow(win Geom2DInt) Geom2DInt {
	var fit Geom2DInt
	fit.Pos.X, fit.Size.X = FitGeomInWindow(gm.Pos.X, gm.Size.X, win.Pos.X, win.Size.X)
	fit.Pos.Y, fit.Size.Y = FitGeomInWindow(gm.Pos.Y, gm.Size.Y, win.Pos.Y, win.Size.Y)
	return fit
}
