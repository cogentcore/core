// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// package bitflag provides simple bit flag setting, checking, and clearing
// methods that take bit position args as ints (from const int eunum iota's)
// and do the bit shifting from there -- although a tiny bit slower, the
// convenience of maintaining ordinal lists of bit positions greatly outweighs
// that cost -- see kit type registry for further enum management functions
package bitflag

// github.com/rcoreily/ki/bitflag

import (
// "fmt"
)

// todo: can add a global debug level setting and test for overflow in bits --
// or maybe better in the enum type registry constructor?
// see also https://github.com/sirupsen/logrus

// we assume 64bit bitflags by default -- 32 bit methods specifically marked

// set bit value(s) for ordinal bit position flags
func Set(bits *int64, flags ...int) {
	for _, f := range flags {
		*bits |= 1 << uint32(f)
	}
}

// set or clear bit value(s) depending on state (on / off) for ordinal bit position flags
func SetState(bits *int64, state bool, flags ...int) {
	if state {
		Set(bits, flags...)
	} else {
		Clear(bits, flags...)
	}
}

// clear bit value(s) for ordinal bit position flags
func Clear(bits *int64, flags ...int) {
	for _, f := range flags {
		*bits = *bits & ^(1 << uint32(f)) // note: ^ is unary bitwise negation, not ~ as in C
	}
}

// toggle state of bit value(s) for ordinal bit position flags
func Toggle(bits *int64, flags ...int) {
	for _, f := range flags {
		if Has(*bits, f) {
			Clear(bits, f)
		} else {
			Set(bits, f)
		}
	}
}

// check if given bit value is set for ordinal bit position flag
func Has(bits int64, flag int) bool {
	return bits&(1<<uint32(flag)) != 0
}

// check if any of a set of flags are set for ordinal bit position flags
func HasAny(bits int64, flags ...int) bool {
	for _, f := range flags {
		if Has(bits, f) {
			return true
		}
	}
	return false
}

// check if all of a set of flags are set for ordinal bit position flags
func HasAll(bits int64, flags ...int) bool {
	for _, f := range flags {
		if !Has(bits, f) {
			return false
		}
	}
	return true
}

// make a mask for checking multiple different flags
func Mask(flags ...int) int64 {
	var mask int64
	for _, f := range flags {
		Set(&mask, f)
	}
	return mask
}

// check if any of the bits in mask are set
func HasMask(bits, mask int64) bool {
	return bits&mask != 0
}

//////////////////////////////
//   32 bit

// set bit value(s) for ordinal bit position flags
func Set(bits *int64, flags ...int) {
	for _, f := range flags {
		*bits |= 1 << uint32(f)
	}
}

// set or clear bit value(s) depending on state (on / off) for ordinal bit position flags
func SetState(bits *int64, state bool, flags ...int) {
	if state {
		Set(bits, flags...)
	} else {
		Clear(bits, flags...)
	}
}

// clear bit value(s) for ordinal bit position flags
func Clear(bits *int64, flags ...int) {
	for _, f := range flags {
		*bits = *bits & ^(1 << uint32(f)) // note: ^ is unary bitwise negation, not ~ as in C
	}
}

// toggle state of bit value(s) for ordinal bit position flags
func Toggle(bits *int64, flags ...int) {
	for _, f := range flags {
		if Has(*bits, f) {
			Clear(bits, f)
		} else {
			Set(bits, f)
		}
	}
}

// check if given bit value is set for ordinal bit position flag
func Has(bits int64, flag int) bool {
	return bits&(1<<uint32(flag)) != 0
}

// check if any of a set of flags are set for ordinal bit position flags
func HasAny(bits int64, flags ...int) bool {
	for _, f := range flags {
		if Has(bits, f) {
			return true
		}
	}
	return false
}

// check if all of a set of flags are set for ordinal bit position flags
func HasAll(bits int64, flags ...int) bool {
	for _, f := range flags {
		if !Has(bits, f) {
			return false
		}
	}
	return true
}

// make a mask for checking multiple different flags
func Mask(flags ...int) int64 {
	var mask int64
	for _, f := range flags {
		Set(&mask, f)
	}
	return mask
}

// check if any of the bits in mask are set
func HasMask(bits, mask int64) bool {
	return bits&mask != 0
}
