// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/parse"
	_ "cogentcore.org/core/text/parse/languages/golang"
	"cogentcore.org/core/text/parse/syms"
	"cogentcore.org/core/text/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// goSrc is a small Go source file for the tests.
const goSrc = `package shapes

// Point is a point.
type Point struct {
	X, Y float64
}

func (p *Point) Add(o Point) {
	p.X += o.X
}

var Origin Point
`

// goParser returns the standard Go parser.
func goParser(t *testing.T) *parse.Parser {
	t.Helper()
	require.NoError(t, parse.LanguageSupport.OpenLanguage(fileinfo.Go))
	lp, err := parse.LanguageSupport.Properties(fileinfo.Go)
	require.NoError(t, err)
	require.NotNil(t, lp.Parser)
	return lp.Parser
}

// parseGo lexes and parses goSrc into a new file state.
func parseGo(t *testing.T) (*parse.FileStates, *parse.FileState) {
	t.Helper()
	pr := goParser(t)
	fss := parse.NewFileStates("shapes.go", "", fileinfo.Go)
	fs := fss.StartProc([]byte(goSrc))
	pr.LexAll(fs)
	pr.ParseAll(fs)
	fss.EndProc()
	return fss, fs
}

func TestLanguageSupport(t *testing.T) {
	lp, err := parse.LanguageSupport.PropertiesByName("Go")
	require.NoError(t, err)
	assert.Equal(t, fileinfo.Go, lp.Known)
	assert.Equal(t, "// ", lp.CommentLn)
	assert.NotNil(t, lp.Lang)
	assert.False(t, lp.HasFlag(parse.IndentSpace))

	md, err := parse.LanguageSupport.Properties(fileinfo.Markdown)
	require.NoError(t, err)
	assert.True(t, md.HasFlag(parse.IndentSpace))

	_, err = parse.LanguageSupport.Properties(fileinfo.Known(-5))
	assert.Error(t, err)
	_, err = parse.LanguageSupport.PropertiesByName("no-such-language")
	assert.Error(t, err)

	assert.NotNil(t, parse.LanguageSupport.LexerByName("Go"))
	assert.Nil(t, parse.LanguageSupport.LexerByName("no-such-language"))
	assert.Nil(t, parse.LanguageSupport.LexerByName("Python"), "no parser")
	assert.NoError(t, parse.LanguageSupport.OpenLanguage(fileinfo.Python), "nothing to open")
}

func TestParseAll(t *testing.T) {
	_, fs := parseGo(t)
	assert.True(t, fs.LexAtEnd())
	assert.True(t, fs.ParseAtEnd())
	assert.False(t, fs.LexHasErrs(), fs.LexErrReport())
	assert.False(t, fs.PassTwoHasErrs(), fs.PassTwoErrReport())
	assert.False(t, fs.ParseHasErrs(), fs.ParseErrReportAll())
	assert.Equal(t, "", fs.ParseErrReport())
	assert.Equal(t, "", fs.ParseErrReportDetailed())
	require.NotEmpty(t, fs.ParseState.Scopes)
	pkg := fs.ParseState.Scopes[0]
	assert.Equal(t, "shapes", pkg.Name)
	assert.Contains(t, pkg.Children, "Point")
	assert.Contains(t, pkg.Children, "Origin")
	assert.NotEmpty(t, fs.LexLine(3))
	assert.Contains(t, fs.AST.String(), "")
}

func TestParseErrors(t *testing.T) {
	pr := goParser(t)
	fs := pr.ParseString("package x\nfunc f( {\n", "bad.go", fileinfo.Go)
	require.NotNil(t, fs)
	assert.True(t, fs.ParseHasErrs())
	assert.NotEmpty(t, fs.ParseErrReport())
	assert.NotEmpty(t, fs.ParseErrReportAll())
	assert.NotEmpty(t, fs.ParseErrReportDetailed())
	assert.Nil(t, pr.ParseString("", "empty.go", fileinfo.Go))
	assert.Nil(t, pr.ParseString("   ", "space.go", fileinfo.Go), "no tokens")
}

func TestParseString(t *testing.T) {
	pr := goParser(t)
	fs := pr.ParseString("x.Y(1, 2)", "expr.go", fileinfo.Go)
	require.NotNil(t, fs)
	assert.True(t, fs.AST.HasChildren())
	assert.NotEmpty(t, fs.ParseRuleString(true))
	assert.NotEmpty(t, fs.ParseRuleString(false))
}

func TestLexAndParseLine(t *testing.T) {
	pr := goParser(t)
	_, fs := parseGo(t)
	ll := pr.LexLine(fs, 2, []rune("// a comment"))
	require.Len(t, ll, 1)
	assert.Equal(t, token.Comment, ll[0].Token.Token.Cat())
	ll = pr.LexLine(fs, 12, []rune("var Origin Point"))
	assert.Len(t, ll, 3)
	assert.Nil(t, pr.LexLine(fs, 100, []rune("x")), "out of range")
	lfs := pr.ParseLine(fs, 12)
	require.NotNil(t, lfs)
	assert.True(t, lfs.AST.HasChildren())
	assert.Nil(t, pr.ParseLine(fs, -1))
	assert.Nil(t, pr.ParseLine(fs, 100))
}

