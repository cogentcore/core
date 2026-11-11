// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"io"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"

	"cogentcore.org/core/base/errors"
	"github.com/andybalholm/cascadia"
	"github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// cssOrigin is the origin of a CSS declaration, which orders the cascade.
type cssOrigin int

const (
	// originUserAgent is for the user agent (default) styles.
	originUserAgent cssOrigin = iota

	// originHint is for the presentational attributes of
	// elements (width, bgcolor etc), which come before author styles.
	originHint

	// originAuthor is for the styles of the page.
	originAuthor

	// originInline is for the style attribute of an element.
	originInline
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

	origin      cssOrigin
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
	if d.origin == originUserAgent {
		return 10
	}
	return 5 + int(d.origin)
}

// addRules adds the rules of the given style sheet with the given
// origin to the declarations of the elements under root that they select.
func (c *Context) addRules(root *html.Node, ss *css.Stylesheet, origin cssOrigin) {
	if c.styles == nil {
		c.styles = map[*html.Node][]*cssDecl{}
	}
	c.addRuleList(root, ss.Rules, origin)
}

// addRuleList adds the given rules, including those in @media rules
// that match and in @supports rules.
func (c *Context) addRuleList(root *html.Node, rules []*css.Rule, origin cssOrigin) {
	for _, rule := range rules {
		if rule.Kind == css.AtRule {
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
				url, media := importURL(rule.Prelude)
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
			selectors = splitSelectors(rule.Prelude)
		}
		for _, sel := range selectors {
			sel = strings.TrimSpace(sel)
			if strings.Contains(sel, ":hover") {
				c.addHoverTrigger(sel)
			}
			if base, kind, ok := pseudoSelector(sel); ok {
				c.addPseudoRule(root, base, kind, rule.Declarations, origin)
				continue
			}
			spec := specificity(sel)
			cs := compileSelector(sel)
			if cs != nil {
				spec = cs.specificity
				c.recordRule(root, sel, cs, rule.Declarations, origin, spec)
			}
			if !c.recordOnly && !c.deferMatch { // (see Context.addedStyle and Context.ReuseStyles)
				for _, n := range c.selectNodes(root, sel) {
					c.addDecls(n, rule.Declarations, origin, spec)
				}
			}
			c.curRule = 0
		}
	}
}

// addDecls adds the given declarations to the given node.
func (c *Context) addDecls(n *html.Node, decls []*css.Declaration, origin cssOrigin, spec int) {
	for _, d := range decls {
		prop, value := strings.ToLower(strings.TrimSpace(d.Property)), strings.TrimSpace(d.Value)
		if !validDeclaration(prop, value) {
			continue // dropped, so that the declaration before it applies
		}
		c.cssOrder++
		c.styles[n] = append(c.styles[n], &cssDecl{Property: prop,
			Value: c.absoluteURLs(value), Important: d.Important, origin: origin, specificity: spec, order: c.cssOrder,
			container: c.curContainer, rule: c.curRule})
	}
	c.varMemo = nil
	if c.curContainer != nil {
		if c.containerQueried == nil {
			c.containerQueried = map[*html.Node]bool{}
		}
		c.containerQueried[n] = true
	}
	delete(c.sorted, n)
}

// otherVendorValue matches a keyword of the vendor prefix of another
// browser engine (as -moz-available), which is invalid in browsers of
// WebKit and Blink, as Cogent Core is.
var otherVendorValue = regexp.MustCompile(`(^|[\s,(])-(moz|ms|o)-[a-z]`)

// validDeclaration returns whether a declaration of the given property
// and value is valid, as in browsers, which drop the declarations that
// are not when they are parsed, so that the declarations before them of
// the same property apply (as height: 100vh before height:
// -webkit-fill-available, of the captcha frames of DataDome for Safari,
// which were of the height of no content). A size of
// -webkit-fill-available is invalid here too, as it is not laid out, and
// the declaration before it is meant as a fallback for it. Custom
// properties and values with variables are only known when they are used.
func validDeclaration(prop, value string) bool {
	if strings.HasPrefix(prop, "--") || strings.Contains(value, "var(") {
		return true
	}
	if otherVendorValue.MatchString(strings.ToLower(value)) {
		return false
	}
	switch prop {
	case "width", "height", "min-width", "min-height", "max-width", "max-height", "inline-size", "block-size":
		if strings.Contains(strings.ToLower(value), "-webkit-fill-available") {
			return false
		}
	}
	return true
}

