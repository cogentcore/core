// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/goki/ki/sliceclone"
	"goki.dev/pi/v2/filecat"
	"goki.dev/pi/v2/token"
)

// File contains the contents of the file being parsed -- all kept in
// memory, and represented by Line as runes, so that positions in
// the file are directly convertible to indexes in Lines structure
type File struct {

	// the current file being lex'd
	Filename string `desc:"the current file being lex'd"`

	// the supported file type, if supported (typically only supported files are processed)
	Sup filecat.Supported `desc:"the supported file type, if supported (typically only supported files are processed)"`

	// base path for reporting file names -- this must be set externally e.g., by gide for the project root path
	BasePath string `desc:"base path for reporting file names -- this must be set externally e.g., by gide for the project root path"`

	// lex'd version of the lines -- allocated to size of Lines
	Lexs []Line `desc:"lex'd version of the lines -- allocated to size of Lines"`

	// comment tokens are stored separately here, so parser doesn't need to worry about them, but they are available for highlighting and other uses
	Comments []Line `desc:"comment tokens are stored separately here, so parser doesn't need to worry about them, but they are available for highlighting and other uses"`

	// stack present at the end of each line -- needed for contextualizing line-at-time lexing while editing
	LastStacks []Stack `desc:"stack present at the end of each line -- needed for contextualizing line-at-time lexing while editing"`

	// token positions per line for the EOS (end of statement) tokens -- very important for scoping top-down parsing
	EosPos []EosPos `desc:"token positions per line for the EOS (end of statement) tokens -- very important for scoping top-down parsing"`

	// contents of the file as lines of runes
	Lines [][]rune `desc:"contents of the file as lines of runes"`
}

// SetSrc sets the source to given content, and alloc Lexs -- if basepath is empty
// then it is set to the path for the filename
func (fl *File) SetSrc(src [][]rune, fname, basepath string, sup filecat.Supported) {
	fl.Filename = fname
	if basepath != "" {
		fl.BasePath = basepath
	}
	fl.Sup = sup
	fl.Lines = src
	fl.AllocLines()
}

// AllocLines allocates the data per line: lex outputs and stack.
// We reset state so stale state is not hanging around.
func (fl *File) AllocLines() {
	if fl.Lines == nil {
		return
	}
	nlines := fl.NLines()
	fl.Lexs = make([]Line, nlines)
	fl.Comments = make([]Line, nlines)
	fl.LastStacks = make([]Stack, nlines)
	fl.EosPos = make([]EosPos, nlines)
}

// LinesInserted inserts new lines -- called e.g., by giv.TextBuf to sync
// the markup with ongoing edits
func (fl *File) LinesInserted(stln, nlns int) {
	// Lexs
	tmplx := make([]Line, nlns)
	nlx := append(fl.Lexs, tmplx...)
	copy(nlx[stln+nlns:], nlx[stln:])
	copy(nlx[stln:], tmplx)
	fl.Lexs = nlx

	// Comments
	tmpcm := make([]Line, nlns)
	ncm := append(fl.Comments, tmpcm...)
	copy(ncm[stln+nlns:], ncm[stln:])
	copy(ncm[stln:], tmpcm)
	fl.Comments = ncm

	// LastStacks
	tmpls := make([]Stack, nlns)
	nls := append(fl.LastStacks, tmpls...)
	copy(nls[stln+nlns:], nls[stln:])
	copy(nls[stln:], tmpls)
	fl.LastStacks = nls

	// EosPos
	tmpep := make([]EosPos, nlns)
	nep := append(fl.EosPos, tmpep...)
	copy(nep[stln+nlns:], nep[stln:])
	copy(nep[stln:], tmpep)
	fl.EosPos = nep
}

// LinesDeleted deletes lines -- called e.g., by giv.TextBuf to sync
// the markup with ongoing edits
func (fl *File) LinesDeleted(stln, edln int) {
	fl.Lexs = append(fl.Lexs[:stln], fl.Lexs[edln:]...)
	fl.Comments = append(fl.Comments[:stln], fl.Comments[edln:]...)
	fl.LastStacks = append(fl.LastStacks[:stln], fl.LastStacks[edln:]...)
	fl.EosPos = append(fl.EosPos[:stln], fl.EosPos[edln:]...)
}

// RunesFromBytes returns the lines of runes from a basic byte array
func RunesFromBytes(b []byte) [][]rune {
	lns := bytes.Split(b, []byte("\n"))
	nlines := len(lns)
	rns := make([][]rune, nlines)
	for ln, txt := range lns {
		rns[ln] = bytes.Runes(txt)
	}
	return rns
}

