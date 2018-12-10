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
	"os"

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
	Lexer    lex.Rule    `desc:"lexer rules for first pass of lexing file"`
	PassTwo  lex.PassTwo `desc:"second pass after lexing -- computes nesting depth and EOS finding"`
	Parser   parse.Rule  `desc:"parser rules for parsing lexed tokens"`
	Filename string      `desc:"file name for overall parser"`
}

// Init initializes the parser -- must be called after creation
func (pr *Parser) Init() {
	pr.Lexer.InitName(&pr.Lexer, "Lexer")
	pr.Parser.InitName(&pr.Parser, "Parser")
}

// NewParser returns a new initialized parser
func NewParser() *Parser {
	pr := &Parser{}
	pr.Init()
	return pr
}

// InitAll initializes everything about the parser -- call this when setting up a new
// parser after it has been loaded etc
func (pr *Parser) InitAll(fs *FileState) {
	fs.Src.AllocLines()
	pr.LexInit(fs)
	pr.ParserInit(fs)
}

// LexInit gets the lexer ready to start lexing
func (pr *Parser) LexInit(fs *FileState) {
	fs.LexState.Init()
	fs.LexState.Time.Now()
	fs.TwoState.Init()
	if fs.Src.NLines() > 0 {
		fs.LexState.SetLine((*fs.Src.Lines)[0])
	}
}

// LexNext does next step of lexing -- returns lowest-level rule that
// matched, and nil when nomatch err or at end of source input
func (pr *Parser) LexNext(fs *FileState) *lex.Rule {
	if fs.LexState.Ln >= fs.Src.NLines() {
		return nil
	}
	for {
		if fs.LexState.AtEol() {
			fs.Src.SetLine(fs.LexState.Ln, fs.LexState.Lex, fs.LexState.Comments, fs.LexState.Stack)
			fs.LexState.Ln++
			if fs.LexState.Ln >= fs.Src.NLines() {
				return nil
			}
			fs.LexState.SetLine((*fs.Src.Lines)[fs.LexState.Ln])
		}
		mrule := pr.Lexer.LexStart(&fs.LexState)
		if mrule != nil {
			return mrule
		}
		if !fs.LexState.AtEol() { // err
			break
		}
	}
	return nil
}

// LexNextLine does next line of lexing -- returns lowest-level rule that
// matched at end, and nil when nomatch err or at end of source input
func (pr *Parser) LexNextLine(fs *FileState) *lex.Rule {
	if fs.LexState.Ln >= fs.Src.NLines() {
		return nil
	}
	var mrule *lex.Rule
	for {
		if fs.LexState.AtEol() {
			fs.Src.SetLine(fs.LexState.Ln, fs.LexState.Lex, fs.LexState.Comments, fs.LexState.Stack)
			fs.LexState.Ln++
			if fs.LexState.Ln >= fs.Src.NLines() {
				return nil
			}
			fs.LexState.SetLine((*fs.Src.Lines)[fs.LexState.Ln])
			return mrule
		}
		mrule = pr.Lexer.LexStart(&fs.LexState)
		if mrule == nil {
			return nil
		}
	}
	return mrule
}

// LexRun keeps running LextNext until it stops
func (pr *Parser) LexRun(fs *FileState) {
	for {
		if pr.LexNext(fs) == nil {
			break
		}
	}
}

// LexLine runs lexer for given single line of source, returns merged
// regular and token comment lines, cloned and ready for use
func (pr *Parser) LexLine(fs *FileState, ln int) lex.Line {
	nlines := fs.Src.NLines()
	if ln > nlines || ln < 0 {
		return nil
	}
	fs.LexState.SetLine((*fs.Src.Lines)[ln])
	pst := fs.Src.PrevStack(ln)
	fs.LexState.Stack = pst.Clone()
	for !fs.LexState.AtEol() {
		mrule := pr.Lexer.LexStart(&fs.LexState)
		if mrule == nil {
			break
		}
	}
	initDepth := fs.Src.PrevDepth(ln)
	pr.PassTwo.NestDepthLine(fs.LexState.Lex, initDepth)                         // important to set this one's depth
	fs.Src.SetLine(ln, fs.LexState.Lex, fs.LexState.Comments, fs.LexState.Stack) // before saving here
	merge := lex.MergeLines(fs.LexState.Lex, fs.LexState.Comments)
	mc := merge.Clone()
	if len(fs.LexState.Comments) > 0 {
		pr.PassTwo.NestDepthLine(mc, initDepth)
	}
	return mc
}

// DoPassTwo does the second pass after lexing
func (pr *Parser) DoPassTwo(fs *FileState) {
	fs.TwoState.SetSrc(&fs.Src)
	pr.PassTwo.NestDepth(&fs.TwoState)
	if pr.PassTwo.DoEos {
		pr.PassTwo.EosDetect(&fs.TwoState)
	}
}

// LexAll runs a complete pass of the lexer and pass two, on current state
func (pr *Parser) LexAll(fs *FileState) {
	pr.LexInit(fs)
	pr.LexRun(fs)
	pr.DoPassTwo(fs)
}

// ParserInit initializes the parser prior to running
func (pr *Parser) ParserInit(fs *FileState) bool {
	fs.ParseState.Init(&fs.Src, &fs.Ast, &fs.TwoState.EosPos)
	parse.Trace.Init()
	ok := pr.Parser.CompileAll(&fs.ParseState)
	if !ok {
		return false
	}
	ok = pr.Parser.Validate(&fs.ParseState)
	return ok
}

