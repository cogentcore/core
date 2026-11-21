// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strings"
	"testing"

	"cogentcore.org/core/htmlcore/css"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAncestorKeys tests the keys that the elements that the elements that
// a selector selects are in must have: of the compound selectors before
// combinators of descendants and children, not of siblings, nor in
// parentheses.
func TestAncestorKeys(t *testing.T) {
	keys := func(ks ...string) css.KeyMask {
		var m css.KeyMask
		for _, k := range ks {
			m = m.Or(css.KeyBit(k))
		}
		return m
	}
	assert.Equal(t, keys(".menu"), css.AncestorKeys(".menu .item"))
	assert.Equal(t, keys("div", "ul"), css.AncestorKeys("div > ul li.x"))
	assert.Equal(t, keys(".b"), css.AncestorKeys(".a + .b .c"))
	assert.Equal(t, keys(".a", "#main"), css.AncestorKeys(".a:hover #main>.b"))
	assert.Equal(t, css.KeyMask{}, css.AncestorKeys(":not(.x) .y"))
	assert.Equal(t, css.KeyMask{}, css.AncestorKeys(".a ~ .b"))
	assert.Equal(t, css.KeyMask{}, css.AncestorKeys(".item"))
	assert.Equal(t, css.KeyMask{}, css.AncestorKeys(`.a\:b .c`), "escapes")
}

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
