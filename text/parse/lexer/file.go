// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer

import (
	"bytes"
	"io"
	"log"
	"os"
	"slices"
	"strings"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
)

// File contains the contents of the file being parsed -- all kept in
// memory, and represented by Line as runes, so that positions in
// the file are directly convertible to indexes in Lines structure
type File struct {

	// the current file being lex'd
	Filename string

	// the known file type, if known (typically only known files are processed)
	Known fileinfo.Known

	// base path for reporting file names -- this must be set externally e.g., by gide for the project root path
	BasePath string

	// lex'd version of the lines -- allocated to size of Lines
	Lexs []Line

	// comment tokens are stored separately here, so parser doesn't need to worry about them, but they are available for highlighting and other uses
	Comments []Line

	// stack present at the end of each line -- needed for contextualizing line-at-time lexing while editing
	LastStacks []Stack

	// token positions per line for the EOS (end of statement) tokens -- very important for scoping top-down parsing
	EosPos []EosPos

	// contents of the file as lines of runes
	Lines [][]rune
}

// SetSrc sets the source to given content, and alloc Lexs -- if basepath is empty
// then it is set to the path for the filename
func (fl *File) SetSrc(src [][]rune, fname, basepath string, known fileinfo.Known) {
	fl.Filename = fname
	if basepath != "" {
		fl.BasePath = basepath
	}
	fl.Known = known
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

// LinesInserted inserts new lines -- called e.g., by core.TextBuf to sync
// the markup with ongoing edits
func (fl *File) LinesInserted(stln, nlns int) {
	// Lexs
	n := len(fl.Lexs)
	if stln > n {
		stln = n
	}
	fl.Lexs = slices.Insert(fl.Lexs, stln, make([]Line, nlns)...)
	fl.Comments = slices.Insert(fl.Comments, stln, make([]Line, nlns)...)
	fl.LastStacks = slices.Insert(fl.LastStacks, stln, make([]Stack, nlns)...)
	fl.EosPos = slices.Insert(fl.EosPos, stln, make([]EosPos, nlns)...)
}

// LinesDeleted deletes lines -- called e.g., by core.TextBuf to sync
// the markup with ongoing edits
func (fl *File) LinesDeleted(stln, edln int) {
	edln = min(edln, len(fl.Lexs))
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
	alltxt, err := io.ReadAll(fp)
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
	known := fileinfo.KnownFromFile(fname)
	fl.SetSrc(rns, fname, "", known)
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
	fl.Lines[ln] = slices.Clone(txt)
	return true
}

// InitFromLine initializes from one line of source file
func (fl *File) InitFromLine(sfl *File, ln int) bool {
	nlines := sfl.NLines()
	if ln >= nlines || ln < 0 {
		return false
	}
	src := [][]rune{sfl.Lines[ln], {}} // need extra blank
	fl.SetSrc(src, sfl.Filename, sfl.BasePath, sfl.Known)
	fl.Lexs = []Line{sfl.Lexs[ln], {}}
	fl.Comments = []Line{sfl.Comments[ln], {}}
	fl.EosPos = []EosPos{sfl.EosPos[ln], {}}
	return true
}

// InitFromString initializes from given string. Returns false if string is empty
func (fl *File) InitFromString(str string, fname string, known fileinfo.Known) bool {
	if str == "" {
		return false
	}
	src := RunesFromString(str)
	if len(src) == 1 { // need more than 1 line
		src = append(src, []rune{})
	}
	fl.SetSrc(src, fname, "", known)
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

// LexLine returns the lexing output for given line,
// combining comments and all other tokens
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
func (fl *File) IsLexPosValid(pos textpos.Pos) bool {
	if pos.Line < 0 || pos.Line >= fl.NLines() {
		return false
	}
	nt := fl.NTokens(pos.Line)
	if pos.Char < 0 || pos.Char >= nt {
		return false
	}
	return true
}

// LexAt returns Lex item at given position, with no checking
func (fl *File) LexAt(cp textpos.Pos) *Lex {
	return &fl.Lexs[cp.Line][cp.Char]
}

// LexAtSafe returns the Lex item at given position, or last lex item if beyond end
func (fl *File) LexAtSafe(cp textpos.Pos) Lex {
	nln := fl.NLines()
	if nln == 0 {
		return Lex{}
	}
	if cp.Line >= nln {
		cp.Line = nln - 1
	}
	sz := len(fl.Lexs[cp.Line])
	if sz == 0 {
		if cp.Line > 0 {
			cp.Line--
			return fl.LexAtSafe(cp)
		}
		return Lex{}
	}
	if cp.Char < 0 {
		cp.Char = 0
	}
	if cp.Char >= sz {
		cp.Char = sz - 1
	}
	return *fl.LexAt(cp)
}

// ValidTokenPos returns the next valid token position starting at given point,
// false if at end of tokens
func (fl *File) ValidTokenPos(pos textpos.Pos) (textpos.Pos, bool) {
	for pos.Char >= fl.NTokens(pos.Line) {
		pos.Line++
		pos.Char = 0
		if pos.Line >= fl.NLines() {
			pos.Line = fl.NLines() - 1 // make valid
			return pos, false
		}
	}
	return pos, true
}

// NextTokenPos returns the next token position, false if at end of tokens
func (fl *File) NextTokenPos(pos textpos.Pos) (textpos.Pos, bool) {
	pos.Char++
	return fl.ValidTokenPos(pos)
}

// PrevTokenPos returns the previous token position, false if at end of tokens
func (fl *File) PrevTokenPos(pos textpos.Pos) (textpos.Pos, bool) {
	pos.Char--
	if pos.Char < 0 {
		pos.Line--
		if pos.Line < 0 {
			return pos, false
		}
		for fl.NTokens(pos.Line) == 0 {
			pos.Line--
			if pos.Line < 0 {
				pos.Line = 0
				pos.Char = 0
				return pos, false
			}
		}
		pos.Char = fl.NTokens(pos.Line) - 1
	}
	return pos, true
}

// Token gets lex token at given Pos (Ch = token index)
func (fl *File) Token(pos textpos.Pos) token.KeyToken {
	return fl.Lexs[pos.Line][pos.Char].Token
}

// PrevDepth returns the depth of the token immediately prior to given line
func (fl *File) PrevDepth(ln int) int {
	pos := textpos.Pos{ln, 0}
	pos, ok := fl.PrevTokenPos(pos)
	if !ok {
		return 0
	}
	lx := fl.LexAt(pos)
	depth := lx.Token.Depth
	if lx.Token.Token.IsPunctGpLeft() {
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
func (fl *File) TokenMapReg(reg textpos.Region) TokenMap {
	m := make(TokenMap)
	cp, ok := fl.ValidTokenPos(reg.Start)
	for ok && cp.IsLess(reg.End) {
		tok := fl.Token(cp).Token
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
func (fl *File) TokenSrc(pos textpos.Pos) []rune {
	if !fl.IsLexPosValid(pos) {
		return nil
	}
	lx := fl.Lexs[pos.Line][pos.Char]
	return fl.Lines[pos.Line][lx.Start:lx.End]
}

// TokenSrcPos returns source reg associated with lex token at given token position
func (fl *File) TokenSrcPos(pos textpos.Pos) textpos.Region {
	if !fl.IsLexPosValid(pos) {
		return textpos.Region{}
	}
	lx := fl.Lexs[pos.Line][pos.Char]
	return textpos.Region{Start: textpos.Pos{pos.Line, lx.Start}, End: textpos.Pos{pos.Line, lx.End}}
}

// TokenSrcReg translates a region of tokens into a region of source
func (fl *File) TokenSrcReg(reg textpos.Region) textpos.Region {
	if !fl.IsLexPosValid(reg.Start) || reg.IsNil() {
		return textpos.Region{}
	}
	st := fl.Lexs[reg.Start.Line][reg.Start.Char].Start
	ep, _ := fl.PrevTokenPos(reg.End) // ed is exclusive -- go to prev
	ed := fl.Lexs[ep.Line][ep.Char].End
	return textpos.Region{Start: textpos.Pos{reg.Start.Line, st}, End: textpos.Pos{ep.Line, ed}}
}

// RegSrc returns the source (as a string) for given region
func (fl *File) RegSrc(reg textpos.Region) string {
	if reg.End.Line == reg.Start.Line {
		if reg.End.Char > reg.Start.Char {
			return string(fl.Lines[reg.End.Line][reg.Start.Char:reg.End.Char])
		}
		return ""
	}
	src := string(fl.Lines[reg.Start.Line][reg.Start.Char:])
	nln := reg.End.Line - reg.Start.Line
	if nln > 10 {
		src += "|>" + string(fl.Lines[reg.Start.Line+1]) + "..."
		src += "|>" + string(fl.Lines[reg.End.Line-1])
		return src
	}
	for ln := reg.Start.Line + 1; ln < reg.End.Line; ln++ {
		src += "|>" + string(fl.Lines[ln])
	}
	src += "|>" + string(fl.Lines[reg.End.Line][:reg.End.Char])
	return src
}

// TokenRegSrc returns the source code associated with the given token region
func (fl *File) TokenRegSrc(reg textpos.Region) string {
	if !fl.IsLexPosValid(reg.Start) {
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
func (fl *File) InsertEos(cp textpos.Pos) textpos.Pos {
	np := textpos.Pos{cp.Line, cp.Char + 1}
	elx := fl.LexAt(cp)
	depth := elx.Token.Depth
	fl.Lexs[cp.Line].Insert(np.Char, Lex{Token: token.KeyToken{Token: token.EOS, Depth: depth}, Start: elx.End, End: elx.End})
	fl.EosPos[np.Line] = append(fl.EosPos[np.Line], np.Char)
	return np
}

// ReplaceEos replaces given token with an EOS
func (fl *File) ReplaceEos(cp textpos.Pos) {
	clex := fl.LexAt(cp)
	clex.Token.Token = token.EOS
	fl.EosPos[cp.Line] = append(fl.EosPos[cp.Line], cp.Char)
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
	ep := textpos.Pos{ln, sz - 1}
	elx := fl.LexAt(ep)
	if elx.Token.Token == token.EOS {
		return
	}
	fl.InsertEos(ep)
}

// NextEos finds the next EOS position at given depth, false if none
func (fl *File) NextEos(stpos textpos.Pos, depth int) (textpos.Pos, bool) {
	// prf := profile.Start("NextEos")
	// defer prf.End()

	ep := stpos
	nlines := fl.NLines()
	if stpos.Line >= nlines {
		return ep, false
	}
	eps := fl.EosPos[stpos.Line]
	for i := range eps {
		if eps[i] < stpos.Char {
			continue
		}
		ep.Char = eps[i]
		lx := fl.LexAt(ep)
		if lx.Token.Depth == depth {
			return ep, true
		}
	}
	for ep.Line = stpos.Line + 1; ep.Line < nlines; ep.Line++ {
		eps := fl.EosPos[ep.Line]
		sz := len(eps)
		if sz == 0 {
			continue
		}
		for i := 0; i < sz; i++ {
			ep.Char = eps[i]
			lx := fl.LexAt(ep)
			if lx.Token.Depth == depth {
				return ep, true
			}
		}
	}
	return ep, false
}

// NextEosAnyDepth finds the next EOS at any depth
func (fl *File) NextEosAnyDepth(stpos textpos.Pos) (textpos.Pos, bool) {
	ep := stpos
	nlines := fl.NLines()
	if stpos.Line >= nlines {
		return ep, false
	}
	eps := fl.EosPos[stpos.Line]
	if np := eps.FindGtEq(stpos.Char); np >= 0 {
		ep.Char = np
		return ep, true
	}
	ep.Char = 0
	for ep.Line = stpos.Line + 1; ep.Line < nlines; ep.Line++ {
		sz := len(fl.EosPos[ep.Line])
		if sz == 0 {
			continue
		}
		return ep, true
	}
	return ep, false
}
