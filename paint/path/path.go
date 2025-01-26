// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package path

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unsafe"

	"cogentcore.org/core/math32"
	"golang.org/x/image/vector"
)

// Path is a collection of MoveTo, LineTo, QuadTo, CubeTo, ArcTo, and Close
// commands, each followed the float32 coordinate data for it.
// The first value is the command itself (as a float32). The last two values
// is the end point position of the pen after the action (x,y).
// QuadTo defines one control point (x,y) in between,
// CubeTo defines two control points.
// ArcTo defines (rx,ry,phi,large+sweep) i.e. the radius in x and y,
// its rotation (in radians) and the large and sweep booleans in one float32.
// ArcTo is generally converted to equivalent CubeTo after path intersection
// computations have been performed, to simplify rasterization.
// Only valid commands are appended, so that LineTo has a non-zero length,
// QuadTo's and CubeTo's control point(s) don't (both) overlap with the start
// and end point.
type Path []Cmd

// Path is a render item.
func (pt Path) isRenderItem() {
}

// Cmd is one path command, or the float32 oordinate data for that command.
type Cmd float32

// Commands
const (
	MoveTo Cmd = 0
	LineTo     = 1
	QuadTo     = 2
	CubeTo     = 3
	ArcTo      = 4
	Close      = 5
)

var cmdLens = [6]int{4, 4, 6, 8, 8, 4}

func (cmd Cmd) cmdLen() int {
	return cmdLens[int(cmd)]
}

type Paths []Path

func (ps Paths) Empty() bool {
	for _, p := range ps {
		if !p.Empty() {
			return false
		}
	}
	return true
}

func (p Path) AsFloat32() []float32 {
	return unsafe.Slice((*float32)(unsafe.SliceData(p)), len(p))
}

func NewPathFromFloat32(d []float32) Path {
	return unsafe.Slice((*Cmd)(unsafe.SliceData(d)), len(d))
}

// toArcFlags converts to the largeArc and sweep boolean flags given its value in the path.
func toArcFlags(f float32) (bool, bool) {
	large := (f == 1.0 || f == 3.0)
	sweep := (f == 2.0 || f == 3.0)
	return large, sweep
}

// fromArcFlags converts the largeArc and sweep boolean flags to a value stored in the path.
func fromArcFlags(large, sweep bool) float32 {
	f := 0.0
	if large {
		f += 1.0
	}
	if sweep {
		f += 2.0
	}
	return f
}

// Reset clears the path but retains the same memory.
// This can be used in loops where you append and process
// paths every iteration, and avoid new memory allocations.
func (p *Path) Reset() {
	*p = (*p)[:0]
}

