// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"fmt"
	"slices"
	"strings"

	"cogentcore.org/core/core"
	"github.com/aymerick/douceur/css"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// styleRule is a style rule that has been added to a context, with one
// selector, by which the declarations of elements are found again when
// they change (see [Context.Restyle]).
type styleRule struct {
	root   *html.Node
	sel    *compiledSelector
	text   string // of its selector
	decls  []*css.Declaration
	origin cssOrigin
	spec   int

	// container is the condition of the @container rule that it is in.
	container *containerCond

	// cssBase is the URL that the URLs of the rule are relative to.
	cssBase string

	// relational is whether its selector has :has().
	relational bool

	// subjectHas is whether its :has() selectors are all in the compound
	// of the elements that it selects (as .a:has(.b), not .a:has(.b) .c),
	// so that it only selects other elements when the descendants or later
	// siblings of an element change (see [Context.relationalChanges]).
	subjectHas bool

	// hasCompound is, for a rule whose :has() selectors are all in one
	// compound that is not its subject (as .a:has(.b) .c), that compound
	// without them (.a), and hasSiblings whether a sibling combinator
	// follows it: the rule only selects other elements in the elements
	// that match it whose descendants (or later siblings) change, and their
	// later siblings if hasSiblings (see [Context.relationalChanges]).
	hasCompound *compiledSelector
	hasSiblings bool
}

// pseudoRule is a style rule of a ::before or ::after pseudo-element that
// has been added to a context (see [styleRule]).
type pseudoRule struct {
	root   *html.Node
	sel    *compiledSelector
	text   string // of its selector
	kind   string
	decls  []*css.Declaration
	origin cssOrigin
	spec   int
}

// ruleIndex has the indexes of the style rules of a context by the key of
// their selectors (see [selectorKey]), so that only the rules that may
// select an element are matched against it, and those without a key.
type ruleIndex struct {
	byID, byClass, byTag map[string][]int
	other                []int

	// relational has the rules with :has() selectors, except those whose
	// :has() selectors are in their subject compound, in subjectHas, and
	// those whose :has() selectors are in one other compound, in hasAbove.
	relational, subjectHas, hasAbove []int

	// n is the number of rules indexed.
	n int
}

// recordRule records the given style rule (of one selector), which is
// being added, as the rule of the declarations that it adds (see
// [Context.Restyle]).
func (c *Context) recordRule(root *html.Node, sel string, cs *compiledSelector, decls []*css.Declaration, origin cssOrigin, spec int) {
	has := strings.Contains(sel, ":has(")
	subject := has && strings.Index(sel, ":has(") >= subjectStart(sel)
	r := &styleRule{root: root, sel: cs, text: sel, decls: decls, origin: origin, spec: spec, container: c.curContainer, cssBase: c.cssBase, relational: has, subjectHas: subject}
	if has && !subject {
		r.hasCompound, r.hasSiblings = hasCompound(sel)
	}
	c.rules = append(c.rules, r)
	c.curRule = len(c.rules)
	c.relational = c.relational || has
}

// index returns the index of the rules of the context, updated for the
// rules added since it was made.
func (c *Context) index() *ruleIndex {
	ix := c.ruleIndex
	if ix == nil {
		ix = &ruleIndex{byID: map[string][]int{}, byClass: map[string][]int{}, byTag: map[string][]int{}}
		c.ruleIndex = ix
	}
	for ; ix.n < len(c.rules); ix.n++ {
		i, r := ix.n, c.rules[ix.n]
		switch r.sel.keyKind {
		case '#':
			ix.byID[r.sel.key] = append(ix.byID[r.sel.key], i)
		case '.':
			ix.byClass[r.sel.key] = append(ix.byClass[r.sel.key], i)
		case 't':
			ix.byTag[r.sel.key] = append(ix.byTag[r.sel.key], i)
		default:
			ix.other = append(ix.other, i)
		}
		switch {
		case r.subjectHas:
			ix.subjectHas = append(ix.subjectHas, i)
		case r.hasCompound != nil:
			ix.hasAbove = append(ix.hasAbove, i)
		case r.relational:
			ix.relational = append(ix.relational, i)
		}
	}
	return ix
}

// candidates returns the indexes of the rules that may select the given
// element, in order.
func (ix *ruleIndex) candidates(n *html.Node) []int {
	cands := slices.Clone(ix.other)
	cands = append(cands, ix.byTag[strings.ToLower(n.Data)]...)
	for _, a := range n.Attr {
		if a.Namespace != "" {
			continue
		}
		switch a.Key {
		case "id":
			cands = append(cands, ix.byID[a.Val]...)
		case "class":
			for _, cl := range strings.Fields(a.Val) {
				cands = append(cands, ix.byClass[cl]...)
			}
		}
	}
	slices.Sort(cands)
	return slices.Compact(cands)
}

