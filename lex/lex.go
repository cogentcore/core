// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

//go:generate goki generate

import (
	"fmt"

	"goki.dev/glop/nptime"
	"goki.dev/pi/v2/token"
)

// Lex represents a single lexical element, with a token, and start and end rune positions
// within a line of a file.  Critically it also contains the nesting depth computed from
// all the parens, brackets, braces.  Todo: also support XML < > </ > tag depth.
type Lex struct {

	// token, includes cache of keyword for keyword types, and also has nesting depth: starting at 0 at start of file and going up for every increment in bracket / paren / start tag and down for every decrement. Is computed once and used extensively in parsing.
	Tok token.KeyToken `desc:"token, includes cache of keyword for keyword types, and also has nesting depth: starting at 0 at start of file and going up for every increment in bracket / paren / start tag and down for every decrement. Is computed once and used extensively in parsing."`

	// start rune index within original source line for this token
	St int `desc:"start rune index within original source line for this token"`

	// end rune index within original source line for this token (exclusive -- ends one before this)
	Ed int `desc:"end rune index within original source line for this token (exclusive -- ends one before this)"`

	// time when region was set -- used for updating locations in the text based on time stamp (using efficient non-pointer time)
	Time nptime.Time `desc:"time when region was set -- used for updating locations in the text based on time stamp (using efficient non-pointer time)"`
}

func NewLex(tok token.KeyToken, st, ed int) Lex {
	lx := Lex{Tok: tok, St: st, Ed: ed}
	return lx
}

// Src returns the rune source for given lex item (does no validity checking)
func (lx *Lex) Src(src []rune) []rune {
	return src[lx.St:lx.Ed]
}

// Now sets the time stamp to now
func (lx *Lex) Now() {
	lx.Time.Now()
}

// String satisfies the fmt.Stringer interface
func (lx *Lex) String() string {
	return fmt.Sprintf("[+%d:%v:%v:%v]", lx.Tok.Depth, lx.St, lx.Ed, lx.Tok.String())
}

// ContainsPos returns true if the Lex element contains given character position
func (lx *Lex) ContainsPos(pos int) bool {
	return pos >= lx.St && pos < lx.Ed
}

// OverlapsReg returns true if the two regions overlap
func (lx *Lex) OverlapsReg(or Lex) bool {
	// start overlaps
	if (lx.St >= or.St && lx.St < or.Ed) || (or.St >= lx.St && or.St < lx.Ed) {
		return true
	}
	// end overlaps
	if (lx.Ed > or.St && lx.Ed <= or.Ed) || (or.Ed > lx.St && or.Ed <= lx.Ed) {
		return true
	}
	return false
}

// Region returns the region for this lexical element, at given line
func (lx *Lex) Region(ln int) Reg {
	return Reg{St: Pos{Ln: ln, Ch: lx.St}, Ed: Pos{Ln: ln, Ch: lx.Ed}}
}
