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
	"golang.org/x/net/html"
)

// Context contains context information about the current state of a coreodm
// reader and its surrounding context.
type Context interface {
	// Node returns the node that is currently being read.
	Node() *html.Node

	// SetNode sets the node that is currently being read.
	SetNode(node *html.Node)

	// Parent returns the current parent widget that a widget
	// associated with the current node should be added to.
	// It may make changes to the widget tree, so the widget
	// must be added to the resulting parent immediately.
	Parent() gi.Widget

	// Config configures the given widget. It needs to be called
	// on all widgets that are not configured through the [New]
	// pathway.
	Config(w gi.Widget)

	// NewParent returns the current parent widget that children of
	// the previously read element should be added to, if any.
	NewParent() gi.Widget

	// SetNewParent sets the current parent widget that children of
	// the previous read element should be added to, if any.
	SetNewParent(pw gi.Widget)

	// BlockParent returns the current parent widget that non-inline elements
	// should be added to.
	BlockParent() gi.Widget

	// SetBlockParent sets the current parent widget that non-inline elements
	// should be added to.
	SetBlockParent(pw gi.Widget)

	// InlineParent returns the current parent widget that inline
	// elements should be added to.
	InlineParent() gi.Widget

	// SetInlineParent sets the current parent widget that inline elements
	// should be added to.
	SetInlineParent(pw gi.Widget)

	// PageURL returns the URL of the current page, and "" if there
	// is no current page.
	PageURL() string

	// OpenURL opens the given URL.
	OpenURL(url string)

	// Style returns the styling rules for the node that is currently being read.
	Style() []*css.Rule

	// AddStyle adds the given CSS style string to the page's compiled styles.
	AddStyle(style string)
}

// BaseContext returns a [Context] with basic implementations of all functions.
func BaseContext() Context {
	return &ContextBase{}
}

// ContextBase contains basic implementations of all [Context] functions.
type ContextBase struct {
	Nd *html.Node

	Rules map[*html.Node][]*css.Rule

	WidgetsForNodes map[*html.Node]gi.Widget
	BlockPw         gi.Widget
	InlinePw        gi.Widget
	NewPw           gi.Widget
}

func (cb *ContextBase) Node() *html.Node {
	return cb.Nd
}

func (cb *ContextBase) SetNode(node *html.Node) {
	cb.Nd = node
}

func (cb *ContextBase) Parent() gi.Widget {
	rules := cb.Style()
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
		par = cb.InlineParent()
	default:
		par = cb.BlockParent()
		cb.SetInlineParent(nil)
	}
	return par
}

func (cb *ContextBase) Config(w gi.Widget) {
	wb := w.AsWidget()
	for _, attr := range cb.Node().Attr {
		switch attr.Key {
		case "id":
			wb.SetName(attr.Val)
		case "class":
			// wb.SetClass(attr.Val)
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
			if cb.Rules == nil {
				cb.Rules = map[*html.Node][]*css.Rule{}
			}
			cb.Rules[cb.Node()] = append(cb.Rules[cb.Node()], rule)
		default:
			wb.SetProp(attr.Key, attr.Val)
		}
	}
	wb.SetProp("tag", cb.Node().Data)
	rules := cb.Style()
	w.Style(func(s *styles.Style) {
		for _, rule := range rules {
			for _, decl := range rule.Declarations {
				// TODO(kai/styprops): parent style and context
				s.StyleFromProp(s, decl.Property, decl.Value, colors.BaseContext(s.Color))
			}
		}
	})
}

func (cb *ContextBase) NewParent() gi.Widget {
	return cb.NewPw
}

func (cb *ContextBase) SetNewParent(pw gi.Widget) {
	cb.NewPw = pw
}

func (cb *ContextBase) BlockParent() gi.Widget {
	return cb.BlockPw
}

func (cb *ContextBase) SetBlockParent(pw gi.Widget) {
	cb.BlockPw = pw
}

func (cb *ContextBase) InlineParent() gi.Widget {
	if cb.InlinePw != nil {
		return cb.InlinePw
	}
	cb.InlinePw = gi.NewLayout(cb.BlockPw, fmt.Sprintf("inline-container-%d", cb.BlockPw.NumLifetimeChildren()))
	cb.InlinePw.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	return cb.InlinePw
}

func (cb *ContextBase) SetInlineParent(pw gi.Widget) {
	cb.InlinePw = pw
}

func (cb *ContextBase) PageURL() string { return "" }

func (cb *ContextBase) OpenURL(url string) {
	goosi.TheApp.OpenURL(url)
}

func (cb *ContextBase) Style() []*css.Rule {
	if cb.Rules == nil {
		return nil
	}
	return cb.Rules[cb.Node()]
}

func (cb *ContextBase) AddStyle(style string) {
	ss, err := parser.Parse(style)
	if grr.Log(err) != nil {
		return
	}

	if cb.Rules == nil {
		cb.Rules = map[*html.Node][]*css.Rule{}
	}

	root := RootNode(cb.Node())

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
			cb.Rules[match] = append(cb.Rules[match], rule)
		}
	}
}
