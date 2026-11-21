// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlengine

import (
	"golang.org/x/net/html"

	"cogentcore.org/core/htmlcore/css"
)

// A page can be read as its composed tree (see [Context.ShadowScope]), in
// which the content of the shadow root of an element (as of the web
// components of reddit.com) is in its place, with the children of the
// element that are assigned to each slot of it in the slot. The style sheets
// of a shadow tree (style elements, and the style sheets that it adopts,
// which scripts put in style elements of it) apply only to its elements,
// with :host for its host, and ::slotted() for the children of the host in
// its slots; those of the document do not apply to the elements of shadow
// trees (which inherit the styles of the elements that they are in).

// enterScope sets the scope of the rules of the given style sheet element
// (see [Context.curScope]), and returns the function that sets it back.
func (c *Context) enterScope(n *html.Node) func() {
	prev := c.curScope
	if c.ShadowScope != nil {
		c.curScope = c.ShadowScope(n)
	}
	return func() { c.curScope = prev }
}

// inScope returns whether a rule of the given scope (nil for the document)
// and kind can select the given element.
func (c *Context) inScope(n, scope *html.Node, kind css.ScopeKind) bool {
	if kind == css.ScopeAny {
		return true
	}
	if c.ShadowScope == nil {
		return scope == nil
	}
	switch kind {
	case css.ScopeHost:
		return n == scope
	case css.ScopeSlotted:
		p := c.parentOf(n)
		return p != nil && p.Type == html.ElementNode && p.Data == "slot" && c.ShadowScope(p) == scope
	}
	return c.ShadowScope(n) == scope
}

// parentOf returns the parent of the given node in the tree that the page
// is read as (see [Context.ComposedParent]).
func (c *Context) parentOf(n *html.Node) *html.Node {
	if c.ComposedParent != nil {
		return c.ComposedParent(n)
	}
	return n.Parent
}
