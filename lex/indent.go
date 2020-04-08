// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"github.com/goki/ki/indent"
	"github.com/goki/pi/token"
)

// these functions support indentation algorithms,
// operating on marked-up rune source.

// LineIndent returns the number of tabs or spaces at start of given rune-line,
// based on target tab-size (only relevant for spaces).
// If line starts with tabs, then those are counted, else spaces --
// combinations of tabs and spaces won't produce sensible results.
func LineIndent(src []rune, tabSz int) (ind int, ichr indent.Char) {
	ichr = indent.Tab
	sz := len(src)
	if sz == 0 {
		return
	}
	if src[0] == ' ' {
		ichr = indent.Space
		ind = 1
	} else if src[0] != '\t' {
		return
	} else {
		ind = 1
	}
	if ichr == indent.Space {
		for i := 1; i < sz; i++ {
			if src[i] == ' ' {
				ind++
			} else {
				ind /= tabSz
				return
			}
		}
		ind /= tabSz
		return
	} else {
		for i := 1; i < sz; i++ {
			if src[i] == '\t' {
				ind++
			} else {
				return
			}
		}
	}
	return
}

// PrevLineIndent returns indentation level of previous line
// from given line that has indentation -- skips blank lines.
// Returns indent level and previous line number, and indent char.
// indent level is in increments of tabSz for spaces, and tabs for tabs.
// Operates on rune source with markup lex tags per line.
func PrevLineIndent(src [][]rune, tags []Line, ln int, tabSz int) (ind, pln int, ichr indent.Char) {
	ln--
	for ln >= 0 {
		if len(src[ln]) == 0 {
			ln--
			continue
		}
		ind, ichr = LineIndent(src[ln], tabSz)
		pln = ln
		return
	}
	ind = 0
	pln = 0
	return
}

// BracketIndentLine returns the indentation level for given line based on
// previous line's indentation level, and any delta change based on
// brackets starting or ending the previous or current line.
// indent level is in increments of tabSz for spaces, and tabs for tabs.
// Operates on rune source with markup lex tags per line.
func BracketIndentLine(src [][]rune, tags []Line, ln int, tabSz int) (pInd, delInd, pLn int, ichr indent.Char) {
	pInd, pLn, ichr = PrevLineIndent(src, tags, ln, tabSz)

	curUnd, _ := LineStartEndBracket(src[ln], tags[ln])
	_, prvInd := LineStartEndBracket(src[pLn], tags[pLn])

	delInd = 0
	switch {
	case prvInd && curUnd:
		delInd = 0 // offset
	case prvInd:
		delInd = 1 // indent
	case curUnd:
		delInd = -1 // undent
	}
	if pInd == 0 && delInd < 0 { // error..
		delInd = 0
	}
	return
}

// LastTokenIgnoreComment returns the last token of the tags, ignoring
// any final comment at end
func LastLexIgnoreComment(tags Line) (*Lex, int) {
	var ll *Lex
	li := -1
	nt := len(tags)
	for i := nt - 1; i >= 0; i-- {
		l := &tags[i]
		if l.Tok.Tok.Cat() == token.Comment || l.Tok.Tok < token.Keyword {
			continue
		}
		ll = l
		li = i
		break
	}
	return ll, li
}

// LineStartEndBracket checks if line starts with a closing bracket
// or ends with an opening bracket. This is used for auto-indent for example.
// Bracket is Paren, Bracket, or Brace.
func LineStartEndBracket(src []rune, tags Line) (start, end bool) {
	if len(src) == 0 {
		return
	}
	nt := len(tags)
	if nt > 0 {
		ftok := tags[0].Tok.Tok
		if ftok.InSubCat(token.PunctGp) {
			if ftok.IsPunctGpRight() {
				start = true
			}
		}
		ll, _ := LastLexIgnoreComment(tags)
		if ll != nil {
			ltok := ll.Tok.Tok
			if ltok.InSubCat(token.PunctGp) {
				if ltok.IsPunctGpLeft() {
					end = true
				}
			}
		}
		return
	}
	// no tags -- do it manually
	fi := FirstNonSpaceRune(src)
	if fi >= 0 {
		bp, rt := BracePair(src[fi])
		if bp != 0 && rt {
			start = true
		}
	}
	li := LastNonSpaceRune(src)
	if li >= 0 {
		bp, rt := BracePair(src[li])
		if bp != 0 && !rt {
			end = true
		}
	}
	return
}
