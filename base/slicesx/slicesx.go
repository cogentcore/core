// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package slicesx provides additional slice functions
// beyond those in the standard [slices] package.
package slicesx

import (
	"slices"
	"unsafe"
)

// GrowTo increases the slice's capacity, if necessary,
// so that it can hold at least n elements.
func GrowTo[S ~[]E, E any](s S, n int) S {
	if n < 0 {
		panic("cannot be negative")
	}
	if n -= cap(s); n > 0 {
		s = append(s[:cap(s)], make([]E, n)...)[:len(s)]
	}
	return s
}

// SetLength sets the length of the given slice,
// re-using and preserving existing values to the extent possible.
func SetLength[E any](s []E, n int) []E {
	if len(s) == n {
		return s
	}
	if s == nil {
		return make([]E, n)
	}
	if cap(s) < n {
		s = GrowTo(s, n)
	}
	s = s[:n]
	return s
}

// CopyFrom efficiently copies from src into dest, using SetLength
// to ensure the destination has sufficient capacity, and returns
// the destination (which may have changed location as a result).
func CopyFrom[E any](dest []E, src []E) []E {
	dest = SetLength(dest, len(src))
	copy(dest, src)
	return dest
}

// Move moves the element in the given slice at the given
// old position to the given new position and returns the
// resulting slice.
func Move[E any](s []E, from, to int) []E {
	temp := s[from]
	s = slices.Delete(s, from, from+1)
	s = slices.Insert(s, to, temp)
	return s
}

// Swap swaps the elements at the given two indices in the given slice.
func Swap[E any](s []E, i, j int) {
	s[i], s[j] = s[j], s[i]
}

// As converts a slice of the given type to a slice of the other given type.
// The underlying types of the slice elements must be equivalent.
func As[F, T any](s []F) []T {
	as := make([]T, len(s))
	for i, v := range s {
		as[i] = any(v).(T)
	}
	return as
}

// Search returns the index of the item in the given slice that matches the target
// according to the given match function, using the given optional starting index
// to optimize the search by searching bidirectionally outward from given index.
// This is much faster when you have some idea about where the item might be.
// If no start index is given, it starts in the middle, which is a good default.
// It returns -1 if no item matching the match function is found.
func Search[E any](slice []E, match func(e E) bool, startIndex ...int) int {
	n := len(slice)
	if n == 0 {
		return -1
	}
	si := -1
	if len(startIndex) > 0 {
		si = startIndex[0]
	}
	if si < 0 {
		si = n / 2
	}
	if si == 0 {
		for idx, e := range slice {
			if match(e) {
				return idx
			}
		}
	} else {
		if si >= n {
			si = n - 1
		}
		ui := si + 1
		di := si
		upo := false
		for {
			if !upo && ui < n {
				if match(slice[ui]) {
					return ui
				}
				ui++
			} else {
				upo = true
			}
			if di >= 0 {
				if match(slice[di]) {
					return di
				}
				di--
			} else if upo {
				break
			}
		}
	}
	return -1
}

// ToBytes returns the underlying bytes of given slice.
// for items not in a slice, make one of length 1.
// This is copied from webgpu.
func ToBytes[E any](src []E) []byte {
	l := uintptr(len(src))
	if l == 0 {
		return nil
	}
	elmSize := unsafe.Sizeof(src[0])
	return unsafe.Slice((*byte)(unsafe.Pointer(&src[0])), l*elmSize)
}
