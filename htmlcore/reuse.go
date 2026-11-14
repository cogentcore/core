// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"fmt"
	"hash/maphash"
	"reflect"
	"slices"
	"strings"

	"golang.org/x/net/html"
)

// ReuseStyles makes the context, which is to read a document that the
// given context has read (as the page that the scripts of a page change
// is read again whole when its changes can not be made otherwise), match
// the style rules of the document only against the elements whose matches
// may have changed since: those that have changed (as their attributes),
// moved, or been added, and those that their changes may change the
// matches of. The others have the declarations of the rules that matched
// them in the given context, as long as its rules are the same (as
// without a new style sheet, or another width of the viewport for its
// media queries); otherwise all of the rules are matched. Matching the
// rules is most of the time of reading a page (as about a second for each
// of the many times that npr.org was read again as it loaded).
func (c *Context) ReuseStyles(prev *Context) {
	c.reuseFrom = prev
}

// Reused returns how the declarations of the elements were found when
// the context read its page (see [Context.ReuseStyles]), for the verbose
// console: "" if they were all matched as usual.
func (c *Context) Reused() string {
	return c.reused
}

// nodeSnap is the state of an element by which style rules select it,
// apart from those of the elements that it is in and those before it (see
// [Context.ReuseStyles]): its parent, attributes and place in its parent.
type nodeSnap struct {
	parent *html.Node
	attrs  uint64

	// class, id, and style are its class, id, and style attributes, and
	// others its other attributes (as name=value lines), for the elements
	// that a change of them may select (see [Context.invalidation]).
	class, id, style, others string

	// index and typeIndex are its index in the elements of its parent,
	// and in those of the same name, as :nth-child selects it.
	index, typeIndex int32

	// last and lastType are whether it is the last element of its parent,
	// and of those of its name, as :last-child and :last-of-type select it,
	// count is the number of elements of its parent (for :nth-last-child),
	// and empty is whether it has no elements or text (for :empty).
	last, lastType, empty bool
	count                 int32
}

// snapSeed is the seed of the hashes of the attributes of elements.
var snapSeed = maphash.MakeSeed()

// attrsHash returns the hash of the attributes of the given element.
func attrsHash(n *html.Node) uint64 {
	var h maphash.Hash
	h.SetSeed(snapSeed)
	for _, a := range n.Attr {
		h.WriteString(a.Namespace)
		h.WriteByte(0)
		h.WriteString(a.Key)
		h.WriteByte(0)
		h.WriteString(a.Val)
		h.WriteByte(0)
	}
	return h.Sum64()
}