// addHint adds a declaration of a presentational attribute to the node.
func (c *Context) addHint(n *html.Node, prop, value string) {
	c.addDecls(n, []*css.Declaration{{Property: prop, Value: value}}, originHint, 0)
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

// declaration returns the value of the given property for the given node
// from its declarations (not inherited), or "" if it has none.
func (c *Context) declaration(n *html.Node, prop string) string {
	ds := c.declarations(n)
	for i := len(ds) - 1; i >= 0; i-- {
		if ds[i].Property == prop {
			if ds[i].invalid {
				return ""
			}
			if prop == "display" {
				return legacyDisplay(ds[i].Value)
			}
			return ds[i].Value
		}
	}
	return ""
}

// legacyDisplay returns the display for the given value of the display
// property, which is that of the prefixed legacy values: -webkit-box (as
// with -webkit-line-clamp, in a column) is a block, and -webkit-flex flex;
// unset and initial are inline.
func legacyDisplay(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "-webkit-box", "-moz-box":
		return "block"
	case "-webkit-inline-box", "-moz-inline-box":
		return "inline-block"
	case "-webkit-flex":
		return "flex"
	case "-webkit-inline-flex":
		return "inline-flex"
	case "unset", "initial":
		// display is not inherited, so these are its initial value (as
		// display: unset of the widgets of the masthead of nytimes.com, in
		// a flex row, which had no box of its own, so that its ticker was
		// as wide as the row, over the logo)
		return "inline"
	}
	return v
}

// lastDeclaration returns the property and value of the declaration of
// the given element that applies of those of the given properties (as a
// shorthand and its longhand): the last of them in the cascade, or "" if
// there is none (or it is invalid).
func (c *Context) lastDeclaration(n *html.Node, props ...string) (prop, value string) {
	ds := c.declarations(n)
	for i := len(ds) - 1; i >= 0; i-- {
		if slices.Contains(props, ds[i].Property) {
			if ds[i].invalid {
				return ds[i].Property, ""
			}
			return ds[i].Property, ds[i].Value
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
	c.addDecls(n, decls, originInline, 0)
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
	c.addDecls(c.Node, decls, originInline, 0)
	return nil
}

// compiledSelector is a compiled selector, with its specificity.
type compiledSelector struct {
	sel         cascadia.Sel
	specificity int

	// keyKind and key are a part of the last compound selector that every
	// node that it selects has, for finding them in a [selectorIndex]:
	// an id ('#'), a class ('.'), or an element name ('t'), or none (0).
	keyKind byte
	key     string
}

// selectorCache has the compiled selectors, with nil for ones
// that are not supported (as pseudo-elements, which select no nodes).
// It is shared by the contexts that read pages, as in other goroutines.
var selectorCache = map[string]*compiledSelector{}

// selectorCacheMu protects selectorCache.
var selectorCacheMu sync.Mutex

// compileSelector returns the compiled selector, or nil if it is not supported.
func compileSelector(sel string) *compiledSelector {
	selectorCacheMu.Lock()
	defer selectorCacheMu.Unlock()
	if cs, ok := selectorCache[sel]; ok {
		return cs
	}
	var cs *compiledSelector
	if s, err := cascadia.Parse(rewriteSelector(sel)); err == nil && s.PseudoElement() == "" {
		sp := s.Specificity()
		cs = &compiledSelector{sel: s, specificity: sp[0]*10000 + sp[1]*100 + sp[2]}
		cs.keyKind, cs.key = selectorKey(sel)
	}
	selectorCache[sel] = cs
	return cs
}

// rewriteSelector rewrites the parts of the given selector that the selector
// library does not support as ones that it does: :is(X) and :where(X) as
// :not(:not(X)), and :focus-within and :focus-visible as :focus (which
// selects nothing, as for :hover, in a static rendering).
func rewriteSelector(sel string) string {
	// :hover selects the elements that are marked hovered (see
	// [Context.Hovered]), or none
	sel = strings.ReplaceAll(sel, ":hover", "["+hoverAttr+"]")
	// :modal selects the dialogs that scripts show as modal (with
	// showModal), which they mark with ModalAttr
	sel = strings.ReplaceAll(sel, ":modal", "["+ModalAttr+"]")
	if !strings.Contains(sel, ":is(") && !strings.Contains(sel, ":where(") && !strings.Contains(sel, ":focus-") {
		return sel
	}
	sel = strings.ReplaceAll(sel, ":focus-within", ":focus")
	sel = strings.ReplaceAll(sel, ":focus-visible", ":focus")
	var sb strings.Builder
	var closes []bool // for each open parenthesis, whether its close is doubled
	for i := 0; i < len(sel); i++ {
		switch {
		case strings.HasPrefix(sel[i:], ":is("):
			sb.WriteString(":not(:not(")
			closes = append(closes, true)
			i += len(":is(") - 1
		case strings.HasPrefix(sel[i:], ":where("):
			sb.WriteString(":not(:not(")
			closes = append(closes, true)
			i += len(":where(") - 1
		case sel[i] == '(':
			sb.WriteByte('(')
			closes = append(closes, false)
		case sel[i] == ')':
			sb.WriteByte(')')
			if n := len(closes); n > 0 {
				if closes[n-1] {
					sb.WriteByte(')')
				}
				closes = closes[:n-1]
			}
		default:
			sb.WriteByte(sel[i])
		}
	}
	return sb.String()
}

// selectorKey returns a part of the last compound selector of the given
// selector that every node that it selects has (see compiledSelector.key):
// its id, else its first class, else its element name, or none if it has
// none of them outside of functions and attribute selectors (or has
// escapes, which are not decoded here).
func selectorKey(sel string) (byte, string) {
	if strings.ContainsAny(sel, "\\|") { // escapes and namespaces
		return 0, ""
	}
	// the start of the last compound selector, after the last combinator
	// at the top level
	start, depth := 0, 0
	for i := 0; i < len(sel); i++ {
		switch ch := sel[i]; ch {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		case ' ', '>', '+', '~', '\t', '\n':
			if depth == 0 {
				start = i + 1
			}
		}
	}
	last := sel[start:]
	isName := func(ch byte) bool {
		return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_' || ch >= 0x80
	}
	name := func(i int) string {
		j := i
		for j < len(last) && isName(last[j]) {
			j++
		}
		return last[i:j]
	}
	var class string
	depth = 0
	for i := 0; i < len(last); i++ {
		switch ch := last[i]; {
		case ch == '(' || ch == '[':
			depth++
		case ch == ')' || ch == ']':
			depth--
		case depth == 0 && ch == '#':
			if id := name(i + 1); id != "" {
				return '#', id
			}
		case depth == 0 && ch == '.' && class == "":
			class = name(i + 1)
		}
	}
	if class != "" {
		return '.', class
	}
	if tag := name(0); tag != "" {
		return 't', strings.ToLower(tag)
	}
	return 0, ""
}

// selectorIndex has the element nodes of a document by id, class, and
// element name, in document order, for selecting the nodes of style rules
// (see [Context.selectNodes]), so that each selector is only matched
// against the nodes that have its key, as in browsers.
type selectorIndex struct {
	root                 *html.Node
	byID, byClass, byTag map[string][]*html.Node
	all                  []*html.Node
}

// newSelectorIndex returns the index of the element nodes under root.
func newSelectorIndex(root *html.Node) *selectorIndex {
	ix := &selectorIndex{root: root, byID: map[string][]*html.Node{}, byClass: map[string][]*html.Node{}, byTag: map[string][]*html.Node{}}
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != html.ElementNode {
				continue
			}
			ix.all = append(ix.all, c)
			ix.byTag[strings.ToLower(c.Data)] = append(ix.byTag[strings.ToLower(c.Data)], c)
			for _, a := range c.Attr {
				if a.Namespace != "" {
					continue
				}
				switch a.Key {
				case "id":
					ix.byID[a.Val] = append(ix.byID[a.Val], c)
				case "class":
					seen := map[string]bool{}
					for _, cl := range strings.Fields(a.Val) {
						if !seen[cl] {
							seen[cl] = true
							ix.byClass[cl] = append(ix.byClass[cl], c)
						}
					}
				}
			}
			walk(c)
		}
	}
	walk(root)
	return ix
}

