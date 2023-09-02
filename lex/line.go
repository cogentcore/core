// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"sort"
	"unicode"

	"goki.dev/ki/v2/sliceclone"
	"goki.dev/pi/v2/token"
)

// Line is one line of Lex'd text
type Line []Lex

// Add adds one element to the lex line (just append)
func (ll *Line) Add(lx Lex) {
	*ll = append(*ll, lx)
}

// Add adds one element to the lex line with given params, returns pointer to that new lex
func (ll *Line) AddLex(tok token.KeyToken, st, ed int) *Lex {
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

// AtPos returns the Lex in place for given position, and index, or nil, -1 if none
func (ll *Line) AtPos(pos int) (*Lex, int) {
	for i := range *ll {
		lx := &((*ll)[i])
		if lx.ContainsPos(pos) {
			return lx, i
		}
	}
	return nil, -1
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
// position, and if at the same start position, then sorted *decreasing*
// by end position -- this allows outer tags to be processed before inner tags
// which fits a stack-based tag markup logic.
func (ll *Line) AddSort(lx Lex) {
	for i, t := range *ll {
		if t.St < lx.St {
			continue
		}
		if t.St == lx.St && t.Ed >= lx.Ed {
			continue
		}
		*ll = append(*ll, lx)
		copy((*ll)[i+1:], (*ll)[i:])
		(*ll)[i] = lx
		return
	}
	*ll = append(*ll, lx)
}

// Sort sorts the lex elements by starting pos, and ending pos *decreasing* if a tie
func (ll *Line) Sort() {
	sort.Slice((*ll), func(i, j int) bool {
		return (*ll)[i].St < (*ll)[j].St || ((*ll)[i].St == (*ll)[j].St && (*ll)[i].Ed > (*ll)[j].Ed)
	})
}

// DeleteIdx deletes at given index
func (ll *Line) DeleteIdx(idx int) {
	*ll = append((*ll)[:idx], (*ll)[idx+1:]...)
}

// DeleteToken deletes a specific token type from list
func (ll *Line) DeleteToken(tok token.Tokens) {
	nt := len(*ll)
	for i := nt - 1; i >= 0; i-- { // remove
		t := (*ll)[i]
		if t.Tok.Tok == tok {
			ll.DeleteIdx(i)
		}
	}
}

// RuneStrings returns array of strings for Lex regions defined in Line, for
// given rune source string
func (ll *Line) RuneStrings(rstr []rune) []string {
	regs := make([]string, len(*ll))
	for i, t := range *ll {
		regs[i] = string(rstr[t.St:t.Ed])
	}
	return regs
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
	tl := make(Line, sz1, tsz)
	copy(tl, t1)
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
		s := t.Src(src)
		str += t.String() + `"` + string(s) + `"` + " "
	}
	return str
}

// Strings returns a slice of strings for each of the Lex items in given rune src
// split by Line Lex's.  Returns nil if Line empty.
func (ll *Line) Strings(src []rune) []string {
	nl := len(*ll)
	if nl == 0 {
		return nil
	}
	sa := make([]string, nl)
	for i, t := range *ll {
		sa[i] = string(t.Src(src))
	}
	return sa
}

// NonCodeWords returns a Line of white-space separated word tokens in given tagged source
// that ignores token.IsCode token regions -- i.e., the "regular" words
// present in the source line -- this is useful for things like spell checking
// or manual parsing.
func (ll *Line) NonCodeWords(src []rune) Line {
	wsrc := sliceclone.Rune(src)
	for _, t := range *ll { // blank out code parts first
		if t.Tok.Tok.IsCode() {
			for i := t.St; i < t.Ed; i++ {
				wsrc[i] = ' '
			}
		}
	}
	return RuneFields(wsrc)
}

// RuneFields returns a Line of Lex's defining the non-white-space "fields"
// in the given rune string
func RuneFields(src []rune) Line {
	if len(src) == 0 {
		return nil
	}
	var ln Line
	cur := Lex{}
	pspc := unicode.IsSpace(src[0])
	cspc := pspc
	for i, r := range src {
		cspc = unicode.IsSpace(r)
		if pspc {
			if !cspc {
				cur.St = i
			}
		} else {
			if cspc {
				cur.Ed = i
				ln.Add(cur)
			}
		}
		pspc = cspc
	}
	if !pspc {
		cur.Ed = len(src)
		cur.Now()
		ln.Add(cur)
	}
	return ln
}
