// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package oswin

import (
	"image"
	"image/color"
	"image/draw"

	"golang.org/x/image/math/f64"
)

// Texture is a pixel buffer, but not one that is directly accessible as a
// []byte. Conceptually, it could live on a GPU, in another process or even be
// across a network, instead of on a CPU in this process.
//
// Images can be uploaded to Textures, and Textures can be drawn on Windows.
//
// When specifying a sub-Texture via Draw, a Texture's top-left pixel is always
// (0, 0) in its own coordinate space.
type Texture interface {
	// Release releases the Texture's resources, after all pending uploads and
	// draws resolve.
	//
	// The behavior of the Texture after Release, whether calling its methods
	// or passing it as an argument, is undefined.
	Release()

	// Size returns the size of the Texture's image.
	Size() image.Point

	// Bounds returns the bounds of the Texture's image. It is equal to
	// image.Rectangle{Max: t.Size()}.
	Bounds() image.Rectangle

	Uploader

	// TODO: also implement Drawer? If so, merge the Uploader and Drawer
	// interfaces??
}

// Drawer is something you can draw Textures on.
//
// Draw is the most general purpose of this interface's methods. It supports
// arbitrary affine transformations, such as translations, scales and
// rotations.
//
// Copy and Scale are more specific versions of Draw. The affected dst pixels
// are an axis-aligned rectangle, quantized to the pixel grid. Copy copies
// pixels in a 1:1 manner, Scale is more general. They have simpler parameters
// than Draw, using ints instead of float64s.
//
// When drawing on a Window, there will not be any visible effect until Publish
// is called.
type Drawer interface {
	// Draw draws the sub-Texture defined by src and sr to the destination (the
	// method receiver). src2dst defines how to transform src coordinates to
	// dst coordinates. For example, if src2dst is the matrix
	//
	// m00 m01 m02
	// m10 m11 m12
	//
	// then the src-space point (sx, sy) maps to the dst-space point
	// (m00*sx + m01*sy + m02, m10*sx + m11*sy + m12).
	Draw(src2dst f64.Aff3, src Texture, sr image.Rectangle, op draw.Op, opts *DrawOptions)

	// DrawUniform is like Draw except that the src is a uniform color instead
	// of a Texture.
	DrawUniform(src2dst f64.Aff3, src color.Color, sr image.Rectangle, op draw.Op, opts *DrawOptions)

	// Copy copies the sub-Texture defined by src and sr to the destination
	// (the method receiver), such that sr.Min in src-space aligns with dp in
	// dst-space.
	Copy(dp image.Point, src Texture, sr image.Rectangle, op draw.Op, opts *DrawOptions)

	// Scale scales the sub-Texture defined by src and sr to the destination
	// (the method receiver), such that sr in src-space is mapped to dr in
	// dst-space.
	Scale(dr image.Rectangle, src Texture, sr image.Rectangle, op draw.Op, opts *DrawOptions)
}

// These draw.Op constants are provided so that users of this package don't
// have to explicitly import "image/draw".
const (
	Over = draw.Over
	Src  = draw.Src
)

// DrawOptions are optional arguments to Draw.
type DrawOptions struct {
	// TODO: transparency in [0x0000, 0xffff]?
	// TODO: scaler (nearest neighbor vs linear)?
}
