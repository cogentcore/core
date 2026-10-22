// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package highlighting

import (
	"embed"

	"github.com/alecthomas/chroma/v3"
)

// lexerRegistry is the registry returned by [Lexers].
var lexerRegistry *chroma.LexerRegistry

// Lexers returns the registry of chroma lexers used for syntax highlighting.
// It is made by [NewGoLexers] by default, which keeps the definitions of the
// hundreds of other languages that chroma supports out of the app, and
// importing cogentcore.org/core/text/highlighting/alllexers switches it to
// all of the chroma lexers, as an editor needs. Code in languages without
// a lexer is not highlighted.
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

// GetLexer returns the chroma lexer in [Lexers] for the given language name,
// alias or file extension, or nil if there is none.
func GetLexer(name string) chroma.Lexer {
	return Lexers().Get(name)
}

// goTemplateLexer is the chroma XML lexer definition of Go templates,
// copied from github.com/alecthomas/chroma/v3/lexers/embedded.
//
//go:embed lexers/go_template.xml
var goTemplateLexer embed.FS

// NewGoLexers returns a new registry of the chroma lexers for Go and Go text
// templates, which highlight the raw strings of Go code. It is the default
// for [Lexers]. The rules of each lexer are loaded when the lexer is first used.
func NewGoLexers() *chroma.LexerRegistry {
	reg := chroma.NewLexerRegistry()
	goTextTemplate := reg.Register(chroma.MustNewXMLLexer(goTemplateLexer, "lexers/go_template.xml").SetConfig(&chroma.Config{
		Name:    "Go Text Template",
		Aliases: []string{"go-text-template"},
	}))
	reg.Register(newGoLexer(goTextTemplate))
	return reg
}