// RunesFromString returns the lines of runes from a string (more efficient
// than converting to bytes)
func RunesFromString(str string) [][]rune {
	lns := strings.Split(str, "\n")
	nlines := len(lns)
	rns := make([][]rune, nlines)
	for ln, txt := range lns {
		rns[ln] = []rune(txt)
	}
	return rns
}

// OpenFileBytes returns bytes in given file, and logs any errors as well
func OpenFileBytes(fname string) ([]byte, error) {
	fp, err := os.Open(fname)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	alltxt, err := ioutil.ReadAll(fp)
	fp.Close()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return alltxt, nil
}

// OpenFile sets source to be parsed from given filename
func (fl *File) OpenFile(fname string) error {
	alltxt, err := OpenFileBytes(fname)
	if err != nil {
		return err
	}
	rns := RunesFromBytes(alltxt)
	sup := filecat.SupportedFromFile(fname)
	fl.SetSrc(rns, fname, "", sup)
	return nil
}

// SetBytes sets source to be parsed from given bytes
func (fl *File) SetBytes(txt []byte) {
	if txt == nil {
		return
	}
	fl.Lines = RunesFromBytes(txt)
	fl.AllocLines()
}

// SetLineSrc sets source runes from given line of runes.
// Returns false if out of range.
func (fl *File) SetLineSrc(ln int, txt []rune) bool {
	nlines := fl.NLines()
	if ln >= nlines || ln < 0 || txt == nil {
		return false
	}
	fl.Lines[ln] = sliceclone.Rune(txt)
	return true
}

// InitFromLine initializes from one line of source file
func (fl *File) InitFromLine(sfl *File, ln int) bool {
	nlines := sfl.NLines()
	if ln >= nlines || ln < 0 {
		return false
	}
	src := [][]rune{sfl.Lines[ln], {}} // need extra blank
	fl.SetSrc(src, sfl.Filename, sfl.BasePath, sfl.Sup)
	fl.Lexs = []Line{sfl.Lexs[ln], {}}
	fl.Comments = []Line{sfl.Comments[ln], {}}
	fl.EosPos = []EosPos{sfl.EosPos[ln], {}}
	return true
}

// InitFromString initializes from given string. Returns false if string is empty
func (fl *File) InitFromString(str string, fname string, sup filecat.Supported) bool {
	if str == "" {
		return false
	}
	src := RunesFromString(str)
	if len(src) == 1 { // need more than 1 line
		src = append(src, []rune{})
	}
	fl.SetSrc(src, fname, "", sup)
	return true
}

///////////////////////////////////////////////////////////////////////////
//  Accessors

// NLines returns the number of lines in source
func (fl *File) NLines() int {
	if fl.Lines == nil {
		return 0
	}
	return len(fl.Lines)
}

// SrcLine returns given line of source, as a string, or "" if out of range
func (fl *File) SrcLine(ln int) string {
	nlines := fl.NLines()
	if ln < 0 || ln >= nlines {
		return ""
	}
	return string(fl.Lines[ln])
}

