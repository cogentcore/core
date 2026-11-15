// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"testing"

	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

// TestCompleteTypeNames tests completing type names, where a type is
// expected: after *, [], chan, and in a var declaration.
func TestCompleteTypeNames(t *testing.T) {
	fss, _, _ := parseSync(t, importSrc())
	pos := textpos.Pos{Line: 50, Char: 4}
	for _, s := range []string{"*Point", "[]Poi", "var x Poi", "chan Po"} {
		md := TheGoLang.CompleteLine(fss, s, pos)
		assert.Contains(t, matchTexts(md), "Point", s)
		assert.NotContains(t, matchTexts(md), "Rect", s)
	}
	md := TheGoLang.CompleteLine(fss, "var x float", pos)
	assert.Subset(t, matchTexts(md), []string{"float32", "float64"}, "builtin types")

	// directly, with the types of the package and builtin ones
	_, pfs, pkg := parseSync(t, importSrc())
	var tm complete.Matches
	TheGoLang.CompleteTypeName(pfs, pkg, "Po", &tm)
	assert.Equal(t, "Po", tm.Seed)
	assert.Contains(t, matchTexts(tm), "Point")
	tm = complete.Matches{}
	TheGoLang.CompleteTypeName(pfs, pkg, "in", &tm)
	assert.Subset(t, matchTexts(tm), []string{"int", "int64"})
	tm = complete.Matches{}
	TheGoLang.CompleteTypeName(pfs, pkg, "func ", &tm)
	assert.NotEmpty(t, tm.Matches, "function types only for a func seed")
	for _, m := range tm.Matches {
		assert.Contains(t, m.Text, "func ")
	}
}

// TestLookupMore tests looking up qualified names, methods through types,
// and type names after * and [], and that other strings do not panic.
func TestLookupMore(t *testing.T) {
	fss, _, _ := parseSync(t, importSrc())
	pos := textpos.Pos{Line: 50, Char: 4}
	tests := []struct {
		str    string
		st, ed int
	}{
		{"Rect.Ar", 26, 28}, // a method of a type
		{"*Point", 12, 12},
		{"[]Poi", 12, 12},
	}
	for _, tt := range tests {
		ld := TheGoLang.Lookup(fss, tt.str, pos)
		assert.NotEmpty(t, ld.Filename, tt.str)
		assert.Equal(t, tt.st, ld.StLine, tt.str)
		assert.Equal(t, tt.ed, ld.EdLine, tt.str)
	}
	ld := TheGoLang.Lookup(fss, "errors.Ne", pos)
	assert.NotEmpty(t, ld.Filename, "a function of an imported package")
	assert.Empty(t, TheGoLang.Lookup(fss, "NoSuch.X", pos).Filename)
	for _, s := range []string{"(", "[]", "*", ")", "{", "x.y.z", "1+", "func(", "map[string]Po", "go", "Point{}.", "fmt.", `"str".`, "r.Size.X."} {
		assert.NotPanics(t, func() {
			TheGoLang.CompleteLine(fss, s, pos)
			TheGoLang.Lookup(fss, s, pos)
		}, s)
	}
}
