// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package runes provides a small subset of functions for rune slices that are found in the
strings and bytes standard packages. For rendering and other logic, it is best to
keep raw data in runes, and not having to convert back and forth to bytes or strings
is more efficient.

These are largely copied from the strings or bytes packages.
*/
package runes

import (
	"unicode"
	"unicode/utf8"

	"cogentcore.org/core/base/slicesx"
)

// SetFromBytes sets slice of runes from given slice of bytes,
// using efficient memory reallocation of existing slice.
// returns potentially modified slice: use assign to update.
func SetFromBytes(rs []rune, s []byte) []rune {
	n := utf8.RuneCount(s)
	rs = slicesx.SetLength(rs, n)
	i := 0
	for len(s) > 0 {
		r, l := utf8.DecodeRune(s)
		rs[i] = r
		i++
		s = s[l:]
	}
	return rs
}

const maxInt = int(^uint(0) >> 1)

// Equal reports whether a and b
// are the same length and contain the same bytes.
// A nil argument is equivalent to an empty slice.
func Equal(a, b []rune) bool {
	// Neither cmd/compile nor gccgo allocates for these string conversions.
	return string(a) == string(b)
}

// // Compare returns an integer comparing two byte slices lexicographically.
// // The result will be 0 if a == b, -1 if a < b, and +1 if a > b.
// // A nil argument is equivalent to an empty slice.
// func Compare(a, b []rune) int {
// 	return bytealg.Compare(a, b)
// }

// Count counts the number of non-overlapping instances of sep in s.
// If sep is an empty slice, Count returns 1 + the number of UTF-8-encoded code points in s.
func Count(s, sep []rune) int {
	n := 0
	for {
		i := Index(s, sep)
		if i == -1 {
			return n
		}
		n++
		s = s[i+len(sep):]
	}
}

// Contains reports whether subslice is within b.
func Contains(b, subslice []rune) bool {
	return Index(b, subslice) != -1
}

// ContainsRune reports whether the rune is contained in the UTF-8-encoded byte slice b.
func ContainsRune(b []rune, r rune) bool {
	return Index(b, []rune{r}) >= 0
	// return IndexRune(b, r) >= 0
}

// ContainsFunc reports whether any of the UTF-8-encoded code points r within b satisfy f(r).
func ContainsFunc(b []rune, f func(rune) bool) bool {
	return IndexFunc(b, f) >= 0
}

// containsRune is a simplified version of strings.ContainsRune
// to avoid importing the strings package.
// We avoid bytes.ContainsRune to avoid allocating a temporary copy of s.
func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

// Trim returns a subslice of s by slicing off all leading and
// trailing UTF-8-encoded code points contained in cutset.
func Trim(s []rune, cutset string) []rune {
	if len(s) == 0 {
		// This is what we've historically done.
		return nil
	}
	if cutset == "" {
		return s
	}
	return TrimLeft(TrimRight(s, cutset), cutset)
}

// TrimLeft returns a subslice of s by slicing off all leading
// UTF-8-encoded code points contained in cutset.
func TrimLeft(s []rune, cutset string) []rune {
	if len(s) == 0 {
		// This is what we've historically done.
		return nil
	}
	if cutset == "" {
		return s
	}
	for len(s) > 0 {
		r := s[0]
		if !containsRune(cutset, r) {
			break
		}
		s = s[1:]
	}
	if len(s) == 0 {
		// This is what we've historically done.
		return nil
	}
	return s
}

// TrimRight returns a subslice of s by slicing off all trailing
// UTF-8-encoded code points that are contained in cutset.
func TrimRight(s []rune, cutset string) []rune {
	if len(s) == 0 || cutset == "" {
		return s
	}
	for len(s) > 0 {
		r := s[len(s)-1]
		if !containsRune(cutset, r) {
			break
		}
		s = s[:len(s)-1]
	}
	return s
}

// TrimSpace returns a subslice of s by slicing off all leading and
// trailing white space, as defined by Unicode.
func TrimSpace(s []rune) []rune {
	return TrimFunc(s, unicode.IsSpace)
}

// TrimLeftFunc treats s as UTF-8-encoded bytes and returns a subslice of s by slicing off
// all leading UTF-8-encoded code points c that satisfy f(c).
func TrimLeftFunc(s []rune, f func(r rune) bool) []rune {
	i := indexFunc(s, f, false)
	if i == -1 {
		return nil
	}
	return s[i:]
}

// TrimRightFunc returns a subslice of s by slicing off all trailing
// UTF-8-encoded code points c that satisfy f(c).
func TrimRightFunc(s []rune, f func(r rune) bool) []rune {
	i := lastIndexFunc(s, f, false)
	return s[0 : i+1]
}

