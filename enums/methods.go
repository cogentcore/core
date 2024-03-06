// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enums

import (
	"strconv"

	"cogentcore.org/core/glop/num"
)

// This file contains implementations of enumgen methods.

// EnumConstraint is the generic type constraint that all enums satisfy.
type EnumConstraint interface {
	Enum
	num.Integer
}

// BitFlagConstraint is the generic type constraint that all bit flags satisfy.
type BitFlagConstraint interface {
	BitFlag
	num.Integer
}

// String returns the string representation of the given
// enum with the given map.
func String[T EnumConstraint](i T, m map[T]string) string {
	if str, ok := m[i]; ok {
		return str
	}
	return strconv.FormatInt(int64(i), 10)
}

// StringExtended returns the string representation of the given enum
// with the given map, with the enum extending the given other enum type.
func StringExtended[T, E EnumConstraint](i T, m map[T]string) string {
	if str, ok := m[i]; ok {
		return str
	}
	return E(i).String()
}

// BitIndexStringExtended returns the string representation of the given enum
// bit index value with the given map, with the enum extending the given other
// enum type.
func BitIndexStringExtended[T, E BitFlagConstraint](i T, m map[T]string) string {
	if str, ok := m[i]; ok {
		return str
	}
	return E(i).BitIndexString()
}
