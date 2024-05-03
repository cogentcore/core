// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package randx

// BoolP is a simple method to generate a true value with given probability
// (else false). It is just rand.Float64() < p but this is more readable
// and explicit.
// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func BoolP(p float64, randOpt ...Rand) bool {
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	return rnd.Float64() < p
}

// BoolP32 is a simple method to generate a true value with given probability
// (else false). It is just rand.Float32() < p but this is more readable
// and explicit.
// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func BoolP32(p float32, randOpt ...Rand) bool {
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	return rnd.Float32() < p
}
