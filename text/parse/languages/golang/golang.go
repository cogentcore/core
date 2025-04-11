// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"unicode"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/indent"
	"cogentcore.org/core/text/parse"
	"cogentcore.org/core/text/parse/languages"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
)

//go:embed go.parse
var parserBytes []byte

// GoLang implements the Lang interface for the Go language
type GoLang struct {
	Pr *parse.Parser
}

// TheGoLang is the instance variable providing support for the Go language
var TheGoLang = GoLang{}

func init() {
	parse.StandardLanguageProperties[fileinfo.Go].Lang = &TheGoLang
	languages.ParserBytes[fileinfo.Go] = parserBytes
}

func (gl *GoLang) Parser() *parse.Parser {
	if gl.Pr != nil {
		return gl.Pr
	}
	lp, _ := parse.LanguageSupport.Properties(fileinfo.Go)
	if lp.Parser == nil {
		parse.LanguageSupport.OpenStandard()
	}
	gl.Pr = lp.Parser
	if gl.Pr == nil {
		return nil
	}
	return gl.Pr
}

// ParseFile is the main point of entry for external calls into the parser
func (gl *GoLang) ParseFile(fss *parse.FileStates, txt []byte) {
	pr := gl.Parser()
	if pr == nil {
		log.Println("ParseFile: no parser; must call parse.LangSupport.OpenStandard() at startup!")
		return
	}
	pfs := fss.StartProc(txt) // current processing one
	ext := filepath.Ext(pfs.Src.Filename)
	if ext == ".mod" { // note: mod doesn't parse!
		fss.EndProc()
		return
	}
	// fmt.Println("\nstarting Parse:", pfs.Src.Filename)
	// lprf := profile.Start("LexAll")
	pr.LexAll(pfs)
	// lprf.End()
	// pprf := profile.Start("ParseAll")
	pr.ParseAll(pfs)
	// pprf.End()
	fss.EndProc() // only symbols still need locking, done separately
	path := filepath.Dir(pfs.Src.Filename)
	// fmt.Println("done parse")
	if len(pfs.ParseState.Scopes) > 0 { // should be for complete files, not for snippets
		pkg := pfs.ParseState.Scopes[0]
		pfs.Syms[pkg.Name] = pkg // keep around..
		// fmt.Printf("main pkg name: %v\n", pkg.Name)
		pfs.WaitGp.Add(1)
		go func() {
			gl.AddPathToSyms(pfs, path)
			gl.AddImportsToExts(fss, pfs, pkg) // will do ResolveTypes when it finishes
			// fmt.Println("done import")
		}()
	} else {
		if TraceTypes {
			fmt.Printf("not importing scope for: %v\n", path)
		}
		pfs.ClearAST()
		if pfs.AST.HasChildren() {
			pfs.AST.DeleteChildren()
		}
		// fmt.Println("done no import")
	}
}

func (gl *GoLang) LexLine(fs *parse.FileState, line int, txt []rune) lexer.Line {
	pr := gl.Parser()
	if pr == nil {
		return nil
	}
	return pr.LexLine(fs, line, txt)
}

func (gl *GoLang) ParseLine(fs *parse.FileState, line int) *parse.FileState {
	pr := gl.Parser()
	if pr == nil {
		return nil
	}
	lfs := pr.ParseLine(fs, line) // should highlight same line?
	return lfs
}

func (gl *GoLang) HighlightLine(fss *parse.FileStates, line int, txt []rune) lexer.Line {
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
		merge := lexer.MergeLines(ll, cml)
		mc := merge.Clone()
		if len(cml) > 0 {
			initDepth := pfs.Src.PrevDepth(line)
			pr.PassTwo.NestDepthLine(mc, initDepth)
		}
		lfs.Syms.WriteDoc(os.Stdout, 0)
		lfs.Destroy()
		return mc
	}
	return ll
}

// IndentLine returns the indentation level for given line based on
// previous line's indentation level, and any delta change based on
// e.g., brackets starting or ending the previous or current line, or
// other language-specific keywords.  See lexer.BracketIndentLine for example.
// Indent level is in increments of tabSz for spaces, and tabs for tabs.
// Operates on rune source with markup lex tags per line.
func (gl *GoLang) IndentLine(fs *parse.FileStates, src [][]rune, tags []lexer.Line, ln int, tabSz int) (pInd, delInd, pLn int, ichr indent.Character) {
	pInd, pLn, ichr = lexer.PrevLineIndent(src, tags, ln, tabSz)

	curUnd, _ := lexer.LineStartEndBracket(src[ln], tags[ln])
	_, prvInd := lexer.LineStartEndBracket(src[pLn], tags[pLn])

	brackParen := false      // true if line only has bracket and paren -- outdent current
	if len(tags[pLn]) >= 2 { // allow for comments
		pl := tags[pLn][0]
		ll := tags[pLn][1]
		if ll.Token.Token == token.PunctGpRParen && pl.Token.Token == token.PunctGpRBrace {
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

	pwrd := lexer.FirstWord(string(src[pLn]))
	cwrd := lexer.FirstWord(string(src[ln]))

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
func (gl *GoLang) AutoBracket(fs *parse.FileStates, bra rune, pos textpos.Pos, curLn []rune) (match, newLine bool) {
	lnLen := len(curLn)
	if bra == '{' {
		if pos.Char == lnLen {
			if lnLen == 0 || unicode.IsSpace(curLn[pos.Char-1]) {
				newLine = true
			}
			match = true
		} else {
			match = unicode.IsSpace(curLn[pos.Char])
		}
	} else {
		match = pos.Char == lnLen || unicode.IsSpace(curLn[pos.Char]) // at end or if space after
	}
	return
}
