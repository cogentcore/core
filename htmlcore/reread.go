// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strconv"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// maxReread is the number of elements beyond which the elements that
// [Context.Reread] would read again are read with the rest of the page:
// they are read with the lock of the page held (unlike a page that is
// read again whole, whose new widgets are only shown with it held), which
// takes about 0.1ms for each (as for theguardian.com, whose body of 5000
// elements took 650ms, while the widgets of all of it were shown and laid
// out in about 100ms). On a page of more elements, a third of them may be
// read again (see [Context.rereadLimit]).
const maxReread = 1000

// rereadLimit returns the number of elements beyond which the elements to
// read again are read with the rest of the page: [maxReread], or a third
// of the elements of a large page, which takes far longer to read again
// whole (as the pull request files of GitHub, of 35000 elements, read
// again whole in 9s for each 3000 elements of the diffs that it loads as
// it is scrolled, which seemed stuck).
func (c *Context) rereadLimit(root *html.Node) int {
	n := 0
	if root != nil {
		walkElements(root, func(*html.Node) { n++ })
	}
	return max(maxReread, n/3)
}

// recordSheet records the given style or link element as one whose style
// sheet has been added (or not, as for its media), so that a page with a
// new one is read again whole (see [Context.Reread]).
func (c *Context) recordSheet(n *html.Node) {
	if c.sheets == nil {
		c.sheets = map[*html.Node]bool{}
	}
	c.sheets[n] = true
}

// Reread updates the widgets of the page that the context has read for
// changes of its document, as scripts make, without reading all of it
// again (which makes new widgets for all of it): self has the elements
// that have changed themselves (as their attributes), and children those
// whose children have changed (as nodes added to them, or the text of
// their text nodes). The nearest element of each that has a frame of its
// own, whose widgets can be made alone (as one that is not a cell of a
// table or an item of a grid), is read again, into new widgets, which
// are reconciled with the old ones with [tree.Plan]: the widgets of the
// elements that have not changed, nor those in them, are kept, as are the
// frames of the elements that have not changed themselves, whose children
// are reconciled in turn, and the others are replaced. The declarations
// of the elements are found again as for [Context.Restyle], and the
// widgets of those that are not read again are styled again. It returns
// false, leaving the page as it was, if the changes can not be made so,
// for the page to be read again: for a new style sheet, a change of the
// html element itself, or more elements to read than [Context.rereadLimit].
func (c *Context) Reread(self, children []*html.Node) bool {
	if len(self) == 0 && len(children) == 0 {
		return true
	}
	if c.reading || c.rest != nil || c.elementWidgets == nil {
		return false
	}
	// the changed elements (that of a text node is its parent, whose
	// children have changed)
	var root *html.Node
	selfDirty, childDirty := map[*html.Node]bool{}, map[*html.Node]bool{}
	var changed []*html.Node // for the declarations of the elements after them
	var roots []*html.Node
	addChanged := func(n *html.Node, own bool) bool {
		if n != nil && n.Type == html.CommentNode || !own && n != nil && n.Type == html.ElementNode && unshown(n) {
			return true // (not rendered)
		}
		if n != nil && n.Type != html.ElementNode {
			n, own = n.Parent, false
		}
		if own && unshown(n) {
			return true
		}
		if n == nil || n.Type != html.ElementNode {
			return false
		}
		if root == nil {
			root = rootNode(n)
		} else if rootNode(n) != root {
			return false
		}
		// (an element that has changed itself may have another box, as
		// for its display, in that of its parent)
		from := n
		if own {
			from = n.Parent
		}
		r := c.rereadRoot(from)
		if r == nil {
			return false
		}
		roots = append(roots, r)
		if own {
			selfDirty[n] = true
			changed = append(changed, n)
		} else {
			childDirty[n] = true
		}
		return true
	}
	for _, n := range self {
		if !addChanged(n, true) {
			return false
		}
	}
	for _, n := range children {
		if !addChanged(n, false) {
			return false
		}
	}
	roots = outermost(roots)
	inRoots := func(n *html.Node) bool {
		for _, r := range roots {
			if contains(r, n) {
				return true
			}
		}
		return false
	}
	if len(roots) == 0 {
		return true // (only elements that are not rendered changed)
	}
	// the elements to read are not too many, have no new style sheets,
	// and their widgets are all in their frames, which have only theirs
	size, limit := 0, c.rereadLimit(root)
	for _, r := range roots {
		ok := true
		walkElements(r, func(n *html.Node) {
			size++
			if (n.Data == "style" || n.Data == "link") && !c.sheets[n] {
				ok = false
			}
			for _, w := range c.elementWidgets[n] {
				if !widgetIn(w, c.elementWidgets[r][0]) {
					ok = false // (as one moved elsewhere)
				}
			}
		})
		if !ok || size > limit {
			return false
		}
		c.elementWidgets[r][0].AsWidget().WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if n := ownNode(cw); n != nil && rootNode(n) == root && !inRoots(n) {
				ok = false
			}
			return ok
		})
		if !ok {
			return false
		}
	}
	u, ok := c.updateDecls(changed, roots, inRoots)
	if !ok {
		return false
	}
	for n := range u.freeChanged {
		selfDirty[n] = true
	}
	// the elements that have changed or are in elements that have
	touched := map[*html.Node]bool{}
	for _, set := range []map[*html.Node]bool{selfDirty, childDirty} {
		for n := range set {
			for p := n; p != nil && !touched[p]; p = p.Parent {
				touched[p] = true
			}
		}
	}

	c.reading = true
	defer func() { c.reading = false }()
	stage := c.StageSize
	c.StageSize = 0
	defer func() { c.StageSize = stage }()
	c.rereads++
	for _, r := range roots {
		old := c.elementWidgets[r][0]
		pw := old.AsTree().Parent.(core.Widget)
		c.forgetRead(r)
		tmp := core.NewFrame()
		c.inlineParent = nil
		readOneNode(c, tmp, r)
		rc := &reconciler{c: c, touched: touched, selfDirty: selfDirty, pairs: map[core.Widget]core.Widget{}}
		rc.replaceIn(pw, old, tmp.Children)
		rc.remap(r)
	}
	u.apply(c)
	return true
}

