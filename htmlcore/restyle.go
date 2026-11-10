// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"slices"
	"strings"

	"cogentcore.org/core/core"
	"github.com/aymerick/douceur/css"
	"golang.org/x/net/html"
)

// styleRule is a style rule that has been added to a context, with one
// selector, by which the declarations of elements are found again when
// they change (see [Context.Restyle]).
type styleRule struct {
	root   *html.Node
	sel    *compiledSelector
	decls  []*css.Declaration
	origin cssOrigin
	spec   int

	// container is the condition of the @container rule that it is in.
	container *containerCond

	// cssBase is the URL that the URLs of the rule are relative to.
	cssBase string

	// relational is whether its selector has :has().
	relational bool
}

// pseudoRule is a style rule of a ::before or ::after pseudo-element that
// has been added to a context (see [styleRule]).
type pseudoRule struct {
	root   *html.Node
	sel    *compiledSelector
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

	// relational has the rules with :has() selectors.
	relational []int

	// n is the number of rules indexed.
	n int
}

// recordRule records the given style rule (of one selector), which is
// being added, as the rule of the declarations that it adds (see
// [Context.Restyle]).
func (c *Context) recordRule(root *html.Node, sel string, cs *compiledSelector, decls []*css.Declaration, origin cssOrigin, spec int) {
	has := strings.Contains(sel, ":has(")
	c.rules = append(c.rules, &styleRule{root: root, sel: cs, decls: decls, origin: origin, spec: spec, container: c.curContainer, cssBase: c.cssBase, relational: has})
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
		if r.relational {
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
		if !contains(r.root, n) || !r.sel.sel.Match(n) {
			continue
		}
		c.cssBase = r.cssBase // for its URLs
		for _, d := range r.decls {
			prop, value := strings.ToLower(strings.TrimSpace(d.Property)), strings.TrimSpace(d.Value)
			if !validDeclaration(prop, value) {
				continue
			}
			c.cssOrder++
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
		if r.kind != kind || r.sel == nil || !contains(r.root, n) || !r.sel.sel.Match(n) {
			continue
		}
		for _, d := range r.decls {
			c.cssOrder++
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
}

// apply styles the widgets of the update again, and lays them out again
// if needed.
func (u *declUpdate) apply(c *Context) {
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
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			if !add(k) {
				return false
			}
		}
		return true
	}
	for _, n := range whole {
		if !add(n) {
			return nil, false
		}
	}
	for _, n := range changed {
		if n == nil || n.Type != html.ElementNode || rootNode(n) != root {
			return nil, false
		}
		if (n.Namespace != "" || inForeign(n)) && !isFree(n) {
			return nil, false // svg and math, which are read differently
		}
		for s := n; s != nil; s = s.NextSibling {
			if !add(s) {
				return nil, false
			}
		}
	}
	restore := c.setCheckedAttrs(root) // for :checked and :hover
	defer restore()
	if c.relational {
		// the elements that :has() selectors select, which may be any,
		// whose matches have changed
		if !c.relationalChanges(root, add) {
			return nil, false
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
		ns.before, ns.hasBefore = c.pseudoStyles[pseudoKey{n, "before"}]
		ns.after, ns.hasAfter = c.pseudoStyles[pseudoKey{n, "after"}]
		saved[n] = ns
	}
	undo := func() {
		for n, ns := range saved {
			c.styles[n], c.sorted[n], c.hinted[n] = ns.styles, ns.sorted, ns.hinted
			if c.containerQueried != nil {
				c.containerQueried[n] = ns.queried
			}
			c.setPseudo(n, "before", ns.before, ns.hasBefore)
			c.setPseudo(n, "after", ns.after, ns.hasAfter)
		}
	}
	// the declarations as they are now
	pseudoChanged := map[*html.Node]bool{}
	for _, n := range order {
		ds := c.matchDecls(n)
		queried := false
		for _, d := range ds {
			queried = queried || d.container != nil
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
			if !isFree(n) {
				undo()
				return nil, false // its pseudo-element, which may be a widget
			}
			c.setPseudo(n, kind, nd, len(nd) > 0)
			pseudoChanged[n] = true
		}
	}
	u := &declUpdate{undo: undo, freeChanged: pseudoChanged}
	for _, n := range order {
		nw := effectiveDecls(c.declarations(n))
		props := changedProps(old[n], nw)
		if len(props) == 0 {
			continue
		}
		if isFree(n) {
			u.freeChanged[n] = true
			continue
		}
		ws := c.elementWidgets[n]
		if c.displayOf(n) != oldDisplay[n] || (len(ws) == 0 && !c.undisplayed(n)) {
			undo()
			return nil, false
		}
		for _, p := range props {
			if !c.restylable(n, p) {
				undo()
				return nil, false
			}
			u.layout = u.layout || !paintProperties[p]
		}
		u.widgets = append(u.widgets, ws...)
	}
	for _, n := range whole {
		if c.displayOf(n) != oldDisplay[n] {
			undo()
			return nil, false // it may not have a frame of its own
		}
	}
	// the declarations of the elements, with those of their attributes, as
	// the widgets are styled with them (not the attributes of the elements
	// then, which the scripts of a page may be changing meanwhile)
	for _, n := range order {
		c.declarations(n)
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

// relationalChanges adds the elements that the rules of :has() selectors
// select now, or did, to the elements whose declarations may change,
// with add, returning false if there are too many of them.
func (c *Context) relationalChanges(root *html.Node, add func(n *html.Node) bool) bool {
	rel := c.index().relational
	if len(rel) == 0 {
		return true
	}
	var walk func(n *html.Node) bool
	walk = func(n *html.Node) bool {
		if n.Type == html.ElementNode {
			had := map[int]bool{}
			for _, d := range c.styles[n] {
				had[d.rule-1] = true
			}
			for _, i := range rel {
				r := c.rules[i]
				if had[i] != (contains(r.root, n) && r.sel.sel.Match(n)) {
					if !add(n) {
						return false
					}
					break
				}
			}
		}
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			if !walk(k) {
				return false
			}
		}
		return true
	}
	return walk(root)
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