// TrimFunc returns a subslice of s by slicing off all leading and trailing
// UTF-8-encoded code points c that satisfy f(c).
func TrimFunc(s []rune, f func(r rune) bool) []rune {
	return TrimRightFunc(TrimLeftFunc(s, f), f)
}

// TrimPrefix returns s without the provided leading prefix string.
// If s doesn't start with prefix, s is returned unchanged.
func TrimPrefix(s, prefix []rune) []rune {
	if HasPrefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

// TrimSuffix returns s without the provided trailing suffix string.
// If s doesn't end with suffix, s is returned unchanged.
func TrimSuffix(s, suffix []rune) []rune {
	if HasSuffix(s, suffix) {
		return s[:len(s)-len(suffix)]
	}
	return s
}

// Replace returns a copy of the slice s with the first n
// non-overlapping instances of old replaced by new.
// The old string cannot be empty.
// If n < 0, there is no limit on the number of replacements.
func Replace(s, old, new []rune, n int) []rune {
	if len(old) == 0 {
		panic("runes Replace: old cannot be empty")
	}
	m := 0
	if n != 0 {
		// Compute number of replacements.
		m = Count(s, old)
	}
	if m == 0 {
		// Just return a copy.
		return append([]rune(nil), s...)
	}
	if n < 0 || m < n {
		n = m
	}

	// Apply replacements to buffer.
	t := make([]rune, len(s)+n*(len(new)-len(old)))
	w := 0
	start := 0
	for i := 0; i < n; i++ {
		j := start
		if len(old) == 0 {
			if i > 0 {
				j++
			}
		} else {
			j += Index(s[start:], old)
		}
		w += copy(t[w:], s[start:j])
		w += copy(t[w:], new)
		start = j + len(old)
	}
	w += copy(t[w:], s[start:])
	return t[0:w]
}

// ReplaceAll returns a copy of the slice s with all
// non-overlapping instances of old replaced by new.
// If old is empty, it matches at the beginning of the slice
// and after each UTF-8 sequence, yielding up to k+1 replacements
// for a k-rune slice.
func ReplaceAll(s, old, new []rune) []rune {
	return Replace(s, old, new, -1)
}

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

// Repeat returns a new rune slice consisting of count copies of b.
//
// It panics if count is negative or if
// the result of (len(b) * count) overflows.
func Repeat(r []rune, count int) []rune {
	if count == 0 {
		return []rune{}
	}
	// Since we cannot return an error on overflow,
	// we should panic if the repeat will generate
	// an overflow.
	// See Issue golang.org/issue/16237.
	if count < 0 {
		panic("runes: negative Repeat count")
	} else if len(r)*count/count != len(r) {
		panic("runes: Repeat count causes overflow")
	}

	nb := make([]rune, len(r)*count)
	bp := copy(nb, r)
	for bp < len(nb) {
		copy(nb[bp:], nb[:bp])
		bp *= 2
	}
	return nb
}

// Generic split: splits after each instance of sep,
// including sepSave bytes of sep in the subslices.
func genSplit(s, sep []rune, sepSave, n int) [][]rune {
	if n == 0 {
		return nil
	}
	if len(sep) == 0 {
		panic("rune split: separator cannot be empty!")
	}
	if n < 0 {
		n = Count(s, sep) + 1
	}
	if n > len(s)+1 {
		n = len(s) + 1
	}

	a := make([][]rune, n)
	n--
	i := 0
	for i < n {
		m := Index(s, sep)
		if m < 0 {
			break
		}
		a[i] = s[: m+sepSave : m+sepSave]
		s = s[m+len(sep):]
		i++
	}
	a[i] = s
	return a[:i+1]
}

// SplitN slices s into subslices separated by sep and returns a slice of
// the subslices between those separators.
// Sep cannot be empty.
// The count determines the number of subslices to return:
//
//	n > 0: at most n subslices; the last subslice will be the unsplit remainder.
//	n == 0: the result is nil (zero subslices)
//	n < 0: all subslices
//
// To split around the first instance of a separator, see Cut.
func SplitN(s, sep []rune, n int) [][]rune { return genSplit(s, sep, 0, n) }

// SplitAfterN slices s into subslices after each instance of sep and
// returns a slice of those subslices.
// If sep is empty, SplitAfterN splits after each UTF-8 sequence.
// The count determines the number of subslices to return:
//
//	n > 0: at most n subslices; the last subslice will be the unsplit remainder.
//	n == 0: the result is nil (zero subslices)
//	n < 0: all subslices
func SplitAfterN(s, sep []rune, n int) [][]rune {
	return genSplit(s, sep, len(sep), n)
}

// Split slices s into all subslices separated by sep and returns a slice of
// the subslices between those separators.
// If sep is empty, Split splits after each UTF-8 sequence.
// It is equivalent to SplitN with a count of -1.
//
// To split around the first instance of a separator, see Cut.
func Split(s, sep []rune) [][]rune { return genSplit(s, sep, 0, -1) }

// SplitAfter slices s into all subslices after each instance of sep and
// returns a slice of those subslices.
// If sep is empty, SplitAfter splits after each UTF-8 sequence.
// It is equivalent to SplitAfterN with a count of -1.
func SplitAfter(s, sep []rune) [][]rune {
	return genSplit(s, sep, len(sep), -1)
}

var asciiSpace = [256]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1}

