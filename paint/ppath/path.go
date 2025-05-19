// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package ppath

import (
	"bytes"
	"encoding/gob"
	"slices"

	"cogentcore.org/core/math32"
)

// ArcToCubeImmediate causes ArcTo commands to be immediately converted into
// corresponding CubeTo commands, instead of doing this later.
// This is faster than using [Path.ReplaceArcs], but when rendering to SVG
// it might be better to turn this off in order to preserve the logical structure
// of the arcs in the SVG output.
var ArcToCubeImmediate = true

// Path is a collection of MoveTo, LineTo, QuadTo, CubeTo, ArcTo, and Close
// commands, each followed the float32 coordinate data for it.
// To enable support bidirectional processing, the command verb is also added
// to the end of the coordinate data as well.
// The last two coordinate values are the end point position of the pen after
// the action (x,y).
// QuadTo defines one control point (x,y) in between.
// CubeTo defines two control points.
// ArcTo defines (rx,ry,phi,large+sweep) i.e. the radius in x and y,
// its rotation (in radians) and the large and sweep booleans in one float32.
// While ArcTo can be converted to CubeTo, it is useful for the path intersection
// computation.
// Only valid commands are appended, so that LineTo has a non-zero length,
// QuadTo's and CubeTo's control point(s) don't (both) overlap with the start
// and end point.
type Path []float32

func New() *Path {
	return &Path{}
}

// Commands
const (
	MoveTo float32 = 0
	LineTo float32 = 1
	QuadTo float32 = 2
	CubeTo float32 = 3
	ArcTo  float32 = 4
	Close  float32 = 5
)

var cmdLens = [6]int{4, 4, 6, 8, 8, 4}

// CmdLen returns the overall length of the command, including
// the command op itself.
func CmdLen(cmd float32) int {
	return cmdLens[int(cmd)]
}

// ToArcFlags converts to the largeArc and sweep boolean flags given its value in the path.
func ToArcFlags(cmd float32) (bool, bool) {
	large := (cmd == 1.0 || cmd == 3.0)
	sweep := (cmd == 2.0 || cmd == 3.0)
	return large, sweep
}

// fromArcFlags converts the largeArc and sweep boolean flags to a value stored in the path.
func fromArcFlags(large, sweep bool) float32 {
	f := float32(0.0)
	if large {
		f += 1.0
	}
	if sweep {
		f += 2.0
	}
	return f
}

// Paths is a collection of Path elements.
type Paths []Path

// Empty returns true if the set of paths is empty.
func (ps Paths) Empty() bool {
	for _, p := range ps {
		if !p.Empty() {
			return false
		}
	}
	return true
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
	return len(p) <= CmdLen(MoveTo)
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
		return !math32.IsNaN(x) && !math32.IsInf(x, 0.0)
	}
	for i := 0; i < len(p); {
		cmd := p[i]
		i += CmdLen(cmd)

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
		j += CmdLen(q[j])
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
		i += CmdLen(p[i])
	}
	return false
}

// Clone returns a copy of p.
func (p Path) Clone() Path {
	return slices.Clone(p)
}

// CopyTo returns a copy of p, using the memory of path q.
func (p Path) CopyTo(q Path) Path {
	if q == nil || len(q) < len(p) {
		q = make(Path, len(p))
	} else {
		q = q[:len(p)]
	}
	copy(q, p)
	return q
}

// Len returns the number of commands in the path.
func (p Path) Len() int {
	n := 0
	for i := 0; i < len(p); {
		i += CmdLen(p[i])
		n++
	}
	return n
}

// Append appends path q to p and returns the extended path p.
func (p Path) Append(qs ...Path) Path {
	if p.Empty() {
		p = Path{}
	}
	for _, q := range qs {
		if !q.Empty() {
			p = append(p, q...)
		}
	}
	return p
}

// Join joins path q to p and returns the extended path p
// (or q if p is empty). It's like executing the commands
// in q to p in sequence, where if the first MoveTo of q
// doesn't coincide with p, or if p ends in Close,
// it will fallback to appending the paths.
func (p Path) Join(q Path) Path {
	if q.Empty() {
		return p
	} else if p.Empty() {
		return q
	}

	if p[len(p)-1] == Close || !Equal(p[len(p)-3], q[1]) || !Equal(p[len(p)-2], q[2]) {
		return append(p, q...)
	}

	d := q[CmdLen(MoveTo):]

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
		large, sweep := ToArcFlags(d[4])
		p.ArcTo(d[1], d[2], d[3], large, sweep, d[5], d[6])
	case Close:
		p.Close()
	}

	i := len(p)
	end := p.StartPos()
	p = append(p, d[CmdLen(cmd):]...)

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
		i += CmdLen(cmd)
	}
	return p

}

