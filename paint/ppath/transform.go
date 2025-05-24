// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package ppath

import (
	"fmt"

	"cogentcore.org/core/math32"
)

// Transform transforms the path by the given transformation matrix
// and returns a new path. It modifies the path in-place.
func (p Path) Transform(m math32.Matrix2) Path {
	xscale, yscale := m.ExtractScale()
	for i := 0; i < len(p); {
		cmd := p[i]
		switch cmd {
		case MoveTo, LineTo, Close:
			if i+2 >= len(p) {
				fmt.Println("path length error:", len(p), i, p)
				return p
			}
			end := m.MulVector2AsPoint(math32.Vec2(p[i+1], p[i+2]))
			p[i+1] = end.X
			p[i+2] = end.Y
		case QuadTo:
			cp := m.MulVector2AsPoint(math32.Vec2(p[i+1], p[i+2]))
			end := m.MulVector2AsPoint(math32.Vec2(p[i+3], p[i+4]))
			p[i+1] = cp.X
			p[i+2] = cp.Y
			p[i+3] = end.X
			p[i+4] = end.Y
		case CubeTo:
			cp1 := m.MulVector2AsPoint(math32.Vec2(p[i+1], p[i+2]))
			cp2 := m.MulVector2AsPoint(math32.Vec2(p[i+3], p[i+4]))
			end := m.MulVector2AsPoint(math32.Vec2(p[i+5], p[i+6]))
			p[i+1] = cp1.X
			p[i+2] = cp1.Y
			p[i+3] = cp2.X
			p[i+4] = cp2.Y
			p[i+5] = end.X
			p[i+6] = end.Y
		case ArcTo:
			rx, ry, phi, large, sweep, end := p.ArcToPoints(i)

			// For ellipses written as the conic section equation in matrix form, we have:
			// [x, y] E [x; y] = 0, with E = [1/rx^2, 0; 0, 1/ry^2]
			// For our transformed ellipse we have [x', y'] = T [x, y], with T the affine
			// transformation matrix so that
			// (T^-1 [x'; y'])^T E (T^-1 [x'; y'] = 0  =>  [x', y'] T^(-T) E T^(-1) [x'; y'] = 0
			// We define Q = T^(-1,T) E T^(-1) the new ellipse equation which is typically rotated
			// from the x-axis. That's why we find the eigenvalues and eigenvectors (the new
			// direction and length of the major and minor axes).
			T := m.Rotate(phi)
			invT := T.Inverse()
			Q := math32.Identity2().Scale(1.0/rx/rx, 1.0/ry/ry)
			Q = invT.Transpose().Mul(Q).Mul(invT)

			lambda1, lambda2, v1, v2 := Q.Eigen()
			rx = 1 / math32.Sqrt(lambda1)
			ry = 1 / math32.Sqrt(lambda2)
			phi = Angle(v1)
			if rx < ry {
				rx, ry = ry, rx
				phi = Angle(v2)
			}
			phi = AngleNorm(phi)
			if math32.Pi <= phi { // phi is canonical within 0 <= phi < 180
				phi -= math32.Pi
			}

			if xscale*yscale < 0.0 { // flip x or y axis needs flipping of the sweep
				sweep = !sweep
			}
			end = m.MulVector2AsPoint(end)

			p[i+1] = rx
			p[i+2] = ry
			p[i+3] = phi
			p[i+4] = fromArcFlags(large, sweep)
			p[i+5] = end.X
			p[i+6] = end.Y
		}
		i += CmdLen(cmd)
	}
	return p
}

// Translate translates the path by (x,y) and returns a new path.
func (p Path) Translate(x, y float32) Path {
	return p.Transform(math32.Identity2().Translate(x, y))
}

// Scale scales the path by (x,y) and returns a new path.
func (p Path) Scale(x, y float32) Path {
	return p.Transform(math32.Identity2().Scale(x, y))
}

// ReplaceArcs replaces ArcTo commands by CubeTo commands and returns a new path.
func (p *Path) ReplaceArcs() Path {
	return p.Replace(nil, nil, nil, ArcToCube)
}

