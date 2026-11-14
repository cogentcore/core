// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"github.com/aymerick/douceur/css"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Fallback returns why the last [Context.Reread] or [Context.Restyle]
// could not make its changes, for the page to be read again whole (as for
// the verbose console of an app), or "" if it could.
func (c *Context) Fallback() string {
	return c.fallback
}

// noChange records the given reason why the changes of the page can not
// be made (see [Context.Fallback]), and returns false.
func (c *Context) noChange(why string) bool {
	c.fallback = why
	return false
}

// describeNode returns a short description of the given element, as
// div#id.class, for the reasons of [Context.Fallback].
func describeNode(n *html.Node) string {
	if n == nil {
		return "nothing"
	}
	if n.Type != html.ElementNode {
		return "a text"
	}
	d := n.Data
	if id := GetAttr(n, "id"); id != "" {
		d += "#" + id
	}
	if cl := strings.Fields(GetAttr(n, "class")); len(cl) > 0 {
		d += "." + cl[0]
	}
	return d
}

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
	if n.DataAtom == atom.Style {
		if c.sheetTexts == nil {
			c.sheetTexts = map[*html.Node]string{}
		}
		c.sheetTexts[n] = nodeText(n)
	}
}

// addedStyle adds the rules added to the given style element (or the
// rules of the given new style element), whose text is that before with
// more at its end, as CSS-in-JS libraries add them as a page is made (as
// emotion on nytimes.com, for each part of the page, each of which read
// the page again whole, which resized its elements each time). The rules
// are recorded, not applied: the declarations of the elements are found
// again with them (see [Context.updateDecls]). It returns false for
// another change of the text of a style sheet, for which the page is
// read again.
func (c *Context) addedStyle(n *html.Node) bool {
	text := nodeText(n)
	old, ok := c.sheetTexts[n]
	if !c.sheets[n] {
		old, ok = "", true // a new style element
	}
	if !ok || !strings.HasPrefix(text, old) {
		return false
	}
	c.recordSheet(n)
	added := text[len(old):]
	if strings.TrimSpace(added) == "" || !c.mediaMatches(GetAttr(n, "media")) {
		return true
	}
	base := c.cssBase
	c.cssBase = ""
	c.recordOnly = true
	c.addRules(rootNode(n), parseCSS(added), originAuthor)
	c.recordOnly = false
	c.cssBase = base
	return true
}

// headChanged handles a change of the children of the given head element:
// a new style element adds its rules (see [Context.addedStyle]), and the
// other elements of the head are not rendered, except a link to a new
// style sheet (which is gotten as the page is read again), for which it
// returns false.
func (c *Context) headChanged(head *html.Node) bool {
	for k := head.FirstChild; k != nil; k = k.NextSibling {
		if k.Type != html.ElementNode || c.sheets[k] {
			continue
		}
		switch k.DataAtom {
		case atom.Style:
			if !c.addedStyle(k) {
				return false
			}
		case atom.Link:
			if strings.EqualFold(GetAttr(k, "rel"), "stylesheet") {
				return false
			}
		}
	}
	return true
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
	apply := c.PrepareReread(self, children)
	if apply == nil {
		return false
	}
	apply()
	return true
}

// PrepareReread reads the given elements again as [Context.Reread] does,
// into new widgets, with the styles of the context locked (see
// [Context.LockStyles]), and without the lock of the widgets of the page
// (as it takes time: as long as a second, on nytimes.com, in which the app
// did not respond), and returns the function that puts the new widgets in
// the page, with the lock of the widgets held, or nil if they can not be
// read again (see [Context.Fallback]).
func (c *Context) PrepareReread(self, children []*html.Node) func() {
	c.styleMu.Lock()
	defer c.styleMu.Unlock()
	var apply func()
	if !c.prepareReread(self, children, &apply) {
		return nil
	}
	if apply == nil {
		apply = func() {}
	}
	return apply
}

