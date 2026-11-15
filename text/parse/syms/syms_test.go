// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syms

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// region returns a region from the start to the end line.
func region(st, ed int) textpos.Region {
	return textpos.Region{Start: textpos.Pos{Line: st}, End: textpos.Pos{Line: ed, Char: 1}}
}

// newTestPkg returns a package symbol with this structure, in the file
// "pkg/a.go":
//
//	pkg (package, lines 0-30)
//	  Point (type, lines 2-6): X, Y (fields), Add (method, lines 8-10)
//	  NewPoint (func, lines 12-15): p (var)
//	  Origin (var Point)
func newTestPkg() *Symbol {
	fn := filepath.Join("pkg", "a.go")
	pkg := NewSymbol("pkg", token.NamePackage, fn, region(0, 30))
	pt := pkg.Children.AddNew("Point", token.NameStruct, fn, region(2, 6))
	pt.Children.AddNew("X", token.NameField, fn, region(3, 3)).Type = "float64"
	pt.Children.AddNew("Y", token.NameField, fn, region(4, 4)).Type = "float64"
	pt.Children.AddNew("Add", token.NameMethod, fn, region(8, 10)).Detail = "func (p *Point) Add(o Point)"
	np := pkg.Children.AddNew("NewPoint", token.NameFunction, fn, region(12, 15))
	np.Detail = "func NewPoint(x, y float64) *Point"
	np.Children.AddNew("p", token.NameVar, fn, region(13, 13)).Type = "*Point"
	pkg.Children.AddNew("Origin", token.NameVarGlobal, fn, region(17, 17)).Type = "Point"
	ty := NewType("Point", Struct)
	ty.Els.Add("X", "float64")
	ty.Els.Add("Y", "float64")
	add := NewType("Point:Add", Method)
	add.Els.Add("p", "*Point")
	add.Els.Add("o", "Point")
	add.Size = []int{2, 0}
	ty.Meths.Add(add)
	pkg.Types.Add(ty)
	return pkg
}

func TestSymbol(t *testing.T) {
	pkg := newTestPkg()
	pt := pkg.Children["Point"]
	assert.True(t, pt.HasChildren())
	assert.False(t, pt.IsTemp())
	assert.Contains(t, pt.String(), "Point")
	assert.Equal(t, "NewPoint(x, y float64) *Point", pkg.Children["NewPoint"].Label())
	assert.Equal(t, "Origin (Point)", pkg.Children["Origin"].Label())
	fn := NewSymbol("f", token.NameFunction, "", textpos.Region{})
	assert.Equal(t, "f()", fn.Label())
	assert.Equal(t, "Point", pkg.Children["NewPoint"].Children["p"].NonPtrTypeName())

	cl := pt.Clone()
	assert.Equal(t, pt.Name, cl.Name)
	assert.Empty(t, cl.Children, "children are not cloned")

	child := NewSymbol("Z", token.NameField, "", textpos.Region{})
	assert.True(t, pt.AddChild(child))
	assert.Equal(t, 3, child.Index)
	assert.False(t, pt.AddChild(NewSymbol("Z", token.NameField, "", textpos.Region{})), "already there")

	var cp Symbol
	cp.CopyFromSrc(pt)
	assert.Equal(t, pt.Kind, cp.Kind)
	assert.Equal(t, pt.Region, cp.Region)
}

