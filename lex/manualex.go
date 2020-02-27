// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"strings"
	"unicode"
)

// the ManuaLex functions provide "manual" lexing support for specific cases
// such as completion where a string must be processed further.

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
