// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tex

import (
	_ "embed"
	"strings"
	"unicode"

	"cogentcore.org/core/fileinfo"
	"cogentcore.org/core/gox/indent"
	"cogentcore.org/core/parse"
	"cogentcore.org/core/parse/langs"
	"cogentcore.org/core/parse/langs/bibtex"
	"cogentcore.org/core/parse/lexer"
	"cogentcore.org/core/parse/syms"
)

//go:embed tex.parse
var parserBytes []byte

// TexLang implements the Lang interface for the Tex / LaTeX language
type TexLang struct {
	Pr *parse.Parser

	// bibliography files that have been loaded, keyed by file path from bibfile metadata stored in filestate
	Bibs bibtex.Files
}

// TheTexLang is the instance variable providing support for the Go language
var TheTexLang = TexLang{}

func init() {
	parse.StandardLangProperties[fileinfo.TeX].Lang = &TheTexLang
	langs.ParserBytes[fileinfo.TeX] = parserBytes
}

func (tl *TexLang) Parser() *parse.Parser {
	if tl.Pr != nil {
		return tl.Pr
	}
	lp, _ := parse.LangSupport.Properties(fileinfo.TeX)
	if lp.Parser == nil {
		parse.LangSupport.OpenStandard()
	}
	tl.Pr = lp.Parser
	if tl.Pr == nil {
		return nil
	}
	return tl.Pr
}

func (tl *TexLang) ParseFile(fss *parse.FileStates, txt []byte) {
	pr := tl.Parser()
	if pr == nil {
		return
	}
	pfs := fss.StartProc(txt) // current processing one
	pr.LexAll(pfs)
	tl.OpenBibfile(fss, pfs)
	fss.EndProc() // now done
	// no parser
}

func (tl *TexLang) LexLine(fs *parse.FileState, line int, txt []rune) lexer.Line {
	pr := tl.Parser()
	if pr == nil {
		return nil
	}
	return pr.LexLine(fs, line, txt)
}

func (tl *TexLang) ParseLine(fs *parse.FileState, line int) *parse.FileState {
	// n/a
	return nil
}

func (tl *TexLang) HiLine(fss *parse.FileStates, line int, txt []rune) lexer.Line {
	fs := fss.Done()
	return tl.LexLine(fs, line, txt)
}

func (tl *TexLang) ParseDir(fs *parse.FileState, path string, opts parse.LangDirOpts) *syms.Symbol {
	// n/a
	return nil
}

// IndentLine returns the indentation level for given line based on
// previous line's indentation level, and any delta change based on
// e.g., brackets starting or ending the previous or current line, or
// other language-specific keywords.  See lexer.BracketIndentLine for example.
// Indent level is in increments of tabSz for spaces, and tabs for tabs.
// Operates on rune source with markup lex tags per line.
func (tl *TexLang) IndentLine(fs *parse.FileStates, src [][]rune, tags []lexer.Line, ln int, tabSz int) (pInd, delInd, pLn int, ichr indent.Char) {
	pInd, pLn, ichr = lexer.PrevLineIndent(src, tags, ln, tabSz)

	curUnd, _ := lexer.LineStartEndBracket(src[ln], tags[ln])
	_, prvInd := lexer.LineStartEndBracket(src[pLn], tags[pLn])

	delInd = 0
	switch {
	case prvInd && curUnd:
		delInd = 0 // offset
	case prvInd:
		delInd = 1 // indent
	case curUnd:
		delInd = -1 // undent
	}

	pst := lexer.FirstNonSpaceRune(src[pLn])
	cst := lexer.FirstNonSpaceRune(src[ln])

	pbeg := false
	if pst >= 0 {
		sts := string(src[pLn][pst:])
		if strings.HasPrefix(sts, "\\begin{") {
			pbeg = true
		}
	}

	cend := false
	if cst >= 0 {
		sts := string(src[ln][cst:])
		if strings.HasPrefix(sts, "\\end{") {
			cend = true
		}
	}

	switch {
	case pbeg && cend:
		delInd = 0
	case pbeg:
		delInd = 1
	case cend:
		delInd = -1
	}

	if pInd == 0 && delInd < 0 { // error..
		delInd = 0
	}
	return
}

// AutoBracket returns what to do when a user types a starting bracket character
// (bracket, brace, paren) while typing.
// pos = position where bra will be inserted, and curLn is the current line
// match = insert the matching ket, and newLine = insert a new line.
func (tl *TexLang) AutoBracket(fs *parse.FileStates, bra rune, pos lexer.Pos, curLn []rune) (match, newLine bool) {
	lnLen := len(curLn)
	match = pos.Ch == lnLen || unicode.IsSpace(curLn[pos.Ch]) // at end or if space after
	newLine = false
	return
}
