// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"io"
	"slices"
	"sort"
	"strings"

	"cogentcore.org/core/htmlcore/css"

	"cogentcore.org/core/base/errors"
	"github.com/andybalholm/cascadia"
	douceur "github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"
	"golang.org/x/net/html"
)

// cssDecl is a CSS declaration that applies to an element,
// with what orders it in the cascade.
type cssDecl struct {
	Property  string
	Value     string
	Important bool

	// invalid is whether the value is invalid, as with a var()
	// of a custom property in a cycle.
	invalid bool

	origin      css.Origin
	specificity int
	order       int

	// container is the condition of the @container rule that the
	// declaration is in, if any (see [Context.containerMatches]).
	container *containerCond

	// rule is the index of the style rule of the declaration in the
	// rules of the context plus one, or 0 if it is not of a rule (see
	// [Context.Restyle]).
	rule int
}

// rank returns the rank of the declaration in the cascade,
// before the specificity and order: important declarations
// come after the others, with those of the user agent last.
func (d *cssDecl) rank() int {
	if !d.Important {
		return int(d.origin)
	}
	if d.origin == css.OriginUserAgent {
		return 10
	}
	return 5 + int(d.origin)
}

// addRules adds the rules of the given style sheet with the given
// origin to the declarations of the elements under root that they select.
func (c *Context) addRules(root *html.Node, ss *douceur.Stylesheet, origin css.Origin) {
	if c.styles == nil {
		c.styles = map[*html.Node][]*cssDecl{}
	}
	c.addRuleList(root, ss.Rules, origin)
}

// addRuleList adds the given rules, including those in @media rules
// that match and in @supports rules.
func (c *Context) addRuleList(root *html.Node, rules []*douceur.Rule, origin css.Origin) {
	for _, rule := range rules {
		if rule.Kind == douceur.AtRule {
			switch strings.ToLower(rule.Name) {
			case "@media":
				if c.mediaMatches(rule.Prelude) {
					c.addRuleList(root, rule.Rules, origin)
				}
			case "@container":
				prev := c.curContainer
				c.curContainer = parseContainerCond(rule.Prelude, prev)
				c.addRuleList(root, rule.Rules, origin)
				c.curContainer = prev
			case "@supports", "@layer", "@scope":
				if cond, ok := strings.CutPrefix(strings.TrimSpace(rule.Prelude), containerMark); ok {
					prev := c.curContainer
					c.curContainer = parseContainerCond(cond, prev)
					c.addRuleList(root, rule.Rules, origin)
					c.curContainer = prev
					continue
				}
				if strings.ToLower(rule.Name) == "@supports" && !supportsMatches(rule.Prelude) {
					continue
				}
				c.addRuleList(root, rule.Rules, origin)
			case "@font-face":
				if c.WebFonts {
					c.addFontFace(rule.Declarations)
				}
			case "@import":
				// url("x") or "x", and a media query list
				url, media := css.ImportURL(rule.Prelude)
				if url != "" && c.mediaMatches(media) {
					c.addLinkedStyle(url)
				}
			}
			continue
		}
		// the parser splits the selectors at commas in parentheses,
		// as in :is(a, b), so they are split from the prelude
		selectors := rule.Selectors
		if strings.Contains(rule.Prelude, "(") {
			selectors = css.SplitSelectors(rule.Prelude)
		}
		for _, sel := range selectors {
			sel = strings.TrimSpace(sel)
			kind := css.ScopeAll
			if c.curScope != nil {
				var ok bool
				if sel, kind, ok = css.ScopedSelector(sel); !ok {
					continue
				}
			} else if strings.Contains(sel, ":host") || strings.Contains(sel, "::slotted") {
				continue // (of shadow trees only)
			} else if origin == css.OriginUserAgent {
				kind = css.ScopeAny // (the styles of the user agent apply to all)
			}
			c.curScopeKind = kind
			if strings.Contains(sel, ":hover") {
				c.addHoverTrigger(sel)
			}
			if base, kind, ok := css.PseudoSelector(sel); ok {
				c.addPseudoRule(root, base, kind, rule.Declarations, origin)
				continue
			}
			spec := css.Specificity(sel)
			cs := css.CompileSelector(sel)
			if cs != nil {
				spec = cs.Specificity
				c.recordRule(root, sel, cs, rule.Declarations, origin, spec)
			}
			if !c.recordOnly && !c.deferMatch { // (see Context.addedStyle and Context.ReuseStyles)
				if kind == css.ScopeHost {
					if cs != nil && c.selMatch(cs, c.curScope) {
						c.addDecls(c.curScope, rule.Declarations, origin, spec)
					}
				} else {
					for _, n := range c.selectNodes(root, sel) {
						if c.inScope(n, c.curScope, kind) {
							c.addDecls(n, rule.Declarations, origin, spec)
						}
					}
				}
			}
			c.curScopeKind = css.ScopeAll
			c.curRule = 0
		}
	}
}

