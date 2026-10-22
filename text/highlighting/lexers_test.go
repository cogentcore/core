// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package highlighting

import (
	"os"
	"slices"
	"testing"

	"github.com/alecthomas/chroma/v3"
	"github.com/alecthomas/chroma/v3/lexers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCoreLexers checks that each lexer of [NewCoreLexers] has the same
// configuration and produces the same tokens as the chroma lexer
// of the same name.
func TestCoreLexers(t *testing.T) {
	var sources []string
	for _, fn := range []string{"highlighter.go", "lexers/bash.xml", "lexers/COPYING", "../../README.md"} {
		b, err := os.ReadFile(fn)
		require.NoError(t, err)
		sources = append(sources, string(b))
	}
	sources = append(sources, "---\ntitle: test\n---\n# Code\n\n```go\nfunc main() {}\n```\n\n```Goal\nx := 1\n```\n")

	core := NewCoreLexers()
	assert.GreaterOrEqual(t, len(core.Lexers), 20)
	for _, lex := range core.Lexers {
		cfg := lex.Config()
		t.Run(cfg.Name, func(t *testing.T) {
			full := lexers.GlobalLexerRegistry.Get(cfg.Name)
			require.NotNil(t, full)
			fcfg := full.Config()
			assert.Equal(t, fcfg.Aliases, cfg.Aliases)
			assert.Equal(t, fcfg.Filenames, cfg.Filenames)
			assert.Equal(t, fcfg.MimeTypes, cfg.MimeTypes)
			for _, src := range sources {
				assert.Equal(t, tokens(t, full, src), tokens(t, lex, src))
			}
		})
	}
	for _, name := range []string{"go", "Goal", "sh", "toml", "json", "yaml", "md", "html", "css", "js", "ts", "python", "c", "c++", "tex", "text", "diff", "makefile", "xml", "ini"} {
		if name == "Goal" { // goal is highlighted as go by content
			assert.Nil(t, core.Get(name))
			continue
		}
		assert.NotNil(t, core.Get(name), name)
	}
	assert.Nil(t, core.Get("solidity"))
}

func tokens(t *testing.T, lex chroma.Lexer, src string) []chroma.Token {
	it, err := lex.Tokenise(nil, src)
	require.NoError(t, err)
	return slices.Collect(it)
}
