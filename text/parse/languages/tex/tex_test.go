// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/parse"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	parse.LanguageSupport.OpenStandard()
}

const testBib = `@article{Smith20,
  author = {Smith, John},
  title = {A Study},
  journal = {Journal},
  year = {2020}
}

@book{Smythe99,
  author = {Smythe, Jane},
  title = {A Book},
  publisher = {Press},
  year = {1999}
}
`

// parseTex returns the file states of the given source, parsed by the
// tex language.
func parseTex(t *testing.T, src string) *parse.FileStates {
	t.Helper()
	fss := parse.NewFileStates("test.tex", "", fileinfo.TeX)
	TheTexLang.ParseFile(fss, []byte(src))
	return fss
}

func completionTexts(md complete.Matches) []string {
	var s []string
	for _, c := range md.Matches {
		s = append(s, c.Text)
	}
	return s
}

// TestLex tests the lexing of the test file and of lines.
func TestLex(t *testing.T) {
	txt, err := os.ReadFile("testdata/tex_test.tex")
	require.NoError(t, err)
	fss := parseTex(t, string(txt))
	fs := fss.Done()
	assert.False(t, fs.LexHasErrs(), fs.LexErrReport())
	tl := &TheTexLang
	assert.NotNil(t, tl.Parser())
	lx := tl.HighlightLine(fss, 0, []rune(`\textbf{bold} % comment`))
	require.NotEmpty(t, lx)
	assert.True(t, lx[0].Token.Token.InCat(token.Name) || lx[0].Token.Token.InCat(token.Keyword), lx[0].Token.Token.String())
	assert.True(t, lx[len(lx)-1].Token.Token.InCat(token.Comment), lx.String())
	assert.Nil(t, tl.ParseLine(fs, 0))
	assert.Nil(t, tl.ParseDir(fs, ".", parse.LanguageDirOptions{}))
}

// TestCompleteCommands tests completing latex commands.
func TestCompleteCommands(t *testing.T) {
	tl := &TheTexLang
	fss := parseTex(t, "text\n")
	md := tl.CompleteLine(fss, `some \textb`, textpos.Pos{})
	assert.Equal(t, `\textb`, md.Seed)
	assert.Equal(t, []string{`\textbf`}, completionTexts(md))
	md = tl.CompleteLine(fss, `\begin{enum`, textpos.Pos{})
	assert.Equal(t, "enum", md.Seed)
	assert.Equal(t, []string{"enumerate"}, completionTexts(md))
	md = tl.CompleteLine(fss, `\citea`, textpos.Pos{})
	assert.Empty(t, md.Matches, "a cite without a bibliography")
	md = tl.CompleteLine(fss, `\ci`, textpos.Pos{})
	assert.Contains(t, completionTexts(md), `\citep`, "cite commands are included")
	assert.Empty(t, tl.CompleteLine(fss, `x`, textpos.Pos{}).Matches, "too short")

	assert.True(t, HasCite(`\citep{a}`))
	assert.False(t, HasCite(`\emph{a}`))

	ed := tl.CompleteEdit(fss, `\textbxx{a}`, 6, complete.Completion{Text: `\textbf`}, `\textb`)
	assert.Equal(t, `\textbf`, ed.NewText)
	assert.Equal(t, 2, ed.ForwardDelete)
	ed = tl.CompleteEdit(fss, `\textb more`, 6, complete.Completion{Text: `\textbf`}, `\textb`)
	assert.Equal(t, 0, ed.ForwardDelete)
	ed = tl.CompleteEdit(fss, `\textbx y`, 6, complete.Completion{Text: `\textbf`}, `\textb`)
	assert.Equal(t, 1, ed.ForwardDelete)
}

// TestCites tests finding the bibliography of a file, and completing and
// looking up its citations.
func TestCites(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "refs")
	require.NoError(t, os.WriteFile(base+".bib", []byte(testBib), 0o644))
	src := "\\documentclass{article}\n\\begin{document}\ntext \\cite{Sm}\n\\bibliography{" + base + "}\n\\end{document}\n"
	fss := parseTex(t, src)
	bf, has := fss.MetaData("bibfile")
	require.True(t, has)
	assert.Equal(t, base+".bib", bf)

	tl := &TheTexLang
	md := tl.CompleteLine(fss, `text \cite{Sm`, textpos.Pos{})
	assert.Equal(t, "Sm", md.Seed)
	assert.ElementsMatch(t, []string{"Smith20", "Smythe99"}, completionTexts(md))
	ld := tl.Lookup(fss, `\citep{Smy`, textpos.Pos{})
	assert.Contains(t, string(ld.Text), "Smythe99")
	assert.NotContains(t, string(ld.Text), "Smith20")
	assert.Empty(t, tl.Lookup(fss, `\citep{Zz`, textpos.Pos{}).Text)
	assert.Empty(t, tl.Lookup(fss, `\emph{Sm`, textpos.Pos{}).Text)

	fss = parseTex(t, "\\bibliography{"+filepath.Join(dir, "missing")+"}\n")
	_, has = fss.MetaData("bibfile")
	assert.False(t, has)
	assert.Empty(t, tl.CompleteLine(fss, `\cite{Sm`, textpos.Pos{}).Matches)
	assert.Empty(t, tl.Lookup(fss, `\cite{Sm`, textpos.Pos{}).Text)

	fs := parse.NewFileState()
	fs.SetSrc(lexer.RunesFromString("text\n\n  \\bibliography{no}\n\\other\n\\"+strings.Repeat("x", 120)+"\n"), "a.tex", "", fileinfo.TeX)
	assert.Equal(t, "", tl.FindBibliography(fs), "only at the start of a line")
}

// TestIndentAutoBracket tests the indentation of environments and
// brackets, and the matching of brackets.
func TestIndentAutoBracket(t *testing.T) {
	tl := &TheTexLang
	fss := parseTex(t, "x\n")
	indent := func(src string, ln int) int {
		lines := lexer.RunesFromString(src)
		tags := make([]lexer.Line, len(lines))
		for i, l := range lines {
			tags[i] = tl.HighlightLine(fss, i, l)
		}
		_, del, _, _ := tl.IndentLine(fss, lines, tags, ln, 4)
		return del
	}
	assert.Equal(t, 1, indent("\\begin{itemize}\n\\item a\n", 1))
	assert.Equal(t, 0, indent("\\begin{itemize}\n\\end{itemize}\n", 1))
	assert.Equal(t, -1, indent("\\begin{itemize}\n    \\item a\n    \\end{itemize}\n", 2))
	assert.Equal(t, 0, indent("\\item a\n\\end{itemize}\n", 1), "no undent below zero")
	assert.Equal(t, 0, indent("text\nmore\n", 1))
	assert.Equal(t, 0, indent("text\n", 0), "the first line")

	m, nl := tl.AutoBracket(fss, '{', textpos.Pos{Char: 3}, []rune("abc"))
	assert.True(t, m)
	assert.False(t, nl)
	m, _ = tl.AutoBracket(fss, '{', textpos.Pos{Char: 1}, []rune("abc"))
	assert.False(t, m)
}
