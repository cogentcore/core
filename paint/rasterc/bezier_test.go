// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/rasterx:
// Copyright 2018 by the rasterx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

package raster

import (
	"math/rand"
	"testing"
	"time"

	"cogentcore.org/core/math32"
)

// Copied from golang.org/x/image/vector
func lerp(t, px, py, qx, qy float32) (x, y float32) {
	return px + t*(qx-px), py + t*(qy-py)
}

// CubeLerpTo and adapted from golang.org/x/image/vector
//
//	adds a cubic Bézier segment, from the pen via (bx, by) and (cx, cy)
//
// to (dx, dy), and moves the pen to (dx, dy).
//
// The coordinates are allowed to be out of the Rasterizer's bounds.
func CubeLerpTo(ax, ay, bx, by, cx, cy, dx, dy float32, LineTo func(ex, ey float32)) {
	devsq := DevSquared(ax, ay, bx, by, dx, dy)
	if devsqAlt := DevSquared(ax, ay, cx, cy, dx, dy); devsq < devsqAlt {
		devsq = devsqAlt
	}
	if devsq >= 0.333 {
		const tol = 3
		n := 1 + int(math32.Sqrt(math32.Sqrt(tol*float32(devsq))))
		t, nInv := float32(0), 1/float32(n)
		for i := 0; i < n-1; i++ {
			t += nInv
			abx, aby := lerp(t, ax, ay, bx, by)
			bcx, bcy := lerp(t, bx, by, cx, cy)
			cdx, cdy := lerp(t, cx, cy, dx, dy)
			abcx, abcy := lerp(t, abx, aby, bcx, bcy)
			bcdx, bcdy := lerp(t, bcx, bcy, cdx, cdy)
			LineTo(lerp(t, abcx, abcy, bcdx, bcdy))
		}
	}
	LineTo(dx, dy)
}

// QuadLerpTo and adapted from golang.org/x/image/vector
func QuadLerpTo(ax, ay, bx, by, cx, cy float32, LineTo func(dx, dy float32)) {
	devsq := DevSquared(ax, ay, bx, by, cx, cy)
	if devsq >= 0.333 {
		const tol = 3
		n := 1 + int(math32.Sqrt(math32.Sqrt(tol*float32(devsq))))
		t, nInv := float32(0), 1/float32(n)
		for i := 0; i < n-1; i++ {
			t += nInv
			abx, aby := lerp(t, ax, ay, bx, by)
			bcx, bcy := lerp(t, bx, by, cx, cy)
			LineTo(lerp(t, abx, aby, bcx, bcy))
		}
	}
	LineTo(cx, cy)
}

var tc = []float32{ //test coorinates
	146.53, 229.95,
	115.55, 209.55,
	146.53, 229.95,
	115.55, 209.55,
	102.50, 211.00,
	95.38, 211.00,
	56.09, 211.00,
	31.17, 182.33}

var fnc = func(ex, ey float32) {}

func BenchmarkBezierQuadLerp(b *testing.B) {
	for i := 0; i < b.N; i++ {
		QuadLerpTo(tc[0], tc[1], tc[2], tc[3], tc[4], tc[5],
			fnc)
	}
}

func BenchmarkBezierQuad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		QuadTo(tc[0], tc[1], tc[2], tc[3], tc[4], tc[5],
			fnc)
	}
}

func BenchmarkBezierCubeLerp(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CubeLerpTo(tc[0], tc[1], tc[2], tc[3], tc[4], tc[5], tc[6], tc[7],
			fnc)
	}
}

func BenchmarkBezierCube(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CubeTo(tc[0], tc[1], tc[2], tc[3], tc[4], tc[5], tc[6], tc[7],
			fnc)
	}
}

func TestBezierCube(t *testing.T) {
	rnd := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
	tests := 50
	var coords []float32
	for i := 0; i < tests*8; i++ {
		coords = append(coords, float32(rnd.Intn(100)))
	}
	epsilon := float32(1e-4) // allowed range for round off error
	for i := 0; i < tests; i++ {
		var r1x, r1y, r2x, r2y []float32
		set := coords[i*8 : (i+1)*8]
		CubeLerpTo(set[0], set[1], set[2], set[3], set[4], set[5], set[6], set[7],
			func(ex, ey float32) {
				r1x = append(r1x, ex)
				r1y = append(r1y, ey)
			})
		CubeTo(set[0], set[1], set[2], set[3], set[4], set[5], set[6], set[7],
			func(ex, ey float32) {
				r2x = append(r2x, ex)
				r2y = append(r2y, ey)
			})
		if len(r1x) != len(r2x) {
			t.Error("x len mismatch", len(r1x), len(r2x))
		}
		if len(r1y) != len(r2y) {
			t.Error("y len mismatch")
		}
		//t.Log("Bez to", len(r1x), "lines")
		for i, v := range r1x {
			if math32.Abs(float32(v-r2x[i])) > epsilon {
				t.Error("x mismatch", v, "vs", r2x[i], " diff ", v-r2x[i])
			}
		}
		for i, v := range r1y {
			if math32.Abs(float32(v-r2y[i])) > epsilon {
				t.Error("y mismatch", v, "vs", r2y[i], " diff ", v-r2y[i])
			}
		}
	}
}

func TestBezierQuad(t *testing.T) {
	rnd := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
	tests := 50
	var coords []float32
	for i := 0; i < tests*8; i++ {
		coords = append(coords, float32(rnd.Intn(100)))
	}
	epsilon := float32(1e-4) // allowed range for round off error
	for i := 0; i < tests; i++ {
		var r1x, r1y, r2x, r2y []float32
		set := coords[i*6 : (i+1)*6]
		QuadLerpTo(set[0], set[1], set[2], set[3], set[4], set[5],
			func(ex, ey float32) {
				r1x = append(r1x, ex)
				r1y = append(r1y, ey)
			})
		QuadTo(set[0], set[1], set[2], set[3], set[4], set[5],
			func(ex, ey float32) {
				r2x = append(r2x, ex)
				r2y = append(r2y, ey)
			})
		if len(r1x) != len(r2x) {
			t.Error("x len mismatch", len(r1x), len(r2x))
		}
		if len(r1y) != len(r2y) {
			t.Error("y len mismatch")
		}
		//t.Log("Bez to", len(r1x), "lines")
		for i, v := range r1x {
			if math32.Abs(float32(v-r2x[i])) > epsilon {
				t.Error("x mismatch", v, "vs", r2x[i], " diff ", v-r2x[i])
			}
		}
		for i, v := range r1y {
			if math32.Abs(float32(v-r2y[i])) > epsilon {
				t.Error("y mismatch", v, "vs", r2y[i], " diff ", v-r2y[i])
			}
		}

	}
}