// GobEncode implements the gob interface.
func (p Path) GobEncode() ([]byte, error) {
	b := bytes.Buffer{}
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(p); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// GobDecode implements the gob interface.
func (p *Path) GobDecode(b []byte) error {
	dec := gob.NewDecoder(bytes.NewReader(b))
	return dec.Decode(p)
}

// Empty returns true if p is an empty path or consists of only MoveTos and Closes.
func (p Path) Empty() bool {
	return len(p) <= cmdLen(MoveTo)
}

// Equals returns true if p and q are equal within tolerance Epsilon.
func (p Path) Equals(q Path) bool {
	if len(p) != len(q) {
		return false
	}
	for i := 0; i < len(p); i++ {
		if !Equal(p[i], q[i]) {
			return false
		}
	}
	return true
}

// Sane returns true if the path is sane, ie. it does not have NaN or infinity values.
func (p Path) Sane() bool {
	sane := func(x float32) bool {
		return !math.IsNaN(x) && !math.IsInf(x, 0.0)
	}
	for i := 0; i < len(p); {
		cmd := p[i]
		i += cmdLen(cmd)

		if !sane(p[i-3]) || !sane(p[i-2]) {
			return false
		}
		switch cmd {
		case QuadTo:
			if !sane(p[i-5]) || !sane(p[i-4]) {
				return false
			}
		case CubeTo, ArcTo:
			if !sane(p[i-7]) || !sane(p[i-6]) || !sane(p[i-5]) || !sane(p[i-4]) {
				return false
			}
		}
	}
	return true
}

// Same returns true if p and q are equal shapes within tolerance Epsilon.
// Path q may start at an offset into path p or may be in the reverse direction.
func (p Path) Same(q Path) bool {
	// TODO: improve, does not handle subpaths or Close vs LineTo
	if len(p) != len(q) {
		return false
	}
	qr := q.Reverse() // TODO: can we do without?
	for j := 0; j < len(q); {
		equal := true
		for i := 0; i < len(p); i++ {
			if !Equal(p[i], q[(j+i)%len(q)]) {
				equal = false
				break
			}
		}
		if equal {
			return true
		}

		// backwards
		equal = true
		for i := 0; i < len(p); i++ {
			if !Equal(p[i], qr[(j+i)%len(qr)]) {
				equal = false
				break
			}
		}
		if equal {
			return true
		}
		j += cmdLen(q[j])
	}
	return false
}

// Closed returns true if the last subpath of p is a closed path.
func (p Path) Closed() bool {
	return 0 < len(p) && p[len(p)-1] == Close
}

// PointClosed returns true if the last subpath of p is a closed path
// and the close command is a point and not a line.
func (p Path) PointClosed() bool {
	return 6 < len(p) && p[len(p)-1] == Close && Equal(p[len(p)-7], p[len(p)-3]) && Equal(p[len(p)-6], p[len(p)-2])
}

// HasSubpaths returns true when path p has subpaths.
// TODO: naming right? A simple path would not self-intersect.
// Add IsXMonotone and IsFlat as well?
func (p Path) HasSubpaths() bool {
	for i := 0; i < len(p); {
		if p[i] == MoveTo && i != 0 {
			return true
		}
		i += cmdLen(p[i])
	}
	return false
}

// Clone returns a copy of p.
func (p Path) Clone() Path {
	return slices.Clone(p)
}

// CopyTo returns a copy of p, using the memory of path q.
func (p *Path) CopyTo(q *Path) *Path {
	if q == nil || len(q) < len(p) {
		q = make([]float32, len(p))
	} else {
		q = q[:len(p)]
	}
	copy(q, p)
	return q
}

// Len returns the number of segments.
func (p Path) Len() int {
	n := 0
	for i := 0; i < len(p); {
		i += cmdLen(p[i])
		n++
	}
	return n
}

// Append appends path q to p and returns the extended path p.
func (p *Path) Append(qs ...Path) Path {
	if p.Empty() {
		p = &Path{}
	}
	for _, q := range qs {
		if !q.Empty() {
			p = append(p, q...)
		}
	}
	return *p
}

// Join joins path q to p and returns the extended path p
// (or q if p is empty). It's like executing the commands
// in q to p in sequence, where if the first MoveTo of q
//
//	doesn't coincide with p, or if p ends in Close,
//
// it will fallback to appending the paths.
func (p Path) Join(q Path) Path {
	if q.Empty() {
		return p
	} else if p.Empty() {
		return q
	}

	if p[len(p)-1] == Close || !Equal(p[len(p)-3], q[1]) || !Equal(p[len(p)-2], q[2]) {
		return &Path{append(p, q...)}
	}

	d := q[cmdLen(MoveTo):]

	// add the first command through the command functions to use the optimization features
	// q is not empty, so starts with a MoveTo followed by other commands
	cmd := d[0]
	switch cmd {
	case MoveTo:
		p.MoveTo(d[1], d[2])
	case LineTo:
		p.LineTo(d[1], d[2])
	case QuadTo:
		p.QuadTo(d[1], d[2], d[3], d[4])
	case CubeTo:
		p.CubeTo(d[1], d[2], d[3], d[4], d[5], d[6])
	case ArcTo:
		large, sweep := toArcFlags(d[4])
		p.ArcTo(d[1], d[2], d[3]*180.0/math.Pi, large, sweep, d[5], d[6])
	case Close:
		p.Close()
	}

	i := len(p)
	end := p.StartPos()
	p = &Path{append(p, d[cmdLen(cmd):]...)}

	// repair close commands
	for i < len(p) {
		cmd := p[i]
		if cmd == MoveTo {
			break
		} else if cmd == Close {
			p[i+1] = end.X
			p[i+2] = end.Y
			break
		}
		i += cmdLen(cmd)
	}
	return p

}

// Pos returns the current position of the path,
// which is the end point of the last command.
func (p Path) Pos() math32.Vector2 {
	if 0 < len(p) {
		return math32.Vector2{p[len(p)-3], p[len(p)-2]}
	}
	return math32.Vector2{}
}

// StartPos returns the start point of the current subpath,
// i.e. it returns the position of the last MoveTo command.
func (p Path) StartPos() math32.Vector2 {
	for i := len(p); 0 < i; {
		cmd := p[i-1]
		if cmd == MoveTo {
			return math32.Vector2{p[i-3], p[i-2]}
		}
		i -= cmdLen(cmd)
	}
	return math32.Vector2{}
}

// Coords returns all the coordinates of the segment
// start/end points. It omits zero-length Closes.
func (p Path) Coords() []math32.Vector2 {
	coords := []math32.Vector2{}
	for i := 0; i < len(p); {
		cmd := p[i]
		i += cmdLen(cmd)
		if len(coords) == 0 || cmd != Close || !coords[len(coords)-1].Equals(math32.Vector2{p[i-3], p[i-2]}) {
			coords = append(coords, math32.Vector2{p[i-3], p[i-2]})
		}
	}
	return coords
}

/////// Accessors

// EndPoint returns the end point for MoveTo, LineTo, and Close commands,
// where the command is at index i.
func (p Path) EndPoint(i int) math32.Vector2 {
	return math32.Vector2{float32(p[i+1]), float32(p[i+2])}
}

// QuadToPoints returns the control point and end for QuadTo command,
// where the command is at index i.
func (p Path) QuadToPoints(i int) (cp, end math32.Vector2) {
	return math32.Vector2{float32(p[i+1]), float32(p[i+2])}, math32.Vector2{float32(p[i+3]), float32(p[i+4])}
}

// CubeToPoints returns the cp1, cp2, and end for CubeTo command,
// where the command is at index i.
func (p Path) CubeToPoints(i int) (cp1, cp2, end math32.Vector2) {
	return math32.Vector2{float32(p[i+1]), float32(p[i+2])}, math32.Vector2{float32(p[i+3]), float32(p[i+4])}, math32.Vector2{float32(p[i+5]), float32(p[i+6])}
}

// ArcToPoints returns the rx, ry, phi, large, sweep values for ArcTo command,
// where the command is at index i.
func (p Path) ArcToPoints(i int) (rx, ry, phi float32, large, sweep bool, end math32.Vector2) {
	rx = float32(p[i+1])
	ry = float32(p[i+2])
	phi = float32(p[i+3])
	large, sweep = toArcFlags(p[i+4])
	end = math32.Vector2{float32(p[i+5]), float32(p[i+6])}
	return
}

/////// Constructors

// MoveTo moves the path to (x,y) without connecting the path. It starts a new independent subpath. Multiple subpaths can be useful when negating parts of a previous path by overlapping it with a path in the opposite direction. The behaviour for overlapping paths depends on the FillRule.
func (p *Path) MoveTo(x, y float32) {
	if 0 < len(p) && p[len(p)-1] == MoveTo {
		p[len(p)-3] = x
		p[len(p)-2] = y
		return
	}
	*p = append(*p, MoveTo, x, y, MoveTo)
}

// LineTo adds a linear path to (x,y).
func (p *Path) LineTo(x, y float32) {
	start := p.Pos()
	end := math32.Vector2{x, y}
	if start.Equals(end) {
		return
	} else if cmdLen(LineTo) <= len(p) && p[len(p)-1] == LineTo {
		prevStart := math32.Vector2{}
		if cmdLen(LineTo) < len(p) {
			prevStart = math32.Vector2{p[len(p)-cmdLen(LineTo)-3], p[len(p)-cmdLen(LineTo)-2]}
		}

		// divide by length^2 since otherwise the perpdot between very small segments may be
		// below Epsilon
		da := start.Sub(prevStart)
		db := end.Sub(start)
		div := da.PerpDot(db)
		if length := da.Length() * db.Length(); Equal(div/length, 0.0) {
			// lines are parallel
			extends := false
			if da.Y < da.X {
				extends = math.Signbit(da.X) == math.Signbit(db.X)
			} else {
				extends = math.Signbit(da.Y) == math.Signbit(db.Y)
			}
			if extends {
				//if Equal(end.Sub(start).AngleBetween(start.Sub(prevStart)), 0.0) {
				p[len(p)-3] = x
				p[len(p)-2] = y
				return
			}
		}
	}

	if len(p) == 0 {
		p.MoveTo(0.0, 0.0)
	} else if p[len(p)-1] == Close {
		p.MoveTo(p[len(p)-3], p[len(p)-2])
	}
	*p = *append(p, LineTo, end.X, end.Y, LineTo)
}

// QuadTo adds a quadratic Bézier path with control point (cpx,cpy) and end point (x,y).
func (p *Path) QuadTo(cpx, cpy, x, y float32) {
	start := p.Pos()
	cp := math32.Vector2{cpx, cpy}
	end := math32.Vector2{x, y}
	if start.Equals(end) && start.Equals(cp) {
		return
	} else if !start.Equals(end) && (start.Equals(cp) || angleEqual(end.Sub(start).AngleBetween(cp.Sub(start)), 0.0)) && (end.Equals(cp) || angleEqual(end.Sub(start).AngleBetween(end.Sub(cp)), 0.0)) {
		p.LineTo(end.X, end.Y)
		return
	}

	if len(p) == 0 {
		p.MoveTo(0.0, 0.0)
	} else if p[len(p)-1] == Close {
		p.MoveTo(p[len(p)-3], p[len(p)-2])
	}
	p = append(p, QuadTo, cp.X, cp.Y, end.X, end.Y, QuadTo)
}

// CubeTo adds a cubic Bézier path with control points (cpx1,cpy1) and (cpx2,cpy2) and end point (x,y).
func (p *Path) CubeTo(cpx1, cpy1, cpx2, cpy2, x, y float32) {
	start := p.Pos()
	cp1 := math32.Vector2{cpx1, cpy1}
	cp2 := math32.Vector2{cpx2, cpy2}
	end := math32.Vector2{x, y}
	if start.Equals(end) && start.Equals(cp1) && start.Equals(cp2) {
		return
	} else if !start.Equals(end) && (start.Equals(cp1) || end.Equals(cp1) || angleEqual(end.Sub(start).AngleBetween(cp1.Sub(start)), 0.0) && angleEqual(end.Sub(start).AngleBetween(end.Sub(cp1)), 0.0)) && (start.Equals(cp2) || end.Equals(cp2) || angleEqual(end.Sub(start).AngleBetween(cp2.Sub(start)), 0.0) && angleEqual(end.Sub(start).AngleBetween(end.Sub(cp2)), 0.0)) {
		p.LineTo(end.X, end.Y)
		return
	}

	if len(p) == 0 {
		p.MoveTo(0.0, 0.0)
	} else if p[len(p)-1] == Close {
		p.MoveTo(p[len(p)-3], p[len(p)-2])
	}
	p = append(p, CubeTo, cp1.X, cp1.Y, cp2.X, cp2.Y, end.X, end.Y, CubeTo)
}

// ArcTo adds an arc with radii rx and ry, with rot the counter clockwise rotation with respect to the coordinate system in degrees, large and sweep booleans (see https://developer.mozilla.org/en-US/docs/Web/SVG/Tutorial/Paths#Arcs), and (x,y) the end position of the pen. The start position of the pen was given by a previous command's end point.
func (p *Path) ArcTo(rx, ry, rot float32, large, sweep bool, x, y float32) {
	start := p.Pos()
	end := math32.Vector2{x, y}
	if start.Equals(end) {
		return
	}
	if Equal(rx, 0.0) || math.IsInf(rx, 0) || Equal(ry, 0.0) || math.IsInf(ry, 0) {
		p.LineTo(end.X, end.Y)
		return
	}

	rx = math.Abs(rx)
	ry = math.Abs(ry)
	if Equal(rx, ry) {
		rot = 0.0 // circle
	} else if rx < ry {
		rx, ry = ry, rx
		rot += 90.0
	}

	phi := angleNorm(rot * math.Pi / 180.0)
	if math.Pi <= phi { // phi is canonical within 0 <= phi < 180
		phi -= math.Pi
	}

	// scale ellipse if rx and ry are too small
	lambda := ellipseRadiiCorrection(start, rx, ry, phi, end)
	if lambda > 1.0 {
		rx *= lambda
		ry *= lambda
	}

	if len(p) == 0 {
		p.MoveTo(0.0, 0.0)
	} else if p[len(p)-1] == Close {
		p.MoveTo(p[len(p)-3], p[len(p)-2])
	}
	p = append(p, ArcTo, rx, ry, phi, fromArcFlags(large, sweep), end.X, end.Y, ArcTo)
}

// Arc adds an elliptical arc with radii rx and ry, with rot the counter clockwise rotation in degrees, and theta0 and theta1 the angles in degrees of the ellipse (before rot is applies) between which the arc will run. If theta0 < theta1, the arc will run in a CCW direction. If the difference between theta0 and theta1 is bigger than 360 degrees, one full circle will be drawn and the remaining part of diff % 360, e.g. a difference of 810 degrees will draw one full circle and an arc over 90 degrees.
func (p *Path) Arc(rx, ry, rot, theta0, theta1 float32) {
	phi := rot * math.Pi / 180.0
	theta0 *= math.Pi / 180.0
	theta1 *= math.Pi / 180.0
	dtheta := math.Abs(theta1 - theta0)

	sweep := theta0 < theta1
	large := math.Mod(dtheta, 2.0*math.Pi) > math.Pi
	p0 := EllipsePos(rx, ry, phi, 0.0, 0.0, theta0)
	p1 := EllipsePos(rx, ry, phi, 0.0, 0.0, theta1)

	start := p.Pos()
	center := start.Sub(p0)
	if dtheta >= 2.0*math.Pi {
		startOpposite := center.Sub(p0)
		p.ArcTo(rx, ry, rot, large, sweep, startOpposite.X, startOpposite.Y)
		p.ArcTo(rx, ry, rot, large, sweep, start.X, start.Y)
		if Equal(math.Mod(dtheta, 2.0*math.Pi), 0.0) {
			return
		}
	}
	end := center.Add(p1)
	p.ArcTo(rx, ry, rot, large, sweep, end.X, end.Y)
}

// Close closes a (sub)path with a LineTo to the start of the path (the most recent MoveTo command). It also signals the path closes as opposed to being just a LineTo command, which can be significant for stroking purposes for example.
func (p *Path) Close() {
	if len(p) == 0 || p[len(p)-1] == Close {
		// already closed or empty
		return
	} else if p[len(p)-1] == MoveTo {
		// remove MoveTo + Close
		p = p[:len(p)-cmdLen(MoveTo)]
		return
	}

	end := p.StartPos()
	if p[len(p)-1] == LineTo && Equal(p[len(p)-3], end.X) && Equal(p[len(p)-2], end.Y) {
		// replace LineTo by Close if equal
		p[len(p)-1] = Close
		p[len(p)-cmdLen(LineTo)] = Close
		return
	} else if p[len(p)-1] == LineTo {
		// replace LineTo by Close if equidirectional extension
		start := math32.Vector2{p[len(p)-3], p[len(p)-2]}
		prevStart := math32.Vector2{}
		if cmdLen(LineTo) < len(p) {
			prevStart = math32.Vector2{p[len(p)-cmdLen(LineTo)-3], p[len(p)-cmdLen(LineTo)-2]}
		}
		if Equal(end.Sub(start).AngleBetween(start.Sub(prevStart)), 0.0) {
			p[len(p)-cmdLen(LineTo)] = Close
			p[len(p)-3] = end.X
			p[len(p)-2] = end.Y
			p[len(p)-1] = Close
			return
		}
	}
	p = append(p, Close, end.X, end.Y, Close)
}

// optimizeClose removes a superfluous first line segment in-place of a subpath. If both the first and last segment are line segments and are colinear, move the start of the path forward one segment
func (p *Path) optimizeClose() {
	if len(p) == 0 || p[len(p)-1] != Close {
		return
	}

	// find last MoveTo
	end := math32.Vector2{}
	iMoveTo := len(p)
	for 0 < iMoveTo {
		cmd := p[iMoveTo-1]
		iMoveTo -= cmdLen(cmd)
		if cmd == MoveTo {
			end = math32.Vector2{p[iMoveTo+1], p[iMoveTo+2]}
			break
		}
	}

	if p[iMoveTo] == MoveTo && p[iMoveTo+cmdLen(MoveTo)] == LineTo && iMoveTo+cmdLen(MoveTo)+cmdLen(LineTo) < len(p)-cmdLen(Close) {
		// replace Close + MoveTo + LineTo by Close + MoveTo if equidirectional
		// move Close and MoveTo forward along the path
		start := math32.Vector2{p[len(p)-cmdLen(Close)-3], p[len(p)-cmdLen(Close)-2]}
		nextEnd := math32.Vector2{p[iMoveTo+cmdLen(MoveTo)+cmdLen(LineTo)-3], p[iMoveTo+cmdLen(MoveTo)+cmdLen(LineTo)-2]}
		if Equal(end.Sub(start).AngleBetween(nextEnd.Sub(end)), 0.0) {
			// update Close
			p[len(p)-3] = nextEnd.X
			p[len(p)-2] = nextEnd.Y

			// update MoveTo
			p[iMoveTo+1] = nextEnd.X
			p[iMoveTo+2] = nextEnd.Y

			// remove LineTo
			p = append(p[:iMoveTo+cmdLen(MoveTo)], p[iMoveTo+cmdLen(MoveTo)+cmdLen(LineTo):]...)
		}
	}
}

////////////////////////////////////////////////////////////////

func (p *Path) simplifyToCoords() []math32.Vector2 {
	coords := p.Coords()
	if len(coords) <= 3 {
		// if there are just two commands, linearizing them gives us an area of no surface. To avoid this we add extra coordinates halfway for QuadTo, CubeTo and ArcTo.
		coords = []math32.Vector2{}
		for i := 0; i < len(p); {
			cmd := p[i]
			if cmd == QuadTo {
				p0 := math32.Vector2{p[i-3], p[i-2]}
				p1 := math32.Vector2{p[i+1], p[i+2]}
				p2 := math32.Vector2{p[i+3], p[i+4]}
				_, _, _, coord, _, _ := quadraticBezierSplit(p0, p1, p2, 0.5)
				coords = append(coords, coord)
			} else if cmd == CubeTo {
				p0 := math32.Vector2{p[i-3], p[i-2]}
				p1 := math32.Vector2{p[i+1], p[i+2]}
				p2 := math32.Vector2{p[i+3], p[i+4]}
				p3 := math32.Vector2{p[i+5], p[i+6]}
				_, _, _, _, coord, _, _, _ := cubicBezierSplit(p0, p1, p2, p3, 0.5)
				coords = append(coords, coord)
			} else if cmd == ArcTo {
				rx, ry, phi := p[i+1], p[i+2], p[i+3]
				large, sweep := toArcFlags(p[i+4])
				cx, cy, theta0, theta1 := ellipseToCenter(p[i-3], p[i-2], rx, ry, phi, large, sweep, p[i+5], p[i+6])
				coord, _, _, _ := ellipseSplit(rx, ry, phi, cx, cy, theta0, theta1, (theta0+theta1)/2.0)
				coords = append(coords, coord)
			}
			i += cmdLen(cmd)
			if cmd != Close || !Equal(coords[len(coords)-1].X, p[i-3]) || !Equal(coords[len(coords)-1].Y, p[i-2]) {
				coords = append(coords, math32.Vector2{p[i-3], p[i-2]})
			}
		}
	}
	return coords
}

// direction returns the direction of the path at the given index into Path and t in [0.0,1.0]. Path must not contain subpaths, and will return the path's starting direction when i points to a MoveTo, or the path's final direction when i points to a Close of zero-length.
func (p Path) direction(i int, t float32) math32.Vector2 {
	last := len(p)
	if p[last-1] == Close && (math32.Vector2{p[last-cmdLen(Close)-3], p[last-cmdLen(Close)-2]}).Equals(math32.Vector2{p[last-3], p[last-2]}) {
		// point-closed
		last -= cmdLen(Close)
	}

	if i == 0 {
		// get path's starting direction when i points to MoveTo
		i = 4
		t = 0.0
	} else if i < len(p) && i == last {
		// get path's final direction when i points to zero-length Close
		i -= cmdLen(p[i-1])
		t = 1.0
	}
	if i < 0 || len(p) <= i || last < i+cmdLen(p[i]) {
		return math32.Vector2{}
	}

	cmd := p[i]
	var start math32.Vector2
	if i == 0 {
		start = math32.Vector2{p[last-3], p[last-2]}
	} else {
		start = math32.Vector2{p[i-3], p[i-2]}
	}

	i += cmdLen(cmd)
	end := math32.Vector2{p[i-3], p[i-2]}
	switch cmd {
	case LineTo, Close:
		return end.Sub(start).Norm(1.0)
	case QuadTo:
		cp := math32.Vector2{p[i-5], p[i-4]}
		return quadraticBezierDeriv(start, cp, end, t).Norm(1.0)
	case CubeTo:
		cp1 := math32.Vector2{p[i-7], p[i-6]}
		cp2 := math32.Vector2{p[i-5], p[i-4]}
		return cubicBezierDeriv(start, cp1, cp2, end, t).Norm(1.0)
	case ArcTo:
		rx, ry, phi := p[i-7], p[i-6], p[i-5]
		large, sweep := toArcFlags(p[i-4])
		_, _, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
		theta := theta0 + t*(theta1-theta0)
		return ellipseDeriv(rx, ry, phi, sweep, theta).Norm(1.0)
	}
	return math32.Vector2{}
}

// Direction returns the direction of the path at the given segment and t in [0.0,1.0] along that path. The direction is a vector of unit length.
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
				pi := &Path{p[iStart:iEnd]}
				return piirection(iSeg-iStart, t)
			}
			iStart = i
		}
		if seg == curSeg {
			iSeg = i
		}
		i += cmdLen(cmd)
	}
	return math32.Vector2{} // if segment doesn't exist
}

