// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

//go:generate core generate -add-types

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/parse/parser"
	"cogentcore.org/core/text/textpos"
)

// Parser is the overall parser for managing the parsing
type Parser struct {

	// lexer rules for first pass of lexing file
	Lexer *lexer.Rule

	// second pass after lexing -- computes nesting depth and EOS finding
	PassTwo lexer.PassTwo

	// parser rules for parsing lexed tokens
	Parser *parser.Rule

	// file name for overall parser (not file being parsed!)
	Filename string

	// if true, reports errors after parsing, to stdout
	ReportErrs bool

	// when loaded from file, this is the modification time of the parser -- re-processes cache if parser is newer than cached files
	ModTime time.Time `json:"-" xml:"-"`
}

// Init initializes the parser -- must be called after creation
func (pr *Parser) Init() {
	pr.Lexer = lexer.NewRule()
	pr.Parser = parser.NewRule()
}

// NewParser returns a new initialized parser
func NewParser() *Parser {
	pr := &Parser{}
	pr.Init()
	return pr
}

// InitAll initializes everything about the parser -- call this when setting up a new
// parser after it has been loaded etc
func (pr *Parser) InitAll() {
	fs := &FileState{} // dummy, for error recording
	fs.Init()
	pr.Lexer.CompileAll(&fs.LexState)
	pr.Lexer.Validate(&fs.LexState)
	pr.Parser.CompileAll(&fs.ParseState)
	pr.Parser.Validate(&fs.ParseState)
}

// LexInit gets the lexer ready to start lexing
func (pr *Parser) LexInit(fs *FileState) {
	fs.LexState.Init()
	fs.LexState.Time.Now()
	fs.TwoState.Init()
	if fs.Src.NLines() > 0 {
		fs.LexState.SetLine(fs.Src.Lines[0])
	}
}

