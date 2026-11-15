// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"testing"

	"cogentcore.org/core/text/parse/syms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const exprSrc = `package exprs

import "strings"

type Base struct{ ID int }

func (b *Base) Ident() int { return b.ID }

type Wrap struct {
	*Base
	strings.Builder
	N int
}

type Ch chan int

func Use(ch Ch, sc <-chan string, z bool, w Wrap) {
	recv := <-ch
	srecv := <-sc
	not := !z
	less := 1 < 2
	eq := recv == 3
	both := z && less
	either := z || less
	cat := "a" + "b"
	sum := 1.5 + 2.5
	ptr := new(Base)
	fn := func(x int) string { return "" }
	res := fn(3)
	base := w.Base
	id := w.Ident()
	bld := w.Builder
	for cv := range ch {
		_ = cv
	}
	_ = []any{recv, srecv, not, less, eq, both, either, cat, sum, ptr, res, base, id, bld}
}
`

// TestExprTypes tests the types inferred for expressions: receiving from
// channels and ranging over them, boolean and string operators, new, func
// literals and their calls, and embedded pointer and qualified fields and
// their promoted methods.
func TestExprTypes(t *testing.T) {
	_, _, pkg := parseSync(t, exprSrc)
	use := pkg.Children["Use"]
	require.NotNil(t, use)
	want := map[string]string{
		"recv":   "int",
		"srecv":  "string",
		"not":    "bool",
		"less":   "bool",
		"eq":     "bool",
		"both":   "bool",
		"either": "bool",
		"cat":    "string",
		"sum":    "float64",
		"ptr":    "*Base",
		"res":    "string", // the result of calling a func literal
		"base":   "Base",   // an embedded *Base
		"id":     "int",    // a method promoted from it
		"cv":     "int",
	}
	for nm, ty := range want {
		sy, ok := use.Children[nm]
		if assert.True(t, ok, nm) {
			assert.Equal(t, ty, sy.Type, nm)
		}
	}
	fn := use.Children["fn"]
	require.NotNil(t, fn)
	fty := pkg.Types[fn.Type]
	require.NotNil(t, fty, "the type of the func literal")
	assert.Equal(t, syms.Func, fty.Kind)

	ch := pkg.Types["Ch"]
	require.NotNil(t, ch)
	assert.Equal(t, syms.Chan, ch.Kind)
	assert.Equal(t, "int", ch.Els[0].Type)
	wrap := pkg.Types["Wrap"]
	require.NotNil(t, wrap)
	assert.Equal(t, "*Base", wrap.Els.ByName("Base").Type, "an embedded pointer field")
	assert.NotNil(t, wrap.Els.ByName("Builder"), "named by its type without the package")
	assert.Nil(t, wrap.Els.ByName("strings.Builder"))
	assert.Contains(t, wrap.Meths, "Ident", "promoted from *Base")
}

// TestQualifyComposite tests qualifying the element types of composite
// type names, with the package outside, and not builtin element types.
func TestQualifyComposite(t *testing.T) {
	assert.Equal(t, "[]byte", QualifyType("pkg", "[]byte"))
	assert.Equal(t, "pkg.[]T", QualifyType("pkg", "[]T"))
	assert.Equal(t, "pkg.*T", QualifyType("pkg", "*T"))
	assert.Equal(t, "*int", QualifyType("pkg", "*int"))
	assert.Equal(t, "pkg.chan T", QualifyType("pkg", "chan T"))
	assert.Equal(t, "other.[]T", QualifyType("pkg", "[]other.T"))
	assert.Equal(t, "map[string]T", QualifyType("pkg", "map[string]T"), "not parsed")
	assert.Equal(t, "func(int)", QualifyType("pkg", "func(int)"))
}
