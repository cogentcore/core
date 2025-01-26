// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package path

import "fmt"

// Tolerance is the maximum deviation from the original path in millimeters
// when e.g. flatting. Used for flattening in the renderers, font decorations,
// and path intersections.
var Tolerance = float32(0.01)

// PixelTolerance is the maximum deviation of the rasterized path from
// the original for flattening purposed in pixels.
var PixelTolerance = float32(0.1)

// FillRule is the algorithm to specify which area is to be filled
// and which not, in particular when multiple subpaths overlap.
// The NonZero rule is the default and will fill any point that is
// being enclosed by an unequal number of paths winding clock-wise
// and counter clock-wise, otherwise it will not be filled.
// The EvenOdd rule will fill any point that is being enclosed by
// an uneven number of paths, whichever their direction.
// Positive fills only counter clock-wise oriented paths,
// while Negative fills only clock-wise oriented paths.
type FillRule int

// see FillRule
const (
	NonZero FillRule = iota
	EvenOdd
	Positive
	Negative
)

func (fillRule FillRule) Fills(windings int) bool {
	switch fillRule {
	case NonZero:
		return windings != 0
	case EvenOdd:
		return windings%2 != 0
	case Positive:
		return 0 < windings
	case Negative:
		return windings < 0
	}
	return false
}

func (fillRule FillRule) String() string {
	switch fillRule {
	case NonZero:
		return "NonZero"
	case EvenOdd:
		return "EvenOdd"
	case Positive:
		return "Positive"
	case Negative:
		return "Negative"
	}
	return fmt.Sprintf("FillRule(%d)", fillRule)
}
