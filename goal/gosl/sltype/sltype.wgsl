// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Original file is in Go package: github.com/cogentcore/core/gpu/gosl/sltype
// See README.md there for documentation.

// This file emulates uint64 (u64) using 2 uint32 integers.
// and defines conversions between uint and float.

// define a u64 type as an alias.
// if / when u64 actually happens, will make it easier to update.
alias su64 = vec2<u32>;

// Uint32Mul64 multiplies two uint32 numbers into a uint64 (using vec2<u32>).
fn Uint32Mul64(a: u32, b: u32) -> su64 {
	let LOMASK = (((u32(1))<<16)-1);
	var r: su64;
	r.x = a * b;               /* full low multiply */
	let ahi = a >> 16;
	let alo = a & LOMASK;
	let bhi = b >> 16;
	let blo = b & LOMASK;
    
	let ahbl = ahi * blo;
	let albh = alo * bhi;

	let ahbl_albh = ((ahbl&LOMASK) + (albh&LOMASK));
	var hit = ahi*bhi + (ahbl>>16) +  (albh>>16);
	hit += ahbl_albh >> 16; /* carry from the sum of lo(ahbl) + lo(albh) ) */
	/* carry from the sum with alo*blo */
	if ((r.x >> u32(16)) < (ahbl_albh&LOMASK)) {
		hit += u32(1);
	}
	r.y = hit; 
	return r;
}

/*
// Uint32Mul64 multiplies two uint32 numbers into a uint64 (using su64).
fn Uint32Mul64(a: u32, b: u32) -> su64 {
	return su64(a) * su64(b);
}
*/


// Uint64Add32 adds given uint32 number to given uint64 (using vec2<u32>).
fn Uint64Add32(a: su64, b: u32) -> su64 {
	if (b == 0) {
		return a;
	}
	var s = a;
	if (s.x > u32(0xffffffff) - b) {
		s.y++;
		s.x = (b - 1) - (u32(0xffffffff) - s.x);
	} else {
		s.x += b;
	}
	return s;
}

// Uint64Incr returns increment of the given uint64 (using vec2<u32>).
fn Uint64Incr(a: su64) -> su64 {
	var s = a;
	if(s.x == 0xffffffff) {
		s.y++;
		s.x = u32(0);
	} else {
		s.x++;
	}
	return s;
}

// Uint32ToFloat32 converts a uint32 integer into a float32
// in the (0,1) interval (i.e., exclusive of 1).
// This differs from the Go standard by excluding 0, which is handy for passing
// directly to Log function, and from the reference Philox code by excluding 1
// which is in the Go standard and most other standard RNGs.
fn Uint32ToFloat32(val: u32) -> f32 {
	let factor = f32(1.0) / (f32(u32(0xffffffff)) + f32(1.0));
	let halffactor = f32(0.5) * factor;
	var f = f32(val) * factor + halffactor;
	if (f == 1.0) { // exclude 1
		return bitcast<f32>(0x3F7FFFFF);
	}
	return f;
}

// note: there is no overloading of user-defined functions
// https://github.com/gpuweb/gpuweb/issues/876

// Uint32ToFloat32Vec2 converts two uint 32 bit integers
// into two corresponding 32 bit f32 values 
// in the (0,1) interval (i.e., exclusive of 1).
fn Uint32ToFloat32Vec2(val: vec2<u32>) -> vec2<f32> {
	var r: vec2<f32>;
	r.x = Uint32ToFloat32(val.x);
	r.y = Uint32ToFloat32(val.y);
	return r;
}

// Uint32ToFloat32Range11 converts a uint32 integer into a float32
// in the [-1..1] interval (inclusive of -1 and 1, never identically == 0).
fn Uint32ToFloat32Range11(val: u32) -> f32 {
	let factor = f32(1.0) / (f32(i32(0x7fffffff)) + f32(1.0));
	let halffactor = f32(0.5) * factor;
	return (f32(val) * factor + halffactor);
}

// Uint32ToFloat32Range11Vec2 converts two uint32 integers into two float32
// in the [-1,1] interval (inclusive of -1 and 1, never identically == 0).
fn Uint32ToFloat32Range11Vec2(val: vec2<u32>) -> vec2<f32> {
	var r: vec2<f32>;
	r.x = Uint32ToFloat32Range11(val.x);
	r.y = Uint32ToFloat32Range11(val.y);
	return r;
}


