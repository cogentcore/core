// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"math"
	"sync"
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

func VectorizeThreaded(threads int, nfun func(tsr ...*Indexed) int, fun func(idx int, tsr ...*Indexed), tsr ...*Indexed) {
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

// NFirst is an N function for Vectorize that returns the number of
// indexes of the first tensor.
func NFirst(tsr ...*Indexed) int {
	if len(tsr) == 0 {
		return 0
	}
	return tsr[0].Len()
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