func TestSymbolScopes(t *testing.T) {
	pkg := newTestPkg()
	pt := pkg.Children["Point"]
	sy := NewSymbol("W", token.NameField, "", textpos.Region{})
	var ss SymStack
	assert.False(t, sy.AddScopesStack(ss))
	ss.Push(pkg)
	ss.Push(pt)
	assert.True(t, sy.AddScopesStack(ss))
	assert.Contains(t, pt.Children, "W")
	nm, ok := sy.Scopes.SubCat(token.NameType)
	assert.True(t, ok)
	assert.Equal(t, "Point", nm)
	_, ok = sy.Scopes.SubCat(token.NameFunction)
	assert.False(t, ok)
	assert.Equal(t, sy.Scopes, sy.Scopes.Clone())

	sy2 := NewSymbol("V", token.NameVar, "", textpos.Region{})
	sy2.AddScopesMap(SymMap{"NewPoint": pkg.Children["NewPoint"]}, true)
	assert.Contains(t, pkg.Children["NewPoint"].Children, "V")
	sy2.AddScopesMap(nil, true)

	ty := NewType("T", Struct)
	ty.AddScopesStack(ss)
	assert.Equal(t, "Point", ty.Scopes[token.NameStruct])
	ty.AddScopesStack(nil)
}

func TestSymStack(t *testing.T) {
	var ss SymStack
	assert.Nil(t, ss.Top())
	assert.Nil(t, ss.Pop())
	pkg := newTestPkg()
	ss.Push(pkg)
	fn := ss.PushNew("f", token.NameFunction, "", textpos.Region{})
	assert.Same(t, fn, ss.Top())
	sy, ok := ss.FindNameScoped("Point")
	assert.True(t, ok)
	assert.Equal(t, "Point", sy.Name)
	_, ok = ss.FindNameScoped("Nope")
	assert.False(t, ok)
	assert.Same(t, fn, ss.Pop())
	assert.Same(t, pkg, ss.Top())
	ss.ClearAST()
	ss.Reset()
	assert.Empty(t, ss)
}

func TestSymMapFind(t *testing.T) {
	pkg := newTestPkg()
	sm := SymMap{"pkg": pkg}
	sy, ok := sm.FindName("X")
	assert.True(t, ok, "recursive")
	assert.Equal(t, "X", sy.Name)
	_, ok = sm.FindName("Nope")
	assert.False(t, ok)
	_, ok = sm.FindNameScoped("Point")
	assert.True(t, ok, "in the package")
	_, ok = sm.FindNameScoped("X")
	assert.False(t, ok, "not in a type")
	var nilMap SymMap
	_, ok = nilMap.FindName("X")
	assert.False(t, ok)
	_, ok = nilMap.FindNameScoped("X")
	assert.False(t, ok)

	var matches SymMap
	sm.FindKind(token.NameField, &matches)
	assert.Len(t, matches, 2)
	matches = nil
	sm.FindKindScoped(token.NameFunction, &matches)
	assert.Contains(t, matches, "NewPoint")

	matches = nil
	fpath, _ := filepath.Abs(filepath.Join("pkg", "a.go"))
	sm.FindContainsRegion(fpath, textpos.Pos{Line: 9}, 0, token.None, &matches)
	assert.Contains(t, matches, "Add")
	assert.Contains(t, matches, "pkg")
	matches = nil
	sm.FindContainsRegion(fpath, textpos.Pos{Line: 16}, 2, token.NameFunction, &matches)
	assert.Contains(t, matches, "NewPoint", "with extra lines")
	matches = nil
	sm.FindContainsRegion("/other.go", textpos.Pos{Line: 9}, 0, token.None, &matches)
	assert.Empty(t, matches)

	matches = nil
	pkg.Children.FindNamePrefix("new", &matches)
	assert.Contains(t, matches, "NewPoint", "lower case is case insensitive")
	matches = nil
	pkg.Children.FindNamePrefix("new", &matches)
	pkg.Children.FindNamePrefix("New", &matches)
	assert.Len(t, matches, 1)
	matches = nil
	pkg.Children.FindNamePrefix("Ne", &matches)
	assert.Len(t, matches, 1)
	matches = nil
	pkg.Children.FindNamePrefix("ne", &matches)
	assert.Len(t, matches, 1)
	matches = nil
	pkg.Children.FindNamePrefix("NE", &matches)
	assert.Empty(t, matches, "upper case is case sensitive")
	matches = nil
	pkg.Children.FindNamePrefixRecursive("a", &matches)
	assert.Contains(t, matches, "Add")
	matches = nil
	imp := SymMap{`"cogentcore.org/core/text"`: NewSymbol(`"cogentcore.org/core/text"`, token.NameLibrary, "", textpos.Region{}), "pkg": pkg}
	imp.FindNamePrefixScoped("te", &matches)
	assert.Contains(t, matches, "text", "the package name of an import path")
	matches = nil
	imp.FindNamePrefixScoped("Or", &matches)
	assert.Contains(t, matches, "Origin", "in the package")
}