// addDecls adds the given declarations to the given node.
func (c *Context) addDecls(n *html.Node, decls []*douceur.Declaration, origin css.Origin, spec int) {
	custom := false
	for _, d := range decls {
		prop, value := css.PropertyName(d.Property), strings.TrimSpace(d.Value)
		if !css.ValidDeclaration(prop, value) {
			continue // dropped, so that the declaration before it applies
		}
		c.cssOrder++
		c.noteVars(value)
		custom = custom || strings.HasPrefix(prop, "--")
		c.styles[n] = append(c.styles[n], &cssDecl{Property: prop,
			Value: c.absoluteURLs(value), Important: d.Important, origin: origin, specificity: spec, order: c.cssOrder,
			container: c.curContainer, rule: c.curRule})
	}
	if custom {
		// (the memo has the values of custom properties only, which the
		// hints and inline styles of each element, as they were added,
		// otherwise cleared for each element, so that each var() looked
		// up the elements that it is in again)
		c.varMemo = nil
	}
	if c.curContainer != nil {
		if c.containerQueried == nil {
			c.containerQueried = map[*html.Node]bool{}
		}
		c.containerQueried[n] = true
	}
	delete(c.sorted, n)
}

// addHint adds a declaration of a presentational attribute to the node.
func (c *Context) addHint(n *html.Node, prop, value string) {
	c.addDecls(n, []*douceur.Declaration{{Property: prop, Value: value}}, css.OriginHint, 0)
}

// declarations returns the declarations that apply to the given node, in
// the order of the cascade, so that the last one of each property wins.
func (c *Context) declarations(n *html.Node) []*cssDecl {
	if n == nil {
		return nil
	}
	if c.sorted == nil {
		c.sorted = map[*html.Node]bool{}
		c.hinted = map[*html.Node]bool{}
	}
	if !c.hinted[n] {
		c.hinted[n] = true
		if n.Type == html.ElementNode {
			c.addHints(n)
			if st := GetAttr(n, "style"); st != "" {
				c.addInline(n, st)
			}
		}
	}
	if !c.sorted[n] {
		c.sorted[n] = true
		ds := c.styles[n]
		defer c.resolveVars(n, ds)
		sort.SliceStable(ds, func(i, j int) bool {
			a, b := ds[i], ds[j]
			if a.rank() != b.rank() {
				return a.rank() < b.rank()
			}
			if a.specificity != b.specificity {
				return a.specificity < b.specificity
			}
			return a.order < b.order
		})
	}
	if c.containerQueried[n] {
		return c.containerDecls(n, c.styles[n])
	}
	return c.styles[n]
}

// allDeclarations returns the declarations of the given node as
// [Context.declarations] does, but with those of all of its @container
// rules, whatever their conditions.
func (c *Context) allDeclarations(n *html.Node) []*cssDecl {
	if n == nil {
		return nil
	}
	if !c.containerQueried[n] {
		return c.declarations(n)
	}
	c.declarations(n) // (sorted, with hints)
	return c.styles[n]
}

// declaration returns the value of the given property for the given node
// from its declarations (not inherited), or "" if it has none.
func (c *Context) declaration(n *html.Node, prop string) string {
	return declarationIn(c.declarations(n), prop)
}

// declarationIn returns the value of the given property in the given
// declarations, as [Context.declaration].
func declarationIn(ds []*cssDecl, prop string) string {
	for ri := len(ds) - 1; ri >= 0; ri-- {
		d := ds[ri]
		if d.Property == prop {
			if d.invalid {
				return ""
			}
			if prop == "display" {
				return css.LegacyDisplay(d.Value)
			}
			return d.Value
		}
	}
	return ""
}

