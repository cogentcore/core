// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pi

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/goki/pi/filecat"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/parse"
	"github.com/goki/pi/syms"
)

// FileState contains the full lexing and parsing state information for a given file.
// It is the master state record for everything that happens in GoPi.  One of these
// should be maintained for each file -- giv.TextBuf has one as PiState field.
//
// Separate State structs are maintained for each stage (Lexing, PassTwo, Parsing) and
// the final output of Parsing goes into the Ast and Syms fields.
//
// The Src lex.File field maintains all the info about the source file, and the basic
// tokenized version of the source produced initially by lexing and updated by the
// remaining passes.  It has everything that is maintained at a line-by-line level.
//
type FileState struct {
	Src        lex.File       `json:"-" xml:"-" desc:"the source to be parsed -- also holds the full lexed tokens"`
	LexState   lex.State      `json:"_" xml:"-" desc:"state for lexing"`
	TwoState   lex.TwoState   `json:"-" xml:"-" desc:"state for second pass nesting depth and EOS matching"`
	ParseState parse.State    `json:"-" xml:"-" desc:"state for parsing"`
	Ast        parse.Ast      `json:"-" xml:"-" desc:"ast output tree from parsing"`
	Syms       syms.SymMap    `json:"-" xml:"-" desc:"symbols contained within this file -- initialized at start of parsing and created by AddSymbol or PushNewScope actions.  These are then processed after parsing by the language-specific code, via Lang interface."`
	ExtSyms    syms.SymMap    `json:"-" xml:"-" desc:"External symbols that are entirely maintained in a language-specific way by the Lang interface code.  These are only here as a convenience and are not accessed in any way by the language-general pi code."`
	SymsMu     sync.RWMutex   `view:"-" json:"-" xml:"-" desc:"mutex protecting updates / reading of Syms symbols"`
	WaitGp     sync.WaitGroup `view:"-" json:"-" xml:"-" desc:"waitgroup for coordinating processing of other items"`
	AnonCtr    int            `view:"-" json:"-" xml:"-" desc:"anonymous counter -- counts up "`
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
	fs.AnonCtr = 0
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
func (fs *FileState) SetSrc(src *[][]rune, fname, basepath string, sup filecat.Supported) {
	fs.Init()
	fs.Src.SetSrc(src, fname, basepath, sup)
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

////////////////////////////////////////////////////////////////////////
//  Syms symbol processing support

// FindNameScoped looks for given symbol name within given map first
// (if non nil) and then in fs.Syms and ExtSyms maps,
// and any children on those global maps that are of subcategory
// token.NameScope (i.e., namespace, module, package, library)
func (fs *FileState) FindNameScoped(nm string, scope syms.SymMap) (*syms.Symbol, bool) {
	var sy *syms.Symbol
	has := false
	if scope != nil {
		sy, has = scope.FindName(nm)
		if has {
			return sy, true
		}
	}
	sy, has = fs.Syms.FindNameScoped(nm)
	if has {
		return sy, true
	}
	sy, has = fs.ExtSyms.FindNameScoped(nm)
	if has {
		return sy, true
	}
	return nil, false
}

// FindChildren fills out map with direct children of given symbol
// If seed is non-empty it is used as a prefix for filtering children names.
// Returns false if no children were found.
func (fs *FileState) FindChildren(sym *syms.Symbol, seed string, scope syms.SymMap, kids *syms.SymMap) bool {
	if len(sym.Children) == 0 {
		if sym.Type != "" {
			typ, got := fs.FindNameScoped(sym.NonPtrTypeName(), scope)
			if got {
				sym = typ
			} else {
				return false
			}
		}
	}
	if seed != "" {
		sym.Children.FindNamePrefix(seed, kids)
	} else {
		kids.CopyFrom(sym.Children)
	}
	return len(*kids) > 0
}

// FindAnyChildren fills out map with either direct children of given symbol
// or those of the type of this symbol -- useful for completion.
// If seed is non-empty it is used as a prefix for filtering children names.
// Returns false if no children were found.
func (fs *FileState) FindAnyChildren(sym *syms.Symbol, seed string, scope syms.SymMap, kids *syms.SymMap) bool {
	if len(sym.Children) == 0 {
		if sym.Type != "" {
			typ, got := fs.FindNameScoped(sym.NonPtrTypeName(), scope)
			if got {
				sym = typ
			} else {
				return false
			}
		}
	}
	if seed != "" {
		sym.Children.FindNamePrefixRecursive(seed, kids)
	} else {
		kids.CopyFrom(sym.Children)
	}
	return len(*kids) > 0
}

// FindNamePrefixScoped looks for given symbol name prefix within given map first
// (if non nil) and then in fs.Syms and ExtSyms maps,
// and any children on those global maps that are of subcategory
// token.NameScope (i.e., namespace, module, package, library)
// adds to given matches map (which can be nil), for more efficient recursive use
func (fs *FileState) FindNamePrefixScoped(seed string, scope syms.SymMap, matches *syms.SymMap) {
	lm := len(*matches)
	if scope != nil {
		scope.FindNamePrefixRecursive(seed, matches)
	}
	if len(*matches) != lm {
		return
	}
	fs.Syms.FindNamePrefixScoped(seed, matches)
	if len(*matches) != lm {
		return
	}
	fs.ExtSyms.FindNamePrefixScoped(seed, matches)
}

// NextAnonName returns the next anonymous name for this file, using counter here
// and given context name (e.g., package name)
func (fs *FileState) NextAnonName(ctxt string) string {
	fs.AnonCtr++
	fn := filepath.Base(fs.Src.Filename)
	ext := filepath.Ext(fn)
	if ext != "" {
		fn = strings.TrimSuffix(fn, ext)
	}
	return fmt.Sprintf("anon_%s_%d", ctxt, fs.AnonCtr)
}
