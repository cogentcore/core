// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package highlighting

// This file contains copies of the Go and markdown lexers of
// github.com/alecthomas/chroma/v3/lexers, which are defined in Go code
// rather than XML, for [NewCoreLexers].
// Copyright (C) 2017 Alec Thomas, under an MIT License (see lexers/COPYING).

import (
	"iter"
	"strings"

	"github.com/alecthomas/chroma/v3"
)

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

// newMarkdownLexer returns the markdown lexer, which gets the YAML lexer
// for frontmatter from the given registry.
func newMarkdownLexer(reg *chroma.LexerRegistry) chroma.Lexer {
	return &markdownLexer{registry: reg, Lexer: chroma.MustNewLexer(
		&chroma.Config{
			Name:      "markdown",
			Aliases:   []string{"md", "mkd"},
			Filenames: []string{"*.md", "*.mkd", "*.markdown"},
			MimeTypes: []string{"text/x-markdown"},
		},
		markdownRules,
	)}
}

// markdownLexer wraps the base Markdown lexer to highlight top-of-file YAML frontmatter.
type markdownLexer struct {
	chroma.Lexer
	registry *chroma.LexerRegistry
}

// Lexes Markdown, highlighting a leading YAML frontmatter block before delegating to Markdown rules.
func (m *markdownLexer) Tokenise(options *chroma.TokeniseOptions, text string) (iter.Seq[chroma.Token], error) {
	frontmatter, rest, ok := splitFrontmatter(text)
	if !ok {
		return m.Lexer.Tokenise(options, text)
	}

	yamlLexer := m.registry.Get("YAML")
	if yamlLexer == nil {
		return m.Lexer.Tokenise(options, text)
	}

	yamlTokens, err := yamlLexer.Tokenise(options, frontmatter)
	if err != nil {
		return nil, err
	}
	markdownTokens, err := m.Lexer.Tokenise(options, rest)
	if err != nil {
		return nil, err
	}
	return chroma.Concaterator(yamlTokens, markdownTokens), nil
}

// Extracts a leading YAML frontmatter block if the document starts with one.
func splitFrontmatter(text string) (frontmatter string, rest string, ok bool) {
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		return "", text, false
	}

	lineEnd := strings.IndexByte(text, '\n')
	if lineEnd < 0 {
		return "", text, false
	}
	if strings.TrimSuffix(text[:lineEnd], "\r") != "---" {
		return "", text, false
	}

	for pos := lineEnd + 1; pos < len(text); {
		next := strings.IndexByte(text[pos:], '\n')
		if next < 0 {
			break
		}
		lineEnd = pos + next
		line := strings.TrimSuffix(text[pos:lineEnd], "\r")
		if line == "---" {
			return text[:lineEnd+1], text[lineEnd+1:], true
		}
		pos = lineEnd + 1
	}
	return "", text, false
}

func markdownRules() chroma.Rules {
	return chroma.Rules{
		"root": {
			{`<!--[\w\W]*?-->`, chroma.CommentMultiline, nil},
			{`^(#[^#].+\n)`, chroma.ByGroups(chroma.GenericHeading), nil},
			{`^(#{2,6}.+\n)`, chroma.ByGroups(chroma.GenericSubheading), nil},
			{`^(\s*)([*-] )(\[[ xX]\])( .+\n)`, chroma.ByGroups(chroma.Text, chroma.Keyword, chroma.Keyword, chroma.UsingSelf("inline")), nil},
			{`^(\s*)([*-])(\s)(.+\n)`, chroma.ByGroups(chroma.Text, chroma.Keyword, chroma.Text, chroma.UsingSelf("inline")), nil},
			{`^(\s*)([0-9]+\.)( .+\n)`, chroma.ByGroups(chroma.Text, chroma.Keyword, chroma.UsingSelf("inline")), nil},
			{`^(\s*>\s)(.+\n)`, chroma.ByGroups(chroma.Keyword, chroma.GenericEmph), nil},
			{"^(```\\n)([\\w\\W]*?)(^```$)", chroma.ByGroups(chroma.String, chroma.Text, chroma.String), nil},
			{
				"^(```)(\\w+)(\\n)([\\w\\W]*?)(^```$)",
				chroma.UsingByGroup(2, 4, chroma.String, chroma.String, chroma.String, chroma.Text, chroma.String),
				nil,
			},
			chroma.Include("inline"),
		},
		"inline": {
			{`<!--[\w\W]*?-->`, chroma.CommentMultiline, nil},
			{`\\.`, chroma.Text, nil},
			{`(\s)(\*|_)((?:(?!\2).)*)(\2)((?=\W|\n))`, chroma.ByGroups(chroma.Text, chroma.GenericEmph, chroma.GenericEmph, chroma.GenericEmph, chroma.Text), nil},
			{`(\s)((\*\*|__).*?)\3((?=\W|\n))`, chroma.ByGroups(chroma.Text, chroma.GenericStrong, chroma.GenericStrong, chroma.Text), nil},
			{`(\s)(~~[^~]+~~)((?=\W|\n))`, chroma.ByGroups(chroma.Text, chroma.GenericDeleted, chroma.Text), nil},
			{"`[^`]+`", chroma.LiteralStringBacktick, nil},
			{`[@#][\w/:]+`, chroma.NameEntity, nil},
			{`(!?\[)([^]]+)(\])(\()([^)]+)(\))`, chroma.ByGroups(chroma.Text, chroma.NameTag, chroma.Text, chroma.Text, chroma.NameAttribute, chroma.Text), nil},
			{`.|\n`, chroma.Text, nil},
		},
	}
}
