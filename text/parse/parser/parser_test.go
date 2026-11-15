// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/parse"
	_ "cogentcore.org/core/text/parse/languages/golang"
	"cogentcore.org/core/text/parse/parser"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// richSrc is Go source with many kinds of declarations, statements, and
// expressions, which exercises much of the Go grammar and its actions.
const richSrc = `package rich

import (
	"fmt"
	str "strings"
)

type (
	// Shape is a shape.
	Shape interface {
		Area() float64
		fmt.Stringer
	}

	Point struct {
		X, Y float64 ` + "`json:\"x\"`" + `
		tags []string
	}

	Handler func(a, b int) (int, error)
	Grid    [4][4]*Point
	Lookup  map[string]chan<- int
)

const (
	A = iota
	B
	C int = 3 << 2
)

var (
	x, y   = 1, "two"
	z      float32
	fn     = func(i int) bool { return i > 0 }
	points = []Point{{X: 1}, {2, 3, nil}}
)

func (p *Point) Area() float64 { return p.X * p.Y }

func Sum(vals ...float64) (total float64) {
	for _, v := range vals {
		total += v
	}
	return
}

func run(ch chan int, done <-chan struct{}) error {
	defer close(ch)
	m := map[string]int{"a": 1}
	if v, ok := m["a"]; ok && v != 0 {
		m["b"] = v + 1
	} else if !ok {
		return fmt.Errorf("no a")
	} else {
		delete(m, "a")
	}
	for i := 0; i < 10; i++ {
		switch {
		case i%2 == 0:
			continue
		case i > 8:
			break
		default:
			i++
		}
	}
	var s any = "x"
	switch t := s.(type) {
	case string:
		_ = str.ToUpper(t)
	case nil:
	}
	select {
	case ch <- 1:
	case <-done:
		return nil
	}
	go func() { ch <- 2 }()
	arr := [...]int{1, 2, 3}
	sl := arr[1:2:3]
	p := &Point{X: -1.5e3}
	p.X, p.Y = p.Y, p.X
	_ = sl[len(sl)-1] &^ 0x0F | 1
	goto end
end:
	return nil
}
`

// goParser returns the standard Go parser.
func goParser(t *testing.T) *parse.Parser {
	t.Helper()
	require.NoError(t, parse.LanguageSupport.OpenLanguage(fileinfo.Go))
	lp, _ := parse.LanguageSupport.Properties(fileinfo.Go)
	require.NotNil(t, lp.Parser)
	return lp.Parser
}

// parseRich parses richSrc with the Go parser.
func parseRich(t *testing.T) *parse.FileState {
	t.Helper()
	fs := goParser(t).ParseString(richSrc, "rich.go", fileinfo.Go)
	require.NotNil(t, fs)
	return fs
}

// astNames returns the names of all of the nodes of the given AST.
func astNames(ast *parser.AST) []string {
	var names []string
	for n := ast; n != nil; n = n.NextAST() {
		names = append(names, n.Name)
	}
	return names
}

func TestParseRich(t *testing.T) {
	fs := parseRich(t)
	assert.False(t, fs.ParseHasErrs(), fs.ParseErrReportAll())
	names := astNames(fs.AST)
	for _, nm := range []string{"ImportAlias", "TypeDeclEl", "ConstSpec", "VarSpec", "MethDecl", "FuncDecl", "IfStmtInit", "ElseIfStmt", "ForRangeNew", "ForClauseStmt", "SwitchExpr", "SwitchTypeName", "SelectStmt", "SelCaseSendStmt", "GoStmt", "DeferStmt", "GotoStmt", "LabeledStmt", "CompositeLit", "SliceThree", "FuncLit", "MapType", "ArrayType", "InterfaceType", "StructType", "FuncType", "FieldTag", "BitAndNotExpr", "ParNameEllipsis"} {
		found := false
		for _, n := range names {
			if strings.HasPrefix(n, nm) {
				found = true
				break
			}
		}
		assert.True(t, found, "AST has a %s", nm)
	}
	pkg := fs.ParseState.Scopes[0]
	assert.Equal(t, "rich", pkg.Name)
	for _, nm := range []string{"Shape", "Point", "Handler", "Grid", "Lookup", "A", "B", "C", "x", "y", "z", "fn", "points", "Sum", "run"} {
		assert.Contains(t, pkg.Children, nm)
	}
	assert.Contains(t, pkg.Children["Point"].Children, "Area")
}