// matchDecls returns the declarations of the style rules that select the
// given element (without those of its attributes, which
// [Context.declarations] adds), as they are added when a page is read.
func (c *Context) matchDecls(n *html.Node) []*cssDecl {
	var ds []*cssDecl
	base, container := c.cssBase, c.curContainer
	defer func() { c.cssBase, c.curContainer = base, container }()
	for _, i := range c.index().candidates(n) {
		r := c.rules[i]
		if !contains(r.root, n) || !c.selMatch(r.sel, n) {
			continue
		}
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

// matchPseudo returns the declarations of the given pseudo-element of the
// given element, as [Context.addPseudoRule] adds them.
func (c *Context) matchPseudo(n *html.Node, kind string) []*cssDecl {
	var ds []*cssDecl
	for _, r := range c.pseudoRules {
		if r.kind != kind || r.sel == nil || !contains(r.root, n) || !c.selMatch(r.sel, n) {
			continue
		}
		for _, d := range r.decls {
			c.cssOrder++
			c.noteVars(d.Value)
			ds = append(ds, &cssDecl{Property: strings.ToLower(strings.TrimSpace(d.Property)), Value: c.absoluteURLs(strings.TrimSpace(d.Value)),
				Important: d.Important, origin: r.origin, specificity: r.spec, order: c.cssOrder})
		}
	}
	return ds
}

// contains returns whether the given node is a or is in it.
func contains(a, n *html.Node) bool {
	for ; n != nil; n = n.Parent {
		if n == a {
			return true
		}
	}
	return false
}

// maxRestyled is the number of elements that [Context.Restyle] matches the
// style rules against, beyond which the page is read again.
const maxRestyled = 20000

// nodeStyles is the state of the declarations of an element in a context,
// which [Context.Restyle] restores if it can not restyle the page.
type nodeStyles struct {
	styles                  []*cssDecl
	sorted, hinted, queried bool
	before, after           []*cssDecl
	hasBefore, hasAfter     bool
	snap                    nodeSnap
	hasSnap                 bool
}

// Restyle updates the styles of the widgets of the page that the context
// has read for the given elements, whose attributes that select them in
// style rules (as their class and style) or states (as their :hover) have
// changed, as for the changes that scripts make to the styles of a page,
// without reading it again (which makes new widgets for all of it). The
// declarations of the elements, those in them and those after them are
// found again from the style rules of the page, and the widgets of those
// whose declarations change are styled again, laid out again if they may
// have another size. It returns false, leaving the styles as they were, if
// a change can not be made so, for the page to be read again: one of a
// property that changes the widgets that elements have (as display, or
// the content of a pseudo-element) or how they are arranged (as in a grid
// or a table), or one of an element that has no widget of its own (as an
// inline element in a run of text).
func (c *Context) Restyle(changed []*html.Node) bool {
	c.containersChanged() // (declarations may change)
	defer c.containersChanged()
	c.fallback = ""
	if len(changed) == 0 {
		return true
	}
	u, ok := c.updateDecls(changed, nil, nil)
	if !ok {
		return false
	}
	u.apply(c)
	return true
}

// declUpdate is an update of the declarations of the elements of a page
// (see [Context.updateDecls]): the widgets to style again, whether they
// are to be laid out again, and the function that undoes it.
type declUpdate struct {
	widgets []core.Widget
	layout  bool
	undo    func()

	// freeChanged has the elements read again whose declarations (or
	// those of their pseudo-elements) have changed.
	freeChanged map[*html.Node]bool

	// transitions are the transitions of the properties that changed.
	transitions []Transition
}

// apply styles the widgets of the update again, and lays them out again
// if needed.
func (u *declUpdate) apply(c *Context) {
	if c.OnTransition != nil && len(u.transitions) > 0 {
		defer c.OnTransition(u.transitions)
	}
	for _, w := range u.widgets {
		if bs := c.blocks[w]; bs != nil {
			wb := w.AsWidget()
			n := NodeOf(w)
			bs.ops = c.compileDecls(c.declarations(n))
			bs.inherit = c.inlineInheritedOps(n, wb)
		}
	}
	for _, w := range u.widgets {
		wb := w.AsWidget()
		wb.StyleTree()
		if u.layout {
			wb.NeedsLayout()
		}
		wb.NeedsRender()
	}
}

// updateDecls finds the declarations of the elements whose declarations
// may change again (see [Context.Restyle]): the given changed elements,
// those in them, and those after them (and those in them), and the given
// whole elements and those in them. Those for which free returns true
// (if not nil) are read again (see [Context.Reread]), so that their
// declarations may change in any way; for the others, it returns false,
// leaving the declarations as they were, if they change in a way that
// their widgets can not be styled again for, and otherwise the widgets to
// style again.
func (c *Context) updateDecls(changed, whole []*html.Node, free func(n *html.Node) bool) (*declUpdate, bool) {
	isFree := func(n *html.Node) bool { return free != nil && free(n) }
	var root *html.Node
	if len(changed) > 0 {
		root = rootNode(changed[0])
	} else if len(whole) > 0 {
		root = rootNode(whole[0])
	} else if len(c.addedShallow) > 0 {
		root = rootNode(c.addedShallow[0])
	} else if len(c.addedDeep) > 0 {
		root = rootNode(c.addedDeep[0])
	} else {
		return &declUpdate{undo: func() {}}, true
	}
	// the elements whose declarations may change: those changed, those
	// in them, and those after them (for the combinators of siblings),
	// and those in them
	affected := map[*html.Node]bool{}
	var order []*html.Node
	var add func(n *html.Node) bool
	add = func(n *html.Node) bool {
		if n.Type != html.ElementNode || affected[n] {
			return true
		}
		if len(order) >= maxRestyled {
			return false
		}
		affected[n] = true
		order = append(order, n)
		if c.matchKept[n] {
			// kept with its widgets (see Context.keptElements): the
			// elements in it match as they did (see Context.shallowKept)
			return true
		}
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			if !add(k) {
				return false
			}
		}
		return true
	}
	tooMany := fmt.Sprintf("more than %d elements to style again", maxRestyled)
	for _, n := range c.addedShallow {
		if !affected[n] {
			affected[n] = true
			order = append(order, n)
		}
	}
	for _, n := range c.addedDeep {
		if !add(n) {
			return nil, c.noChange(tooMany)
		}
	}
	for _, n := range whole {
		if !add(n) {
			return nil, c.noChange(tooMany)
		}
	}
	for _, n := range changed {
		if n == nil || n.Type != html.ElementNode || rootNode(n) != root {
			return nil, c.noChange("a change that is not of an element of the page")
		}
		if (n.Namespace != "" || inForeign(n)) && !isFree(n) {
			// svg and math, which are read differently
			return nil, c.noChange("a change in svg or math: " + describeNode(n))
		}
		selects, siblings, ok := c.invalidation(n)
		if ok {
			// only the elements that the rules of its changed classes
			// and id may select, as the invalidation sets of browsers (a
			// class of the html or body element of npr.org matched all of
			// its 8000 elements again)
			var walk func(k *html.Node) bool
			walk = func(k *html.Node) bool {
				for ; k != nil; k = k.NextSibling {
					if k.Type != html.ElementNode {
						continue
					}
					if selects(k) && !affected[k] {
						if len(order) >= maxRestyled {
							return false
						}
						affected[k] = true
						order = append(order, k)
					}
					if !walk(k.FirstChild) {
						return false
					}
				}
				return true
			}
			if !affected[n] {
				affected[n] = true
				order = append(order, n)
			}
			if !walk(n.FirstChild) {
				return nil, c.noChange(tooMany)
			}
			if siblings {
				for s := n.NextSibling; s != nil; s = s.NextSibling {
					if !add(s) {
						return nil, c.noChange(tooMany)
					}
				}
			}
			continue
		}
		for s := n; s != nil; s = s.NextSibling {
			if !add(s) {
				return nil, c.noChange(tooMany)
			}
		}
	}
	restore := c.setCheckedAttrs(root) // for :checked and :hover
	defer c.keepAncestorMasks()()
	defer restore()
	if c.relational {
		// the elements that :has() selectors select, which may be any,
		// whose matches have changed
		if !c.relationalChanges(root, append(append([]*html.Node{}, changed...), whole...), add) {
			return nil, c.noChange("the elements that :has() selects")
		}
	}
	// the declarations as they were (all of them first, as those of an
	// element depend on those of the elements that it is in, for var())
	saved := map[*html.Node]*nodeStyles{}
	old := map[*html.Node]map[string]string{}
	oldDisplay := map[*html.Node]string{}
	for _, n := range order {
		// (an element that was not read, as one in an element that is
		// not displayed, has the declarations that it has now)
		old[n] = effectiveDecls(c.declarations(n))
		oldDisplay[n] = c.displayOf(n)
		ns := &nodeStyles{styles: c.styles[n], sorted: c.sorted[n], hinted: c.hinted[n], queried: c.containerQueried[n]}
		ns.snap, ns.hasSnap = c.snaps[n]
		ns.before, ns.hasBefore = c.pseudoStyles[pseudoKey{n, "before"}]
		ns.after, ns.hasAfter = c.pseudoStyles[pseudoKey{n, "after"}]
		saved[n] = ns
	}
	undo := func() {
		c.varMemo = nil
		for n, ns := range saved {
			c.styles[n], c.sorted[n], c.hinted[n] = ns.styles, ns.sorted, ns.hinted
			if c.containerQueried != nil {
				c.containerQueried[n] = ns.queried
			}
			c.setPseudo(n, "before", ns.before, ns.hasBefore)
			c.setPseudo(n, "after", ns.after, ns.hasAfter)
			if c.snaps != nil {
				if ns.hasSnap {
					c.snaps[n] = ns.snap
				} else {
					delete(c.snaps, n)
				}
			}
		}
	}
	// the states of the elements, with which their declarations are found
	// (see Context.ReuseStyles)
	if c.snaps != nil {
		parents := map[*html.Node]bool{}
		for _, n := range order {
			if n.Parent != nil {
				parents[n.Parent] = true
			}
		}
		for p := range parents {
			snapChildren(p, func(k *html.Node, s nodeSnap) {
				if affected[k] {
					c.snaps[k] = s
				}
			})
		}
	}
	// the declarations as they are now (the values of custom properties
	// are found again if any of them changed)
	pseudoChanged := map[*html.Node]bool{}
	varsChanged := false
	for _, n := range order {
		ds := c.matchDecls(n)
		queried := false
		for _, d := range ds {
			queried = queried || d.container != nil
		}
		if hasCustomDecls(c.styles[n]) || hasCustomDecls(ds) {
			varsChanged = true
			c.varMemo = nil // (as the elements in it inherit them)
		}
		c.styles[n] = ds
		delete(c.sorted, n)
		delete(c.hinted, n) // its attributes are added again
		if queried {
			if c.containerQueried == nil {
				c.containerQueried = map[*html.Node]bool{}
			}
			c.containerQueried[n] = true
		} else if c.containerQueried != nil {
			delete(c.containerQueried, n)
		}
		for _, kind := range []string{"before", "after"} {
			nd := c.matchPseudo(n, kind)
			od := saved[n].before
			if kind == "after" {
				od = saved[n].after
			}
			if samePseudo(od, nd) {
				continue
			}
			if c.undisplayed(n) {
				// not rendered (as the icons of the video players of
				// npr.org in a hidden dialog, which changed as they
				// played, each time reading the page whole, as no element
				// that they are in had widgets of its own)
				c.setPseudo(n, kind, nd, len(nd) > 0)
				continue
			}
			if !isFree(n) {
				undo()
				// its pseudo-element, which may be a widget
				return nil, c.noChange("a change of the ::" + kind + " of " + describeNode(n))
			}
			c.setPseudo(n, kind, nd, len(nd) > 0)
			pseudoChanged[n] = true
		}
	}
	if varsChanged {
		c.varMemo = nil
	}
	u := &declUpdate{undo: undo, freeChanged: pseudoChanged}
	for _, n := range order {
		if inHead(n) || neverRendered(n) {
			// not rendered (as a style element that is added, or the
			// source elements of a picture, whose display the slideshows
			// of npr.org changed many times a second, each reading the
			// page again)
			continue
		}
		nw := effectiveDecls(c.declarations(n))
		props := changedProps(old[n], nw)
		if len(props) == 0 {
			continue
		}
		if oldDisplay[n] != "none" && c.displayOf(n) != "none" {
			u.transitions = append(u.transitions, transitionsOf(n, nw, props)...)
		}
		if isFree(n) {
			u.freeChanged[n] = true
			continue
		}
		if c.hiddenAbove(n, affected, oldDisplay) {
			continue // not rendered, before and after
		}
		ws := c.elementWidgets[n]
		if d := c.displayOf(n); d != oldDisplay[n] {
			undo()
			return nil, c.noChange(fmt.Sprintf("display of %s from %q to %q", describeNode(n), oldDisplay[n], d))
		}
		if len(ws) == 0 && !c.undisplayed(n) && !c.unusedCustomProps(props) {
			undo()
			return nil, c.noChange(fmt.Sprintf("%s of %s, which has no widget of its own", strings.Join(props, ", "), describeNode(n)))
		}
		for _, p := range props {
			if p == "position" && inFlowPosition(old[n][p]) && inFlowPosition(nw[p]) {
				// static and relative are both in the flow, so the widget
				// is restyled for the change, with the layout (as for the
				// images of npr.org as they load, hundreds of times)
				u.layout = true
				continue
			}
			if !c.restylable(n, p) {
				undo()
				return nil, c.noChange(fmt.Sprintf("%s of %s", p, describeNode(n)))
			}
			u.layout = u.layout || !paintProperties[p]
		}
		u.widgets = append(u.widgets, ws...)
	}
	for _, n := range whole {
		if d := c.displayOf(n); d != oldDisplay[n] {
			undo()
			// it may not have a frame of its own
			return nil, c.noChange(fmt.Sprintf("display of %s from %q to %q", describeNode(n), oldDisplay[n], d))
		}
	}
	// the declarations of the elements, with those of their attributes, as
	// the widgets are styled with them (not the attributes of the elements
	// then, which the scripts of a page may be changing meanwhile)
	for _, n := range order {
		c.declarations(n)
	}
	if html := documentElement(root); html != nil && affected[html] {
		// all of the elements have been matched with all of the rules (as
		// with those added to style sheets; see Context.ReuseStyles)
		c.matchedRules = len(c.rules)
		c.matchedPseudo = len(c.pseudoRules)
	}
	return u, true
}

// setPseudo sets the declarations of the given pseudo-element of the
// given element, or deletes them if it has none.
func (c *Context) setPseudo(n *html.Node, kind string, ds []*cssDecl, has bool) {
	key := pseudoKey{n, kind}
	if !has {
		delete(c.pseudoStyles, key)
		return
	}
	if c.pseudoStyles == nil {
		c.pseudoStyles = map[pseudoKey][]*cssDecl{}
	}
	c.pseudoStyles[key] = ds
}

// samePseudo returns whether the given declarations of a pseudo-element
// are the same.
func samePseudo(a, b []*cssDecl) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Property != b[i].Property || a[i].Value != b[i].Value || a[i].Important != b[i].Important {
			return false
		}
	}
	return true
}