// lastDeclaration returns the property and value of the declaration of
// the given element that applies of those of the given properties (as a
// shorthand and its longhand): the last of them in the cascade, or "" if
// there is none (or it is invalid).
func (c *Context) lastDeclaration(n *html.Node, props ...string) (prop, value string) {
	ds := c.declarations(n)
	for ri := len(ds) - 1; ri >= 0; ri-- {
		d := ds[ri]
		if slices.Contains(props, d.Property) {
			if d.invalid {
				return d.Property, ""
			}
			return d.Property, d.Value
		}
	}
	return "", ""
}

// ComputedValue returns the value of the given CSS property for the given
// element, as cascaded by the style sheets of the page that it was read
// with (with var() resolved): that of the declaration of the element that
// applies, or, for an inherited property (or one that is inherit), that
// of its nearest ancestor with one, or "" if there is none (as for the
// initial value). It is for the computed styles of scripts, which can
// convert its values (as colors and relative lengths) as they need.
func (c *Context) ComputedValue(n *html.Node, prop string) string {
	prop = strings.TrimSpace(prop)
	if strings.HasPrefix(prop, "--") { // custom properties are inherited
		v, _ := c.customProperty(n, prop)
		return strings.TrimSpace(v)
	}
	prop = strings.ToLower(prop)
	for ; n != nil; n = n.Parent {
		if n.Type != html.ElementNode {
			continue
		}
		v := strings.TrimSpace(c.declaration(n, prop))
		switch strings.ToLower(v) {
		case "inherit":
			continue
		case "initial", "revert", "revert-layer":
			return ""
		case "unset":
			if !inheritedProperties[prop] {
				return ""
			}
			continue
		case "":
			if !inheritedProperties[prop] {
				return ""
			}
			continue
		}
		return v
	}
	return ""
}

// addInline adds the declarations of the given style attribute.
func (c *Context) addInline(n *html.Node, style string) {
	// our CSS parser is strict about semicolons, but
	// they aren't needed in normal inline styles in HTML
	if !strings.HasSuffix(strings.TrimSpace(style), ";") {
		style += ";"
	}
	decls, err := parser.ParseDeclarations(style)
	if err != nil {
		return
	}
	c.addDecls(n, decls, css.OriginInline, 0)
}

// SetStyleAttr adds the declarations of the given style attribute
// string to the styles of the current node.
func (c *Context) SetStyleAttr(node *html.Node, style string) error {
	if !strings.HasSuffix(strings.TrimSpace(style), ";") {
		style += ";"
	}
	decls, err := parser.ParseDeclarations(style)
	if err != nil {
		return err
	}
	if c.styles == nil {
		c.styles = map[*html.Node][]*cssDecl{}
	}
	c.addDecls(c.Node, decls, css.OriginInline, 0)
	return nil
}

// selectNodes returns the nodes under root that the given selector selects,
// using the index of the nodes of root, which it makes for the first root
// (the nodes of a document do not change while it is read).
func (c *Context) selectNodes(root *html.Node, sel string) []*html.Node {
	cs := css.CompileSelector(sel)
	if cs == nil || cs.KeyKind == 0 {
		return css.SelectNodes(root, sel)
	}
	if c.selIndex == nil || c.selIndex.Root != root {
		c.selIndex = css.NewSelectorIndex(root)
	}
	var cands []*html.Node
	switch cs.KeyKind {
	case '#':
		cands = c.selIndex.ByID[cs.Key]
	case '.':
		cands = c.selIndex.ByClass[cs.Key]
	case 't':
		cands = c.selIndex.ByTag[cs.Key]
	}
	var matches []*html.Node
	for _, n := range cands {
		if c.selMatch(cs, n) {
			matches = append(matches, n)
		}
	}
	return matches
}

