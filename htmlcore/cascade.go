// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"sort"
	"strings"

	"github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"
	selcss "github.com/ericchiang/css"
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

	origin      cssOrigin
	specificity int
	order       int
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
	for _, rule := range ss.Rules {
		if rule.Kind != css.QualifiedRule { // @media etc are not supported yet
			continue
		}
		for _, sel := range rule.Selectors {
			sel = strings.TrimSpace(sel)
			spec := specificity(sel)
			for _, n := range selectNodes(root, sel) {
				c.addDecls(n, rule.Declarations, origin, spec)
			}
		}
	}
}

// addDecls adds the given declarations to the given node.
func (c *Context) addDecls(n *html.Node, decls []*css.Declaration, origin cssOrigin, spec int) {
	for _, d := range decls {
		c.cssOrder++
		c.styles[n] = append(c.styles[n], &cssDecl{Property: strings.ToLower(strings.TrimSpace(d.Property)),
			Value: strings.TrimSpace(d.Value), Important: d.Important, origin: origin, specificity: spec, order: c.cssOrder})
	}
	delete(c.sorted, n)
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
	return c.styles[n]
}

// declaration returns the value of the given property for the given node
// from its declarations (not inherited), or "" if it has none.
func (c *Context) declaration(n *html.Node, prop string) string {
	ds := c.declarations(n)
	for i := len(ds) - 1; i >= 0; i-- {
		if ds[i].Property == prop {
			return ds[i].Value
		}
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

// selectorCache has the compiled selectors, with nil for ones
// that are not supported.
var selectorCache = map[string]*selcss.Selector{}

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
	s, ok := selectorCache[sel]
	if !ok {
		s, _ = selcss.Parse(sel) // unsupported selectors (:hover etc) select nothing
		selectorCache[sel] = s
	}
	if s == nil {
		return nil
	}
	return s.Select(root)
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