// Pos returns the current position of the path,
// which is the end point of the last command.
func (p Path) Pos() math32.Vector2 {
	if 0 < len(p) {
		return math32.Vec2(p[len(p)-3], p[len(p)-2])
	}
	return math32.Vector2{}
}

// StartPos returns the start point of the current subpath,
// i.e. it returns the position of the last MoveTo command.
func (p Path) StartPos() math32.Vector2 {
	for i := len(p); 0 < i; {
		cmd := p[i-1]
		if cmd == MoveTo {
			return math32.Vec2(p[i-3], p[i-2])
		}
		i -= CmdLen(cmd)
	}
	return math32.Vector2{}
}

// Coords returns all the coordinates of the segment
// start/end points. It omits zero-length Closes.
func (p Path) Coords() []math32.Vector2 {
	coords := []math32.Vector2{}
	for i := 0; i < len(p); {
		cmd := p[i]
		i += CmdLen(cmd)
		if len(coords) == 0 || cmd != Close || !EqualPoint(coords[len(coords)-1], math32.Vec2(p[i-3], p[i-2])) {
			coords = append(coords, math32.Vec2(p[i-3], p[i-2]))
		}
	}
	return coords
}

/////// Accessors

// EndPoint returns the end point for MoveTo, LineTo, and Close commands,
// where the command is at index i.
func (p Path) EndPoint(i int) math32.Vector2 {
	return math32.Vec2(p[i+1], p[i+2])
}

// QuadToPoints returns the control point and end for QuadTo command,
// where the command is at index i.
func (p Path) QuadToPoints(i int) (cp, end math32.Vector2) {
	return math32.Vec2(p[i+1], p[i+2]), math32.Vec2(p[i+3], p[i+4])
}

// CubeToPoints returns the cp1, cp2, and end for CubeTo command,
// where the command is at index i.
func (p Path) CubeToPoints(i int) (cp1, cp2, end math32.Vector2) {
	return math32.Vec2(p[i+1], p[i+2]), math32.Vec2(p[i+3], p[i+4]), math32.Vec2(p[i+5], p[i+6])
}

// ArcToPoints returns the rx, ry, phi, large, sweep values for ArcTo command,
// where the command is at index i.
func (p Path) ArcToPoints(i int) (rx, ry, phi float32, large, sweep bool, end math32.Vector2) {
	rx = p[i+1]
	ry = p[i+2]
	phi = p[i+3]
	large, sweep = ToArcFlags(p[i+4])
	end = math32.Vec2(p[i+5], p[i+6])
	return
}

/////// Constructors

// MoveTo moves the path to (x,y) without connecting the path.
// It starts a new independent subpath. Multiple subpaths can be useful
// when negating parts of a previous path by overlapping it with a path
// in the opposite direction. The behaviour for overlapping paths depends
// on the FillRules.
func (p *Path) MoveTo(x, y float32) {
	if 0 < len(*p) && (*p)[len(*p)-1] == MoveTo {
		(*p)[len(*p)-3] = x
		(*p)[len(*p)-2] = y
		return
	}
	*p = append(*p, MoveTo, x, y, MoveTo)
}

// LineTo adds a linear path to (x,y).
func (p *Path) LineTo(x, y float32) {
	start := p.Pos()
	end := math32.Vector2{x, y}
	if EqualPoint(start, end) {
		return
	} else if CmdLen(LineTo) <= len(*p) && (*p)[len(*p)-1] == LineTo {
		prevStart := math32.Vector2{}
		if CmdLen(LineTo) < len(*p) {
			prevStart = math32.Vec2((*p)[len(*p)-CmdLen(LineTo)-3], (*p)[len(*p)-CmdLen(LineTo)-2])
		}

		// divide by length^2 since otherwise the perpdot between very small segments may be
		// below Epsilon
		da := start.Sub(prevStart)
		db := end.Sub(start)
		div := da.Cross(db)
		if length := da.Length() * db.Length(); Equal(div/length, 0.0) {
			// lines are parallel
			extends := false
			if da.Y < da.X {
				extends = math32.Signbit(da.X) == math32.Signbit(db.X)
			} else {
				extends = math32.Signbit(da.Y) == math32.Signbit(db.Y)
			}
			if extends {
				//if Equal(end.Sub(start).AngleBetween(start.Sub(prevStart)), 0.0) {
				(*p)[len(*p)-3] = x
				(*p)[len(*p)-2] = y
				return
			}
		}
	}

	if len(*p) == 0 {
		p.MoveTo(0.0, 0.0)
	} else if (*p)[len(*p)-1] == Close {
		p.MoveTo((*p)[len(*p)-3], (*p)[len(*p)-2])
	}
	*p = append(*p, LineTo, end.X, end.Y, LineTo)
}

