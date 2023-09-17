// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package grr provides easy, context-wrapped error handling in Go.
package grr

import (
	"errors"
	"fmt"
	"strings"
)

// Error is the main type of grr and represents an error with
// a base error and a stack trace.
type Error struct {
	Base  error
	Stack []string
}

// Wrap wraps the given error into an error object with
// a stack trace. It returns nil if the given error is nil.
// If it is not nil, the result is guaranteed to be of type [*Error].
func Wrap(err error) error {
	if err == nil {
		return nil
	}
	return &Error{
		Base: err,
	}
}

// New returns a new error with the given text, wrapped with
// a stack trace via [Wrap]. The result guaranteed to be of
// type [*Error]. It is the grr equivalent of [errors.New].
func New(text string) error {
	return Wrap(errors.New(text))
}

// Errorf returns a new error with the given format and arguments,
// wrapped with a stack trace via [Wrap]. The result guaranteed to be of
// type [*Error]. It is the grr equivalent of [fmt.Errorf].
func Errorf(format string, a ...any) error {
	return Wrap(fmt.Errorf(format, a...))
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