func TestLexStepping(t *testing.T) {
	pr := goParser(t)
	fs := parse.NewFileState()
	fs.SetSrc([][]rune{[]rune("package x"), []rune("var y = 1")}, "x.go", "", fileinfo.Go)
	pr.LexInit(fs)
	n := 0
	for !fs.LexAtEnd() && n < 100 {
		pr.LexNext(fs)
		n++
	}
	assert.True(t, fs.LexAtEnd())
	assert.NotEmpty(t, fs.Src.Lexs[1])

	fs.SetSrc([][]rune{[]rune("package x"), []rune("var y = 1")}, "x.go", "", fileinfo.Go)
	pr.LexInit(fs)
	pr.LexNextLine(fs)
	assert.Equal(t, "var y = 1", fs.LexNextSrcLine())
	pr.LexRun(fs)
	pr.DoPassTwo(fs)
	assert.True(t, pr.ParserInit(fs))
	pr.ParseNext(fs)
	pr.ParseRun(fs)
	assert.True(t, fs.ParseAtEnd())
	fs.Destroy()
}

func TestFileStates(t *testing.T) {
	fss := parse.NewFileStates("a.go", "/base", fileinfo.Go)
	assert.Equal(t, "a.go", fss.Filename)
	assert.Equal(t, fileinfo.Go, fss.Known)
	done, proc := fss.Done(), fss.Proc()
	assert.NotSame(t, done, proc)
	assert.Same(t, done, fss.DoneNoLock())
	assert.Same(t, proc, fss.ProcNoLock())
	pfs := fss.StartProc([]byte("package a\n"))
	assert.Same(t, proc, pfs)
	assert.Equal(t, "/base", pfs.Src.BasePath)
	fss.EndProc()
	assert.Same(t, proc, fss.Done(), "switched")

	_, ok := fss.MetaData("k")
	assert.False(t, ok)
	fss.SetMetaData("k", "v")
	v, ok := fss.MetaData("k")
	assert.True(t, ok)
	assert.Equal(t, "v", v)
	fss.DeleteMetaData("k")
	_, ok = fss.MetaData("k")
	assert.False(t, ok)

	fss.SetSrc("b.md", "", fileinfo.Markdown)
	assert.Equal(t, "b.md", fss.Filename)
	assert.Equal(t, "b.md", fss.Done().Src.Filename)
}

func TestFileStateSymbols(t *testing.T) {
	_, fs := parseGo(t)
	pkg := fs.ParseState.Scopes[0]
	fs.Syms[pkg.Name] = pkg
	sy, ok := fs.FindNameScoped("Point", nil)
	require.True(t, ok)
	assert.Equal(t, token.NameType, sy.Kind)
	_, ok = fs.FindNameScoped("Nope", nil)
	assert.False(t, ok)
	_, ok = fs.FindNameScoped("Add", sy.Children)
	assert.True(t, ok, "in the given scope")

	kids := syms.SymMap{}
	assert.True(t, fs.FindChildren(sy, "", nil, &kids))
	assert.Contains(t, kids, "X")
	kids = syms.SymMap{}
	assert.True(t, fs.FindChildren(sy, "Y", nil, &kids))
	assert.Len(t, kids, 1)
	kids = syms.SymMap{}
	origin := pkg.Children["Origin"]
	origin.Type = "*Point"
	assert.True(t, fs.FindAnyChildren(origin, "A", nil, &kids), "the children of its type")
	assert.Contains(t, kids, "Add")
	kids = syms.SymMap{}
	origin.Type = "Nope"
	assert.False(t, fs.FindChildren(origin, "", nil, &kids))

	var matches syms.SymMap
	fs.FindNamePrefixScoped("Or", nil, &matches)
	assert.Contains(t, matches, "Origin")
	matches = nil
	fs.FindNamePrefixScoped("A", sy.Children, &matches)
	assert.Contains(t, matches, "Add")

	assert.Equal(t, "anon_shapes_1", fs.NextAnonName("shapes"))
	assert.Equal(t, "anon_shapes_2", fs.NextAnonName("shapes"))

	_, ok = fs.PathMapLoad("x")
	assert.False(t, ok)
	fs.PathMapStore("x", "/abs/x")
	abs, ok := fs.PathMapLoad("x")
	assert.True(t, ok)
	assert.Equal(t, "/abs/x", abs)
}

func TestSaveOpenParser(t *testing.T) {
	pr := goParser(t)
	dir := t.TempDir()
	fn := filepath.Join(dir, "go.parse")
	require.NoError(t, pr.SaveJSON(fn))
	npr := parse.NewParser()
	require.NoError(t, npr.OpenJSON(fn))
	npr.InitAll()
	fs := npr.ParseString("package x\nvar y int\n", "x.go", fileinfo.Go)
	require.NotNil(t, fs)
	assert.False(t, fs.ParseHasErrs())

	gfn := filepath.Join(dir, "go.parsegrammar")
	require.NoError(t, pr.SaveGrammar(gfn))
	b, err := os.ReadFile(gfn)
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(b), "Lexer"))
	assert.Error(t, pr.SaveGrammar(filepath.Join(dir, "no", "such", "dir")))
	assert.Error(t, npr.OpenJSON(filepath.Join(dir, "nope.parse")))
}