// ParseNext does next step of parsing -- returns lowest-level rule that matched
// or nil if no match error or at end
func (pr *Parser) ParseNext(fs *FileState) *parse.Rule {
	mrule := pr.Parser.StartParse(&fs.ParseState)
	return mrule
}

// ParseRun continues running the parser until nothing matches anymore
func (pr *Parser) ParseRun(fs *FileState) {
	for {
		mrule := pr.Parser.StartParse(&fs.ParseState)
		if mrule == nil {
			break
		}
	}
}

// ParseAll does full parsing, including ParseInit and ParseRun, assuming LexAll
// has been done already
func (pr *Parser) ParseAll(fs *FileState) {
	if !pr.ParserInit(fs) {
		return
	}
	pr.ParseRun(fs)
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

// SaveGrammar saves lexer and parser grammar rules to BNF-like .pig file
func (pr *Parser) SaveGrammar(filename string) error {
	ofl, err := os.Create(filename)
	if err != nil {
		log.Println(err)
		return err
	}
	fmt.Fprintf(ofl, "// %v Lexer\n\n", filename)
	pr.Lexer.WriteGrammar(ofl, 0)
	fmt.Fprintf(ofl, "\n\n///////////////////////////////////////////////////\n// %v Parser\n\n", filename)
	pr.Parser.WriteGrammar(ofl, 0)
	return ofl.Close()
}

/////////////////////////////////////////////////////////////////////////////
//	FileState

// FileState is the parsing state information for a given file
type FileState struct {
	Src        lex.File     `json:"-" xml:"-" desc:"the source to be parsed -- also holds the full lexed tokens"`
	LexState   lex.State    `json:"_" xml:"-" desc:"state for lexing"`
	TwoState   lex.TwoState `json:"-" xml:"-" desc:"state for second pass nesting depth and EOS matching"`
	ParseState parse.State  `json:"_" xml:"-" desc:"state for parsing"`
	Ast        parse.Ast    `json:"_" xml:"-" desc:"ast output tree from parsing"`
}

// Init initializes the file state
func (fs *FileState) Init() {
	fs.Ast.InitName(&fs.Ast, "Ast")
	fs.LexState.Init()
	fs.TwoState.Init()
	fs.ParseState.Init(&fs.Src, &fs.Ast, &fs.TwoState.EosPos)
}

// NewFileState returns a new initialized file state
func NewFileState() *FileState {
	fs := &FileState{}
	fs.Init()
	return fs
}

// SetSrc sets source to be parsed, and filename it came from, and also the
// base path for project for reporting filenames relative to
// (if empty, path to filename is used)
func (fs *FileState) SetSrc(src *[][]rune, fname string) {
	fs.Init()
	fs.Src.SetSrc(src, fname)
	fs.LexState.Filename = fname
}

// LexAtEnd returns true if lexing state is now at end of source
func (fs *FileState) LexAtEnd() bool {
	return fs.LexState.Ln >= fs.Src.NLines()
}

// LexLine returns the lexing output for given line, combining comments and all other tokens
// and allocating new memory using clone
func (fs *FileState) LexLine(ln int) lex.Line {
	return fs.Src.LexLine(ln)
}

// LexLineString returns a string rep of the current lexing output for the current line
func (fs *FileState) LexLineString() string {
	return fs.LexState.LineString()
}

// LexNextSrcLine returns the next line of source that the lexer is currently at
func (fs *FileState) LexNextSrcLine() string {
	return fs.LexState.NextSrcLine()
}

// LexHasErrs returns true if there were errors from lexing
func (fs *FileState) LexHasErrs() bool {
	return len(fs.LexState.Errs) > 0
}

// LexErrReport returns a report of all the lexing errors -- these should only
// occur during development of lexer so we use a detailed report format
func (fs *FileState) LexErrReport() string {
	return fs.LexState.Errs.Report(0, fs.Src.BasePath, true, true)
}

// PassTwoHasErrs returns true if there were errors from pass two processing
func (fs *FileState) PassTwoHasErrs() bool {
	return len(fs.TwoState.Errs) > 0
}

// PassTwoErrString returns all the pass two errors as a string -- these should
// only occur during development so we use a detailed report format
func (fs *FileState) PassTwoErrReport() string {
	return fs.TwoState.Errs.Report(0, fs.Src.BasePath, true, true)
}

// ParseAtEnd returns true if parsing state is now at end of source
func (fs *FileState) ParseAtEnd() bool {
	return fs.ParseState.AtEof()
}

// ParseNextSrcLine returns the next line of source that the parser is currently at
func (fs *FileState) ParseNextSrcLine() string {
	return fs.ParseState.NextSrcLine()
}

// ParseHasErrs returns true if there were errors from parsing
func (fs *FileState) ParseHasErrs() bool {
	return len(fs.ParseState.Errs) > 0
}

// ParseErrReport returns at most 10 parsing errors in end-user format, sorted
func (fs *FileState) ParseErrReport() string {
	fs.ParseState.Errs.Sort()
	return fs.ParseState.Errs.Report(10, fs.Src.BasePath, true, false)
}

// ParseErrReportDetailed returns at most 10 parsing errors in detailed format, sorted
func (fs *FileState) ParseErrReportDetailed() string {
	fs.ParseState.Errs.Sort()
	return fs.ParseState.Errs.Report(10, fs.Src.BasePath, true, true)
}

// RuleString returns the rule info for entire source -- if full
// then it includes the full stack at each point -- otherwise just the top
// of stack
func (fs *FileState) ParseRuleString(full bool) string {
	return fs.ParseState.RuleString(full)
}