// selectNodes returns the nodes under root that the given selector selects,
// using the index of the nodes of root, which it makes for the first root
// (the nodes of a document do not change while it is read).
func (c *Context) selectNodes(root *html.Node, sel string) []*html.Node {
	cs := compileSelector(sel)
	if cs == nil || cs.keyKind == 0 {
		return selectNodes(root, sel)
	}
	if c.selIndex == nil || c.selIndex.root != root {
		c.selIndex = newSelectorIndex(root)
	}
	var cands []*html.Node
	switch cs.keyKind {
	case '#':
		cands = c.selIndex.byID[cs.key]
	case '.':
		cands = c.selIndex.byClass[cs.key]
	case 't':
		cands = c.selIndex.byTag[cs.key]
	}
	var matches []*html.Node
	for _, n := range cands {
		if cs.sel.Match(n) {
			matches = append(matches, n)
		}
	}
	return matches
}

// selectNodes returns the nodes under root that the given selector selects.
func selectNodes(root *html.Node, sel string) []*html.Node {
	// Selectors of element names, such as the user agent styles have, select
	// the elements with those names, which one walk of the tree finds.
	if a := atom.Lookup([]byte(sel)); a != 0 && a.String() == sel {
		var matches []*html.Node
		var walk func(n *html.Node)
		walk = func(n *html.Node) {
			if n.DataAtom == a {
				matches = append(matches, n)
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode {
					walk(c)
				}
			}
		}
		walk(root)
		return matches
	}
	cs := compileSelector(sel)
	if cs == nil {
		return nil
	}
	return cascadia.QueryAll(root, cs.sel)
}

