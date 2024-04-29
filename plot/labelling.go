// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copied directly from gonum/plot:
// Copyright ©2017 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is an implementation of the Talbot, Lin and Hanrahan algorithm
// described in doi:10.1109/TVCG.2010.130 with reference to the R
// implementation in the labeling package, ©2014 Justin Talbot (Licensed
// MIT+file LICENSE|Unlimited).

package plot

import "cogentcore.org/core/math32"

const (
	// dlamchE is the machine epsilon. For IEEE this is 2^{-53}.
	dlamchE = 1.0 / (1 << 53)

	// dlamchB is the radix of the machine (the base of the number system).
	dlamchB = 2

	// dlamchP is base * eps.
	dlamchP = dlamchB * dlamchE
)

const (
	// free indicates no restriction on label containment.
	free = iota
	// containData specifies that all the data range lies
	// within the interval [label_min, label_max].
	containData
	// withinData specifies that all labels lie within the
	// interval [dMin, dMax].
	withinData
)

// talbotLinHanrahan returns an optimal set of approximately want label values
// for the data range [dMin, dMax], and the step and magnitude of the step between values.
// containment is specifies are guarantees for label and data range containment, valid
// values are free, containData and withinData.
// The optional parameters Q, nice numbers, and w, weights, allow tuning of the
// algorithm but by default (when nil) are set to the parameters described in the
// paper.
// The legibility function allows tuning of the legibility assessment for labels.
// By default, when nil, legbility will set the legibility score for each candidate
// labelling scheme to 1.
// See the paper for an explanation of the function of Q, w and legibility.
func talbotLinHanrahan(dMin, dMax float32, want int, containment int, Q []float32, w *weights, legibility func(lMin, lMax, lStep float32) float32) (values []float32, step, q float32, magnitude int) {
	const eps = dlamchP * 100

	if dMin > dMax {
		panic("labelling: invalid data range: min greater than max")
	}

	if Q == nil {
		Q = []float32{1, 5, 2, 2.5, 4, 3}
	}
	if w == nil {
		w = &weights{
			simplicity: 0.25,
			coverage:   0.2,
			density:    0.5,
			legibility: 0.05,
		}
	}
	if legibility == nil {
		legibility = unitLegibility
	}

	if r := dMax - dMin; r < eps {
		l := make([]float32, want)
		step := r / float32(want-1)
		for i := range l {
			l[i] = dMin + float32(i)*step
		}
		magnitude = minAbsMag(dMin, dMax)
		return l, step, 0, magnitude
	}

	type selection struct {
		// n is the number of labels selected.
		n int
		// lMin and lMax are the selected min
		// and max label values. lq is the q
		// chosen.
		lMin, lMax, lStep, lq float32
		// score is the score for the selection.
		score float32
		// magnitude is the magnitude of the
		// label step distance.
		magnitude int
	}
	best := selection{score: -2}

outer:
	for skip := 1; ; skip++ {
		for _, q := range Q {
			sm := maxSimplicity(q, Q, skip)
			if w.score(sm, 1, 1, 1) < best.score {
				break outer
			}

			for have := 2; ; have++ {
				dm := maxDensity(have, want)
				if w.score(sm, 1, dm, 1) < best.score {
					break
				}

				delta := (dMax - dMin) / float32(have+1) / float32(skip) / q

				const maxExp = 309
				for mag := int(math32.Ceil(math32.Log10(delta))); mag < maxExp; mag++ {
					step := float32(skip) * q * math32.Pow10(mag)

					cm := maxCoverage(dMin, dMax, step*float32(have-1))
					if w.score(sm, cm, dm, 1) < best.score {
						break
					}

					fracStep := step / float32(skip)
					kStep := step * float32(have-1)

					minStart := (math32.Floor(dMax/step) - float32(have-1)) * float32(skip)
					maxStart := math32.Ceil(dMax/step) * float32(skip)
					for start := minStart; start <= maxStart && start != start-1; start++ {
						lMin := start * fracStep
						lMax := lMin + kStep

						switch containment {
						case containData:
							if dMin < lMin || lMax < dMax {
								continue
							}
						case withinData:
							if lMin < dMin || dMax < lMax {
								continue
							}
						case free:
							// Free choice.
						}

						score := w.score(
							simplicity(q, Q, skip, lMin, lMax, step),
							coverage(dMin, dMax, lMin, lMax),
							density(have, want, dMin, dMax, lMin, lMax),
							legibility(lMin, lMax, step),
						)
						if score > best.score {
							best = selection{
								n:         have,
								lMin:      lMin,
								lMax:      lMax,
								lStep:     float32(skip) * q,
								lq:        q,
								score:     score,
								magnitude: mag,
							}
						}
					}
				}
			}
		}
	}

	if best.score == -2 {
		l := make([]float32, want)
		step := (dMax - dMin) / float32(want-1)
		for i := range l {
			l[i] = dMin + float32(i)*step
		}
		magnitude = minAbsMag(dMin, dMax)
		return l, step, 0, magnitude
	}

	l := make([]float32, best.n)
	step = best.lStep * math32.Pow10(best.magnitude)
	for i := range l {
		l[i] = best.lMin + float32(i)*step
	}
	return l, best.lStep, best.lq, best.magnitude
}

