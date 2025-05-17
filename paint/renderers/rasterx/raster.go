// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/rasterx:
// Copyright 2018 by the rasterx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

package rasterx

//go:generate core generate

import (
	"image"

	"golang.org/x/image/math/fixed"
)

// Raster is the interface for rasterizer types. It extends the [Adder]
// interface to include LineF and JoinF functions.
type Raster interface {
	Adder
	LineF(b fixed.Point26_6)
	JoinF()
}

// Adder is the interface for types that can accumulate path commands
type Adder interface {
	// Start starts a new curve at the given point.
	Start(a fixed.Point26_6)

	// Line adds a line segment to the path
	Line(b fixed.Point26_6)

	// QuadBezier adds a quadratic bezier curve to the path
	QuadBezier(b, c fixed.Point26_6)

	// CubeBezier adds a cubic bezier curve to the path
	CubeBezier(b, c, d fixed.Point26_6)

	// Closes the path to the start point if closeLoop is true
	Stop(closeLoop bool)
}

// Scanner is the interface for path generating types
type Scanner interface {
	Start(a fixed.Point26_6)
	Line(b fixed.Point26_6)
	Draw()
	GetPathExtent() fixed.Rectangle26_6
	SetBounds(w, h int)

	// SetColor sets the color used for rendering.
	SetColor(color image.Image)

	SetWinding(useNonZeroWinding bool)
	Clear()

	// SetClip sets an optional clipping rectangle to restrict rendering only to
	// that region. If rect is zero (image.Rectangle{}), then clipping is disabled.
	SetClip(rect image.Rectangle)
}
