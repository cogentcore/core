// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package randx provides randomization functionality built on top of standard math/rand
// random number generation functions.
//
// randx.Rand is an interface that enables calling the standard global rand functions,
// or a rand.Rand separate source, and is used for all methods in this package.
// Methods also take a thr thread arg to support a random generator that handles separate
// threads, such as gosl/slrand.
//
// randx.StdRand implements the interface.
//
//   - RandParams: specifies parameters for random number generation according to various distributions,
//     used e.g., for initializing random weights and generating random noise in neurons
//
// - Permute*: basic convenience methods calling rand.Shuffle on e.g., []int slice
//
// - BoolP: boolean for given probability
package randx