// resolveVars substitutes the values of the custom properties (variables)
// in the var() functions of the given declarations of the given node.
// A custom property whose value is in a cycle is invalid, as is a property
// with a var() of an invalid custom property and no fallback.
func (c *Context) resolveVars(n *html.Node, ds []*cssDecl) {
	for _, d := range ds {
		if !strings.Contains(d.Value, "var(") {
			continue
		}
		var visiting map[string]bool
		if strings.HasPrefix(d.Property, "--") {
			visiting = map[string]bool{d.Property: true}
		}
		v, ok := c.substituteVars(n, d.Value, visiting)
		if ok && d.Property == "font" && !css.FontShorthandValid(v) {
			// a value of var() that is not a valid font, as a style and a
			// family without a size, is invalid at computed-value time,
			// which unsets the font (as the rules of reddit.com, whose
			// headings had the font of a theme that had no size, which
			// browsers draw in the font of the text around them)
			ok = false
		}
		d.Value = v
		d.invalid = !ok
	}
}

// substituteVars returns the given value with its var() functions replaced
// by the values of the custom properties for the given node, which are
// inherited, or their fallback values, and whether it is valid. The names
// being substituted are in visiting: a reference to one is a cycle, which
// makes the value invalid.
func (c *Context) substituteVars(n *html.Node, value string, visiting map[string]bool) (string, bool) {
	var sb strings.Builder
	for {
		i := strings.Index(value, "var(")
		if i < 0 {
			sb.WriteString(value)
			return sb.String(), true
		}
		// the matching parenthesis
		end, level := -1, 0
		for j := i + 3; j < len(value); j++ {
			if value[j] == '(' {
				level++
			} else if value[j] == ')' {
				level--
				if level == 0 {
					end = j
					break
				}
			}
		}
		if end < 0 {
			return "", false
		}
		name, fallback, hasFallback := strings.Cut(value[i+4:end], ",")
		name = strings.ToLower(strings.TrimSpace(name)) // property names are lowercase
		if visiting[name] {
			return "", false // a cycle
		}
		v, ok := c.customProperty(n, name)
		if ok && strings.Contains(v, "var(") {
			if visiting == nil {
				visiting = map[string]bool{}
			}
			visiting[name] = true
			v, ok = c.substituteVars(n, v, visiting)
			delete(visiting, name)
		}
		if !ok {
			if !hasFallback {
				return "", false
			}
			v, ok = c.substituteVars(n, strings.TrimSpace(fallback), visiting)
			if !ok {
				return "", false
			}
		}
		sb.WriteString(value[:i])
		sb.WriteString(v)
		value = value[end+1:]
	}
}

// customProperty returns the value of the given custom property for the
// given node, from it or the nearest ancestor that declares it.
func (c *Context) customProperty(n *html.Node, name string) (string, bool) {
	// the values of the elements on the way to the one that has it are
	// memoized (see varMemo), as they are looked up for each var() of
	// each element, through all of the elements that it is in
	var path []*html.Node
	found := func(v varValue) (string, bool) {
		if strings.Contains(v.value, "var(") {
			return v.value, v.ok // (not substituted yet, so not memoized)
		}
		if c.varMemo == nil {
			c.varMemo = map[varKey]varValue{}
		}
		for _, p := range path {
			c.varMemo[varKey{p, name}] = v
		}
		return v.value, v.ok
	}
	for p := n; p != nil; p = p.Parent {
		if p.Type != html.ElementNode {
			continue
		}
		if v, ok := c.varMemo[varKey{p, name}]; ok {
			return found(v)
		}
		ds := c.styles[p]
		if !c.hinted[p] || !c.sorted[p] {
			ds = c.declarations(p)
		}
		path = append(path, p)
		if d := c.customDecl(p, ds, name); d != nil {
			if strings.EqualFold(strings.TrimSpace(d.Value), "initial") {
				return found(varValue{}) // the guaranteed-invalid value, as when not set
			}
			return found(varValue{d.Value, !d.invalid})
		}
	}
	return found(varValue{})
}

// varKey is the key of [Context.varMemo]: an element and the name of a
// custom property.
type varKey struct {
	n    *html.Node
	name string
}

// varValue is the value of a custom property of an element (see
// [Context.customProperty]).
type varValue struct {
	value string
	ok    bool
}

// customIndex is an index of the custom properties of the sorted
// declarations of an element (see [Context.customDecl]).
type customIndex struct {
	ds []*cssDecl          // the declarations that it is of
	m  map[string]*cssDecl // the last declaration of each custom property
}

// minCustomIndex is the number of declarations of an element from which
// its custom properties are indexed.
const minCustomIndex = 32

