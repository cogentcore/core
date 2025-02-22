// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"net/http"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
	"github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"
	selcss "github.com/ericchiang/css"
	"golang.org/x/net/html"
)

// Context contains context information about the current state of a htmlcore
// reader and its surrounding context. It should be created with [NewContext].
type Context struct {

	// Node is the node that is currently being read.
	Node *html.Node

	// styles are the CSS styling rules for each node.
	styles map[*html.Node][]*css.Rule

	// NewParent is the current parent widget that children of
	// the previously read element should be added to, if any.
	NewParent core.Widget

	// BlockParent is the current parent widget that non-inline elements
	// should be added to.
	BlockParent core.Widget

	// inlineParent is the current parent widget that inline
	// elements should be added to; it must be got through
	// [Context.InlineParent], as it may need to be constructed
	// on the fly. However, it can be set directly.
	inlineParent core.Widget

	// PageURL, if not "", is the URL of the current page.
	// Otherwise, there is no current page.
	PageURL string

	// OpenURL is the function used to open URLs,
	// which defaults to [system.App.OpenURL].
	OpenURL func(url string)

	// GetURL is the function used to get resources from URLs,
	// which defaults to [http.Get].
	GetURL func(url string) (*http.Response, error)

	// ElementHandlers is a map of handler functions for each HTML element
	// type (eg: "button", "input", "p"). It is empty by default, but can be
	// used by anyone in need of behavior different than the default behavior
	// defined in [handleElement] (for example, for custom elements).
	// If the handler for an element returns false, then the default behavior
	// for an element is used.
	ElementHandlers map[string]func(ctx *Context) bool

	// WikilinkHandlers is a list of handlers to use for wikilinks.
	// If one returns "", "", the next ones will be tried instead.
	// The functions are tried in sequential ascending order.
	// See [Context.AddWikilinkHandler] to add a new handler.
	WikilinkHandlers []WikilinkHandler
}

// NewContext returns a new [Context] with basic defaults.
func NewContext() *Context {
	return &Context{
		styles:          map[*html.Node][]*css.Rule{},
		OpenURL:         system.TheApp.OpenURL,
		GetURL:          http.Get,
		ElementHandlers: map[string]func(ctx *Context) bool{},
	}
}

// Parent returns the current parent widget that a widget
// associated with the current node should be added to.
// It may make changes to the widget tree, so the widget
// must be added to the resulting parent immediately.
func (c *Context) Parent() core.Widget {
	rules := c.styles[c.Node]
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
		c.inlineParent = nil
	}
	return parent
}

// config configures the given widget. It needs to be called
// on all widgets that are not configured through the [New]
// pathway.
func (c *Context) config(w core.Widget) {
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
			if c.styles == nil {
				c.styles = map[*html.Node][]*css.Rule{}
			}
			c.styles[c.Node] = append(c.styles[c.Node], rule)
		default:
			wb.SetProperty(attr.Key, attr.Val)
		}
	}
	wb.SetProperty("tag", c.Node.Data)
	rules := c.styles[c.Node]
	wb.Styler(func(s *styles.Style) {
		for _, rule := range rules {
			for _, decl := range rule.Declarations {
				// TODO(kai/styproperties): parent style and context
				s.StyleFromProperty(s, decl.Property, decl.Value, colors.BaseContext(colors.ToUniform(s.Color)))
			}
		}
	})
}

// InlineParent returns the current parent widget that inline
// elements should be added to.
func (c *Context) InlineParent() core.Widget {
	if c.inlineParent != nil {
		return c.inlineParent
	}
	c.inlineParent = core.NewFrame(c.BlockParent)
	c.inlineParent.AsTree().SetName("inline-container")
	tree.SetUniqueName(c.inlineParent)
	c.inlineParent.AsWidget().Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
	})
	return c.inlineParent
}

// addStyle adds the given CSS style string to the page's compiled styles.
func (c *Context) addStyle(style string) {
	ss, err := parser.Parse(style)
	if errors.Log(err) != nil {
		return
	}

	root := rootNode(c.Node)

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
			c.styles[match] = append(c.styles[match], rule)
		}
	}
}

// LinkButton is a helper function that makes the given button
// open the given link when clicked on, using [Context.OpenURL].
// The advantage of using this is that it does [tree.NodeBase.SetProperty]
// of "href" to the given url, allowing generatehtml to create an <a> element
// for HTML preview and SEO purposes.
//
// See also [Context.LinkButtonUpdating] for a dynamic version.
func (c *Context) LinkButton(bt *core.Button, url string) *core.Button {
	bt.SetProperty("tag", "a")
	bt.SetProperty("href", url)
	bt.OnClick(func(e events.Event) {
		c.OpenURL(url)
	})
	return bt
}

// LinkButtonUpdating is a version of [Context.LinkButton] that is robust to a changing/dynamic
// URL, using an Updater and a URL function instead of a variable.
func (c *Context) LinkButtonUpdating(bt *core.Button, url func() string) *core.Button {
	bt.SetProperty("tag", "a")
	bt.Updater(func() {
		bt.SetProperty("href", url())
	})
	bt.OnClick(func(e events.Event) {
		c.OpenURL(url())
	})
	return bt
}