// LexNext does next step of lexing -- returns lowest-level rule that
// matched, and nil when nomatch err or at end of source input
func (pr *Parser) LexNext(fs *FileState) *lexer.Rule {
	if fs.LexState.Line >= fs.Src.NLines() {
		return nil
	}
	for {
		if fs.LexState.AtEol() {
			fs.Src.SetLine(fs.LexState.Line, fs.LexState.Lex, fs.LexState.Comments, fs.LexState.Stack)
			fs.LexState.Line++
			if fs.LexState.Line >= fs.Src.NLines() {
				return nil
			}
			fs.LexState.SetLine(fs.Src.Lines[fs.LexState.Line])
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
func (pr *Parser) LexNextLine(fs *FileState) *lexer.Rule {
	if fs.LexState.Line >= fs.Src.NLines() {
		return nil
	}
	var mrule *lexer.Rule
	for {
		if fs.LexState.AtEol() {
			fs.Src.SetLine(fs.LexState.Line, fs.LexState.Lex, fs.LexState.Comments, fs.LexState.Stack)
			fs.LexState.Line++
			if fs.LexState.Line >= fs.Src.NLines() {
				return nil
			}
			fs.LexState.SetLine(fs.Src.Lines[fs.LexState.Line])
			return mrule
		}
		mrule = pr.Lexer.LexStart(&fs.LexState)
		if mrule == nil {
			return nil
		}
	}
}

// LexRun keeps running LextNext until it stops
func (pr *Parser) LexRun(fs *FileState) {
	for {
		if pr.LexNext(fs) == nil {
			break
		}
	}
}

// LexLine runs lexer for given single line of source, which is updated
// from the given text (if non-nil)
// Returns merged regular and token comment lines, cloned and ready for use.
func (pr *Parser) LexLine(fs *FileState, ln int, txt []rune) lexer.Line {
	nlines := fs.Src.NLines()
	if ln >= nlines || ln < 0 {
		return nil
	}
	fs.Src.SetLineSrc(ln, txt)
	fs.LexState.SetLine(fs.Src.Lines[ln])
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
	fs.TwoState.SetSrc(&fs.Src)
	fs.Src.EosPos[ln] = nil // reset eos
	pr.PassTwo.EosDetectPos(&fs.TwoState, textpos.Pos{Line: ln}, 1)
	merge := lexer.MergeLines(fs.LexState.Lex, fs.LexState.Comments)
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
	// lprf := profile.Start("LexRun") // quite fast now..
	pr.LexRun(fs)
	// fs.LexErrReport()
	// lprf.End()
	pr.DoPassTwo(fs) // takes virtually no time
}

// ParserInit initializes the parser prior to running
func (pr *Parser) ParserInit(fs *FileState) bool {
	fs.AnonCtr = 0
	fs.ParseState.Init(&fs.Src, fs.AST)
	return true
}

// ParseNext does next step of parsing -- returns lowest-level rule that matched
// or nil if no match error or at end
func (pr *Parser) ParseNext(fs *FileState) *parser.Rule {
	mrule := pr.Parser.StartParse(&fs.ParseState)
	return mrule
}

// ParseRun continues running the parser until the end of the file
func (pr *Parser) ParseRun(fs *FileState) {
	for {
		pr.Parser.StartParse(&fs.ParseState)
		if fs.ParseState.AtEofNext() {
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
	if pr.ReportErrs {
		if fs.ParseHasErrs() {
			fmt.Println(fs.ParseErrReport())
		}
	}
}

// ParseLine runs parser for given single line of source
// does Parsing in a separate FileState and returns that with
// AST etc (or nil if nothing).  Assumes LexLine has already
// been run on given line.
func (pr *Parser) ParseLine(fs *FileState, ln int) *FileState {
	nlines := fs.Src.NLines()
	if ln >= nlines || ln < 0 {
		return nil
	}
	lfs := NewFileState()
	lfs.Src.InitFromLine(&fs.Src, ln)
	lfs.Src.EnsureFinalEos(0)
	lfs.ParseState.Init(&lfs.Src, lfs.AST)
	pr.ParseRun(lfs)
	return lfs
}

// ParseString runs lexer and parser on given string of text, returning
// FileState of results (can be nil if string is empty or no lexical tokens).
// Also takes supporting contextual info for file / language that this string
// is associated with (only for reference)
func (pr *Parser) ParseString(str string, fname string, sup fileinfo.Known) *FileState {
	if str == "" {
		return nil
	}
	lfs := NewFileState()
	lfs.Src.InitFromString(str, fname, sup)
	// lfs.ParseState.Trace.FullOn()
	// lfs.ParseSTate.Trace.StdOut()
	lfs.ParseState.Init(&lfs.Src, lfs.AST)
	pr.LexAll(lfs)
	lxs := lfs.Src.Lexs[0]
	if len(lxs) == 0 {
		return nil
	}
	lfs.Src.EnsureFinalEos(0)
	pr.ParseAll(lfs)
	return lfs
}

// ReadJSON opens lexer and parser rules from Bytes, in a standard JSON-formatted file
func (pr *Parser) ReadJSON(b []byte) error {
	err := json.Unmarshal(b, pr)
	return errors.Log(err)
}

// OpenJSON opens lexer and parser rules from the given filename, in a standard JSON-formatted file
func (pr *Parser) OpenJSON(filename string) error {
	err := jsonx.Open(pr, filename)
	return errors.Log(err)
}

// SaveJSON saves lexer and parser rules, in a standard JSON-formatted file
func (pr *Parser) SaveJSON(filename string) error {
	err := jsonx.Save(pr, filename)
	return errors.Log(err)
}

// SaveGrammar saves lexer and parser grammar rules to BNF-like .parsegrammar file
func (pr *Parser) SaveGrammar(filename string) error {
	ofl, err := os.Create(filename)
	if err != nil {
		return errors.Log(err)
	}
	fmt.Fprintf(ofl, "// %v Lexer\n\n", filename)
	pr.Lexer.WriteGrammar(ofl, 0)
	fmt.Fprintf(ofl, "\n\n///////////////////////////////////////////////////\n// %v Parser\n\n", filename)
	pr.Parser.WriteGrammar(ofl, 0)
	return ofl.Close()
}
