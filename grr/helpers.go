// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grr

import (
	"log/slog"
)

// Log takes the given error and logs it if it is non-nil.
// The intended usage is:
//
//	grr.Log(MyFunc(v))
//	// or
//	return grr.Log(MyFunc(v))
func Log(err error) error {
	if err != nil {
		slog.Error(err.Error())
	}
	return err
}

// Log1 takes the given value and error and returns the value if
// the error is nil, and logs the error and returns a zero value
// if the error is non-nil. The intended usage is:
//
//	a := grr.Log1(MyFunc(v))
func Log1[T any](v T, err error) T {
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

// Must takes the given error and panics if it is non-nil.
// The intended usage is:
//
//	grr.Must(MyFunc(v))
func Must(err error) {
	if err != nil {
		panic(err)
	}
}

// Must1 takes the given value and error and returns the value if
// the error is nil, and panics if the error is non-nil. The intended usage is:
//
//	a := grr.Must1(MyFunc(v))
func Must1[T any](v T, err error) T {
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

// TestingT is an interface wrapper around [*testing.T]
type TestingT interface {
	Error(args ...any)
	Fatal(args ...any)

	// Helper marks the calling function as a test helper function.
	Helper()
}

// Test takes the given error and errors the test it if it is non-nil.
// The intended usage is:
//
//	grr.Test(t, MyFunc(v))
func Test(t TestingT, err error) error {
	t.Helper()
	if err != nil {
		t.Error(err)
	}
	return err
}

// TestFatal takes the given error and fatally errors the test it if it is non-nil.
// The intended usage is:
//
//	grr.TestFatal(t, MyFunc(v))
func TestFatal(t TestingT, err error) error {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	return err
}

// Ignore1 ignores an error return value for a function returning
// a value and an error, allowing direct usage of the value.
// The intended usage is:
//
//	a := grr.Ignore1(MyFunc(v))
func Ignore1[T any](v T, err error) T {
	return v
}

// Ignore2 ignores an error return value for a function returning
// two values and an error, allowing direct usage of the values.
// The intended usage is:
//
//	a, b := grr.Ignore2(MyFunc(v))
func Ignore2[T1, T2 any](v1 T1, v2 T2, err error) (T1, T2) {
	return v1, v2
}

// Ignore3 ignores an error return value for a function returning
// three values and an error, allowing direct usage of the values.
// The intended usage is:
//
//	a, b, c := grr.Ignore3(MyFunc(v))
func Ignore3[T1, T2, T3 any](v1 T1, v2 T2, v3 T3, err error) (T1, T2, T3) {
	return v1, v2, v3
}

// Note: errors.Join is the std way to do AllErrors