// customDecl returns the last of the given sorted declarations of the
// given element of the given property (a custom property), or nil. The
// custom properties of elements with many declarations are indexed, as
// they are looked up for each var() of the elements in them (as those of
// the root of nytimes.com, hundreds of them, which took seconds to scan
// in each read of its page). The index is made again when the
// declarations are others (or more).
func (c *Context) customDecl(n *html.Node, ds []*cssDecl, name string) *cssDecl {
	if len(ds) < minCustomIndex {
		for ri := len(ds) - 1; ri >= 0; ri-- {
			d := ds[ri]
			if d.Property == name {
				return d
			}
		}
		return nil
	}
	idx := c.customIndexes[n]
	if idx == nil || len(idx.ds) != len(ds) || &idx.ds[0] != &ds[0] {
		idx = &customIndex{ds: ds, m: map[string]*cssDecl{}}
		for _, d := range ds {
			if strings.HasPrefix(d.Property, "--") {
				idx.m[d.Property] = d
			}
		}
		if c.customIndexes == nil {
			c.customIndexes = map[*html.Node]*customIndex{}
		}
		c.customIndexes[n] = idx
	}
	return idx.m[name]
}

// displayOf returns the display of the given node, with the children of
// flex and grid containers blockified, as in CSS:
// they are flex or grid items, which are block-level.
func (c *Context) displayOf(n *html.Node) string {
	if c.Scripting && n != nil && n.Type == html.ElementNode && n.Data == "noscript" {
		return "none" // its content is for browsers without scripts
	}
	if c.Unrendered != nil && n != nil && n.Type == html.ElementNode && c.Unrendered(n) {
		return "none" // as a child of a shadow host that is in no slot
	}
	d := c.declaration(n, "display")
	if n == nil || n.Type != html.ElementNode {
		return d
	}
	parent := c.parentOf(n) // (in the composed tree, see shadow.go)
	if parent == nil || parent.Type != html.ElementNode {
		return d
	}
	// absolutely positioned and floating elements are blockified too
	pos := c.declaration(n, "position")
	if f := c.declaration(n, "float"); f != "" && f != "none" && pos != "absolute" && pos != "fixed" {
		pos = "absolute"
	}
	switch pos {
	case "absolute", "fixed":
		switch d {
		case "", "inline", "inline-block", "list-item":
			return "block"
		case "inline-flex":
			return "flex"
		}
	}
	// (the parent of an element whose own box is none, as a slot, is that of
	// the box that its children are in)
	for i := 0; i < 8 && parent != nil && parent.Type == html.ElementNode && c.declaration(parent, "display") == "contents"; i++ {
		parent = c.parentOf(parent)
	}
	if parent == nil || parent.Type != html.ElementNode {
		return d
	}
	switch c.declaration(parent, "display") {
	case "flex", "inline-flex", "grid", "inline-grid":
		switch d {
		case "", "inline", "inline-block", "list-item", "flow-root":
			return "block"
		case "inline-flex":
			return "flex"
		case "inline-grid":
			return "grid"
		case "inline-table":
			return "table"
		}
	}
	if (d == "" || d == "inline") && richTags[n.Data] && n.Data != "br" && c.isInlineBox(n) {
		return "inline-block" // a box of its own in its line (see isInlineBox)
	}
	return d
}

// addStyleSheets adds the style sheets of the style and link elements of
// the document of the given root node, in document order, so that they
// apply to all of the elements, as in browsers, wherever they are.
func (c *Context) addStyleSheets(root *html.Node) {
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "template":
				return
			case "style":
				c.recordSheet(n)
				if c.mediaMatches(GetAttr(n, "media")) {
					defer c.enterScope(n)()
					c.addStyleOrigin(nodeText(n), css.OriginAuthor)
				}
				return
			case "link":
				c.recordSheet(n)
				if strings.EqualFold(GetAttr(n, "rel"), "stylesheet") && c.mediaMatches(GetAttr(n, "media")) {
					defer c.enterScope(n)()
					c.addLinkedStyle(GetAttr(n, "href"))
				}
				return
			case "svg":
				return
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(root)
}

