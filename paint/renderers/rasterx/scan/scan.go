// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/scanx:
// Copyright 2018 by the scanx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

// This is the anti-aliasing algorithm from the golang
// translation of FreeType. It has been adapted for use by the scan package
// which replaces the painter interface with the spanner interface.

// Copyright 2010 The Freetype-Go Authors. All rights reserved.
// Use of this source code is governed by your choice of either the
// FreeType License or the GNU General Public License version 2 (or
// any later version), both of which can be found in the LICENSE file.

// Package scan provides an anti-aliasing 2-D rasterizer, which is
// based on the larger Freetype suite of font-related packages, but the
// raster package is not specific to font rasterization, and can be used
// standalone without any other Freetype package.
// Rasterization is done by the same area/coverage accumulation algorithm as
// the Freetype "smooth" module, and the Anti-Grain Geometry library. A
// description of the area/coverage algorithm is at
// http://projects.tuxee.net/cl-vectors/section-the-cl-aa-algorithm
package scan

import (
	"image"
	"math"

	"golang.org/x/image/math/fixed"
)

// Scanner is a refactored version of the freetype scanner
type Scanner struct {
	// If false, the behavior is to use the even-odd winding fill
	// rule during Rasterize.
	UseNonZeroWinding bool

	// The Width of the Rasterizer. The height is implicit in len(cellIndex).
	Width int

	// The current pen position.
	A fixed.Point26_6

	// The current cell and its area/coverage being accumulated.
	Xi, Yi int
	Area   int
	Cover  int

	// The clip bounds of the scanner
	Clip image.Rectangle

	// The saved cells.
	Cell []Cell

	// Linked list of cells, one per row.
	CellIndex []int

	// The spanner that we use.
	Spanner Spanner

	// The bounds.
	MinX, MinY, MaxX, MaxY fixed.Int26_6
}

// SpanFunc is the type of a span function
type SpanFunc func(yi, xi0, xi1 int, alpha uint32)

// A Spanner consumes spans as they are created by the Scanner Draw function
type Spanner interface {
	// SetColor sets the color used for rendering.
	SetColor(color image.Image)

	// This returns a function that is efficient given the Spanner parameters.
	GetSpanFunc() SpanFunc
}

// Cell is part of a linked list (for a given yi co-ordinate) of accumulated
// area/coverage for the pixel at (xi, yi).
type Cell struct {
	Xi    int
	Area  int
	Cover int
	Next  int
}

func (s *Scanner) Set(a fixed.Point26_6) {
	if s.MaxX < a.X {
		s.MaxX = a.X
	}
	if s.MaxY < a.Y {
		s.MaxY = a.Y
	}
	if s.MinX > a.X {
		s.MinX = a.X
	}
	if s.MinY > a.Y {
		s.MinY = a.Y
	}
}

// SetWinding set the winding rule for the polygons
func (s *Scanner) SetWinding(useNonZeroWinding bool) {
	s.UseNonZeroWinding = useNonZeroWinding
}

// SetColor sets the color used for rendering.
func (s *Scanner) SetColor(clr image.Image) {
	s.Spanner.SetColor(clr)
}

// FindCell returns the index in [Scanner.Cell] for the cell corresponding to
// (r.xi, r.yi). The cell is created if necessary.
func (s *Scanner) FindCell() int {
	yi := s.Yi
	if yi < 0 || yi >= len(s.CellIndex) {
		return -1
	}
	xi := s.Xi
	if xi < 0 {
		xi = -1
	} else if xi > s.Width {
		xi = s.Width
	}
	i, prev := s.CellIndex[yi], -1
	for i != -1 && s.Cell[i].Xi <= xi {
		if s.Cell[i].Xi == xi {
			return i
		}
		i, prev = s.Cell[i].Next, i
	}
	c := len(s.Cell)
	s.Cell = append(s.Cell, Cell{xi, 0, 0, i})
	if prev == -1 {
		s.CellIndex[yi] = c
	} else {
		s.Cell[prev].Next = c
	}
	return c
}