// CoordDirections returns the direction of the segment start/end points. It will return the average direction at the intersection of two end points, and for an open path it will simply return the direction of the start and end points of the path.
func (p Path) CoordDirections() []math32.Vector2 {
	if len(p) <= 4 {
		return []math32.Vector2{{}}
	}
	last := len(p)
	if p[last-1] == Close && (math32.Vector2{p[last-cmdLen(Close)-3], p[last-cmdLen(Close)-2]}).Equals(math32.Vector2{p[last-3], p[last-2]}) {
		// point-closed
		last -= cmdLen(Close)
	}

	dirs := []math32.Vector2{}
	var closed bool
	var dirPrev math32.Vector2
	for i := 4; i < last; {
		cmd := p[i]
		dir := pirection(i, 0.0)
		if i == 0 {
			dirs = append(dirs, dir)
		} else {
			dirs = append(dirs, dirPrev.Add(dir).Norm(1.0))
		}
		dirPrev = pirection(i, 1.0)
		closed = cmd == Close
		i += cmdLen(cmd)
	}
	if closed {
		dirs[0] = dirs[0].Add(dirPrev).Norm(1.0)
		dirs = append(dirs, dirs[0])
	} else {
		dirs = append(dirs, dirPrev)
	}
	return dirs
}

// curvature returns the curvature of the path at the given index into Path and t in [0.0,1.0]. Path must not contain subpaths, and will return the path's starting curvature when i points to a MoveTo, or the path's final curvature when i points to a Close of zero-length.
func (p Path) curvature(i int, t float32) float32 {
	last := len(p)
	if p[last-1] == Close && (math32.Vector2{p[last-cmdLen(Close)-3], p[last-cmdLen(Close)-2]}).Equals(math32.Vector2{p[last-3], p[last-2]}) {
		// point-closed
		last -= cmdLen(Close)
	}

	if i == 0 {
		// get path's starting direction when i points to MoveTo
		i = 4
		t = 0.0
	} else if i < len(p) && i == last {
		// get path's final direction when i points to zero-length Close
		i -= cmdLen(p[i-1])
		t = 1.0
	}
	if i < 0 || len(p) <= i || last < i+cmdLen(p[i]) {
		return 0.0
	}

	cmd := p[i]
	var start math32.Vector2
	if i == 0 {
		start = math32.Vector2{p[last-3], p[last-2]}
	} else {
		start = math32.Vector2{p[i-3], p[i-2]}
	}

	i += cmdLen(cmd)
	end := math32.Vector2{p[i-3], p[i-2]}
	switch cmd {
	case LineTo, Close:
		return 0.0
	case QuadTo:
		cp := math32.Vector2{p[i-5], p[i-4]}
		return 1.0 / quadraticBezierCurvatureRadius(start, cp, end, t)
	case CubeTo:
		cp1 := math32.Vector2{p[i-7], p[i-6]}
		cp2 := math32.Vector2{p[i-5], p[i-4]}
		return 1.0 / cubicBezierCurvatureRadius(start, cp1, cp2, end, t)
	case ArcTo:
		rx, ry, phi := p[i-7], p[i-6], p[i-5]
		large, sweep := toArcFlags(p[i-4])
		_, _, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
		theta := theta0 + t*(theta1-theta0)
		return 1.0 / ellipseCurvatureRadius(rx, ry, sweep, theta)
	}
	return 0.0
}

// Curvature returns the curvature of the path at the given segment and t in [0.0,1.0] along that path. It is zero for straight lines and for non-existing segments.
func (p Path) Curvature(seg int, t float32) float32 {
	if len(p) <= 4 {
		return 0.0
	}

	curSeg := 0
	iStart, iSeg, iEnd := 0, 0, 0
	for i := 0; i < len(p); {
		cmd := p[i]
		if cmd == MoveTo {
			if seg < curSeg {
				pi := &Path{p[iStart:iEnd]}
				return pi.curvature(iSeg-iStart, t)
			}
			iStart = i
		}
		if seg == curSeg {
			iSeg = i
		}
		i += cmdLen(cmd)
	}
	return 0.0 // if segment doesn't exist
}

