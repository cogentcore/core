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