// unshown returns whether the given element is never shown, nor its
// children, so that its changes change nothing that is rendered: a
// script, or an element of the head other than those of style sheets and
// the title (as the meta elements that the scripts of github.com change).
func unshown(n *html.Node) bool {
	if n == nil || n.Type != html.ElementNode {
		return false
	}
	switch n.DataAtom {
	case atom.Script, atom.Noscript, atom.Template:
		return true
	case atom.Style, atom.Link, atom.Title, atom.Base:
		return false
	}
	return n.Parent != nil && n.Parent.DataAtom == atom.Head
}

// rereadRoot returns the element that is read again for a change of the
// given element (see [Context.Reread]): the nearest one that it is in (or
// it) that has a frame of its own, in the frame of its parent, whose
// widgets can be made alone (or the body), or nil if there is none (as for
// a change of the html element itself, which is read again whole).
func (c *Context) rereadRoot(n *html.Node) *html.Node {
	for e := n; e != nil && e.Type == html.ElementNode; e = e.Parent {
		switch e.DataAtom {
		case atom.Html, atom.Head, atom.Style, atom.Link, atom.Title, atom.Base, atom.Meta, atom.Template:
			return nil
		}
		if c.rereadable(e) {
			return e
		}
	}
	return nil
}

// rereadable returns whether the widgets of the given element can be made
// alone (see [Context.rereadRoot]).
func (c *Context) rereadable(e *html.Node) bool {
	p := e.Parent
	if p == nil || p.Type != html.ElementNode || e.Namespace != "" || inForeign(e) {
		return false
	}
	ws := c.elementWidgets[e]
	if len(ws) != 1 || ownNode(ws[0]) != e || frameOf(ws[0]) == nil {
		return false
	}
	pw, ok := ws[0].AsTree().Parent.(core.Widget)
	if ok && e.DataAtom == atom.Body {
		return core.AsFrame(pw) != nil // (the widget that the page is read into)
	}
	if !ok || frameOf(pw) == nil || ownNode(pw) != p {
		return false // (as in the frames of columns)
	}
	if c.isInlineLevel(e) || ancestorTable(e) != nil || tableElements[e.Data] {
		return false // in a run of inline content, or in the grid of a table
	}
	if fl := c.declaration(e, "float"); fl != "" && fl != "none" || c.flattened[e] || c.floatSiblings[p] {
		return false
	}
	// the items of grids and columns, which are arranged as they are read
	return !c.arrangedChildren(p)
}

// tableElements are the elements of tables, which are read as the grids
// of their tables.
var tableElements = map[string]bool{"table": true, "td": true, "th": true, "tr": true, "col": true, "colgroup": true,
	"caption": true, "thead": true, "tbody": true, "tfoot": true}

// arrangedChildren returns whether the widgets of the children of the
// given element are arranged once they are read (see [Context.arrange]),
// as in a grid, so that they can not be made alone, nor be kept when
// others are made (see [Context.Reread]).
func (c *Context) arrangedChildren(n *html.Node) bool {
	d := c.displayOf(n)
	if strings.Contains(d, "grid") || strings.Contains(d, "table") || n.Data == "details" || tableElements[n.Data] {
		return true
	}
	if strings.Contains(d, "flex") && strings.HasSuffix(c.flexDirection(n), "-reverse") {
		return true
	}
	for _, p := range []string{"columns", "column-count", "column-width"} {
		if v := c.declaration(n, p); v != "" && v != "auto" {
			return true
		}
	}
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		if k.Type == html.ElementNode && strings.TrimSpace(c.declaration(k, "order")) != "" {
			return true
		}
	}
	return false
}