// windings counts intersections of ray with path. Paths that cross downwards are negative and upwards are positive. It returns the windings excluding the start position and the windings of the start position itself. If the windings of the start position is not zero, the start position is on a boundary.
func windings(zs []Intersection) (int, bool) {
	// There are four particular situations to be aware of. Whenever the path is horizontal it
	// will be parallel to the ray, and usually overlapping. Either we have:
	// - a starting point to the left of the overlapping section: ignore the overlapping
	//   intersections so that it appears as a regular intersection, albeit at the endpoints
	//   of two segments, which may either cancel out to zero (top or bottom edge) or add up to
	//   1 or -1 if the path goes upwards or downwards respectively before/after the overlap.
	// - a starting point on the left-hand corner of an overlapping section: ignore if either
	//   intersection of an endpoint pair (t=0,t=1) is overlapping, but count for nb upon
	//   leaving the overlap.
	// - a starting point in the middle of an overlapping section: same as above
	// - a starting point on the right-hand corner of an overlapping section: intersections are
	//   tangent and thus already ignored for n, but for nb we should ignore the intersection with
	//   a 0/180 degree direction, and count the other

	n := 0
	boundary := false
	for i := 0; i < len(zs); i++ {
		z := zs[i]
		if z.T[0] == 0.0 {
			boundary = true
			continue
		}

		d := 1
		if z.Into() {
			d = -1 // downwards
		}
		if z.T[1] != 0.0 && z.T[1] != 1.0 {
			if !z.Same {
				n += d
			}
		} else {
			same := z.Same || zs[i+1].Same
			if !same {
				if z.Into() == zs[i+1].Into() {
					n += d
				}
			}
			i++
		}
	}
	return n, boundary
}

// Windings returns the number of windings at the given point, i.e. the sum of windings for each time a ray from (x,y) towards (∞,y) intersects the path. Counter clock-wise intersections count as positive, while clock-wise intersections count as negative. Additionally, it returns whether the point is on a path's boundary (which counts as being on the exterior).
func (p Path) Windings(x, y float32) (int, bool) {
	n := 0
	boundary := false
	for _, pi := range p.Split() {
		zs := pi.RayIntersections(x, y)
		if ni, boundaryi := windings(zs); boundaryi {
			boundary = true
		} else {
			n += ni
		}
	}
	return n, boundary
}

// Crossings returns the number of crossings with the path from the given point outwards, i.e. the number of times a ray from (x,y) towards (∞,y) intersects the path. Additionally, it returns whether the point is on a path's boundary (which does not count towards the number of crossings).
func (p Path) Crossings(x, y float32) (int, bool) {
	n := 0
	boundary := false
	for _, pi := range p.Split() {
		// Count intersections of ray with path. Count half an intersection on boundaries.
		ni := 0.0
		for _, z := range pi.RayIntersections(x, y) {
			if z.T[0] == 0.0 {
				boundary = true
			} else if !z.Same {
				if z.T[1] == 0.0 || z.T[1] == 1.0 {
					ni += 0.5
				} else {
					ni += 1.0
				}
			} else if z.T[1] == 0.0 || z.T[1] == 1.0 {
				ni -= 0.5
			}
		}
		n += int(ni)
	}
	return n, boundary
}

// Contains returns whether the point (x,y) is contained/filled by the path. This depends on the
// FillRule. It uses a ray from (x,y) toward (∞,y) and counts the number of intersections with
// the path. When the point is on the boundary it is considered to be on the path's exterior.
func (p Path) Contains(x, y float32, fillRule FillRule) bool {
	n, boundary := p.Windings(x, y)
	if boundary {
		return true
	}
	return fillRule.Fills(n)
}

// CCW returns true when the path is counter clockwise oriented at its bottom-right-most
// coordinate. It is most useful when knowing that the path does not self-intersect as it will
// tell you if the entire path is CCW or not. It will only return the result for the first subpath.
// It will return true for an empty path or a straight line. It may not return a valid value when
// the right-most point happens to be a (self-)overlapping segment.
func (p Path) CCW() bool {
	if len(p) <= 4 || (p[4] == LineTo || p[4] == Close) && len(p) <= 4+cmdLen(p[4]) {
		// empty path or single straight segment
		return true
	}

	p = p.XMonotone()

	// pick bottom-right-most coordinate of subpath, as we know its left-hand side is filling
	k, kMax := 4, len(p)
	if p[kMax-1] == Close {
		kMax -= cmdLen(Close)
	}
	for i := 4; i < len(p); {
		cmd := p[i]
		if cmd == MoveTo {
			// only handle first subpath
			kMax = i
			break
		}
		i += cmdLen(cmd)
		if x, y := p[i-3], p[i-2]; p[k-3] < x || Equal(p[k-3], x) && y < p[k-2] {
			k = i
		}
	}

	// get coordinates of previous and next segments
	var kPrev int
	if k == 4 {
		kPrev = kMax
	} else {
		kPrev = k - cmdLen(p[k-1])
	}

	var angleNext float32
	anglePrev := angleNorm(pirection(kPrev, 1.0).Angle() + math.Pi)
	if k == kMax {
		// use implicit close command
		angleNext = math32.Vector2{p[1], p[2]}.Sub(math32.Vector2{p[k-3], p[k-2]}).Angle()
	} else {
		angleNext = pirection(k, 0.0).Angle()
	}
	if Equal(anglePrev, angleNext) {
		// segments have the same direction at their right-most point
		// one or both are not straight lines, check if curvature is different
		var curvNext float32
		curvPrev := -p.curvature(kPrev, 1.0)
		if k == kMax {
			// use implicit close command
			curvNext = 0.0
		} else {
			curvNext = p.curvature(k, 0.0)
		}
		if !Equal(curvPrev, curvNext) {
			// ccw if curvNext is smaller than curvPrev
			return curvNext < curvPrev
		}
	}
	return (angleNext - anglePrev) < 0.0
}

// Filling returns whether each subpath gets filled or not. Whether a path is filled depends on
// the FillRule and whether it negates another path. If a subpath is not closed, it is implicitly
// assumed to be closed.
func (p Path) Filling(fillRule FillRule) []bool {
	ps := p.Split()
	filling := make([]bool, len(ps))
	for i, pi := range ps {
		// get current subpath's winding
		n := 0
		if pi.CCW() {
			n++
		} else {
			n--
		}

		// sum windings from other subpaths
		pos := math32.Vector2{pi[1], pi[2]}
		for j, pj := range ps {
			if i == j {
				continue
			}
			zs := pj.RayIntersections(pos.X, pos.Y)
			if ni, boundaryi := windings(zs); !boundaryi {
				n += ni
			} else {
				// on the boundary, check if around the interior or exterior of pos
			}
		}
		filling[i] = fillRule.Fills(n)
	}
	return filling
}

// FastBounds returns the maximum bounding box rectangle of the path. It is quicker than Bounds.
func (p Path) FastBounds() math32.Box2 {
	if len(p) < 4 {
		return math32.Box2{}
	}

	// first command is MoveTo
	start, end := math32.Vector2{p[1], p[2]}, math32.Vector2{}
	xmin, xmax := start.X, start.X
	ymin, ymax := start.Y, start.Y
	for i := 4; i < len(p); {
		cmd := p[i]
		switch cmd {
		case MoveTo, LineTo, Close:
			end = math32.Vector2{p[i+1], p[i+2]}
			xmin = math32.Min(xmin, end.X)
			xmax = math32.Max(xmax, end.X)
			ymin = math32.Min(ymin, end.Y)
			ymax = math32.Max(ymax, end.Y)
		case QuadTo:
			cp := math32.Vector2{p[i+1], p[i+2]}
			end = math32.Vector2{p[i+3], p[i+4]}
			xmin = math32.Min(xmin, math32.Min(cp.X, end.X))
			xmax = math32.Max(xmax, math32.Max(cp.X, end.X))
			ymin = math32.Min(ymin, math32.Min(cp.Y, end.Y))
			ymax = math32.Max(ymax, math32.Max(cp.Y, end.Y))
		case CubeTo:
			cp1 := math32.Vector2{p[i+1], p[i+2]}
			cp2 := math32.Vector2{p[i+3], p[i+4]}
			end = math32.Vector2{p[i+5], p[i+6]}
			xmin = math32.Min(xmin, math32.Min(cp1.X, math32.Min(cp2.X, end.X)))
			xmax = math32.Max(xmax, math32.Max(cp1.X, math32.Min(cp2.X, end.X)))
			ymin = math32.Min(ymin, math32.Min(cp1.Y, math32.Min(cp2.Y, end.Y)))
			ymax = math32.Max(ymax, math32.Max(cp1.Y, math32.Min(cp2.Y, end.Y)))
		case ArcTo:
			rx, ry, phi := p[i+1], p[i+2], p[i+3]
			large, sweep := toArcFlags(p[i+4])
			end = math32.Vector2{p[i+5], p[i+6]}
			cx, cy, _, _ := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
			r := math32.Max(rx, ry)
			xmin = math32.Min(xmin, cx-r)
			xmax = math32.Max(xmax, cx+r)
			ymin = math32.Min(ymin, cy-r)
			ymax = math32.Max(ymax, cy+r)

		}
		i += cmdLen(cmd)
		start = end
	}
	return math32.Box2{xmin, ymin, xmax, ymax}
}