// minAbsMag returns the minumum magnitude of the absolute values of a and b.
func minAbsMag(a, b float32) int {
	return int(math32.Min(math32.Floor(math32.Log10(math32.Abs(a))), (math32.Floor(math32.Log10(math32.Abs(b))))))
}

// simplicity returns the simplicity score for how will the curent q, lMin, lMax,
// lStep and skip match the given nice numbers, Q.
func simplicity(q float32, Q []float32, skip int, lMin, lMax, lStep float32) float32 {
	const eps = dlamchP * 100

	for i, v := range Q {
		if v == q {
			m := math32.Mod(lMin, lStep)
			v = 0
			if (m < eps || lStep-m < eps) && lMin <= 0 && 0 <= lMax {
				v = 1
			}
			return 1 - float32(i)/(float32(len(Q))-1) - float32(skip) + v
		}
	}
	panic("labelling: invalid q for Q")
}

// maxSimplicity returns the maximum simplicity for q, Q and skip.
func maxSimplicity(q float32, Q []float32, skip int) float32 {
	for i, v := range Q {
		if v == q {
			return 1 - float32(i)/(float32(len(Q))-1) - float32(skip) + 1
		}
	}
	panic("labelling: invalid q for Q")
}

// coverage returns the coverage score for based on the average
// squared distance between the extreme labels, lMin and lMax, and
// the extreme data points, dMin and dMax.
func coverage(dMin, dMax, lMin, lMax float32) float32 {
	r := 0.1 * (dMax - dMin)
	max := dMax - lMax
	min := dMin - lMin
	return 1 - 0.5*(max*max+min*min)/(r*r)
}

// maxCoverage returns the maximum coverage achievable for the data
// range.
func maxCoverage(dMin, dMax, span float32) float32 {
	r := dMax - dMin
	if span <= r {
		return 1
	}
	h := 0.5 * (span - r)
	r *= 0.1
	return 1 - (h*h)/(r*r)
}

// density returns the density score which measures the goodness of
// the labelling density compared to the user defined target
// based on the want parameter given to talbotLinHanrahan.
func density(have, want int, dMin, dMax, lMin, lMax float32) float32 {
	rho := float32(have-1) / (lMax - lMin)
	rhot := float32(want-1) / (math32.Max(lMax, dMax) - math32.Min(dMin, lMin))
	if d := rho / rhot; d >= 1 {
		return 2 - d
	}
	return 2 - rhot/rho
}

// maxDensity returns the maximum density score achievable for have and want.
func maxDensity(have, want int) float32 {
	if have < want {
		return 1
	}
	return 2 - float32(have-1)/float32(want-1)
}

// unitLegibility returns a default legibility score ignoring label
// spacing.
func unitLegibility(_, _, _ float32) float32 {
	return 1
}

// weights is a helper type to calcuate the labelling scheme's total score.
type weights struct {
	simplicity, coverage, density, legibility float32
}

// score returns the score for a labelling scheme with simplicity, s,
// coverage, c, density, d and legibility l.
func (w *weights) score(s, c, d, l float32) float32 {
	return w.simplicity*s + w.coverage*c + w.density*d + w.legibility*l
}
