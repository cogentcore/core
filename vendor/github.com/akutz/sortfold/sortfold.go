/*
Package sortfold enables sorting string slices in increasing order using
a case-insensitive comparison.
*/
package sortfold

import (
	"sort"

	"unicode"
	"unicode/utf8"
)

// StringSlice attaches the methods of Interface to []string,
// sorting in increasing order using a case insensitive comparison.
type StringSlice []string

func (p StringSlice) Len() int { return len(p) }
func (p StringSlice) Less(i, j int) bool {
	return CompareFold(p[i], p[j]) < 0
}
func (p StringSlice) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

// Sort is a convenience method.
func (p StringSlice) Sort() { sort.Sort(p) }

// Strings sorts a slice of strings in increasing order using a case
// insensitive comparison.
func Strings(a []string) { sort.Sort(StringSlice(a)) }

// StringsAreSorted ¶ tests whether a slice of strings is sorted in increasing
// order using a case insensitive comparison.
func StringsAreSorted(a []string) bool {
	return sort.IsSorted(StringSlice(a))
}

// CompareFold reports whether s and t, interpreted as UTF-8 strings,
// are equal under Unicode case-folding. A return value of 0 means s==t,
// <0 means s<t, and >0 means s>t.
//
// This function is a modification of the golang strings.EqualFold function
// from strings/strings.go at https://goo.gl/yiMur6.
func CompareFold(s, t string) int {
	for s != "" && t != "" {

		// Extract first rune from each string.
		var sr, tr rune
		if s[0] < utf8.RuneSelf {
			sr, s = rune(s[0]), s[1:]
		} else {
			r, size := utf8.DecodeRuneInString(s)
			sr, s = r, s[size:]
		}
		if t[0] < utf8.RuneSelf {
			tr, t = rune(t[0]), t[1:]
		} else {
			r, size := utf8.DecodeRuneInString(t)
			tr, t = r, t[size:]
		}

		// Easy case.
		if sr == tr {
			continue
		}

		// Make sr < tr to simplify what follows.
		result := 1
		if tr < sr {
			tr, sr = sr, tr
			result = -result
		}

		// Fast check for ASCII. Since sr < tr, if tr is ASCII then sr
		// is ASCII.
		if tr < utf8.RuneSelf {
			// If tr <= 'Z' then so is sr, which means there is no mixed-case
			// and that sr < tr.
			if tr <= 'Z' {
				return -result
			}

			// At this point the following is known:
			//   1. sr < tr
			//   2. tr > 'Z', meaning tr is lower-case
			// The following is unknown:
			//   1. The case of sr
			// The case of sr is required to determine the order of sr and tr.

			// If sr <= 'Z' then it must be folded to determine the order of
			// sr and tr.
			if sr <= 'Z' {
				sr2 := sr + 'a' - 'A'

				// When folded as sr2, sr == tr, so continue to compare the
				// string.
				if sr2 == tr {
					continue
				}

				// If sr2 < tr then sr < tr.
				if sr2 < tr {
					return -result
				}

				// sr > tr
				return result
			}

			// Because sr > 'Z' and tr > 'Z', sr and tr can be
			// compared directly.
			if sr < tr {
				return -result
			}
			return result
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
		if r < tr {
			return -result
		}
		if r > tr {
			return result
		}
	}

	// One string is empty. Are both?
	if s == "" && t != "" {
		return -1
	}
	if s != "" && t == "" {
		return 1
	}
	return 0
}
