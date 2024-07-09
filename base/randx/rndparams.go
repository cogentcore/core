// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package randx

// RandParams provides parameterized random number generation according to different distributions
// and variance, mean params
type RandParams struct { //git:add

	// distribution to generate random numbers from
	Dist RandDists

	// mean of random distribution -- typically added to generated random variants
	Mean float64

	// variability parameter for the random numbers (gauss = standard deviation, not variance; uniform = half-range, others as noted in RandDists)
	Var float64

	// extra parameter for distribution (depends on each one)
	Par float64
}

func (rp *RandParams) Defaults() {
	rp.Var = 1
	rp.Par = 1
}

func (rp *RandParams) ShouldDisplay(field string) bool {
	switch field {
	case "Par":
		return rp.Dist == Gamma || rp.Dist == Binomial || rp.Dist == Beta
	}
	return true
}

// Gen generates a random variable according to current parameters.

// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func (rp *RandParams) Gen(randOpt ...Rand) float64 {
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	switch rp.Dist {
	case Uniform:
		return UniformMeanRange(rp.Mean, rp.Var, rnd)
	case Binomial:
		return rp.Mean + BinomialGen(rp.Par, rp.Var, rnd)
	case Poisson:
		return rp.Mean + PoissonGen(rp.Var, rnd)
	case Gamma:
		return rp.Mean + GammaGen(rp.Par, rp.Var, rnd)
	case Gaussian:
		return GaussianGen(rp.Mean, rp.Var, rnd)
	case Beta:
		return rp.Mean + BetaGen(rp.Var, rp.Par, rnd)
	}
	return rp.Mean
}

// RandDists are different random number distributions
type RandDists int32 //enums:enum

// The random number distributions
const (
	// Uniform has a uniform probability distribution over Var = range on either side of the Mean
	Uniform RandDists = iota

	// Binomial represents number of 1's in n (Par) random (Bernouli) trials of probability p (Var)
	Binomial

	// Poisson represents number of events in interval, with event rate (lambda = Var) plus Mean
	Poisson

	// Gamma represents maximum entropy distribution with two parameters: scaling parameter (Var)
	// and shape parameter k (Par) plus Mean
	Gamma

	// Gaussian normal with Var = stddev plus Mean
	Gaussian

	// Beta with Var = alpha and Par = beta shape parameters
	Beta

	// Mean is just the constant Mean, no randomness
	Mean
)

// IntZeroN returns uniform random integer in the range between 0 and n, exclusive of n: [0,n).
// Thr is an optional parallel thread index (-1 0 to ignore).
// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func IntZeroN(n int64, randOpt ...Rand) int64 {
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	return rnd.Int63n(n)
}

// IntMinMax returns uniform random integer in range between min and max, exclusive of max: [min,max).

// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func IntMinMax(min, max int64, randOpt ...Rand) int64 {
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	return min + rnd.Int63n(max-min)
}

// IntMeanRange returns uniform random integer with given range on either side of the mean:
// [mean - range, mean + range]

// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func IntMeanRange(mean, rnge int64, randOpt ...Rand) int64 {
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	return mean + (rnd.Int63n(2*rnge+1) - rnge)
}

// ZeroOne returns a uniform random number between zero and one (exclusive of 1)

// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func ZeroOne(randOpt ...Rand) float64 {
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	return rnd.Float64()
}

// UniformMinMax returns uniform random number between min and max values inclusive
// (Do not use for generating integers - will not include max!)

// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func UniformMinMax(min, max float64, randOpt ...Rand) float64 {
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	return min + (max-min)*rnd.Float64()
}

// UniformMeanRange returns uniform random number with given range on either size of the mean:
// [mean - range, mean + range]

// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func UniformMeanRange(mean, rnge float64, randOpt ...Rand) float64 {
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	return mean + rnge*2.0*(rnd.Float64()-0.5)
}
