// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package src provides source file structures
package lex

import (
	"fmt"

	"github.com/goki/pi/token"
)

// Pos is a position within the source file -- it is recorded always in 0, 0
// offset positions, but is converted into 1,1 offset for public consumption
// Ch positions are always in runes, not bytes.  Also used for lex token indexes.
type Pos struct {
	Ln int
	Ch int
}

// String satisfies the fmt.Stringer interferace
func (ps Pos) String() string {
	s := fmt.Sprintf("%d", ps.Ln+1)
	if ps.Ch != 0 {
		s += fmt.Sprintf(":%d", ps.Ch)
	}
	return s
}

// PosZero is the uninitialized zero text position (which is
// still a valid position)
var PosZero = Pos{}

// PosErr represents an error text position (-1 for both line and char)
// used as a return value for cases where error positions are possible
var PosErr = Pos{-1, -1}

// IsLess returns true if receiver position is less than given comparison
func (ps *Pos) IsLess(cmp Pos) bool {
	switch {
	case ps.Ln < cmp.Ln:
		return true
	case ps.Ln == cmp.Ln:
		return ps.Ch < cmp.Ch
	default:
		return false
	}
}

// Reg is a contiguous region within the source file
type Reg struct {
	St Pos `desc:"starting position of region"`
	Ed Pos `desc:"ending position of region"`
}

// RegZero is the zero region
var RegZero = Reg{}

// IsNil checks if the region is empty, because the start is after or equal to the end
func (tr Reg) IsNil() bool {
	return !tr.St.IsLess(tr.Ed)
}

// TokenMap is a token map, for optimizing token exclusion
type TokenMap map[token.Tokens]struct{}

// Set sets map for given token
func (tm TokenMap) Set(tok token.Tokens) {
	tm[tok] = struct{}{}
}

// Has returns true if given token is in the map
func (tm TokenMap) Has(tok token.Tokens) bool {
	_, has := tm[tok]
	return has
}

// File contains the contents of the file being parsed -- all kept in
// memory, and represented by Line as runes, so that positions in
// the file are directly convertible to indexes in Lines structure
type File struct {
	Filename string   `desc:"the current file being lex'd"`
	Lines    [][]rune `desc:"contents of the file as lines of runes"`
	Lexs     []Line   `desc:"lex'd version of the lines -- allocated to size of Lines"`
}

// SetSrc sets the source to given content, and alloc Lexs
func (fl *File) SetSrc(src [][]rune, fname string) {
	fl.Filename = fname
	fl.Lines = src
	fl.Lexs = make([]Line, len(src))
}

// NLines returns the number of lines in source
func (fl *File) NLines() int {
	return len(fl.Lines)
}

// SetLexs sets the lex output for given line -- does a copy
func (fl *File) SetLexs(ln int, lexs Line) {
	fl.Lexs[ln] = lexs.Clone()
}

// NTokens returns number of lex tokens for given line
func (fl *File) NTokens(ln int) int {
	if fl.Lexs == nil {
		return 0
	}
	return len(fl.Lexs[ln])
}

// IsLexPosValid returns true if given lexical token position is valid
func (fl *File) IsLexPosValid(pos Pos) bool {
	if pos.Ln < 0 || pos.Ln >= fl.NLines() {
		return false
	}
	nt := fl.NTokens(pos.Ln)
	if pos.Ch < 0 || pos.Ch >= nt {
		return false
	}
	return true
}

// LexAt returns Lex item at given position, with no checking
func (fl *File) LexAt(cp Pos) *Lex {
	return &fl.Lexs[cp.Ln][cp.Ch]
}

// LexAtSafe returns the Lex item at given position, or last lex item if beyond end
func (fl *File) LexAtSafe(cp Pos) Lex {
	nln := fl.NLines()
	if nln == 0 {
		return Lex{}
	}
	if cp.Ln >= nln {
		cp.Ln = nln - 1
	}
	sz := len(fl.Lexs[cp.Ln])
	if sz == 0 {
		if cp.Ln > 0 {
			cp.Ln--
			return fl.LexAtSafe(cp)
		}
		return Lex{}
	}
	if cp.Ch < 0 {
		cp.Ch = 0
	}
	if cp.Ch >= sz {
		cp.Ch = sz - 1
	}
	return *fl.LexAt(cp)
}

// ValidTokenPos returns the next valid token position starting at given point,
// false if at end of tokens
func (fl *File) ValidTokenPos(pos Pos) (Pos, bool) {
	for pos.Ch >= fl.NTokens(pos.Ln) {
		pos.Ln++
		pos.Ch = 0
		if pos.Ln >= fl.NLines() {
			pos.Ln = fl.NLines() - 1 // make valid
			return pos, false
		}
	}
	return pos, true
}

