// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pi

import (
	"github.com/goki/gi/complete"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/syms"
)

// Lang provides a general interface for language-specific management
// of the lexing, parsing, and symbol lookup process.
// The GoPi lexer and parser machinery is entirely language-general
// but specific languages may need specific ways of managing these
// processes, and processing their outputs, to best support the
// features of those languages.  That is what this interface provides.
//
// Each language defines a type supporting this interface, which is
// in turn registered with the StdLangProps map.  Each supported
// language has its own .go file in this pi package that defines its
// own implementation of the interface and any other associated
// functionality.
//
// The Lang is responsible for accessing the appropriate pi.Parser for this
// language (initialized and managed via LangSupport.OpenStd() etc)
// and the pi.FileState structure contains all the input and output
// state information for a given file.
//
// This interface is likely to evolve as we expand the range of supported
// languages.
type Lang interface {
	// Parser returns the pi.Parser for this language
	Parser() *Parser

	// ParseFile does the complete processing of a given single file, as appropriate
	// for the language -- e.g., runs the lexer followed by the parser, and
	// manages any symbol output from parsing as appropriate for the language / format.
	ParseFile(fs *FileState)

	// LexLine does just the lexing of a given line of the file, using existing context
	// if available from prior lexing / parsing. Line is in 0-indexed "internal" line indexes.
	// The rune source information is assumed to have already been updated in FileState.
	// languages can run the parser on the line to augment the lex token output as appropriate.
	LexLine(fs *FileState, line int) lex.Line

	// ParseLine does complete parser processing of a single line from given file, and returns
	// the FileState for just that line.  Line is in 0-indexed "internal" line indexes.
	// The rune source information is assumed to have already been updated in FileState
	// Existing context information from full-file parsing is used as appropriate, but
	// the results will NOT be used to update any existing full-file Ast representation --
	// should call ParseFile to update that as appropriate.
	ParseLine(fs *FileState, line int) *FileState

	// HiLine does the lexing and potentially parsing of a given line of the file,
	// for purposes of syntax highlighting -- uses existing context
	// if available from prior lexing / parsing. Line is in 0-indexed "internal" line indexes.
	// The rune source information is assumed to have already been updated in FileState.
	// languages can run the parser on the line to augment the lex token output as appropriate.
	HiLine(fs *FileState, line int) lex.Line

	// CompleteLine provides the list of relevant completions for given text
	// which is at given position within the file.
	// Typically the language will call ParseLine on that line, and use the Ast
	// to guide the selection of relevant symbols that can complete the code at
	// the given point.  A stack (slice) of symbols is returned so that the completer
	// can control the order of items presented, as compared to the SymMap.
	CompleteLine(fs *FileState, text string, pos lex.Pos) complete.MatchData

	// CompleteEdit returns the completion edit data for integrating the selected completion
	// into the source
	CompleteEdit(fs *FileState, text string, cp int, comp complete.Completion, seed string) (ed complete.EditData)

	// ParseDir does the complete processing of a given directory, optionally including
	// subdirectories, and optionally forcing the re-processing of the directory(s),
	// instead of using cached symbols.  Typically the cache will be used unless files
	// have a more recent modification date than the cache file.  This returns the
	// language-appropriate set of symbols for the directory(s), which could then provide
	// the symbols for a given package, library, or module at that path.
	ParseDir(path string, opts LangDirOpts) *syms.Symbol
}

// LangDirOpts provides options for Lang ParseDir method
type LangDirOpts struct {
	Subdirs bool `desc:"process subdirectories -- otherwise not"`
	Rebuild bool `desc:"rebuild the symbols by reprocessing from scratch instead of using cache"`
	Nocache bool `desc:"do not update the cache with results from processing"`
}
