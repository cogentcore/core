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

////////////////////////////////////////////////////////////////////////
//  Core Mask Impl methods, take the full bitmask

// SetMask sets bits in mask
func SetMask(bits *int64, mask int64) {
	*bits |= mask
}

// SetMaskAtomic sets bits in mask
// using atomic compare-and-swap loop, safe for concurrent access
func SetMaskAtomic(bits *int64, mask int64) {
	for {
		cr := atomic.LoadInt64(bits)
		nw := cr | mask
		if atomic.CompareAndSwapInt64(bits, cr, nw) {
			break
		}
	}
}

// ClearMask clears all of the bits in the mask
func ClearMask(bits *int64, mask int64) {
	*bits &^= mask
}

// ClearMaskAtomic clears all of the bits in the mask
// using atomic compare-and-swap loop, safe for concurrent access
func ClearMaskAtomic(bits *int64, mask int64) {
	for {
		cr := atomic.LoadInt64(bits)
		nw := cr &^ mask
		if atomic.CompareAndSwapInt64(bits, cr, nw) {
			break
		}
	}
}

// HasAnyMask checks if *any* of the bits in mask are set (logical OR)
func HasAnyMask(bits, mask int64) bool {
	return bits&mask != 0
}

// HasAllMask checks if *all* of the bits in mask are set (logical AND)
func HasAllMask(bits, mask int64) bool {
	return bits&mask == mask
}

// HasAnyMaskAtomic checks if *any* of the bits in mask are set (logical OR)
// using atomic compare-and-swap loop, safe for concurrent access
func HasAnyMaskAtomic(bits *int64, mask int64) bool {
	return (atomic.LoadInt64(bits) & mask) != 0
}

// HasAllMaskAtomic checks if *all* of the bits in mask are set (logical AND)
// using atomic compare-and-swap loop, safe for concurrent access
func HasAllMaskAtomic(bits *int64, mask int64) bool {
	return (atomic.LoadInt64(bits) & mask) == mask
}

////////////////////////////////////////////////////////////////////////
//  Convenience methods for ordinal bitflags

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
	SetMask(bits, Mask(flags...))
}

