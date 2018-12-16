// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package lex provides all the lexing functions that transform text into
// lexical tokens, using token types defined in the pi/token package.
// It also has the basic file source and position / region management
// functionality.
package lex

import (
	"fmt"
	"sort"

	"github.com/goki/ki/nptime"
	"github.com/goki/pi/token"
)

// Lex represents a single lexical element, with a token, and start and end rune positions
// within a line of a file.  Critically it also contains the nesting depth computed from
// all the parens, brackets, braces.  Todo: also support XML < > </ > tag depth.
type Lex struct {
	Tok   token.Tokens `desc:"token"`
	Depth int          `desc:"nesting depth, starting at 0 at start of file and going up for every increment in bracket / paren / start tag and down for every decrement.  Coloring background according to this depth should give direct information about mismatches etc.  Is computed once and used extensively in parsing."`
	St    int          `desc:"start rune index within original source line for this token"`
	Ed    int          `desc:"end rune index within original source line for this token (exclusive -- ends one before this)"`
	Time  nptime.Time  `desc:"time when region was set -- used for updating locations in the text based on time stamp (using efficient non-pointer time)"`
}

func NewLex(tok token.Tokens, st, ed int) Lex {
	lx := Lex{Tok: tok, St: st, Ed: ed}
	return lx
}

// String satisfies the fmt.Stringer interface
func (lx Lex) String() string {
	return fmt.Sprintf("[+%d:%v:%v:%v]", lx.Depth, lx.St, lx.Ed, lx.Tok.String())
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

// Line is one line of Lex'd text
type Line []Lex

// Add adds one element to the lex line (just append)
func (ll *Line) Add(lx Lex) {
	*ll = append(*ll, lx)
}

// Add adds one element to the lex line with given params, returns pointer to that new lex
func (ll *Line) AddLex(tok token.Tokens, st, ed int) *Lex {
	lx := NewLex(tok, st, ed)
	li := len(*ll)
	ll.Add(lx)
	return &(*ll)[li]
}

// Insert inserts one element to the lex line at given point
func (ll *Line) Insert(idx int, lx Lex) {
	sz := len(*ll)
	*ll = append(*ll, lx)
	if idx < sz {
		copy((*ll)[idx+1:], (*ll)[idx:sz])
		(*ll)[idx] = lx
	}
}

// Clone returns a new copy of the line
func (ll *Line) Clone() Line {
	if len(*ll) == 0 {
		return nil
	}
	cp := make(Line, len(*ll))
	for i := range *ll {
		cp[i] = (*ll)[i]
	}
	return cp
}

// AddSort adds a new lex element in sorted order to list, sorted by start
// position, and if at the same start position, then sorted by end position
func (ll *Line) AddSort(lx Lex) {
	for i, t := range *ll {
		if t.St < lx.St {
			continue
		}
		if t.St == lx.St && lx.Ed >= t.Ed {
			continue
		}
		*ll = append(*ll, lx)
		copy((*ll)[i+1:], (*ll)[i:])
		(*ll)[i] = lx
		return
	}
	*ll = append(*ll, lx)
}

// Sort sorts the lex elements by starting pos, and ending pos if a tie
func (ll *Line) Sort() {
	sort.Slice((*ll), func(i, j int) bool {
		return (*ll)[i].St < (*ll)[j].St || ((*ll)[i].St == (*ll)[j].St && (*ll)[i].Ed < (*ll)[j].Ed)
	})
}

// MergeLines merges the two lines of lex regions into a combined list
// properly ordered by sequence of tags within the line.
func MergeLines(t1, t2 Line) Line {
	sz1 := len(t1)
	sz2 := len(t2)
	if sz1 == 0 {
		return t2
	}
	if sz2 == 0 {
		return t1
	}
	tsz := sz1 + sz2
	tl := make(Line, 0, tsz)
	for i := 0; i < sz1; i++ {
		tl = append(tl, t1[i])
	}
	for i := 0; i < sz2; i++ {
		tl.AddSort(t2[i])
	}
	return tl
}

// String satisfies the fmt.Stringer interface
func (ll *Line) String() string {
	str := ""
	for _, t := range *ll {
		str += t.String() + " "
	}
	return str
}

// TagSrc returns the token-tagged source
func (ll *Line) TagSrc(src []rune) string {
	str := ""
	for _, t := range *ll {
		s := src[t.St:t.Ed]
		str += t.String() + `"` + string(s) + `"` + " "
	}
	return str
}
