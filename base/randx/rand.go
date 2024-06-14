// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package randx

//go:generate core generate -add-types

import "math/rand"

// Rand provides an interface with most of the standard
// rand.Rand methods, to support the use of either the
// global rand generator or a separate Rand source.
type Rand interface {
	// Seed uses the provided seed value to initialize the generator to a deterministic state.
	// Seed should not be called concurrently with any other Rand method.
	Seed(seed int64)

	// Int63 returns a non-negative pseudo-random 63-bit integer as an int64.
	Int63() int64

	// Uint32 returns a pseudo-random 32-bit value as a uint32.
	Uint32() uint32

	// Uint64 returns a pseudo-random 64-bit value as a uint64.
	Uint64() uint64

	// Int31 returns a non-negative pseudo-random 31-bit integer as an int32.
	Int31() int32

	// Int returns a non-negative pseudo-random int.
	Int() int

	// Int63n returns, as an int64, a non-negative pseudo-random number in the half-open interval [0,n).
	// It panics if n <= 0.
	Int63n(n int64) int64

	// Int31n returns, as an int32, a non-negative pseudo-random number in the half-open interval [0,n).
	// It panics if n <= 0.
	Int31n(n int32) int32

	// Intn returns, as an int, a non-negative pseudo-random number in the half-open interval [0,n).
	// It panics if n <= 0.
	Intn(n int) int

	// Float64 returns, as a float64, a pseudo-random number in the half-open interval [0.0,1.0).
	Float64() float64

	// Float32 returns, as a float32, a pseudo-random number in the half-open interval [0.0,1.0).
	Float32() float32

	// NormFloat64 returns a normally distributed float64 in the range
	// [-math.MaxFloat64, +math.MaxFloat64] with
	// standard normal distribution (mean = 0, stddev = 1)
	// from the default Source.
	// To produce a different normal distribution, callers can
	// adjust the output using:
	//
	//	sample = NormFloat64() * desiredStdDev + desiredMean
	NormFloat64() float64

	// ExpFloat64 returns an exponentially distributed float64 in the range
	// (0, +math.MaxFloat64] with an exponential distribution whose rate parameter
	// (lambda) is 1 and whose mean is 1/lambda (1) from the default Source.
	// To produce a distribution with a different rate parameter,
	// callers can adjust the output using:
	//
	//	sample = ExpFloat64() / desiredRateParameter
	ExpFloat64() float64

	// Perm returns, as a slice of n ints, a pseudo-random permutation of the integers
	// in the half-open interval [0,n).
	Perm(n int) []int

	// Shuffle pseudo-randomizes the order of elements.
	// n is the number of elements. Shuffle panics if n < 0.
	// swap swaps the elements with indexes i and j.
	Shuffle(n int, swap func(i, j int))
}

// SysRand supports the system random number generator
// for either a separate rand.Rand source, or, if that
// is nil, the global rand stream.
type SysRand struct {

	// if non-nil, use this random number source instead of the global default one
	Rand *rand.Rand `display:"-"`
}

// NewGlobalRand returns a new SysRand that implements the
// randx.Rand interface, with the system global rand source.
func NewGlobalRand() *SysRand {
	r := &SysRand{}
	return r
}

// NewSysRand returns a new SysRand with a new
// rand.Rand random source with given initial seed.
func NewSysRand(seed int64) *SysRand {
	r := &SysRand{}
	r.NewRand(seed)
	return r
}

// NewRand sets Rand to a new rand.Rand source using given seed.
func (r *SysRand) NewRand(seed int64) {
	r.Rand = rand.New(rand.NewSource(seed))
}

// Seed uses the provided seed value to initialize the generator to a deterministic state.
// Seed should not be called concurrently with any other Rand method.
func (r *SysRand) Seed(seed int64) {
	if r.Rand == nil {
		rand.Seed(seed)
		return
	}
	r.Rand.Seed(seed)
}

// Int63 returns a non-negative pseudo-random 63-bit integer as an int64.
func (r *SysRand) Int63() int64 {
	if r.Rand == nil {
		return rand.Int63()
	}
	return r.Rand.Int63()
}

