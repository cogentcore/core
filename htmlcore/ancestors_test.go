// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cogentcore.org/core/htmlcore/css"
)

// TestAncestorFilter tests that the selectors that the ancestor filter
// skips are those that do not select an element, as matched.
func TestAncestorFilter(t *testing.T) {
	ctx := NewContext()
	doc, err := ParseHTML(ctx, strings.NewReader(`<html><body><div class="menu open" id="m"><ul><li class="item" id="i">x</li></ul></div><p id="p">y</p></body></html>`))
	require.NoError(t, err)
	defer ctx.keepAncestorMasks()()
	for _, n := range []string{"i", "p"} {
		el := elementByID(doc, n)
		for _, sel := range []string{".menu .item", ".open li", "#m > ul > li", "body p", ".other .item", "ul p", "div.menu li#i", "section li"} {
			cs := css.CompileSelector(sel)
			assert.Equal(t, cs.Sel.Match(el), ctx.selMatch(cs, el), "%s for #%s", sel, n)
		}
	}
}