// TestFindNamePrefixScopedKeys tests that the matches of a case-insensitive
// prefix are keyed by their names, as in the other find functions.
func TestFindNamePrefixScopedKeys(t *testing.T) {
	pkg := newTestPkg()
	var matches SymMap
	sm := SymMap{"pkg": pkg}
	sm.FindNamePrefixScoped("or", &matches)
	assert.Contains(t, matches, "Origin")
}

func TestSymMapOps(t *testing.T) {
	pkg := newTestPkg()
	sm := pkg.Children
	assert.Equal(t, []string{"NewPoint", "Origin", "Point"}, sm.Names(true))
	assert.Len(t, sm.KindNames(true), 3)
	sl := sm.Slice(true)
	assert.Equal(t, "NewPoint", sl[0].Name)
	one := SymMap{"a": NewSymbol("a", token.Name, "", textpos.Region{})}
	assert.Equal(t, "a", one.First().Name)
	var empty SymMap
	assert.Nil(t, empty.First())
	empty.Add(NewSymbol("b", token.Name, "", textpos.Region{}))
	assert.Len(t, empty, 1)
	empty.Reset()
	assert.Empty(t, empty)

	// copying merges the children of symbols with the same name
	other := SymMap{}
	op := other.AddNew("Point", token.NameStruct, "b.go", region(1, 2))
	op.Children.AddNew("Z", token.NameField, "b.go", region(1, 1))
	other.AddNew("Extra", token.NameVar, "b.go", region(3, 3))
	sm.CopyFrom(other, true)
	assert.Contains(t, sm, "Extra")
	assert.Contains(t, sm["Point"].Children, "Z")
	assert.Contains(t, sm["Point"].Children, "X")
	typed := SymMap{}
	typed.AddNew("Origin", token.NameVarGlobal, "b.go", region(1, 1))
	sm.CopyFrom(typed, false)
	assert.Equal(t, "Point", sm["Origin"].Type, "keeps the typed one")
	untyped := SymMap{"Origin": NewSymbol("Origin", token.NameVarGlobal, "", textpos.Region{})}
	untyped.CopyFrom(SymMap{"Origin": sm["Origin"]}, true)
	assert.Equal(t, "Point", untyped["Origin"].Type, "takes the typed one")

	var buf bytes.Buffer
	sm.WriteDoc(&buf, 0)
	assert.Contains(t, buf.String(), "NewPoint")
	pkg.ClearAST()
}

func TestSymbolFindAnyChildren(t *testing.T) {
	pkg := newTestPkg()
	origin := pkg.Children["Origin"]
	kids := SymMap{}
	assert.True(t, origin.FindAnyChildren("", pkg.Children, nil, &kids), "children of its type")
	assert.Len(t, kids, 3)
	kids = SymMap{}
	assert.True(t, origin.FindAnyChildren("X", nil, pkg.Children, &kids))
	assert.Len(t, kids, 1)
	kids = SymMap{}
	origin.Type = "Nope"
	assert.False(t, origin.FindAnyChildren("", pkg.Children, nil, &kids))
}

