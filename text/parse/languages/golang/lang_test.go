// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/base/goscript"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/parse"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/parse/syms"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSrc is a small Go package used to test parsing, type resolution,
// completion, and lookup.
const testSrc = `package shapes

// Shape is a geometric shape.
type Shape interface {
	Area() float64
}

// Point is a point in the plane.
type Point struct {
	X, Y float64
}

// Rect is a rectangle.
type Rect struct {
	Point
	Size  Point
	Name  string
	Tags  []string
	Props map[string]int
}

// Area returns the area of the rectangle.
func (r *Rect) Area() float64 {
	return r.Size.X * r.Size.Y
}

// Scale scales the rectangle.
func (r *Rect) Scale(f float64) {
	r.Size.X *= f
	r.Size.Y *= f
}

const Pi = 3.14159

var Default = Rect{Name: "default"}

// NewRect returns a new rectangle.
func NewRect(name string, w, h float64) (*Rect, error) {
	r := &Rect{Name: name}
	r.Size.X = w
	r.Size.Y = h
	for i, t := range r.Tags {
		_ = i
		_ = t
	}
	total := 0
	for k, v := range r.Props {
		_ = k
		total += v
	}
	if total > 0 {
		return r, nil
	}
	return nil, nil
}
`

// parseSync parses the given source as a file in a new directory, as
// [GoLang.ParseFile] does, but synchronously, including the resolution of
// the types of the symbols in functions that ParseFile does in a goroutine.
func parseSync(t *testing.T, src string) (*parse.FileStates, *parse.FileState, *syms.Symbol) {
	t.Helper()
	dir := t.TempDir()
	fn := filepath.Join(dir, "shapes.go")
	require.NoError(t, os.WriteFile(fn, []byte(src), 0666))
	fss := parse.NewFileStates(fn, "", fileinfo.Go)
	pr := TheGoLang.Parser()
	pfs := fss.StartProc([]byte(src))
	pr.LexAll(pfs)
	pr.ParseAll(pfs)
	fss.EndProc()
	require.NotEmpty(t, pfs.ParseState.Scopes)
	pkg := pfs.ParseState.Scopes[0]
	pfs.Syms[pkg.Name] = pkg
	pfs.WaitGp.Add(1)
	TheGoLang.AddPathToSyms(pfs, dir)
	TheGoLang.AddImportsToExts(fss, pfs, pkg)
	return fss, pfs, pkg
}

// TestResolveLocalTypes tests that the types of the receivers and local
// variables of functions are resolved.
func TestResolveLocalTypes(t *testing.T) {
	_, _, pkg := parseSync(t, importSrc())
	rect := pkg.Children["Rect"]
	require.NotNil(t, rect)
	assert.Equal(t, "Rect", rect.Children["Area"].Children["r"].Type, "the type of a receiver is its type name")
	nr := pkg.Children["NewRect"]
	assert.ElementsMatch(t, []string{"name string", "w float64", "h float64"}, TheGoLang.FuncParams(nr))
	assert.Equal(t, "*Rect", nr.Children["r"].Type)
	assert.Equal(t, "int", nr.Children["total"].Type)
}

// TestFuncParamsOrder tests that the parameters of a function are in order.
func TestFuncParamsOrder(t *testing.T) {
	_, _, pkg := parseSync(t, importSrc())
	assert.Equal(t, []string{"name string", "w float64", "h float64"}, TheGoLang.FuncParams(pkg.Children["NewRect"]))
}

// TestResolveLocalTypesNoImports tests that the types of the receivers and
// local variables of functions are resolved in a file without imports.
func TestResolveLocalTypesNoImports(t *testing.T) {
	_, _, pkg := parseSync(t, testSrc)
	assert.Equal(t, "Rect", pkg.Children["Rect"].Children["Area"].Children["r"].Type)
}

// importSrc returns [testSrc] with an import, which is needed for the
// types of the symbols in functions to be resolved (see
// TestResolveLocalTypesNoImports). The import adds 4 lines before the
// code of testSrc after its package line.
func importSrc() string {
	return strings.Replace(testSrc, "package shapes\n", "package shapes\n\nimport \"errors\"\n\nvar ErrNone = errors.New(\"none\")\n", 1)
}

// matchTexts returns the texts of the given completion matches.
func matchTexts(md complete.Matches) []string {
	var ts []string
	for _, m := range md.Matches {
		ts = append(ts, m.Text)
	}
	return ts
}

