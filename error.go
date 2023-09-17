// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grr

import "strings"

// Error is the main type of grr and represents an error with
// a base error and a stack trace.
type Error struct {
	Base  error
	Stack []string
}

// Error returns the error as a string, wrapping the string of
// the base error with the stack trace.
func (e *Error) Error() string {
	res := e.Base.Error()
	if len(e.Stack) > 0 {
		res += " (" + strings.Join(e.Stack, ": ") + ")"
	}
	return res
}

// String returns the error as a string, wrapping the string of
// the base error with the stack trace.
func (e *Error) String() string {
	return e.Error()
}

// Unwrap returns the underlying base error of the Error.
func (e *Error) Unwrap() error {
	return e.Base
}
