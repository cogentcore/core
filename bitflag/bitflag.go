// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// package bitflag provides simple bit flag setting, checking, and clearing
// methods that take bit position args as ints (from const int eunum iota's)
// and do the bit shifting from there -- although a tiny bit slower, the
// convenience of maintaining ordinal lists of bit positions greatly outweighs
// that cost -- see kit type registry for further enum management functions
package bitflag

import "sync/atomic"

// todo: can add a global debug level setting and test for overflow in bits --
// or maybe better in the enum type registry constructor?
// see also https://github.com/sirupsen/logrus

// we assume 64bit bitflags by default -- 32 bit methods specifically marked

// Mask makes a mask for checking multiple different flags
func Mask(flags ...int) int64 {
	var mask int64
	for _, f := range flags {
		mask |= 1 << uint32(f)
	}
	return mask
}

// Set sets bit value(s) for ordinal bit position flags
func Set(bits *int64, flags ...int) {
	mask := Mask(flags...)
	*bits |= mask
}

// SetAtomic sets bit value(s) for ordinal bit position flags, using atomic
// compare-and-swap loop, safe for concurrent access
func SetAtomic(bits *int64, flags ...int) {
	mask := Mask(flags...)
	for {
		cr := *bits
		nw := cr | mask
		if atomic.CompareAndSwapInt64(bits, cr, nw) {
			break
		}
	}
}

// Clear clears bit value(s) for ordinal bit position flags
func Clear(bits *int64, flags ...int) {
	mask := Mask(flags...)
	*bits = *bits & ^mask
}

// ClearAtomic clears bit value(s) for ordinal bit position flags, using atomic
// compare-and-swap loop, safe for concurrent access
func ClearAtomic(bits *int64, flags ...int) {
	mask := Mask(flags...)
	for {
		cr := *bits
		nw := cr & ^mask
		if atomic.CompareAndSwapInt64(bits, cr, nw) {
			break
		}
	}
}

// SetState sets or clears bit value(s) depending on state (on / off) for
// ordinal bit position flags
func SetState(bits *int64, state bool, flags ...int) {
	if state {
		Set(bits, flags...)
	} else {
		Clear(bits, flags...)
	}
}

// SetStateAtomic sets or clears bit value(s) depending on state (on / off)
// for ordinal bit position flags, protected by atomic -- safe for concurrent access
func SetStateAtomic(bits *int64, state bool, flags ...int) {
	if state {
		SetAtomic(bits, flags...)
	} else {
		ClearAtomic(bits, flags...)
	}
}

// Toggle toggles state of bit value(s) for ordinal bit position flags
func Toggle(bits *int64, flags ...int) {
	for _, f := range flags {
		if Has(*bits, f) {
			Clear(bits, f)
		} else {
			Set(bits, f)
		}
	}
}

// Has checks if given bit value is set for ordinal bit position flag
func Has(bits int64, flag int) bool {
	return bits&(1<<uint32(flag)) != 0
}

// HasAtomic checks if given bit value is set for ordinal bit position flag,
// using an atomic load, safe for concurrent access
func HasAtomic(bits *int64, flag int) bool {
	return atomic.LoadInt64(bits)&(1<<uint32(flag)) != 0
}

// HasAny checks if any of a set of flags are set for ordinal bit position flags (logical OR)
func HasAny(bits int64, flags ...int) bool {
	for _, f := range flags {
		if Has(bits, f) {
			return true
		}
	}
	return false
}

// HasAll checks if all of a set of flags are set for ordinal bit position flags (logical AND)
func HasAll(bits int64, flags ...int) bool {
	for _, f := range flags {
		if !Has(bits, f) {
			return false
		}
	}
	return true
}

// HasMask checks if any of the bits in mask are set
func HasMask(bits, mask int64) bool {
	return bits&mask != 0
}

// ClearMask clears all of the bits in the mask
func ClearMask(bits *int64, mask int64) {
	*bits = *bits & ^mask
}

/////////////////////////////////////////////////////////////////////////////////
//   32 bit

// Mask32 makes a mask for checking multiple different flags
func Mask32(flags ...int) int32 {
	var mask int32
	for _, f := range flags {
		mask |= 1 << uint32(f)
	}
	return mask
}

// Set32 sets bit value(s) for ordinal bit position flags
func Set32(bits *int32, flags ...int) {
	mask := Mask32(flags...)
	*bits |= mask
}

// SetState32 sets or clears bit value(s) depending on state (on / off) for
// ordinal bit position flags
func SetState32(bits *int32, state bool, flags ...int) {
	if state {
		Set32(bits, flags...)
	} else {
		Clear32(bits, flags...)
	}
}

// Clear32 clears bit value(s) for ordinal bit position flags
func Clear32(bits *int32, flags ...int) {
	mask := Mask32(flags...)
	*bits = *bits & ^mask
}

// Toggle32 toggles state of bit value(s) for ordinal bit position flags
func Toggle32(bits *int32, flags ...int) {
	for _, f := range flags {
		if Has32(*bits, f) {
			Clear32(bits, f)
		} else {
			Set32(bits, f)
		}
	}
}

// Has32 checks if given bit value is set for ordinal bit position flag
func Has32(bits int32, flag int) bool {
	return bits&(1<<uint32(flag)) != 0
}

// HasAny32 checks if any of a set of flags are set for ordinal bit position flags
func HasAny32(bits int32, flags ...int) bool {
	for _, f := range flags {
		if Has32(bits, f) {
			return true
		}
	}
	return false
}

// HasAll32 checks if all of a set of flags are set for ordinal bit position flags
func HasAll32(bits int32, flags ...int) bool {
	for _, f := range flags {
		if !Has32(bits, f) {
			return false
		}
	}
	return true
}

// HasMask32 checks if any of the bits in mask are set
func HasMask32(bits, mask int32) bool {
	return bits&mask != 0
}

// ClearMask32 clears all of the bits in the mask
func ClearMask32(bits *int32, mask int32) {
	*bits = *bits & ^mask
}
