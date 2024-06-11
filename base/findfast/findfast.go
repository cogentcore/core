// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package findfast implements an optimized bidirectional slice searching
// algorithm that can save a lot of time if you have some rough idea
// as to where an item might be.
package findfast

// FindFunc returns index of item in slice that matches target
// according to given match function, using the given optional
// starting index to optimize the search by searching bidirectionally
// outward from given index.  This is much faster when you have
// some idea about where it might be.
// Returns -1 if not found.
func FindFunc[T any](s []T, match func(e T) bool, startIndex ...int) int {
	n := len(s)
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
		for idx, e := range s {
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
				if match(s[ui]) {
					return ui
				}
				ui++
			} else {
				upo = true
			}
			if di >= 0 {
				if match(s[di]) {
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