// Bounds returns the exact bounding box rectangle of the path.
func (p Path) Bounds() math32.Box2 {
	if len(p) < 4 {
		return math32.Box2{}
	}

	// first command is MoveTo
	start, end := math32.Vector2{p[1], p[2]}, math32.Vector2{}
	xmin, xmax := start.X, start.X
	ymin, ymax := start.Y, start.Y
	for i := 4; i < len(p); {
		cmd := p[i]
		switch cmd {
		case MoveTo, LineTo, Close:
			end = math32.Vector2{p[i+1], p[i+2]}
			xmin = math32.Min(xmin, end.X)
			xmax = math32.Max(xmax, end.X)
			ymin = math32.Min(ymin, end.Y)
			ymax = math32.Max(ymax, end.Y)
		case QuadTo:
			cp := math32.Vector2{p[i+1], p[i+2]}
			end = math32.Vector2{p[i+3], p[i+4]}

			xmin = math32.Min(xmin, end.X)
			xmax = math32.Max(xmax, end.X)
			if tdenom := (start.X - 2*cp.X + end.X); !Equal(tdenom, 0.0) {
				if t := (start.X - cp.X) / tdenom; IntervalExclusive(t, 0.0, 1.0) {
					x := quadraticBezierPos(start, cp, end, t)
					xmin = math32.Min(xmin, x.X)
					xmax = math32.Max(xmax, x.X)
				}
			}

			ymin = math32.Min(ymin, end.Y)
			ymax = math32.Max(ymax, end.Y)
			if tdenom := (start.Y - 2*cp.Y + end.Y); !Equal(tdenom, 0.0) {
				if t := (start.Y - cp.Y) / tdenom; IntervalExclusive(t, 0.0, 1.0) {
					y := quadraticBezierPos(start, cp, end, t)
					ymin = math32.Min(ymin, y.Y)
					ymax = math32.Max(ymax, y.Y)
				}
			}
		case CubeTo:
			cp1 := math32.Vector2{p[i+1], p[i+2]}
			cp2 := math32.Vector2{p[i+3], p[i+4]}
			end = math32.Vector2{p[i+5], p[i+6]}

			a := -start.X + 3*cp1.X - 3*cp2.X + end.X
			b := 2*start.X - 4*cp1.X + 2*cp2.X
			c := -start.X + cp1.X
			t1, t2 := solveQuadraticFormula(a, b, c)

			xmin = math32.Min(xmin, end.X)
			xmax = math32.Max(xmax, end.X)
			if !math.IsNaN(t1) && IntervalExclusive(t1, 0.0, 1.0) {
				x1 := cubicBezierPos(start, cp1, cp2, end, t1)
				xmin = math32.Min(xmin, x1.X)
				xmax = math32.Max(xmax, x1.X)
			}
			if !math.IsNaN(t2) && IntervalExclusive(t2, 0.0, 1.0) {
				x2 := cubicBezierPos(start, cp1, cp2, end, t2)
				xmin = math32.Min(xmin, x2.X)
				xmax = math32.Max(xmax, x2.X)
			}

			a = -start.Y + 3*cp1.Y - 3*cp2.Y + end.Y
			b = 2*start.Y - 4*cp1.Y + 2*cp2.Y
			c = -start.Y + cp1.Y
			t1, t2 = solveQuadraticFormula(a, b, c)

			ymin = math32.Min(ymin, end.Y)
			ymax = math32.Max(ymax, end.Y)
			if !math.IsNaN(t1) && IntervalExclusive(t1, 0.0, 1.0) {
				y1 := cubicBezierPos(start, cp1, cp2, end, t1)
				ymin = math32.Min(ymin, y1.Y)
				ymax = math32.Max(ymax, y1.Y)
			}
			if !math.IsNaN(t2) && IntervalExclusive(t2, 0.0, 1.0) {
				y2 := cubicBezierPos(start, cp1, cp2, end, t2)
				ymin = math32.Min(ymin, y2.Y)
				ymax = math32.Max(ymax, y2.Y)
			}
		case ArcTo:
			rx, ry, phi := p[i+1], p[i+2], p[i+3]
			large, sweep := toArcFlags(p[i+4])
			end = math32.Vector2{p[i+5], p[i+6]}
			cx, cy, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)

			// find the four extremes (top, bottom, left, right) and apply those who are between theta1 and theta2
			// x(theta) = cx + rx*cos(theta)*cos(phi) - ry*sin(theta)*sin(phi)
			// y(theta) = cy + rx*cos(theta)*sin(phi) + ry*sin(theta)*cos(phi)
			// be aware that positive rotation appears clockwise in SVGs (non-Cartesian coordinate system)
			// we can now find the angles of the extremes

			sinphi, cosphi := math.Sincos(phi)
			thetaRight := math.Atan2(-ry*sinphi, rx*cosphi)
			thetaTop := math.Atan2(rx*cosphi, ry*sinphi)
			thetaLeft := thetaRight + math.Pi
			thetaBottom := thetaTop + math.Pi

			dx := math.Sqrt(rx*rx*cosphi*cosphi + ry*ry*sinphi*sinphi)
			dy := math.Sqrt(rx*rx*sinphi*sinphi + ry*ry*cosphi*cosphi)
			if angleBetween(thetaLeft, theta0, theta1) {
				xmin = math32.Min(xmin, cx-dx)
			}
			if angleBetween(thetaRight, theta0, theta1) {
				xmax = math32.Max(xmax, cx+dx)
			}
			if angleBetween(thetaBottom, theta0, theta1) {
				ymin = math32.Min(ymin, cy-dy)
			}
			if angleBetween(thetaTop, theta0, theta1) {
				ymax = math32.Max(ymax, cy+dy)
			}
			xmin = math32.Min(xmin, end.X)
			xmax = math32.Max(xmax, end.X)
			ymin = math32.Min(ymin, end.Y)
			ymax = math32.Max(ymax, end.Y)
		}
		i += cmdLen(cmd)
		start = end
	}
	return math32.Box2{xmin, ymin, xmax, ymax}
}

// Length returns the length of the path in millimeters. The length is approximated for cubic Béziers.
func (p Path) Length() float32 {
	d := 0.0
	var start, end math32.Vector2
	for i := 0; i < len(p); {
		cmd := p[i]
		switch cmd {
		case MoveTo:
			end = math32.Vector2{p[i+1], p[i+2]}
		case LineTo, Close:
			end = math32.Vector2{p[i+1], p[i+2]}
			d += end.Sub(start).Length()
		case QuadTo:
			cp := math32.Vector2{p[i+1], p[i+2]}
			end = math32.Vector2{p[i+3], p[i+4]}
			d += quadraticBezierLength(start, cp, end)
		case CubeTo:
			cp1 := math32.Vector2{p[i+1], p[i+2]}
			cp2 := math32.Vector2{p[i+3], p[i+4]}
			end = math32.Vector2{p[i+5], p[i+6]}
			d += cubicBezierLength(start, cp1, cp2, end)
		case ArcTo:
			rx, ry, phi := p[i+1], p[i+2], p[i+3]
			large, sweep := toArcFlags(p[i+4])
			end = math32.Vector2{p[i+5], p[i+6]}
			_, _, theta1, theta2 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
			d += ellipseLength(rx, ry, theta1, theta2)
		}
		i += cmdLen(cmd)
		start = end
	}
	return d
}

// Transform transforms the path by the given transformation matrix and returns a new path. It modifies the path in-place.
func (p Path) Transform(m math32.Matrix2) Path {
	_, _, _, xscale, yscale, _ := mecompose()
	for i := 0; i < len(p); {
		cmd := p[i]
		switch cmd {
		case MoveTo, LineTo, Close:
			end := mot(math32.Vector2{p[i+1], p[i+2]})
			p[i+1] = end.X
			p[i+2] = end.Y
		case QuadTo:
			cp := mot(math32.Vector2{p[i+1], p[i+2]})
			end := mot(math32.Vector2{p[i+3], p[i+4]})
			p[i+1] = cp.X
			p[i+2] = cp.Y
			p[i+3] = end.X
			p[i+4] = end.Y
		case CubeTo:
			cp1 := mot(math32.Vector2{p[i+1], p[i+2]})
			cp2 := mot(math32.Vector2{p[i+3], p[i+4]})
			end := mot(math32.Vector2{p[i+5], p[i+6]})
			p[i+1] = cp1.X
			p[i+2] = cp1.Y
			p[i+3] = cp2.X
			p[i+4] = cp2.Y
			p[i+5] = end.X
			p[i+6] = end.Y
		case ArcTo:
			rx := p[i+1]
			ry := p[i+2]
			phi := p[i+3]
			large, sweep := toArcFlags(p[i+4])
			end := math32.Vector2{p[i+5], p[i+6]}

			// For ellipses written as the conic section equation in matrix form, we have:
			// [x, y] E [x; y] = 0, with E = [1/rx^2, 0; 0, 1/ry^2]
			// For our transformed ellipse we have [x', y'] = T [x, y], with T the affine
			// transformation matrix so that
			// (T^-1 [x'; y'])^T E (T^-1 [x'; y'] = 0  =>  [x', y'] T^(-T) E T^(-1) [x'; y'] = 0
			// We define Q = T^(-1,T) E T^(-1) the new ellipse equation which is typically rotated
			// from the x-axis. That's why we find the eigenvalues and eigenvectors (the new
			// direction and length of the major and minor axes).
			T := m.Rotate(phi * 180.0 / math.Pi)
			invT := T.Inv()
			Q := Identity.Scale(1.0/rx/rx, 1.0/ry/ry)
			Q = invT.T().Mul(Q).Mul(invT)

			lambda1, lambda2, v1, v2 := Q.Eigen()
			rx = 1 / math.Sqrt(lambda1)
			ry = 1 / math.Sqrt(lambda2)
			phi = v1.Angle()
			if rx < ry {
				rx, ry = ry, rx
				phi = v2.Angle()
			}
			phi = angleNorm(phi)
			if math.Pi <= phi { // phi is canonical within 0 <= phi < 180
				phi -= math.Pi
			}

			if xscale*yscale < 0.0 { // flip x or y axis needs flipping of the sweep
				sweep = !sweep
			}
			end = mot(end)

			p[i+1] = rx
			p[i+2] = ry
			p[i+3] = phi
			p[i+4] = fromArcFlags(large, sweep)
			p[i+5] = end.X
			p[i+6] = end.Y
		}
		i += cmdLen(cmd)
	}
	return p
}

// Translate translates the path by (x,y) and returns a new path.
func (p Path) Translate(x, y float32) Path {
	return p.Transform(Identity.Translate(x, y))
}

// Scale scales the path by (x,y) and returns a new path.
func (p Path) Scale(x, y float32) Path {
	return p.Transform(Identity.Scale(x, y))
}

// Flat returns true if the path consists of solely line segments, that is only MoveTo, LineTo and Close commands.
func (p Path) Flat() bool {
	for i := 0; i < len(p); {
		cmd := p[i]
		if cmd != MoveTo && cmd != LineTo && cmd != Close {
			return false
		}
		i += cmdLen(cmd)
	}
	return true
}

// Flatten flattens all Bézier and arc curves into linear segments and returns a new path. It uses tolerance as the maximum deviation.
func (p *Path) Flatten(tolerance float32) *Path {
	quad := func(p0, p1, p2 math32.Vector2) *Path {
		return flattenQuadraticBezier(p0, p1, p2, tolerance)
	}
	cube := func(p0, p1, p2, p3 math32.Vector2) *Path {
		return flattenCubicBezier(p0, p1, p2, p3, tolerance)
	}
	arc := func(start math32.Vector2, rx, ry, phi float32, large, sweep bool, end math32.Vector2) *Path {
		return flattenEllipticArc(start, rx, ry, phi, large, sweep, end, tolerance)
	}
	return p.replace(nil, quad, cube, arc)
}

// ReplaceArcs replaces ArcTo commands by CubeTo commands and returns a new path.
func (p *Path) ReplaceArcs() *Path {
	return p.replace(nil, nil, nil, arcToCube)
}

// XMonotone replaces all Bézier and arc segments to be x-monotone and returns a new path, that is each path segment is either increasing or decreasing with X while moving across the segment. This is always true for line segments.
func (p *Path) XMonotone() *Path {
	quad := func(p0, p1, p2 math32.Vector2) *Path {
		return xmonotoneQuadraticBezier(p0, p1, p2)
	}
	cube := func(p0, p1, p2, p3 math32.Vector2) *Path {
		return xmonotoneCubicBezier(p0, p1, p2, p3)
	}
	arc := func(start math32.Vector2, rx, ry, phi float32, large, sweep bool, end math32.Vector2) *Path {
		return xmonotoneEllipticArc(start, rx, ry, phi, large, sweep, end)
	}
	return p.replace(nil, quad, cube, arc)
}

