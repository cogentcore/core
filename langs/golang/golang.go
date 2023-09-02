// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"goki.dev/ki/v2/indent"
	"goki.dev/pi/v2/filecat"
	"goki.dev/pi/v2/langs"
	"goki.dev/pi/v2/lex"
	"goki.dev/pi/v2/pi"
	"goki.dev/pi/v2/token"
)

//go:embed go.pi
var parserBytes []byte

// GoLang implements the Lang interface for the Go language
type GoLang struct {
	Pr *pi.Parser
}

// TheGoLang is the instance variable providing support for the Go language
var TheGoLang = GoLang{}

func init() {
	pi.StdLangProps[filecat.Go].Lang = &TheGoLang
	langs.ParserBytes[filecat.Go] = parserBytes
}

func (gl *GoLang) Parser() *pi.Parser {
	if gl.Pr != nil {
		return gl.Pr
	}
	lp, _ := pi.LangSupport.Props(filecat.Go)
	if lp.Parser == nil {
		pi.LangSupport.OpenStd()
	}
	gl.Pr = lp.Parser
	if gl.Pr == nil {
		return nil
	}
	return gl.Pr
}

// ParseFile is the main point of entry for external calls into the parser
func (gl *GoLang) ParseFile(fss *pi.FileStates, txt []byte) {
	pr := gl.Parser()
	if pr == nil {
		log.Println("ParseFile: no parser -- must call pi.LangSupport.OpenStd() at startup!")
		return
	}
	pfs := fss.StartProc(txt) // current processing one
	// lprf := prof.Start("LexAll")
	pr.LexAll(pfs)
	// lprf.End()
	// pprf := prof.Start("ParseAll")
	pr.ParseAll(pfs)
	// pprf.End()
	fss.EndProc() // only symbols still need locking, done separately
	path, _ := filepath.Split(pfs.Src.Filename)
	if len(pfs.ParseState.Scopes) > 0 { // should be for complete files, not for snippets
		pkg := pfs.ParseState.Scopes[0]
		pfs.Syms[pkg.Name] = pkg // keep around..
		// fmt.Printf("main pkg name: %v\n", pkg.Name)
		path = strings.TrimSuffix(path, string([]rune{filepath.Separator}))
		pfs.WaitGp.Add(1)
		go gl.AddPathToSyms(pfs, path)
		go gl.AddImportsToExts(fss, pfs, pkg) // will do ResolveTypes when it finishes
	} else {
		if TraceTypes {
			fmt.Printf("not importing scope for: %v\n", path)
		}
	}
}

func (gl *GoLang) LexLine(fs *pi.FileState, line int, txt []rune) lex.Line {
	pr := gl.Parser()
	if pr == nil {
		return nil
	}
	return pr.LexLine(fs, line, txt)
}

func (gl *GoLang) ParseLine(fs *pi.FileState, line int) *pi.FileState {
	pr := gl.Parser()
	if pr == nil {
		return nil
	}
	lfs := pr.ParseLine(fs, line) // should highlight same line?
	return lfs
}

func (gl *GoLang) HiLine(fss *pi.FileStates, line int, txt []rune) lex.Line {
	pr := gl.Parser()
	if pr == nil {
		return nil
	}
	pfs := fss.Done()
	ll := pr.LexLine(pfs, line, txt)
	lfs := pr.ParseLine(pfs, line)
	if lfs != nil {
		ll = lfs.Src.Lexs[0]
		cml := pfs.Src.Comments[line]
		merge := lex.MergeLines(ll, cml)
		mc := merge.Clone()
		if len(cml) > 0 {
			initDepth := pfs.Src.PrevDepth(line)
			pr.PassTwo.NestDepthLine(mc, initDepth)
		}
		lfs.Syms.WriteDoc(os.Stdout, 0)
		return mc
	} else {
		return ll
	}
}

// IndentLine returns the indentation level for given line based on
// previous line's indentation level, and any delta change based on
// e.g., brackets starting or ending the previous or current line, or
// other language-specific keywords.  See lex.BracketIndentLine for example.
// Indent level is in increments of tabSz for spaces, and tabs for tabs.
// Operates on rune source with markup lex tags per line.
func (gl *GoLang) IndentLine(fs *pi.FileStates, src [][]rune, tags []lex.Line, ln int, tabSz int) (pInd, delInd, pLn int, ichr indent.Char) {
	pInd, pLn, ichr = lex.PrevLineIndent(src, tags, ln, tabSz)

	curUnd, _ := lex.LineStartEndBracket(src[ln], tags[ln])
	_, prvInd := lex.LineStartEndBracket(src[pLn], tags[pLn])

	brackParen := false      // true if line only has bracket and paren -- outdent current
	if len(tags[pLn]) >= 2 { // allow for comments
		pl := tags[pLn][0]
		ll := tags[pLn][1]
		if ll.Tok.Tok == token.PunctGpRParen && pl.Tok.Tok == token.PunctGpRBrace {
			brackParen = true
		}
	}

	delInd = 0
	if brackParen {
		delInd-- // outdent
	}
	switch {
	case prvInd && curUnd:
	case prvInd:
		delInd++
	case curUnd:
		delInd--
	}

	pwrd := lex.FirstWord(string(src[pLn]))
	cwrd := lex.FirstWord(string(src[ln]))

	if cwrd == "case" || cwrd == "default" {
		if pwrd == "switch" {
			delInd = 0
		} else if pwrd == "case" {
			delInd = 0
		} else {
			delInd = -1
		}
	} else if pwrd == "case" || pwrd == "default" {
		delInd = 1
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
func (gl *GoLang) AutoBracket(fs *pi.FileStates, bra rune, pos lex.Pos, curLn []rune) (match, newLine bool) {
	lnLen := len(curLn)
	if bra == '{' {
		if pos.Ch == lnLen {
			if lnLen == 0 || unicode.IsSpace(curLn[pos.Ch-1]) {
				newLine = true
			}
			match = true
		} else {
			match = unicode.IsSpace(curLn[pos.Ch])
		}
	} else {
		match = pos.Ch == lnLen || unicode.IsSpace(curLn[pos.Ch]) // at end or if space after
	}
	return
}
