// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package norm

import (
	"fmt"
	"log/slog"
	"math"
	"reflect"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/tensor"
)

// Abs32 applies the Abs function to each element in given slice
func Abs32(a []float32) {
	for i, av := range a {
		if math32.IsNaN(av) {
			continue
		}
		a[i] = math32.Abs(av)
	}
}

// Abs64 applies the Abs function to each element in given slice
func Abs64(a []float64) {
	for i, av := range a {
		if math.IsNaN(av) {
			continue
		}
		a[i] = math.Abs(av)
	}
}

func FloatOnlyError() error {
	err := fmt.Errorf("Only float32 or float64 data types supported")
	slog.Error(err.Error())
	return err
}

// TensorAbs applies the Abs function to each element in given tensor,
// for float32 and float64 data types.
func TensorAbs(a tensor.Tensor) {
	switch a.DataType() {
	case reflect.Float32:
		Abs32(a.(*tensor.Number[float32]).Values)
	case reflect.Float64:
		Abs64(a.(*tensor.Number[float64]).Values)
	default:
		FloatOnlyError()
	}
}