// SaveCell saves any accumulated [Scanner.Area] or [Scanner.Cover] for ([Scanner.Xi], [Scanner.Yi]).
func (s *Scanner) SaveCell() {
	if s.Area != 0 || s.Cover != 0 {
		i := s.FindCell()
		if i != -1 {
			s.Cell[i].Area += s.Area
			s.Cell[i].Cover += s.Cover
		}
		s.Area = 0
		s.Cover = 0
	}
}

// SetCell sets the (xi, yi) cell that r is accumulating area/coverage for.
func (s *Scanner) SetCell(xi, yi int) {
	if s.Xi != xi || s.Yi != yi {
		s.SaveCell()
		s.Xi, s.Yi = xi, yi
	}
}

// Scan accumulates area/coverage for the yi'th scanline, going from
// x0 to x1 in the horizontal direction (in 26.6 fixed point co-ordinates)
// and from y0f to y1f fractional vertical units within that scanline.
func (s *Scanner) Scan(yi int, x0, y0f, x1, y1f fixed.Int26_6) {
	// Break the 26.6 fixed point X co-ordinates into integral and fractional parts.
	x0i := int(x0) / 64
	x0f := x0 - fixed.Int26_6(64*x0i)
	x1i := int(x1) / 64
	x1f := x1 - fixed.Int26_6(64*x1i)

	// A perfectly horizontal scan.
	if y0f == y1f {
		s.SetCell(x1i, yi)
		return
	}
	dx, dy := x1-x0, y1f-y0f
	// A single cell scan.
	if x0i == x1i {
		s.Area += int((x0f + x1f) * dy)
		s.Cover += int(dy)
		return
	}
	// There are at least two cells. Apart from the first and last cells,
	// all intermediate cells go through the full width of the cell,
	// or 64 units in 26.6 fixed point format.
	var (
		p, q, edge0, edge1 fixed.Int26_6
		xiDelta            int
	)
	if dx > 0 {
		p, q = (64-x0f)*dy, dx
		edge0, edge1, xiDelta = 0, 64, 1
	} else {
		p, q = x0f*dy, -dx
		edge0, edge1, xiDelta = 64, 0, -1
	}
	yDelta, yRem := p/q, p%q
	if yRem < 0 {
		yDelta--
		yRem += q
	}
	// Do the first cell.
	xi, y := x0i, y0f
	s.Area += int((x0f + edge1) * yDelta)
	s.Cover += int(yDelta)
	xi, y = xi+xiDelta, y+yDelta
	s.SetCell(xi, yi)
	if xi != x1i {
		// Do all the intermediate cells.
		p = 64 * (y1f - y + yDelta)
		fullDelta, fullRem := p/q, p%q
		if fullRem < 0 {
			fullDelta--
			fullRem += q
		}
		yRem -= q
		if (xiDelta > 0 && xi < x1i) || (xiDelta < 0 && xi > x1i) {
			for xi != x1i {
				yDelta = fullDelta
				yRem += fullRem
				if yRem >= 0 {
					yDelta++
					yRem -= q
				}
				s.Area += int(64 * yDelta)
				s.Cover += int(yDelta)
				xi, y = xi+xiDelta, y+yDelta
				s.SetCell(xi, yi)
			}
		}
	}
	// Do the last cell.
	yDelta = y1f - y
	s.Area += int((edge0 + x1f) * yDelta)
	s.Cover += int(yDelta)
}

// Start starts a new path at the given point.
func (s *Scanner) Start(a fixed.Point26_6) {
	s.Set(a)
	s.SetCell(int(a.X/64), int(a.Y/64))
	s.A = a
}

