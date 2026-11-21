// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlengine

import (
	"golang.org/x/net/html"

	"cogentcore.org/core/htmlcore/css"
)

// The ancestor filter of selectors, as browsers have (a Bloom filter of
// the keys of the elements that an element is in): a selector whose
// compound selectors before its last one, that are of the elements that
// the element that it selects is in (as .menu in .menu .item), have keys
// (ids, classes, and names of elements) that no element that an element
// is in has, does not select it, which is found without matching it (most
// of the time of matching the elements of npr.org again, as its scripts
// changed them, was of the ancestors of each element).

// ancestorMask returns the mask of the keys of the elements that the given
// element is in, from c.ancestorMasks while it is set (for the elements
// matched at once, whose ancestors do not change meanwhile).
func (c *Context) ancestorMask(n *html.Node) css.KeyMask {
	p := n.Parent
	if p == nil || p.Type != html.ElementNode {
		return css.KeyMask{}
	}
	if c.ancestorMasks != nil {
		if m, ok := c.ancestorMasks[n]; ok {
			return m
		}
	}
	m := c.ancestorMask(p).Or(css.OwnKeys(p))
	if c.ancestorMasks != nil {
		c.ancestorMasks[n] = m
	}
	return m
}

// selMatch returns whether the given selector selects the given element,
// without matching it if it can not for its ancestors (see
// [css.AncestorKeys]).
func (c *Context) selMatch(cs *css.CompiledSelector, n *html.Node) bool {
	if cs.Anc != (css.KeyMask{}) && !c.ancestorMask(n).Has(cs.Anc) {
		return false
	}
	return cs.Sel.Match(n)
}

// keepAncestorMasks keeps the masks of the ancestors of the elements as
// they are matched (see [Context.ancestorMask]), until the returned
// function is called, for elements that do not change meanwhile.
func (c *Context) keepAncestorMasks() func() {
	if c.ancestorMasks != nil {
		return func() {}
	}
	c.ancestorMasks = map[*html.Node]css.KeyMask{}
	return func() { c.ancestorMasks = nil }
}
