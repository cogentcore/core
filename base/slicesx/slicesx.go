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