// otherAttrs returns the attributes of the given element but its class,
// id, and style, as name=value lines (with the namespaces of those that
// have one), or "" for none.
func otherAttrs(n *html.Node) string {
	var sb strings.Builder
	for _, a := range n.Attr {
		if a.Namespace == "" && (a.Key == "class" || a.Key == "id" || a.Key == "style") {
			continue
		}
		if a.Namespace != "" {
			sb.WriteString(a.Namespace)
			sb.WriteByte(':')
		}
		sb.WriteString(a.Key)
		sb.WriteByte('=')
		sb.WriteString(a.Val)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// attrNames returns the names of the attributes in the given lines of
// [otherAttrs] with their values.
func attrValues(lines string) map[string]string {
	m := map[string]string{}
	for _, l := range strings.Split(lines, "\n") {
		if k, v, ok := strings.Cut(l, "="); ok {
			m[k] = v
		}
	}
	return m
}

// snapChildren calls the given function with the state of each of the
// elements of the given parent (see [nodeSnap]).
func snapChildren(p *html.Node, f func(k *html.Node, s nodeSnap)) {
	var kids []*html.Node
	for k := p.FirstChild; k != nil; k = k.NextSibling {
		if k.Type == html.ElementNode {
			kids = append(kids, k)
		}
	}
	snaps := make([]nodeSnap, len(kids))
	var names []string // seen, for the indexes of the names
	var counts []int32
	for i, k := range kids {
		j := slices.Index(names, k.Data)
		if j < 0 {
			j = len(names)
			names = append(names, k.Data)
			counts = append(counts, 0)
		}
		empty := true
		for g := k.FirstChild; g != nil && empty; g = g.NextSibling {
			empty = g.Type != html.ElementNode && (g.Type != html.TextNode || g.Data == "")
		}
		snaps[i] = nodeSnap{parent: p, attrs: attrsHash(k), class: GetAttr(k, "class"), id: GetAttr(k, "id"), style: GetAttr(k, "style"), others: otherAttrs(k),
			index: int32(i), typeIndex: counts[j],
			last: i == len(kids)-1, empty: empty, count: int32(len(kids))}
		counts[j]++
	}
	names = names[:0]
	for i := len(kids) - 1; i >= 0; i-- {
		if !slices.Contains(names, kids[i].Data) {
			names = append(names, kids[i].Data)
			snaps[i].lastType = true
		}
	}
	for i, k := range kids {
		f(k, snaps[i])
	}
}

// snapshot records the states of the elements of the given document (see
// [nodeSnap]), with which the declarations that the context has found
// for them are found.
func (c *Context) snapshot(root *html.Node) {
	c.snaps, c.snapRoot = map[*html.Node]nodeSnap{}, root
	var walk func(p *html.Node)
	walk = func(p *html.Node) {
		snapChildren(p, func(k *html.Node, s nodeSnap) {
			c.snaps[k] = s
			walk(k)
		})
	}
	walk(root)
}

// sameRules returns whether the context has the same style rules as the
// given context, in the same order.
func (c *Context) sameRules(prev *Context) bool {
	if len(c.rules) != len(prev.rules) {
		return false
	}
	for i, r := range c.rules {
		p := prev.rules[i]
		if r.sel != p.sel || r.text != p.text || r.root != p.root || r.origin != p.origin || r.spec != p.spec ||
			r.cssBase != p.cssBase || len(r.decls) != len(p.decls) || !reflect.DeepEqual(r.container, p.container) {
			return false
		}
		for j, d := range r.decls {
			e := p.decls[j]
			if d.Property != e.Property || d.Value != e.Value || d.Important != e.Important {
				return false
			}
		}
	}
	return true
}

// placeKinds are the kinds of pseudo-classes of the places of elements
// in their parents (as :first-child), by which the changes of the places
// of elements change their matches (see [nodeSnap]).
type placeKinds struct {
	index, typeIndex, last, lastType, count, typeCount, empty bool
}

// placeClasses are the pseudo-classes of the places of elements, with
// the kinds of places that they select by.
var placeClasses = map[string]placeKinds{
	":first-child": {index: true}, ":nth-child": {index: true}, ":only-child": {index: true, last: true},
	":last-child": {last: true}, ":nth-last-child": {count: true},
	":first-of-type": {typeIndex: true}, ":nth-of-type": {typeIndex: true}, ":only-of-type": {typeIndex: true, lastType: true},
	":last-of-type": {lastType: true}, ":nth-last-of-type": {typeCount: true}, ":empty": {empty: true},
}

// or returns the kinds of both.
func (k placeKinds) or(o placeKinds) placeKinds {
	return placeKinds{k.index || o.index, k.typeIndex || o.typeIndex, k.last || o.last, k.lastType || o.lastType,
		k.count || o.count, k.typeCount || o.typeCount, k.empty || o.empty}
}

// differ returns whether the given states of an element differ in a way
// that may change its matches, for the kinds of places selected by.
func (k placeKinds) differ(o, s nodeSnap) bool {
	return k.index && o.index != s.index || k.typeIndex && o.typeIndex != s.typeIndex ||
		k.last && o.last != s.last || k.lastType && o.lastType != s.lastType ||
		k.count && o.count != s.count || k.typeCount && (o.count != s.count || o.typeIndex != s.typeIndex) ||
		k.empty && o.empty != s.empty
}

// placeUses returns the kinds of places that the style rules of the
// context select elements by, and those that they select by in compound
// selectors before their subjects (as li:first-child a, or
// p:last-of-type + div), for which the elements in and after an element
// whose place changes may have other matches.
func (c *Context) placeUses() (uses, deep placeKinds) {
	for _, r := range c.rules {
		t := r.text
		if !strings.Contains(t, ":") {
			continue
		}
		start := -1
		for pc, k := range placeClasses {
			i := strings.Index(t, pc)
			if i < 0 {
				continue
			}
			uses = uses.or(k)
			if start < 0 {
				start = subjectStart(t)
			}
			if i < start {
				deep = deep.or(k)
			}
		}
	}
	return
}

// siblingCombinators returns whether a style rule of the context may
// select elements by those before them (with + or ~), so that the changes
// of an element may change the matches of those after it, and whether
// one may select elements in those (as .a ~ .b .c).
func (c *Context) siblingCombinators() (siblings, in bool) {
	for _, r := range c.rules {
		s, i := siblingCombinator(r.text)
		siblings, in = siblings || s, in || i
		if in {
			break
		}
	}
	return
}

// siblingCombinator returns whether the given selector has a combinator
// of siblings (+ or ~, not in brackets or parentheses, as of :nth-child(2n+1)
// and [a~=b]), and whether one is followed by a combinator of descendants.
func siblingCombinator(sel string) (sibling, in bool) {
	depth := 0
	var quote byte
	for i := 0; i < len(sel); i++ {
		ch := sel[i]
		switch {
		case quote != 0:
			if ch == '\\' {
				i++
			} else if ch == quote {
				quote = 0
			}
		case ch == '"' || ch == '\'':
			quote = ch
		case ch == '(' || ch == '[':
			depth++
		case ch == ')' || ch == ']':
			depth--
		case depth > 0:
		case ch == '+' || ch == '~':
			sibling = true
		case sibling && ch == '>':
			in = true
		case sibling && ch == ' ':
			// a combinator of descendants, not the spaces around another
			prev := strings.TrimRight(sel[:i], " ")
			next := strings.TrimLeft(sel[i:], " ")
			if next != "" && prev != "" && !strings.ContainsAny(next[:1], "+~>") && !strings.ContainsAny(prev[len(prev)-1:], "+~>") {
				in = true
			}
		}
	}
	return
}

// reuseMatches finds the declarations of the elements of the given
// document from the rules that matched them in the given context, which
// has read it, matching the rules again only against the elements whose
// matches may have changed (see [Context.ReuseStyles]). It returns false,
// with no declarations found, if the rules are not those of the given
// context, or too many elements are to be matched again.
func (c *Context) reuseMatches(root *html.Node, prev *Context) bool {
	if !c.sameRules(prev) {
		c.reused = "the style rules changed"
		return false
	}
	c.snapshot(root)
	// the elements to match again: those that have changed, moved, or
	// are new, and those in them, and those after them for the
	// combinators of siblings, and those whose children have changed
	// (whole has those added with the elements in them)
	affected, whole := map[*html.Node]bool{}, map[*html.Node]bool{}
	var changed []*html.Node
	var add func(n *html.Node) bool
	add = func(n *html.Node) bool {
		if n.Type != html.ElementNode || whole[n] {
			return true
		}
		affected[n], whole[n] = true, true
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			add(k)
		}
		return true
	}
	siblings, inSiblings := c.siblingCombinators()
	uses, deep := c.placeUses()
	// later adds the elements after the given one (from it, if first), for
	// the combinators of siblings
	later := func(k *html.Node) {
		if !siblings {
			return
		}
		for ; k != nil; k = k.NextSibling {
			if inSiblings {
				add(k)
			} else if k.Type == html.ElementNode {
				affected[k] = true
			}
		}
	}
	// the parents of the elements that were removed, with the least index
	// that one had in it: those after it are at or after that index now
	// (with those inserted before them)
	removedFrom := map[*html.Node]int32{}
	removed := func(p *html.Node, index int32) {
		if i, ok := removedFrom[p]; !ok || index < i {
			removedFrom[p] = index
		}
	}
	for n, s := range c.snaps {
		o, ok := prev.snaps[n]
		switch {
		case !ok || o.parent != s.parent || o.attrs != s.attrs || deep.differ(o, s):
			// (and those of a place that selectors select those in or after
			// it by)
			changed = append(changed, n)
			add(n)
			later(n.NextSibling)
			if ok && o.parent != s.parent && o.parent != nil {
				removed(o.parent, o.index)
			}
		case uses.differ(o, s):
			// moved in its parent, or with other children, as a
			// selector of the place of elements selects them
			changed = append(changed, n)
			affected[n] = true
		}
	}
	for n, o := range prev.snaps {
		if _, ok := c.snaps[n]; !ok && o.parent != nil {
			removed(o.parent, o.index)
		}
	}
	for p, index := range removedFrom {
		if _, ok := c.snaps[p]; !ok {
			continue // (not in the document)
		}
		changed = append(changed, p)
		affected[p] = true
		i := int32(0)
		for k := p.FirstChild; k != nil; k = k.NextSibling {
			if k.Type != html.ElementNode {
				continue
			}
			if i >= index {
				later(k)
				break
			}
			i++
		}
	}
	if len(affected) > len(c.snaps)/2 {
		c.reused = fmt.Sprintf("%d of %d elements changed", len(affected), len(c.snaps))
		return false
	}
	// the declarations of the rules that matched the others
	c.styles = map[*html.Node][]*cssDecl{}
	var ids []int
	for n, ds := range prev.styles {
		if affected[n] {
			continue
		}
		if _, ok := c.snaps[n]; !ok {
			continue // (not in the document)
		}
		ids = ids[:0]
		for _, d := range ds {
			if d.rule > 0 {
				ids = append(ids, d.rule-1)
			}
		}
		slices.Sort(ids)
		ids = slices.Compact(ids)
		c.setMatched(n, c.ruleDecls(n, ids))
	}
	if c.relational {
		// those that :has() selectors select, by the changed elements
		// (only those whose matches change: the elements in them have
		// those that they have, as the matches of selectors are by the
		// elements of the document, not by those of others, and the
		// checks of the :has() selectors find all of those that change)
		one := func(n *html.Node) bool {
			if n.Type == html.ElementNode {
				affected[n] = true
			}
			return true
		}
		if !c.relationalChanges(root, changed, one) || len(affected) > len(c.snaps)/2 {
			c.styles, c.containerQueried = nil, nil
			c.reused = "too many elements that :has() selects changed"
			return false
		}
	}
	for n := range affected {
		if n.Type == html.ElementNode {
			c.setMatched(n, c.matchDecls(n))
		}
	}
	c.reused = fmt.Sprintf("%d of %d elements matched again", len(affected), len(c.snaps))
	return true
}

// setMatched sets the declarations of the rules that match the given
// element.
func (c *Context) setMatched(n *html.Node, ds []*cssDecl) {
	if hasCustomDecls(ds) || hasCustomDecls(c.styles[n]) {
		c.varMemo = nil
	}
	if len(ds) == 0 {
		delete(c.styles, n)
	} else {
		c.styles[n] = ds
	}
	for _, d := range ds {
		if d.container != nil {
			if c.containerQueried == nil {
				c.containerQueried = map[*html.Node]bool{}
			}
			c.containerQueried[n] = true
			return
		}
	}
	if c.containerQueried != nil {
		delete(c.containerQueried, n)
	}
}

// ruleDecls returns the declarations of the given rules (indexes in the
// rules of the context, in order) for the given element, which they
// select, as [Context.matchDecls] does.
func (c *Context) ruleDecls(n *html.Node, ids []int) []*cssDecl {
	var ds []*cssDecl
	base := c.cssBase
	defer func() { c.cssBase = base }()
	for _, i := range ids {
		r := c.rules[i]
		c.cssBase = r.cssBase // for its URLs
		for _, d := range r.decls {
			prop, value := strings.ToLower(strings.TrimSpace(d.Property)), strings.TrimSpace(d.Value)
			if !validDeclaration(prop, value) {
				continue
			}
			c.cssOrder++
			c.noteVars(value)
			ds = append(ds, &cssDecl{Property: prop, Value: c.absoluteURLs(value), Important: d.Important,
				origin: r.origin, specificity: r.spec, order: c.cssOrder, container: r.container, rule: i + 1})
		}
	}
	return ds
}

// applyRules adds the declarations of the style rules of the context to
// the elements that they select, as they are added when they are not
// only recorded (see [Context.addRuleList]).
func (c *Context) applyRules() {
	if c.styles == nil {
		c.styles = map[*html.Node][]*cssDecl{}
	}
	base, container := c.cssBase, c.curContainer
	defer func() { c.cssBase, c.curContainer, c.curRule = base, container, 0 }()
	for i, r := range c.rules {
		c.cssBase, c.curContainer, c.curRule = r.cssBase, r.container, i+1
		for _, n := range c.selectNodes(r.root, r.text) {
			c.addDecls(n, r.decls, r.origin, r.spec)
		}
	}
}
