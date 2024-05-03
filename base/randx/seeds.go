// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package randx

import (
	"time"
)

// Seeds is a set of random seeds, typically used one per Run
type Seeds []int64

// Init allocates given number of seeds and initializes them to
// sequential numbers 1..n
func (rs *Seeds) Init(n int) {
	*rs = make([]int64, n)
	for i := range *rs {
		(*rs)[i] = int64(i) + 1
	}
}

// Set sets the given seed to either the single Rand
// interface passed, or the system global Rand source.
func (rs *Seeds) Set(idx int, randOpt ...Rand) {
	var rnd Rand
	if len(randOpt) == 0 {
		rnd = NewGlobalRand()
	} else {
		rnd = randOpt[0]
	}
	rnd.Seed((*rs)[idx])
}

// NewSeeds sets a new set of random seeds based on current time
func (rs *Seeds) NewSeeds() {
	rn := time.Now().UnixNano()
	for i := range *rs {
		(*rs)[i] = rn + int64(i)
	}
}
