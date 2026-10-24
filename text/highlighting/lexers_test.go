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

// TestGoLexers checks that each lexer of [NewGoLexers] has the same
// configuration and produces the same tokens as the chroma lexer
// of the same name.
func TestGoLexers(t *testing.T) {
	var sources []string
	for _, fn := range []string{"highlighter.go", "lexers/go_template.xml", "lexers/COPYING", "../../README.md"} {
		b, err := os.ReadFile(fn)
		require.NoError(t, err)
		sources = append(sources, string(b))
	}
	sources = append(sources, "package main\n\nvar t = `{{range .X}}{{.}}{{end}}`\n")

	gol := NewGoLexers()
	assert.Equal(t, 2, len(gol.Lexers))
	for _, lex := range gol.Lexers {
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
	for _, name := range []string{"go", "golang", "go-text-template", "main.go"} {
		assert.NotNil(t, gol.Get(name), name)
	}
	for _, name := range []string{"Goal", "sh", "toml", "md"} {
		assert.Nil(t, gol.Get(name), name)
	}
}

func tokens(t *testing.T, lex chroma.Lexer, src string) []chroma.Token {
	it, err := lex.Tokenise(nil, src)
	require.NoError(t, err)
	return slices.Collect(it)
}
