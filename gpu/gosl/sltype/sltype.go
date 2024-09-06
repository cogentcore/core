// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sltype

import (
	"math"
)

// Uint32Mul64 multiplies two uint32 numbers into a uint64.
func Uint32Mul64(a, b uint32) uint64 {
	return uint64(a) * uint64(b)
}

// Uint64ToLoHi splits a uint64 number into lo and hi uint32 components.
func Uint64ToLoHi(a uint64) Uint32Vec2 {
	var r Uint32Vec2
	r.Y = uint32(a >> 32)
	r.X = uint32(a)
	return r
}

// Uint64FromLoHi combines lo and hi uint32 components into a uint64 value.
func Uint64FromLoHi(a Uint32Vec2) uint64 {
	return uint64(a.X) + uint64(a.Y)<<32
}

// Uint64Add32 adds given uint32 number to given uint64.
func Uint64Add32(a uint64, b uint32) uint64 {
	return a + uint64(b)
}

// Uint64Incr returns increment of the given uint64.
func Uint64Incr(a uint64) uint64 {
	return a + 1
}

// Uint32ToFloat32 converts a uint32 integer into a float32
// in the (0,1) interval (i.e., exclusive of 1).
// This differs from the Go standard by excluding 0, which is handy for passing
// directly to Log function, and from the reference Philox code by excluding 1
// which is in the Go standard and most other standard RNGs.
func Uint32ToFloat32(val uint32) float32 {
	const factor = float32(1.) / (float32(0xffffffff) + float32(1.))
	const halffactor = float32(0.5) * factor
	f := float32(val)*factor + halffactor
	if f == 1 { // exclude 1
		return math.Float32frombits(0x3F7FFFFF)
	}
	return f
}

// Uint32ToFloat32Vec2 converts two uint32 bit integers
// into two corresponding 32 bit f32 values
// in the (0,1) interval (i.e., exclusive of 1).
func Uint32ToFloat32Vec2(val Uint32Vec2) Float32Vec2 {
	var r Float32Vec2
	r.X = Uint32ToFloat32(val.X)
	r.Y = Uint32ToFloat32(val.Y)
	return r
}

// Uint32ToFloat32Range11 converts a uint32 integer into a float32
// in the [-1..1] interval (inclusive of -1 and 1, never identically == 0).
func Uint32ToFloat32Range11(val uint32) float32 {
	const factor = float32(1.) / (float32(0x7fffffff) + float32(1.))
	const halffactor = float32(0.5) * factor
	return (float32(int32(val))*factor + halffactor)
}

// Uint32ToFloat32Range11Vec2 converts two uint32 integers into two float32
// in the [-1,1] interval (inclusive of -1 and 1, never identically == 0)
func Uint32ToFloat32Range11Vec2(val Uint32Vec2) Float32Vec2 {
	var r Float32Vec2
	r.X = Uint32ToFloat32Range11(val.X)
	r.Y = Uint32ToFloat32Range11(val.Y)
	return r
}
