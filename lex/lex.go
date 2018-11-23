// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
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
