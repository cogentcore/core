// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package lexer provides all the lexing functions that transform text
// into lexical tokens, using token types defined in the token package.
// It also has the basic file source and position / region management
// functionality.
package lexer

//go:generate core generate

import (
	"fmt"

	"cogentcore.org/core/base/nptime"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
)

// Lex represents a single lexical element, with a token, and start and end rune positions
// within a line of a file.  Critically it also contains the nesting depth computed from
// all the parens, brackets, braces.  Todo: also support XML < > </ > tag depth.
type Lex struct {

	// Token includes cache of keyword for keyword types, and also has nesting depth: starting at 0 at start of file and going up for every increment in bracket / paren / start tag and down for every decrement. Is computed once and used extensively in parsing.
	Token token.KeyToken

	// start rune index within original source line for this token
	Start int

	// end rune index within original source line for this token (exclusive -- ends one before this)
	End int

	// time when region was set -- used for updating locations in the text based on time stamp (using efficient non-pointer time)
	Time nptime.Time
}

func NewLex(tok token.KeyToken, st, ed int) Lex {
	lx := Lex{Token: tok, Start: st, End: ed}
	return lx
}

// Src returns the rune source for given lex item (does no validity checking)
func (lx *Lex) Src(src []rune) []rune {
	return src[lx.Start:lx.End]
}

// Now sets the time stamp to now
func (lx *Lex) Now() {
	lx.Time.Now()
}

// String satisfies the fmt.Stringer interface
func (lx *Lex) String() string {
	return fmt.Sprintf("[+%d:%v:%v:%v]", lx.Token.Depth, lx.Start, lx.End, lx.Token.String())
}

// ContainsPos returns true if the Lex element contains given character position
func (lx *Lex) ContainsPos(pos int) bool {
	return pos >= lx.Start && pos < lx.End
}

// OverlapsReg returns true if the two regions overlap
func (lx *Lex) OverlapsReg(or Lex) bool {
	// start overlaps
	if (lx.Start >= or.Start && lx.Start < or.End) || (or.Start >= lx.Start && or.Start < lx.End) {
		return true
	}
	// end overlaps
	return (lx.End > or.Start && lx.End <= or.End) || (or.End > lx.Start && or.End <= lx.End)
}

// Region returns the region for this lexical element, at given line
func (lx *Lex) Region(ln int) textpos.Region {
	return textpos.Region{Start: textpos.Pos{Line: ln, Char: lx.Start}, End: textpos.Pos{Line: ln, Char: lx.End}}
}
