// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tmath

import (
	"testing"

	"cogentcore.org/core/tensor"
	"github.com/stretchr/testify/assert"
)

func TestBoolOps(t *testing.T) {
	ar := tensor.NewSliceInts(12)
	// fmt.Println(v)
	bo := tensor.NewBool()
	sc := tensor.NewIntScalar(6)

	EqualOut(ar, sc, bo)
	for i, v := range ar.Values {
		assert.Equal(t, v == 6, bo.Bool1D(i))
	}

	LessOut(ar, sc, bo)
	for i, v := range ar.Values {
		assert.Equal(t, v < 6, bo.Bool1D(i))
	}

	GreaterOut(ar, sc, bo)
	// fmt.Println(bo)
	for i, v := range ar.Values {
		assert.Equal(t, v > 6, bo.Bool1D(i))
	}

	NotEqualOut(ar, sc, bo)
	for i, v := range ar.Values {
		assert.Equal(t, v != 6, bo.Bool1D(i))
	}

	LessEqualOut(ar, sc, bo)
	for i, v := range ar.Values {
		assert.Equal(t, v <= 6, bo.Bool1D(i))
	}

	GreaterEqualOut(ar, sc, bo)
	// fmt.Println(bo)
	for i, v := range ar.Values {
		assert.Equal(t, v >= 6, bo.Bool1D(i))
	}

}