// QuadTo adds a quadratic Bézier path with control point (cpx,cpy) and end point (x,y).
func (p *Path) QuadTo(cpx, cpy, x, y float32) {
	start := p.Pos()
	cp := math32.Vector2{cpx, cpy}
	end := math32.Vector2{x, y}
	if EqualPoint(start, end) && EqualPoint(start, cp) {
		return
	} else if !EqualPoint(start, end) && (EqualPoint(start, cp) || AngleEqual(AngleBetween(end.Sub(start), cp.Sub(start)), 0.0)) && (EqualPoint(end, cp) || AngleEqual(AngleBetween(end.Sub(start), end.Sub(cp)), 0.0)) {
		p.LineTo(end.X, end.Y)
		return
	}

	if len(*p) == 0 {
		p.MoveTo(0.0, 0.0)
	} else if (*p)[len(*p)-1] == Close {
		p.MoveTo((*p)[len(*p)-3], (*p)[len(*p)-2])
	}
	*p = append(*p, QuadTo, cp.X, cp.Y, end.X, end.Y, QuadTo)
}

// CubeTo adds a cubic Bézier path with control points
// (cpx1,cpy1) and (cpx2,cpy2) and end point (x,y).
func (p *Path) CubeTo(cpx1, cpy1, cpx2, cpy2, x, y float32) {
	start := p.Pos()
	cp1 := math32.Vector2{cpx1, cpy1}
	cp2 := math32.Vector2{cpx2, cpy2}
	end := math32.Vector2{x, y}
	if EqualPoint(start, end) && EqualPoint(start, cp1) && EqualPoint(start, cp2) {
		return
	} else if !EqualPoint(start, end) && (EqualPoint(start, cp1) || EqualPoint(end, cp1) || AngleEqual(AngleBetween(end.Sub(start), cp1.Sub(start)), 0.0) && AngleEqual(AngleBetween(end.Sub(start), end.Sub(cp1)), 0.0)) && (EqualPoint(start, cp2) || EqualPoint(end, cp2) || AngleEqual(AngleBetween(end.Sub(start), cp2.Sub(start)), 0.0) && AngleEqual(AngleBetween(end.Sub(start), end.Sub(cp2)), 0.0)) {
		p.LineTo(end.X, end.Y)
		return
	}

	if len(*p) == 0 {
		p.MoveTo(0.0, 0.0)
	} else if (*p)[len(*p)-1] == Close {
		p.MoveTo((*p)[len(*p)-3], (*p)[len(*p)-2])
	}
	*p = append(*p, CubeTo, cp1.X, cp1.Y, cp2.X, cp2.Y, end.X, end.Y, CubeTo)
}

// ArcTo adds an arc with radii rx and ry, with rot the counter clockwise
// rotation with respect to the coordinate system in radians, large and sweep booleans
// (see https://developer.mozilla.org/en-US/docs/Web/SVG/Tutorial/Paths#Arcs),
// and (x,y) the end position of the pen. The start position of the pen was
// given by a previous command's end point.
func (p *Path) ArcTo(rx, ry, rot float32, large, sweep bool, x, y float32) {
	start := p.Pos()
	end := math32.Vector2{x, y}
	if EqualPoint(start, end) {
		return
	}
	if Equal(rx, 0.0) || math32.IsInf(rx, 0) || Equal(ry, 0.0) || math32.IsInf(ry, 0) {
		p.LineTo(end.X, end.Y)
		return
	}

	rx = math32.Abs(rx)
	ry = math32.Abs(ry)
	if Equal(rx, ry) {
		rot = 0.0 // circle
	} else if rx < ry {
		rx, ry = ry, rx
		rot += math32.Pi / 2.0
	}

	phi := AngleNorm(rot)
	if math32.Pi <= phi { // phi is canonical within 0 <= phi < 180
		phi -= math32.Pi
	}

	// scale ellipse if rx and ry are too small
	lambda := EllipseRadiiCorrection(start, rx, ry, phi, end)
	if lambda > 1.0 {
		rx *= lambda
		ry *= lambda
	}

	if len(*p) == 0 {
		p.MoveTo(0.0, 0.0)
	} else if (*p)[len(*p)-1] == Close {
		p.MoveTo((*p)[len(*p)-3], (*p)[len(*p)-2])
	}
	if ArcToCubeImmediate {
		for _, bezier := range ellipseToCubicBeziers(start, rx, ry, phi, large, sweep, end) {
			p.CubeTo(bezier[1].X, bezier[1].Y, bezier[2].X, bezier[2].Y, bezier[3].X, bezier[3].Y)
		}
	} else {
		*p = append(*p, ArcTo, rx, ry, phi, fromArcFlags(large, sweep), end.X, end.Y, ArcTo)
	}
}