// Line adds a linear segment to the current curve.
func (s *Scanner) Line(b fixed.Point26_6) {
	s.Set(b)
	x0, y0 := s.A.X, s.A.Y
	x1, y1 := b.X, b.Y
	dx, dy := x1-x0, y1-y0
	// Break the 26.6 fixed point Y co-ordinates into integral and fractional
	// parts.
	y0i := int(y0) / 64
	y0f := y0 - fixed.Int26_6(64*y0i)
	y1i := int(y1) / 64
	y1f := y1 - fixed.Int26_6(64*y1i)

	if y0i == y1i {
		// There is only one scanline.
		s.Scan(y0i, x0, y0f, x1, y1f)

	} else if dx == 0 {
		// This is a vertical line segment. We avoid calling r.scan and instead
		// manipulate r.area and r.cover directly.
		var (
			edge0, edge1 fixed.Int26_6
			yiDelta      int
		)
		if dy > 0 {
			edge0, edge1, yiDelta = 0, 64, 1
		} else {
			edge0, edge1, yiDelta = 64, 0, -1
		}
		x0i, yi := int(x0)/64, y0i
		x0fTimes2 := (int(x0) - (64 * x0i)) * 2
		// Do the first pixel.
		dcover := int(edge1 - y0f)
		darea := int(x0fTimes2 * dcover)
		s.Area += darea
		s.Cover += dcover
		yi += yiDelta
		s.SetCell(x0i, yi)
		// Do all the intermediate pixels.
		dcover = int(edge1 - edge0)
		darea = int(x0fTimes2 * dcover)
		if (yiDelta > 0 && yi < y1i) || (yiDelta < 0 && yi > y1i) {
			for yi != y1i {
				s.Area += darea
				s.Cover += dcover
				yi += yiDelta
				s.SetCell(x0i, yi)
			}
		}
		// Do the last pixel.
		dcover = int(y1f - edge0)
		darea = int(x0fTimes2 * dcover)
		s.Area += darea
		s.Cover += dcover

	} else {
		// There are at least two scanlines. Apart from the first and last
		// scanlines, all intermediate scanlines go through the full height of
		// the row, or 64 units in 26.6 fixed point format.
		var (
			p, q, edge0, edge1 fixed.Int26_6
			yiDelta            int
		)
		if dy > 0 {
			p, q = (64-y0f)*dx, dy
			edge0, edge1, yiDelta = 0, 64, 1
		} else {
			p, q = y0f*dx, -dy
			edge0, edge1, yiDelta = 64, 0, -1
		}
		xDelta, xRem := p/q, p%q
		if xRem < 0 {
			xDelta--
			xRem += q
		}
		// Do the first scanline.
		x, yi := x0, y0i
		s.Scan(yi, x, y0f, x+xDelta, edge1)
		x, yi = x+xDelta, yi+yiDelta
		s.SetCell(int(x)/64, yi)
		if yi != y1i {
			// Do all the intermediate scanlines.
			p = 64 * dx
			fullDelta, fullRem := p/q, p%q
			if fullRem < 0 {
				fullDelta--
				fullRem += q
			}
			xRem -= q
			if (yiDelta > 0 && yi < y1i) || (yiDelta < 0 && yi > y1i) {
				for yi != y1i {
					xDelta = fullDelta
					xRem += fullRem
					if xRem >= 0 {
						xDelta++
						xRem -= q
					}
					s.Scan(yi, x, edge0, x+xDelta, edge1)
					x, yi = x+xDelta, yi+yiDelta
					s.SetCell(int(x)/64, yi)
				}
			}
		}
		// Do the last scanline.
		s.Scan(yi, x, edge0, x1, y1f)
	}
	// The next lineTo starts from b.
	s.A = b
}

// AreaToAlpha converts an area value to a uint32 alpha value. A completely
// filled pixel corresponds to an area of 64*64*2, and an alpha of 0xffff. The
// conversion of area values greater than this depends on the winding rule:
// even-odd or non-zero.
func (s *Scanner) AreaToAlpha(area int) uint32 {
	// The C Freetype implementation (version 2.3.12) does "alpha := area>>1"
	// without the +1. Round-to-nearest gives a more symmetric result than
	// round-down. The C implementation also returns 8-bit alpha, not 16-bit
	// alpha.
	a := (area + 1) >> 1
	if a < 0 {
		a = -a
	}
	alpha := uint32(a)
	if s.UseNonZeroWinding {
		if alpha > 0x0fff {
			alpha = 0x0fff
		}
	} else {
		alpha &= 0x1fff
		if alpha > 0x1000 {
			alpha = 0x2000 - alpha
		} else if alpha == 0x1000 {
			alpha = 0x0fff
		}
	}
	// alpha is now in the range [0x0000, 0x0fff]. Convert that 12-bit alpha to
	// 16-bit alpha.
	return alpha<<4 | alpha>>8
}