// addLinkedStyle adds the style sheet at the given URL.
// Its URL is relative to the URL of the style sheet that it is in,
// cssBase, if it is imported by one, or else to the page.
func (c *Context) addLinkedStyle(href string) {
	if strings.TrimSpace(href) == "" {
		return // as in browsers (it would be the page itself)
	}
	u, err := parseRelativeURL(href, c.cssBaseURL())
	if errors.Log(err) != nil {
		return
	}
	if c.cssImports == nil {
		c.cssImports = map[string]bool{}
	}
	if c.cssImports[u.String()] || len(c.cssImports) > 100 { // cycles
		return
	}
	c.cssImports[u.String()] = true
	resp, err := Get(c, u.String())
	if logGetError(err) != nil {
		return
	}
	defer resp.Body.Close()
	if ct := strings.ToLower(resp.Header.Get("Content-Type")); ct != "" && !strings.Contains(ct, "css") && !strings.Contains(ct, "text/plain") && !strings.Contains(ct, "octet-stream") {
		// as in browsers, a style sheet must be CSS (not, as for a link
		// of a style sheet that is missing, a page of HTML, on which the
		// CSS parser does not end)
		return
	}
	b, err := io.ReadAll(resp.Body)
	if errors.Log(err) != nil {
		return
	}
	base := c.cssBase
	c.cssBase = u.String()
	c.addStyleOrigin(string(b), css.OriginAuthor)
	c.cssBase = base
}

// absoluteURLs returns the given CSS value with the relative URLs of its
// url() functions made absolute, for the URL of the current style sheet
// (to which they are relative, as the icons of MDN in its style sheets),
// as they are used later relative to the page (see [Context.cssBaseURL]).
func (c *Context) absoluteURLs(v string) string {
	if c.cssBase == "" || !strings.Contains(v, "url(") {
		return v
	}
	return css.URLPattern.ReplaceAllStringFunc(v, func(m string) string {
		sm := css.URLPattern.FindStringSubmatch(m)
		u := sm[1] + sm[2] + sm[3]
		if u == "" || strings.HasPrefix(u, "data:") || strings.HasPrefix(u, "#") {
			return m
		}
		abs, err := parseRelativeURL(u, c.cssBase)
		if err != nil {
			return m
		}
		return `url("` + strings.ReplaceAll(abs.String(), `"`, `%22`) + `")`
	})
}

// cssBaseURL returns the URL that the URLs in the current style sheet are
// relative to: the URL of the style sheet, or that of the page.
func (c *Context) cssBaseURL() string {
	if c.cssBase != "" {
		return c.cssBase
	}
	return c.PageURL
}

// addHoverTrigger adds the trigger of the given selector with :hover, if
// the compound selector with :hover is followed by others (as .menuItem of
// .menuItem:hover > a ~ .submenu): the hover of the elements that it
// selects changes the style of others (see [Context.HoverAffects]).
func (c *Context) addHoverTrigger(sel string) {
	i := strings.Index(sel, ":hover")
	if i < 0 {
		return
	}
	isCombinator := func(b byte) bool { return b == ' ' || b == '>' || b == '+' || b == '~' }
	depth, start := 0, 0
	for j := i - 1; j >= 0; j-- {
		switch sel[j] {
		case ')':
			depth++
		case '(':
			depth--
		}
		if depth == 0 && isCombinator(sel[j]) {
			start = j + 1
			break
		}
	}
	end := len(sel)
	depth = 0
	for j := i + len(":hover"); j < len(sel); j++ {
		switch sel[j] {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth == 0 && isCombinator(sel[j]) {
			end = j
			break
		}
	}
	if strings.TrimSpace(sel[end:]) == "" {
		return // the hovered element itself, which its widget shows
	}
	compound := strings.ReplaceAll(sel[start:end], ":hover", "")
	if strings.TrimSpace(compound) == "" {
		compound = "*"
	}
	if m, err := cascadia.Parse(css.RewriteSelector(compound)); err == nil {
		c.hoverTriggers = append(c.hoverTriggers, m)
	}
}

// HoverAffects returns whether the hover of the given element changes the
// styles of other elements, by the :hover selectors of the style sheets of
// the page that was read (as the menu items of pkg.go.dev, whose submenus
// are shown on their hover), for which an app that sets [Context.Hovered]
// reads the page again when the hover of the element changes.
func (c *Context) HoverAffects(n *html.Node) bool {
	if n == nil || n.Type != html.ElementNode {
		return false
	}
	for _, m := range c.hoverTriggers {
		if m.Match(n) {
			return true
		}
	}
	return false
}
