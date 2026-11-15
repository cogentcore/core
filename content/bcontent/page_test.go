// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bcontent

import (
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testFS = fstest.MapFS{
	"widgets.md": {Data: []byte("+++\nCategories = [\"Widgets\"]\nTitle = \"All widgets\"\nDate = \"2026-05-01\"\n+++\n\nThe widgets.\n")},
	"button-widget.md": {Data: []byte(`+++
Categories = ["Widgets"]
+++

A button.

{id="figure_one"}
![one](one.png)

{id="table_two" title="Two"}
| a |

{id="figure_three" style="x"}
![three](three.png)

{id="nounderscore"}
{id="x"}
`)},
	"plain.md":    {Data: []byte("No front matter.\n")},
	"unclosed.md": {Data: []byte("+++\nTitle = \"x\"\n")},
	"bad-toml.md": {Data: []byte("+++\nTitle = \n+++\n")},
	"bad-date.md": {Data: []byte("+++\nDate = \"May 1\"\n+++\n")},
	"switches.md": {Data: []byte("+++\nCategories = [\"Widgets\"]\n+++\n\nSwitches.\n")},
}

// TestPage tests making pages, with their metadata from front matter, and
// reading their content.
func TestPage(t *testing.T) {
	pg, err := NewPage(testFS, "widgets.md")
	require.NoError(t, err)
	assert.Equal(t, "Widgets", pg.Name)
	assert.Equal(t, "widgets", pg.URL)
	assert.Equal(t, "All widgets", pg.Title)
	assert.Equal(t, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), pg.Date)

	bt, err := NewPage(testFS, "button-widget.md")
	require.NoError(t, err)
	assert.Equal(t, "Button widget", bt.Name)
	assert.Equal(t, "button-widget", bt.URL)
	assert.Equal(t, "Button widget", bt.Title, "the default title")
	assert.Equal(t, []string{"Widgets"}, bt.Categories)

	plain, err := NewPage(testFS, "plain.md")
	require.NoError(t, err)
	assert.Equal(t, []string{"Other"}, plain.Categories)
	b, err := plain.ReadContent(nil)
	require.NoError(t, err)
	assert.Equal(t, "No front matter.\n", string(b))

	_, err = NewPage(testFS, "bad-toml.md")
	assert.Error(t, err)
	_, err = NewPage(testFS, "bad-date.md")
	assert.Error(t, err)
	_, err = NewPage(testFS, "missing.md")
	assert.Error(t, err)

	un, err := NewPage(testFS, "unclosed.md")
	require.NoError(t, err)
	_, err = un.ReadContent(nil)
	assert.ErrorContains(t, err, "unclosed front matter")
	un.Filename = "missing.md"
	_, err = un.ReadContent(nil)
	assert.Error(t, err)

	sw, err := NewPage(testFS, "switches.md")
	require.NoError(t, err)
	byCat := map[string][]*Page{"Widgets": {pg, bt, sw}}
	b, err = pg.ReadContent(byCat)
	require.NoError(t, err)
	assert.Equal(t, "\n\nThe widgets.\n\n* [[Button widget]]\n* [[Switches]]\n", string(b), "the links of the category page")
	b, err = bt.ReadContent(byCat)
	require.NoError(t, err)
	assert.NotContains(t, string(b), "[[Switches]]", "not a category")
}

// TestSpecials tests the special elements of a page: figures and tables,
// by their ids, and their labels.
func TestSpecials(t *testing.T) {
	assert.Equal(t, "figure", SpecialName("figure_one"))
	assert.Equal(t, "", SpecialName("plain"))
	assert.Equal(t, "my-heading", SpecialToKebab("My Heading"))
	assert.Equal(t, "figure_the-drops", SpecialToKebab("figure_The Drops"))
	assert.Equal(t, "table_my-table:row/Col", SpecialToKebab("table_My Table:row/Col"))

	bt, err := NewPage(testFS, "button-widget.md")
	require.NoError(t, err)
	assert.Equal(t, "Figure 1", bt.SpecialLabel("figure_one"), "parsed when needed")
	assert.Equal(t, "Figure 2", bt.SpecialLabel("figure_three"))
	assert.Equal(t, "Table 1", bt.SpecialLabel("table_two"))
	assert.Equal(t, map[string][]string{"figure": {"figure_one", "figure_three"}, "table": {"table_two"}}, bt.Specials)
	assert.Equal(t, "", bt.SpecialLabel("figure_missing"))
	assert.Equal(t, "", bt.SpecialLabel("equation_one"))
	assert.Equal(t, "", bt.SpecialLabel("plain"))
	bt.ParseSpecials([]byte(`{id="figure_other"}`))
	assert.Len(t, bt.Specials["figure"], 2, "parsed once")

	un, err := NewPage(testFS, "unclosed.md")
	require.NoError(t, err)
	assert.Equal(t, "", un.SpecialLabel("figure_one"), "an error reading the content")
}