// keyIn returns whether the given element has the key of the selector
// (see [compiledSelector.key]), which every element that it selects has,
// or it has none.
func (cs *compiledSelector) keyIn(n *html.Node) bool {
	switch cs.keyKind {
	case '#':
		return GetAttr(n, "id") == cs.key
	case '.':
		for _, cl := range strings.Fields(GetAttr(n, "class")) {
			if cl == cs.key {
				return true
			}
		}
		return false
	case 't':
		return strings.EqualFold(n.Data, cs.key)
	}
	return true
}

// relationalChanges adds the elements that the rules of :has() selectors
// select now, or did, to the elements whose declarations may change,
// with add, returning false if there are too many of them. A change of an
// element (in changed) only changes what :has() matches for the elements
// that it is in and their previous siblings (for :has(+ x) and :has(~ x)),
// so those are the only ones that the rules whose :has() selectors are in
// their subject compound are matched against. The others are matched
// against all of the elements. (All of them for all of the rules, about
// 600 of them, over the 5000 elements of npr.org, took long enough on each
// read of the layout by its scripts that the page hung.)
func (c *Context) relationalChanges(root *html.Node, changed []*html.Node, add func(n *html.Node) bool) bool {
	ix := c.index()
	if len(ix.relational) == 0 && len(ix.subjectHas) == 0 && len(ix.hasAbove) == 0 {
		return true
	}
	check := func(n *html.Node, rules []int) bool {
		if n.Type != html.ElementNode {
			return true
		}
		ds := c.styles[n]
		had := func(i int) bool { // (without a map, as it is called for each rule)
			for _, d := range ds {
				if d.rule-1 == i {
					return true
				}
			}
			return false
		}
		for _, i := range rules {
			r := c.rules[i]
			// (the key of its selector first, which most elements do not
			// have, then its scope, after a match, as it walks up)
			if had(i) != (r.sel.keyIn(n) && c.selMatch(r.sel, n) && contains(r.root, n)) {
				return add(n)
			}
		}
		return true
	}
	if len(ix.relational) > 0 {
		var walk func(n *html.Node) bool
		walk = func(n *html.Node) bool {
			if !check(n, ix.relational) {
				return false
			}
			for k := n.FirstChild; k != nil; k = k.NextSibling {
				if !walk(k) {
					return false
				}
			}
			return true
		}
		if !walk(root) {
			return false
		}
	}
	if len(ix.subjectHas) == 0 && len(ix.hasAbove) == 0 {
		return true
	}
	// the rules whose :has() is in another compound are matched against
	// the elements in those that match it (without its :has()), and their
	// later siblings if a sibling combinator follows it: the elements that
	// match them are found first (in above), then the elements in each are
	// walked once, matched only against the rules whose key they have
	// (most of them are matched in the body, as of npr.org)
	regions := map[*html.Node][]int{} // the rules of each
	var regionOrder []*html.Node
	above := func(h *html.Node) bool {
		if h.Type != html.ElementNode {
			return true
		}
		for _, i := range ix.hasAbove {
			if c.rules[i].hasCompound.sel.Match(h) {
				if _, ok := regions[h]; !ok {
					regionOrder = append(regionOrder, h)
				}
				regions[h] = append(regions[h], i)
			}
		}
		return true
	}
	walkRegions := func() bool {
		inOuter := func(h *html.Node, i int) bool { // (walked with an outer region)
			for p := h.Parent; p != nil; p = p.Parent {
				if slices.Contains(regions[p], i) {
					return true
				}
			}
			return false
		}
		for _, h := range regionOrder {
			var rules []int
			for _, i := range regions[h] {
				if !inOuter(h, i) {
					rules = append(rules, i)
				}
			}
			if len(rules) == 0 {
				continue
			}
			sub := &ruleIndex{byID: map[string][]int{}, byClass: map[string][]int{}, byTag: map[string][]int{}}
			siblings := false
			for _, i := range rules {
				r := c.rules[i]
				siblings = siblings || r.hasSiblings
				switch r.sel.keyKind {
				case '#':
					sub.byID[r.sel.key] = append(sub.byID[r.sel.key], i)
				case '.':
					sub.byClass[r.sel.key] = append(sub.byClass[r.sel.key], i)
				case 't':
					sub.byTag[r.sel.key] = append(sub.byTag[r.sel.key], i)
				default:
					sub.other = append(sub.other, i)
				}
			}
			var walk func(n *html.Node) bool
			walk = func(n *html.Node) bool {
				if n.Type == html.ElementNode && !check(n, sub.candidates(n)) {
					return false
				}
				for k := n.FirstChild; k != nil; k = k.NextSibling {
					if !walk(k) {
						return false
					}
				}
				return true
			}
			for k := h.FirstChild; k != nil; k = k.NextSibling {
				if !walk(k) {
					return false
				}
			}
			if siblings {
				for k := h.NextSibling; k != nil; k = k.NextSibling {
					if !walk(k) {
						return false
					}
				}
			}
		}
		return true
	}
	seen := map[*html.Node]bool{}
	for _, ch := range changed {
		if ch == nil {
			continue
		}
		// (from its last sibling: those after it changed too)
		last := ch
		for last.NextSibling != nil {
			last = last.NextSibling
		}
		for a := last; a != nil; a = a.Parent {
			for s := a; s != nil; s = s.PrevSibling {
				if seen[s] {
					continue
				}
				seen[s] = true
				if !check(s, ix.subjectHas) || !above(s) {
					return false
				}
			}
		}
	}
	return walkRegions()
}

