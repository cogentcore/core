// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package option provides optional (nullable) types.
package option

// Option represents an optional (nullable) type. If Valid is true, Option
// represents Value. Otherwise, it represents a null/unset/invalid value.
type Option[T any] struct {
	Valid bool `label:"Set"`
	Value T
}

// New returns a new [Option] set to the given value.
func New[T any](v T) *Option[T] {
	o := &Option[T]{}
	o.Set(v)
	return o
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

// Or returns the value of the option if it is not marked
// as null/unset/invalid, and otherwise it returns the given value.
func (o *Option[T]) Or(or T) T {
	if o.Valid {
		return o.Value
	}
	return or
}

func (o *Option[T]) ShouldSave() bool {
	return o.Valid
}

func (o *Option[T]) ShouldDisplay(field string) bool {
	switch field {
	case "Value":
		return o.Valid
	}
	return true
}
