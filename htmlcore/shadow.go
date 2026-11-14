// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
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

// scopeKind is the kind of the elements that a rule of a style sheet of a
// shadow tree selects (see [Context.inScope]).
type scopeKind int8

const (
	// scopeAll is the elements of its tree (or, for a rule of the
	// document, the elements of the document).
	scopeAll scopeKind = iota

	// scopeHost is the host of the shadow root (:host).
	scopeHost

	// scopeSlotted is the children of the host in the slots of the shadow
	// tree (::slotted()).
	scopeSlotted

	// scopeAny is all elements, of the document and of shadow trees, for
	// the rules of the user agent.
	scopeAny
)

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
func (c *Context) inScope(n, scope *html.Node, kind scopeKind) bool {
	if kind == scopeAny {
		return true
	}
	if c.ShadowScope == nil {
		return scope == nil
	}
	switch kind {
	case scopeHost:
		return n == scope
	case scopeSlotted:
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

// hostFunc matches :host and :host() at the start of a selector, with its
// argument.
var hostFunc = regexp.MustCompile(`^:host(?:\(([^()]*(?:\([^()]*\))?[^()]*)\))?`)

// slottedFunc matches ::slotted() in a selector, with the compound before it
// and its argument.
var slottedFunc = regexp.MustCompile(`([^\s>+~]*)::slotted\(([^()]*(?:\([^()]*\))?[^()]*)\)`)

// scopedSelector returns the given selector of a style sheet of a shadow
// tree as one of the elements of the composed tree, and the kind of the
// elements that it selects, or false if it can not be: :host (or :host(X))
// alone selects the host (as X); :host before other compounds is the host
// (as X, or any element, which the host is of the elements of its tree);
// ::slotted(X) selects the children X of the slots (as slot > X);
// :host-context() and ::part() are not supported.
func scopedSelector(sel string) (string, scopeKind, bool) {
	if strings.Contains(sel, ":host-context") || strings.Contains(sel, "::part") {
		return "", 0, false
	}
	kind := scopeAll
	if m := hostFunc.FindStringSubmatchIndex(sel); m != nil {
		// the compound of the host: the argument of :host, and what is
		// after it before a combinator (as :host(.open):hover)
		host := ""
		if m[2] >= 0 {
			host = strings.TrimSpace(sel[m[2]:m[3]])
		}
		rest := sel[m[1]:]
		end := strings.IndexAny(rest, " >+~")
		if end < 0 {
			end = len(rest)
		}
		host += rest[:end]
		if host == "" {
			host = "*"
		}
		rest = rest[end:]
		if strings.TrimSpace(rest) == "" {
			return host, scopeHost, true
		}
		sel = host + rest
	}
	if strings.Contains(sel, "::slotted(") {
		sel = slottedFunc.ReplaceAllStringFunc(sel, func(s string) string {
			m := slottedFunc.FindStringSubmatch(s)
			pre := m[1]
			if pre == "" {
				pre = "slot"
			}
			return pre + " > " + m[2]
		})
		kind = scopeSlotted
	}
	if strings.Contains(sel, ":host") || strings.Contains(sel, "::") && !strings.Contains(sel, "::before") && !strings.Contains(sel, "::after") {
		return "", 0, false
	}
	return sel, kind, true
}
