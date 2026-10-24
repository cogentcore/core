// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package highlighting

import (
	"cogentcore.org/core/text/highlighting/lexers"
	"github.com/alecthomas/chroma/v3"
)

// lexerRegistry is the registry returned by [Lexers].
var lexerRegistry *chroma.LexerRegistry

// Lexers returns the registry of chroma lexers used for syntax highlighting.
// It is made by [NewGoLexers] by default, which keeps the definitions of the
// hundreds of other languages that chroma supports out of the app.
// [AddLexers] adds the lexers for other languages, and importing
// cogentcore.org/core/text/highlighting/alllexers switches it to all of the
// chroma lexers, as an editor of any language needs. Code in languages
// without a lexer is not highlighted.
func Lexers() *chroma.LexerRegistry {
	if lexerRegistry == nil {
		lexerRegistry = NewGoLexers()
	}
	return lexerRegistry
}

// SetLexers sets the registry of chroma lexers returned by [Lexers].
// It must be called before any highlighting, typically in an init function.
func SetLexers(reg *chroma.LexerRegistry) {
	lexerRegistry = reg
}

// AddLexers adds lexers to [Lexers] with the given functions of the
// cogentcore.org/core/text/highlighting/lexers package, such as:
//
//	highlighting.AddLexers(lexers.Bash, lexers.TOML)
//
// Only the definitions of the given languages are linked into the app.
// It must be called before any highlighting, typically in an init function
// or at the start of main.
func AddLexers(lexers ...func(reg *chroma.LexerRegistry)) {
	reg := Lexers()
	for _, add := range lexers {
		add(reg)
	}
}

// GetLexer returns the chroma lexer in [Lexers] for the given language name,
// alias or file extension, or nil if there is none.
func GetLexer(name string) chroma.Lexer {
	return Lexers().Get(name)
}

// NewGoLexers returns a new registry with the chroma lexers for Go and Go
// text templates, which highlight the raw strings of Go code. It is the
// default for [Lexers].
func NewGoLexers() *chroma.LexerRegistry {
	reg := chroma.NewLexerRegistry()
	lexers.Go(reg)
	return reg
}
