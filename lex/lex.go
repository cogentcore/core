// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"fmt"

	"github.com/goki/pi/token"
)

// Lex represents a single lexical element, with a token, and start and end rune positions
// within a line of a file
type Lex struct {
	Token token.Tokens
	St    int
	Ed    int
}

// String satisfies the fmt.Stringer interface
func (lx Lex) String() string {
	return fmt.Sprintf("[%v:%v:%v]", lx.St, lx.Ed, lx.Token.String())
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
	cp := make(Line, len(*ll))
	for i := range *ll {
		cp[i] = (*ll)[i]
	}
	return cp
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
		str += t.String() + string(s) + " "
	}
	return str
}
