// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package vector provides standard vector math functions that
// always operate on 1D views of tensor inputs regardless of the original
// vector shape.
package vector

import (
	"math"

	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/tmath"
)

// Inner multiplies two vectors element-wise, returning the vector.
func Inner(a, b tensor.Tensor) tensor.Values {
	return tensor.CallOut2Float64(InnerOut, a, b)
}

// InnerOut multiplies two vectors element-wise.
func InnerOut(a, b tensor.Tensor, out tensor.Values) error {
	return tmath.MulOut(tensor.As1D(a), tensor.As1D(b), out)
}

// Sum returns the sum of all values in the tensor, as a scalar.
func Sum(a tensor.Tensor) tensor.Values {
	n := a.Len()
	sum := 0.0
	tensor.Vectorize(func(tsr ...tensor.Tensor) int { return n },
		func(idx int, tsr ...tensor.Tensor) {
			sum += tsr[0].Float1D(idx)
		}, a)
	return tensor.NewFloat64Scalar(sum)
}

// Dot performs a dot product, returning a scalar dot product.
func Dot(a, b tensor.Tensor) tensor.Values {
	return Sum(Inner(a, b))
}

// NormL2 returns the length of the vector as the L2Norm:
// square root of the sum of squared values of the vector, as a scalar.
func NormL2(a tensor.Tensor) tensor.Values {
	sum := Sum(Inner(a, a)).Float1D(0)
	return tensor.NewFloat64Scalar(math.Sqrt(sum))
}

// NormL1 returns the length of the vector as the L1Norm:
// sum of the absolute values of the tensor, as a scalar.
func NormL1(a tensor.Tensor) tensor.Values {
	n := a.Len()
	sum := 0.0
	tensor.Vectorize(func(tsr ...tensor.Tensor) int { return n },
		func(idx int, tsr ...tensor.Tensor) {
			sum += math.Abs(tsr[0].Float1D(idx))
		}, a)
	return tensor.NewFloat64Scalar(sum)
}