// TestParseFile tests the symbols of a file parsed by ParseFile.
func TestParseFile(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "shapes.go")
	require.NoError(t, os.WriteFile(fn, []byte(testSrc), 0666))
	fss := parse.NewFileStates(fn, "", fileinfo.Go)
	TheGoLang.ParseFile(fss, []byte(testSrc))
	fs := fss.Done()
	fs.WaitGp.Wait()
	fs.SymsMu.Lock()
	defer fs.SymsMu.Unlock()
	pkg := fs.ParseState.Scopes[0]
	assert.Equal(t, "shapes", pkg.Name)
	for _, nm := range []string{"Shape", "Point", "Rect", "NewRect", "Pi", "Default"} {
		assert.Contains(t, pkg.Children, nm)
	}
	rect := pkg.Types["Rect"]
	require.NotNil(t, rect)
	assert.Equal(t, syms.Struct, rect.Kind)
	assert.Contains(t, rect.Meths, "Area")
	assert.Contains(t, rect.Meths, "Scale")
	var els []string
	for _, e := range rect.Els {
		els = append(els, e.Name+" "+e.Type)
	}
	assert.Equal(t, []string{"Point Point", "X float64", "Y float64", "Size Point", "Name string", "Tags []string", "Props map[string]int"}, els, "with the inherited fields of Point")
	assert.Equal(t, syms.Interface, pkg.Types["Shape"].Kind)
	assert.Equal(t, token.NameConstant, pkg.Children["Pi"].Kind)
	assert.Equal(t, "Rect", pkg.Children["Default"].Type)

	// a go.mod file is not parsed
	mfss := parse.NewFileStates(filepath.Join(dir, "go.mod"), "", fileinfo.Go)
	TheGoLang.ParseFile(mfss, []byte("module x\n"))
	assert.Empty(t, mfss.Done().ParseState.Scopes)
}

// TestCompleteLine tests completing names, fields, and methods.
func TestCompleteLine(t *testing.T) {
	goscript.Skip(t, "the sources of the Go standard library")
	fss, _, _ := parseSync(t, importSrc())
	inScale := textpos.Pos{Line: 32, Char: 4}   // in Scale
	inNewRect := textpos.Pos{Line: 50, Char: 4} // in NewRect
	tests := []struct {
		str  string
		pos  textpos.Pos
		seed string
		want []string
	}{
		{"r.Si", inScale, "Si", []string{"Size"}},
		{"r.Size.", inScale, "", []string{"X", "Y"}},
		{"r.Sc", inNewRect, "Sc", []string{"Scale()"}},
		{"Default.N", inNewRect, "N", []string{"Name"}},
		{"Point.", inNewRect, "", []string{"X", "Y"}},
		{"Re", inNewRect, "Re", []string{"Rect"}},
		{"tot", inNewRect, "tot", []string{"total"}},
		{"appe", inNewRect, "appe", []string{"append()"}},
		{"errors.N", inNewRect, "N", []string{"New()"}},
	}
	for _, tt := range tests {
		md := TheGoLang.CompleteLine(fss, tt.str, tt.pos)
		assert.Equal(t, tt.seed, md.Seed, tt.str)
		for _, w := range tt.want {
			assert.Contains(t, matchTexts(md), w, tt.str)
		}
	}
	md := TheGoLang.CompleteLine(fss, "fl", inNewRect)
	assert.Subset(t, matchTexts(md), []string{"float32", "float64"})
	assert.Empty(t, TheGoLang.CompleteLine(fss, "", inNewRect).Matches)
	assert.Empty(t, TheGoLang.CompleteLine(fss, "f(x)", inNewRect).Matches, "not after a closing paren")
}

// TestLookup tests looking up the definitions of names.
func TestLookup(t *testing.T) {
	fss, _, _ := parseSync(t, importSrc())
	pos := textpos.Pos{Line: 50, Char: 4}
	tests := []struct {
		str    string
		st, ed int
	}{
		{"NewRect", 41, 58},
		{"Rect", 17, 17},
		{"Pi", 36, 36},
		{"Point", 12, 12},
		{"r.Area", 26, 28},
	}
	for _, tt := range tests {
		ld := TheGoLang.Lookup(fss, tt.str, pos)
		assert.NotEmpty(t, ld.Filename, tt.str)
		assert.Equal(t, tt.st, ld.StLine, tt.str)
		assert.Equal(t, tt.ed, ld.EdLine, tt.str)
	}
	assert.Empty(t, TheGoLang.Lookup(fss, "", pos).Filename)
}

// TestCompleteEdit tests the edits for completions.
func TestCompleteEdit(t *testing.T) {
	ed := TheGoLang.CompleteEdit(nil, "x.ChildByName()", 7, complete.Completion{Text: "Children"}, "Child")
	assert.Equal(t, "Children", ed.NewText)
	assert.Equal(t, 6, ed.ForwardDelete, "the rest of the name")
	ed = TheGoLang.CompleteEdit(nil, "x.Fo(a)", 3, complete.Completion{Text: "Foo()"}, "F")
	assert.Equal(t, "Foo", ed.NewText, "the parens are already there")
	assert.Equal(t, 1, ed.ForwardDelete)
	ed = TheGoLang.CompleteEdit(nil, "x.Fo", 4, complete.Completion{Text: "Foo()"}, "Fo")
	assert.Equal(t, "Foo()", ed.NewText)
}