// hasCompound returns, for the given selector (of one complex selector)
// whose :has() selectors are all in one compound that is not its last,
// that compound without them, compiled, and whether a sibling combinator
// follows it, or nil if they are not (or it does not compile).
func hasCompound(sel string) (*compiledSelector, bool) {
	at := strings.Index(sel, ":has(")
	combs := combinators(sel)
	start, end, siblings := 0, len(sel), false
	for _, cb := range combs {
		switch {
		case cb.at <= at:
			start = cb.at + 1
		case end == len(sel):
			end = cb.start
			fallthrough
		default:
			siblings = siblings || cb.sibling
		}
	}
	if strings.Contains(sel[end:], ":has(") {
		return nil, false
	}
	// the compound without its pseudo-classes that have :has() selectors
	// (as :where(:not(:has(.c))) in .a.b:where(:not(:has(.c))) .x), which
	// is what an element must match for them to matter
	comp := sel[start:end]
	var b strings.Builder
	for k := 0; k < len(comp); {
		if comp[k] != ':' || k+1 >= len(comp) || comp[k+1] == ':' {
			b.WriteByte(comp[k])
			k++
			continue
		}
		// a pseudo-class, to the end of its arguments if it has any
		e := k + 1
		for e < len(comp) && (comp[e] == '-' || comp[e] == '_' || comp[e] >= 'a' && comp[e] <= 'z' || comp[e] >= 'A' && comp[e] <= 'Z' || comp[e] >= '0' && comp[e] <= '9') {
			e++
		}
		if e < len(comp) && comp[e] == '(' {
			depth := 0
			for ; e < len(comp); e++ {
				if comp[e] == '(' {
					depth++
				} else if comp[e] == ')' {
					if depth--; depth == 0 {
						e++
						break
					}
				}
			}
		}
		if !strings.Contains(comp[k:e], ":has(") {
			b.WriteString(comp[k:e])
		}
		k = e
	}
	compound := strings.TrimSpace(b.String())
	if compound == "" {
		compound = "*"
	}
	cs := compileSelector(compound)
	if cs == nil {
		return nil, false
	}
	return cs, siblings
}

