// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pi provides the main interactive parser structure for running the parse
// The piv sub-package provides the GUI for constructing and testing a parser
package pi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/goki/pi/lex"
	"github.com/goki/pi/parse"
)

// VersionInfo returns Pi version information
func VersionInfo() string {
	vinfo := Version + " date: " + VersionDate + " UTC; git commit-1: " + GitCommit
	return vinfo
}

// Parser is the overall parser for managing the parsing
type Parser struct {
	Lexer      lex.Rule    `desc:"lexer rules for first pass of lexing file"`
	Eoser      parse.Eoser `desc:"end-of-statement finder -- step after lexing before parsing"`
	Parser     parse.Rule  `desc:"parser rules for second pass of parsing lexed tokens"`
	Src        lex.File    `json:"-" xml:"-" desc:"the source to be parsed -- also holds the full lexed tokens"`
	LexState   lex.State   `json:"_" xml:"-" desc:"state for lexing"`
	ParseState parse.State `json:"_" xml:"-" desc:"state for parsing"`
	Ast        parse.Ast   `json:"_" xml:"-" desc:"ast output tree from parsing"`
	Filename   string      `desc:"file name for overall parser"`
}

func (pr *Parser) Init() {
	pr.Lexer.InitName(&pr.Lexer, "Lexer")
	pr.Parser.InitName(&pr.Parser, "Parser")
	pr.Ast.InitName(&pr.Ast, "Ast")
	pr.LexState.Init()
}

// SetSrc sets source to be parsed, and filename it came from
func (pr *Parser) SetSrc(src [][]rune, fname string) {
	if len(src) == 0 {
		pr.Init()
		return
	}
	pr.Src.SetSrc(src, fname)
	pr.LexState.Init()
	pr.LexState.Filename = fname
	pr.LexState.SetLine(src[0])
	pr.Lexer.Validate(&pr.LexState)
	pr.ParseState.Init(&pr.Src, &pr.Ast, pr.Eoser.State.EosPos)
}

// LexAtEnd returns true if lexing state is now at end of source
func (pr *Parser) LexAtEnd() bool {
	return pr.LexState.Ln >= pr.Src.NLines()
}

// LexNext does next step of lexing -- returns lowest-level rule that
// matched, and nil when nomatch err or at end of source input
func (pr *Parser) LexNext() *lex.Rule {
	if pr.LexState.Ln >= pr.Src.NLines() {
		return nil
	}
	for {
		if pr.LexState.AtEol() {
			pr.Src.SetLexs(pr.LexState.Ln, pr.LexState.Lex)
			pr.LexState.Ln++
			if pr.LexState.Ln >= pr.Src.NLines() {
				return nil
			}
			pr.LexState.SetLine(pr.Src.Lines[pr.LexState.Ln])
		}
		cpos := pr.LexState.Pos
		rval := pr.Lexer.Lex(&pr.LexState)
		if !pr.LexState.AtEol() && cpos == pr.LexState.Pos {
			msg := fmt.Sprintf("did not advance position -- need more rules to match current input: %v", string(pr.LexState.Src[cpos:]))
			pr.LexState.Error(cpos, msg)
			return nil
		}
		if rval != nil {
			return rval
		}
	}
}

// LexAll does all the lexing
func (pr *Parser) LexAll() {
	for {
		if pr.LexNext() == nil {
			break
		}
	}
}

// LexLineOut returns the current lexing output for the current line
func (pr *Parser) LexLineOut() string {
	return pr.LexState.LineOut()
}

// LexHasErrs returns true if there were errors from lexing
func (pr *Parser) LexHasErrs() bool {
	return len(pr.LexState.Errs) > 0
}

// LexErrString returns all the lexing errors as a string
func (pr *Parser) LexErrString() string {
	return pr.LexState.Errs.AllString()
}

// Eosify does all the finding of EOS end-of-statements, if supported for this grammar
func (pr *Parser) Eosify() {
	if pr.Eoser.Do {
		pr.Eoser.Eosify(&pr.Src)
	}
}

// EoserHasErrs returns true if there were errors from eosifying
func (pr *Parser) EoserHasErrs() bool {
	return len(pr.Eoser.State.Errs) > 0
}

// EoserErrString returns all the eoser errors as a string
func (pr *Parser) EoserErrString() string {
	return pr.Eoser.State.Errs.AllString()
}

// ParserInit initializes the parser prior to running
func (pr *Parser) ParserInit() bool {
	pr.ParseState.Init(&pr.Src, &pr.Ast, pr.Eoser.State.EosPos)
	ok := pr.Parser.CompileAll(&pr.ParseState)
	if !ok {
		return false
	}
	ok = pr.Parser.Validate(&pr.ParseState)
	return ok
}

// ParseNext does next step of parsing -- returns lowest-level rule that matched
// or nil if no match error or at end
func (pr *Parser) ParseNext() *parse.Rule {
	mrule := pr.Parser.Parse(&pr.ParseState, nil, nil)
	return mrule
}

// ParseAtEnd returns true if parsing state is now at end of source
func (pr *Parser) ParseAtEnd() bool {
	return pr.ParseState.AtEof()
}

// ParseHasErrs returns true if there were errors from parsing
func (pr *Parser) ParseHasErrs() bool {
	return len(pr.ParseState.Errs) > 0
}

// ParseErrString returns all the parsing errors as a string
func (pr *Parser) ParseErrString() string {
	return pr.ParseState.Errs.AllString()
}

// OpenJSON opens lexer and parser rules to current filename, in a standard JSON-formatted file
func (pr *Parser) OpenJSON(filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, pr)
	if err == nil {
		pr.Lexer.UnmarshalPost()
		pr.Parser.UnmarshalPost()
	}
	return err
}

// SaveJSON saves lexer and parser rules, in a standard JSON-formatted file
func (pr *Parser) SaveJSON(filename string) error {
	b, err := json.MarshalIndent(pr, "", "  ")
	if err != nil {
		log.Println(err) // unlikely
		return err
	}
	err = ioutil.WriteFile(filename, b, 0644)
	if err != nil {
		log.Println(err)
	}
	return err
}
