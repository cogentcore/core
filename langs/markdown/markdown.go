// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package markdown

import (
	_ "embed"
	"strings"
	"unicode"

	"goki.dev/glop/indent"
	"goki.dev/pi/v2/complete"
	"goki.dev/pi/v2/filecat"
	"goki.dev/pi/v2/langs"
	"goki.dev/pi/v2/langs/bibtex"
	"goki.dev/pi/v2/lex"
	"goki.dev/pi/v2/pi"
	"goki.dev/pi/v2/syms"
	"goki.dev/pi/v2/token"
)

//go:embed markdown.pi
var parserBytes []byte

// MarkdownLang implements the Lang interface for the Markdown language
type MarkdownLang struct {
	Pr *pi.Parser

	// bibliography files that have been loaded, keyed by file path from bibfile metadata stored in filestate
	Bibs bibtex.Files `desc:"bibliography files that have been loaded, keyed by file path from bibfile metadata stored in filestate"`
}

// TheMarkdownLang is the instance variable providing support for the Markdown language
var TheMarkdownLang = MarkdownLang{}

func init() {
	pi.StdLangProps[filecat.Markdown].Lang = &TheMarkdownLang
	langs.ParserBytes[filecat.Markdown] = parserBytes
}

func (ml *MarkdownLang) Parser() *pi.Parser {
	if ml.Pr != nil {
		return ml.Pr
	}
	lp, _ := pi.LangSupport.Props(filecat.Markdown)
	if lp.Parser == nil {
		pi.LangSupport.OpenStd()
	}
	ml.Pr = lp.Parser
	if ml.Pr == nil {
		return nil
	}
	return ml.Pr
}

func (ml *MarkdownLang) ParseFile(fss *pi.FileStates, txt []byte) {
	pr := ml.Parser()
	if pr == nil {
		return
	}
	pfs := fss.StartProc(txt) // current processing one
	pr.LexAll(pfs)
	ml.OpenBibfile(fss, pfs)
	fss.EndProc() // now done
	// no parser
}

func (ml *MarkdownLang) LexLine(fs *pi.FileState, line int, txt []rune) lex.Line {
	pr := ml.Parser()
	if pr == nil {
		return nil
	}
	return pr.LexLine(fs, line, txt)
}

func (ml *MarkdownLang) ParseLine(fs *pi.FileState, line int) *pi.FileState {
	// n/a
	return nil
}

func (ml *MarkdownLang) HiLine(fss *pi.FileStates, line int, txt []rune) lex.Line {
	fs := fss.Done()
	return ml.LexLine(fs, line, txt)
}

func (ml *MarkdownLang) CompleteLine(fss *pi.FileStates, str string, pos lex.Pos) (md complete.Matches) {
	origStr := str
	lfld := lex.LastField(str)
	str = lex.InnerBracketScope(lfld, "[", "]")
	if len(str) > 1 {
		if str[0] == '@' {
			return ml.CompleteCite(fss, origStr, str[1:], pos)
		}
	}
	// n/a
	return md
}

// Lookup is the main api called by completion code in giv/complete.go to lookup item
func (ml *MarkdownLang) Lookup(fss *pi.FileStates, str string, pos lex.Pos) (ld complete.Lookup) {
	origStr := str
	lfld := lex.LastField(str)
	str = lex.InnerBracketScope(lfld, "[", "]")
	if len(str) > 1 {
		if str[0] == '@' {
			return ml.LookupCite(fss, origStr, str[1:], pos)
		}
	}
	return
}

func (ml *MarkdownLang) CompleteEdit(fs *pi.FileStates, text string, cp int, comp complete.Completion, seed string) (ed complete.Edit) {
	// if the original is ChildByName() and the cursor is between d and B and the comp is Children,
	// then delete the portion after "Child" and return the new comp and the number or runes past
	// the cursor to delete
	s2 := text[cp:]
	// gotParen := false
	if len(s2) > 0 && lex.IsLetterOrDigit(rune(s2[0])) {
		for i, c := range s2 {
			if c == '{' {
				// gotParen = true
				s2 = s2[:i]
				break
			}
			isalnum := c == '_' || unicode.IsLetter(c) || unicode.IsDigit(c)
			if !isalnum {
				s2 = s2[:i]
				break
			}
		}
	} else {
		s2 = ""
	}

	var nw = comp.Text
	// if gotParen && strings.HasSuffix(nw, "()") {
	// 	nw = nw[:len(nw)-2]
	// }

	// fmt.Printf("text: %v|%v  comp: %v  s2: %v\n", text[:cp], text[cp:], nw, s2)
	ed.NewText = nw
	ed.ForwardDelete = len(s2)
	return ed
}

func (ml *MarkdownLang) ParseDir(fs *pi.FileState, path string, opts pi.LangDirOpts) *syms.Symbol {
	// n/a
	return nil
}

// List keywords (for indent)
var ListKeys = map[string]struct{}{"*": {}, "+": {}, "-": {}}

// IndentLine returns the indentation level for given line based on
// previous line's indentation level, and any delta change based on
// e.g., brackets starting or ending the previous or current line, or
// other language-specific keywords.  See lex.BracketIndentLine for example.
// Indent level is in increments of tabSz for spaces, and tabs for tabs.
// Operates on rune source with markup lex tags per line.
func (ml *MarkdownLang) IndentLine(fs *pi.FileStates, src [][]rune, tags []lex.Line, ln int, tabSz int) (pInd, delInd, pLn int, ichr indent.Char) {
	pInd, pLn, ichr = lex.PrevLineIndent(src, tags, ln, tabSz)
	delInd = 0
	ptg := tags[pLn]
	ctg := tags[ln]
	if len(ptg) == 0 || len(ctg) == 0 {
		return
	}
	fpt := ptg[0]
	fct := ctg[0]
	if fpt.Tok.Tok != token.Keyword || fct.Tok.Tok != token.Keyword {
		return
	}
	pk := strings.TrimSpace(string(fpt.Src(src[pLn])))
	ck := strings.TrimSpace(string(fct.Src(src[ln])))
	// fmt.Printf("pk: %v  ck: %v\n", string(pk), string(ck))
	if len(pk) >= 1 && len(ck) >= 1 {
		_, pky := ListKeys[pk]
		_, cky := ListKeys[ck]
		if unicode.IsDigit(rune(pk[0])) {
			pk = "1"
			pky = true
		}
		if unicode.IsDigit(rune(ck[0])) {
			ck = "1"
			cky = true
		}
		if pky && cky {
			if pk != ck {
				delInd = 1
				return
			}
			return
		}
	}
	return
}

// AutoBracket returns what to do when a user types a starting bracket character
// (bracket, brace, paren) while typing.
// pos = position where bra will be inserted, and curLn is the current line
// match = insert the matching ket, and newLine = insert a new line.
func (ml *MarkdownLang) AutoBracket(fs *pi.FileStates, bra rune, pos lex.Pos, curLn []rune) (match, newLine bool) {
	lnLen := len(curLn)
	match = pos.Ch == lnLen || unicode.IsSpace(curLn[pos.Ch]) // at end or if space after
	newLine = false
	return
}
