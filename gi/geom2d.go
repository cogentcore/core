// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"github.com/goki/mat32"
)

const (
	X = mat32.X
	Y = mat32.Y
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

// SizeRect converts geom to rect version of size at 0 pos
func (gm *Geom2DInt) SizeRect() image.Rectangle {
	return image.Rect(0, 0, gm.Size.X, gm.Size.Y)
}

// SetRect sets values from image.Rectangle
func (gm *Geom2DInt) SetRect(r image.Rectangle) {
	gm.Pos = r.Min
	gm.Size = r.Size()
}
