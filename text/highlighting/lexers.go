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
// On native platforms, it is the registry of all chroma lexers by default.
// On the web, it is made by [NewCoreLexers] by default, which keeps the
// definitions of all other languages out of the app, and importing
// cogentcore.org/core/text/highlighting/alllexers switches it to all of the
// chroma lexers. Code in languages without a lexer is not highlighted.
func Lexers() *chroma.LexerRegistry {
	if lexerRegistry == nil {
		lexerRegistry = NewCoreLexers()
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

// coreLexerFiles are the chroma XML lexer definitions of [NewCoreLexers],
// copied from github.com/alecthomas/chroma/v3/lexers/embedded.
//
//go:embed lexers/*.xml
var coreLexerFiles embed.FS

// NewCoreLexers returns a new registry of the chroma lexers for a set of
// common languages: Go, Go templates, Bash, C, C++, CSS, diff, HTML, INI,
// JavaScript, JSON, makefile, markdown, plain text, Python, TeX, TOML,
// TypeScript, XML and YAML. It is the default for [Lexers] on the web.
// The rules of each lexer are loaded when the lexer is first used.
func NewCoreLexers() *chroma.LexerRegistry {
	reg := chroma.NewLexerRegistry()
	files, err := coreLexerFiles.ReadDir("lexers")
	if err != nil {
		panic(err)
	}
	for _, f := range files {
		reg.Register(chroma.MustNewXMLLexer(coreLexerFiles, "lexers/"+f.Name()))
	}

	// The Go template lexers, as in chroma.
	goTemplate := func() *chroma.RegexLexer {
		return chroma.MustNewXMLLexer(coreLexerFiles, "lexers/go_template.xml")
	}
	html := chroma.MustNewXMLLexer(coreLexerFiles, "lexers/html.xml")
	reg.Register(chroma.DelegatingLexer(html, goTemplate().SetConfig(&chroma.Config{
		Name:    "Go HTML Template",
		Aliases: []string{"go-html-template"},
	})))
	goTextTemplate := reg.Register(goTemplate().SetConfig(&chroma.Config{
		Name:    "Go Text Template",
		Aliases: []string{"go-text-template"},
	}))
	reg.Register(newGoLexer(goTextTemplate))
	reg.Register(newMarkdownLexer(reg))
	return reg
}
