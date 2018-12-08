// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"github.com/goki/pi/token"
)

// File contains the contents of the file being parsed -- all kept in
// memory, and represented by Line as runes, so that positions in
// the file are directly convertible to indexes in Lines structure
type File struct {
	Filename string    `desc:"the current file being lex'd"`
	Lines    *[][]rune `desc:"contents of the file as lines of runes"`
	Lexs     []Line    `desc:"lex'd version of the lines -- allocated to size of Lines"`
	Comments []Line    `desc:"comment tokens are stored separately here, so parser doesn't need to worry about them, but they are available for highlighting and other uses"`
}

// SetSrc sets the source to given content, and alloc Lexs
func (fl *File) SetSrc(src *[][]rune, fname string) {
	fl.Filename = fname
	fl.Lines = src
	fl.AllocLexs()
}

// AllocLexs allocates the lexs output lines
func (fl *File) AllocLexs() {
	if fl.Lines == nil {
		return
	}
	nlines := fl.NLines()
	if fl.Lexs != nil {
		if cap(fl.Lexs) >= nlines {
			fl.Lexs = fl.Lexs[:nlines]
		} else {
			fl.Lexs = make([]Line, nlines)
		}
	} else {
		fl.Lexs = make([]Line, nlines)
	}
	if fl.Comments != nil {
		if cap(fl.Comments) >= nlines {
			fl.Comments = fl.Comments[:nlines]
		} else {
			fl.Comments = make([]Line, nlines)
		}
	} else {
		fl.Comments = make([]Line, nlines)
	}
}

// NLines returns the number of lines in source
func (fl *File) NLines() int {
	if fl.Lines == nil {
		return 0
	}
	return len(*fl.Lines)
}

// SetLexs sets the lex output for given line -- does a copy
func (fl *File) SetLexs(ln int, lexs, comments Line) {
	if len(fl.Lexs) <= ln {
		fl.AllocLexs()
	}
	if len(fl.Lexs) <= ln {
		return
	}
	fl.Lexs[ln] = lexs.Clone()
	fl.Comments[ln] = comments.Clone()
}

// LexLine returns the lexing output for given line, combining comments and all other tokens
// and allocating new memory using clone
func (fl *File) LexLine(ln int) Line {
	if len(fl.Lexs) <= ln {
		return nil
	}
	merge := MergeLines(fl.Lexs[ln], fl.Comments[ln])
	return merge.Clone()
}

// NTokens returns number of lex tokens for given line
func (fl *File) NTokens(ln int) int {
	if fl.Lexs == nil {
		return 0
	}
	if len(fl.Lexs) <= ln {
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

// PrevDepth returns the depth of the token immediately prior to given line
func (fl *File) PrevDepth(ln int) int {
	pos := Pos{ln, 0}
	pos, ok := fl.PrevTokenPos(pos)
	if !ok {
		return 0
	}
	return fl.LexAt(pos).Depth
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
	return (*fl.Lines)[pos.Ln][lx.St:lx.Ed]
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
	if !fl.IsLexPosValid(reg.St) || reg.IsNil() {
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
			return string((*fl.Lines)[reg.Ed.Ln][reg.St.Ch:reg.Ed.Ch])
		} else {
			return ""
		}
	}
	src := string((*fl.Lines)[reg.St.Ln][reg.St.Ch:])
	for ln := reg.St.Ln + 1; ln < reg.Ed.Ln; ln++ {
		src += "|" + string((*fl.Lines)[ln])
	}
	src += "|" + string((*fl.Lines)[reg.Ed.Ln][:reg.Ed.Ch])
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
	return fl.Lexs[ln].TagSrc((*fl.Lines)[ln])
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