// specificity returns the specificity of the given selector,
// as ids*10000 + (classes, attributes, pseudo-classes)*100 + types.
func specificity(sel string) int {
	ids, classes, types := 0, 0, 0
	start := true // at the start of a compound selector
	for i := 0; i < len(sel); i++ {
		ch := sel[i]
		switch {
		case ch == '#':
			ids++
			start = false
		case ch == '.':
			classes++
			start = false
		case ch == '[':
			classes++
			for i < len(sel) && sel[i] != ']' {
				i++
			}
			start = false
		case ch == ':':
			if i+1 < len(sel) && sel[i+1] == ':' {
				types++ // pseudo-element
				i++
			} else {
				classes++
			}
			start = false
		case ch == ' ' || ch == '>' || ch == '+' || ch == '~' || ch == ',':
			start = true
			continue
		case ch == '*':
			start = false
		default:
			if start && (ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z') {
				types++
			}
			start = false
		}
		// skip the rest of the name
		for i+1 < len(sel) && isNameChar(sel[i+1]) {
			i++
		}
	}
	return ids*10000 + classes*100 + types
}

func isNameChar(ch byte) bool {
	return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_'
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
		for i := len(ds) - 1; i >= 0; i-- {
			if ds[i].Property == name {
				return ds[i]
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
	if n == nil || n.Type != html.ElementNode || n.Parent == nil || n.Parent.Type != html.ElementNode {
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
	switch c.declaration(n.Parent, "display") {
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

// importURL returns the URL and the media query list of the given
// prelude of an @import rule (url("x") media, or "x" media).
func importURL(prelude string) (url, media string) {
	p := strings.TrimSpace(prelude)
	if rest, ok := strings.CutPrefix(p, "url("); ok {
		end := strings.Index(rest, ")")
		if end < 0 {
			return "", ""
		}
		url, media = rest[:end], rest[end+1:]
	} else if len(p) > 0 && (p[0] == '"' || p[0] == '\'') {
		end := strings.IndexByte(p[1:], p[0])
		if end < 0 {
			return "", ""
		}
		url, media = p[:end+2], p[end+2:]
	} else {
		return "", ""
	}
	url = strings.Trim(strings.TrimSpace(url), `"'`)
	media = strings.TrimSpace(media)
	// layer() and supports() conditions are not supported, and are ignored
	if strings.HasPrefix(media, "layer") || strings.HasPrefix(media, "supports(") {
		media = ""
	}
	return url, media
}

// splitSelectors splits the given selector list at the commas that are
// not in parentheses, brackets, or quotes.
func splitSelectors(list string) []string {
	var sels []string
	depth := 0
	var quote byte
	start := 0
	for i := 0; i < len(list); i++ {
		ch := list[i]
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
		case ch == ',' && depth == 0:
			sels = append(sels, strings.TrimSpace(list[start:i]))
			start = i + 1
		}
	}
	return append(sels, strings.TrimSpace(list[start:]))
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
					c.addStyleOrigin(nodeText(n), originAuthor)
				}
				return
			case "link":
				c.recordSheet(n)
				if strings.EqualFold(GetAttr(n, "rel"), "stylesheet") && c.mediaMatches(GetAttr(n, "media")) {
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
	c.addStyleOrigin(string(b), originAuthor)
	c.cssBase = base
}

// cssURLs matches the url() of CSS values, with its URL quoted or not.
var cssURLs = regexp.MustCompile(`url\(\s*(?:"([^"]*)"|'([^']*)'|([^)'"\s]*))\s*\)`)

// absoluteURLs returns the given CSS value with the relative URLs of its
// url() functions made absolute, for the URL of the current style sheet
// (to which they are relative, as the icons of MDN in its style sheets),
// as they are used later relative to the page (see [Context.cssBaseURL]).
func (c *Context) absoluteURLs(v string) string {
	if c.cssBase == "" || !strings.Contains(v, "url(") {
		return v
	}
	return cssURLs.ReplaceAllStringFunc(v, func(m string) string {
		sm := cssURLs.FindStringSubmatch(m)
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

// ModalAttr is the attribute of the dialogs that are shown as modal (as
// by the showModal of scripts), which the :modal selector of CSS selects
// (and the style of browsers for them, in the top layer).
const ModalAttr = "data-htmlcore-modal"

// hoverAttr is the attribute of the elements that are hovered while a page
// is read (see [Context.Hovered]), which no element of a document has.
const hoverAttr = "data-htmlcore-hover"

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
	if m, err := cascadia.Parse(rewriteSelector(compound)); err == nil {
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
