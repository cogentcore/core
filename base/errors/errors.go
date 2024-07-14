// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package errors provides a set of error handling helpers,
// extending the standard library errors package.
package errors

import (
	"log/slog"
	"runtime"
	"strconv"
)

// Log takes the given error and logs it if it is non-nil.
// The intended usage is:
//
//	errors.Log(MyFunc(v))
//	// or
//	return errors.Log(MyFunc(v))
func Log(err error) error {
	if err != nil {
		slog.Error(err.Error() + " | " + CallerInfo())
	}
	return err
}

// Log1 takes the given value and error and returns the value if
// the error is nil, and logs the error and returns a zero value
// if the error is non-nil. The intended usage is:
//
//	a := errors.Log1(MyFunc(v))
func Log1[T any](v T, err error) T { //yaegi:add
	if err != nil {
		slog.Error(err.Error() + " | " + CallerInfo())
	}
	return v
}

// Log2 takes the given two values and error and returns the values if
// the error is nil, and logs the error and returns zero values
// if the error is non-nil. The intended usage is:
//
//	a, b := errors.Log2(MyFunc(v))
func Log2[T1, T2 any](v1 T1, v2 T2, err error) (T1, T2) {
	if err != nil {
		slog.Error(err.Error() + " | " + CallerInfo())
	}
	return v1, v2
}

// Must takes the given error and panics if it is non-nil.
// The intended usage is:
//
//	errors.Must(MyFunc(v))
func Must(err error) {
	if err != nil {
		panic(err)
	}
}

// Must1 takes the given value and error and returns the value if
// the error is nil, and panics if the error is non-nil. The intended usage is:
//
//	a := errors.Must1(MyFunc(v))
func Must1[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// Must2 takes the given two values and error and returns the values if
// the error is nil, and panics if the error is non-nil. The intended usage is:
//
//	a, b := errors.Must2(MyFunc(v))
func Must2[T1, T2 any](v1 T1, v2 T2, err error) (T1, T2) {
	if err != nil {
		panic(err)
	}
	return v1, v2
}

// Ignore1 ignores an error return value for a function returning
// a value and an error, allowing direct usage of the value.
// The intended usage is:
//
//	a := errors.Ignore1(MyFunc(v))
func Ignore1[T any](v T, err error) T {
	return v
}

// Ignore2 ignores an error return value for a function returning
// two values and an error, allowing direct usage of the values.
// The intended usage is:
//
//	a, b := errors.Ignore2(MyFunc(v))
func Ignore2[T1, T2 any](v1 T1, v2 T2, err error) (T1, T2) {
	return v1, v2
}

// CallerInfo returns string information about the caller
// of the function that called CallerInfo.
func CallerInfo() string {
	pc, file, line, _ := runtime.Caller(2)
	return runtime.FuncForPC(pc).Name() + " " + file + ":" + strconv.Itoa(line)
}
