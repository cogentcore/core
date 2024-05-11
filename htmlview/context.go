// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlview

import (
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
	"github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"
	selcss "github.com/ericchiang/css"
	"go.abhg.dev/goldmark/wikilink"
	"golang.org/x/net/html"
)

// Context contains context information about the current state of a htmlview
// reader and its surrounding context. It should be created with [NewContext].
type Context struct {
	// Node is the node that is currently being read.
	Node *html.Node

	// Styles are the CSS styling rules for each node.
	Styles map[*html.Node][]*css.Rule

	// Widgets are the core widgets for each node.
	Widgets map[*html.Node]core.Widget

	// NewParent is the current parent widget that children of
	// the previously read element should be added to, if any.
	NewParent core.Widget

	// BlockParent is the current parent widget that non-inline elements
	// should be added to.
	BlockParent core.Widget

	// InlinePw is the current parent widget that inline
	// elements should be added to; it must be got through
	// [Context.InlineParent], as it may need to be constructed
	// on the fly. However, it can be set directly.
	InlinePw core.Widget

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
		Widgets: map[*html.Node]core.Widget{},
		OpenURL: system.TheApp.OpenURL,
	}
}

// Parent returns the current parent widget that a widget
// associated with the current node should be added to.
// It may make changes to the widget tree, so the widget
// must be added to the resulting parent immediately.
func (c *Context) Parent() core.Widget {
	rules := c.Styles[c.Node]
	display := ""
	for _, rule := range rules {
		for _, decl := range rule.Declarations {
			if decl.Property == "display" {
				display = decl.Value
			}
		}
	}
	var parent core.Widget
	switch display {
	case "inline", "inline-block", "":
		parent = c.InlineParent()
	default:
		parent = c.BlockParent
		c.InlinePw = nil
	}
	return parent
}

// Config configures the given widget. It needs to be called
// on all widgets that are not configured through the [New]
// pathway.
func (c *Context) Config(w core.Widget) {
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
			if errors.Log(err) != nil {
				continue
			}
			rule := &css.Rule{Declarations: decls}
			if c.Styles == nil {
				c.Styles = map[*html.Node][]*css.Rule{}
			}
			c.Styles[c.Node] = append(c.Styles[c.Node], rule)
		default:
			wb.SetProperty(attr.Key, attr.Val)
		}
	}
	wb.SetProperty("tag", c.Node.Data)
	rules := c.Styles[c.Node]
	w.Style(func(s *styles.Style) {
		for _, rule := range rules {
			for _, decl := range rule.Declarations {
				// TODO(kai/styproperties): parent style and context
				s.StyleFromProp(s, decl.Property, decl.Value, colors.BaseContext(colors.ToUniform(s.Color)))
			}
		}
	})
}

// InlineParent returns the current parent widget that inline
// elements should be added to.
func (c *Context) InlineParent() core.Widget {
	if c.InlinePw != nil {
		return c.InlinePw
	}
	c.InlinePw = core.NewLayout(c.BlockParent)
	c.InlinePw.SetName("inline-container")
	tree.SetUniqueName(c.InlinePw)
	c.InlinePw.Style(func(s *styles.Style) {
		s.Grow.Set(1, 0)
	})
	return c.InlinePw
}

// AddStyle adds the given CSS style string to the page's compiled styles.
func (c *Context) AddStyle(style string) {
	ss, err := parser.Parse(style)
	if errors.Log(err) != nil {
		return
	}

	root := RootNode(c.Node)

	for _, rule := range ss.Rules {
		var sel *selcss.Selector
		if len(rule.Selectors) > 0 {
			s, err := selcss.Parse(strings.Join(rule.Selectors, ","))
			if errors.Log(err) != nil {
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
