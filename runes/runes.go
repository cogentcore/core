// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package runes provides a small subset of functions that are found in strings, bytes
standard packages, for rune slices.  For rendering, and other logic, it is best to
keep raw data in runes, and not having to convert back and forth to byte or string
is more efficient.

These are largely copied from strings or bytes packages.
*/
package runes

import (
	"unicode"
	"unicode/utf8"
)

// EqualFold reports whether s and t are equal under Unicode case-folding.
// copied from strings.EqualFold
func EqualFold(s, t []rune) bool {
	for len(s) > 0 && len(t) > 0 {
		// Extract first rune from each string.
		var sr, tr rune
		sr, s = s[0], s[1:]
		tr, t = t[0], t[1:]
		// If they match, keep going; if not, return false.

		// Easy case.
		if tr == sr {
			continue
		}

		// Make sr < tr to simplify what follows.
		if tr < sr {
			tr, sr = sr, tr
		}
		// Fast check for ASCII.
		if tr < utf8.RuneSelf {
			// ASCII only, sr/tr must be upper/lower case
			if 'A' <= sr && sr <= 'Z' && tr == sr+'a'-'A' {
				continue
			}
			return false
		}

		// General case. SimpleFold(x) returns the next equivalent rune > x
		// or wraps around to smaller values.
		r := unicode.SimpleFold(sr)
		for r != sr && r < tr {
			r = unicode.SimpleFold(r)
		}
		if r == tr {
			continue
		}
		return false
	}

	// One string is empty. Are both?
	return len(s) == len(t)
}

// Index returns the index of given rune string in the text, returning -1 if not found.
func Index(txt, find []rune) int {
	fsz := len(find)
	if fsz == 0 {
		return -1
	}
	tsz := len(txt)
	if tsz < fsz {
		return -1
	}
	mn := tsz - fsz
	for i := 0; i <= mn; i++ {
		found := true
		for j := range find {
			if txt[i+j] != find[j] {
				found = false
				break
			}
		}
		if found {
			return i
		}
	}
	return -1
}

// IndexFold returns the index of given rune string in the text, using case folding
// (i.e., case insensitive matching).  Returns -1 if not found.
func IndexFold(txt, find []rune) int {
	fsz := len(find)
	if fsz == 0 {
		return -1
	}
	tsz := len(txt)
	if tsz < fsz {
		return -1
	}
	mn := tsz - fsz
	for i := 0; i <= mn; i++ {
		if EqualFold(txt[i:i+fsz], find) {
			return i
		}
	}
	return -1
}
