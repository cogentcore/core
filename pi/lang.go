// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pi

import (
	"github.com/goki/ki/indent"
	"github.com/goki/pi/complete"
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

	// ParseFile does the complete processing of a given single file, given by txt bytes,
	// as appropriate for the language -- e.g., runs the lexer followed by the parser, and
	// manages any symbol output from parsing as appropriate for the language / format.
	// This is to be used for files of "primary interest" -- it does full type inference
	// and symbol resolution etc.  The Proc() FileState is locked during parsing,
	// and Switch is called after, so Done() will contain the processed info after this call.
	// If txt is nil then any existing source in fs is used.
	ParseFile(fs *FileStates, txt []byte)

	// HiLine does the lexing and potentially parsing of a given line of the file,
	// for purposes of syntax highlighting -- uses Done() FileState of existing context
	// if available from prior lexing / parsing. Line is in 0-indexed "internal" line indexes,
	// and provides relevant context for the overall parsing, which is performed
	// on the given line of text runes, and also updates corresponding source in FileState
	// (via a copy).  If txt is nil then any existing source in fs is used.
	HiLine(fs *FileStates, line int, txt []rune) lex.Line

	// CompleteLine provides the list of relevant completions for given text
	// which is at given position within the file.
	// Typically the language will call ParseLine on that line, and use the Ast
	// to guide the selection of relevant symbols that can complete the code at
	// the given point.
	CompleteLine(fs *FileStates, text string, pos lex.Pos) complete.Matches

	// CompleteEdit returns the completion edit data for integrating the
	// selected completion into the source
	CompleteEdit(fs *FileStates, text string, cp int, comp complete.Completion, seed string) (ed complete.Edit)

	// Lookup returns lookup results for given text which is at given position
	// within the file.  This can either be a file and position in file to
	// open and view, or direct text to show.
	Lookup(fs *FileStates, text string, pos lex.Pos) complete.Lookup

	// IndentLine returns the indentation level for given line based on
	// previous line's indentation level, and any delta change based on
	// e.g., brackets starting or ending the previous or current line, or
	// other language-specific keywords.  See lex.BracketIndentLine for example.
	// Indent level is in increments of tabSz for spaces, and tabs for tabs.
	// Operates on rune source with markup lex tags per line.
	IndentLine(fs *FileStates, src [][]rune, tags []lex.Line, ln int, tabSz int) (pInd, delInd, pLn int, ichr indent.Char)

	// AutoBracket returns what to do when a user types a starting bracket character
	// (bracket, brace, paren) while typing.
	// pos = position where bra will be inserted, and curLn is the current line
	// match = insert the matching ket, and newLine = insert a new line.
	AutoBracket(fs *FileStates, bra rune, pos lex.Pos, curLn []rune) (match, newLine bool)

	// below are more implementational methods not called externally typically

	// ParseDir does the complete processing of a given directory, optionally including
	// subdirectories, and optionally forcing the re-processing of the directory(s),
	// instead of using cached symbols.  Typically the cache will be used unless files
	// have a more recent modification date than the cache file.  This returns the
	// language-appropriate set of symbols for the directory(s), which could then provide
	// the symbols for a given package, library, or module at that path.
	ParseDir(fs *FileState, path string, opts LangDirOpts) *syms.Symbol

	// LexLine is a lower-level call (mostly used internally to the language) that
	// does just the lexing of a given line of the file, using existing context
	// if available from prior lexing / parsing.
	// Line is in 0-indexed "internal" line indexes.
	// The rune source is updated from the given text if non-nil.
	LexLine(fs *FileState, line int, txt []rune) lex.Line

	// ParseLine is a lower-level call (mostly used internally to the language) that
	// does complete parser processing of a single line from given file, and returns
	// the FileState for just that line.  Line is in 0-indexed "internal" line indexes.
	// The rune source information is assumed to have already been updated in FileState
	// Existing context information from full-file parsing is used as appropriate, but
	// the results will NOT be used to update any existing full-file Ast representation --
	// should call ParseFile to update that as appropriate.
	ParseLine(fs *FileState, line int) *FileState
}

// LangDirOpts provides options for Lang ParseDir method
type LangDirOpts struct {
	Subdirs bool `desc:"process subdirectories -- otherwise not"`
	Rebuild bool `desc:"rebuild the symbols by reprocessing from scratch instead of using cache"`
	Nocache bool `desc:"do not update the cache with results from processing"`
}
