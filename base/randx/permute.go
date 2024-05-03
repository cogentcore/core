// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package randx

// SequentialInts initializes slice of ints to sequential start..start+N-1
// numbers -- for cases where permuting the order is optional.
func SequentialInts(ins []int, start int) {
	for i := range ins {
		ins[i] = start + i
	}
}

// PermuteInts permutes (shuffles) the order of elements in the given int slice
// using the standard Fisher-Yates shuffle
// https://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle
// So you don't have to remember how to call rand.Shuffle.
// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func PermuteInts(ins []int, randOpt ...Rand) {
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	rnd.Shuffle(len(ins), func(i, j int) {
		ins[i], ins[j] = ins[j], ins[i]
	})
}

// PermuteStrings permutes (shuffles) the order of elements in the given string slice
// using the standard Fisher-Yates shuffle
// https://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle
// So you don't have to remember how to call rand.Shuffle
// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func PermuteStrings(ins []string, randOpt ...Rand) {
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	rnd.Shuffle(len(ins), func(i, j int) {
		ins[i], ins[j] = ins[j], ins[i]
	})
}

// PermuteFloat32s permutes (shuffles) the order of elements in the given float32 slice
// using the standard Fisher-Yates shuffle
// https://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle
// So you don't have to remember how to call rand.Shuffle
// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func PermuteFloat32s(ins []float32, randOpt ...Rand) {
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	rnd.Shuffle(len(ins), func(i, j int) {
		ins[i], ins[j] = ins[j], ins[i]
	})
}

// PermuteFloat64s permutes (shuffles) the order of elements in the given float64 slice
// using the standard Fisher-Yates shuffle
// https://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle
// So you don't have to remember how to call rand.Shuffle
// Optionally can pass a single Rand interface to use --
// otherwise uses system global Rand source.
func PermuteFloat64s(ins []float64, randOpt ...Rand) {
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	rnd.Shuffle(len(ins), func(i, j int) {
		ins[i], ins[j] = ins[j], ins[i]
	})
}
