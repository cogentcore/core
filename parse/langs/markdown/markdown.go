// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package markdown

import (
	_ "embed"
	"strings"
	"unicode"

	"cogentcore.org/core/fileinfo"
	"cogentcore.org/core/gox/indent"
	"cogentcore.org/core/parse"
	"cogentcore.org/core/parse/complete"
	"cogentcore.org/core/parse/langs"
	"cogentcore.org/core/parse/langs/bibtex"
	"cogentcore.org/core/parse/lexer"
	"cogentcore.org/core/parse/syms"
	"cogentcore.org/core/parse/token"
)

//go:embed markdown.parse
var parserBytes []byte

// MarkdownLang implements the Lang interface for the Markdown language
type MarkdownLang struct {
	Pr *parse.Parser

	// bibliography files that have been loaded, keyed by file path from bibfile metadata stored in filestate
	Bibs bibtex.Files
}

// TheMarkdownLang is the instance variable providing support for the Markdown language
var TheMarkdownLang = MarkdownLang{}

func init() {
	parse.StandardLangProperties[fileinfo.Markdown].Lang = &TheMarkdownLang
	langs.ParserBytes[fileinfo.Markdown] = parserBytes
}

func (ml *MarkdownLang) Parser() *parse.Parser {
	if ml.Pr != nil {
		return ml.Pr
	}
	lp, _ := parse.LangSupport.Properties(fileinfo.Markdown)
	if lp.Parser == nil {
		parse.LangSupport.OpenStandard()
	}
	ml.Pr = lp.Parser
	if ml.Pr == nil {
		return nil
	}
	return ml.Pr
}

func (ml *MarkdownLang) ParseFile(fss *parse.FileStates, txt []byte) {
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

func (ml *MarkdownLang) LexLine(fs *parse.FileState, line int, txt []rune) lexer.Line {
	pr := ml.Parser()
	if pr == nil {
		return nil
	}
	return pr.LexLine(fs, line, txt)
}

func (ml *MarkdownLang) ParseLine(fs *parse.FileState, line int) *parse.FileState {
	// n/a
	return nil
}

func (ml *MarkdownLang) HiLine(fss *parse.FileStates, line int, txt []rune) lexer.Line {
	fs := fss.Done()
	return ml.LexLine(fs, line, txt)
}

func (ml *MarkdownLang) CompleteLine(fss *parse.FileStates, str string, pos lexer.Pos) (md complete.Matches) {
	origStr := str
	lfld := lexer.LastField(str)
	str = lexer.InnerBracketScope(lfld, "[", "]")
	if len(str) > 1 {
		if str[0] == '@' {
			return ml.CompleteCite(fss, origStr, str[1:], pos)
		}
	}
	// n/a
	return md
}

// Lookup is the main api called by completion code in giv/complete.go to lookup item
func (ml *MarkdownLang) Lookup(fss *parse.FileStates, str string, pos lexer.Pos) (ld complete.Lookup) {
	origStr := str
	lfld := lexer.LastField(str)
	str = lexer.InnerBracketScope(lfld, "[", "]")
	if len(str) > 1 {
		if str[0] == '@' {
			return ml.LookupCite(fss, origStr, str[1:], pos)
		}
	}
	return
}

func (ml *MarkdownLang) CompleteEdit(fs *parse.FileStates, text string, cp int, comp complete.Completion, seed string) (ed complete.Edit) {
	// if the original is ChildByName() and the cursor is between d and B and the comp is Children,
	// then delete the portion after "Child" and return the new comp and the number or runes past
	// the cursor to delete
	s2 := text[cp:]
	// gotParen := false
	if len(s2) > 0 && lexer.IsLetterOrDigit(rune(s2[0])) {
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

func (ml *MarkdownLang) ParseDir(fs *parse.FileState, path string, opts parse.LangDirOpts) *syms.Symbol {
	// n/a
	return nil
}

// List keywords (for indent)
var ListKeys = map[string]struct{}{"*": {}, "+": {}, "-": {}}

// IndentLine returns the indentation level for given line based on
// previous line's indentation level, and any delta change based on
// e.g., brackets starting or ending the previous or current line, or
// other language-specific keywords.  See lexer.BracketIndentLine for example.
// Indent level is in increments of tabSz for spaces, and tabs for tabs.
// Operates on rune source with markup lex tags per line.
func (ml *MarkdownLang) IndentLine(fs *parse.FileStates, src [][]rune, tags []lexer.Line, ln int, tabSz int) (pInd, delInd, pLn int, ichr indent.Char) {
	pInd, pLn, ichr = lexer.PrevLineIndent(src, tags, ln, tabSz)
	delInd = 0
	ptg := tags[pLn]
	ctg := tags[ln]
	if len(ptg) == 0 || len(ctg) == 0 {
		return
	}
	fpt := ptg[0]
	fct := ctg[0]
	if fpt.Token.Token != token.Keyword || fct.Token.Token != token.Keyword {
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
func (ml *MarkdownLang) AutoBracket(fs *parse.FileStates, bra rune, pos lexer.Pos, curLn []rune) (match, newLine bool) {
	lnLen := len(curLn)
	match = pos.Ch == lnLen || unicode.IsSpace(curLn[pos.Ch]) // at end or if space after
	newLine = false
	return
}
