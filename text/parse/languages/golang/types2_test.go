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

const typesSrc = `package kinds

import (
	"errors"
	"strings"
)

type Base struct {
	ID int
}

func (b *Base) Ident() int { return b.ID }

type Named interface {
	Name() string
}

type Both interface {
	Named
	Ident() int
}

type Arr [4]float32

type Fn func(a int, b string) (bool, error)

type Ch chan int

type RecvCh <-chan string

type PtrT *Base

type MapT map[string][]*Base

type Inner struct {
	*Base
	strings.Builder
	Vals [3]int
	Fun  func(x int) int
	Ch   chan<- bool
	Anon struct {
		A, B int
	}
	Iface interface{ M() }
	Err   error
}

func (in Inner) Name() string { return "inner" }

func Make() (Inner, error) { return Inner{}, nil }

func Count(xs ...int) int { return len(xs) }

func Use(ch Ch, m MapT, arr Arr, s []string, fn Fn) {
	a := Make
	b, err := Make()
	c := b.Vals
	d := b.Base
	e := m["x"]
	f := arr[1]
	g := s[1:]
	h := <-ch
	i := &b
	j := *i
	k := Count(1, 2)
	l := float64(k)
	n := []int{1, 2}
	o := map[string]int{"a": 1}
	p := func(x int) int { return x }
	q := p(3)
	r := strings.ToUpper("x")
	t := errors.New("x")
	u := b.Ident()
	v := len(s)
	w := "str" + "ing"
	x := 1.5
	y := 'r'
	z := true
	aa := b.Anon.A
	bb, ok := o["a"]
	cc := Inner{}.Name()
	dd := new(Base)
	ee := make([]Base, 2)
	ff := struct{ X int }{1}
	gg := 1 + 2*3
	hh := !z
	ii := -x
	jj := [2]int{1, 2}
	var kk, ll int
	var mm = Inner{}
	for idx, ch := range "abc" {
		_, _ = idx, ch
	}
	for kv := range o {
		_ = kv
	}
	for cv := range ch {
		_ = cv
	}
	for _, sv := range s {
		_ = sv
	}
	for ai, av := range arr {
		_, _ = ai, av
	}
	switch tv := fn.(type) {
	default:
		_ = tv
	}
	fmt := fn
	_ = []any{a, b, err, c, d, e, f, g, h, i, j, k, l, n, o, p, q, r, t, u, v, w, x, y, z, aa, bb, ok, cc, dd, ee, ff, gg, hh, ii, jj, kk, ll, mm, fmt}
}
`

// TestTypeKinds tests the kinds of the types of a file.
func TestTypeKinds(t *testing.T) {
	_, _, pkg := parseSync(t, typesSrc)
	kinds := map[string]syms.Kinds{
		"Base":           syms.Struct,
		"Named":          syms.Interface,
		"Both":           syms.Interface,
		"Arr":            syms.Array,
		"Fn":             syms.Func,
		"MapT":           syms.Map,
		"PtrT":           syms.Ptr,
		"Inner":          syms.Struct,
		"map[string]int": syms.Map,
		"[]string":       syms.List,
	}
	for nm, k := range kinds {
		ty, ok := pkg.Types[nm]
		if assert.True(t, ok, nm) {
			assert.Equal(t, k, ty.Kind, nm)
		}
	}
	fn := pkg.Types["Fn"]
	assert.Equal(t, []int{2, 2}, fn.Size, "params and return values")
	mk := pkg.Types["func Make"]
	require.NotNil(t, mk)
	assert.Equal(t, []int{0, 2}, mk.Size)
	assert.Equal(t, "Inner", mk.Els[0].Type)
	assert.Equal(t, "error", mk.Els[1].Type)
	inner := pkg.Types["Inner"]
	for _, f := range []string{"Vals", "Fun", "Ch", "Anon", "Iface", "Err"} {
		assert.NotNil(t, inner.Els.ByName(f), f)
	}
	assert.Contains(t, inner.Meths, "Name")
	assert.Equal(t, "Both:Ident", pkg.Types["Both:Ident"].Name)
}

// TestVarTypes tests the types inferred for the variables of a function.
func TestVarTypes(t *testing.T) {
	_, _, pkg := parseSync(t, typesSrc)
	use := pkg.Children["Use"]
	require.NotNil(t, use)
	want := map[string]string{
		"a":   "func Make", // a function value
		"b":   "Inner",     // the first of the values of a call
		"err": "error",     // the second
		"bb":  "int",       // the value of a map
		"ok":  "bool",      // and whether it is there
		"c":   "[]int",
		"e":   "[]Base",
		"f":   "float32",  // an element of an array
		"g":   "[]string", // a slice of a slice
		"i":   "*Inner",
		"j":   "Inner",
		"k":   "int",
		"l":   "float64", // a conversion
		"n":   "[]int",
		"o":   "map[string]int",
		"v":   "int",
		"x":   "float64",
		"y":   "rune",
		"z":   "bool",
		"aa":  "int",
		"cc":  "Inner",
		"ee":  "[]Base",
		"jj":  "[]int",
		"kk":  "int",
		"ll":  "int",
		"mm":  "Inner",
		"fn":  "Fn",
		"fmt": "Fn",
		"idx": "int",
		"ch":  "rune",
		"kv":  "string",
		"sv":  "string",
		"ai":  "int",
		"av":  "float32",
	}
	for nm, ty := range want {
		sy, ok := use.Children[nm]
		if assert.True(t, ok, nm) {
			assert.Equal(t, ty, sy.Type, nm)
		}
	}
}
