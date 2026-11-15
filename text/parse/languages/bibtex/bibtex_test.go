// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bibtex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testBib is a small bibliography with string variables, a preamble,
// a comment, and entries with braces, quotes, numbers, and concatenation.
const testBib = `% a comment
@string{jn = "Journal of Tests"}
@string(pub = {Test Press})
@preamble{"\newcommand{\noop}[1]{}"}
@comment{ignored}

@Article{Smith20,
  author = {Smith, John and Doe, Jane},
  title = "A {Study} of Tests",
  journal = jn,
  year = 2020,
  note = jn # " and " # {more},
}

@book(Doe19,
  author = {Doe, Jane},
  title = {Book},
  publisher = pub,
  year = {2019}
)
`

// TestParse tests parsing entries, fields, string variables, preambles,
// and concatenation.
func TestParse(t *testing.T) {
	bib, err := Parse(strings.NewReader(testBib))
	require.NoError(t, err)
	require.Len(t, bib.Entries, 2)
	sm := bib.Entries[0]
	assert.Equal(t, "article", sm.Type, "lower case")
	assert.Equal(t, "Smith20", sm.CiteName)
	assert.Equal(t, "Smith, John and Doe, Jane", sm.Fields["author"].String())
	assert.Equal(t, "A Study of Tests", sm.Fields["title"].String(), "the braces in quotes are removed")
	assert.Equal(t, "Journal of Tests", sm.Fields["journal"].String(), "a string variable")
	assert.Equal(t, "2020", sm.Fields["year"].String())
	assert.Equal(t, "Journal of Tests and more", sm.Fields["note"].String(), "concatenated")
	doe := bib.Entries[1]
	assert.Equal(t, "book", doe.Type)
	assert.Equal(t, "Test Press", doe.Fields["publisher"].String())
	require.Len(t, bib.Preambles, 1)
	assert.Contains(t, bib.Preambles[0].String(), `\newcommand`)
	assert.Len(t, bib.StringVar, 2)

	be, ok := bib.Lookup("Doe19")
	assert.True(t, ok)
	assert.Same(t, doe, be)
	_, ok = bib.Lookup("None")
	assert.False(t, ok)
	bib.SortEntries()
	assert.Equal(t, "Doe19", bib.Entries[0].CiteName)
}

// TestStrings tests the plain, raw, and pretty strings of a bibliography,
// and that the raw string parses to the same entries.
func TestStrings(t *testing.T) {
	bib, err := Parse(strings.NewReader(testBib))
	require.NoError(t, err)
	s := bib.String()
	assert.Contains(t, s, "@article{Smith20,")
	// (the fields are in map order, so the last one has no comma)
	assert.Regexp(t, `\n  year = 2020,?\n`, s)
	assert.Regexp(t, `\n  note = \{Journal of Tests and more\},?\n`, s)

	raw := bib.RawString()
	assert.Contains(t, raw, "@string{jn = {Journal of Tests}}")
	assert.Contains(t, raw, "@preamble{")
	assert.Contains(t, raw, `note = jn # { and } # {more}`, "the concatenation of a variable and constants")
	again, err := Parse(strings.NewReader(raw))
	require.NoError(t, err)
	require.Len(t, again.Entries, 2)
	for i, e := range bib.Entries {
		for k, v := range e.Fields {
			assert.Equal(t, v.String(), again.Entries[i].Fields[k].String(), "%s %s", e.CiteName, k)
		}
	}

	pretty := bib.PrettyString()
	ti := strings.Index(pretty, "title")
	ai := strings.Index(pretty, "author")
	assert.Less(t, ti, ai, "the title first")
	assert.Contains(t, pretty, `"A Study of Tests"`)
	assert.Contains(t, pretty, `"Book"`)
	assert.Contains(t, pretty, "2020,")
}