// forgetRead forgets the state of the reading of the given element and
// those in it, which are read again: their widgets (which are added again
// as they are read), and the state of their tables and floats. Their
// containers start with the sizes that they have.
func (c *Context) forgetRead(r *html.Node) {
	walkElements(r, func(n *html.Node) {
		delete(c.elementWidgets, n)
		delete(c.tables, n)
		delete(c.tableRowState, n)
		delete(c.flattened, n)
		delete(c.floatSiblings, n)
		delete(c.columnUses, n)
		if cf := c.containers[n]; cf != nil && cf.size.X > 0 {
			if c.keptContainers == nil {
				c.keptContainers = map[*html.Node]containerSizes{}
			}
			c.containerMu.Lock()
			c.keptContainers[n] = containerSizes{cf.size, cf.prev}
			c.containerMu.Unlock()
		}
	})
}

// reconciler reconciles the widgets of elements that are read again with
// their old ones (see [Context.Reread]).
type reconciler struct {
	c *Context

	// touched has the elements that have changed, or are in elements
	// that have, and selfDirty those that have changed themselves.
	touched, selfDirty map[*html.Node]bool

	// pairs has the old widget that is kept for each new widget.
	pairs map[core.Widget]core.Widget

	// added has the new widgets that are added in place of old ones,
	// and removed the old widgets that are removed.
	added, removed []core.Widget
}

// replaceIn updates the children of the given parent for the given new
// widgets in place of the given old child.
func (rc *reconciler) replaceIn(parent core.Widget, old core.Widget, news []tree.Node) {
	p := &tree.Plan{EnforceEmpty: true}
	names := childNames(parent)
	for _, k := range parent.AsTree().Children {
		if k != tree.Node(old) {
			rc.keep(p, k)
			continue
		}
		rc.planChildren(p, []tree.Node{old}, news, names)
	}
	p.Update(parent)
	rc.updateAdded()
}

// keep adds the given old child to the given plan, as it is.
func (rc *reconciler) keep(p *tree.Plan, k tree.Node) {
	p.Add(k.AsTree().Name, func() tree.Node { return k }, func(tree.Node) {})
}

// childNames returns the names of the children of the given node.
func childNames(n tree.Node) map[string]bool {
	names := map[string]bool{}
	for _, k := range n.AsTree().Children {
		names[k.AsTree().Name] = true
	}
	return names
}

// planChildren adds to the given plan the given new children, each as
// the old one of the same element and type (among the given old ones)
// where it can be kept, reconciling the children of those kept for
// changes in them; the old ones that are not kept are removed. The new
// ones get names that are not in names (of the children of the parent).
func (rc *reconciler) planChildren(p *tree.Plan, olds, news []tree.Node, names map[string]bool) {
	type key struct {
		n   *html.Node
		typ string
		i   int
	}
	keyOf := func(k tree.Node, counts map[key]int) key {
		w, _ := k.(core.Widget)
		kk := key{typ: k.AsTree().NodeType().Name}
		if w != nil {
			kk.n = ownNode(w)
		}
		kk.i = counts[kk]
		counts[kk]++
		return kk
	}
	oldBy := map[key]tree.Node{}
	counts := map[key]int{}
	for _, k := range olds {
		if kk := keyOf(k, counts); kk.n != nil {
			oldBy[kk] = k
		}
	}
	kept := map[tree.Node]bool{}
	counts = map[key]int{}
	for _, nk := range news {
		kk := keyOf(nk, counts)
		ok := oldBy[kk]
		if ok != nil && rc.reconcile(ok.(core.Widget), nk.(core.Widget)) {
			kept[ok] = true
			rc.keep(p, ok)
			continue
		}
		// a new widget, with a name that no old one has (so that it is
		// not taken for one of them)
		name := nk.AsTree().Name
		if names[name] {
			name += "~" + strconv.Itoa(rc.c.rereads)
		}
		names[name] = true
		nw := nk
		p.Add(name, func() tree.Node { return nw }, func(tree.Node) {})
		if w, ok := nk.(core.Widget); ok {
			rc.added = append(rc.added, w)
		}
	}
	for _, k := range olds {
		if !kept[k] {
			if w, ok := k.(core.Widget); ok {
				rc.removed = append(rc.removed, w)
			}
		}
	}
}

