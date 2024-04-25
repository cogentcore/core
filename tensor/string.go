// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

// NewString returns a new n-dimensional tensor of string values
// with the given sizes per dimension (shape), and optional dimension names.
// Nulls are initialized to nil.
func NewString(sizes []int, names ...string) *Float64 {
	tsr := &Float64{} // todo: replace with String once written
	tsr.SetShape(sizes, names...)
	tsr.Values = make([]float64, tsr.Len())
	return tsr
}