// TestBuild tests building a bibliography in code.
func TestBuild(t *testing.T) {
	bib := NewBibTex()
	e := NewBibEntry(" Misc ", "my key")
	assert.Equal(t, "misc", e.Type)
	assert.Equal(t, "mykey", e.CiteName, "spaces removed")
	e.AddField(" title ", NewBibConst("T"))
	bib.AddEntry(e)
	v := bib.GetStringVar("undefined")
	assert.Equal(t, "undefined", v.String(), "an undefined variable is its name")
	assert.Equal(t, "undefined", v.RawString())
	var nv *BibVar
	assert.Equal(t, "", nv.String())
	c := NewBibComposite(NewBibConst("a"))
	c = c.Append(v)
	assert.Equal(t, "aundefined", c.String())
	assert.Equal(t, "{a} # undefined", c.RawString())
	assert.Equal(t, "{T}", NewBibConst("T").RawString())
	assert.Equal(t, "T", e.Fields["title"].String())
}

// TestParseErrors tests the errors of malformed bibliographies.
func TestParseErrors(t *testing.T) {
	for _, src := range []string{
		`@article{a, title = {x}`,
		`@article{a title = {x}}`,
		`@article{a, title = {a {b}}`,
	} {
		_, err := Parse(strings.NewReader(src))
		assert.Error(t, err, src)
	}
	pe := &ErrParse{Pos: TokenPos{Lines: []int{10}, Char: 3}, Err: "bad"}
	assert.Contains(t, pe.Error(), "bad")
	assert.Contains(t, pe.Error(), "2:3")
}

// TestFiles tests opening bibliography files, which are read again only
// when they change, and finding them on BIBINPUTS.
func TestFiles(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "refs.bib")
	require.NoError(t, os.WriteFile(fn, []byte(testBib), 0o644))

	fls := Files{}
	fl, err := fls.Open(fn)
	require.NoError(t, err)
	assert.Len(t, fl.BibTex.Entries, 2)
	b := fl.BibTex
	fl2, err := fls.Open(fn)
	require.NoError(t, err)
	assert.Same(t, fl, fl2)
	assert.Same(t, b, fl2.BibTex, "not read again")

	later := time.Now().Add(time.Hour)
	require.NoError(t, os.WriteFile(fn, []byte(testBib+"\n@misc{New, title = {N}}\n"), 0o644))
	require.NoError(t, os.Chtimes(fn, later, later))
	fl3, err := fls.Open(fn)
	require.NoError(t, err)
	assert.Len(t, fl3.BibTex.Entries, 3, "read again after a change")

	t.Setenv("BIBINPUTS", dir)
	t.Setenv("TEXINPUTS", "")
	p, err := FullPath("refs.bib")
	require.NoError(t, err)
	assert.Equal(t, fn, p)
	_, err = FullPath("missing.bib")
	assert.Error(t, err)
	t.Setenv("BIBINPUTS", "")
	_, err = FullPath("missing.bib")
	assert.ErrorContains(t, err, "empty")

	bad := filepath.Join(dir, "bad.bib")
	require.NoError(t, os.WriteFile(bad, []byte(`@article{a, title = {x}`), 0o644))
	_, err = fls.Open(bad)
	assert.Error(t, err)
}

// TestParseForms tests text outside of entries, which is ignored (as
// comments), comments with any text, the parenthesized forms of entries,
// and that a parse after an error starts anew.
func TestParseForms(t *testing.T) {
	src := `This text and % comments are ignored, as in BibTeX.
@comment{any text: with, spaces = and {braces}}
@comment(more text)
@string(pre = "Pre")
@preamble("p")
@misc(Key1, title = pre # {fix})
@misc{Key2, title = {T}}
trailing text`
	bib, err := Parse(strings.NewReader(src))
	require.NoError(t, err)
	require.Len(t, bib.Entries, 2)
	assert.Equal(t, "Prefix", bib.Entries[0].Fields["title"].String())
	assert.Equal(t, "T", bib.Entries[1].Fields["title"].String())
	assert.Len(t, bib.Preambles, 1)
	_, has := bib.StringVar["any"]
	assert.False(t, has, "the text of a comment is not read as variables")

	_, err = Parse(strings.NewReader(`@misc{a, title = {x`))
	assert.Error(t, err)
	bib, err = Parse(strings.NewReader(`@misc{b, year = 2001}`))
	require.NoError(t, err, "a parse after an error")
	assert.Equal(t, "2001", bib.Entries[0].Fields["year"].String())
}
