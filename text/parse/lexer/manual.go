// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer

import (
	"strings"
	"unicode"

	"cogentcore.org/core/text/token"
)

// These functions provide "manual" lexing support for specific cases, such as completion, where a string must be processed further.

// FirstWord returns the first contiguous sequence of purely [unicode.IsLetter] runes within the given string.
// It skips over any leading non-letters until a letter is found.
// Note that this function does not include numbers. For that, you can use the FirstWordDigits function.
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

// FirstWordDigits returns the first contiguous sequence of purely [IsLetterOrDigit]
// runes within the given string. It skips over any leading non-letters until a letter
// (not digit) is found.
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

// FirstWordApostrophe returns the first contiguous sequence of purely
// [unicode.IsLetter] runes that can also contain an apostrophe within
// the word but not at the end. This is for spell checking: also excludes
// any _ values.
func FirstWordApostrophe(str string) string {
	rstr := ""
	for _, s := range str {
		if !(IsLetterNoUnderbar(s) || s == '\'') {
			if len(rstr) == 0 {
				continue
			}
			break
		}
		if len(rstr) == 0 && s == '\'' { // can't start with '
			continue
		}
		rstr += string(s)
	}
	rstr = strings.TrimRight(rstr, "'") // get rid of any trailing ones!
	return rstr
}

// TrimLeftToAlpha returns string without any leading non-alpha runes.
func TrimLeftToAlpha(nm string) string {
	return strings.TrimLeftFunc(nm, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
}

// FirstNonSpaceRune returns the index of first non-space rune, -1 if not found
func FirstNonSpaceRune(src []rune) int {
	for i, s := range src {
		if !unicode.IsSpace(s) {
			return i
		}
	}
	return -1
}

// LastNonSpaceRune returns the index of last non-space rune, -1 if not found
func LastNonSpaceRune(src []rune) int {
	sz := len(src)
	if sz == 0 {
		return -1
	}
	for i := sz - 1; i >= 0; i-- {
		s := src[i]
		if !unicode.IsSpace(s) {
			return i
		}
	}
	return -1
}

// InnerBracketScope returns the inner scope for a given bracket type if it is
// imbalanced. It is important to do completion based on just that inner scope
// if that is where the user is at.
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

// ObjPathAt returns the starting Lex, before given lex,
// that include sequences of PunctSepPeriod and NameTag
// which are used for object paths (e.g., field.field.field)
func ObjPathAt(line Line, lx *Lex) *Lex {
	stlx := lx
	if lx.Start > 1 {
		_, lxidx := line.AtPos(lx.Start - 1)
		for i := lxidx; i >= 0; i-- {
			clx := &line[i]
			if clx.Token.Token == token.PunctSepPeriod || clx.Token.Token.InCat(token.Name) {
				stlx = clx
			} else {
				break
			}
		}
	}
	return stlx
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

// HasUpperCase returns true if string has an upper-case letter
func HasUpperCase(str string) bool {
	for _, r := range str {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// MatchCase uses the source string case (upper / lower) to set corresponding
// case in target string, returning that string.
func MatchCase(src, trg string) string {
	rsc := []rune(src)
	rtg := []rune(trg)
	mx := min(len(rsc), len(rtg))
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