// reconcile returns whether the given old widget is kept for the given
// new one (of the same element and type): as it is, if the element and
// those in it have not changed, and the widgets are of the same shape;
// with its children reconciled, if it is the frame of an element that has
// not changed itself, whose children are not arranged as they are read.
func (rc *reconciler) reconcile(old, nw core.Widget) bool {
	n := ownNode(old)
	if !rc.touched[n] {
		if !sameShape(old, nw) {
			return false
		}
		rc.pairShapes(old, nw)
		return true
	}
	if rc.selfDirty[n] || frameOf(old) == nil || frameOf(nw) == nil || rc.c.arrangedChildren(n) {
		return false
	}
	rc.pairs[nw] = old
	p := &tree.Plan{EnforceEmpty: true}
	rc.planChildren(p, old.AsTree().Children, nw.AsTree().Children, childNames(old))
	p.Update(old)
	old.AsWidget().NeedsLayout()
	return true
}

// sameShape returns whether the given widgets are of the same type, of
// the same elements, with children of the same shapes.
func sameShape(a, b tree.Node) bool {
	if a.AsTree().NodeType() != b.AsTree().NodeType() {
		return false
	}
	aw, aok := a.(core.Widget)
	bw, bok := b.(core.Widget)
	if aok != bok || aok && ownNode(aw) != ownNode(bw) {
		return false
	}
	ak, bk := a.AsTree().Children, b.AsTree().Children
	if len(ak) != len(bk) {
		return false
	}
	for i := range ak {
		if !sameShape(ak[i], bk[i]) {
			return false
		}
	}
	return true
}

// pairShapes pairs the widgets of the given old widget with those of the
// given new one of the same shape, which are kept.
func (rc *reconciler) pairShapes(old, nw tree.Node) {
	if ow, ok := old.(core.Widget); ok {
		rc.pairs[nw.(core.Widget)] = ow
	}
	ok, nk := old.AsTree().Children, nw.AsTree().Children
	for i := range ok {
		rc.pairShapes(ok[i], nk[i])
	}
}

// updateAdded styles the new widgets that have been added.
func (rc *reconciler) updateAdded() {
	for _, w := range rc.added {
		w.AsWidget().Update()
	}
	rc.added = nil
}

// remap updates the state of the context for the old widgets that are
// kept, in place of the new ones made for them, after the given element
// is read again, and forgets that of those that are removed.
func (rc *reconciler) remap(r *html.Node) {
	c := rc.c
	newOf := func(w core.Widget) core.Widget {
		if o := rc.pairs[w]; o != nil {
			return o
		}
		return w
	}
	walkElements(r, func(n *html.Node) {
		ws := c.elementWidgets[n]
		for i, w := range ws {
			ws[i] = newOf(w)
		}
		if w := c.nodeWidgets[n]; w != nil {
			c.nodeWidgets[n] = newOf(w)
		}
		if cf := c.containers[n]; cf != nil {
			if o, ok := newOf(cf).(*containerFrame); ok {
				c.containers[n] = o
			}
		}
		if sw := c.hoverTargets[n]; sw != nil {
			if o, ok := newOf(sw).(*core.Switch); ok {
				c.hoverTargets[n] = o
			}
		}
	})
	for nw := range rc.pairs {
		delete(c.blocks, nw)
	}
	// the state of the removed widgets, and of the elements that are no
	// longer in the page
	root := rootNode(r)
	for _, w := range rc.removed {
		w.AsWidget().WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			delete(c.blocks, cw)
			if n := ownNode(cw); n != nil && rootNode(n) != root {
				delete(c.elementWidgets, n)
				delete(c.nodeWidgets, n)
				delete(c.styles, n)
				delete(c.sorted, n)
				delete(c.hinted, n)
				delete(c.containers, n)
				c.varMemo = nil
			}
			return tree.Continue
		})
	}
}

// ownNode returns the element of the given widget itself (not that of a
// widget that it is in, as [NodeOf] does), or nil.
func ownNode(w core.Widget) *html.Node {
	n, _ := w.AsWidget().Property("node").(*html.Node)
	return n
}

// widgetIn returns whether the given widget is the given one or in it.
func widgetIn(w, in core.Widget) bool {
	for t := tree.Node(w); t != nil; t = t.AsTree().Parent {
		if t == tree.Node(in) {
			return true
		}
	}
	return false
}

// walkElements calls the given function for the given element and the
// elements in it.
func walkElements(n *html.Node, f func(n *html.Node)) {
	if n.Type == html.ElementNode {
		f(n)
	}
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		walkElements(k, f)
	}
}

// outermost returns the given elements without those in others of them
// (and duplicates).
func outermost(ns []*html.Node) []*html.Node {
	var out []*html.Node
	for i, n := range ns {
		in := false
		for j, o := range ns {
			if i != j && o != n && contains(o, n) || o == n && j < i {
				in = true
				break
			}
		}
		if !in {
			out = append(out, n)
		}
	}
	return out
}
