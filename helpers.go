// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grr

import (
	"log/slog"
)

// Log takes the given value and error and returns the value if
// the error is nil, and logs the error and returns a zero value
// if the error is non-nil. The intended usage is:
//
//	a := grr.Log(MyFunc(v))
func Log[T any](v T, err error) T {
	if err != nil {
		slog.Error(err.Error())
	}
	return v
}

// Log2 takes the given two values and error and returns the values if
// the error is nil, and logs the error and returns zero values
// if the error is non-nil. The intended usage is:
//
//	a, b := grr.Log2(MyFunc(v))
func Log2[T1, T2 any](v1 T1, v2 T2, err error) (T1, T2) {
	if err != nil {
		slog.Error(err.Error())
	}
	return v1, v2
}

// Log3 takes the given three values and error and returns the values if
// the error is nil, and logs the error and returns zero values
// if the error is non-nil. The intended usage is:
//
//	a, b, c := grr.Log3(MyFunc(v))
func Log3[T1, T2, T3 any](v1 T1, v2 T2, v3 T3, err error) (T1, T2, T3) {
	if err != nil {
		slog.Error(err.Error())
	}
	return v1, v2, v3
}

// Log4 takes the given four values and error and returns the values if
// the error is nil, and logs the error and returns zero values
// if the error is non-nil. The intended usage is:
//
//	a, b, c, d := grr.Log4(MyFunc(v))
func Log4[T1, T2, T3, T4 any](v1 T1, v2 T2, v3 T3, v4 T4, err error) (T1, T2, T3, T4) {
	if err != nil {
		slog.Error(err.Error())
	}
	return v1, v2, v3, v4
}

// Must takes the given value and error and returns the value if
// the error is nil, and panics if the error is non-nil. The intended usage is:
//
//	a := grr.Must(MyFunc(v))
func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// Must2 takes the given two values and error and returns the values if
// the error is nil, and panics if the error is non-nil. The intended usage is:
//
//	a, b := grr.Must2(MyFunc(v))
func Must2[T1, T2 any](v1 T1, v2 T2, err error) (T1, T2) {
	if err != nil {
		panic(err)
	}
	return v1, v2
}

// Must3 takes the given three values and error and returns the values if
// the error is nil, and panics if the error is non-nil. The intended usage is:
//
//	a, b, c := grr.Must3(MyFunc(v))
func Must3[T1, T2, T3 any](v1 T1, v2 T2, v3 T3, err error) (T1, T2, T3) {
	if err != nil {
		panic(err)
	}
	return v1, v2, v3
}

// Must4 takes the given four values and error and returns the values if
// the error is nil, and panics if the error is non-nil. The intended usage is:
//
//	a, b, c, d := grr.Must4(MyFunc(v))
func Must4[T1, T2, T3, T4 any](v1 T1, v2 T2, v3 T3, v4 T4, err error) (T1, T2, T3, T4) {
	if err != nil {
		panic(err)
	}
	return v1, v2, v3, v4
}
