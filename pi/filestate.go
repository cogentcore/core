// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pi

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/goki/gi/filecat"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/parse"
	"github.com/goki/pi/syms"
)

// FileState is the parsing state information for a given file
type FileState struct {
	Src        lex.File     `json:"-" xml:"-" desc:"the source to be parsed -- also holds the full lexed tokens"`
	LexState   lex.State    `json:"_" xml:"-" desc:"state for lexing"`
	TwoState   lex.TwoState `json:"-" xml:"-" desc:"state for second pass nesting depth and EOS matching"`
	ParseState parse.State  `json:"-" xml:"-" desc:"state for parsing"`
	Ast        parse.Ast    `json:"-" xml:"-" desc:"ast output tree from parsing"`
	Syms       syms.SymMap  `json:"-" xml:"-" desc:"aggregate symbols for this file -- the language is responsible for managing these symbols to contain those relevant for the given file, and these are used for lookup (again managed through the Lang interface)"`
	SymsMu     sync.RWMutex `json:"-" xml:"-" desc:"mutex protecting updates / reading of Syms symbols"`
}

// Init initializes the file state
func (fs *FileState) Init() {
	fs.Ast.InitName(&fs.Ast, "Ast")
	fs.LexState.Init()
	fs.TwoState.Init()
	fs.ParseState.Init(&fs.Src, &fs.Ast)
	fs.SymsMu.Lock()
	fs.Syms = make(syms.SymMap)
	fs.SymsMu.Unlock()
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
func (fs *FileState) SetSrc(src *[][]rune, fname string, sup filecat.Supported) {
	fs.Init()
	fs.Src.SetSrc(src, fname, sup)
	fs.LexState.Filename = fname
}

// RunesFromBytes returns the lines of runes from a basic byte array
func (fs *FileState) RunesFromBytes(b []byte) [][]rune {
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
func (fs *FileState) RunesFromString(str string) [][]rune {
	lns := strings.Split(str, "\n")
	nlines := len(lns)
	rns := make([][]rune, nlines)
	for ln, txt := range lns {
		rns[ln] = []rune(txt)
	}
	return rns
}

// OpenFile sets source to be parsed from given filename
func (fs *FileState) OpenFile(fname string) error {
	fp, err := os.Open(fname)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	alltxt, err := ioutil.ReadAll(fp)
	fp.Close()
	if err != nil {
		log.Println(err.Error())
		return err
	}
	rns := fs.RunesFromBytes(alltxt)
	sup := filecat.SupportedFromFile(fname)
	fs.SetSrc(&rns, fname, sup)
	return nil
}

// InitFromLine initializes from one line of source fs
func (fs *FileState) InitFromLine(sfs *FileState, ln int) bool {
	nlines := sfs.Src.NLines()
	if ln > nlines || ln < 0 {
		return false
	}
	src := [][]rune{(*sfs.Src.Lines)[ln], []rune{}} // need extra blank
	fs.SetSrc(&src, sfs.Src.Filename, sfs.Src.Sup)
	fs.Src.Lexs = []lex.Line{sfs.Src.Lexs[ln], lex.Line{}}
	fs.Src.Comments = []lex.Line{sfs.Src.Comments[ln], lex.Line{}}
	fs.Src.EosPos = []lex.EosPos{sfs.Src.EosPos[ln], lex.EosPos{}}
	return true
}

// InitFromString initializes from given string
// returns false if string is empty
func (fs *FileState) InitFromString(str string, fname string, sup filecat.Supported) bool {
	if str == "" {
		return false
	}
	src := fs.RunesFromString(str)
	if len(src) == 1 { // need more than 1 line
		src = append(src, []rune{})
	}
	fs.SetSrc(&src, fname, sup)
	return true
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
	return fs.ParseState.AtEofNext()
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

// ParseErrReportAll returns all parsing errors in end-user format, sorted
func (fs *FileState) ParseErrReportAll() string {
	fs.ParseState.Errs.Sort()
	return fs.ParseState.Errs.Report(0, fs.Src.BasePath, true, false)
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
