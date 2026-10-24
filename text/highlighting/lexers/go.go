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
			{`\n`, chroma.TextWhitespace, nil},
			{`\s+`, chroma.TextWhitespace, nil},
			{`//[^\s\n\r][^\n\r]*`, chroma.CommentPreproc, nil},
			{`//[^\n\r]*`, chroma.CommentSingle, nil},
			{`/(\\\n)?[*](.|\n)*?[*](\\\n)?/`, chroma.CommentMultiline, nil},
			{`(import|package)\b`, chroma.KeywordNamespace, nil},
			{`(var|func|struct|map|chan|type|interface|const)\b`, chroma.KeywordDeclaration, nil},
			{chroma.Words(``, `\b`, `break`, `default`, `select`, `case`, `defer`, `go`, `else`, `goto`, `switch`, `fallthrough`, `if`, `range`, `continue`, `for`, `return`), chroma.Keyword, nil},
			{`(true|false|iota|nil)\b`, chroma.KeywordConstant, nil},
			{chroma.Words(``, `\b(\()`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `int`, `int8`, `int16`, `int32`, `int64`, `float`, `float32`, `float64`, `complex64`, `complex128`, `byte`, `rune`, `string`, `bool`, `error`, `uintptr`, `print`, `println`, `panic`, `recover`, `close`, `complex`, `real`, `imag`, `len`, `cap`, `append`, `copy`, `delete`, `new`, `make`, `clear`, `min`, `max`), chroma.ByGroups(chroma.NameBuiltin, chroma.Punctuation), nil},
			{chroma.Words(``, `\b`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `int`, `int8`, `int16`, `int32`, `int64`, `float`, `float32`, `float64`, `complex64`, `complex128`, `byte`, `rune`, `string`, `bool`, `error`, `uintptr`, `any`), chroma.KeywordType, nil},
			{`\d[\d_]*i`, chroma.LiteralNumber, nil},
			{`\d[\d_]*\.[\d_]*([Ee][-+]?\d[\d_]*)?i`, chroma.LiteralNumber, nil},
			{`\.\d[\d_]*([Ee][-+]?\d[\d_]*)?i`, chroma.LiteralNumber, nil},
			{`\d[\d_]*[Ee][-+]?\d[\d_]*i`, chroma.LiteralNumber, nil},
			{`\d[\d_]*(\.[\d_]*([eE][+\-]?\d[\d_]*)?|[eE][+\-]?\d[\d_]*)`, chroma.LiteralNumberFloat, nil},
			{`\.\d[\d_]*([eE][+\-]?\d[\d_]*)?`, chroma.LiteralNumberFloat, nil},
			{`0[xX][0-9a-fA-F_]*\.?[0-9a-fA-F_]*[pP][+\-]?[0-9_]+`, chroma.LiteralNumberFloat, nil},
			{`0[oO]?[0-7_]+`, chroma.LiteralNumberOct, nil},
			{`0[xX][0-9a-fA-F_]+`, chroma.LiteralNumberHex, nil},
			{`0[bB][01_]+`, chroma.LiteralNumberBin, nil},
			{`(0|[1-9][0-9_]*)`, chroma.LiteralNumberInteger, nil},
			{`'(\\['"\\abfnrtv]|\\x[0-9a-fA-F]{2}|\\[0-7]{1,3}|\\u[0-9a-fA-F]{4}|\\U[0-9a-fA-F]{8}|[^\\])'`, chroma.LiteralStringChar, nil},
			{"(`)([^`]*)(`)", chroma.ByGroups(chroma.LiteralString, chroma.UsingLexer(chroma.TypeRemappingLexer(goTextTemplate, chroma.TypeMapping{{chroma.Other, chroma.LiteralString, nil}})), chroma.LiteralString), nil},
			{`"(\\\\|\\"|[^"])*"`, chroma.LiteralString, nil},
			{`(<<=|>>=|<<|>>|<=|>=|&\^=|&\^|\+=|-=|\*=|/=|%=|&=|\|=|&&|\|\||<-|\+\+|--|==|!=|:=|\.\.\.|[+\-*/%&])`, chroma.Operator, nil},
			{`([a-zA-Z_]\w*)(\s*)(\()`, chroma.ByGroups(chroma.NameFunction, chroma.UsingSelf("root"), chroma.Punctuation), nil},
			{`[|^<>=!()\[\]{}.,;:~]`, chroma.Punctuation, nil},
			{`[^\W\d]\w*`, chroma.NameOther, nil},
		},
	}
}
