// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexers

import (
	"slices"
	"testing"

	"github.com/alecthomas/chroma/v3"
	"github.com/alecthomas/chroma/v3/lexers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLexers checks that each function adds a lexer with the same
// configuration as the chroma lexer of the same name, which produces the
// same tokens, together with the lexers that it uses.
func TestLexers(t *testing.T) {
	// Short sources, as some lexers take very long on long ones.
	sources := []string{
		"#!/bin/sh\n[section]\nkey = \"value\" # comment\n<a href='x'>{{.X}}</a>\n",
		"func f(x int) { return x+1; } // done\n/* block */ 'c' 0x1F 3.5e2\n",
		"<style>p { color: red }</style><script>let x = [1, 2];</script>\n",
	}

	funcs := map[string]func(reg *chroma.LexerRegistry){"Go": Go}
	for name, f := range all {
		funcs[name] = f
	}
	for name, f := range funcs {
		t.Run(name, func(t *testing.T) {
			reg := chroma.NewLexerRegistry()
			f(reg)
			lex := reg.Get(name)
			require.NotNil(t, lex)
			full := lexers.GlobalLexerRegistry.Get(name)
			require.NotNil(t, full)
			cfg, fcfg := lex.Config(), full.Config()
			assert.Equal(t, fcfg.Aliases, cfg.Aliases)
			assert.Equal(t, fcfg.Filenames, cfg.Filenames)
			assert.Equal(t, fcfg.MimeTypes, cfg.MimeTypes)
			for _, src := range sources {
				assert.Equal(t, tokens(t, full, src), tokens(t, lex, src))
			}
		})
	}
}

func tokens(t *testing.T, lex chroma.Lexer, src string) []chroma.Token {
	it, err := lex.Tokenise(nil, src)
	require.NoError(t, err)
	return slices.Collect(it)
}
