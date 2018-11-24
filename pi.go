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
	Lexer    lex.Rule   `desc:"lexer rules for first pass of lexing file"`
	Parser   parse.Rule `desc:"parser rules for second pass of parsing lexed tokens"`
	Src      lex.File   `json:"-" xml:"-" desc:"the source to be parsed"`
	LexState lex.State  `json:"_" xml:"-" desc:"state for lexing"`
	Ast      parse.Ast  `json:"_" xml:"-" desc:"abstract syntax tree output from parsing"`
	Filename string     `desc:"file name for overall parser"`
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
	pr.Src.SetSrc(src)
	pr.LexState.Init()
	pr.LexState.Filename = fname
	pr.LexState.SetLine(src[0])
	pr.Lexer.Validate()
}

// LexAtEnd returns true if lexing state is now at end of source
func (pr *Parser) LexAtEnd() bool {
	return pr.LexState.Ln >= pr.Src.NLines()
}

// LexNext does next step of lexing -- returns lowest-level rule that
// matched, and nil none or at end of source input
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