// Fields interprets s as a sequence of UTF-8-encoded code points.
// It splits the slice s around each instance of one or more consecutive white space
// characters, as defined by unicode.IsSpace, returning a slice of subslices of s or an
// empty slice if s contains only white space.
func Fields(s []rune) [][]rune {
	return FieldsFunc(s, unicode.IsSpace)
}

// FieldsFunc interprets s as a sequence of UTF-8-encoded code points.
// It splits the slice s at each run of code points c satisfying f(c) and
// returns a slice of subslices of s. If all code points in s satisfy f(c), or
// len(s) == 0, an empty slice is returned.
//
// FieldsFunc makes no guarantees about the order in which it calls f(c)
// and assumes that f always returns the same value for a given c.
func FieldsFunc(s []rune, f func(rune) bool) [][]rune {
	// A span is used to record a slice of s of the form s[start:end].
	// The start index is inclusive and the end index is exclusive.
	type span struct {
		start int
		end   int
	}
	spans := make([]span, 0, 32)

	// Find the field start and end indices.
	// Doing this in a separate pass (rather than slicing the string s
	// and collecting the result substrings right away) is significantly
	// more efficient, possibly due to cache effects.
	start := -1 // valid span start if >= 0
	for end, rune := range s {
		if f(rune) {
			if start >= 0 {
				spans = append(spans, span{start, end})
				// Set start to a negative value.
				// Note: using -1 here consistently and reproducibly
				// slows down this code by a several percent on amd64.
				start = ^start
			}
		} else {
			if start < 0 {
				start = end
			}
		}
	}

	// Last field might end at EOF.
	if start >= 0 {
		spans = append(spans, span{start, len(s)})
	}

	// Create strings from recorded field indices.
	a := make([][]rune, len(spans))
	for i, span := range spans {
		a[i] = s[span.start:span.end:span.end] // last end makes it copy
	}

	return a
}

// Join concatenates the elements of s to create a new byte slice. The separator
// sep is placed between elements in the resulting slice.
func Join(s [][]rune, sep []rune) []rune {
	if len(s) == 0 {
		return []rune{}
	}
	if len(s) == 1 {
		// Just return a copy.
		return append([]rune(nil), s[0]...)
	}

	var n int
	if len(sep) > 0 {
		if len(sep) >= maxInt/(len(s)-1) {
			panic("bytes: Join output length overflow")
		}
		n += len(sep) * (len(s) - 1)
	}
	for _, v := range s {
		if len(v) > maxInt-n {
			panic("bytes: Join output length overflow")
		}
		n += len(v)
	}

	b := make([]rune, n)
	bp := copy(b, s[0])
	for _, v := range s[1:] {
		bp += copy(b[bp:], sep)
		bp += copy(b[bp:], v)
	}
	return b
}

// HasPrefix reports whether the byte slice s begins with prefix.
func HasPrefix(s, prefix []rune) bool {
	return len(s) >= len(prefix) && Equal(s[0:len(prefix)], prefix)
}

// HasSuffix reports whether the byte slice s ends with suffix.
func HasSuffix(s, suffix []rune) bool {
	return len(s) >= len(suffix) && Equal(s[len(s)-len(suffix):], suffix)
}

// IndexFunc interprets s as a sequence of UTF-8-encoded code points.
// It returns the byte index in s of the first Unicode
// code point satisfying f(c), or -1 if none do.
func IndexFunc(s []rune, f func(r rune) bool) int {
	return indexFunc(s, f, true)
}

// LastIndexFunc interprets s as a sequence of UTF-8-encoded code points.
// It returns the byte index in s of the last Unicode
// code point satisfying f(c), or -1 if none do.
func LastIndexFunc(s []rune, f func(r rune) bool) int {
	return lastIndexFunc(s, f, true)
}

// indexFunc is the same as IndexFunc except that if
// truth==false, the sense of the predicate function is
// inverted.
func indexFunc(s []rune, f func(r rune) bool, truth bool) int {
	for i, r := range s {
		if f(r) == truth {
			return i
		}
	}
	return -1
}

// lastIndexFunc is the same as LastIndexFunc except that if
// truth==false, the sense of the predicate function is
// inverted.
func lastIndexFunc(s []rune, f func(r rune) bool, truth bool) int {
	for i := len(s) - 1; i >= 0; i-- {
		if f(s[i]) == truth {
			return i
		}
	}
	return -1
}
