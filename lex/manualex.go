// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"strings"
	"unicode"

	"github.com/goki/ki/ints"
)

// the ManuaLex functions provide "manual" lexing support for specific cases
// such as completion where a string must be processed further.

// FirstWord returns the first contiguous sequence of purely unicode.IsLetter runes
// within given string -- skips over any leading non-letters until a letter is found.
// This does not include numbers -- use FirstWordDigits for that
func FirstWord(str string) string {
	rstr := ""
	for _, s := range str {
		if !IsLetter(s) {
			if len(rstr) == 0 {
				continue
			}
			break
		}
		rstr += string(s)
	}
	return rstr
}

// FirstWordDigits returns the first contiguous sequence of purely IsLetterOrDigit runes
// within given string -- skips over any leading non-letters until a *letter* (not digit) is found.
func FirstWordDigits(str string) string {
	rstr := ""
	for _, s := range str {
		if !IsLetterOrDigit(s) {
			if len(rstr) == 0 {
				continue
			}
			break
		}
		if len(rstr) == 0 && IsDigit(s) { // can't start with digit
			continue
		}
		rstr += string(s)
	}
	return rstr
}

// FirstWordApostrophe returns the first contiguous sequence of purely unicode.IsLetter runes.
// that can also contain an apostrophe *within* the word but not at the end
func FirstWordApostrophe(str string) string {
	rstr := ""
	for _, s := range str {
		if !(IsLetter(s) || s == '\'') {
			if len(rstr) == 0 {
				continue
			}
			break
		}
		rstr += string(s)
	}
	rstr = strings.TrimRight(rstr, "'") // get rid of any trailing ones!
	return rstr
}

// IsWordBreak defines what counts as a word break for the purposes of selecting words
// r1 is the rune in question, r2 is the rune past r1 in the direction you are moving
// Pass rune(-1) for r2 if there is no rune past r1
func IsWordBreak(r1, r2 rune) bool {
	if r2 == rune(-1) {
		if unicode.IsSpace(r1) || unicode.IsSymbol(r1) || unicode.IsPunct(r1) {
			return true
		}
		return false
	}
	if unicode.IsSpace(r1) || unicode.IsSymbol(r1) {
		return true
	}
	if unicode.IsPunct(r1) && r1 != rune('\'') {
		return true
	}
	if unicode.IsPunct(r1) && r1 == rune('\'') {
		if unicode.IsSpace(r2) || unicode.IsSymbol(r2) || unicode.IsPunct(r2) {
			return true
		}
		return false
	}
	return false
}

// TrimLeftToAlpha returns string without any leading non-alpha runes
func TrimLeftToAlpha(nm string) string {
	return strings.TrimLeftFunc(nm, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
}

// InnerBracketScope returns the inner-scope for given bracket type
// if it is imbalanced -- it is important to do completion based
// just on that inner scope if that is where the user is at.
func InnerBracketScope(str string, brl, brr string) string {
	nlb := strings.Count(str, brl)
	nrb := strings.Count(str, brr)
	if nlb == nrb {
		return str
	}
	if nlb > nrb {
		li := strings.LastIndex(str, brl)
		if li == len(str)-1 {
			return InnerBracketScope(str[:li], brl, brr) // get rid of open ending and try again
		}
		str = str[li+1:]
		ri := strings.Index(str, brr)
		if ri < 0 {
			return str
		}
		return str[:ri]
	}
	// nrb > nlb -- we're missing the left guys -- go to first rb
	ri := strings.Index(str, brr)
	if ri == 0 {
		return InnerBracketScope(str[1:], brl, brr) // get rid of opening and try again
	}
	str = str[:ri]
	li := strings.Index(str, brl)
	if li < 0 {
		return str
	}
	return str[li+1:]
}

// LastField returns the last white-space separated string
func LastField(str string) string {
	if str == "" {
		return ""
	}
	flds := strings.Fields(str)
	return flds[len(flds)-1]
}

// LastScopedString returns the last white-space separated, and bracket
// enclosed string from given string.
func LastScopedString(str string) string {
	str = LastField(str)
	bstr := str
	str = InnerBracketScope(str, "{", "}")
	str = InnerBracketScope(str, "(", ")")
	str = InnerBracketScope(str, "[", "]")
	if str == "" {
		return bstr
	}
	str = TrimLeftToAlpha(str)
	if str == "" {
		str = TrimLeftToAlpha(bstr)
		if str == "" {
			return bstr
		}
	}
	flds := strings.Split(str, ",")
	return flds[len(flds)-1]
}

// MatchCase uses the source string case (upper / lower) to set corresponding
// case in target string, returning that string.
func MatchCase(src, trg string) string {
	rsc := []rune(src)
	rtg := []rune(trg)
	mx := ints.MinInt(len(rsc), len(rtg))
	for i := 0; i < mx; i++ {
		t := rtg[i]
		if unicode.IsUpper(rsc[i]) {
			if !unicode.IsUpper(t) {
				rtg[i] = unicode.ToUpper(t)
			}
		} else {
			if !unicode.IsLower(t) {
				rtg[i] = unicode.ToLower(t)
			}
		}
	}
	return string(rtg)
}