// prepareReread is [Context.PrepareReread], setting the function that
// applies it, if any, and returning whether it can.
func (c *Context) prepareReread(self, children []*html.Node, apply *func()) bool {
	c.containersChanged() // (declarations may change)
	defer c.containersChanged()
	if len(self) == 0 && len(children) == 0 {
		return true
	}
	c.fallback = ""
	if c.reading || c.rest != nil || c.elementWidgets == nil {
		return c.noChange("the page is being read")
	}
	// the changed elements (that of a text node is its parent, whose
	// children have changed)
	var root *html.Node
	selfDirty, childDirty := map[*html.Node]bool{}, map[*html.Node]bool{}
	var changed []*html.Node // for the declarations of the elements after them
	var roots []*html.Node
	restyleAll := false         // for rules added to style sheets (see addedStyle)
	var htmlChildren *html.Node // the html element, if its children changed
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
			return c.noChange("a change of the document itself")
		}
		if !own && n.DataAtom == atom.Style && c.sheets[n] {
			if !c.addedStyle(n) {
				return c.noChange("a change of the style sheet of " + describeNode(n) + " that is not an addition to it")
			}
			if root == nil {
				root = rootNode(n)
			}
			restyleAll = true
			return true
		}
		if !own && n.DataAtom == atom.Head {
			if !c.headChanged(n) {
				return c.noChange("a new style sheet linked in the head")
			}
			if root == nil {
				root = rootNode(n)
			}
			restyleAll = true
			return true
		}
		if root == nil {
			root = rootNode(n)
		} else if rootNode(n) != root {
			return c.noChange("changes of different documents")
		}
		// (an element that has changed itself may have another box, as
		// for its display, in that of its parent)
		from := n
		if own {
			from = n.Parent
		}
		if from != nil && from.Type == html.ElementNode && !inHead(from) && c.undisplayed(from) && !hasNewSheet(c, from) {
			// in an element that is not displayed, which has no widgets
			// (as the video players of npr.org in its hidden dialogs,
			// whose changes as they played read the page whole, as they
			// were in main): its declarations are found again, and it is
			// read when it is displayed (see Context.updateDecls)
			if own {
				changed = append(changed, n)
			}
			return true
		}
		r := c.rereadRoot(from)
		if r == nil && own && (n.DataAtom == atom.Html || n.DataAtom == atom.Body) {
			// a change of the attributes of the html or body element (as
			// the classes that scripts add to them, for the page as it
			// loads, as on npr.org): it is styled again, with the elements
			// that the change may select differently (see
			// Context.invalidation), unless it needs to be read again (as
			// for its display; see Context.updateDecls)
			changed = append(changed, n)
			return true
		}
		if r == nil && !own && n.DataAtom == atom.Html {
			// a change of the children of the html element, as an element
			// added after the body (as scripts measure things with, as on
			// npr.org): its children are read again, but the body, which
			// is kept (see Context.rereadHTML)
			htmlChildren = n
			return true
		}
		if r == nil {
			return c.noChange("no element of " + describeNode(from) + " can be read again alone")
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
	if htmlChildren != nil && !c.htmlRereadable(htmlChildren, roots, selfDirty) {
		return c.noChange("a change of the children of the html element")
	}
	if len(roots) == 0 && !restyleAll && len(changed) == 0 && htmlChildren == nil {
		return true // (only elements that are not rendered changed)
	}
	targeted := false
	if restyleAll {
		html := documentElement(root)
		if shallow, deep, ok := c.addedRuleTargets(html); ok {
			// only the elements that the added rules select (as for those
			// that the CSS-in-JS of nytimes.com adds as it loads, each
			// time matching its 10000 elements again, for 1s)
			c.addedShallow, c.addedDeep = shallow, deep
			targeted, restyleAll = true, false
			defer func() { c.addedShallow, c.addedDeep = nil, nil }()
		} else if html != nil {
			// the declarations of all of the elements are found again,
			// with the rules added to the style sheets, as for a restyle
			changed = append(changed, html)
		}
	}
	// the elements to read are not too many (not counting those that are
	// kept without being read; see Context.keptElements), have no new
	// style sheets, and their widgets are all in their frames, which have
	// only theirs
	size, limit := 0, c.rereadLimit(root)
	prelim := c.keptElements(roots, touchedBy(selfDirty, childDirty), selfDirty)
	for _, r := range roots {
		ok := true
		walkElements(r, func(n *html.Node) {
			if !inKept(prelim, n) {
				size++
			}
			if (n.Data == "style" || n.Data == "link") && !c.sheets[n] {
				ok = false
			}
			for _, w := range c.elementWidgets[n] {
				if !widgetIn(w, c.elementWidgets[r][0]) {
					ok = false // (as one moved elsewhere)
				}
			}
		})
		if !ok {
			return c.noChange("a new style sheet, or an element moved, in " + describeNode(r))
		}
		if size > limit {
			return c.noChange(fmt.Sprintf("%d elements to read again (more than %d), in %s", size, limit, describeNode(r)))
		}
		c.elementWidgets[r][0].AsWidget().WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if n := ownNode(cw); n != nil && rootNode(n) == root && !inRoots(n) {
				ok = false
			}
			return ok
		})
		if !ok {
			return c.noChange("widgets of other elements in those of " + describeNode(r))
		}
	}
	c.matchKept = c.shallowKept(prelim)
	c.matchAll = restyleAll
	u, ok := c.updateDecls(changed, roots, inRoots)
	c.matchKept, c.matchAll = nil, false
	if !ok {
		return false // (with the fallback of updateDecls)
	}
	if targeted {
		// (the other elements do not match the added rules)
		c.matchedRules, c.matchedPseudo = len(c.rules), len(c.pseudoRules)
	}
	for n := range u.freeChanged {
		selfDirty[n] = true
	}
	touched := touchedBy(selfDirty, childDirty)
	keep := c.keptElements(roots, touched, selfDirty)

	c.reading = true
	defer func() { c.reading = false }()
	stage := c.StageSize
	c.StageSize = 0
	defer func() { c.StageSize = stage }()
	c.rereads++
	c.keepRead = keep
	defer func() { c.keepRead = nil }()
	// the elements are read into new widgets (here, without the lock of
	// the widgets of the page), which are put in the page by apply
	type read struct {
		r       *html.Node
		old, pw core.Widget
		tmp     *core.Frame
	}
	var reads []read
	for _, r := range roots {
		old := c.elementWidgets[r][0]
		pw := old.AsTree().Parent.(core.Widget)
		c.forgetRead(r, keep)
		tmp := core.NewFrame()
		c.inlineParent = nil
		readOneNode(c, tmp, r)
		c.lockStylesOf(tmp)
		reads = append(reads, read{r, old, pw, tmp})
	}
	var applyHTML func()
	if htmlChildren != nil {
		applyHTML = c.rereadHTML(htmlChildren, touched, selfDirty)
	}
	*apply = func() {
		for _, rd := range reads {
			rc := &reconciler{c: c, touched: touched, selfDirty: selfDirty, kept: keep, pairs: map[core.Widget]core.Widget{}}
			rc.replaceIn(rd.pw, rd.old, rd.tmp.Children)
			rc.placeKept()
			rc.remap(rd.r)
		}
		if applyHTML != nil {
			applyHTML()
		}
		u.apply(c)
	}
	return true
}

