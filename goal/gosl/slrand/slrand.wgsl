// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Original file is in Go package: github.com/cogentcore/core/gpu/gosl/slrand
// See README.md there for documentation.

// These random number generation (RNG) functions are optimized for
// use on the GPU, with equivalent Go versions available in slrand.go.
// This is using the Philox2x32 counter-based RNG.

// #include "sltype.wgsl"

// Philox2x32round does one round of updating of the counter.
fn Philox2x32round(counter: su64, key: u32) -> su64 {
	let mul = Uint32Mul64(u32(0xD256D193), counter.x);
	var ctr: su64;
	ctr.x = mul.y ^ key ^ counter.y;
	ctr.y = mul.x;
	return ctr;
}

// Philox2x32bumpkey does one round of updating of the key
fn Philox2x32bumpkey(key: u32) -> u32 {
	return key + u32(0x9E3779B9);
}

// Philox2x32 implements the stateless counter-based RNG algorithm
// returning a random number as two uint32 values, given a
// counter and key input that determine the result.
// The input counter is not modified.
fn Philox2x32(counter: su64, key: u32) -> vec2<u32> {
	// this is an unrolled loop of 10 updates based on initial counter and key,
	// which produces the random deviation deterministically based on these inputs.
	var ctr = Philox2x32round(counter, key); // 1
	var ky = Philox2x32bumpkey(key);
	ctr = Philox2x32round(ctr, ky); // 2
	ky = Philox2x32bumpkey(ky);
	ctr = Philox2x32round(ctr, ky); // 3
	ky = Philox2x32bumpkey(ky);
	ctr = Philox2x32round(ctr, ky); // 4
	ky = Philox2x32bumpkey(ky);
	ctr = Philox2x32round(ctr, ky); // 5
	ky = Philox2x32bumpkey(ky);
	ctr = Philox2x32round(ctr, ky); // 6
	ky = Philox2x32bumpkey(ky);
	ctr = Philox2x32round(ctr, ky); // 7
	ky = Philox2x32bumpkey(ky);
	ctr = Philox2x32round(ctr, ky); // 8
	ky = Philox2x32bumpkey(ky);
	ctr = Philox2x32round(ctr, ky); // 9
	ky = Philox2x32bumpkey(ky);
	
	return Philox2x32round(ctr, ky); // 10
}

////////////////////////////////////////////////////////////
// Methods below provide a standard interface  with more
// readable names, mapping onto the Go rand methods.
//
// They assume a global shared counter, which is then
// incremented by a function index, defined for each function
// consuming random numbers that _could_ be called within a parallel
// processing loop.  At the end of the loop, the global counter should
// be incremented by the total possible number of such functions.
// This results in fully resproducible results, invariant to
// specific processing order, and invariant to whether any one function
// actually calls the random number generator.

// RandUint32Vec2 returns two uniformly distributed 32 unsigned integers,
// based on given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
fn RandUint32Vec2(counter: su64, funcIndex: u32, key: u32) -> vec2<u32> {
	return Philox2x32(Uint64Add32(counter, funcIndex), key);
}

// RandUint32 returns a uniformly distributed 32 unsigned integer,
// based on given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
fn RandUint32(counter: su64, funcIndex: u32, key: u32) -> u32 {
	return Philox2x32(Uint64Add32(counter, funcIndex), key).x;
}

// RandFloat32Vec2 returns two uniformly distributed float32 values in range (0,1),
// based on given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
fn RandFloat32Vec2(counter: su64, funcIndex: u32, key: u32) -> vec2<f32> {
	return Uint32ToFloat32Vec2(RandUint32Vec2(counter, funcIndex, key));
}

// RandFloat32 returns a uniformly distributed float32 value in range (0,1),
// based on given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
fn RandFloat32(counter: su64, funcIndex: u32, key: u32) -> f32 { 
	return Uint32ToFloat32(RandUint32(counter, funcIndex, key));
}

// RandFloat32Range11Vec2 returns two uniformly distributed float32 values in range [-1,1],
// based on given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
fn RandFloat32Range11Vec2(counter: su64, funcIndex: u32, key: u32) -> vec2<f32> {
	return Uint32ToFloat32Vec2(RandUint32Vec2(counter, funcIndex, key));
}

// RandFloat32Range11 returns a uniformly distributed float32 value in range [-1,1],
// based on given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
fn RandFloat32Range11(counter: su64, funcIndex: u32, key: u32) -> f32 { 
	return Uint32ToFloat32Range11(RandUint32(counter, funcIndex, key));
}

// RandBoolP returns a bool true value with probability p
// based on given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
fn RandBoolP(counter: su64, funcIndex: u32, key: u32, p: f32) -> bool { 
	return (RandFloat32(counter, funcIndex, key) < p);
}

fn sincospi(x: f32) -> vec2<f32> {
	let PIf = 3.1415926535897932;
	var r: vec2<f32>;
	r.x = cos(PIf*x);
	r.y = sin(PIf*x);
	return r;
}

// RandFloat32NormVec2 returns two random float32 numbers
// distributed according to the normal, Gaussian distribution
// with zero mean and unit variance.
// This is done very efficiently using the Box-Muller algorithm
// that consumes two random 32 bit uint values.
// Uses given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
fn RandFloat32NormVec2(counter: su64, funcIndex: u32, key: u32) -> vec2<f32> { 
	let ur = RandUint32Vec2(counter, funcIndex, key);
	var f = sincospi(Uint32ToFloat32Range11(ur.x));
	let r = sqrt(-2.0 * log(Uint32ToFloat32(ur.y))); // guaranteed to avoid 0.
	return f * r;
}

// RandFloat32Norm returns a random float32 number
// distributed according to the normal, Gaussian distribution
// with zero mean and unit variance.
// Uses given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
fn RandFloat32Norm(counter: su64, funcIndex: u32, key: u32) -> f32 { 
	return RandFloat32Vec2(counter, funcIndex, key).x;
}

// RandUint32N returns a uint32 in the range [0,N).
// Uses given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
fn RandUint32N(counter: su64, funcIndex: u32, key: u32, n: u32) -> u32 { 
	let v = RandFloat32(counter, funcIndex, key);
	return u32(v * f32(n));
}

// Counter is used for storing the random counter using aligned 16 byte
// storage, with convenience functions  for typical use cases.
// It retains a copy of the last Seed value, which is applied to
// the Hi uint32 value.
struct RandCounter {
	Counter: su64,
	HiSeed: u32,
	pad: u32,
}
	
// Reset resets counter to last set Seed state.
fn RandCounter_Reset(ct: ptr<function,RandCounter>) {
	(*ct).Counter.x = u32(0);
	(*ct).Counter.y = (*ct).HiSeed;
}
	
// Seed sets the Hi uint32 value from given seed, saving it in Seed field.
// Each increment in seed generates a unique sequence of over 4 billion numbers,
// so it is reasonable to just use incremental values there, but more widely
// spaced numbers will result in longer unique sequences.
// Resets Lo to 0.
// This same seed will be restored during Reset
fn RandCounter_Seed(ct: ptr<function,RandCounter>, seed: u32) {
	(*ct).HiSeed = seed;
	RandCounter_Reset(ct);
}

// Add increments the counter by given amount.
// Call this after completing a pass of computation
// where the value passed here is the max of funcIndex+1
// used for any possible random calls during that pass.
fn RandCounter_Add(ct: ptr<function,RandCounter>, inc: u32) {
	(*ct).Counter = Uint64Add32((*ct).Counter, inc);
}
