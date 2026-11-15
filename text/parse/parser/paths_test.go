// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser_test

import (
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const pathsSrc = `package p

func f(a, b int, c string) (int, error) {
	x := g
	y, err := f(1, 2, "")
	u, v, w := 1, x, y
	_, z := 3, 4
	return y, err
}
`

// TestActionPaths tests the paths of the nodes of the actions of rules,
// with the symbols of variables that the actions add: a name is the first
// node of the name, and "Name..." is also all of the other names in a
// comma-separated list after it, but not the names after it that are not
// in the list (as the value in x := g).
func TestActionPaths(t *testing.T) {
	fs := goParser(t).ParseString(pathsSrc, "p.go", fileinfo.Go)
	require.NotNil(t, fs)
	assert.False(t, fs.ParseHasErrs(), fs.ParseErrReportAll())
	pkg := fs.ParseState.Scopes[0]
	f := pkg.Children["f"]
	require.NotNil(t, f)
	for _, nm := range []string{"x", "y", "err", "u", "v", "w", "z"} {
		sy, ok := f.Children[nm]
		if assert.True(t, ok, nm) {
			assert.Equal(t, token.NameVar, sy.Kind, nm)
		}
	}
	for _, nm := range []string{"a", "b", "c"} {
		sy, ok := f.Children[nm]
		if assert.True(t, ok, nm) {
			assert.Equal(t, token.NameVarParam, sy.Kind, nm)
		}
	}
	assert.NotContains(t, f.Children, "g", "the value of x := g is not a variable")
	assert.NotContains(t, f.Children, "_")

	ps := &fs.ParseState
	rs := ps.RuleString(false)
	assert.Contains(t, rs, "ExprListEl")
	assert.Contains(t, ps.RuleString(true), "AsgnNew")
	assert.NotEmpty(t, ps.Matches[2][0])
	assert.Contains(t, ps.Matches[2][0][0].String(), "FuncDecl")
	ps.Pos.Line, ps.Pos.Char = 3, 0
	assert.Contains(t, ps.NextSrcLine(), "x")
}
