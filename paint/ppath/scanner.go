// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package ppath

import (
	"cogentcore.org/core/math32"
)

// Scanner returns a path scanner.
func (p Path) Scanner() *Scanner {
	return &Scanner{p, -1}
}

// ReverseScanner returns a path scanner in reverse order.
func (p Path) ReverseScanner() ReverseScanner {
	return ReverseScanner{p, len(p)}
}

// Scanner scans the path.
type Scanner struct {
	p Path
	i int // i is at the end of the current command
}

// Scan scans a new path segment and should be called before the other methods.
func (s *Scanner) Scan() bool {
	if s.i+1 < len(s.p) {
		s.i += CmdLen(s.p[s.i+1])
		return true
	}
	return false
}

// Cmd returns the current path segment command.
func (s *Scanner) Cmd() float32 {
	return s.p[s.i]
}

// Index returns the index in path of the current command.
func (s *Scanner) Index() int {
	return (s.i - CmdLen(s.p[s.i])) + 1
}

// Values returns the current path segment values.
func (s *Scanner) Values() []float32 {
	return s.p[s.i-CmdLen(s.p[s.i])+2 : s.i]
}

// Start returns the current path segment start position.
func (s *Scanner) Start() math32.Vector2 {
	i := s.i - CmdLen(s.p[s.i])
	if i == -1 {
		return math32.Vector2{}
	}
	return math32.Vector2{s.p[i-2], s.p[i-1]}
}

// CP1 returns the first control point for quadratic and cubic Béziers.
func (s *Scanner) CP1() math32.Vector2 {
	if s.p[s.i] != QuadTo && s.p[s.i] != CubeTo {
		panic("must be quadratic or cubic Bézier")
	}
	i := s.i - CmdLen(s.p[s.i]) + 1
	return math32.Vector2{s.p[i+1], s.p[i+2]}
}

// CP2 returns the second control point for cubic Béziers.
func (s *Scanner) CP2() math32.Vector2 {
	if s.p[s.i] != CubeTo {
		panic("must be cubic Bézier")
	}
	i := s.i - CmdLen(s.p[s.i]) + 1
	return math32.Vector2{s.p[i+3], s.p[i+4]}
}

// Arc returns the arguments for arcs (rx,ry,rot,large,sweep).
func (s *Scanner) Arc() (float32, float32, float32, bool, bool) {
	if s.p[s.i] != ArcTo {
		panic("must be arc")
	}
	i := s.i - CmdLen(s.p[s.i]) + 1
	large, sweep := ToArcFlags(s.p[i+4])
	return s.p[i+1], s.p[i+2], s.p[i+3], large, sweep
}

// End returns the current path segment end position.
func (s *Scanner) End() math32.Vector2 {
	return math32.Vector2{s.p[s.i-2], s.p[s.i-1]}
}

// Path returns the current path segment.
func (s *Scanner) Path() Path {
	p := Path{}
	p.MoveTo(s.Start().X, s.Start().Y)
	switch s.Cmd() {
	case LineTo:
		p.LineTo(s.End().X, s.End().Y)
	case QuadTo:
		p.QuadTo(s.CP1().X, s.CP1().Y, s.End().X, s.End().Y)
	case CubeTo:
		p.CubeTo(s.CP1().X, s.CP1().Y, s.CP2().X, s.CP2().Y, s.End().X, s.End().Y)
	case ArcTo:
		rx, ry, rot, large, sweep := s.Arc()
		p.ArcTo(rx, ry, rot, large, sweep, s.End().X, s.End().Y)
	}
	return p
}

// ReverseScanner scans the path in reverse order.
type ReverseScanner struct {
	p Path
	i int
}

// Scan scans a new path segment and should be called before the other methods.
func (s *ReverseScanner) Scan() bool {
	if 0 < s.i {
		s.i -= CmdLen(s.p[s.i-1])
		return true
	}
	return false
}

// Cmd returns the current path segment command.
func (s *ReverseScanner) Cmd() float32 {
	return s.p[s.i]
}

// Values returns the current path segment values.
func (s *ReverseScanner) Values() []float32 {
	return s.p[s.i+1 : s.i+CmdLen(s.p[s.i])-1]
}

// Start returns the current path segment start position.
func (s *ReverseScanner) Start() math32.Vector2 {
	if s.i == 0 {
		return math32.Vector2{}
	}
	return math32.Vector2{s.p[s.i-3], s.p[s.i-2]}
}

// CP1 returns the first control point for quadratic and cubic Béziers.
func (s *ReverseScanner) CP1() math32.Vector2 {
	if s.p[s.i] != QuadTo && s.p[s.i] != CubeTo {
		panic("must be quadratic or cubic Bézier")
	}
	return math32.Vector2{s.p[s.i+1], s.p[s.i+2]}
}

// CP2 returns the second control point for cubic Béziers.
func (s *ReverseScanner) CP2() math32.Vector2 {
	if s.p[s.i] != CubeTo {
		panic("must be cubic Bézier")
	}
	return math32.Vector2{s.p[s.i+3], s.p[s.i+4]}
}

// Arc returns the arguments for arcs (rx,ry,rot,large,sweep).
func (s *ReverseScanner) Arc() (float32, float32, float32, bool, bool) {
	if s.p[s.i] != ArcTo {
		panic("must be arc")
	}
	large, sweep := ToArcFlags(s.p[s.i+4])
	return s.p[s.i+1], s.p[s.i+2], s.p[s.i+3], large, sweep
}

// End returns the current path segment end position.
func (s *ReverseScanner) End() math32.Vector2 {
	i := s.i + CmdLen(s.p[s.i])
	return math32.Vector2{s.p[i-3], s.p[i-2]}
}

// Path returns the current path segment.
func (s *ReverseScanner) Path() Path {
	p := Path{}
	p.MoveTo(s.Start().X, s.Start().Y)
	switch s.Cmd() {
	case LineTo:
		p.LineTo(s.End().X, s.End().Y)
	case QuadTo:
		p.QuadTo(s.CP1().X, s.CP1().Y, s.End().X, s.End().Y)
	case CubeTo:
		p.CubeTo(s.CP1().X, s.CP1().Y, s.CP2().X, s.CP2().Y, s.End().X, s.End().Y)
	case ArcTo:
		rx, ry, rot, large, sweep := s.Arc()
		p.ArcTo(rx, ry, rot, large, sweep, s.End().X, s.End().Y)
	}
	return p
}
