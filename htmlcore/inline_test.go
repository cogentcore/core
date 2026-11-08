// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpaceBetweenInlineBlocks tests that the white space between inline
// blocks (as the words of the heading of github.com, each in an inline
// block for animating them) is a space, as in browsers, but not the white
// space at the start and end of the line.
func TestSpaceBetweenInlineBlocks(t *testing.T) {
	b := core.NewBody()
	ctx := NewContext()
	assert.NoError(t, ReadHTMLString(ctx, b, "<h1> <span style=\"display: inline-block\">Software</span> <span style=\"display: inline-block\">still</span> </h1>"))
	var items []string
	var sp *core.WidgetBase
	b.WalkDown(func(n tree.Node) bool {
		if tx, ok := n.(*core.Text); ok {
			items = append(items, tx.Text)
		} else if strings.HasPrefix(n.AsTree().Name, "space") {
			items = append(items, "space")
			sp = core.AsWidget(n)
		}
		return tree.Continue
	})
	assert.Equal(t, []string{"Software", "space", "still"}, items)
	if sp == nil {
		return
	}
	b.RenderImage(func() {
		assert.Greater(t, sp.Geom.Size.Actual.Total.X, float32(5), "as wide as a space")
	})
}

// TestInlineBoxes tests the inline elements that are boxes of their own in
// their lines: a tag with a border that does not wrap (as the tags of
// lobste.rs) has its border, and links with horizontal padding and no text
// beside them (as the footer of lobste.rs) have their padding, while code
// with padding in a paragraph is text in it.
func TestInlineBoxes(t *testing.T) {
	src := `<p id="story">A story <a id="tag" href="#t" style="border: 1px solid #cc0; border-radius: 5px; white-space: nowrap; padding: 0 4px">rust</a> example.com</p>` +
		`<nav id="nav"><a id="about" href="/about" style="padding-left: 12px">About</a> <a href="/tags" style="padding-left: 12px">Tags</a></nav>` +
		`<p id="para">Some <code id="code" style="padding: 0 4px">code</code> in text.</p>` +
		`<div id="side" style="font-size: 16px; line-height: 20px"><a id="pad" href="/a" style="padding: 8px 2px">Archive</a></div>`
	widgetsByID(t, src, func(ws map[string]*core.WidgetBase, dp float32) {
		require.NotNil(t, ws["tag"], "the tag is a box")
		assert.InDelta(t, 1*dp, ws["tag"].Styles.Border.Width.Top.Dots, 0.5, "with its border")
		require.NotNil(t, ws["about"], "the padded link is a box")
		assert.InDelta(t, 12*dp, ws["about"].Styles.Padding.Left.Dots, 1, "with its padding")
		assert.Nil(t, ws["code"], "the code in text is text")
		require.NotNil(t, ws["pad"])
		assert.Less(t, float32(ws["side"].Geom.TotalBBox.Dy()), 26*dp, "its vertical padding does not make its line higher")
	})
}