// combinator is a combinator of a selector, from the index of its first
// character (start) to that of its last (at), which is a sibling
// combinator (+ or ~) or not (a descendant or child one).
type combinator struct {
	start, at int
	sibling   bool
}

// combinators returns the combinators of the given selector (of one
// complex selector) that are not in parentheses, brackets or quotes, one
// for each run of them (as " > ").
func combinators(sel string) []combinator {
	var combs []combinator
	depth := 0
	var quote byte
	in := false // in a run of combinator characters
	for i := 0; i < len(sel); i++ {
		ch := sel[i]
		isComb := false
		switch {
		case quote != 0:
			if ch == '\\' {
				i++
			} else if ch == quote {
				quote = 0
			}
		case ch == '"' || ch == '\'':
			quote = ch
		case ch == '\\':
			i++
		case ch == '(' || ch == '[':
			depth++
		case ch == ')' || ch == ']':
			depth--
		case depth == 0 && (ch == ' ' || ch == '>' || ch == '+' || ch == '~' || ch == '\t' || ch == '\n'):
			isComb = true
			if !in {
				combs = append(combs, combinator{start: i, at: i})
			}
			if ch == '+' || ch == '~' {
				combs[len(combs)-1].sibling = true
			}
			combs[len(combs)-1].at = i // (its last character)
		}
		in = isComb
	}
	return combs
}

// subjectStart returns the index in the given selector (of one complex
// selector) of its last compound selector, the one of the elements that it
// selects: after its last combinator that is not in parentheses, brackets
// or quotes.
func subjectStart(sel string) int {
	combs := combinators(sel)
	if len(combs) == 0 {
		return 0
	}
	return combs[len(combs)-1].at + 1
}

// inForeign returns whether the given element is in an svg or math element.
func inForeign(n *html.Node) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && (p.Namespace != "" || p.Data == "svg" || p.Data == "math") {
			return true
		}
	}
	return false
}

