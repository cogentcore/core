// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slrand

import (
	"math"

	"cogentcore.org/core/math32"
	"github.com/emer/gosl/v2/sltype"
)

// These are Go versions of the same Philox2x32 based random number generator
// functions available in .HLSL.

// MulHiLo64 is the fast, simpler version when 64 bit uints become available
func MulHiLo64(a, b uint32) (lo, hi uint32) {
	prod := uint64(a) * uint64(b)
	hi = uint32(prod >> 32)
	lo = uint32(prod)
	return
}

// Philox2x32round does one round of updating of the counter
func Philox2x32round(counter *sltype.Uint2, key uint32) {
	lo, hi := MulHiLo64(0xD256D193, counter.X)
	counter.X = hi ^ key ^ counter.Y
	counter.Y = lo
}

// Philox2x32bumpkey does one round of updating of the key
func Philox2x32bumpkey(key *uint32) {
	*key += 0x9E3779B9
}

// Philox2x32 implements the stateless counter-based RNG algorithm
// returning a random number as 2 uint3232 32 bit values, given a
// counter and key input that determine the result.
func Philox2x32(counter sltype.Uint2, key uint32) sltype.Uint2 {
	Philox2x32round(&counter, key) // 1
	Philox2x32bumpkey(&key)
	Philox2x32round(&counter, key) // 2
	Philox2x32bumpkey(&key)
	Philox2x32round(&counter, key) // 3
	Philox2x32bumpkey(&key)
	Philox2x32round(&counter, key) // 4
	Philox2x32bumpkey(&key)
	Philox2x32round(&counter, key) // 5
	Philox2x32bumpkey(&key)
	Philox2x32round(&counter, key) // 6
	Philox2x32bumpkey(&key)
	Philox2x32round(&counter, key) // 7
	Philox2x32bumpkey(&key)
	Philox2x32round(&counter, key) // 8
	Philox2x32bumpkey(&key)
	Philox2x32round(&counter, key) // 9
	Philox2x32bumpkey(&key)

	Philox2x32round(&counter, key) // 10
	return counter
}

// Uint32ToFloat converts a uint32 32 bit integer into a 32 bit float
// in the (0,1) interval (i.e., exclusive of 0 and 1).
// This differs from the Go standard by excluding 0, which is handy for passing
// directly to Log function, and from the reference Philox code by excluding 1
// which is in the Go standard and most other standard RNGs.
func Uint32ToFloat(val uint32) float32 {
	const factor = float32(1.) / (float32(0xffffffff) + float32(1.))
	const halffactor = float32(0.5) * factor
	f := float32(val)*factor + halffactor
	if f == 1 { // exclude 1
		return math.Float32frombits(0x3F7FFFFF)
	}
	return f
}

// Uint32ToFloat11 converts a uint32 32 bit integer into a 32 bit float
// in the [1,1] interval (inclusive of -1 and 1, never identically == 0)
func Uint32ToFloat11(val uint32) float32 {
	const factor = float32(1.) / (float32(0x7fffffff) + float32(1.))
	const halffactor = float32(0.5) * factor
	return (float32(int32(val))*factor + halffactor)
}

// Uint2ToFloat converts two uint32 32 bit integers (Uint2)
// into two corresponding 32 bit float values (float2)
// in the (0,1) interval (i.e., exclusive of 1).
func Uint2ToFloat(val sltype.Uint2) sltype.Float2 {
	var r sltype.Float2
	r.X = Uint32ToFloat(val.X)
	r.Y = Uint32ToFloat(val.Y)
	return r
}

// Uint2ToFloat11 converts two uint32 32 bit integers (Uint2)
// into two corresponding 32 bit float values (float2)
// in the (0,1) interval (i.e., exclusive of 1).
func Uint2ToFloat11(val sltype.Uint2) sltype.Float2 {
	var r sltype.Float2
	r.X = Uint32ToFloat11(val.X)
	r.Y = Uint32ToFloat11(val.Y)
	return r
}

// CounterIncr increments the given counter as if it was
// a uint64 integer.
func CounterIncr(counter *sltype.Uint2) {
	if counter.X == 0xffffffff {
		counter.Y++
		counter.X = 0
	} else {
		counter.X++
	}
}

// CounterAdd adds the given increment to the counter
func CounterAdd(counter *sltype.Uint2, inc uint32) {
	if inc == 0 {
		return
	}
	if counter.X > 0xffffffff-inc {
		counter.Y++
		counter.X = (inc - 1) - (0xffffffff - counter.X)
	} else {
		counter.X += inc
	}
}

////////////////////////////////////////////////////////////
//   Methods below provide a standard interface
//   with more readable names, mapping onto the Go rand methods.
//   These are what should be called by end-user code.

// Uint2 returns two uniformly distributed 32 unsigned integers,
// based on given counter and key.
// The counter is incremented by 1 (in a 64-bit equivalent manner)
// as a result of this call, ensuring that the next call will produce
// the next random numberin the sequence.  The key should be the
// unique index of the element being updated.
func Uint2(counter *sltype.Uint2, key uint32) sltype.Uint2 {
	res := Philox2x32(*counter, key)
	CounterIncr(counter)
	return res
}

