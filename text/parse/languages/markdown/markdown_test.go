// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package markdown

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

@article{Jones10,
  author = {Jones, Ann},
  title = {Other},
  journal = {Journal},
  year = {2010}
}
`

// writeBib writes the test bibliography to a temporary file.
func writeBib(t *testing.T) string {
	t.Helper()
	fn := filepath.Join(t.TempDir(), "refs.bib")
	require.NoError(t, os.WriteFile(fn, []byte(testBib), 0o644))
	return fn
}

// parseMarkdown returns the file states of the given markdown source,
// parsed by the markdown language.
func parseMarkdown(t *testing.T, src string) *parse.FileStates {
	t.Helper()
	fss := parse.NewFileStates("test.md", "", fileinfo.Markdown)
	TheMarkdownLang.ParseFile(fss, []byte(src))
	return fss
}

// TestLex tests the lexing of the test file and of lines.
func TestLex(t *testing.T) {
	txt, err := os.ReadFile("testdata/markdown_test.md")
	require.NoError(t, err)
	fss := parseMarkdown(t, string(txt))
	fs := fss.Done()
	assert.False(t, fs.LexHasErrs(), fs.LexErrReport())
	ml := &TheMarkdownLang
	assert.NotNil(t, ml.Parser())
	lx := ml.HighlightLine(fss, 0, []rune("# Head"))
	require.NotEmpty(t, lx)
	assert.Equal(t, token.TextStyleHeading, lx[0].Token.Token)
	lx = ml.HighlightLine(fss, 0, []rune("some **bold** text"))
	found := false
	for _, l := range lx {
		if l.Token.Token == token.TextStyleStrong {
			found = true
		}
	}
	assert.True(t, found, "bold: %v", lx)
	assert.Nil(t, ml.ParseLine(fs, 0))
	assert.Nil(t, ml.ParseDir(fs, ".", parse.LanguageDirOptions{}))
}

// TestFindBibliography tests finding the bibliography file in the
// front matter of a file, in yaml and toml forms.
func TestFindBibliography(t *testing.T) {
	ml := &TheMarkdownLang
	find := func(src string) string {
		fs := parse.NewFileState()
		fs.SetSrc(lexer.RunesFromString(src), "test.md", "", fileinfo.Markdown)
		return ml.FindBibliography(fs)
	}
	assert.Equal(t, "refs.bib", find("---\ntitle: x\nbibfile: refs.bib\n---\ntext\n"))
	assert.Equal(t, "refs.json", find("---\nbibfile: \"refs.json\"\n---\n"))
	assert.Equal(t, "refs.bib", find("+++\nbibfile = \"refs.bib\"\n+++\n"))
	assert.Equal(t, "", find("+++\nbibfile = refs.bib x\n+++\n"))
	assert.Equal(t, "", find("---\nbibfile\n---\n"), "no value")
	assert.Equal(t, "", find("---\ntitle: x\n---\nbibfile: refs.bib\n"), "after the front matter")
	assert.Equal(t, "", find("text\nbibfile: refs.bib\nmore\n"), "no front matter")
	assert.Equal(t, "", find("---\n"), "too short")
}

// TestCitesBibTeX tests completing and looking up citations of a BibTeX
// file named in the front matter.
func TestCitesBibTeX(t *testing.T) {
	bib := writeBib(t)
	fss := parseMarkdown(t, "---\nbibfile: "+bib+"\n---\nAs shown [@Sm")
	bf, has := fss.MetaData("bibfile")
	require.True(t, has)
	assert.Equal(t, bib, bf)
	ml := &TheMarkdownLang
	line := "As shown [@Sm"
	md := ml.CompleteLine(fss, line, textpos.Pos{Line: 3, Char: len(line)})
	assert.Equal(t, "Sm", md.Seed)
	assert.ElementsMatch(t, []string{"Smith20", "Smythe99"}, completionTexts(md))
	md = ml.CompleteLine(fss, "As shown [@^Jo", textpos.Pos{})
	assert.Equal(t, []string{"Jones10"}, completionTexts(md), "narrative")
	assert.Empty(t, ml.CompleteLine(fss, "plain text", textpos.Pos{}).Matches)

	ld := ml.Lookup(fss, "see [@Jones", textpos.Pos{})
	assert.Contains(t, string(ld.Text), "Jones10")
	assert.NotContains(t, string(ld.Text), "Smith20")
	assert.Empty(t, ml.Lookup(fss, "see [@Nobody", textpos.Pos{}).Text)
	assert.Empty(t, ml.Lookup(fss, "plain", textpos.Pos{}).Text)

	// without a bibliography
	fss = parseMarkdown(t, "no front matter\n")
	_, has = fss.MetaData("bibfile")
	assert.False(t, has)
	assert.Empty(t, ml.CompleteLine(fss, "[@Sm", textpos.Pos{}).Matches)
	assert.Empty(t, ml.Lookup(fss, "[@Sm", textpos.Pos{}).Text)

	// a missing file
	fss = parseMarkdown(t, "---\nbibfile: "+filepath.Join(t.TempDir(), "missing.bib")+"\n---\n")
	_, has = fss.MetaData("bibfile")
	assert.False(t, has)
	fss = parseMarkdown(t, "---\nbibfile: "+filepath.Join(t.TempDir(), "missing.json")+"\n---\n")
	_, has = fss.MetaData("bibfile")
	assert.False(t, has)
}

// TestCitesCSL tests completing and looking up citations of a CSL json
// file.
func TestCitesCSL(t *testing.T) {
	cfile, err := filepath.Abs(filepath.Join("..", "..", "..", "csl", "testdata", "test.json"))
	require.NoError(t, err)
	fss := parseMarkdown(t, "---\nbibfile: "+cfile+"\n---\n")
	_, has := fss.MetaData("bibfile")
	require.True(t, has)
	ml := &TheMarkdownLang
	md := ml.CompleteLine(fss, "[@Aalto", textpos.Pos{})
	assert.Contains(t, completionTexts(md), "AaltoBruckLaineEtAl05")
	for _, c := range completionTexts(md) {
		assert.True(t, strings.HasPrefix(c, "Aalto"), c)
	}
	ld := ml.Lookup(fss, "[@Aalto", textpos.Pos{})
	assert.Contains(t, string(ld.Text), "AaltoBruckLaineEtAl05")
	assert.Empty(t, ml.Lookup(fss, "[@Zzz", textpos.Pos{}).Text)
}

func completionTexts(md complete.Matches) []string {
	var s []string
	for _, c := range md.Matches {
		s = append(s, c.Text)
	}
	return s
}

// TestCompleteEdit tests the edit of a completion, which replaces the
// rest of the word after the cursor.
func TestCompleteEdit(t *testing.T) {
	ml := &TheMarkdownLang
	comp := complete.Completion{Text: "Smith20"}
	ed := ml.CompleteEdit(nil, "[@Smxyz] more", 4, comp, "Sm")
	assert.Equal(t, "Smith20", ed.NewText)
	assert.Equal(t, 3, ed.ForwardDelete)
	ed = ml.CompleteEdit(nil, "[@Smab{c}", 4, comp, "Sm")
	assert.Equal(t, 2, ed.ForwardDelete, "up to a brace")
	ed = ml.CompleteEdit(nil, "[@Sm] more", 4, comp, "Sm")
	assert.Equal(t, 0, ed.ForwardDelete)
	ed = ml.CompleteEdit(nil, "[@Sm", 10, comp, "Sm")
	assert.Equal(t, 0, ed.ForwardDelete, "past the end")
}

// TestIndentAutoBracket tests the indentation of list items and the
// matching of brackets.
func TestIndentAutoBracket(t *testing.T) {
	ml := &TheMarkdownLang
	fss := parseMarkdown(t, "x\n")
	indent := func(src string, ln int) int {
		lines := lexer.RunesFromString(src)
		tags := make([]lexer.Line, len(lines))
		for i, l := range lines {
			tags[i] = ml.HighlightLine(fss, i, l)
		}
		_, del, _, _ := ml.IndentLine(fss, lines, tags, ln, 4)
		return del
	}
	assert.Equal(t, 0, indent("* a\n* b\n", 1), "the same list")
	assert.Equal(t, 1, indent("* a\n- b\n", 1), "a nested list")
	assert.Equal(t, 0, indent("1. a\n2. b\n", 1), "a numbered list")
	assert.Equal(t, 1, indent("1. a\n* b\n", 1), "a nested list in a numbered list")
	assert.Equal(t, 0, indent("text\nmore\n", 1))
	assert.Equal(t, 0, indent("* a\n", 0), "the first line")

	m, nl := ml.AutoBracket(fss, '(', textpos.Pos{Char: 3}, []rune("abc"))
	assert.True(t, m)
	assert.False(t, nl)
	m, _ = ml.AutoBracket(fss, '(', textpos.Pos{Char: 1}, []rune("abc"))
	assert.False(t, m)
	m, _ = ml.AutoBracket(fss, '(', textpos.Pos{Char: 1}, []rune("a c"))
	assert.True(t, m)
}