// SetAtomic sets bit value(s) for ordinal bit position flags
// using atomic compare-and-swap loop, safe for concurrent access
func SetAtomic(bits *int64, flags ...int) {
	SetMaskAtomic(bits, Mask(flags...))
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

// Clear clears bit value(s) for ordinal bit position flags
func Clear(bits *int64, flags ...int) {
	ClearMask(bits, Mask(flags...))
}

// ClearAtomic clears bit value(s) for ordinal bit position flags
// using atomic compare-and-swap loop, safe for concurrent access
func ClearAtomic(bits *int64, flags ...int) {
	ClearMaskAtomic(bits, Mask(flags...))
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

// HasAny checks if *any* of a set of flags are set for ordinal bit position flags (logical OR)
func HasAny(bits int64, flags ...int) bool {
	return HasAnyMask(bits, Mask(flags...))
}

// HasAll checks if *all* of a set of flags are set for ordinal bit position flags (logical AND)
func HasAll(bits int64, flags ...int) bool {
	return HasAllMask(bits, Mask(flags...))
}

// HasAnyAtomic checks if *any* of a set of flags are set for ordinal bit position flags (logical OR)
// using atomic compare-and-swap loop, safe for concurrent access
func HasAnyAtomic(bits *int64, flags ...int) bool {
	return HasAnyMaskAtomic(bits, Mask(flags...))
}

// HasAllAtomic checks if *all* of a set of flags are set for ordinal bit position flags (logical AND)
// using atomic compare-and-swap loop, safe for concurrent access
func HasAllAtomic(bits *int64, flags ...int) bool {
	return HasAllMaskAtomic(bits, Mask(flags...))
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

// ToggleAtomic toggles state of bit value(s) for ordinal bit position flags
// using atomic compare-and-swap loop, safe for concurrent access, but sequentially
func ToggleAtomic(bits *int64, flags ...int) {
	for _, f := range flags {
		if HasAtomic(bits, f) {
			ClearAtomic(bits, f)
		} else {
			SetAtomic(bits, f)
		}
	}
}

/////////////////////////////////////////////////////////////////////////////////
//   32 bit, core mask impls

// SetMask32 sets bits in mask
func SetMask32(bits *int32, mask int32) {
	*bits |= mask
}

// SetMaskAtomic32 sets bits in mask
// using atomic compare-and-swap loop, safe for concurrent access
func SetMaskAtomic32(bits *int32, mask int32) {
	for {
		cr := atomic.LoadInt32(bits)
		nw := cr | mask
		if atomic.CompareAndSwapInt32(bits, cr, nw) {
			break
		}
	}
}

// ClearMask32 clears all of the bits in the mask
func ClearMask32(bits *int32, mask int32) {
	*bits &^= mask
}

// ClearMaskAtomic32 clears all of the bits in the mask
// using atomic compare-and-swap loop, safe for concurrent access
func ClearMaskAtomic32(bits *int32, mask int32) {
	for {
		cr := atomic.LoadInt32(bits)
		nw := cr &^ mask
		if atomic.CompareAndSwapInt32(bits, cr, nw) {
			break
		}
	}
}

// HasAnyMask32 checks if *any* of the bits in mask are set (logical OR)
func HasAnyMask32(bits, mask int32) bool {
	return bits&mask != 0
}

// HasAllMask32 checks if *all* of the bits in mask are set (logical AND)
func HasAllMask32(bits, mask int32) bool {
	return bits&mask == mask
}

// HasAnyMaskAtomic32 checks if *any* of the bits in mask are set (logical OR)
// using atomic compare-and-swap loop, safe for concurrent access
func HasAnyMaskAtomic32(bits *int32, mask int32) bool {
	return (atomic.LoadInt32(bits) & mask) != 0
}

// HasAllMaskAtomic32 checks if *all* of the bits in mask are set (logical AND)
// using atomic compare-and-swap loop, safe for concurrent access
func HasAllMaskAtomic32(bits *int32, mask int32) bool {
	return (atomic.LoadInt32(bits) & mask) == mask
}

////////////////////////////////////////////////////////////////////////
//  Convenience methods for ordinal bitflags

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
	SetMask32(bits, Mask32(flags...))
}

// SetAtomic32 sets bit value(s) for ordinal bit position flags
// using atomic compare-and-swap loop, safe for concurrent access
func SetAtomic32(bits *int32, flags ...int) {
	SetMaskAtomic32(bits, Mask32(flags...))
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

// SetStateAtomic32 sets or clears bit value(s) depending on state (on / off)
// for ordinal bit position flags, protected by atomic -- safe for concurrent access
func SetStateAtomic32(bits *int32, state bool, flags ...int) {
	if state {
		SetAtomic32(bits, flags...)
	} else {
		ClearAtomic32(bits, flags...)
	}
}

// Clear32 clears bit value(s) for ordinal bit position flags
func Clear32(bits *int32, flags ...int) {
	ClearMask32(bits, Mask32(flags...))
}

// ClearAtomic32 clears bit value(s) for ordinal bit position flags
// using atomic compare-and-swap loop, safe for concurrent access
func ClearAtomic32(bits *int32, flags ...int) {
	ClearMaskAtomic32(bits, Mask32(flags...))
}

// Has32 checks if given bit value is set for ordinal bit position flag
func Has32(bits int32, flag int) bool {
	return bits&(1<<uint32(flag)) != 0
}

// HasAtomic32 checks if given bit value is set for ordinal bit position flag,
// using an atomic load, safe for concurrent access
func HasAtomic32(bits *int32, flag int) bool {
	return atomic.LoadInt32(bits)&(1<<uint32(flag)) != 0
}

// HasAny32 checks if *any* of a set of flags are set for ordinal bit position flags (logical OR)
func HasAny32(bits int32, flags ...int) bool {
	return HasAnyMask32(bits, Mask32(flags...))
}

// HasAll32 checks if *all* of a set of flags are set for ordinal bit position flags (logical AND)
func HasAll32(bits int32, flags ...int) bool {
	return HasAllMask32(bits, Mask32(flags...))
}

// HasAnyAtomic32 checks if *any* of a set of flags are set for ordinal bit position flags (logical OR)
// using atomic compare-and-swap loop, safe for concurrent access
func HasAnyAtomic32(bits *int32, flags ...int) bool {
	return HasAnyMaskAtomic32(bits, Mask32(flags...))
}

// HasAllAtomic32 checks if *all* of a set of flags are set for ordinal bit position flags (logical AND)
// using atomic compare-and-swap loop, safe for concurrent access
func HasAllAtomic32(bits *int32, flags ...int) bool {
	return HasAllMaskAtomic32(bits, Mask32(flags...))
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

// ToggleAtomic32 toggles state of bit value(s) for ordinal bit position flags
// using atomic compare-and-swap loop, safe for concurrent access, but sequentially
func ToggleAtomic32(bits *int32, flags ...int) {
	for _, f := range flags {
		if HasAtomic32(bits, f) {
			ClearAtomic32(bits, f)
		} else {
			SetAtomic32(bits, f)
		}
	}
}