func TestAST(t *testing.T) {
	fs := parseRich(t)
	root := fs.AST
	first := root.ChildAST(0)
	require.NotNil(t, first)
	assert.Same(t, root, first.ParentAST())
	assert.Nil(t, root.ParentAST())
	assert.Same(t, first, root.NextAST())
	assert.Nil(t, root.ChildASTTry(-1))
	assert.Nil(t, root.ChildASTTry(100000))
	assert.Same(t, first, root.ChildASTTry(0))
	if root.NumChildren() > 1 {
		assert.Same(t, root.ChildAST(1), first.NextSiblingAST())
	}
	last := root
	for n := root.NextAST(); n != nil; n = n.NextAST() {
		last = n
	}
	assert.Nil(t, last.NextAST())
	prev := last.PrevAST()
	require.NotNil(t, prev)
	assert.Same(t, last, prev.NextAST())
	assert.Nil(t, root.PrevAST())

	var buf bytes.Buffer
	root.WriteTree(&buf, 0)
	assert.Contains(t, buf.String(), "FuncDecl")
	assert.Contains(t, buf.String(), "Point")

	n := parser.NewAST()
	n.SetTokReg(textpos.Region{Start: textpos.Pos{Line: 0, Char: 0}, End: textpos.Pos{Line: 0, Char: 2}}, &fs.Src)
	assert.NotEmpty(t, n.Src)
	n.SetTokRegEnd(textpos.Pos{Line: 0, Char: 1}, &fs.Src)
	assert.Equal(t, textpos.Pos{Line: 0, Char: 1}, n.TokReg.End)
	n.Destroy()
}

func TestTrace(t *testing.T) {
	pr := goParser(t)
	fs := parse.NewFileState()
	fs.Src.InitFromString("package x\nvar y = 1\n", "x.go", fileinfo.Go)
	fs.ParseState.Init(&fs.Src, fs.AST)
	out, err := os.Create(filepath.Join(t.TempDir(), "trace.txt"))
	require.NoError(t, err)
	tr := &fs.ParseState.Trace
	tr.FullOn()
	tr.OutWrite = out
	tr.Init()
	pr.LexAll(fs)
	fs.Src.EnsureFinalEos(0)
	pr.ParseAll(fs)
	out.Close()
	b, err := os.ReadFile(out.Name())
	require.NoError(t, err)
	assert.Contains(t, string(b), "Match")
	assert.Contains(t, string(b), "VarDecl")

	var ot parser.TraceOptions
	ot.CopyOpts(tr)
	assert.True(t, ot.On)
	assert.True(t, ot.RunAct)
	ot.Rules = "VarDecl TypeDecl"
	assert.True(t, ot.CheckRule("TypeDecl"))
	assert.False(t, ot.CheckRule("FuncDecl"))
	ot.Rules = ""
	ot.Init()
	assert.True(t, ot.CheckRule("FuncDecl"), "all rules")
	ot.PipeOut()
	assert.NotNil(t, ot.OutRead)
	ot.OutWrite.Close()
	ot.OutRead.Close()
	ot.Stdout()
	assert.Same(t, os.Stdout, ot.OutWrite)
	ot.On = false
	assert.False(t, ot.Out(&fs.ParseState, pr.Parser, parser.Match, textpos.Pos{}, textpos.Region{}, fs.AST, ""))
}

func TestState(t *testing.T) {
	fs := parseRich(t)
	ps := &fs.ParseState
	assert.True(t, ps.AtEof())
	all := textpos.Region{End: textpos.Pos{Line: fs.Src.NLines(), Char: 0}}
	kt := token.KeyToken{Token: token.Keyword, Key: "func"}
	pos, ok := ps.FindToken(kt, all)
	require.True(t, ok)
	assert.True(t, ps.MatchToken(kt, pos))
	rpos, ok := ps.FindTokenReverse(kt, all)
	require.True(t, ok)
	assert.True(t, pos.IsLess(rpos), "the first func is before the last")
	_, ok = ps.FindToken(token.KeyToken{Token: token.Keyword, Key: "fallthrough"}, all)
	assert.False(t, ok)

	sy, ok := ps.FindNameTopScope("Point")
	assert.True(t, ok)
	assert.Equal(t, "Point", sy.Name)
	_, ok = ps.FindNameScoped("Nope")
	assert.False(t, ok)

	ps.Error(textpos.Pos{Line: 1}, "test error", nil)
	assert.Equal(t, 1, ps.Errs.Len())
	ps.ResetNonMatches()
	ps.GotoEof()
	assert.True(t, ps.AtEofNext())
	ps.Destroy()
}

func TestRules(t *testing.T) {
	pr := goParser(t)
	root := pr.Parser
	require.NotNil(t, root)
	found := root.Find("FuncDecl")
	require.NotEmpty(t, found)
	assert.Equal(t, "FuncDecl", found[0].Name)
	assert.Empty(t, root.Find("NoSuchRule"))
	var buf bytes.Buffer
	root.WriteGrammar(&buf, 0)
	assert.Contains(t, buf.String(), "FuncDecl")
	assert.True(t, root.IsGroup())

	var ac parser.Acts
	assert.Equal(t, "", ac.String())
	act := parser.Act{Act: parser.ChangeToken, Token: token.NameFunction}
	assert.Contains(t, act.String(), "ChangeToken")
}