// replace replaces path segments by their respective functions, each returning the path that will replace the segment or nil if no replacement is to be performed. The line function will take the start and end points. The bezier function will take the start point, control point 1 and 2, and the end point (i.e. a cubic Bézier, quadratic Béziers will be implicitly converted to cubic ones). The arc function will take a start point, the major and minor radii, the radial rotaton counter clockwise, the large and sweep booleans, and the end point. The replacing path will replace the path segment without any checks, you need to make sure the be moved so that its start point connects with the last end point of the base path before the replacement. If the end point of the replacing path is different that the end point of what is replaced, the path that follows will be displaced.
func (p *Path) replace(
	line func(math32.Vector2, math32.Vector2) *Path,
	quad func(math32.Vector2, math32.Vector2, math32.Vector2) *Path,
	cube func(math32.Vector2, math32.Vector2, math32.Vector2, math32.Vector2) *Path,
	arc func(math32.Vector2, float32, float32, float32, bool, bool, math32.Vector2) *Path,
) *Path {
	copied := false
	var start, end math32.Vector2
	for i := 0; i < len(p); {
		var q *Path
		cmd := p[i]
		switch cmd {
		case LineTo, Close:
			if line != nil {
				end = math32.Vector2{p[i+1], p[i+2]}
				q = line(start, end)
				if cmd == Close {
					q.Close()
				}
			}
		case QuadTo:
			if quad != nil {
				cp := math32.Vector2{p[i+1], p[i+2]}
				end = math32.Vector2{p[i+3], p[i+4]}
				q = quad(start, cp, end)
			}
		case CubeTo:
			if cube != nil {
				cp1 := math32.Vector2{p[i+1], p[i+2]}
				cp2 := math32.Vector2{p[i+3], p[i+4]}
				end = math32.Vector2{p[i+5], p[i+6]}
				q = cube(start, cp1, cp2, end)
			}
		case ArcTo:
			if arc != nil {
				rx, ry, phi := p[i+1], p[i+2], p[i+3]
				large, sweep := toArcFlags(p[i+4])
				end = math32.Vector2{p[i+5], p[i+6]}
				q = arc(start, rx, ry, phi, large, sweep, end)
			}
		}

		if q != nil {
			if !copied {
				p = p.Copy()
				copied = true
			}

			r := &Path{append([]float32{MoveTo, end.X, end.Y, MoveTo}, p[i+cmdLen(cmd):]...)}

			p = p[: i : i+cmdLen(cmd)] // make sure not to overwrite the rest of the path
			p = p.Join(q)
			if cmd != Close {
				p.LineTo(end.X, end.Y)
			}

			i = len(p)
			p = p.Join(r) // join the rest of the base path
		} else {
			i += cmdLen(cmd)
		}
		start = math32.Vector2{p[i-3], p[i-2]}
	}
	return p
}

// Markers returns an array of start, mid and end marker paths along the path at the coordinates between commands. Align will align the markers with the path direction so that the markers orient towards the path's left.
func (p *Path) Markers(first, mid, last *Path, align bool) []*Path {
	markers := []*Path{}
	coordPos := p.Coords()
	coordDir := p.CoordDirections()
	for i := range coordPos {
		q := mid
		if i == 0 {
			q = first
		} else if i == len(coordPos)-1 {
			q = last
		}

		if q != nil {
			pos, dir := coordPos[i], coordDir[i]
			m := Identity.Translate(pos.X, pos.Y)
			if align {
				m = m.Rotate(dir.Angle() * 180.0 / math.Pi)
			}
			markers = append(markers, q.Copy().Transform(m))
		}
	}
	return markers
}

// Split splits the path into its independent subpaths. The path is split before each MoveTo command.
func (p Path) Split() []Path {
	if p == nil {
		return nil
	}
	var i, j int
	ps := []Path{}
	for j < len(p) {
		cmd := p[j]
		if i < j && cmd == MoveTo {
			ps = append(ps, &Path{p[i:j:j]})
			i = j
		}
		j += cmdLen(cmd)
	}
	if i+cmdLen(MoveTo) < j {
		ps = append(ps, &Path{p[i:j:j]})
	}
	return ps
}

// SplitAt splits the path into separate paths at the specified intervals (given in millimeters) along the path.
func (p Path) SplitAt(ts ...float32) []Path {
	if len(ts) == 0 {
		return []Path{p}
	}

	sort.Float32s(ts)
	if ts[0] == 0.0 {
		ts = ts[1:]
	}

	j := 0   // index into ts
	T := 0.0 // current position along curve

	qs := []Path{}
	q := &Path{}
	push := func() {
		qs = append(qs, q)
		q = &Path{}
	}

	if 0 < len(p) && p[0] == MoveTo {
		q.MoveTo(p[1], p[2])
	}
	for _, ps := range p.Split() {
		var start, end math32.Vector2
		for i := 0; i < len(ps); {
			cmd := ps[i]
			switch cmd {
			case MoveTo:
				end = math32.Vector2{p[i+1], p[i+2]}
			case LineTo, Close:
				end = math32.Vector2{p[i+1], p[i+2]}

				if j == len(ts) {
					q.LineTo(end.X, end.Y)
				} else {
					dT := end.Sub(start).Length()
					Tcurve := T
					for j < len(ts) && T < ts[j] && ts[j] <= T+dT {
						tpos := (ts[j] - T) / dT
						pos := start.Interpolate(end, tpos)
						Tcurve = ts[j]

						q.LineTo(pos.X, pos.Y)
						push()
						q.MoveTo(pos.X, pos.Y)
						j++
					}
					if Tcurve < T+dT {
						q.LineTo(end.X, end.Y)
					}
					T += dT
				}
			case QuadTo:
				cp := math32.Vector2{p[i+1], p[i+2]}
				end = math32.Vector2{p[i+3], p[i+4]}

				if j == len(ts) {
					q.QuadTo(cp.X, cp.Y, end.X, end.Y)
				} else {
					speed := func(t float32) float32 {
						return quadraticBezierDeriv(start, cp, end, t).Length()
					}
					invL, dT := invSpeedPolynomialChebyshevApprox(20, gaussLegendre7, speed, 0.0, 1.0)

					t0 := 0.0
					r0, r1, r2 := start, cp, end
					for j < len(ts) && T < ts[j] && ts[j] <= T+dT {
						t := invL(ts[j] - T)
						tsub := (t - t0) / (1.0 - t0)
						t0 = t

						var q1 math32.Vector2
						_, q1, _, r0, r1, r2 = quadraticBezierSplit(r0, r1, r2, tsub)

						q.QuadTo(q1.X, q1.Y, r0.X, r0.Y)
						push()
						q.MoveTo(r0.X, r0.Y)
						j++
					}
					if !Equal(t0, 1.0) {
						q.QuadTo(r1.X, r1.Y, r2.X, r2.Y)
					}
					T += dT
				}
			case CubeTo:
				cp1 := math32.Vector2{p[i+1], p[i+2]}
				cp2 := math32.Vector2{p[i+3], p[i+4]}
				end = math32.Vector2{p[i+5], p[i+6]}

				if j == len(ts) {
					q.CubeTo(cp1.X, cp1.Y, cp2.X, cp2.Y, end.X, end.Y)
				} else {
					speed := func(t float32) float32 {
						// splitting on inflection points does not improve output
						return cubicBezierDeriv(start, cp1, cp2, end, t).Length()
					}
					N := 20 + 20*cubicBezierNumInflections(start, cp1, cp2, end) // TODO: needs better N
					invL, dT := invSpeedPolynomialChebyshevApprox(N, gaussLegendre7, speed, 0.0, 1.0)

					t0 := 0.0
					r0, r1, r2, r3 := start, cp1, cp2, end
					for j < len(ts) && T < ts[j] && ts[j] <= T+dT {
						t := invL(ts[j] - T)
						tsub := (t - t0) / (1.0 - t0)
						t0 = t

						var q1, q2 math32.Vector2
						_, q1, q2, _, r0, r1, r2, r3 = cubicBezierSplit(r0, r1, r2, r3, tsub)

						q.CubeTo(q1.X, q1.Y, q2.X, q2.Y, r0.X, r0.Y)
						push()
						q.MoveTo(r0.X, r0.Y)
						j++
					}
					if !Equal(t0, 1.0) {
						q.CubeTo(r1.X, r1.Y, r2.X, r2.Y, r3.X, r3.Y)
					}
					T += dT
				}
			case ArcTo:
				rx, ry, phi := p[i+1], p[i+2], p[i+3]
				large, sweep := toArcFlags(p[i+4])
				end = math32.Vector2{p[i+5], p[i+6]}
				cx, cy, theta1, theta2 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)

				if j == len(ts) {
					q.ArcTo(rx, ry, phi*180.0/math.Pi, large, sweep, end.X, end.Y)
				} else {
					speed := func(theta float32) float32 {
						return ellipseDeriv(rx, ry, 0.0, true, theta).Length()
					}
					invL, dT := invSpeedPolynomialChebyshevApprox(10, gaussLegendre7, speed, theta1, theta2)

					startTheta := theta1
					nextLarge := large
					for j < len(ts) && T < ts[j] && ts[j] <= T+dT {
						theta := invL(ts[j] - T)
						mid, large1, large2, ok := ellipseSplit(rx, ry, phi, cx, cy, startTheta, theta2, theta)
						if !ok {
							panic("theta not in elliptic arc range for splitting")
						}

						q.ArcTo(rx, ry, phi*180.0/math.Pi, large1, sweep, mid.X, mid.Y)
						push()
						q.MoveTo(mid.X, mid.Y)
						startTheta = theta
						nextLarge = large2
						j++
					}
					if !Equal(startTheta, theta2) {
						q.ArcTo(rx, ry, phi*180.0/math.Pi, nextLarge, sweep, end.X, end.Y)
					}
					T += dT
				}
			}
			i += cmdLen(cmd)
			start = end
		}
	}
	if cmdLen(MoveTo) < len(q) {
		push()
	}
	return qs
}

