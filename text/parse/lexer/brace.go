// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer

import (
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
)

// BracePair returns the matching brace-like punctuation for given rune,
// which must be a left or right brace {}, bracket [] or paren ().
// Also returns true if it is *right*
func BracePair(r rune) (match rune, right bool) {
	right = false
	switch r {
	case '{':
		match = '}'
	case '}':
		right = true
		match = '{'
	case '(':
		match = ')'
	case ')':
		right = true
		match = '('
	case '[':
		match = ']'
	case ']':
		right = true
		match = '['
	}
	return
}

// BraceMatch finds the brace, bracket, or paren that is the partner
// of the one passed to function, within maxLns lines of start.
// Operates on rune source with markup lex tags per line (tags exclude comments).
func BraceMatch(src [][]rune, tags []Line, r rune, st textpos.Pos, maxLns int) (en textpos.Pos, found bool) {
	en.Line = -1
	found = false
	match, rt := BracePair(r)
	var left int
	var right int
	if rt {
		right++
	} else {
		left++
	}
	ch := st.Char
	ln := st.Line
	nln := len(src)
	mx := min(nln-ln, maxLns)
	mn := min(ln, maxLns)
	// isCode returns whether the rune at i in line l is not in a comment.
	isCode := func(l, i int) bool {
		lx, _ := tags[l].AtPos(i)
		return lx == nil || lx.Token.Token.Cat() != token.Comment
	}
	if left > right {
		// lines ln to ln+mx-1, after the brace on its line
		for l := ln; l < ln+mx && !found; l++ {
			txt := src[l]
			from := 0
			if l == ln {
				from = ch + 1
			}
			for i := from; i < len(txt); i++ {
				switch txt[i] {
				case r:
					if isCode(l, i) {
						left++
					}
				case match:
					if isCode(l, i) {
						right++
						if left == right {
							en = textpos.Pos{Line: l, Char: i}
							found = true
						}
					}
				}
				if found {
					break
				}
			}
		}
	} else {
		// lines ln down to ln-mn, before the brace on its line
		for l := ln; l >= ln-mn && !found; l-- {
			txt := src[l]
			from := len(txt) - 1
			if l == ln {
				from = min(ch, len(txt)) - 1
			}
			for i := from; i >= 0; i-- {
				switch txt[i] {
				case r:
					if isCode(l, i) {
						right++
					}
				case match:
					if isCode(l, i) {
						left++
						if left == right {
							en = textpos.Pos{Line: l, Char: i}
							found = true
						}
					}
				}
				if found {
					break
				}
			}
		}
	}
	return en, found
}
