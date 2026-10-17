// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gsbench

import (
	"math/rand"
	randv2 "math/rand/v2"
	"testing"
)

// Benchmarks of math/rand and math/rand/v2, which simulations use heavily.
// Each op makes 1000 draws (Perm: one permutation of 1000).

var sinkF64 float64

// randDraws is the number of draws per op.
const randDraws = 1000

func BenchmarkRandFloat32(b *testing.B) {
	for range b.N {
		var s float32
		for range randDraws {
			s += rand.Float32()
		}
		sinkF = s
	}
}

func BenchmarkRandFloat64(b *testing.B) {
	for range b.N {
		var s float64
		for range randDraws {
			s += rand.Float64()
		}
		sinkF64 = s
	}
}

func BenchmarkRandIntn(b *testing.B) {
	for range b.N {
		s := 0
		for range randDraws {
			s += rand.Intn(100)
		}
		sinkI = s
	}
}

func BenchmarkRandInt63(b *testing.B) {
	for range b.N {
		var s int64
		for range randDraws {
			s ^= rand.Int63()
		}
		sinkI = int(s & 0xffff)
	}
}

func BenchmarkRandNormFloat64(b *testing.B) {
	for range b.N {
		var s float64
		for range randDraws {
			s += rand.NormFloat64()
		}
		sinkF64 = s
	}
}

func BenchmarkRandPerm(b *testing.B) {
	for range b.N {
		p := rand.Perm(randDraws)
		sinkI = p[0]
	}
}

// newSeeded returns a *rand.Rand made the way simulations make their own
// reproducible sources.
func newSeeded() *rand.Rand { return rand.New(rand.NewSource(42)) }

func BenchmarkRandSeededFloat32(b *testing.B) {
	r := newSeeded()
	for range b.N {
		var s float32
		for range randDraws {
			s += r.Float32()
		}
		sinkF = s
	}
}

func BenchmarkRandSeededFloat64(b *testing.B) {
	r := newSeeded()
	for range b.N {
		var s float64
		for range randDraws {
			s += r.Float64()
		}
		sinkF64 = s
	}
}

func BenchmarkRandSeededIntn(b *testing.B) {
	r := newSeeded()
	for range b.N {
		s := 0
		for range randDraws {
			s += r.Intn(100)
		}
		sinkI = s
	}
}

func BenchmarkRandSeededInt63(b *testing.B) {
	r := newSeeded()
	for range b.N {
		var s int64
		for range randDraws {
			s ^= r.Int63()
		}
		sinkI = int(s & 0xffff)
	}
}

func BenchmarkRandSeededNormFloat64(b *testing.B) {
	r := newSeeded()
	for range b.N {
		var s float64
		for range randDraws {
			s += r.NormFloat64()
		}
		sinkF64 = s
	}
}

func BenchmarkRandSeededPerm(b *testing.B) {
	r := newSeeded()
	for range b.N {
		p := r.Perm(randDraws)
		sinkI = p[0]
	}
}

// BenchmarkRandSeed reseeds a source, which simulations do at the start of
// each run.
func BenchmarkRandSeed(b *testing.B) {
	r := newSeeded()
	for i := range b.N {
		r.Seed(int64(i))
	}
}

func BenchmarkRandV2Float32(b *testing.B) {
	for range b.N {
		var s float32
		for range randDraws {
			s += randv2.Float32()
		}
		sinkF = s
	}
}

func BenchmarkRandV2IntN(b *testing.B) {
	for range b.N {
		s := 0
		for range randDraws {
			s += randv2.IntN(100)
		}
		sinkI = s
	}
}

func BenchmarkRandV2PCGFloat64(b *testing.B) {
	r := randv2.New(randv2.NewPCG(1, 2))
	for range b.N {
		var s float64
		for range randDraws {
			s += r.Float64()
		}
		sinkF64 = s
	}
}

func BenchmarkRandV2ChaCha8Float64(b *testing.B) {
	r := randv2.New(randv2.NewChaCha8([32]byte{1, 2, 3}))
	for range b.N {
		var s float64
		for range randDraws {
			s += r.Float64()
		}
		sinkF64 = s
	}
}

// simRand mirrors the Rand interface of cogentcore.org/lab/base/randx, which
// simulations use to draw from either the global source or their own.
type simRand interface {
	Init(seed int64)
	Float32() float32
	Float64() float64
	Intn(n int) int
	NormFloat64() float64
	Shuffle(n int, swap func(i, j int))
}

// sysRand mirrors randx.SysRand: a seeded *rand.Rand, or the global source
// when Rand is nil.
type sysRand struct{ Rand *rand.Rand }

func (r *sysRand) Init(seed int64) {
	if r.Rand == nil {
		r.Rand = rand.New(rand.NewSource(seed))
		return
	}
	r.Rand.Seed(seed)
}

func (r *sysRand) Float32() float32 {
	if r.Rand == nil {
		return rand.Float32()
	}
	return r.Rand.Float32()
}

func (r *sysRand) Float64() float64 {
	if r.Rand == nil {
		return rand.Float64()
	}
	return r.Rand.Float64()
}

func (r *sysRand) Intn(n int) int {
	if r.Rand == nil {
		return rand.Intn(n)
	}
	return r.Rand.Intn(n)
}

func (r *sysRand) NormFloat64() float64 {
	if r.Rand == nil {
		return rand.NormFloat64()
	}
	return r.Rand.NormFloat64()
}

func (r *sysRand) Shuffle(n int, swap func(i, j int)) {
	if r.Rand == nil {
		rand.Shuffle(n, swap)
		return
	}
	r.Rand.Shuffle(n, swap)
}

// simUnit is a neuron-like unit with noisy state.
type simUnit struct {
	Act, Ge, Noise float32
	Spike          bool
}

// gaussianGen mirrors randx.GaussianGen.
func gaussianGen(mean, sigma float64, rnd simRand) float64 {
	return mean + sigma*rnd.NormFloat64()
}

// simCycles runs one epoch of a small simulation: the trial order is
// permuted, and every cycle of each trial adds Gaussian noise to every unit,
// spikes with a probability, and picks a random unit to stimulate, all
// through the simRand interface as randx users do.
func simCycles(rnd simRand, units []simUnit, order []int) {
	for i := range order {
		order[i] = i
	}
	rnd.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
	for range order {
		for range 10 {
			for u := range units {
				un := &units[u]
				un.Noise = float32(gaussianGen(0, 0.1, rnd))
				un.Ge += 0.1*(un.Noise-un.Ge) + 0.01
				un.Spike = rnd.Float32() < un.Ge
			}
			units[rnd.Intn(len(units))].Act += 0.5
		}
	}
}

// BenchmarkRandSimSeeded runs simCycles (10 trials x 10 cycles x 100 units,
// about 20000 draws) with a seeded source, as reproducible simulations do.
func BenchmarkRandSimSeeded(b *testing.B) {
	rnd := &sysRand{Rand: newSeeded()}
	units := make([]simUnit, 100)
	order := make([]int, 10)
	for range b.N {
		simCycles(rnd, units, order)
	}
	sinkF = units[0].Act
}

// BenchmarkRandSimGlobal runs simCycles with the global source.
func BenchmarkRandSimGlobal(b *testing.B) {
	rnd := &sysRand{}
	units := make([]simUnit, 100)
	order := make([]int, 10)
	for range b.N {
		simCycles(rnd, units, order)
	}
	sinkF = units[0].Act
}