func dashStart(offset float32, d []float32) (int, float32) {
	i0 := 0 // index in d
	for d[i0] <= offset {
		offset -= d[i0]
		i0++
		if i0 == len(d) {
			i0 = 0
		}
	}
	pos0 := -offset // negative if offset is halfway into dash
	if offset < 0.0 {
		dTotal := 0.0
		for _, dd := range d {
			dTotal += dd
		}
		pos0 = -(dTotal + offset) // handle negative offsets
	}
	return i0, pos0
}

// dashCanonical returns an optimized dash array.
func dashCanonical(offset float32, d []float32) (float32, []float32) {
	if len(d) == 0 {
		return 0.0, []float32{}
	}

	// remove zeros except first and last
	for i := 1; i < len(d)-1; i++ {
		if Equal(d[i], 0.0) {
			d[i-1] += d[i+1]
			d = append(d[:i], d[i+2:]...)
			i--
		}
	}

	// remove first zero, collapse with second and last
	if Equal(d[0], 0.0) {
		if len(d) < 3 {
			return 0.0, []float32{0.0}
		}
		offset -= d[1]
		d[len(d)-1] += d[1]
		d = d[2:]
	}

	// remove last zero, collapse with fist and second to last
	if Equal(d[len(d)-1], 0.0) {
		if len(d) < 3 {
			return 0.0, []float32{}
		}
		offset += d[len(d)-2]
		d[0] += d[len(d)-2]
		d = d[:len(d)-2]
	}

	// if there are zeros or negatives, don't draw any dashes
	for i := 0; i < len(d); i++ {
		if d[i] < 0.0 || Equal(d[i], 0.0) {
			return 0.0, []float32{0.0}
		}
	}

	// remove repeated patterns
REPEAT:
	for len(d)%2 == 0 {
		mid := len(d) / 2
		for i := 0; i < mid; i++ {
			if !Equal(d[i], d[mid+i]) {
				break REPEAT
			}
		}
		d = d[:mid]
	}
	return offset, d
}

func (p Path) checkDash(offset float32, d []float32) ([]float32, bool) {
	offset, d = dashCanonical(offset, d)
	if len(d) == 0 {
		return d, true // stroke without dashes
	} else if len(d) == 1 && d[0] == 0.0 {
		return d[:0], false // no dashes, no stroke
	}

	length := p.Length()
	i, pos := dashStart(offset, d)
	if length <= d[i]-pos {
		if i%2 == 0 {
			return d[:0], true // first dash covers whole path, stroke without dashes
		}
		return d[:0], false // first space covers whole path, no stroke
	}
	return d, true
}

// Dash returns a new path that consists of dashes. The elements in d specify the width of the dashes and gaps. It will alternate between dashes and gaps when picking widths. If d is an array of odd length, it is equivalent of passing d twice in sequence. The offset specifies the offset used into d (or negative offset into the path). Dash will be applied to each subpath independently.
func (p Path) Dash(offset float32, d ...float32) Path {
	offset, d = dashCanonical(offset, d)
	if len(d) == 0 {
		return p
	} else if len(d) == 1 && d[0] == 0.0 {
		return &Path{}
	}

	if len(d)%2 == 1 {
		// if d is uneven length, dash and space lengths alternate. Duplicate d so that uneven indices are always spaces
		d = append(d, d...)
	}

	i0, pos0 := dashStart(offset, d)

	q := &Path{}
	for _, ps := range p.Split() {
		i := i0
		pos := pos0

		t := []float32{}
		length := ps.Length()
		for pos+d[i]+Epsilon < length {
			pos += d[i]
			if 0.0 < pos {
				t = append(t, pos)
			}
			i++
			if i == len(d) {
				i = 0
			}
		}

		j0 := 0
		endsInDash := i%2 == 0
		if len(t)%2 == 1 && endsInDash || len(t)%2 == 0 && !endsInDash {
			j0 = 1
		}

		qd := &Path{}
		pd := ps.SplitAt(t...)
		for j := j0; j < len(pd)-1; j += 2 {
			qd = qd.Append(pd[j])
		}
		if endsInDash {
			if ps.Closed() {
				qd = pd[len(pd)-1].Join(qd)
			} else {
				qd = qd.Append(pd[len(pd)-1])
			}
		}
		q = q.Append(qd)
	}
	return q
}

// Reverse returns a new path that is the same path as p but in the reverse direction.
func (p Path) Reverse() Path {
	if len(p) == 0 {
		return p
	}

	end := math32.Vector2{p[len(p)-3], p[len(p)-2]}
	q := &Path{d: make([]float32, 0, len(p))}
	q = append(q, MoveTo, end.X, end.Y, MoveTo)

	closed := false
	first, start := end, end
	for i := len(p); 0 < i; {
		cmd := p[i-1]
		i -= cmdLen(cmd)

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
			if !start.Equals(end) {
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
			rx, ry, phi := p[i+1], p[i+2], p[i+3]
			large, sweep := toArcFlags(p[i+4])
			q = append(q, ArcTo, rx, ry, phi, fromArcFlags(large, !sweep), end.X, end.Y, ArcTo)
		}
		start = end
	}
	if closed {
		q = append(q, Close, first.X, first.Y, Close)
	}
	return q
}

////////////////////////////////////////////////////////////////

func skipCommaWhitespace(path []byte) int {
	i := 0
	for i < len(path) && (path[i] == ' ' || path[i] == ',' || path[i] == '\n' || path[i] == '\r' || path[i] == '\t') {
		i++
	}
	return i
}

// MustParseSVGPath parses an SVG path data string and panics if it fails.
func MustParseSVGPath(s string) Path {
	p, err := ParseSVGPath(s)
	if err != nil {
		panic(err)
	}
	return p
}

// ParseSVGPath parses an SVG path data string.
func ParseSVGPath(s string) (Path, error) {
	if len(s) == 0 {
		return &Path{}, nil
	}

	i := 0
	path := []byte(s)
	i += skipCommaWhitespace(path[i:])
	if path[0] == ',' || path[i] < 'A' {
		return nil, fmt.Errorf("bad path: path should start with command")
	}

	cmdLens := map[byte]int{
		'M': 2,
		'Z': 0,
		'L': 2,
		'H': 1,
		'V': 1,
		'C': 6,
		'S': 4,
		'Q': 4,
		'T': 2,
		'A': 7,
	}
	f := [7]float32{}

	p := &Path{}
	var q, c math32.Vector2
	var p0, p1 math32.Vector2
	prevCmd := byte('z')
	for {
		i += skipCommaWhitespace(path[i:])
		if len(path) <= i {
			break
		}

		cmd := prevCmd
		repeat := true
		if cmd == 'z' || cmd == 'Z' || !(path[i] >= '0' && path[i] <= '9' || path[i] == '.' || path[i] == '-' || path[i] == '+') {
			cmd = path[i]
			repeat = false
			i++
			i += skipCommaWhitespace(path[i:])
		}

		CMD := cmd
		if 'a' <= cmd && cmd <= 'z' {
			CMD -= 'a' - 'A'
		}
		for j := 0; j < cmdLens[CMD]; j++ {
			if CMD == 'A' && (j == 3 || j == 4) {
				// parse largeArc and sweep booleans for A command
				if i < len(path) && path[i] == '1' {
					f[j] = 1.0
				} else if i < len(path) && path[i] == '0' {
					f[j] = 0.0
				} else {
					return nil, fmt.Errorf("bad path: largeArc and sweep flags should be 0 or 1 in command '%c' at position %d", cmd, i+1)
				}
				i++
			} else {
				num, n := strconv.ParseFloat(path[i:])
				if n == 0 {
					if repeat && j == 0 && i < len(path) {
						return nil, fmt.Errorf("bad path: unknown command '%c' at position %d", path[i], i+1)
					} else if 1 < cmdLens[CMD] {
						return nil, fmt.Errorf("bad path: sets of %d numbers should follow command '%c' at position %d", cmdLens[CMD], cmd, i+1)
					} else {
						return nil, fmt.Errorf("bad path: number should follow command '%c' at position %d", cmd, i+1)
					}
				}
				f[j] = num
				i += n
			}
			i += skipCommaWhitespace(path[i:])
		}

		switch cmd {
		case 'M', 'm':
			p1 = math32.Vector2{f[0], f[1]}
			if cmd == 'm' {
				p1 = p1.Add(p0)
				cmd = 'l'
			} else {
				cmd = 'L'
			}
			p.MoveTo(p1.X, p1.Y)
		case 'Z', 'z':
			p1 = p.StartPos()
			p.Close()
		case 'L', 'l':
			p1 = math32.Vector2{f[0], f[1]}
			if cmd == 'l' {
				p1 = p1.Add(p0)
			}
			p.LineTo(p1.X, p1.Y)
		case 'H', 'h':
			p1.X = f[0]
			if cmd == 'h' {
				p1.X += p0.X
			}
			p.LineTo(p1.X, p1.Y)
		case 'V', 'v':
			p1.Y = f[0]
			if cmd == 'v' {
				p1.Y += p0.Y
			}
			p.LineTo(p1.X, p1.Y)
		case 'C', 'c':
			cp1 := math32.Vector2{f[0], f[1]}
			cp2 := math32.Vector2{f[2], f[3]}
			p1 = math32.Vector2{f[4], f[5]}
			if cmd == 'c' {
				cp1 = cp1.Add(p0)
				cp2 = cp2.Add(p0)
				p1 = p1.Add(p0)
			}
			p.CubeTo(cp1.X, cp1.Y, cp2.X, cp2.Y, p1.X, p1.Y)
			c = cp2
		case 'S', 's':
			cp1 := p0
			cp2 := math32.Vector2{f[0], f[1]}
			p1 = math32.Vector2{f[2], f[3]}
			if cmd == 's' {
				cp2 = cp2.Add(p0)
				p1 = p1.Add(p0)
			}
			if prevCmd == 'C' || prevCmd == 'c' || prevCmd == 'S' || prevCmd == 's' {
				cp1 = p0.Mul(2.0).Sub(c)
			}
			p.CubeTo(cp1.X, cp1.Y, cp2.X, cp2.Y, p1.X, p1.Y)
			c = cp2
		case 'Q', 'q':
			cp := math32.Vector2{f[0], f[1]}
			p1 = math32.Vector2{f[2], f[3]}
			if cmd == 'q' {
				cp = cp.Add(p0)
				p1 = p1.Add(p0)
			}
			p.QuadTo(cp.X, cp.Y, p1.X, p1.Y)
			q = cp
		case 'T', 't':
			cp := p0
			p1 = math32.Vector2{f[0], f[1]}
			if cmd == 't' {
				p1 = p1.Add(p0)
			}
			if prevCmd == 'Q' || prevCmd == 'q' || prevCmd == 'T' || prevCmd == 't' {
				cp = p0.Mul(2.0).Sub(q)
			}
			p.QuadTo(cp.X, cp.Y, p1.X, p1.Y)
			q = cp
		case 'A', 'a':
			rx := f[0]
			ry := f[1]
			rot := f[2]
			large := f[3] == 1.0
			sweep := f[4] == 1.0
			p1 = math32.Vector2{f[5], f[6]}
			if cmd == 'a' {
				p1 = p1.Add(p0)
			}
			p.ArcTo(rx, ry, rot, large, sweep, p1.X, p1.Y)
		default:
			return nil, fmt.Errorf("bad path: unknown command '%c' at position %d", cmd, i+1)
		}
		prevCmd = cmd
		p0 = p1
	}
	return p, nil
}

