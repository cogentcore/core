// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package coredom

import (
	"fmt"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/styles"
	"github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"
	selcss "github.com/ericchiang/css"
	"go.abhg.dev/goldmark/wikilink"
	"golang.org/x/net/html"
)

// Context contains context information about the current state of a coredom
// reader and its surrounding context. It should be created with [NewContext].
type Context struct {
	// Node is the node that is currently being read.
	Node *html.Node

	// Styles are the CSS styling rules for each node.
	Styles map[*html.Node][]*css.Rule

	// Widgets are the gi widgets for each node.
	Widgets map[*html.Node]gi.Widget

	// NewParent is the current parent widget that children of
	// the previously read element should be added to, if any.
	NewParent gi.Widget

	// BlockParent is the current parent widget that non-inline elements
	// should be added to.
	BlockParent gi.Widget

	// InlinePw is the current parent widget that inline
	// elements should be added to; it must be got through
	// [Context.InlineParent], as it may need to be constructed
	// on the fly. However, it can be set directly.
	InlinePw gi.Widget

	// PageURL, if not "", is the URL of the current page.
	// Otherwise, there is no current page.
	PageURL string

	// OpenURL is the function used to open URLs.
	OpenURL func(url string)

	// WikilinkResolver, if specified, is the function used to convert wikilinks into full URLs.
	WikilinkResolver func(w *wikilink.Node) (destination []byte, err error)
}

// NewContext returns a new [Context] with basic defaults.
func NewContext() *Context {
	return &Context{
		Styles:  map[*html.Node][]*css.Rule{},
		Widgets: map[*html.Node]gi.Widget{},
		OpenURL: goosi.TheApp.OpenURL,
	}
}

// Parent returns the current parent widget that a widget
// associated with the current node should be added to.
// It may make changes to the widget tree, so the widget
// must be added to the resulting parent immediately.
func (c *Context) Parent() gi.Widget {
	rules := c.Styles[c.Node]
	display := ""
	for _, rule := range rules {
		for _, decl := range rule.Declarations {
			if decl.Property == "display" {
				display = decl.Value
			}
		}
	}
	var par gi.Widget
	switch display {
	case "inline", "inline-block", "":
		par = c.InlineParent()
	default:
		par = c.BlockParent
		c.InlinePw = nil
	}
	return par
}

// Config configures the given widget. It needs to be called
// on all widgets that are not configured through the [New]
// pathway.
func (c *Context) Config(w gi.Widget) {
	wb := w.AsWidget()
	for _, attr := range c.Node.Attr {
		switch attr.Key {
		case "id":
			wb.SetName(attr.Val)
		case "style":
			// our CSS parser is strict about semicolons, but
			// they aren't needed in normal inline styles in HTML
			if !strings.HasSuffix(attr.Val, ";") {
				attr.Val += ";"
			}
			decls, err := parser.ParseDeclarations(attr.Val)
			if grr.Log(err) != nil {
				continue
			}
			rule := &css.Rule{Declarations: decls}
			if c.Styles == nil {
				c.Styles = map[*html.Node][]*css.Rule{}
			}
			c.Styles[c.Node] = append(c.Styles[c.Node], rule)
		default:
			wb.SetProp(attr.Key, attr.Val)
		}
	}
	wb.SetProp("tag", c.Node.Data)
	rules := c.Styles[c.Node]
	w.Style(func(s *styles.Style) {
		for _, rule := range rules {
			for _, decl := range rule.Declarations {
				// TODO(kai/styprops): parent style and context
				s.StyleFromProp(s, decl.Property, decl.Value, colors.BaseContext(colors.ToUniform(s.Color)))
			}
		}
	})
}

// InlineParent returns the current parent widget that inline
// elements should be added to.
func (c *Context) InlineParent() gi.Widget {
	if c.InlinePw != nil {
		return c.InlinePw
	}
	c.InlinePw = gi.NewLayout(c.BlockParent, fmt.Sprintf("inline-container-%d", c.BlockParent.NumLifetimeChildren()))
	c.InlinePw.Style(func(s *styles.Style) {
		s.Grow.Set(1, 0)
	})
	return c.InlinePw
}

// AddStyle adds the given CSS style string to the page's compiled styles.
func (c *Context) AddStyle(style string) {
	ss, err := parser.Parse(style)
	if grr.Log(err) != nil {
		return
	}

	root := RootNode(c.Node)

	for _, rule := range ss.Rules {
		var sel *selcss.Selector
		if len(rule.Selectors) > 0 {
			s, err := selcss.Parse(strings.Join(rule.Selectors, ","))
			if grr.Log(err) != nil {
				s = &selcss.Selector{}
			}
			sel = s
		} else {
			sel = &selcss.Selector{}
		}

		matches := sel.Select(root)
		for _, match := range matches {
			c.Styles[match] = append(c.Styles[match], rule)
		}
	}
}