// Uint32 returns a pseudo-random 32-bit value as a uint32.
func (r *SysRand) Uint32() uint32 {
	if r.Rand == nil {
		return rand.Uint32()
	}
	return r.Rand.Uint32()
}

// Uint64 returns a pseudo-random 64-bit value as a uint64.
func (r *SysRand) Uint64() uint64 {
	if r.Rand == nil {
		return rand.Uint64()
	}
	return r.Rand.Uint64()
}

// Int31 returns a non-negative pseudo-random 31-bit integer as an int32.
func (r *SysRand) Int31() int32 {
	if r.Rand == nil {
		return rand.Int31()
	}
	return r.Rand.Int31()
}

// Int returns a non-negative pseudo-random int.
func (r *SysRand) Int() int {
	if r.Rand == nil {
		return rand.Int()
	}
	return r.Rand.Int()
}

// Int63n returns, as an int64, a non-negative pseudo-random number in the half-open interval [0,n).
// It panics if n <= 0.
func (r *SysRand) Int63n(n int64) int64 {
	if r.Rand == nil {
		return rand.Int63n(n)
	}
	return r.Rand.Int63n(n)
}

// Int31n returns, as an int32, a non-negative pseudo-random number in the half-open interval [0,n).
// It panics if n <= 0.
func (r *SysRand) Int31n(n int32) int32 {
	if r.Rand == nil {
		return rand.Int31n(n)
	}
	return r.Rand.Int31n(n)
}

// Intn returns, as an int, a non-negative pseudo-random number in the half-open interval [0,n).
// It panics if n <= 0.
func (r *SysRand) Intn(n int) int {
	if r.Rand == nil {
		return rand.Intn(n)
	}
	return r.Rand.Intn(n)
}

// Float64 returns, as a float64, a pseudo-random number in the half-open interval [0.0,1.0).
func (r *SysRand) Float64() float64 {
	if r.Rand == nil {
		return rand.Float64()
	}
	return r.Rand.Float64()
}

// Float32 returns, as a float32, a pseudo-random number in the half-open interval [0.0,1.0).
func (r *SysRand) Float32() float32 {
	if r.Rand == nil {
		return rand.Float32()
	}
	return r.Rand.Float32()
}

// NormFloat64 returns a normally distributed float64 in the range
// [-math.MaxFloat64, +math.MaxFloat64] with
// standard normal distribution (mean = 0, stddev = 1)
// from the default Source.
// To produce a different normal distribution, callers can
// adjust the output using:
//
//	sample = NormFloat64() * desiredStdDev + desiredMean
func (r *SysRand) NormFloat64() float64 {
	if r.Rand == nil {
		return rand.NormFloat64()
	}
	return r.Rand.NormFloat64()
}

// ExpFloat64 returns an exponentially distributed float64 in the range
// (0, +math.MaxFloat64] with an exponential distribution whose rate parameter
// (lambda) is 1 and whose mean is 1/lambda (1) from the default Source.
// To produce a distribution with a different rate parameter,
// callers can adjust the output using:
//
//	sample = ExpFloat64() / desiredRateParameter
func (r *SysRand) ExpFloat64() float64 {
	if r.Rand == nil {
		return rand.ExpFloat64()
	}
	return r.Rand.ExpFloat64()
}

// Perm returns, as a slice of n ints, a pseudo-random permutation of the integers
// in the half-open interval [0,n).
func (r *SysRand) Perm(n int) []int {
	if r.Rand == nil {
		return rand.Perm(n)
	}
	return r.Rand.Perm(n)
}

// Shuffle pseudo-randomizes the order of elements.
// n is the number of elements. Shuffle panics if n < 0.
// swap swaps the elements with indexes i and j.
func (r *SysRand) Shuffle(n int, swap func(i, j int)) {
	if r.Rand == nil {
		rand.Shuffle(n, swap)
		return
	}
	r.Rand.Shuffle(n, swap)
}