// String returns a string that represents the path similar to the SVG path data format (but not necessarily valid SVG).
func (p Path) String() string {
	sb := strings.Builder{}
	for i := 0; i < len(p); {
		cmd := p[i]
		switch cmd {
		case MoveTo:
			fmt.Fprintf(&sb, "M%g %g", p[i+1], p[i+2])
		case LineTo:
			fmt.Fprintf(&sb, "L%g %g", p[i+1], p[i+2])
		case QuadTo:
			fmt.Fprintf(&sb, "Q%g %g %g %g", p[i+1], p[i+2], p[i+3], p[i+4])
		case CubeTo:
			fmt.Fprintf(&sb, "C%g %g %g %g %g %g", p[i+1], p[i+2], p[i+3], p[i+4], p[i+5], p[i+6])
		case ArcTo:
			rot := p[i+3] * 180.0 / math.Pi
			large, sweep := toArcFlags(p[i+4])
			sLarge := "0"
			if large {
				sLarge = "1"
			}
			sSweep := "0"
			if sweep {
				sSweep = "1"
			}
			fmt.Fprintf(&sb, "A%g %g %g %s %s %g %g", p[i+1], p[i+2], rot, sLarge, sSweep, p[i+5], p[i+6])
		case Close:
			fmt.Fprintf(&sb, "z")
		}
		i += cmdLen(cmd)
	}
	return sb.String()
}

// ToSVG returns a string that represents the path in the SVG path data format with minification.
func (p Path) ToSVG() string {
	if p.Empty() {
		return ""
	}

	sb := strings.Builder{}
	var x, y float32
	for i := 0; i < len(p); {
		cmd := p[i]
		switch cmd {
		case MoveTo:
			x, y = p[i+1], p[i+2]
			fmt.Fprintf(&sb, "M%v %v", num(x), num(y))
		case LineTo:
			xStart, yStart := x, y
			x, y = p[i+1], p[i+2]
			if Equal(x, xStart) && Equal(y, yStart) {
				// nothing
			} else if Equal(x, xStart) {
				fmt.Fprintf(&sb, "V%v", num(y))
			} else if Equal(y, yStart) {
				fmt.Fprintf(&sb, "H%v", num(x))
			} else {
				fmt.Fprintf(&sb, "L%v %v", num(x), num(y))
			}
		case QuadTo:
			x, y = p[i+3], p[i+4]
			fmt.Fprintf(&sb, "Q%v %v %v %v", num(p[i+1]), num(p[i+2]), num(x), num(y))
		case CubeTo:
			x, y = p[i+5], p[i+6]
			fmt.Fprintf(&sb, "C%v %v %v %v %v %v", num(p[i+1]), num(p[i+2]), num(p[i+3]), num(p[i+4]), num(x), num(y))
		case ArcTo:
			rx, ry := p[i+1], p[i+2]
			rot := p[i+3] * 180.0 / math.Pi
			large, sweep := toArcFlags(p[i+4])
			x, y = p[i+5], p[i+6]
			sLarge := "0"
			if large {
				sLarge = "1"
			}
			sSweep := "0"
			if sweep {
				sSweep = "1"
			}
			if 90.0 <= rot {
				rx, ry = ry, rx
				rot -= 90.0
			}
			fmt.Fprintf(&sb, "A%v %v %v %s%s%v %v", num(rx), num(ry), num(rot), sLarge, sSweep, num(p[i+5]), num(p[i+6]))
		case Close:
			x, y = p[i+1], p[i+2]
			fmt.Fprintf(&sb, "z")
		}
		i += cmdLen(cmd)
	}
	return sb.String()
}

// ToPS returns a string that represents the path in the PostScript data format.
func (p Path) ToPS() string {
	if p.Empty() {
		return ""
	}

	sb := strings.Builder{}
	var x, y float32
	for i := 0; i < len(p); {
		cmd := p[i]
		switch cmd {
		case MoveTo:
			x, y = p[i+1], p[i+2]
			fmt.Fprintf(&sb, " %v %v moveto", dec(x), dec(y))
		case LineTo:
			x, y = p[i+1], p[i+2]
			fmt.Fprintf(&sb, " %v %v lineto", dec(x), dec(y))
		case QuadTo, CubeTo:
			var start, cp1, cp2 math32.Vector2
			start = math32.Vector2{x, y}
			if cmd == QuadTo {
				x, y = p[i+3], p[i+4]
				cp1, cp2 = quadraticToCubicBezier(start, math32.Vector2{p[i+1], p[i+2]}, math32.Vector2{x, y})
			} else {
				cp1 = math32.Vector2{p[i+1], p[i+2]}
				cp2 = math32.Vector2{p[i+3], p[i+4]}
				x, y = p[i+5], p[i+6]
			}
			fmt.Fprintf(&sb, " %v %v %v %v %v %v curveto", dec(cp1.X), dec(cp1.Y), dec(cp2.X), dec(cp2.Y), dec(x), dec(y))
		case ArcTo:
			x0, y0 := x, y
			rx, ry, phi := p[i+1], p[i+2], p[i+3]
			large, sweep := toArcFlags(p[i+4])
			x, y = p[i+5], p[i+6]

			cx, cy, theta0, theta1 := ellipseToCenter(x0, y0, rx, ry, phi, large, sweep, x, y)
			theta0 = theta0 * 180.0 / math.Pi
			theta1 = theta1 * 180.0 / math.Pi
			rot := phi * 180.0 / math.Pi

			fmt.Fprintf(&sb, " %v %v %v %v %v %v %v ellipse", dec(cx), dec(cy), dec(rx), dec(ry), dec(theta0), dec(theta1), dec(rot))
			if !sweep {
				fmt.Fprintf(&sb, "n")
			}
		case Close:
			x, y = p[i+1], p[i+2]
			fmt.Fprintf(&sb, " closepath")
		}
		i += cmdLen(cmd)
	}
	return sb.String()[1:] // remove the first space
}

// ToPDF returns a string that represents the path in the PDF data format.
func (p Path) ToPDF() string {
	if p.Empty() {
		return ""
	}
	p = p.ReplaceArcs()

	sb := strings.Builder{}
	var x, y float32
	for i := 0; i < len(p); {
		cmd := p[i]
		switch cmd {
		case MoveTo:
			x, y = p[i+1], p[i+2]
			fmt.Fprintf(&sb, " %v %v m", dec(x), dec(y))
		case LineTo:
			x, y = p[i+1], p[i+2]
			fmt.Fprintf(&sb, " %v %v l", dec(x), dec(y))
		case QuadTo, CubeTo:
			var start, cp1, cp2 math32.Vector2
			start = math32.Vector2{x, y}
			if cmd == QuadTo {
				x, y = p[i+3], p[i+4]
				cp1, cp2 = quadraticToCubicBezier(start, math32.Vector2{p[i+1], p[i+2]}, math32.Vector2{x, y})
			} else {
				cp1 = math32.Vector2{p[i+1], p[i+2]}
				cp2 = math32.Vector2{p[i+3], p[i+4]}
				x, y = p[i+5], p[i+6]
			}
			fmt.Fprintf(&sb, " %v %v %v %v %v %v c", dec(cp1.X), dec(cp1.Y), dec(cp2.X), dec(cp2.Y), dec(x), dec(y))
		case ArcTo:
			panic("arcs should have been replaced")
		case Close:
			x, y = p[i+1], p[i+2]
			fmt.Fprintf(&sb, " h")
		}
		i += cmdLen(cmd)
	}
	return sb.String()[1:] // remove the first space
}

// ToRasterizer rasterizes the path using the given rasterizer and resolution.
func (p Path) ToRasterizer(ras *vector.Rasterizer, resolution Resolution) {
	// TODO: smoothen path using Ramer-...

	dpmm := resolutionPMM()
	tolerance := PixelTolerance / dpmm // tolerance of 1/10 of a pixel
	dy := float32(ras.Bounds().Size().Y)
	for i := 0; i < len(p); {
		cmd := p[i]
		switch cmd {
		case MoveTo:
			ras.MoveTo(float32(p[i+1]*dpmm), float32(dy-p[i+2]*dpmm))
		case LineTo:
			ras.LineTo(float32(p[i+1]*dpmm), float32(dy-p[i+2]*dpmm))
		case QuadTo, CubeTo, ArcTo:
			// flatten
			var q Path
			var start math32.Vector2
			if 0 < i {
				start = math32.Vector2{p[i-3], p[i-2]}
			}
			if cmd == QuadTo {
				cp := math32.Vector2{p[i+1], p[i+2]}
				end := math32.Vector2{p[i+3], p[i+4]}
				q = flattenQuadraticBezier(start, cp, end, tolerance)
			} else if cmd == CubeTo {
				cp1 := math32.Vector2{p[i+1], p[i+2]}
				cp2 := math32.Vector2{p[i+3], p[i+4]}
				end := math32.Vector2{p[i+5], p[i+6]}
				q = flattenCubicBezier(start, cp1, cp2, end, tolerance)
			} else {
				rx, ry, phi := p[i+1], p[i+2], p[i+3]
				large, sweep := toArcFlags(p[i+4])
				end := math32.Vector2{p[i+5], p[i+6]}
				q = flattenEllipticArc(start, rx, ry, phi, large, sweep, end, tolerance)
			}
			for j := 4; j < len(q); j += 4 {
				ras.LineTo(float32(q[j+1]*dpmm), float32(dy-q[j+2]*dpmm))
			}
		case Close:
			ras.ClosePath()
		default:
			panic("quadratic and cubic Béziers and arcs should have been replaced")
		}
		i += cmdLen(cmd)
	}
	if !p.Closed() {
		// implicitly close path
		ras.ClosePath()
	}
}