// NextTokenPos returns the next token position, false if at end of tokens
func (fl *File) NextTokenPos(pos Pos) (Pos, bool) {
	pos.Ch++
	return fl.ValidTokenPos(pos)
}

// PrevTokenPos returns the previous token position, false if at end of tokens
func (fl *File) PrevTokenPos(pos Pos) (Pos, bool) {
	pos.Ch--
	if pos.Ch < 0 {
		pos.Ln--
		if pos.Ln < 0 {
			return pos, false
		}
		for fl.NTokens(pos.Ln) == 0 {
			pos.Ln--
			if pos.Ln < 0 {
				pos.Ln = 0
				pos.Ch = 0
				return pos, false
			}
		}
		pos.Ch = fl.NTokens(pos.Ln) - 1
	}
	return pos, true
}

// Token gets lex token at given Pos (Ch = token index)
func (fl *File) Token(pos Pos) token.Tokens {
	return fl.Lexs[pos.Ln][pos.Ch].Tok
}

// TokenMapReg creates a TokenMap of tokens in region, including their
// Cat and SubCat levels -- err's on side of inclusiveness -- used
// for optimizing token matching
func (fl *File) TokenMapReg(reg Reg) TokenMap {
	m := make(TokenMap)
	cp, ok := fl.ValidTokenPos(reg.St)
	for ok && cp.IsLess(reg.Ed) {
		tok := fl.Token(cp)
		m.Set(tok)
		subc := tok.SubCat()
		if subc != tok {
			m.Set(subc)
		}
		cat := tok.Cat()
		if cat != tok {
			m.Set(cat)
		}
		cp, ok = fl.NextTokenPos(cp)
	}
	return m
}

// TokenSrc gets source runes for given token position
func (fl *File) TokenSrc(pos Pos) []rune {
	if !fl.IsLexPosValid(pos) {
		return nil
	}
	lx := fl.Lexs[pos.Ln][pos.Ch]
	return fl.Lines[pos.Ln][lx.St:lx.Ed]
}

// TokenSrcPos returns source reg associated with lex token at given token position
func (fl *File) TokenSrcPos(pos Pos) Reg {
	if !fl.IsLexPosValid(pos) {
		return Reg{}
	}
	lx := fl.Lexs[pos.Ln][pos.Ch]
	return Reg{St: Pos{pos.Ln, lx.St}, Ed: Pos{pos.Ln, lx.Ed}}
}

// TokenSrcReg translates a region of tokens into a region of source
func (fl *File) TokenSrcReg(reg Reg) Reg {
	if !fl.IsLexPosValid(reg.St) {
		return Reg{}
	}
	st := fl.Lexs[reg.St.Ln][reg.St.Ch].St
	ep, _ := fl.PrevTokenPos(reg.Ed) // ed is exclusive -- go to prev
	ed := fl.Lexs[ep.Ln][ep.Ch].Ed
	return Reg{St: Pos{reg.St.Ln, st}, Ed: Pos{ep.Ln, ed}}
}

// RegSrc returns the source (as a string) for given region
func (fl *File) RegSrc(reg Reg) string {
	if reg.Ed.Ln == reg.St.Ln {
		if reg.Ed.Ch > reg.St.Ch {
			return string(fl.Lines[reg.Ed.Ln][reg.St.Ch:reg.Ed.Ch])
		} else {
			return ""
		}
	}
	src := string(fl.Lines[reg.St.Ln][reg.St.Ch:])
	for ln := reg.St.Ln + 1; ln < reg.Ed.Ln; ln++ {
		src += "|" + string(fl.Lines[ln])
	}
	src += "|" + string(fl.Lines[reg.Ed.Ln][:reg.Ed.Ch])
	return src
}

// TokenRegSrc returns the source code associated with the given token region
func (fl *File) TokenRegSrc(reg Reg) string {
	if !fl.IsLexPosValid(reg.St) {
		return ""
	}
	srcreg := fl.TokenSrcReg(reg)
	return fl.RegSrc(srcreg)
}

// LexTagSrcLn returns the lex'd tagged source line for given line
func (fl *File) LexTagSrcLn(ln int) string {
	return fl.Lexs[ln].TagSrc(fl.Lines[ln])
}

// LexTagSrc returns the lex'd tagged source for entire source
func (fl *File) LexTagSrc() string {
	txt := ""
	nlines := fl.NLines()
	for ln := 0; ln < nlines; ln++ {
		txt += fl.LexTagSrcLn(ln) + "\n"
	}
	return txt
}