// hiddenAbove returns whether an element that the given element is in
// has a display of none, and had one before the declarations of the given
// affected elements changed (with the given displays before), so that the
// given element was not rendered, nor is (see [Context.updateDecls]).
func (c *Context) hiddenAbove(n *html.Node, affected map[*html.Node]bool, oldDisplay map[*html.Node]string) bool {
	for p := n.Parent; p != nil && p.Type == html.ElementNode; p = p.Parent {
		if c.displayOf(p) == "none" && (!affected[p] || oldDisplay[p] == "none") {
			return true
		}
	}
	return false
}

// undisplayed returns whether the given element is not displayed (it or
// an element that it is in has a display of none), so that it has no
// widget.
func (c *Context) undisplayed(n *html.Node) bool {
	for p := n; p != nil && p.Type == html.ElementNode; p = p.Parent {
		if c.displayOf(p) == "none" {
			return true
		}
	}
	return false
}

// effectiveDecls returns the values of the properties of the given
// declarations, in the order of the cascade, that apply (the last of each).
func effectiveDecls(ds []*cssDecl) map[string]string {
	m := make(map[string]string, len(ds))
	for _, d := range ds {
		v := d.Value
		if d.invalid {
			v = "\x00invalid"
		}
		m[d.Property] = v
	}
	return m
}

// changedProps returns the properties whose values differ.
func changedProps(a, b map[string]string) []string {
	var props []string
	for p, v := range a {
		if bv, ok := b[p]; !ok || bv != v {
			props = append(props, p)
		}
	}
	for p := range b {
		if _, ok := a[p]; !ok {
			props = append(props, p)
		}
	}
	return props
}

// restylable returns whether a change of the given property of the given
// element can be made by styling its widgets again, without reading the
// page again: it only applies in styling them, not in reading the page.
func (c *Context) restylable(n *html.Node, prop string) bool {
	if strings.HasPrefix(prop, "--") {
		return true // its uses are changes of their own
	}
	if p, ok := strings.CutPrefix(prop, "-webkit-"); ok && webkitAliases[p] {
		prop = p
	}
	if !restylableProperties[prop] {
		return false
	}
	if c.displayOf(n) == "inline" {
		// an inline element with a widget is a box by its padding and
		// border (see isInlineBox), and its text is in a run
		return paintProperties[prop] && prop != "color" && prop != "background-color" && prop != "background"
	}
	if prop == "filter" && n.Data == "img" {
		return false // in the image as it is read
	}
	if n.Data == "tr" && strings.HasPrefix(prop, "background") {
		return false // in the backgrounds of its cells as they are read
	}
	if c.arranged(n) {
		// the sizes of the items of tables, grids and columns, which are
		// arranged as they are read (as are those hidden in the stacks of
		// grids, by their opacity and visibility)
		return paintProperties[prop] && prop != "opacity" && prop != "visibility"
	}
	switch n.Data {
	case "input", "select", "textarea", "button", "img", "iframe", "video", "audio", "canvas", "object", "embed":
		// the widgets of controls and replaced elements, which are made
		// for their styles in part
		return paintProperties[prop] || prop == "margin" || strings.HasPrefix(prop, "margin-") ||
			prop == "top" || prop == "bottom" || prop == "left" || prop == "right" || prop == "inset"
	}
	return true
}

// inFlowPosition returns whether the given value of the position property
// (as declared, "" for none) keeps an element in the flow of the layout of
// its parent (static and relative, not absolute, fixed, or sticky).
func inFlowPosition(v string) bool {
	v = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(v), "!important"))
	return v == "" || v == "static" || v == "relative" || v == "initial" || v == "unset" || v == "revert"
}

// arranged returns whether the given element is or is in a table, a grid,
// or an element of columns, which are arranged as they are read.
func (c *Context) arranged(n *html.Node) bool {
	switch n.Data {
	case "table", "td", "th", "tr", "col", "colgroup", "caption", "thead", "tbody", "tfoot":
		return true
	}
	for _, e := range []*html.Node{n, n.Parent} {
		if e == nil || e.Type != html.ElementNode {
			continue
		}
		if d := c.displayOf(e); strings.Contains(d, "grid") || strings.Contains(d, "table") {
			return true
		}
		for _, p := range []string{"columns", "column-count", "column-width"} {
			if v := c.declaration(e, p); v != "" && v != "auto" {
				return true
			}
		}
	}
	return false
}

// paintProperties are the properties whose changes only change how the
// widgets are rendered, not their sizes or positions in their layout
// (transforms are applied to the positions of widgets in the layout).
var paintProperties = map[string]bool{
	"color": true, "background-color": true, "background": true, "opacity": true,
	"visibility": true, "cursor": true, "pointer-events": true, "z-index": true,
	"box-shadow": true, "outline": true, "outline-color": true, "outline-style": true, "outline-width": true, "outline-offset": true,
	"border-color": true, "border-top-color": true, "border-right-color": true, "border-bottom-color": true, "border-left-color": true,
	"border-radius": true, "border-top-left-radius": true, "border-top-right-radius": true, "border-bottom-left-radius": true, "border-bottom-right-radius": true,
	"text-decoration": true, "text-decoration-line": true, "text-decoration-color": true, "text-decoration-style": true,
	"filter": true, "backdrop-filter": true, "will-change": true, "user-select": true,
	"transition": true, "transition-property": true, "transition-duration": true, "transition-delay": true, "transition-timing-function": true,
	"animation": true, "animation-name": true, "animation-duration": true, "animation-delay": true, "animation-iteration-count": true,
	"animation-fill-mode": true, "animation-play-state": true, "animation-timing-function": true, "animation-direction": true,
	"fill": true, "stroke": true, "caret-color": true, "accent-color": true, "outline-radius": true,
}

