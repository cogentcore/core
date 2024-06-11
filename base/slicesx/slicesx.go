// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package slicesx provides additional slice functions
// beyond those in the standard [slices] package.
package slicesx

import "slices"

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
