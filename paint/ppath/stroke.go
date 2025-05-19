// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package ppath

import "cogentcore.org/core/math32"

//go:generate core generate

// FillRules specifies the algorithm for which area is to be filled and which not,
// in particular when multiple subpaths overlap. The NonZero rule is the default
// and will fill any point that is being enclosed by an unequal number of paths
// winding clock-wise and counter clock-wise, otherwise it will not be filled.
// The EvenOdd rule will fill any point that is being enclosed by an uneven number
// of paths, whichever their direction. Positive fills only counter clock-wise
// oriented paths, while Negative fills only clock-wise oriented paths.
type FillRules int32 //enums:enum -transform lower

const (
	NonZero FillRules = iota
	EvenOdd
	Positive
	Negative
)

func (fr FillRules) Fills(windings int) bool {
	switch fr {
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

// todo: these need serious work:

// VectorEffects contains special effects for rendering
type VectorEffects int32 //enums:enum -trim-prefix VectorEffect -transform kebab

const (
	VectorEffectNone VectorEffects = iota

	// VectorEffectNonScalingStroke means that the stroke width is not affected by
	// transform properties
	VectorEffectNonScalingStroke
)

// Caps specifies the end-cap of a stroked line: stroke-linecap property in SVG
type Caps int32 //enums:enum -trim-prefix Cap -transform kebab

const (
	// CapButt indicates to draw no line caps; it draws a
	// line with the length of the specified length.
	CapButt Caps = iota

	// CapRound indicates to draw a semicircle on each line
	// end with a diameter of the stroke width.
	CapRound

	// CapSquare indicates to draw a rectangle on each line end
	// with a height of the stroke width and a width of half of the
	// stroke width.
	CapSquare
)

// Joins specifies the way stroked lines are joined together:
// stroke-linejoin property in SVG
type Joins int32 //enums:enum -trim-prefix Join -transform kebab

const (
	JoinMiter Joins = iota
	JoinMiterClip
	JoinRound
	JoinBevel
	JoinArcs
	JoinArcsClip
)

// Dash patterns
var (
	Solid              = []float32{}
	Dotted             = []float32{1.0, 2.0}
	DenselyDotted      = []float32{1.0, 1.0}
	SparselyDotted     = []float32{1.0, 4.0}
	Dashed             = []float32{3.0, 3.0}
	DenselyDashed      = []float32{3.0, 1.0}
	SparselyDashed     = []float32{3.0, 6.0}
	Dashdotted         = []float32{3.0, 2.0, 1.0, 2.0}
	DenselyDashdotted  = []float32{3.0, 1.0, 1.0, 1.0}
	SparselyDashdotted = []float32{3.0, 4.0, 1.0, 4.0}
)

func ScaleDash(scale float32, offset float32, d []float32) (float32, []float32) {
	d2 := make([]float32, len(d))
	for i := range d {
		d2[i] = d[i] * scale
	}
	return offset * scale, d2
}

// DirectionIndex returns the direction of the path at the given index
// into Path and t in [0.0,1.0]. Path must not contain subpaths,
// and will return the path's starting direction when i points
// to a MoveTo, or the path's final direction when i points to
// a Close of zero-length.
func DirectionIndex(p Path, i int, t float32) math32.Vector2 {
	last := len(p)
	if p[last-1] == Close && EqualPoint(math32.Vec2(p[last-CmdLen(Close)-3], p[last-CmdLen(Close)-2]), math32.Vec2(p[last-3], p[last-2])) {
		// point-closed
		last -= CmdLen(Close)
	}

	if i == 0 {
		// get path's starting direction when i points to MoveTo
		i = 4
		t = 0.0
	} else if i < len(p) && i == last {
		// get path's final direction when i points to zero-length Close
		i -= CmdLen(p[i-1])
		t = 1.0
	}
	if i < 0 || len(p) <= i || last < i+CmdLen(p[i]) {
		return math32.Vector2{}
	}

	cmd := p[i]
	var start math32.Vector2
	if i == 0 {
		start = math32.Vec2(p[last-3], p[last-2])
	} else {
		start = math32.Vec2(p[i-3], p[i-2])
	}

	i += CmdLen(cmd)
	end := math32.Vec2(p[i-3], p[i-2])
	switch cmd {
	case LineTo, Close:
		return end.Sub(start).Normal()
	case QuadTo:
		cp := math32.Vec2(p[i-5], p[i-4])
		return QuadraticBezierDeriv(start, cp, end, t).Normal()
	case CubeTo:
		cp1 := math32.Vec2(p[i-7], p[i-6])
		cp2 := math32.Vec2(p[i-5], p[i-4])
		return CubicBezierDeriv(start, cp1, cp2, end, t).Normal()
	case ArcTo:
		rx, ry, phi := p[i-7], p[i-6], p[i-5]
		large, sweep := ToArcFlags(p[i-4])
		_, _, theta0, theta1 := EllipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
		theta := theta0 + t*(theta1-theta0)
		return EllipseDeriv(rx, ry, phi, sweep, theta).Normal()
	}
	return math32.Vector2{}
}

// Direction returns the direction of the path at the given
// segment and t in [0.0,1.0] along that path.
// The direction is a vector of unit length.
func (p Path) Direction(seg int, t float32) math32.Vector2 {
	if len(p) <= 4 {
		return math32.Vector2{}
	}

	curSeg := 0
	iStart, iSeg, iEnd := 0, 0, 0
	for i := 0; i < len(p); {
		cmd := p[i]
		if cmd == MoveTo {
			if seg < curSeg {
				pi := p[iStart:iEnd]
				return DirectionIndex(pi, iSeg-iStart, t)
			}
			iStart = i
		}
		if seg == curSeg {
			iSeg = i
		}
		i += CmdLen(cmd)
	}
	return math32.Vector2{} // if segment doesn't exist
}

// CoordDirections returns the direction of the segment start/end points.
// It will return the average direction at the intersection of two
// end points, and for an open path it will simply return the direction
// of the start and end points of the path.
func (p Path) CoordDirections() []math32.Vector2 {
	if len(p) <= 4 {
		return []math32.Vector2{{}}
	}
	last := len(p)
	if p[last-1] == Close && EqualPoint(math32.Vec2(p[last-CmdLen(Close)-3], p[last-CmdLen(Close)-2]), math32.Vec2(p[last-3], p[last-2])) {
		// point-closed
		last -= CmdLen(Close)
	}

	dirs := []math32.Vector2{}
	var closed bool
	var dirPrev math32.Vector2
	for i := 4; i < last; {
		cmd := p[i]
		dir := DirectionIndex(p, i, 0.0)
		if i == 0 {
			dirs = append(dirs, dir)
		} else {
			dirs = append(dirs, dirPrev.Add(dir).Normal())
		}
		dirPrev = DirectionIndex(p, i, 1.0)
		closed = cmd == Close
		i += CmdLen(cmd)
	}
	if closed {
		dirs[0] = dirs[0].Add(dirPrev).Normal()
		dirs = append(dirs, dirs[0])
	} else {
		dirs = append(dirs, dirPrev)
	}
	return dirs
}