// restylableProperties are the properties whose changes can be made by
// styling the widgets of elements again (see [Context.restylable]): those
// that apply in styling them, not in reading the page.
var restylableProperties = func() map[string]bool {
	m := map[string]bool{
		"width": true, "height": true, "min-width": true, "min-height": true, "max-width": true, "max-height": true,
		"inline-size": true, "block-size": true, "min-inline-size": true, "min-block-size": true, "max-inline-size": true, "max-block-size": true,
		"top": true, "right": true, "bottom": true, "left": true, "inset": true,
		"inset-block": true, "inset-inline": true, "inset-block-start": true, "inset-block-end": true, "inset-inline-start": true, "inset-inline-end": true,
		"margin": true, "margin-top": true, "margin-right": true, "margin-bottom": true, "margin-left": true,
		"margin-block": true, "margin-inline": true, "margin-block-start": true, "margin-block-end": true, "margin-inline-start": true, "margin-inline-end": true,
		"padding": true, "padding-top": true, "padding-right": true, "padding-bottom": true, "padding-left": true,
		"padding-block": true, "padding-inline": true, "padding-block-start": true, "padding-block-end": true, "padding-inline-start": true, "padding-inline-end": true,
		"border": true, "border-width": true, "border-style": true,
		"border-top": true, "border-right": true, "border-bottom": true, "border-left": true,
		"border-top-width": true, "border-right-width": true, "border-bottom-width": true, "border-left-width": true,
		"border-top-style": true, "border-right-style": true, "border-bottom-style": true, "border-left-style": true,
		"overflow": true, "overflow-x": true, "overflow-y": true,
		"font-weight": true, "font-style": true, "text-align": true,
		"flex": true, "flex-grow": true, "flex-shrink": true, "flex-basis": true,
		"align-items": true, "align-self": true, "justify-content": true, "align-content": true, "gap": true, "row-gap": true, "column-gap": true,
		"box-sizing": true,
		"transform":  true, "translate": true, "rotate": true, "scale": true, "transform-origin": true,
	}
	for p := range paintProperties {
		m[p] = true
	}
	return m
}()

// inHead returns whether the given node is the head element or in it.
func inHead(n *html.Node) bool {
	for p := n; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.DataAtom == atom.Head {
			return true
		}
	}
	return false
}

// unusedCustomProps returns whether the given properties are all custom
// properties that no declaration of the page uses (with var()), so that
// their changes change nothing (as the --vwo-el-* properties that the
// A/B testing scripts of volkskrant.nl set on the html element, which has
// no widget of its own, for which the page was read whole).
func (c *Context) unusedCustomProps(props []string) bool {
	for _, p := range props {
		if !strings.HasPrefix(p, "--") {
			return false
		}
	}
	for _, p := range props {
		if c.varsUsed[p] {
			return false
		}
	}
	return true
}

// noteVars records the custom properties that the given value of a
// declaration uses (with var()), for [Context.unusedCustomProps].
func (c *Context) noteVars(value string) {
	for {
		i := strings.Index(value, "var(")
		if i < 0 {
			return
		}
		value = strings.TrimLeft(value[i+4:], " \t\n")
		j := 0
		for j < len(value) && isNameChar(value[j]) {
			j++
		}
		if j > 0 {
			if c.varsUsed == nil {
				c.varsUsed = map[string]bool{}
			}
			c.varsUsed[value[:j]] = true
		}
		value = value[j:]
	}
}

// invalidation returns, for the given element whose attributes have
// changed, whether the elements in it are selected by a rule that may
// select them differently for the change, and whether its siblings after
// it may be, if only its class and id changed (and its style, which
// applies to it alone): the invalidation sets of browsers, as for a class
// added to the body element, which only the rules that have the class
// before their last compound selectors (as .modal-open .modal) select the
// elements in it differently for. It returns ok false, for all of the
// elements in it and after it to be matched again, if it can not tell.
func (c *Context) invalidation(n *html.Node) (selects func(k *html.Node) bool, siblings, ok bool) {
	old, has := c.snaps[n]
	if !has || c.matchAll {
		// (as for rules added to the style sheets)
		return nil, false, false
	}
	var tokens []string
	if old.attrs == attrsHash(n) {
		// not a change of its attributes: of its states (as :hover and
		// :checked), or an attribute set to the value that it had
		tokens = statePseudoClasses
	}
	if strings.Contains(old.style, "--") || strings.Contains(GetAttr(n, "style"), "--") {
		return nil, false, false // custom properties, which the elements in it inherit
	}
	oldClasses, newClasses := strings.Fields(old.class), strings.Fields(GetAttr(n, "class"))
	for _, cl := range oldClasses {
		if !slices.Contains(newClasses, cl) {
			tokens = append(tokens, "."+cl)
		}
	}
	for _, cl := range newClasses {
		if !slices.Contains(oldClasses, cl) {
			tokens = append(tokens, "."+cl)
		}
	}
	if id := GetAttr(n, "id"); id != old.id {
		for _, v := range []string{old.id, id} {
			if v != "" {
				tokens = append(tokens, "#"+v)
			}
		}
	}
	// the other attributes, as [data-state] or [aria-expanded=true]
	var attrTokens []string
	if others := otherAttrs(n); others != old.others {
		ov, nv := attrValues(old.others), attrValues(others)
		for k, v := range ov {
			if w, ok := nv[k]; !ok || w != v {
				attrTokens = append(attrTokens, k)
			}
		}
		for k := range nv {
			if _, ok := ov[k]; !ok {
				attrTokens = append(attrTokens, k)
			}
		}
		for _, a := range attrTokens {
			if a == hoverAttr {
				// (the attribute of :hover, while the states are set)
				tokens = append(tokens, ":hover")
			}
			tokens = append(tokens, attrPseudoClasses[a]...)
		}
	}
	var keys []*compiledSelector
	all := false
	check := func(text string, cs *compiledSelector, decls []*css.Declaration) bool {
		prefix, sib, last := splitLastCompound(text)
		if (len(tokens) > 0 && !sameStates(tokens)) && (strings.Contains(prefix, "[class") || strings.Contains(prefix, "[id")) {
			// (selected by its class or id other than by the names of
			// them, as [class*=icon-])
			if sib {
				siblings = true
			}
			if cs == nil || cs.keyKind == 0 {
				all = true
			} else {
				keys = append(keys, cs)
			}
		}
		for _, a := range attrTokens {
			// (the selectors of attributes, of [name, as with a value)
			inLast, inPrefix := hasAttrSelector(last, a), hasAttrSelector(prefix, a)
			if inLast && hasCustomProps(decls) {
				return false
			}
			if inPrefix {
				if sib {
					siblings = true
				}
				if cs == nil || cs.keyKind == 0 {
					all = true
				} else {
					keys = append(keys, cs)
				}
			}
		}
		for _, t := range tokens {
			if hasToken(last, t) && hasCustomProps(decls) && (sameStates(tokens) || len(attrTokens) > 0 || cs == nil || c.matchChanged(n, old, cs)) {
				// a rule that selects it differently, with custom
				// properties, which the elements in it inherit
				return false
			}
			if !hasToken(prefix, t) {
				continue
			}
			if sib {
				siblings = true
			}
			if cs == nil || cs.keyKind == 0 {
				all = true
			} else {
				keys = append(keys, cs)
			}
		}
		return true
	}
	for _, r := range c.rules {
		if !check(r.text, r.sel, r.decls) {
			return nil, false, false
		}
	}
	for _, r := range c.pseudoRules {
		if !check(r.text, r.sel, nil) {
			return nil, false, false
		}
	}
	selects = func(k *html.Node) bool {
		if all {
			return true
		}
		for _, cs := range keys {
			if cs.keyIn(k) {
				return true
			}
		}
		return false
	}
	return selects, siblings, true
}

