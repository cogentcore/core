// Copyright 2016 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package drawer provides functions that help implement screen.Drawer methods.
package drawer

import (
	"image"
	"image/draw"

	"github.com/goki/gi/oswin"
	"github.com/goki/mat32"
)

// Copy implements the Copy method of the oswin.Drawer interface by calling
// the Draw method of that same interface.
func Copy(dst oswin.Drawer, dp image.Point, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	dst.Draw(mat32.Mat3{
		1, 0, 0,
		0, 1, 0,
		float32(dp.X - sr.Min.X), float32(dp.Y - sr.Min.Y), 1,
	}, src, sr, op, opts)
}

// Scale implements the Scale method of the oswin.Drawer interface by calling
// the Draw method of that same interface.
func Scale(dst oswin.Drawer, dr image.Rectangle, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	rx := float32(dr.Dx()) / float32(sr.Dx())
	ry := float32(dr.Dy()) / float32(sr.Dy())
	dst.Draw(mat32.Mat3{
		rx, 0, 0,
		0, ry, 0,
		float32(dr.Min.X) - rx*float32(sr.Min.X), float32(dr.Min.Y) - ry*float32(sr.Min.Y), 1,
	}, src, sr, op, opts)
}
