// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package enums provides common interfaces for enums
// and bit flag enums and utilities for using them
package enums

import "fmt"

// Enumer is the interface that all enum types satisfy.
// Enum types must be convertable to and from strings
// and int64s, and must be able to return all possible
// enum values and string representations.
type Enumer interface {
	fmt.Stringer
	// SetString sets the enum value from its
	// string representation, and returns an
	// error if the string is invalid.
	SetString(s string) error
	// Int64 returns the enum value as an int64.
	Int64() int64
	// SetInt64 sets the enum value from an int64.
	SetInt64(i int64)
	// Desc returns the description of the enum value.
	Desc() string
	// Values returns all possible values this
	// enum type has. This slice will be in the
	// same order as those returned by [Enumer.Strings]
	// and [Enumer.Descs].
	Values() []Enumer
	// Strings returns the string encodings of
	// all possible values this enum type has.
	// This slice will be in the same order as
	// those returned by [Enumer.Values]
	// and [Enumer.Descs].
	Strings() []string
	// Descs returns the descriptions of all
	// possible values this enum type has.
	// This slice will be in the same order as
	// those returned by [Enumer.Values]
	// and [Enumer.Strings].
	Descs() []string
}

// BitEnumer is the interface that all bit flag enum types
// satisfy. Bit flag enum types support all of the operations
// that standard enums do, and additionally can get and set
// bit flags.
type BitEnumer interface {
	Enumer
	// HasBitFlag returns whether these flags
	// have the given flag set.
	HasBitFlag(f BitEnumer) bool
	// SetBitFlag sets the value of the given
	// flag in these flags to the given value.
	SetBitFlag(f BitEnumer, b bool)
}
