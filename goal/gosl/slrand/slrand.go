// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slrand

import (
	"cogentcore.org/core/goal/gosl/sltype"
	"cogentcore.org/core/math32"
)

// These are Go versions of the same Philox2x32 based random number generator
// functions available in .WGSL.

// Philox2x32round does one round of updating of the counter.
func Philox2x32round(counter uint64, key uint32) uint64 {
	ctr := sltype.Uint64ToLoHi(counter)
	mul := sltype.Uint64ToLoHi(sltype.Uint32Mul64(0xD256D193, ctr.X))
	ctr.X = mul.Y ^ key ^ ctr.Y
	ctr.Y = mul.X
	return sltype.Uint64FromLoHi(ctr)
}

// Philox2x32bumpkey does one round of updating of the key
func Philox2x32bumpkey(key uint32) uint32 {
	return key + 0x9E3779B9
}

// Philox2x32 implements the stateless counter-based RNG algorithm
// returning a random number as two uint32 values, given a
// counter and key input that determine the result.
func Philox2x32(counter uint64, key uint32) sltype.Uint32Vec2 {
	// this is an unrolled loop of 10 updates based on initial counter and key,
	// which produces the random deviation deterministically based on these inputs.
	counter = Philox2x32round(counter, key) // 1
	key = Philox2x32bumpkey(key)
	counter = Philox2x32round(counter, key) // 2
	key = Philox2x32bumpkey(key)
	counter = Philox2x32round(counter, key) // 3
	key = Philox2x32bumpkey(key)
	counter = Philox2x32round(counter, key) // 4
	key = Philox2x32bumpkey(key)
	counter = Philox2x32round(counter, key) // 5
	key = Philox2x32bumpkey(key)
	counter = Philox2x32round(counter, key) // 6
	key = Philox2x32bumpkey(key)
	counter = Philox2x32round(counter, key) // 7
	key = Philox2x32bumpkey(key)
	counter = Philox2x32round(counter, key) // 8
	key = Philox2x32bumpkey(key)
	counter = Philox2x32round(counter, key) // 9
	key = Philox2x32bumpkey(key)

	return sltype.Uint64ToLoHi(Philox2x32round(counter, key)) // 10
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

// Uint32Vec2 returns two uniformly distributed 32 unsigned integers,
// based on given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
func Uint32Vec2(counter uint64, funcIndex uint32, key uint32) sltype.Uint32Vec2 {
	return Philox2x32(sltype.Uint64Add32(counter, funcIndex), key)
}

// Uint32 returns a uniformly distributed 32 unsigned integer,
// based on given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
func Uint32(counter uint64, funcIndex uint32, key uint32) uint32 {
	return Philox2x32(sltype.Uint64Add32(counter, funcIndex), key).X
}

// Float32Vec2 returns two uniformly distributed float32 values in range (0,1),
// based on given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
func Float32Vec2(counter uint64, funcIndex uint32, key uint32) sltype.Float32Vec2 {
	return sltype.Uint32ToFloat32Vec2(Uint32Vec2(counter, funcIndex, key))
}

// Float32 returns a uniformly distributed float32 value in range (0,1),
// based on given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
func Float32(counter uint64, funcIndex uint32, key uint32) float32 {
	return sltype.Uint32ToFloat32(Uint32(counter, funcIndex, key))
}

// Float32Range11Vec2 returns two uniformly distributed float32 values in range [-1,1],
// based on given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
func Float32Range11Vec2(counter uint64, funcIndex uint32, key uint32) sltype.Float32Vec2 {
	return sltype.Uint32ToFloat32Vec2(Uint32Vec2(counter, funcIndex, key))
}

// Float32Range11 returns a uniformly distributed float32 value in range [-1,1],
// based on given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
func Float32Range11(counter uint64, funcIndex uint32, key uint32) float32 {
	return sltype.Uint32ToFloat32Range11(Uint32(counter, funcIndex, key))
}

// BoolP returns a bool true value with probability p
// based on given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
func BoolP(counter uint64, funcIndex uint32, key uint32, p float32) bool {
	return (Float32(counter, funcIndex, key) < p)
}

func SincosPi(x float32) sltype.Float32Vec2 {
	const PIf = 3.1415926535897932
	var r sltype.Float32Vec2
	r.Y, r.X = math32.Sincos(PIf * x)
	return r
}

// Float32NormVec2 returns two random float32 numbers
// distributed according to the normal, Gaussian distribution
// with zero mean and unit variance.
// This is done very efficiently using the Box-Muller algorithm
// that consumes two random 32 bit uint values.
// Uses given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
func Float32NormVec2(counter uint64, funcIndex uint32, key uint32) sltype.Float32Vec2 {
	ur := Uint32Vec2(counter, funcIndex, key)
	f := SincosPi(sltype.Uint32ToFloat32Range11(ur.X))
	r := math32.Sqrt(-2.0 * math32.Log(sltype.Uint32ToFloat32(ur.Y))) // guaranteed to avoid 0.
	return f.MulScalar(r)
}

// Float32Norm returns a random float32 number
// distributed according to the normal, Gaussian distribution
// with zero mean and unit variance.
// Uses given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
func Float32Norm(counter uint64, funcIndex uint32, key uint32) float32 {
	return Float32Vec2(counter, funcIndex, key).X
}

// Uint32N returns a uint32 in the range [0,N).
// Uses given global shared counter, function index offset from that
// counter for this specific random number call, and key as unique
// index of the item being processed.
func Uint32N(counter uint64, funcIndex uint32, key uint32, n uint32) uint32 {
	v := Float32(counter, funcIndex, key)
	return uint32(v * float32(n))
}

// Counter is used for storing the random counter using aligned 16 byte storage,
// with convenience methods for typical use cases.
// It retains a copy of the last Seed value, which is applied to the Hi uint32 value.
type Counter struct {
	// Counter value
	Counter uint64

	// last seed value set by Seed method, restored by Reset()
	HiSeed uint32

	pad uint32
}

// Reset resets counter to last set Seed state
func (ct *Counter) Reset() {
	ct.Counter = sltype.Uint64FromLoHi(sltype.Uint32Vec2{0, ct.HiSeed})
}

// Seed sets the Hi uint32 value from given seed, saving it in HiSeed field.
// Each increment in seed generates a unique sequence of over 4 billion numbers,
// so it is reasonable to just use incremental values there, but more widely
// spaced numbers will result in longer unique sequences.
// Resets Lo to 0.
// This same seed will be restored during Reset
func (ct *Counter) Seed(seed uint32) {
	ct.HiSeed = seed
	ct.Reset()
}

// Add increments the counter by given amount.
// Call this after completing a pass of computation
// where the value passed here is the max of funcIndex+1
// used for any possible random calls during that pass.
func (ct *Counter) Add(inc uint32) {
	ct.Counter = sltype.Uint64Add32(ct.Counter, inc)
}
