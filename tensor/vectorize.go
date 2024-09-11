// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"math"
	"runtime"
	"sync"
)

var (
	// ThreadingThreshod is the threshold in number of tensor elements
	// to engage actual parallel processing.
	// Heuristically, numbers below this threshold do not result in
	// an overall speedup, due to overhead costs.
	ThreadingThreshold = 100_000

	// NumThreads is the number of threads to use for parallel threading.
	// The default of 0 causes the [runtime.GOMAXPROCS] to be used.
	NumThreads = 0
)

// Vectorize applies given function 'fun' to tensor elements indexed
// by given index, with the 'nfun' providing the number of indexes
// to vectorize over, and initializing any output vectors.
// Thus the nfun is often specific to a particular class of functions.
// Both functions are called with the same set
// of Indexed Tensors passed as the final argument(s).
// The role of each tensor is function-dependent: there could be multiple
// inputs and outputs, and the output could be effectively scalar,
// as in a sum operation.  The interpretation of the index is
// function dependent as well, but often is used to iterate over
// the outer-most row dimension of the tensor.
// This version runs purely sequentially on on this go routine.
// See VectorizeThreaded and VectorizeGPU for other versions.
func Vectorize(nfun func(tsr ...*Indexed) int, fun func(idx int, tsr ...*Indexed), tsr ...*Indexed) {
	n := nfun(tsr...)
	if n <= 0 {
		return
	}
	for idx := range n {
		fun(idx, tsr...)
	}
}

// VectorizeThreaded is a version of [Vectorize] that will automatically
// distribute the computation in parallel across multiple "threads" (goroutines)
// if the number of elements to be computed times the given flops
// (floating point operations) for the function exceeds the [ThreadingThreshold].
// Heuristically, numbers below this threshold do not result
// in an overall speedup, due to overhead costs.
func VectorizeThreaded(flops int, nfun func(tsr ...*Indexed) int, fun func(idx int, tsr ...*Indexed), tsr ...*Indexed) {
	n := nfun(tsr...)
	if n <= 0 {
		return
	}
	if flops < 0 {
		flops = 1
	}
	if n*flops < ThreadingThreshold {
		Vectorize(nfun, fun, tsr...)
		return
	}
	VectorizeOnThreads(0, nfun, fun, tsr...)
}

// DefaultNumThreads returns the default number of threads to use:
// NumThreads if non-zero, otherwise [runtime.GOMAXPROCS].
func DefaultNumThreads() int {
	if NumThreads > 0 {
		return NumThreads
	}
	return runtime.GOMAXPROCS(0)
}

// VectorizeOnThreads runs given [Vectorize] function on given number
// of threads.  Use [VectorizeThreaded] to only use parallel threads when
// it is likely to be beneficial, in terms of the ThreadingThreshold.
// If threads is 0, then the [DefaultNumThreads] will be used:
// GOMAXPROCS subject to NumThreads constraint if non-zero.
func VectorizeOnThreads(threads int, nfun func(tsr ...*Indexed) int, fun func(idx int, tsr ...*Indexed), tsr ...*Indexed) {
	if threads == 0 {
		threads = DefaultNumThreads()
	}
	n := nfun(tsr...)
	if n <= 0 {
		return
	}
	nper := int(math.Ceil(float64(n) / float64(threads)))
	wait := sync.WaitGroup{}
	for start := 0; start < n; start += nper {
		end := start + nper
		if end > n {
			end = n
		}
		wait.Add(1) // todo: move out of loop
		go func() {
			for idx := start; idx < end; idx++ {
				fun(idx, tsr...)
			}
			wait.Done()
		}()
	}
	wait.Wait()
}

// NFirstRows is an N function for Vectorize that returns the number of
// outer-dimension rows (or Indexes) of the first tensor.
func NFirstRows(tsr ...*Indexed) int {
	if len(tsr) == 0 {
		return 0
	}
	return tsr[0].Len()
}

// NFirstLen is an N function for Vectorize that returns the number of
// elements in the tensor, including the Indexes view.
func NFirstLen(tsr ...*Indexed) int {
	if len(tsr) == 0 {
		return 0
	}
	ft := tsr[0]
	_, cells := ft.Tensor.RowCellSize()
	return cells * ft.Len()
}

// NMinNotLast is an N function for Vectorize that returns the min number of
// indexes of all but the last tensor.  This is used when the last tensor is
// the output of the function, operating on the prior vector(s).
func NMinNotLast(tsr ...*Indexed) int {
	nt := len(tsr)
	if nt < 2 {
		return 0
	}
	n := tsr[0].Len()
	for i := 1; i < nt-1; i++ {
		n = min(n, tsr[0].Len())
	}
	return n
}
