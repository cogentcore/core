// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexers

// This file contains a copy of the Go lexer of
// github.com/alecthomas/chroma/v3/lexers, which is defined in Go code
// rather than XML.
// Copyright (C) 2017 Alec Thomas, under an MIT License (see COPYING).

import (
	_ "embed"
	"strings"

	"github.com/alecthomas/chroma/v3"
)

//go:embed go_template.xml
var xmlGoTemplate string

// Go adds the chroma lexers for Go (go, golang) and Go text templates
// (go-text-template), which highlight the raw strings of Go code,
// to the registry.
func Go(reg *chroma.LexerRegistry) {
	goTextTemplate := reg.Register(chroma.MustNewXMLLexer(xmlFS{"go_template.xml", xmlGoTemplate}, "go_template.xml").SetConfig(&chroma.Config{
		Name:    "Go Text Template",
		Aliases: []string{"go-text-template"},
	}))
	reg.Register(newGoLexer(goTextTemplate))
}

// newGoLexer returns the Go lexer, using the given Go text template
// lexer for raw strings.
func newGoLexer(goTextTemplate chroma.Lexer) chroma.Lexer {
	return chroma.MustNewLexer(
		&chroma.Config{
			Name:      "Go",
			Aliases:   []string{"go", "golang"},
			Filenames: []string{"*.go"},
			MimeTypes: []string{"text/x-gosrc"},
		},
		func() chroma.Rules { return goRules(goTextTemplate) },
	).SetAnalyser(func(text string) float32 {
		if strings.Contains(text, "fmt.") && strings.Contains(text, "package ") {
			return 0.5
		}
		if strings.Contains(text, "package ") {
			return 0.1
		}
		return 0.0
	})
}

func goRules(goTextTemplate chroma.Lexer) chroma.Rules {
	return chroma.Rules{
		"root": {
			{Pattern: `\n`, Type: chroma.TextWhitespace, Mutator: nil},
			{Pattern: `\s+`, Type: chroma.TextWhitespace, Mutator: nil},
			{Pattern: `//[^\s\n\r][^\n\r]*`, Type: chroma.CommentPreproc, Mutator: nil},
			{Pattern: `//[^\n\r]*`, Type: chroma.CommentSingle, Mutator: nil},
			{Pattern: `/(\\\n)?[*](.|\n)*?[*](\\\n)?/`, Type: chroma.CommentMultiline, Mutator: nil},
			{Pattern: `(import|package)\b`, Type: chroma.KeywordNamespace, Mutator: nil},
			{Pattern: `(var|func|struct|map|chan|type|interface|const)\b`, Type: chroma.KeywordDeclaration, Mutator: nil},
			{Pattern: chroma.Words(``, `\b`, `break`, `default`, `select`, `case`, `defer`, `go`, `else`, `goto`, `switch`, `fallthrough`, `if`, `range`, `continue`, `for`, `return`), Type: chroma.Keyword, Mutator: nil},
			{Pattern: `(true|false|iota|nil)\b`, Type: chroma.KeywordConstant, Mutator: nil},
			{Pattern: chroma.Words(``, `\b(\()`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `int`, `int8`, `int16`, `int32`, `int64`, `float`, `float32`, `float64`, `complex64`, `complex128`, `byte`, `rune`, `string`, `bool`, `error`, `uintptr`, `print`, `println`, `panic`, `recover`, `close`, `complex`, `real`, `imag`, `len`, `cap`, `append`, `copy`, `delete`, `new`, `make`, `clear`, `min`, `max`), Type: chroma.ByGroups(chroma.NameBuiltin, chroma.Punctuation), Mutator: nil},
			{Pattern: chroma.Words(``, `\b`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `int`, `int8`, `int16`, `int32`, `int64`, `float`, `float32`, `float64`, `complex64`, `complex128`, `byte`, `rune`, `string`, `bool`, `error`, `uintptr`, `any`), Type: chroma.KeywordType, Mutator: nil},
			{Pattern: `\d[\d_]*i`, Type: chroma.LiteralNumber, Mutator: nil},
			{Pattern: `\d[\d_]*\.[\d_]*([Ee][-+]?\d[\d_]*)?i`, Type: chroma.LiteralNumber, Mutator: nil},
			{Pattern: `\.\d[\d_]*([Ee][-+]?\d[\d_]*)?i`, Type: chroma.LiteralNumber, Mutator: nil},
			{Pattern: `\d[\d_]*[Ee][-+]?\d[\d_]*i`, Type: chroma.LiteralNumber, Mutator: nil},
			{Pattern: `\d[\d_]*(\.[\d_]*([eE][+\-]?\d[\d_]*)?|[eE][+\-]?\d[\d_]*)`, Type: chroma.LiteralNumberFloat, Mutator: nil},
			{Pattern: `\.\d[\d_]*([eE][+\-]?\d[\d_]*)?`, Type: chroma.LiteralNumberFloat, Mutator: nil},
			{Pattern: `0[xX][0-9a-fA-F_]*\.?[0-9a-fA-F_]*[pP][+\-]?[0-9_]+`, Type: chroma.LiteralNumberFloat, Mutator: nil},
			{Pattern: `0[oO]?[0-7_]+`, Type: chroma.LiteralNumberOct, Mutator: nil},
			{Pattern: `0[xX][0-9a-fA-F_]+`, Type: chroma.LiteralNumberHex, Mutator: nil},
			{Pattern: `0[bB][01_]+`, Type: chroma.LiteralNumberBin, Mutator: nil},
			{Pattern: `(0|[1-9][0-9_]*)`, Type: chroma.LiteralNumberInteger, Mutator: nil},
			{Pattern: `'(\\['"\\abfnrtv]|\\x[0-9a-fA-F]{2}|\\[0-7]{1,3}|\\u[0-9a-fA-F]{4}|\\U[0-9a-fA-F]{8}|[^\\])'`, Type: chroma.LiteralStringChar, Mutator: nil},
			{Pattern: "(`)([^`]*)(`)", Type: chroma.ByGroups(chroma.LiteralString, chroma.UsingLexer(chroma.TypeRemappingLexer(goTextTemplate, chroma.TypeMapping{{chroma.Other, chroma.LiteralString, nil}})), chroma.LiteralString), Mutator: nil},
			{Pattern: `"(\\\\|\\"|[^"])*"`, Type: chroma.LiteralString, Mutator: nil},
			{Pattern: `(<<=|>>=|<<|>>|<=|>=|&\^=|&\^|\+=|-=|\*=|/=|%=|&=|\|=|&&|\|\||<-|\+\+|--|==|!=|:=|\.\.\.|[+\-*/%&])`, Type: chroma.Operator, Mutator: nil},
			{Pattern: `([a-zA-Z_]\w*)(\s*)(\()`, Type: chroma.ByGroups(chroma.NameFunction, chroma.UsingSelf("root"), chroma.Punctuation), Mutator: nil},
			{Pattern: `[|^<>=!()\[\]{}.,;:~]`, Type: chroma.Punctuation, Mutator: nil},
			{Pattern: `[^\W\d]\w*`, Type: chroma.NameOther, Mutator: nil},
		},
	}
}
