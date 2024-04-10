// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tolassert provides functions for asserting the equality of numbers
// with tolerance (in other words, it checks whether numbers are about equal).
package tolassert

import (
	"cogentcore.org/core/xgo/num"
	"github.com/stretchr/testify/assert"
)

// Equal asserts that the given two numbers are about equal to each other,
// using a default tolerance of 0.001.
func Equal[T num.Float](t assert.TestingT, expected T, actual T, msgAndArgs ...any) bool {
	if h, ok := t.(interface{ Helper() }); ok {
		h.Helper()
	}
	return EqualTol(t, expected, actual, 0.001, msgAndArgs...)
}

// EqualTol asserts that the given two numbers are about equal to each other,
// using the given tolerance value.
func EqualTol[T num.Float](t assert.TestingT, expected T, actual, tolerance T, msgAndArgs ...any) bool {
	if h, ok := t.(interface{ Helper() }); ok {
		h.Helper()
	}
	if num.Abs(actual-expected) > tolerance {
		return assert.Equal(t, expected, actual, msgAndArgs...)
	}
	return true
}