// Uint32 returns a uniformly distributed 32 unsigned integer,
// based on given counter and key.
// The counter is incremented by 1 (in a 64-bit equivalent manner)
// as a result of this call, ensuring that the next call will produce
// the next random number in the sequence.  The key should be the
// unique index of the element being updated.
func Uint32(counter *sltype.Uint2, key uint32) uint32 {
	res := Philox2x32(*counter, key)
	CounterIncr(counter)
	return res.X
}

// Float2 returns two uniformly distributed 32 floats
// in range (0,1) based on given counter and key.
// The counter is incremented by 1 (in a 64-bit equivalent manner)
// as a result of this call, ensuring that the next call will produce
// the next random number in the sequence.  The key should be the
// unique index of the element being updated.
func Float2(counter *sltype.Uint2, key uint32) sltype.Float2 {
	return Uint2ToFloat(Uint2(counter, key))
}

// Float returns a uniformly distributed 32 float
// in range (0,1) based on given counter and key.
// The counter is incremented by 1 (in a 64-bit equivalent manner)
// as a result of this call, ensuring that the next call will produce
// the next random number in the sequence.  The key should be the
// unique index of the element being updated.
func Float(counter *sltype.Uint2, key uint32) float32 {
	return Uint32ToFloat(Uint32(counter, key))
}

// Float112 returns two uniformly distributed 32 floats
// in range [-1,1] based on given counter and key.
// The counter is incremented by 1 (in a 64-bit equivalent manner)
// as a result of this call, ensuring that the next call will produce
// the next random number in the sequence.  The key should be the
// unique index of the element being updated.
func Float112(counter *sltype.Uint2, key uint32) sltype.Float2 {
	return Uint2ToFloat11(Uint2(counter, key))
}

// Float11 returns a uniformly distributed 32 float
// in range [-1,1] based on given counter and key.
// The counter is incremented by 1 (in a 64-bit equivalent manner)
// as a result of this call, ensuring that the next call will produce
// the next random number in the sequence.  The key should be the
// unique index of the element being updated.
func Float11(counter *sltype.Uint2, key uint32) float32 {
	return Uint32ToFloat11(Uint32(counter, key))
}

// BoolP returns a bool true value with probability p
func BoolP(counter *sltype.Uint2, key uint32, p float32) bool {
	return (Float(counter, key) < p)
}

func SincosPi(x float32) (s, c float32) {
	const PIf = 3.1415926535897932
	s, c = math32.Sincos(PIf * x)
	return
}

// NormFloat2 returns two random 32 bit floating numbers
// distributed according to the normal, Gaussian distribution
// with zero mean and unit variance.
// This is done very efficiently using the Box-Muller algorithm
// that consumes two random 32 bit uint32 values.
func NormFloat2(counter *sltype.Uint2, key uint32) sltype.Float2 {
	ur := Uint2(counter, key)
	var f sltype.Float2
	f.X, f.Y = SincosPi(Uint32ToFloat11(ur.X))
	r := math32.Sqrt(-2. * math32.Log(Uint32ToFloat(ur.Y))) // guaranteed to avoid 0
	f.X *= r
	f.Y *= r
	return f
}

// NormFloat returns a random 32 bit floating number
// distributed according to the normal, Gaussian distribution
// with zero mean and unit variance.
func NormFloat(counter *sltype.Uint2, key uint32) float32 {
	f := NormFloat2(counter, key)
	return f.X
}

// Uintn returns a uint32 in the range [0,n)
func Uintn(counter *sltype.Uint2, key uint32, n uint32) uint32 {
	v := Float(counter, key)
	return uint32(v * float32(n))
}

// Counter is used for storing the random counter using aligned 16 byte storage,
// with convenience methods for typical use cases.
// It retains a copy of the last Seed value, which is applied to the Hi uint32 value.
type Counter struct {

	// lower 32 bits of counter, incremented first
	Lo uint32

	// higher 32 bits of counter, incremented only when Lo turns over
	Hi uint32

	// last seed value set by Seed method, restored by Reset()
	HiSeed uint32

	pad uint32
}

// Reset resets counter to last set Seed state
func (ct *Counter) Reset() {
	ct.Lo = 0
	ct.Hi = ct.HiSeed
}

// Uint2 returns counter as a Uint2
func (ct *Counter) Uint2() sltype.Uint2 {
	return sltype.Uint2{ct.Lo, ct.Hi}
}

// Set sets the counter from a Uint2
func (ct *Counter) Set(c sltype.Uint2) {
	ct.Lo = c.X
	ct.Hi = c.Y
}

// Seed sets the Hi uint32 value from given seed, saving it in HiSeed field.
// Each increment in seed generates a unique sequence of over 4 billion numbers,
// so it is reasonable to just use incremental values there, but more widely
// spaced numbers will result in longer unique sequences.
// Resets Lo to 0.
// This same seed will be restored during Reset
func (ct *Counter) Seed(seed uint32) {
	ct.Lo = 0
	ct.Hi = seed
	ct.HiSeed = seed
}

// Add increments the counter by given amount.
// Call this after thread completion with number of random numbers
// generated per thread.
func (ct *Counter) Add(inc uint32) sltype.Uint2 {
	c := ct.Uint2()
	CounterAdd(&c, inc)
	ct.Set(c)
	return c
}
