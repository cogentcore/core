// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package randx

// PChoose32 chooses an index in given slice of float32's at random according
// to the probilities of each item (must be normalized to sum to 1).
// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func PChoose32(ps []float32, randOpt ...Rand) int {
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	pv := rnd.Float32()
	sum := float32(0)
	for i, p := range ps {
		sum += p
		if pv < sum { // note: lower values already excluded
			return i
		}
	}
	return len(ps) - 1
}

// PChoose64 chooses an index in given slice of float64's at random according
// to the probilities of each item (must be normalized to sum to 1)
// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func PChoose64(ps []float64, randOpt ...Rand) int {
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	pv := rnd.Float64()
	sum := float64(0)
	for i, p := range ps {
		sum += p
		if pv < sum { // note: lower values already excluded
			return i
		}
	}
	return len(ps) - 1
}