// Draw converts r's accumulated curves into Spans for p. The Spans passed
// to the spanner are non-overlapping, and sorted by Y and then X. They all have non-zero
// width (and 0 <= X0 < X1 <= r.width) and non-zero A, except for the final
// Span, which has Y, X0, X1 and A all equal to zero.
func (s *Scanner) Draw() {
	b := image.Rect(0, 0, s.Width, len(s.CellIndex))
	if s.Clip.Dx() != 0 && s.Clip.Dy() != 0 {
		b = b.Intersect(s.Clip)
	}
	s.SaveCell()
	span := s.Spanner.GetSpanFunc()
	for yi := b.Min.Y; yi < b.Max.Y; yi++ {
		xi, cover := 0, 0
		for c := s.CellIndex[yi]; c != -1; c = s.Cell[c].Next {
			if cover != 0 && s.Cell[c].Xi > xi {
				alpha := s.AreaToAlpha(cover * 64 * 2)
				if alpha != 0 {
					xi0, xi1 := xi, s.Cell[c].Xi
					if xi0 < b.Min.X {
						xi0 = b.Min.X
					}
					if xi1 > b.Max.X {
						xi1 = b.Max.X
					}
					if xi0 < xi1 {
						span(yi, xi0, xi1, alpha)
					}
				}
			}
			cover += s.Cell[c].Cover
			alpha := s.AreaToAlpha(cover*64*2 - s.Cell[c].Area)
			xi = s.Cell[c].Xi + 1
			if alpha != 0 {
				xi0, xi1 := s.Cell[c].Xi, xi
				if xi0 < b.Min.X {
					xi0 = b.Min.X
				}
				if xi1 > b.Max.X {
					xi1 = b.Max.X
				}
				if xi0 < xi1 {
					span(yi, xi0, xi1, alpha)
				}
			}
		}
	}
}

// GetPathExtent returns the bounds of the accumulated path extent
func (s *Scanner) GetPathExtent() fixed.Rectangle26_6 {
	return fixed.Rectangle26_6{
		Min: fixed.Point26_6{X: s.MinX, Y: s.MinY},
		Max: fixed.Point26_6{X: s.MaxX, Y: s.MaxY}}
}

// Clear cancels any previous accumulated scans
func (s *Scanner) Clear() {
	s.A = fixed.Point26_6{}
	s.Xi = 0
	s.Yi = 0
	s.Area = 0
	s.Cover = 0
	s.Cell = s.Cell[:0]
	for i := 0; i < len(s.CellIndex); i++ {
		s.CellIndex[i] = -1
	}
	const mxfi = fixed.Int26_6(math.MaxInt32)
	s.MinX, s.MinY, s.MaxX, s.MaxY = mxfi, mxfi, -mxfi, -mxfi
}

// SetBounds sets the maximum width and height of the rasterized image and
// calls Clear. The width and height are in pixels, not fixed.Int26_6 units.
func (s *Scanner) SetBounds(width, height int) {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	s.Width = width
	s.Cell = s.Cell[:0]
	if height > cap(s.CellIndex) {
		s.CellIndex = make([]int, height)
	}
	// Make sure length of cellIndex = height
	s.CellIndex = s.CellIndex[0:height]
	s.Width = width
	s.Clear()
}

// NewScanner creates a new Scanner with the given bounds.
func NewScanner(xs Spanner, width, height int) (sc *Scanner) {
	sc = &Scanner{Spanner: xs, UseNonZeroWinding: true}
	sc.SetBounds(width, height)
	return
}

// SetClip will not affect accumulation of scans, but it will
// clip drawing of the spans int the Draw func by the clip rectangle.
func (s *Scanner) SetClip(r image.Rectangle) {
	s.Clip = r
}