// SetLine sets the line data from the lexer -- does a clone to keep the copy
func (fl *File) SetLine(ln int, lexs, comments Line, stack Stack) {
	if len(fl.Lexs) <= ln {
		fl.AllocLines()
	}
	if len(fl.Lexs) <= ln {
		return
	}
	fl.Lexs[ln] = lexs.Clone()
	fl.Comments[ln] = comments.Clone()
	fl.LastStacks[ln] = stack.Clone()
	fl.EosPos[ln] = nil
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
	if fl == nil || fl.Lexs == nil {
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
func (fl *File) Token(pos Pos) token.KeyToken {
	return fl.Lexs[pos.Ln][pos.Ch].Tok
}

// PrevDepth returns the depth of the token immediately prior to given line
func (fl *File) PrevDepth(ln int) int {
	pos := Pos{ln, 0}
	pos, ok := fl.PrevTokenPos(pos)
	if !ok {
		return 0
	}
	lx := fl.LexAt(pos)
	depth := lx.Tok.Depth
	if lx.Tok.Tok.IsPunctGpLeft() {
		depth++
	}
	return depth
}

// PrevStack returns the stack from the previous line
func (fl *File) PrevStack(ln int) Stack {
	if ln <= 0 {
		return nil
	}
	if len(fl.LastStacks) <= ln {
		return nil
	}
	return fl.LastStacks[ln-1]
}

// TokenMapReg creates a TokenMap of tokens in region, including their
// Cat and SubCat levels -- err's on side of inclusiveness -- used
// for optimizing token matching
func (fl *File) TokenMapReg(reg Reg) TokenMap {
	m := make(TokenMap)
	cp, ok := fl.ValidTokenPos(reg.St)
	for ok && cp.IsLess(reg.Ed) {
		tok := fl.Token(cp).Tok
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

/////////////////////////////////////////////////////////////////////
//  Source access from pos, reg, tok

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
			return string(fl.Lines[reg.Ed.Ln][reg.St.Ch:reg.Ed.Ch])
		} else {
			return ""
		}
	}
	src := string(fl.Lines[reg.St.Ln][reg.St.Ch:])
	nln := reg.Ed.Ln - reg.St.Ln
	if nln > 10 {
		src += "|>" + string(fl.Lines[reg.St.Ln+1]) + "..."
		src += "|>" + string(fl.Lines[reg.Ed.Ln-1])
		return src
	}
	for ln := reg.St.Ln + 1; ln < reg.Ed.Ln; ln++ {
		src += "|>" + string(fl.Lines[ln])
	}
	src += "|>" + string(fl.Lines[reg.Ed.Ln][:reg.Ed.Ch])
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

/////////////////////////////////////////////////////////////////
// EOS end of statement processing

// InsertEos inserts an EOS just after the given token position
// (e.g., cp = last token in line)
func (fl *File) InsertEos(cp Pos) Pos {
	np := Pos{cp.Ln, cp.Ch + 1}
	elx := fl.LexAt(cp)
	depth := elx.Tok.Depth
	fl.Lexs[cp.Ln].Insert(np.Ch, Lex{Tok: token.KeyToken{Tok: token.EOS, Depth: depth}, St: elx.Ed, Ed: elx.Ed})
	fl.EosPos[np.Ln] = append(fl.EosPos[np.Ln], np.Ch)
	return np
}

// ReplaceEos replaces given token with an EOS
func (fl *File) ReplaceEos(cp Pos) {
	clex := fl.LexAt(cp)
	clex.Tok.Tok = token.EOS
	fl.EosPos[cp.Ln] = append(fl.EosPos[cp.Ln], cp.Ch)
}

// EnsureFinalEos makes sure that the given line ends with an EOS (if it
// has tokens).
// Used for line-at-time parsing just to make sure it matches even if
// you haven't gotten to the end etc.
func (fl *File) EnsureFinalEos(ln int) {
	if ln >= fl.NLines() {
		return
	}
	sz := len(fl.Lexs[ln])
	if sz == 0 {
		return // can't get depth or anything -- useless
	}
	ep := Pos{ln, sz - 1}
	elx := fl.LexAt(ep)
	if elx.Tok.Tok == token.EOS {
		return
	}
	fl.InsertEos(ep)
}

// NextEos finds the next EOS position at given depth, false if none
func (fl *File) NextEos(stpos Pos, depth int) (Pos, bool) {
	// prf := prof.Start("NextEos")
	// defer prf.End()

	ep := stpos
	nlines := fl.NLines()
	if stpos.Ln >= nlines {
		return ep, false
	}
	eps := fl.EosPos[stpos.Ln]
	for i := range eps {
		if eps[i] < stpos.Ch {
			continue
		}
		ep.Ch = eps[i]
		lx := fl.LexAt(ep)
		if lx.Tok.Depth == depth {
			return ep, true
		}
	}
	for ep.Ln = stpos.Ln + 1; ep.Ln < nlines; ep.Ln++ {
		eps := fl.EosPos[ep.Ln]
		sz := len(eps)
		if sz == 0 {
			continue
		}
		for i := 0; i < sz; i++ {
			ep.Ch = eps[i]
			lx := fl.LexAt(ep)
			if lx.Tok.Depth == depth {
				return ep, true
			}
		}
	}
	return ep, false
}

// NextEosAnyDepth finds the next EOS at any depth
func (fl *File) NextEosAnyDepth(stpos Pos) (Pos, bool) {
	ep := stpos
	nlines := fl.NLines()
	if stpos.Ln >= nlines {
		return ep, false
	}
	eps := fl.EosPos[stpos.Ln]
	if np := eps.FindGtEq(stpos.Ch); np >= 0 {
		ep.Ch = np
		return ep, true
	}
	ep.Ch = 0
	for ep.Ln = stpos.Ln + 1; ep.Ln < nlines; ep.Ln++ {
		sz := len(fl.EosPos[ep.Ln])
		if sz == 0 {
			continue
		}
		return ep, true
	}
	return ep, false
}