// splitLastCompound returns the given selector before its last compound
// selector, whether it has a combinator of siblings (+ or ~) there, and
// the last compound selector.
func splitLastCompound(sel string) (prefix string, siblings bool, last string) {
	depth, at := 0, -1
	for i := 0; i < len(sel); i++ {
		switch sel[i] {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		case ' ', '>', '+', '~':
			if depth == 0 {
				at = i
			}
		}
	}
	if at < 0 {
		return "", false, sel
	}
	prefix = sel[:at+1]
	return prefix, strings.ContainsAny(prefix, "+~"), sel[at+1:]
}

// hasToken returns whether the given selector text has the given class
// (.name) or id (#name), not as the start of a longer name.
func hasToken(text, token string) bool {
	for i := 0; ; {
		j := strings.Index(text[i:], token)
		if j < 0 {
			return false
		}
		end := i + j + len(token)
		if end >= len(text) || !isNameChar(text[end]) {
			return true
		}
		i = end
	}
}

// hasAttrSelector returns whether the given selector text selects elements
// by the given attribute ([name, [name=, [name~= and so on).
func hasAttrSelector(text, name string) bool {
	for i := 0; ; {
		j := strings.Index(text[i:], "[")
		if j < 0 {
			return false
		}
		rest := strings.TrimLeft(text[i+j+1:], " ")
		if strings.HasPrefix(rest, name) && (len(rest) == len(name) || !isNameChar(rest[len(name)])) {
			return true
		}
		i += j + 1
	}
}

// hasCustomProps returns whether the given declarations have custom
// properties (--name).
func hasCustomProps(decls []*css.Declaration) bool {
	for _, d := range decls {
		if strings.HasPrefix(strings.TrimSpace(d.Property), "--") {
			return true
		}
	}
	return false
}

// attrPseudoClasses are the pseudo-classes that select elements by the
// given attributes (see [Context.invalidation]).
var attrPseudoClasses = map[string][]string{
	"disabled":        {":disabled", ":enabled"},
	"checked":         {":checked", ":indeterminate", ":default"},
	"selected":        {":checked", ":default"},
	"required":        {":required", ":optional"},
	"readonly":        {":read-only", ":read-write"},
	"open":            {":open", ":closed"},
	"lang":            {":lang"},
	"dir":             {":dir"},
	"href":            {":link", ":any-link", ":visited"},
	"placeholder":     {":placeholder-shown"},
	"value":           {":placeholder-shown", ":valid", ":invalid"},
	"type":            {":checked", ":valid", ":invalid", ":read-only", ":read-write"},
	"contenteditable": {":read-only", ":read-write"},
	"hidden":          {},
}

// statePseudoClasses are the pseudo-classes of the states of elements,
// which a change of them may select others by (see [Context.invalidation]).
var statePseudoClasses = []string{":hover", ":checked", ":focus", ":focus-within", ":focus-visible", ":active", ":target", ":default", ":indeterminate", ":open"}

// sameStates returns whether the given tokens are those of the states
// (see [statePseudoClasses]).
func sameStates(tokens []string) bool {
	return len(tokens) > 0 && &tokens[0] == &statePseudoClasses[0]
}

// matchChanged returns whether the given selector matches the given
// element differently with its class and id now and with those of the
// given state of it (as before they changed).
func (c *Context) matchChanged(n *html.Node, old nodeSnap, cs *compiledSelector) bool {
	now := cs.sel.Match(n)
	saved := n.Attr
	attrs := make([]html.Attribute, 0, len(saved)+2)
	for _, a := range saved {
		if a.Namespace == "" && (a.Key == "class" || a.Key == "id") {
			continue
		}
		attrs = append(attrs, a)
	}
	if old.class != "" {
		attrs = append(attrs, html.Attribute{Key: "class", Val: old.class})
	}
	if old.id != "" {
		attrs = append(attrs, html.Attribute{Key: "id", Val: old.id})
	}
	n.Attr = attrs
	before := cs.sel.Match(n)
	n.Attr = saved
	return now != before
}

// neverRendered returns whether the given element never has a box of its
// own, whatever its styles: the sources and tracks of media, and the
// elements of scripts and of the head.
func neverRendered(n *html.Node) bool {
	switch n.DataAtom {
	case atom.Source, atom.Track, atom.Script, atom.Noscript, atom.Template, atom.Style, atom.Link, atom.Meta, atom.Base, atom.Param:
		return true
	}
	return false
}

// hasCustomDecls returns whether the given declarations have custom
// properties.
func hasCustomDecls(ds []*cssDecl) bool {
	for _, d := range ds {
		if strings.HasPrefix(d.Property, "--") {
			return true
		}
	}
	return false
}