func TestSymbolJSON(t *testing.T) {
	pkg := newTestPkg()
	dir := t.TempDir()
	fn := filepath.Join(dir, "pkg.json")
	require.NoError(t, pkg.SaveJSON(fn))
	var sy Symbol
	require.NoError(t, sy.OpenJSON(fn))
	assert.Equal(t, "pkg", sy.Name)
	assert.Contains(t, sy.Children["Point"].Children, "X")
	assert.Contains(t, sy.Types, "Point")

	mfn := filepath.Join(dir, "map.json")
	require.NoError(t, pkg.Children.SaveJSON(mfn))
	var sm SymMap
	require.NoError(t, sm.OpenJSON(mfn))
	assert.Contains(t, sm, "NewPoint")
	assert.Error(t, sm.OpenJSON(filepath.Join(dir, "nope.json")))

	var to Symbol
	to.Children = SymMap{}
	to.CopyFromScope(pkg)
	assert.Contains(t, to.Children, "Point")
	assert.Contains(t, to.Types, "Point")
}

func TestSymCache(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cdir, err := ParseCacheDir(fileinfo.Go)
	require.NoError(t, err)
	assert.DirExists(t, cdir)

	fn := filepath.Join(t.TempDir(), "x", "a.go")
	cfn, err := CacheFilename(fileinfo.Go, fn)
	require.NoError(t, err)
	assert.Equal(t, cdir, filepath.Dir(cfn))
	pkg := newTestPkg()
	require.NoError(t, SaveSymCache(pkg, fileinfo.Go, fn))
	require.NoError(t, SaveSymDoc(pkg, fileinfo.Go, fn))
	sy, tm, err := OpenSymCache(fileinfo.Go, fn)
	require.NoError(t, err)
	assert.False(t, tm.IsZero())
	assert.Equal(t, "pkg", sy.Name)
	_, _, err = OpenSymCache(fileinfo.Go, filepath.Join(t.TempDir(), "y", "b.go"))
	assert.True(t, os.IsNotExist(err))

	rel, err := GoRelPath(filepath.Join(os.Getenv("HOME"), "src", "a.go"))
	require.NoError(t, err)
	assert.NotEmpty(t, rel)
}

func TestType(t *testing.T) {
	pkg := newTestPkg()
	ty := pkg.Types["Point"]
	assert.Equal(t, "Point: Struct", ty.String())
	assert.Equal(t, "X", ty.Els.ByName("X").Name)
	assert.Nil(t, ty.Els.ByName("Z"))
	assert.Equal(t, "X", ty.Els.ByType("float64").Name)
	assert.Nil(t, ty.Els.ByType("int"))
	assert.Equal(t, "X float64, Y float64", ty.Els.String())
	assert.Equal(t, "X float64", ty.Els.StringRange(0, 1))
	assert.Equal(t, "X float64", ty.Els[0].String())

	fn := NewType("f", Func)
	fn.Els.Add("a", "int")
	fn.Els.Add("", "error")
	fn.Size = []int{1, 1}
	assert.Equal(t, "func f(a int) error", fn.String())
	assert.Equal(t, "a int", fn.ArgString())
	assert.Equal(t, "error", fn.ReturnString())
	fn.Els.Add("", "bool")
	fn.Size = []int{1, 2}
	assert.Equal(t, "(error, bool)", fn.ReturnString())
	assert.Contains(t, fn.String(), "(error, bool)")
	fn.Size = []int{1, 0}
	assert.Equal(t, "", fn.ReturnString())

	meth := ty.Meths["Point:Add"]
	assert.Equal(t, "func (p *Point) Point:Add(o Point)", meth.String())
	assert.Equal(t, "o Point", meth.ArgString())
	assert.Equal(t, "", ty.ArgString(), "not a function")
	assert.Equal(t, "", ty.ReturnString())

	mp := NewType("m", Map)
	mp.Els.Add("", "string")
	mp.Els.Add("", "int")
	assert.Equal(t, "map[string]int", mp.String())
	ls := NewType("l", List)
	ls.Els.Add("", "int")
	assert.Equal(t, "[]int", ls.String())
	assert.Equal(t, "string", NewType("s", String).String())

	ptr := NewType("*Point", Ptr)
	ptr.Els.Add("", "Point")
	assert.Equal(t, "Point", ptr.NonPtrType())
	assert.Equal(t, "Point", ty.NonPtrType())

	cl := ty.Clone()
	assert.Equal(t, ty.Els, cl.Els)
	cl.Els[0].Name = "Q"
	assert.Equal(t, "X", ty.Els[0].Name, "deep copy")
	var els TypeEls
	els.CopyFrom(ty.Els)
	assert.Equal(t, ty.Els, els)

	var buf bytes.Buffer
	ty.WriteDoc(&buf, 0)
	fn.WriteDoc(&buf, 0)
	assert.Contains(t, buf.String(), "Point")
	ty.ClearAST()
}

