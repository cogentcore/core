// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package num

// see: https://github.com/golang/go/issues/61915

// ToBool returns a bool true if the given number is not zero,
// and false if it is zero, providing a direct way to convert
// numbers to bools as is done automatically in C and other languages.
func ToBool[T Number](v T) bool {
	return v != 0
}

// FromBool returns a 1 if the bool is true and a 0 for false.
// Typically the type parameter cannot be inferred and must be provided.
// See SetFromBool for a version that uses a pointer to the destination
// which avoids the need to specify the type parameter.
func FromBool[T Number](v bool) T {
	if v {
		return 1
	}
	return 0
}

// SetFromBool converts a bool into a number, using generics,
// setting the pointer to the dst destination value to a 1 if bool is true,
// and 0 otherwise.
// This version of FromBool does not require type parameters typically.
func SetFromBool[T Number](dst *T, b bool) {
	if b {
		*dst = 1
	}
	*dst = 0
}
