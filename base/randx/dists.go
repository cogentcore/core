// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package randx

import (
	"math"
)

// note: this file contains random distribution functions
// from gonum.org/v1/gonum/stat/distuv
// which we modified only to use the randx.Rand interface.
// BinomialGen returns binomial with n trials (par) each of probability p (var)
// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func BinomialGen(n, p float64, randOpt ...Rand) float64 {
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	// NUMERICAL RECIPES IN C: THE ART OF SCIENTIFIC COMPUTING (ISBN 0-521-43108-5)
	// p. 295-6
	// http://www.aip.de/groups/soe/local/numres/bookcpdf/c7-3.pdf

	porg := p

	if p > 0.5 {
		p = 1 - p
	}
	am := n * p

	if n < 25 {
		// Use direct method.
		bnl := 0.0
		for i := 0; i < int(n); i++ {
			if rnd.Float64() < p {
				bnl++
			}
		}
		if p != porg {
			return n - bnl
		}
		return bnl
	}

	if am < 1 {
		// Use rejection method with Poisson proposal.
		const logM = 2.6e-2 // constant for rejection sampling (https://en.wikipedia.org/wiki/Rejection_sampling)
		var bnl float64
		z := -p
		pclog := (1 + 0.5*z) * z / (1 + (1+1.0/6*z)*z) // Padé approximant of log(1 + x)
		for {
			bnl = 0.0
			t := 0.0
			for i := 0; i < int(n); i++ {
				t += rnd.ExpFloat64()
				if t >= am {
					break
				}
				bnl++
			}
			bnlc := n - bnl
			z = -bnl / n
			log1p := (1 + 0.5*z) * z / (1 + (1+1.0/6*z)*z)
			t = (bnlc+0.5)*log1p + bnl - bnlc*pclog + 1/(12*bnlc) - am + logM // Uses Stirling's expansion of log(n!)
			if rnd.ExpFloat64() >= t {
				break
			}
		}
		if p != porg {
			return n - bnl
		}
		return bnl
	}
	// Original algorithm samples from a Poisson distribution with the
	// appropriate expected value. However, the Poisson approximation is
	// asymptotic such that the absolute deviation in probability is O(1/n).
	// Rejection sampling produces exact variates with at worst less than 3%
	// rejection with miminal additional computation.

	// Use rejection method with Cauchy proposal.
	g, _ := math.Lgamma(n + 1)
	plog := math.Log(p)
	pclog := math.Log1p(-p)
	sq := math.Sqrt(2 * am * (1 - p))
	for {
		var em, y float64
		for {
			y = math.Tan(math.Pi * rnd.Float64())
			em = sq*y + am
			if em >= 0 && em < n+1 {
				break
			}
		}
		em = math.Floor(em)
		lg1, _ := math.Lgamma(em + 1)
		lg2, _ := math.Lgamma(n - em + 1)
		t := 1.2 * sq * (1 + y*y) * math.Exp(g-lg1-lg2+em*plog+(n-em)*pclog)
		if rnd.Float64() <= t {
			if p != porg {
				return n - em
			}
			return em
		}
	}
}

// PoissonGen returns poisson variable, as number of events in interval,
// with event rate (lmb = Var) plus mean
// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func PoissonGen(lambda float64, randOpt ...Rand) float64 {
	// NUMERICAL RECIPES IN C: THE ART OF SCIENTIFIC COMPUTING (ISBN 0-521-43108-5)
	// p. 294
	// <http://www.aip.de/groups/soe/local/numres/bookcpdf/c7-3.pdf>
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	if lambda < 10.0 {
		// Use direct method.
		var em float64
		t := 0.0
		for {
			t += rnd.ExpFloat64()
			if t >= lambda {
				break
			}
			em++
		}
		return em
	}
	// Generate using:
	//  W. Hörmann. "The transformed rejection method for generating Poisson
	//  random variables." Insurance: Mathematics and Economics
	//  12.1 (1993): 39-45.

	b := 0.931 + 2.53*math.Sqrt(lambda)
	a := -0.059 + 0.02483*b
	invalpha := 1.1239 + 1.1328/(b-3.4)
	vr := 0.9277 - 3.6224/(b-2)
	for {
		U := rnd.Float64() - 0.5
		V := rnd.Float64()
		us := 0.5 - math.Abs(U)
		k := math.Floor((2*a/us+b)*U + lambda + 0.43)
		if us >= 0.07 && V <= vr {
			return k
		}
		if k <= 0 || (us < 0.013 && V > us) {
			continue
		}
		lg, _ := math.Lgamma(k + 1)
		if math.Log(V*invalpha/(a/(us*us)+b)) <= k*math.Log(lambda)-lambda-lg {
			return k
		}
	}
}

