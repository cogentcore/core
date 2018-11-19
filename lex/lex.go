// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"unicode"
	"unicode/utf8"

	"github.com/goki/pi/token"
)

// Lex represents a single lexical element, with a token, and start and end rune positions
// within a line of a file
type Lex struct {
	Token token.Tokens
	St    int
	Ed    int
}

// ContainsPos returns true if the Lex element contains given character position
func (lx *Lex) ContainsPos(pos int) bool {
	return pos >= lx.St && pos < lx.Ed
}

// Line is one line of Lex'd text
type Line []Lex

// Add adds one element to the lex line (just append)
func (ll *Line) Add(lx Lex) {
	*ll = append(*ll, lx)
}

// AddSort adds a new lex element in sorted order to list
func (ll *Line) AddSort(lx Lex) {
	for i, t := range *ll {
		if t.St < lx.St {
			continue
		}
		*ll = append(*ll, lx)
		copy((*ll)[i+1:], (*ll)[i:])
		(*ll)[i] = lx
		return
	}
	*ll = append(*ll, lx)
}

////////////////////////////////////////////////////////////////////////////////
//  State

// State is the state maintained for lexing
type State struct {
	Src   []rune   `desc:"the current line of source being processed"`
	Lex   Line     `desc:"the lex output for this line"`
	Pos   int      `desc:"the current position within the line"`
	State []string `desc:"state stack"`
}

// String gets the string at given offset and length from current position, returns false if out of range
func (ls *State) String(off, sz int) (string, bool) {
	idx := ls.Pos + off
	ei := idx + sz
	if ei > len(ls.Src) {
		return "", false
	}
	return string(ls.Src[idx:ei]), true
}

// Rune gets the rune at given offset from current position, returns false if out of range
func (ls *State) Rune(off int) (rune, bool) {
	idx := ls.Pos + off
	if idx >= len(ls.Src) {
		return 0, false
	}
	return ls.Src[idx], true
}

// Next moves to next position using given increment in source line -- returns false if at end
func (ls *State) Next(inc int) bool {
	sz := len(ls.Src)
	ls.Pos += inc
	if ls.Pos >= sz {
		ls.Pos = sz
		return false
	}
	return true
}

// Add adds a lex token for given region
func (ls *State) Add(tok token.Tokens, st, ed int) {
	ls.Lex.Add(Lex{tok, st, ed})
}

func (ls *State) PushState(st string) {
	ls.State = append(ls.State, st)
}

func (ls *State) CurState() string {
	sz := len(ls.State)
	if sz == 0 {
		return ""
	}
	return ls.State[sz-1]
}

func (ls *State) PopState() string {
	sz := len(ls.State)
	if sz == 0 {
		return ""
	}
	st := ls.CurState()
	ls.State = ls.State[:sz-1]
	return st
}

//////////////////////////////////////////////////////////////////////////////
// Lex utils

func IsLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}

func IsDigit(ch rune) bool {
	return '0' <= ch && ch <= '9' || ch >= utf8.RuneSelf && unicode.IsDigit(ch)
}

func IsWhiteSpace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}