// htmlRereadable returns whether the children of the given html element
// can be read again with its body kept as it is (see [Context.rereadHTML]):
// its body has a frame of its own in the widget that the page is read
// into, and nothing in it changes.
func (c *Context) htmlRereadable(h *html.Node, roots []*html.Node, selfDirty map[*html.Node]bool) bool {
	var body *html.Node
	for k := h.FirstChild; k != nil; k = k.NextSibling {
		if k.DataAtom == atom.Body {
			body = k
		}
	}
	if body == nil || !c.rereadable(body) || selfDirty[body] {
		return false
	}
	for _, r := range roots {
		if contains(body, r) {
			return false
		}
	}
	return true
}

// rereadHTML reads the children of the given html element again, but its
// body, whose widgets are kept as they are (see [Context.keptElements]),
// in the widget that the page is read into, returning the function that
// puts them in it (with the lock of the widgets held).
func (c *Context) rereadHTML(h *html.Node, touched, selfDirty map[*html.Node]bool) func() {
	var body *html.Node
	for k := h.FirstChild; k != nil; k = k.NextSibling {
		if k.DataAtom == atom.Body {
			body = k
		}
	}
	bw := c.elementWidgets[body][0]
	content := bw.AsTree().Parent.(core.Widget)
	keep := map[*html.Node]core.Widget{body: bw}
	for k := h.FirstChild; k != nil; k = k.NextSibling {
		if k.Type == html.ElementNode && k != body && k.DataAtom != atom.Head {
			c.forgetRead(k, keep)
		}
	}
	c.keepRead = keep
	defer func() { c.keepRead = nil }()
	tmp := core.NewFrame()
	c.inlineParent = nil
	c.Node = h
	readHTMLNode(c, tmp, h.FirstChild)
	c.lockStylesOf(tmp)
	return func() {
		rc := &reconciler{c: c, touched: touched, selfDirty: selfDirty, kept: keep, pairs: map[core.Widget]core.Widget{}}
		p := &tree.Plan{EnforceEmpty: true}
		rc.planChildren(p, content.AsTree().Children, tmp.Children, childNames(content))
		p.Update(content)
		rc.updateAdded()
		rc.placeKept()
		rc.remap(h)
		content.AsWidget().NeedsLayout()
	}
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
func (c *Context) forgetRead(r *html.Node, keep map[*html.Node]core.Widget) {
	walkElementsSkipping(r, keep, func(n *html.Node) {
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

	// kept has the old widgets of the elements that are kept without
	// being read again (see [Context.keptElements]), which are in the new
	// widgets as placeholders (see [keptPlaceholder]), and placed has
	// those that have been put in their places.
	kept   map[*html.Node]core.Widget
	placed map[*html.Node]bool

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
		if kn := keptPlaceholderNode(nk); kn != nil {
			// an element kept as it is, without being read again
			if old := rc.kept[kn]; old != nil && slices.Contains(olds, tree.Node(old)) {
				kept[old] = true
				rc.keep(p, old)
				rc.markPlaced(kn)
				continue
			}
		}
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
				if hasCustomDecls(c.styles[n]) {
					c.varMemo = nil
				}
				delete(c.elementWidgets, n)
				delete(c.nodeWidgets, n)
				delete(c.styles, n)
				delete(c.sorted, n)
				delete(c.hinted, n)
				delete(c.containers, n)
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

// documentElement returns the root element (html) of the given document.
func documentElement(doc *html.Node) *html.Node {
	if doc == nil {
		return nil
	}
	for k := doc.FirstChild; k != nil; k = k.NextSibling {
		if k.Type == html.ElementNode {
			return k
		}
	}
	return nil
}

// hasNewSheet returns whether the given element has a style sheet in it
// (or is one) that has not been added to the page.
func hasNewSheet(c *Context, n *html.Node) bool {
	found := false
	walkElements(n, func(k *html.Node) {
		if (k.Data == "style" || k.Data == "link") && !c.sheets[k] {
			found = true
		}
	})
	return found
}

// touchedBy returns the elements that have changed (in the given sets),
// and those that they are in.
func touchedBy(sets ...map[*html.Node]bool) map[*html.Node]bool {
	touched := map[*html.Node]bool{}
	for _, set := range sets {
		for n := range set {
			for p := n; p != nil && !touched[p]; p = p.Parent {
				touched[p] = true
			}
		}
	}
	return touched
}

// keptElements returns the elements in the given elements to read again
// (see [Context.Reread]) that are kept as they are, with their widgets,
// without being read again: as browsers lay out again only the boxes
// that have changed, or have changed boxes in them. They are those that
// have not changed, nor any element in them, in an element whose children
// have changed (only), whose frame is kept (see [reconciler.reconcile]),
// as the old widgets of those elements are kept anyway: a script that
// added an element to the body of npr.org read all of its 5000 elements
// again, many times as it loaded. The widgets of each must be its frame
// alone in that of its parent (see [Context.rereadable]), and not an item
// of a list (whose markers are numbered as they are read).
func (c *Context) keptElements(roots []*html.Node, touched, selfDirty map[*html.Node]bool) map[*html.Node]core.Widget {
	keep := map[*html.Node]core.Widget{}
	var find func(e *html.Node)
	find = func(e *html.Node) {
		if !touched[e] || selfDirty[e] || len(c.elementWidgets[e]) != 1 || frameOf(c.elementWidgets[e][0]) == nil || c.arrangedChildren(e) {
			return
		}
		switch e.DataAtom {
		case atom.Ol, atom.Ul, atom.Menu, atom.Dl:
			return
		}
		var kids []*html.Node
		for k := e.FirstChild; k != nil; k = k.NextSibling {
			if k.Type == html.ElementNode {
				kids = append(kids, k)
			}
		}
		// new elements (not read yet), whose neighbors are read again
		isNew := func(i int) bool { return i >= 0 && i < len(kids) && len(c.elementWidgets[kids[i]]) == 0 }
		for i, k := range kids {
			if touched[k] {
				find(k)
				continue
			}
			// the first and last elements, and those next to new ones,
			// are read again if they have vertical margins, which may
			// collapse with those of their parent or their new siblings
			top, bottom := c.verticalMargins(k)
			if (i == 0 || isNew(i-1)) && top || (i == len(kids)-1 || isNew(i+1)) && bottom {
				if len(c.elementWidgets[k]) > 0 {
					// (with new widgets, not the old ones that the
					// reconciler would keep for them otherwise)
					touched[k], selfDirty[k] = true, true
				}
				continue
			}
			if k.DataAtom != atom.Li && c.rereadable(k) {
				keep[k] = c.elementWidgets[k][0]
			}
		}
	}
	for _, r := range roots {
		find(r)
	}
	return keep
}

// inKept returns whether the given element is one of the given kept ones
// or in one.
func inKept(keep map[*html.Node]core.Widget, n *html.Node) bool {
	for p := n; p != nil; p = p.Parent {
		if keep[p] != nil {
			return true
		}
	}
	return false
}

// walkElementsSkipping calls the given function for the given element and
// the elements in it, except the given ones and those in them.
func walkElementsSkipping(n *html.Node, skip map[*html.Node]core.Widget, f func(n *html.Node)) {
	if skip[n] != nil {
		return
	}
	if n.Type == html.ElementNode {
		f(n)
	}
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		walkElementsSkipping(k, skip, f)
	}
}

// keptPlaceholder adds a placeholder to the given parent for the given
// element, which is kept as it is (see [Context.keptElements]): its old
// widget takes its place (see [reconciler.planChildren]).
func keptPlaceholder(parent core.Widget, n *html.Node) {
	ph := core.NewFrame(parent)
	ph.SetProperty("keptNode", n)
}

// keptPlaceholderNode returns the element of the given widget if it is a
// placeholder (see [keptPlaceholder]), or nil.
func keptPlaceholderNode(k tree.Node) *html.Node {
	n, _ := k.AsTree().Property("keptNode").(*html.Node)
	return n
}

// markPlaced records that the old widget of the given kept element has
// been put in its place.
func (rc *reconciler) markPlaced(n *html.Node) {
	if rc.placed == nil {
		rc.placed = map[*html.Node]bool{}
	}
	rc.placed[n] = true
}

// placeKept puts the old widgets of the kept elements that are not in
// their places yet (in new widgets, which are not reconciled with old
// ones) in the places of their placeholders.
func (rc *reconciler) placeKept() {
	if len(rc.placed) == len(rc.kept) {
		return
	}
	for n, old := range rc.kept {
		if rc.placed[n] {
			continue
		}
		// the placeholder, in the new widgets of the element that n is in
		var ph core.Widget
		for p := n.Parent; p != nil && ph == nil; p = p.Parent {
			for _, w := range rc.c.elementWidgets[p] {
				w.AsWidget().WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
					if keptPlaceholderNode(cw) == n {
						ph = cw
						return tree.Break
					}
					return tree.Continue
				})
				if ph != nil {
					break
				}
			}
		}
		if ph == nil {
			continue // not read, as in an element that is not displayed
		}
		np := ph.AsTree().Parent
		i := tree.IndexOf(np.AsTree().Children, tree.Node(ph))
		if op := old.AsTree().Parent; op != nil {
			if j := tree.IndexOf(op.AsTree().Children, tree.Node(old)); j >= 0 {
				op.AsTree().Children = slices.Delete(op.AsTree().Children, j, j+1)
			}
		}
		np.AsTree().Children[i] = old
		tree.SetParent(old, np)
		rc.markPlaced(n)
	}
}

// shallowKept returns those of the given elements that may be kept as they
// are (see [Context.keptElements]) whose declarations are found again only
// for themselves (see [Context.updateDecls]), not for those in them, which
// match the style rules as they did: those whose place in their parents
// has not changed in a way that the selectors of the rules select the
// elements in them by (as .item:first-child a, or .tab + .panel p; see
// [Context.placeUses]), as browsers match only the elements that the
// changes may affect again (a script that added an element to the body of
// npr.org matched all of its 8000 elements again, for 1s).
func (c *Context) shallowKept(keep map[*html.Node]core.Widget) map[*html.Node]bool {
	if len(keep) == 0 || c.snaps == nil {
		return nil
	}
	_, deep := c.placeUses()
	_, inSiblings := c.siblingCombinators()
	parents := map[*html.Node]bool{}
	for k := range keep {
		parents[k.Parent] = true
	}
	shallow := map[*html.Node]bool{}
	for p := range parents {
		newBefore := false // an element before it that is new (inserted)
		snapChildren(p, func(k *html.Node, s nodeSnap) {
			o, ok := c.snaps[k]
			if keep[k] != nil && ok && !deep.differ(o, s) && !(inSiblings && (newBefore || o.index != s.index)) {
				shallow[k] = true
			}
			newBefore = newBefore || !ok
		})
	}
	return shallow
}

// verticalMargins returns whether the given element has a top and a bottom
// margin that is not zero (which may collapse with others).
func (c *Context) verticalMargins(n *html.Node) (top, bottom bool) {
	nonZero := func(v string) bool {
		v = strings.TrimSpace(v)
		if v == "" || v == "auto" {
			return false
		}
		f, err := strconv.ParseFloat(strings.TrimRight(v, "abcdefghijklmnopqrstuvwxyz%"), 64)
		return err != nil || f != 0
	}
	if m := strings.Fields(c.declaration(n, "margin")); len(m) > 0 {
		top = nonZero(m[0])
		bottom = len(m) >= 3 && nonZero(m[2]) || len(m) < 3 && nonZero(m[0])
	}
	if m := strings.Fields(c.declaration(n, "margin-block")); len(m) > 0 {
		top = top || nonZero(m[0])
		bottom = bottom || nonZero(m[len(m)-1])
	}
	top = top || nonZero(c.declaration(n, "margin-top")) || nonZero(c.declaration(n, "margin-block-start"))
	bottom = bottom || nonZero(c.declaration(n, "margin-bottom")) || nonZero(c.declaration(n, "margin-block-end"))
	return
}

// addedRuleTargets returns the elements in the given root element that the
// rules added to the style sheets since the elements were matched select
// (see [Context.addedStyle]): those whose declarations of the rules have
// custom properties are deep (the elements in them inherit them), and the
// others are shallow, as browsers find the elements that new style sheets
// may select by the keys of their selectors (or, for those without one, as
// *, by matching them alone). It returns ok false, for all of the elements
// to be matched again, for a rule with :has().
func (c *Context) addedRuleTargets(root *html.Node) (shallow, deep []*html.Node, ok bool) {
	if root == nil || c.matchedRules > len(c.rules) || c.matchedPseudo > len(c.pseudoRules) {
		return nil, nil, false
	}
	type added struct {
		cs   *compiledSelector
		vars bool
	}
	byKey := map[string][]added{} // by "#id", ".class", or "tag"
	var keyless []added           // matched against all of the elements
	addRule := func(cs *compiledSelector, text string, decls []*css.Declaration) bool {
		if cs == nil || strings.Contains(text, ":has(") {
			return false
		}
		if cs.keyKind == 0 {
			keyless = append(keyless, added{cs, hasCustomProps(decls)})
			return true
		}
		key := cs.key
		if cs.keyKind != 't' {
			key = string(cs.keyKind) + key
		}
		byKey[key] = append(byKey[key], added{cs, hasCustomProps(decls)})
		return true
	}
	for _, r := range c.rules[c.matchedRules:] {
		if !addRule(r.sel, r.text, r.decls) {
			return nil, nil, false
		}
	}
	for _, r := range c.pseudoRules[c.matchedPseudo:] {
		if !addRule(r.sel, r.text, nil) {
			return nil, nil, false
		}
	}
	if len(byKey) == 0 && len(keyless) == 0 {
		return nil, nil, true
	}
	defer c.keepAncestorMasks()()
	walkElements(root, func(n *html.Node) {
		match, vars := false, false
		check := func(key string) {
			for _, a := range byKey[key] {
				if c.selMatch(a.cs, n) {
					match = true
					vars = vars || a.vars
				}
			}
		}
		for _, a := range keyless {
			if c.selMatch(a.cs, n) {
				match = true
				vars = vars || a.vars
			}
		}
		check(n.Data)
		if id := GetAttr(n, "id"); id != "" {
			check("#" + id)
		}
		for _, cl := range strings.Fields(GetAttr(n, "class")) {
			check("." + cl)
		}
		switch {
		case vars:
			deep = append(deep, n)
		case match:
			shallow = append(shallow, n)
		}
	})
	return shallow, deep, true
}