// TestCompleteEditParenAfterCursor tests completing a name that is
// followed by its arguments, with the cursor at the end of the name.
func TestCompleteEditParenAfterCursor(t *testing.T) {
	ed := TheGoLang.CompleteEdit(nil, "x.Fo(a)", 4, complete.Completion{Text: "Foo()"}, "Fo")
	assert.Equal(t, "Foo", ed.NewText)
}

// TestIndentLine tests the indentation of lines after brackets and cases.
func TestIndentLine(t *testing.T) {
	src := "func f() {\n\tx := 1\n\tswitch x {\n\tcase 1:\n\t\treturn\n\t\tdefault:\n\t}\n}"
	fss := parse.NewFileStates("f.go", "", fileinfo.Go)
	pr := TheGoLang.Parser()
	fs := fss.StartProc([]byte(src))
	pr.LexAll(fs)
	fss.EndProc()
	lines := fs.Src.Lines
	tags := fs.Src.Lexs
	delta := func(ln int) int {
		_, del, _, _ := TheGoLang.IndentLine(fss, lines, tags, ln, 4)
		return del
	}
	assert.Equal(t, 1, delta(1), "after an opening brace")
	assert.Equal(t, 0, delta(3), "case after switch")
	assert.Equal(t, 1, delta(4), "after a case")
	assert.Equal(t, -1, delta(5), "default after a statement")
}

// TestAutoBracket tests matching brackets as they are typed.
func TestAutoBracket(t *testing.T) {
	ln := []rune("if x ")
	match, nl := TheGoLang.AutoBracket(nil, '{', textpos.Pos{Char: 5}, ln)
	assert.True(t, match)
	assert.True(t, nl, "a new line after a space at the end")
	match, nl = TheGoLang.AutoBracket(nil, '{', textpos.Pos{Char: 1}, ln)
	assert.False(t, match, "not before text")
	assert.False(t, nl)
	match, _ = TheGoLang.AutoBracket(nil, '(', textpos.Pos{Char: 4}, ln)
	assert.True(t, match, "before a space")
	match, _ = TheGoLang.AutoBracket(nil, '(', textpos.Pos{Char: 0}, ln)
	assert.False(t, match)
}

// TestHighlightLine tests lexing and parsing a line for highlighting.
func TestHighlightLine(t *testing.T) {
	fss, _, _ := parseSync(t, importSrc())
	ll := TheGoLang.HighlightLine(fss, 4, []rune("const Pi = 3.14159 // pi"))
	require.NotEmpty(t, ll)
	assert.Equal(t, token.Keyword, ll[0].Token.Token.Cat())
	assert.Equal(t, token.Comment, ll[len(ll)-1].Token.Token.Cat())
	lfs := TheGoLang.ParseLine(fss.Done(), 0)
	assert.NotNil(t, lfs)
	assert.NotEmpty(t, TheGoLang.LexLine(fss.Done(), 0, []rune("x := 1")))
}

// TestTypeNames tests the helpers for qualified type names.
func TestTypeNames(t *testing.T) {
	assert.True(t, IsQualifiedType("fmt.Stringer"))
	assert.False(t, IsQualifiedType("[]fmt.Stringer"))
	assert.False(t, IsQualifiedType("map[string]fmt.Stringer"))
	assert.Equal(t, "pkg.T", QualifyType("pkg", "T"))
	assert.Equal(t, "int", QualifyType("pkg", "int"), "builtin")
	assert.Equal(t, "other.T", QualifyType("pkg", "other.T"))
	assert.Equal(t, "T", QualifyType("", "T"))
	p, n := SplitType("pkg.T")
	assert.Equal(t, "pkg", p)
	assert.Equal(t, "T", n)
	p, n = SplitType("T")
	assert.Equal(t, "", p)
	assert.Equal(t, "T", n)
	assert.Equal(t, "pkg.*T", PrefixType("*", "pkg.T"))
	path, base, pkg := TheGoLang.ImportPathPkg(`"cogentcore.org/core/text"`)
	assert.Equal(t, "cogentcore.org/core/text", path)
	assert.Equal(t, "cogentcore.org/core/", base)
	assert.Equal(t, "text", pkg)
	path, _, _ = TheGoLang.ImportPathPkg("x")
	assert.Equal(t, "", path, "too short")
}
