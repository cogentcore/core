// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package ints provides a standard Inter interface and basic functions
defined on Inter types that support core things like Max, Min, Abs.
Furthermore, fully generic slice sort and conversion methods in the kit type
kit package attempt to use this interface, before falling back on reflection.
If you have a struct that can be converted into an int64, then this is the
only way to allow it to be sorted using those generic functions, as the
reflect.Kind fallback will fail.

It also includes Max, Min, Abs for builtin int64, int32 types.
*/
package ints

// Inter converts a type from an int64, used in kit.ToInt and in sorting
// comparisons.  See also Floater in floats package.
type Inter interface {
	Int() int64
}

// IntSetter is an Inter that can also be set from an int.  Satisfying this
// interface requires a pointer to the underlying type.
type IntSetter interface {
	Inter
	FromInt(val int64)
}

////////////////////////////////////
//   Inter

// Max computes the maximum of the two Inter args
func Max(a, b Inter) Inter {
	if a.Int() > b.Int() {
		return a
	}
	return b
}

// Min computes the minimum of the two Inter args
func Min(a, b Inter) Inter {
	if a.Int() < b.Int() {
		return a
	}
	return b
}

// Abs computes the absolute value of the given value
func Abs(a Inter) int64 {
	if a.Int() < 0 {
		return -a.Int()
	}
	return a.Int()
}

////////////////////////////////////
//   int

// MaxInt computes the maximum of the two int args
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MinInt computes the minimum of the two int args
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// AbsInt computes the absolute value of the given value
func AbsInt(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// ClipInt clips int within min, max range (max exclusive, min inclusive)
func ClipInt(a, min, max int) int {
	if a >= max {
		return max - 1
	}
	if a < min {
		return min
	}
	return a
}

////////////////////////////////////
//   int64

// Max64 computes the maximum of the two int64 args
func Max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// Min64 computes the minimum of the two int64 args
func Min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// Abs64 computes the absolute value of the given value
func Abs64(a int64) int64 {
	if a < 0 {
		return -a
	}
	return a
}

////////////////////////////////////
//   int32

// Max32 computes the maximum of the two int32 args
func Max32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

// Min32 computes the minimum of the two int32 args
func Min32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

// Abs32 computes the absolute value of the given value
func Abs32(a int32) int32 {
	if a < 0 {
		return -a
	}
	return a
}