// Replace replaces path segments by their respective functions,
// each returning the path that will replace the segment or nil
// if no replacement is to be performed. The line function will
// take the start and end points. The bezier function will take
// the start point, control point 1 and 2, and the end point
// (i.e. a cubic Bézier, quadratic Béziers will be implicitly
// converted to cubic ones). The arc function will take a start point,
// the major and minor radii, the radial rotaton counter clockwise,
// the large and sweep booleans, and the end point.
// The replacing path will replace the path segment without any checks,
// you need to make sure the be moved so that its start point connects
// with the last end point of the base path before the replacement.
// If the end point of the replacing path is different that the end point
// of what is replaced, the path that follows will be displaced.
func (p Path) Replace(
	line func(math32.Vector2, math32.Vector2) Path,
	quad func(math32.Vector2, math32.Vector2, math32.Vector2) Path,
	cube func(math32.Vector2, math32.Vector2, math32.Vector2, math32.Vector2) Path,
	arc func(math32.Vector2, float32, float32, float32, bool, bool, math32.Vector2) Path,
) Path {
	copied := false
	var start, end, cp1, cp2 math32.Vector2
	for i := 0; i < len(p); {
		var q Path
		cmd := p[i]
		switch cmd {
		case LineTo, Close:
			if line != nil {
				end = p.EndPoint(i)
				q = line(start, end)
				if cmd == Close {
					q.Close()
				}
			}
		case QuadTo:
			if quad != nil {
				cp1, end = p.QuadToPoints(i)
				q = quad(start, cp1, end)
			}
		case CubeTo:
			if cube != nil {
				cp1, cp2, end = p.CubeToPoints(i)
				q = cube(start, cp1, cp2, end)
			}
		case ArcTo:
			if arc != nil {
				var rx, ry, phi float32
				var large, sweep bool
				rx, ry, phi, large, sweep, end = p.ArcToPoints(i)
				q = arc(start, rx, ry, phi, large, sweep, end)
			}
		}

		if q != nil {
			if !copied {
				p = p.Clone()
				copied = true
			}

			r := append(Path{MoveTo, end.X, end.Y, MoveTo}, p[i+CmdLen(cmd):]...)

			p = p[: i : i+CmdLen(cmd)] // make sure not to overwrite the rest of the path
			p = p.Join(q)
			if cmd != Close {
				p.LineTo(end.X, end.Y)
			}

			i = len(p)
			p = p.Join(r) // join the rest of the base path
		} else {
			i += CmdLen(cmd)
		}
		start = math32.Vec2(p[i-3], p[i-2])
	}
	return p
}

// Split splits the path into its independent subpaths.
// The path is split before each MoveTo command.
func (p Path) Split() []Path {
	if p == nil {
		return nil
	}
	var i, j int
	ps := []Path{}
	for j < len(p) {
		cmd := p[j]
		if i < j && cmd == MoveTo {
			ps = append(ps, p[i:j:j])
			i = j
		}
		j += CmdLen(cmd)
	}
	if i+CmdLen(MoveTo) < j {
		ps = append(ps, p[i:j:j])
	}
	return ps
}

// Reverse returns a new path that is the same path as p but in the reverse direction.
func (p Path) Reverse() Path {
	if len(p) == 0 {
		return p
	}

	end := math32.Vector2{p[len(p)-3], p[len(p)-2]}
	q := make(Path, 0, len(p))
	q = append(q, MoveTo, end.X, end.Y, MoveTo)

	closed := false
	first, start := end, end
	for i := len(p); 0 < i; {
		cmd := p[i-1]
		i -= CmdLen(cmd)

		end = math32.Vector2{}
		if 0 < i {
			end = math32.Vector2{p[i-3], p[i-2]}
		}

		switch cmd {
		case MoveTo:
			if closed {
				q = append(q, Close, first.X, first.Y, Close)
				closed = false
			}
			if i != 0 {
				q = append(q, MoveTo, end.X, end.Y, MoveTo)
				first = end
			}
		case Close:
			if !EqualPoint(start, end) {
				q = append(q, LineTo, end.X, end.Y, LineTo)
			}
			closed = true
		case LineTo:
			if closed && (i == 0 || p[i-1] == MoveTo) {
				q = append(q, Close, first.X, first.Y, Close)
				closed = false
			} else {
				q = append(q, LineTo, end.X, end.Y, LineTo)
			}
		case QuadTo:
			cx, cy := p[i+1], p[i+2]
			q = append(q, QuadTo, cx, cy, end.X, end.Y, QuadTo)
		case CubeTo:
			cx1, cy1 := p[i+1], p[i+2]
			cx2, cy2 := p[i+3], p[i+4]
			q = append(q, CubeTo, cx2, cy2, cx1, cy1, end.X, end.Y, CubeTo)
		case ArcTo:
			rx, ry, phi, large, sweep, _ := p.ArcToPoints(i)
			q = append(q, ArcTo, rx, ry, phi, fromArcFlags(large, !sweep), end.X, end.Y, ArcTo)
		}
		start = end
	}
	if closed {
		q = append(q, Close, first.X, first.Y, Close)
	}
	return q
}
