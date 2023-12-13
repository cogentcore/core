// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package option provides optional (nullable) types.
package option

// Option represents an optional (nullable) type. If Valid is true, Option
// represents Value. Otherwise, it represents a null/unset/invalid value.
type Option[T any] struct {
	Value T
	Valid bool
}

// Set sets the value to the given value.
func (o *Option[T]) Set(v T) *Option[T] {
	o.Value = v
	o.Valid = true
	return o
}

// Clear marks the value as null/unset/invalid.
func (o *Option[T]) Clear() *Option[T] {
	o.Valid = false
	return o
}
