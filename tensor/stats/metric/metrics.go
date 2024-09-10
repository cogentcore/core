// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate core generate

package metric

import (
	"fmt"

	"cogentcore.org/core/tensor"
)

// Funcs is a registry of named metric functions,
// which can then be called by standard enum or
// string name for custom functions.
var Funcs map[string]MetricFunc

func init() {
	Funcs = make(map[string]MetricFunc)
}

// Standard calls a standard Metrics enum function on given tensors.
// Output results are in the out tensor.
func Standard(metric Metrics, a, b, out *tensor.Indexed) {
	Funcs[metric.String()](a, b, out)
}

// Call calls a registered stats function on given tensors.
// Output results are in the out tensor.  Returns an
// error if name not found.
func Call(name string, a, b, out *tensor.Indexed) error {
	f, ok := Funcs[name]
	if !ok {
		return fmt.Errorf("metric.Call: function %q not registered", name)
	}
	f(a, b, out)
	return nil
}

// Metrics are standard metric functions
type Metrics int32 //enums:enum

const (
	Euclidean Metrics = iota
	SumSquares
	Abs
	Hamming

	EuclideanBinTol
	SumSquaresBinTol

	// InvCosine is 1-Cosine -- useful to convert into an Increasing metric
	InvCosine

	// InvCorrelation is 1-Correlation -- useful to convert into an Increasing metric
	InvCorrelation

	CrossEntropy

	// Everything below here is !Increasing -- larger = closer, not farther
	InnerProduct
	Covariance
	Correlation
	Cosine
)

// Increasing returns true if the distance metric is such that metric
// values increase as a function of distance (e.g., Euclidean)
// and false if metric values decrease as a function of distance
// (e.g., Cosine, Correlation)
func (m Metrics) Increasing() bool {
	if m >= InnerProduct {
		return false
	}
	return true
}
