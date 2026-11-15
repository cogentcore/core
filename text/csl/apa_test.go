// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csl

//core:js-test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testKeyList returns the key list of the items of the test file.
func testKeyList(t *testing.T) *KeyList {
	items, err := Open("testdata/test.json")
	require.NoError(t, err)
	return NewKeyList(items)
}

// TestCiteRefAPA tests the APA citations and references of all of the
// item types of the test file.
func TestCiteRefAPA(t *testing.T) {
	kl := testKeyList(t)
	for _, key := range kl.AlphaKeys() {
		it := kl.At(key)
		cite := Cite(APA, Parenthetical, it)
		assert.NotEmpty(t, cite, key)
		assert.Equal(t, cite, CiteDefault(Parenthetical, it))
		ref := Ref(APA, it)
		txt := string(ref.Join())
		assert.NotEmpty(t, txt, key)
		if it.Title != "" {
			assert.Contains(t, txt, strings.TrimSuffix(it.Title, "."), key)
		}
	}

	it := kl.At("AaltoBruckLaineEtAl05")
	assert.Equal(t, "Aalto et al., 2005", CiteAPA(Parenthetical, it))
	assert.Equal(t, "Aalto et al. (2005)", CiteAPA(Narrative, it))
	ref := string(RefAPA(it).Join())
	assert.True(t, strings.HasPrefix(ref, "Aalto, S., Brück, A., Laine, M., Någren, K., & Rinne, J.O. (2005)."), ref)

	refs, items := RefsDefault(kl)
	assert.Len(t, refs, len(kl.Keys))
	assert.Len(t, items, len(kl.Keys))
	for i, key := range kl.AlphaKeys() {
		assert.Equal(t, key, items[i].CitationKey, "sorted by key")
	}
}

// TestNames tests the formatting of names.
func TestNames(t *testing.T) {
	nms := []Name{{Family: "Smith", Given: "John Paul"}, {Literal: "Jane Doe"}, {Given: "Prince"}}
	fam, giv := NameFamilyGiven(&nms[1])
	assert.Equal(t, "Doe", fam)
	assert.Equal(t, "Jane", giv)
	fam, giv = NameFamilyGiven(&Name{Literal: "Doe, Jane"})
	assert.Equal(t, "Doe", fam)
	assert.Equal(t, "Jane", giv)
	fam, giv = NameFamilyGiven(&nms[2])
	assert.Equal(t, "Prince", fam)
	assert.Equal(t, "", giv)
	fam, _ = NameFamilyGiven(&Name{})
	assert.Equal(t, "", fam)

	assert.Equal(t, "Smith, J.P., Doe, J., & Prince", NamesLastFirstInitialCommaAmpersand(nms))
	assert.Equal(t, "J.P. Smith, J. Doe, & Prince", NamesFirstInitialLastCommaAmpersand(nms))
	assert.Equal(t, "Smith et al.", NamesCiteEtAl(nms))
	assert.Equal(t, "Smith & Doe", NamesCiteEtAl(nms[:2]))
	assert.Equal(t, "Smith", NamesCiteEtAl(nms[:1]))
	assert.Equal(t, "end.", EnsurePeriod("end"))
	assert.Equal(t, "end.", EnsurePeriod("end."))
}

// TestMarkdown tests extracting citations from markdown files, and
// writing their references.
func TestMarkdown(t *testing.T) {
	kl := testKeyList(t)
	fsys := fstest.MapFS{
		"a.md": {Data: []byte("As shown [@Anderson86; @Abbott92], and [@^AaltoBruckLaineEtAl05].\n")},
		"b.md": {Data: []byte("Unknown [@Nobody99].\n")},
	}
	var buf bytes.Buffer
	cited, err := GenerateMarkdown(&buf, fsys, "## References", kl, APA, "a.md", "b.md", "missing.md")
	assert.Error(t, err, "a citation not found and a missing file")
	assert.Equal(t, []string{"AaltoBruckLaineEtAl05", "Abbott92", "Anderson86"}, cited.AlphaKeys())
	out := buf.String()
	assert.True(t, strings.HasPrefix(out, "## References\n\n"))
	assert.Contains(t, out, `<p id="Anderson86">`)
	assert.Equal(t, 3, strings.Count(out, "<p id="))
	assert.Contains(t, cited.PrettyString(), "Anderson86")
}

// TestFiles tests opening csl files, which are read again only when they
// change, and saving key lists.
func TestFiles(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "refs.json")
	kl := testKeyList(t)
	require.NoError(t, SaveKeyList(kl, fn))

	fls := Files{}
	fl, err := fls.Open(fn)
	require.NoError(t, err)
	assert.Equal(t, len(kl.Keys), len(fl.Items.Keys))
	fl2, err := fls.Open(fn)
	require.NoError(t, err)
	assert.Same(t, fl, fl2, "the same file")

	_, err = fls.Open(filepath.Join(dir, "missing.json"))
	assert.Error(t, err)

	items, err := OpenFS(os.DirFS(dir), "refs.json")
	require.NoError(t, err)
	assert.Len(t, items, len(kl.Keys))
	var more KeyList
	more.AddItems(items[:2])
	assert.Len(t, more.Keys, 2)
}