// ArcToDeg is a version of [Path.ArcTo] with the angle in degrees instead of radians.
// It adds an arc with radii rx and ry, with rot the counter clockwise
// rotation with respect to the coordinate system in degrees, large and sweep booleans
// (see https://developer.mozilla.org/en-US/docs/Web/SVG/Tutorial/Paths#Arcs),
// and (x,y) the end position of the pen. The start position of the pen was
// given by a previous command's end point.
func (p *Path) ArcToDeg(rx, ry, rot float32, large, sweep bool, x, y float32) {
	p.ArcTo(rx, ry, math32.DegToRad(rot), large, sweep, x, y)
}

// Arc adds an elliptical arc with radii rx and ry, with rot the
// counter clockwise rotation in radians, and theta0 and theta1
// the angles in radians of the ellipse (before rot is applies)
// between which the arc will run. If theta0 < theta1,
// the arc will run in a CCW direction. If the difference between
// theta0 and theta1 is bigger than 360 degrees, one full circle
// will be drawn and the remaining part of diff % 360,
// e.g. a difference of 810 degrees will draw one full circle
// and an arc over 90 degrees.
func (p *Path) Arc(rx, ry, phi, theta0, theta1 float32) {
	dtheta := math32.Abs(theta1 - theta0)

	sweep := theta0 < theta1
	large := math32.Mod(dtheta, 2.0*math32.Pi) > math32.Pi
	p0 := EllipsePos(rx, ry, phi, 0.0, 0.0, theta0)
	p1 := EllipsePos(rx, ry, phi, 0.0, 0.0, theta1)

	start := p.Pos()
	center := start.Sub(p0)
	if dtheta >= 2.0*math32.Pi {
		startOpposite := center.Sub(p0)
		p.ArcTo(rx, ry, phi, large, sweep, startOpposite.X, startOpposite.Y)
		p.ArcTo(rx, ry, phi, large, sweep, start.X, start.Y)
		if Equal(math32.Mod(dtheta, 2.0*math32.Pi), 0.0) {
			return
		}
	}
	end := center.Add(p1)
	p.ArcTo(rx, ry, phi, large, sweep, end.X, end.Y)
}

// ArcDeg is a version of [Path.Arc] that uses degrees instead of radians,
// to add an elliptical arc with radii rx and ry, with rot the
// counter clockwise rotation in degrees, and theta0 and theta1
// the angles in degrees of the ellipse (before rot is applied)
// between which the arc will run.
func (p *Path) ArcDeg(rx, ry, rot, theta0, theta1 float32) {
	p.Arc(rx, ry, math32.DegToRad(rot), math32.DegToRad(theta0), math32.DegToRad(theta1))
}

// Close closes a (sub)path with a LineTo to the start of the path
// (the most recent MoveTo command). It also signals the path closes
// as opposed to being just a LineTo command, which can be significant
// for stroking purposes for example.
func (p *Path) Close() {
	if len(*p) == 0 || (*p)[len(*p)-1] == Close {
		// already closed or empty
		return
	} else if (*p)[len(*p)-1] == MoveTo {
		// remove MoveTo + Close
		*p = (*p)[:len(*p)-CmdLen(MoveTo)]
		return
	}

	end := p.StartPos()
	if (*p)[len(*p)-1] == LineTo && Equal((*p)[len(*p)-3], end.X) && Equal((*p)[len(*p)-2], end.Y) {
		// replace LineTo by Close if equal
		(*p)[len(*p)-1] = Close
		(*p)[len(*p)-CmdLen(LineTo)] = Close
		return
	} else if (*p)[len(*p)-1] == LineTo {
		// replace LineTo by Close if equidirectional extension
		start := math32.Vec2((*p)[len(*p)-3], (*p)[len(*p)-2])
		prevStart := math32.Vector2{}
		if CmdLen(LineTo) < len(*p) {
			prevStart = math32.Vec2((*p)[len(*p)-CmdLen(LineTo)-3], (*p)[len(*p)-CmdLen(LineTo)-2])
		}
		if Equal(AngleBetween(end.Sub(start), start.Sub(prevStart)), 0.0) {
			(*p)[len(*p)-CmdLen(LineTo)] = Close
			(*p)[len(*p)-3] = end.X
			(*p)[len(*p)-2] = end.Y
			(*p)[len(*p)-1] = Close
			return
		}
	}
	*p = append(*p, Close, end.X, end.Y, Close)
}
