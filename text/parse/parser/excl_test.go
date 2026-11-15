// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser_test

import (
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/parse/parser"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExclusionRule tests compiling a rule with exclusionary elements
// (after a !), and that they exclude the matches that they match.
func TestExclusionRule(t *testing.T) {
	src := "package p\n\nvar x = a + b\nvar y = a + 1\nvar z = 2 + b\n"
	fs := goParser(t).ParseString(src, "x.go", fileinfo.Go)
	require.NotNil(t, fs)
	ps := &fs.ParseState

	pr := parser.NewRule()
	pr.SetName("PlusNames")
	pr.Rule = "'+' ! 'Name' '+' 'Name'" // a + that is not between two names
	require.True(t, pr.Compile(ps))
	assert.Equal(t, 0, pr.ExclKeyIndex)
	assert.Len(t, pr.ExclRev, 1, "the tokens before the key")
	assert.Len(t, pr.ExclFwd, 1, "the tokens after it")

	excluded := map[int]bool{}
	for ln := range fs.Src.Lexs {
		for i, lx := range fs.Src.Lexs[ln] {
			if lx.Token.Token != token.OpMathAdd {
				continue
			}
			scope := textpos.Region{Start: textpos.Pos{Line: ln}, End: textpos.Pos{Line: ln, Char: len(fs.Src.Lexs[ln])}}
			kt := textpos.Region{Start: textpos.Pos{Line: ln, Char: i}, End: textpos.Pos{Line: ln, Char: i + 1}}
			excluded[ln] = pr.MatchExclude(ps, scope, kt, 0, nil)
		}
	}
	assert.Equal(t, map[int]bool{2: true, 3: false, 4: false}, excluded, "only a + b is excluded")

	// the next sibling of the first declaration of the file is the next one
	decls := fs.ParseState.AST.ChildAST(0)
	require.NotNil(t, decls)
	if nd := decls.ChildAST(0); assert.NotNil(t, nd) {
		sib := nd.NextSiblingAST()
		require.NotNil(t, sib)
		assert.Equal(t, decls.ChildAST(1), sib)
		assert.Nil(t, decls.ChildAST(decls.NumChildren()-1).NextSiblingAST(), "the last one")
	}

	opt := parser.NewRule()
	opt.SetName("OptPlus")
	opt.Rule = "'+' ! ?'Name' '+'" // an optional token before
	require.True(t, opt.Compile(ps))
	assert.True(t, opt.ExclRev[0].Opt)

	bad := parser.NewRule()
	bad.SetName("NoKey")
	bad.Rule = "'+' ! 'Name' 'Name'" // the key token is not in the exclusion
	assert.False(t, bad.Compile(ps))
	none := parser.NewRule()
	none.SetName("NoToken")
	none.Rule = "Expr ! 'Name'" // no token to exclude
	assert.False(t, none.Compile(ps))
	assert.NotEmpty(t, ps.Errs)
}
