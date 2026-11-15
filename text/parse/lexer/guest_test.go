// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer_test

import (
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/parse"
	_ "cogentcore.org/core/text/parse/languages/markdown"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGuestLexer tests that a fenced code block of markdown in a language
// is lexed by the lexer of that language, and the text after it again by
// that of markdown.
func TestGuestLexer(t *testing.T) {
	require.NoError(t, parse.LanguageSupport.OpenLanguage(fileinfo.Markdown))
	lp, _ := parse.LanguageSupport.Properties(fileinfo.Markdown)
	require.NotNil(t, lp.Parser)
	fs := parse.NewFileState()
	fs.Src.InitFromString("# Title\n\n```go\nfunc x() {}\n```\n\nText here\n", "x.md", fileinfo.Markdown)
	lp.Parser.LexAll(fs)
	has := func(ln int, tk token.Tokens) bool {
		for _, lx := range fs.Src.Lexs[ln] {
			if lx.Token.Token == tk {
				return true
			}
		}
		return false
	}
	assert.True(t, has(3, token.Keyword), "func is a Go keyword in the code block")
	assert.False(t, has(6, token.Keyword), "the text after it is markdown")
}

// TestLexTrace tests lexing with the trace of the rules printed.
func TestLexTrace(t *testing.T) {
	lexer.Trace = true
	defer func() { lexer.Trace = false }()
	fs := lexGo(t, "x\n")
	assert.NotEmpty(t, fs.Src.Lexs[0])
}

// TestNextEos tests finding the next end of statement at a depth, on the
// same line and later ones, and at any depth.
func TestNextEos(t *testing.T) {
	fs := lexGo(t, "package x\n\nfunc f() {\n\ta := 1\n\tb := 2\n}\n")
	src := &fs.Src
	ep, ok := src.NextEos(textpos.Pos{Line: 3}, 1)
	require.True(t, ok)
	assert.Equal(t, 3, ep.Line, "the end of a := 1, in the block")
	ep, ok = src.NextEos(textpos.Pos{Line: 2}, 1)
	require.True(t, ok)
	assert.Equal(t, 3, ep.Line, "from the line before")
	_, ok = src.NextEos(textpos.Pos{Line: 99}, 0)
	assert.False(t, ok)
	_, ok = src.NextEos(textpos.Pos{Line: 3}, 7)
	assert.False(t, ok, "no statement at that depth")
	ep, ok = src.NextEosAnyDepth(textpos.Pos{Line: 1})
	require.True(t, ok)
	assert.Greater(t, ep.Line, 1, "on a later line")
	_, ok = src.NextEosAnyDepth(textpos.Pos{Line: 99})
	assert.False(t, ok)
}