// GammaGen represents maximum entropy distribution with two parameters:
// a shape parameter (Alpha, Par in RandParams),
// and a scaling parameter (Beta, Var in RandParams).
// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func GammaGen(alpha, beta float64, randOpt ...Rand) float64 {
	const (
		// The 0.2 threshold is from https://www4.stat.ncsu.edu/~rmartin/Codes/rgamss.R
		// described in detail in https://arxiv.org/abs/1302.1884.
		smallAlphaThresh = 0.2
	)
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	if beta <= 0 {
		panic("GammaGen: beta <= 0")
	}

	a := alpha
	b := beta
	switch {
	case a <= 0:
		panic("gamma: alpha <= 0")
	case a == 1:
		// Generate from exponential
		return rnd.ExpFloat64() / b
	case a < smallAlphaThresh:
		// Generate using
		//  Liu, Chuanhai, Martin, Ryan and Syring, Nick. "Simulating from a
		//  gamma distribution with small shape parameter"
		//  https://arxiv.org/abs/1302.1884
		//   use this reference: http://link.springer.com/article/10.1007/s00180-016-0692-0

		// Algorithm adjusted to work in log space as much as possible.
		lambda := 1/a - 1
		lr := -math.Log1p(1 / lambda / math.E)
		for {
			e := rnd.ExpFloat64()
			var z float64
			if e >= -lr {
				z = e + lr
			} else {
				z = -rnd.ExpFloat64() / lambda
			}
			eza := math.Exp(-z / a)
			lh := -z - eza
			var lEta float64
			if z >= 0 {
				lEta = -z
			} else {
				lEta = -1 + lambda*z
			}
			if lh-lEta > -rnd.ExpFloat64() {
				return eza / b
			}
		}
	case a >= smallAlphaThresh:
		// Generate using:
		//  Marsaglia, George, and Wai Wan Tsang. "A simple method for generating
		//  gamma variables." ACM Transactions on Mathematical Software (TOMS)
		//  26.3 (2000): 363-372.
		d := a - 1.0/3
		m := 1.0
		if a < 1 {
			d += 1.0
			m = math.Pow(rnd.Float64(), 1/a)
		}
		c := 1 / (3 * math.Sqrt(d))
		for {
			x := rnd.NormFloat64()
			v := 1 + x*c
			if v <= 0.0 {
				continue
			}
			v = v * v * v
			u := rnd.Float64()
			if u < 1.0-0.0331*(x*x)*(x*x) {
				return m * d * v / b
			}
			if math.Log(u) < 0.5*x*x+d*(1-v+math.Log(v)) {
				return m * d * v / b
			}
		}
	}
	panic("unreachable")
}

// GaussianGen returns gaussian (normal) random number with given
// mean and sigma standard deviation.
// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func GaussianGen(mean, sigma float64, randOpt ...Rand) float64 {
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	return mean + sigma*rnd.NormFloat64()
}

// BetaGen returns beta random number with two shape parameters
// alpha > 0 and beta > 0
// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func BetaGen(alpha, beta float64, randOpt ...Rand) float64 {
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	ga := GammaGen(alpha, 1, rnd)
	gb := GammaGen(beta, 1, rnd)

	return ga / (ga + gb)
}
