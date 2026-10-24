// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"cogentcore.org/core/core"
	"cogentcore.org/core/text/csl"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// TestFigureTooltip checks that the tooltip of a figure image (which is
// also its alt text in the pre-rendered HTML) is the plain text of its
// caption, with the citation and page wikilinks replaced by their text, as
// in the caption, and without markup.
func TestFigureTooltip(t *testing.T) {
	png, err := fs.ReadFile(exampleContent, "media/fig_one.png")
	assert.NoError(t, err)
	fsys := fstest.MapFS{
		"figure.md": {Data: []byte(`{id="figure_drops"}
![The droplets of [[@^CouderFort05]], also in [[Button]] (e.g., [[@BushOza21]]), are **small** <i>and</i>   fast.](media/fig_one.png)

![An image with [[Button|a link]] and no caption.](media/fig_one.png)
`)},
		"button.md":         {Data: []byte("# Button\n\nA button.\n")},
		"media/fig_one.png": {Data: png},
	}
	b := core.NewBody()
	ct := NewContent(b).SetSource(fsys)
	ct.References = csl.NewKeyList([]csl.Item{
		{CitationKey: "CouderFort05", Type: csl.ArticleJournal, Author: []csl.Name{{Family: "Couder"}, {Family: "Fort"}}, Issued: csl.Date{DateParts: [][]any{{"2005"}}}},
		{CitationKey: "BushOza21", Type: csl.ArticleJournal, Author: []csl.Name{{Family: "Bush"}, {Family: "Oza"}}, Issued: csl.Date{DateParts: [][]any{{"2021"}}}},
	})
	// The page is rendered without a window, which would change the
	// size of the windows of the next tests.
	ct.Open("figure")
	b.UpdateTree()
	var tips []string
	ct.rightFrame.WalkDown(func(n tree.Node) bool {
		if img, ok := n.(*core.Image); ok {
			tips = append(tips, img.Tooltip)
		}
		return tree.Continue
	})
	assert.Equal(t, []string{
		"The droplets of Couder & Fort (2005), also in Button (e.g., Bush & Oza, 2021), are small and fast.",
		"An image with a link and no caption.",
	}, tips)
}
