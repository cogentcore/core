// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"github.com/goki/ki/ints"
	"github.com/goki/pi/token"
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
func BraceMatch(src [][]rune, tags []Line, r rune, st Pos, maxLns int) (en Pos, found bool) {
	en.Ln = -1
	found = false
	match, rt := BracePair(r)
	var left int
	var right int
	if rt {
		right++
	} else {
		left++
	}
	ch := st.Ch
	ln := st.Ln
	nln := len(src)
	max := ints.MinInt(nln-ln, maxLns)
	min := ints.MinInt(ln, maxLns)
	txt := src[ln]
	tln := tags[ln]
	if left > right {
		for l := ln + 1; l < ln+max; l++ {
			for i := ch + 1; i < len(txt); i++ {
				if txt[i] == r {
					lx := tln.AtPos(i)
					if lx == nil || lx.Tok.Tok.Cat() != token.Comment {
						left++
						continue
					}
				}
				if txt[i] == match {
					lx := tln.AtPos(i)
					if lx == nil || lx.Tok.Tok.Cat() != token.Comment {
						right++
						if left == right {
							en.Ln = l - 1
							en.Ch = i
							break
						}
					}
				}
			}
			if en.Ln >= 0 {
				found = true
				break
			}
			txt = src[l]
			tln = tags[l]
			ch = -1
		}
	} else {
		for l := ln - 1; l >= ln-min; l-- {
			ch = ints.MinInt(ch, len(txt))
			for i := ch - 1; i >= 0; i-- {
				if txt[i] == r {
					lx := tln.AtPos(i)
					if lx == nil || lx.Tok.Tok.Cat() != token.Comment {
						right++
						continue
					}
				}
				if txt[i] == match {
					lx := tln.AtPos(i)
					if lx == nil || lx.Tok.Tok.Cat() != token.Comment {
						left++
						if left == right {
							en.Ln = l + 1
							en.Ch = i
							break
						}
					}
				}
			}
			if en.Ln >= 0 {
				found = true
				break
			}
			txt = src[l]
			tln = tags[l]
			ch = len(txt)
		}
	}
	return en, found
}
