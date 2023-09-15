// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package num

// Convert converts any number to any other, using generics,
// with the return number specified by the first type argument,
// and the source number by the second.
// Typically the source type can be inferred but the return cannot.
// See SetNumber for a version that uses a pointer to the destination
// which avoids the need to specify the type parameter.
func Convert[Tr Number, Ts Number](src Ts) Tr {
	return Tr(src)
}

// SetNumber converts any number to any other, using generics,
// setting the pointer to the dst destination value to the src value.
// This version of Convert does not require type parameters typically.
func SetNumber[Td Number, Ts Number](dst *Td, src Ts) { *dst = Td(src) }
