// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package enums provides common interfaces for enums
// and bit flag enums and utilities for using them
package enums

import "fmt"

// Enum is the interface that all enum types satisfy.
// Enum types must be convertable to strings and int64s,
// must be able to return a description of their value,
// must be able to report if they are valid, and must
// be able to return all possible enum values
// and string and description representations.
type Enum interface {
	fmt.Stringer
	// Int64 returns the enum value as an int64.
	Int64() int64
	// Desc returns the description of the enum value.
	Desc() string
	// IsValid returns whether the value is a
	// valid option for its enum type.
	IsValid() bool
	// Values returns all possible values this
	// enum type has. This slice will be in the
	// same order as those returned by Strings and Descs.
	Values() []Enum
	// Strings returns the string representations of
	// all possible values this enum type has.
	// This slice will be in the same order as
	// those returned by Values and Descs.
	Strings() []string
	// Descs returns the descriptions of all
	// possible values this enum type has.
	// This slice will be in the same order as
	// those returned by Values and Strings.
	Descs() []string
}

// EnumSetter is an expanded interface that all pointers
// to enum types satisfy. Pointers to enum types must
// satisfy all of the methods of [Enum], and must also
// be settable from strings and int64s.
type EnumSetter interface {
	Enum
	// SetString sets the enum value from its
	// string representation, and returns an
	// error if the string is invalid.
	SetString(s string) error
	// SetInt64 sets the enum value from an int64.
	SetInt64(i int64)
}

// BitFlag is the interface that all bit flag enum types
// satisfy. Bit flag enum types support all of the operations
// that standard enums do, and additionally can check if they
// have a given bit flag.
type BitFlag interface {
	Enum
	// HasFlag returns whether these flags
	// have the given flag set.
	HasFlag(f BitFlag) bool
}

// BitFlagSetter is an expanded interface that all pointers
// to bit flag enum types satisfy. Pointers to bit flag
// enum types must  satisfy all of the methods of [EnumSetter]
// and [BitFlag], and must also be able to set a given bit flag.
type BitFlagSetter interface {
	EnumSetter
	BitFlag
	// SetFlag sets the value of the given
	// flags in these flags to the given value.
	SetFlag(on bool, f ...BitFlag)
}
