// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/parse"
	_ "cogentcore.org/core/text/parse/languages/golang"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// goLexSrc is Go source with many kinds of tokens.
const goLexSrc = `package lex

// A comment, with "quotes".
/* A block
   comment */
var s = "a \"quoted\" string\n"
var r = '\''
var raw = ` + "`raw\nstring`" + `
var n = []float64{1, 0x1F, 0o17, 0b101, 1.5e-3, 2i, 1_000}

func f(a int) (b int) {
	if a > 0 && a != 3 {
		b = a << 2 >> 1 &^ 1
	}
	return
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

// lexGo lexes the given source with the Go lexer.
func lexGo(t *testing.T, src string) *parse.FileState {
	t.Helper()
	fs := parse.NewFileState()
	fs.Src.InitFromString(src, "lex.go", fileinfo.Go)
	goParser(t).LexAll(fs)
	return fs
}

// lineTokens returns the tokens of the given line.
func lineTokens(fs *parse.FileState, ln int) []token.Tokens {
	var toks []token.Tokens
	for _, lx := range fs.Src.Lexs[ln] {
		toks = append(toks, lx.Token.Token)
	}
	return toks
}

func TestLexGo(t *testing.T) {
	fs := lexGo(t, goLexSrc)
	src := &fs.Src
	assert.False(t, fs.LexHasErrs(), fs.LexErrReport())
	assert.Equal(t, []token.Tokens{token.Keyword, token.Name, token.EOS}, lineTokens(fs, 0))
	assert.Equal(t, token.Comment, src.Comments[2][0].Token.Token.Cat())
	assert.Equal(t, token.LitStr, lineTokens(fs, 5)[3].SubCat())
	assert.Equal(t, token.LitStrSingle, lineTokens(fs, 6)[3])
	assert.Equal(t, token.LitStrBacktick, lineTokens(fs, 7)[3])
	var nums int
	for _, tk := range lineTokens(fs, 9) {
		if tk.Cat() == token.Literal && tk.SubCat() == token.LitNum {
			nums++
		}
	}
	assert.Equal(t, 7, nums)
	assert.Contains(t, lineTokens(fs, 13), token.OpBitAndNot)

	// tag source
	assert.Contains(t, src.LexTagSrcLn(0), "package")
	assert.Contains(t, src.LexTagSrc(), "lex")
	assert.Equal(t, "package", string(src.TokenSrc(textpos.Pos{Line: 0, Char: 0})))
	assert.Equal(t, "lex", string(src.TokenSrc(textpos.Pos{Line: 0, Char: 1})))
	assert.Equal(t, "package lex", src.TokenRegSrc(textpos.Region{Start: textpos.Pos{Line: 0}, End: textpos.Pos{Line: 0, Char: 2}}))
	reg := src.TokenSrcPos(textpos.Pos{Line: 0, Char: 1})
	assert.Equal(t, textpos.Pos{Line: 0, Char: 8}, reg.Start)
	assert.Equal(t, "package lex", src.RegSrc(textpos.Region{Start: textpos.Pos{}, End: textpos.Pos{Line: 0, Char: 11}}))
	assert.Contains(t, src.RegSrc(textpos.Region{Start: textpos.Pos{Line: 0, Char: 8}, End: textpos.Pos{Line: 5, Char: 3}}), "var")
}

func TestFileTokens(t *testing.T) {
	fs := lexGo(t, "package x\n\nvar y = 1\n")
	src := &fs.Src
	assert.Equal(t, 4, src.NLines(), "with the empty line after the last newline")
	assert.Equal(t, "var y = 1", src.SrcLine(2))
	assert.Equal(t, "", src.SrcLine(10))
	assert.Equal(t, 3, src.NTokens(0), "with the end of statement")
	assert.Equal(t, 0, src.NTokens(1))
	assert.Equal(t, 0, src.NTokens(10))
	assert.True(t, src.IsLexPosValid(textpos.Pos{Line: 2, Char: 4}))
	assert.False(t, src.IsLexPosValid(textpos.Pos{Line: 2, Char: 5}))
	assert.False(t, src.IsLexPosValid(textpos.Pos{Line: 9}))
	assert.Equal(t, token.Keyword, src.Token(textpos.Pos{Line: 2}).Token)
	assert.Equal(t, "var", string(src.LexAt(textpos.Pos{Line: 2}).Src(src.Lines[2])))
	assert.Equal(t, token.Keyword, src.LexAtSafe(textpos.Pos{Line: 1}).Token.Token, "from the line before an empty one")
	assert.Equal(t, token.EOS, src.LexAtSafe(textpos.Pos{Line: 9, Char: 9}).Token.Token)

	nxt, ok := src.NextTokenPos(textpos.Pos{Line: 0, Char: 2})
	assert.True(t, ok)
	assert.Equal(t, textpos.Pos{Line: 2, Char: 0}, nxt, "skips the empty line")
	prv, ok := src.PrevTokenPos(textpos.Pos{Line: 2, Char: 0})
	assert.True(t, ok)
	assert.Equal(t, textpos.Pos{Line: 0, Char: 2}, prv)
	_, ok = src.PrevTokenPos(textpos.Pos{Line: 0, Char: 0})
	assert.False(t, ok)
	_, ok = src.NextTokenPos(textpos.Pos{Line: 2, Char: 4})
	assert.False(t, ok)
	vp, ok := src.ValidTokenPos(textpos.Pos{Line: 1, Char: 0})
	assert.True(t, ok)
	assert.Equal(t, textpos.Pos{Line: 2, Char: 0}, vp)

	tm := src.TokenMapReg(textpos.Region{Start: textpos.Pos{Line: 0}, End: textpos.Pos{Line: 2, Char: 4}})
	assert.True(t, tm.Has(token.Keyword))
	assert.True(t, tm.Has(token.OpAsgnAssign))
	assert.False(t, tm.Has(token.PunctGpLBrace))

	// end of statement handling
	src.EnsureFinalEos(2)
	assert.Equal(t, token.EOS, src.Lexs[2][len(src.Lexs[2])-1].Token.Token)
	eos, ok := src.NextEos(textpos.Pos{Line: 2}, 0)
	assert.True(t, ok)
	assert.Equal(t, 2, eos.Line)
	eos2, ok := src.NextEosAnyDepth(textpos.Pos{Line: 2})
	assert.True(t, ok)
	assert.Equal(t, eos, eos2)
	ins := src.InsertEos(textpos.Pos{Line: 2, Char: 1})
	assert.Equal(t, token.EOS, src.Token(ins).Token)
	src.ReplaceEos(textpos.Pos{Line: 2, Char: 0})
	assert.Equal(t, token.EOS, src.Token(textpos.Pos{Line: 2, Char: 0}).Token)
	assert.Equal(t, 0, src.PrevDepth(2))
	assert.Empty(t, src.PrevStack(0))
}

func TestFileInit(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "x.go")
	require.NoError(t, os.WriteFile(fn, []byte("package x\r\nvar y int\n"), 0666))
	b, err := lexer.OpenFileBytes(fn)
	require.NoError(t, err)
	assert.Equal(t, "package x\r\nvar y int\n", string(b))
	_, err = lexer.OpenFileBytes(filepath.Join(dir, "nope.go"))
	assert.Error(t, err)

	var fl lexer.File
	require.NoError(t, fl.OpenFile(fn))
	assert.Equal(t, 3, fl.NLines())
	assert.Equal(t, fileinfo.Go, fl.Known)
	assert.Error(t, fl.OpenFile(filepath.Join(dir, "nope.go")))

	assert.True(t, fl.SetLineSrc(1, []rune("var z int")))
	assert.Equal(t, "var z int", fl.SrcLine(1))
	assert.False(t, fl.SetLineSrc(5, nil))

	var one lexer.File
	assert.True(t, one.InitFromLine(&fl, 1))
	assert.Equal(t, 2, one.NLines(), "with an extra blank line")
	assert.Equal(t, "var z int", one.SrcLine(0))
	assert.False(t, one.InitFromLine(&fl, 10))
	assert.False(t, one.InitFromString("", "e.go", fileinfo.Go))
	assert.Equal(t, [][]rune{[]rune("a"), []rune("b")}, lexer.RunesFromString("a\nb"))

	fl.SetLine(0, lexer.Line{lexer.NewLex(token.KeyToken{Token: token.Keyword}, 0, 7)}, nil, lexer.Stack{"x"})
	assert.Len(t, fl.LexLine(0), 1)
	assert.Nil(t, fl.LexLine(10))
	assert.Equal(t, lexer.Stack{"x"}, fl.PrevStack(1))
}

func TestBraces(t *testing.T) {
	src := "func f() {\n\tif x {\n\t\ty()\n\t}\n}\n"
	fs := lexGo(t, src)
	lines := fs.Src.Lines
	tags := fs.Src.Lexs
	en, ok := lexer.BraceMatch(lines, tags, '{', textpos.Pos{Line: 0, Char: 9}, 100)
	assert.True(t, ok)
	assert.Equal(t, textpos.Pos{Line: 4, Char: 0}, en)
	st, ok := lexer.BraceMatch(lines, tags, '}', textpos.Pos{Line: 3, Char: 1}, 100)
	assert.True(t, ok)
	assert.Equal(t, textpos.Pos{Line: 1, Char: 6}, st)
	_, ok = lexer.BraceMatch(lines, tags, '(', textpos.Pos{Line: 0, Char: 9}, 100)
	assert.False(t, ok, "not a paren")
	m, right := lexer.BracePair('{')
	assert.Equal(t, '}', m)
	assert.False(t, right)
	m, right = lexer.BracePair(']')
	assert.Equal(t, '[', m)
	assert.True(t, right)
	m, _ = lexer.BracePair('x')
	assert.Equal(t, rune(0), m)

	// indentation
	ind, ichr := lexer.LineIndent(lines[2], 4)
	assert.Equal(t, 2, ind)
	assert.Equal(t, "Tab", ichr.String())
	ind, _ = lexer.LineIndent([]rune("        x"), 4)
	assert.Equal(t, 2, ind, "spaces")
	pind, pln, _ := lexer.PrevLineIndent(lines, tags, 2, 4)
	assert.Equal(t, 1, pind)
	assert.Equal(t, 1, pln)
	_, del, _, _ := lexer.BracketIndentLine(lines, tags, 2, 4)
	assert.Equal(t, 1, del, "after an opening brace")
	_, del, _, _ = lexer.BracketIndentLine(lines, tags, 3, 4)
	assert.Equal(t, -1, del, "a closing brace")
	stb, enb := lexer.LineStartEndBracket(lines[0], tags[0])
	assert.False(t, stb)
	assert.True(t, enb)
	lx, idx := lexer.LastLexIgnoreComment(tags[0])
	require.NotNil(t, lx)
	assert.Equal(t, len(tags[0])-1, idx)
	lx, _ = lexer.LastLexIgnoreComment(nil)
	assert.Nil(t, lx)
}

func TestPassTwo(t *testing.T) {
	fs := lexGo(t, "package x\nfunc f() {\n\tg(1, [2]\n}\n")
	pr := goParser(t)
	pr.DoPassTwo(fs)
	assert.True(t, fs.PassTwoHasErrs(), "mismatched paren")
	assert.NotEmpty(t, fs.PassTwoErrReport())
	ts := &fs.TwoState
	assert.NotEmpty(t, ts.NestStackStr()+"x")

	ok := lexGo(t, "package x\nfunc f() {\n\tg(1, [2])\n}\n")
	pr.DoPassTwo(ok)
	assert.False(t, ok.PassTwoHasErrs(), ok.PassTwoErrReport())
	assert.Greater(t, ok.Src.Lexs[2][1].Token.Depth, 0, "nested depth")
	ln := ok.Src.Lexs[2].Clone()
	pr.PassTwo.NestDepthLine(ln, 1)
	assert.Equal(t, 1, ln[0].Token.Depth)
}

func TestRules(t *testing.T) {
	pr := goParser(t)
	lx := pr.Lexer
	require.NotNil(t, lx)
	found := lx.Find("Comment")
	assert.NotEmpty(t, found)
	assert.Empty(t, lx.Find("NoSuchRule"))
	var buf bytes.Buffer
	lx.WriteGrammar(&buf, 0)
	assert.True(t, strings.Contains(buf.String(), "String"))
	ls := &lexer.State{}
	lx.CompileAll(ls)
	lx.Validate(ls) // note: reports that the Builtins group rule has an empty String
	assert.Less(t, ls.Errs.Len(), 5)
	lx.ComputeMatchLen(ls)
}

func TestErrors(t *testing.T) {
	var el lexer.ErrorList
	assert.Nil(t, el.Err())
	el.Add(textpos.Pos{Line: 2, Char: 1}, "b.go", "second", "src line", nil)
	el.Add(textpos.Pos{Line: 1, Char: 5}, "b.go", "first", "", nil)
	el.Add(textpos.Pos{Line: 1, Char: 5}, "b.go", "first again", "", nil)
	el.Add(textpos.Pos{Line: 1, Char: 0}, "a.go", "other file", "", nil)
	assert.Equal(t, 4, el.Len())
	el.Sort()
	assert.Equal(t, "a.go", el[0].Filename)
	assert.Equal(t, "first", el[1].Msg)
	assert.Contains(t, el.Error(), "(and 3 more errors)")
	assert.Error(t, el.Err())
	rep := el.Report(2, "", true, true)
	assert.Contains(t, rep, "other file")
	assert.Contains(t, rep, "more errors")
	assert.Contains(t, el[3].Report("", true, false), "rc line", "the source from the error")
	assert.Equal(t, "b.go:3:1: second", el[3].Error())
	el.RemoveMultiples()
	assert.Equal(t, 3, el.Len(), "one error per line")
	var buf bytes.Buffer
	lexer.PrintError(&buf, el)
	assert.Contains(t, buf.String(), "second")
	buf.Reset()
	lexer.PrintError(&buf, os.ErrNotExist)
	assert.Contains(t, buf.String(), "not exist")
	el.Reset()
	assert.Equal(t, "no errors", el.Error())
}

// TestErrorsReportMax tests that a report lists at most the given number
// of errors.
func TestErrorsReportMax(t *testing.T) {
	var el lexer.ErrorList
	for ln := range 4 {
		el.Add(textpos.Pos{Line: ln}, "a.go", "err", "", nil)
	}
	rep := el.Report(2, "", false, false)
	assert.Equal(t, 2, strings.Count(rep, "a.go:"))
	assert.Contains(t, rep, "and 2 more errors")
}