func TestTypeMap(t *testing.T) {
	tm := TypeMap{}
	tm.Add(NewType("A", Struct))
	tm.Add(NewType("B", Unknown))
	assert.Equal(t, []string{"A", "B"}, tm.Names(true))
	assert.Len(t, tm.KindNames(true), 2)
	src := TypeMap{"B": NewType("B", Int), "C": NewType("C", Int)}
	tm.CopyFrom(src, true)
	assert.Equal(t, Int, tm["B"].Kind, "a known kind replaces an unknown one")
	assert.Contains(t, tm, "C")
	tm.CopyFrom(TypeMap{"A": NewType("A", Unknown)}, false)
	assert.Equal(t, Struct, tm["A"].Kind)
	cl := tm.Clone()
	assert.Len(t, cl, 3)
	var empty TypeMap
	assert.Nil(t, empty.Clone())
	var buf bytes.Buffer
	tm.WriteDoc(&buf, 0)
	assert.Contains(t, buf.String(), "A")
	tm.PrintUnknowns()
	tm.ClearAST()
}

func TestKinds(t *testing.T) {
	assert.Equal(t, Primitive, Int.Cat())
	assert.Equal(t, Numeric, Int.SubCat())
	assert.Equal(t, Integer, Int.Sub2Cat())
	assert.True(t, Primitive.IsCat())
	assert.True(t, Numeric.IsSubCat())
	assert.True(t, Integer.IsSub2Cat())
	assert.False(t, Int.IsCat())
	assert.True(t, Int.InCat(Primitive))
	assert.True(t, Int.InSubCat(Numeric))
	assert.True(t, Int.InSub2Cat(Integer))
	assert.True(t, Ptr.IsPtr())
	assert.True(t, Int.IsPrimitiveNonPtr())
	assert.False(t, Ptr.IsPrimitiveNonPtr())
	assert.Equal(t, Function, Method.Cat())
}

func TestComplete(t *testing.T) {
	pkg := newTestPkg()
	var md complete.Matches
	AddCompleteSyms(pkg.Children, "pkg", &md)
	require.Len(t, md.Matches, 3)
	assert.Equal(t, "NewPoint()", md.Matches[0].Text)
	assert.Equal(t, "NewPoint(x, y float64) *Point (pkg.)", md.Matches[0].Label)
	AddCompleteSyms(nil, "", &md)
	AddCompleteSyms(SymMap{"_x": NewSymbol("_x", token.Name, "", textpos.Region{})}, "", &md)
	assert.Len(t, md.Matches, 3, "internal names are skipped")

	md = complete.Matches{}
	AddCompleteSymsPrefix(pkg.Children, "", "Or", &md)
	assert.Equal(t, "Or", md.Seed)
	assert.Len(t, md.Matches, 1)
	md = complete.Matches{}
	AddCompleteSymsPrefix(pkg.Children, "", "", &md)
	assert.Len(t, md.Matches, 3)

	md = complete.Matches{}
	AddCompleteTypeNames(pkg.Types["Point"], "Point", "", &md)
	var texts []string
	for _, m := range md.Matches {
		texts = append(texts, m.Text)
	}
	assert.Equal(t, []string{"X", "Y", "Point:Add()"}, texts)
	md = complete.Matches{}
	AddCompleteTypeNames(pkg.Types["Point"], "", "Y", &md)
	assert.Len(t, md.Matches, 1)
}
