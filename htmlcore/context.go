// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"io"
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
	"github.com/gomarkdown/markdown/ast"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
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

	// TableParent is the current table being generated.
	TableParent *core.Frame

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
	OpenURL func(url string, e events.Event)

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

	// AttributeHandlers is a map of markdown render handler functions
	// for custom attribute values that are specified in {tag: value}
	// attributes prior to markdown elements in the markdown source.
	// The map key is the tag in the attribute, which is then passed
	// to the function, along with the markdown node being rendered.
	// Alternative or additional HTML output can be written to the given writer.
	// If the handler function returns true, then the default HTML code
	// will not be generated.
	AttributeHandlers map[string]func(ctx *Context, w io.Writer, node ast.Node, entering bool, tag, value string) bool

	// WidgetHandlers is a list of handler functions for each Widget,
	// called in sequential order for each widget that is created,
	// after it has been configured by the existing handlers etc.
	// This allows for additional styling to be applied based on the
	// type of widget, for example.
	WidgetHandlers []func(w core.Widget)

	// DelayedImageLoad causes images to be loaded at a delay.
	// if not set, then they are loaded immediately.
	DelayedImageLoad bool

	// ImageCache, if non-nil, caches the decoded images of img elements
	// by their resolved URL, so that rebuilding a page does not decode
	// them again. It must only be set when [Context.GetURL] returns the
	// same content for the same URL until the cache is reset. It is nil
	// by default; see [NewImageCache].
	ImageCache *ImageCache

	// StageSize, if positive, is the size of the content of the body (in
	// runes of text, with estimates for other elements) after which
	// [ReadHTML] stops adding the widgets of its top-level elements,
	// leaving the rest for [Context.ReadRest]. This allows the first part
	// of a long page to be shown sooner.
	StageSize int

	// readState is the state of the current read of HTML.
	readState

	// reading is whether HTML is being read, so that HTML read by an
	// element handler meanwhile is read within that read (see [ReadHTML]).
	reading bool
}

// readState is the state of a read of HTML by [ReadHTML].
type readState struct {

	// firstRow indicates the start of a table, where number of columns is counted.
	firstRow bool

	// stageSize is the size of the content of the body read so far.
	stageSize int

	// rest is the first top-level element of the body that has not been read,
	// and restParent the widget that its widgets are added to.
	rest       *html.Node
	restParent core.Widget
}

// HasRest returns whether [ReadHTML] has left elements to read with
// [Context.ReadRest] (see [Context.StageSize]).
func (c *Context) HasRest() bool {
	return c.rest != nil
}

// ReadRest adds the widgets of the elements that [ReadHTML] left (see
// [Context.StageSize]), and returns the widget that they are added to
// and the index of the first child added, which need to be updated.
// The context must not have been used to read other HTML since.
func (c *Context) ReadRest() (core.Widget, int, error) {
	if c.rest == nil {
		return nil, 0, nil
	}
	n, parent := c.rest, c.restParent
	c.rest, c.restParent = nil, nil
	c.StageSize = 0
	start := parent.AsTree().NumChildren()
	c.reading = true
	defer func() { c.reading = false }()
	err := readHTMLNode(c, parent, n)
	return parent, start, err
}

// stageAt returns whether the given node is a top-level element of the body
// that starts the rest of a staged body (see [Context.StageSize]), adding
// its size to the size of the body read so far otherwise. The rest starts at
// a block element, whose widgets are new children of the body.
func (c *Context) stageAt(n *html.Node) bool {
	if c.StageSize <= 0 || n.Parent == nil || n.Parent.DataAtom != atom.Body {
		return false
	}
	if c.stageSize >= c.StageSize && n.Type == html.ElementNode && stageBlocks[n.DataAtom] {
		return true
	}
	c.stageSize += nodeSize(n)
	return false
}

// stageBlocks are the block elements that the rest of a staged body can start at.
var stageBlocks = map[atom.Atom]bool{
	atom.P: true, atom.H1: true, atom.H2: true, atom.H3: true, atom.H4: true, atom.H5: true, atom.H6: true,
	atom.Ul: true, atom.Ol: true, atom.Pre: true, atom.Table: true, atom.Div: true, atom.Blockquote: true,
	atom.Hr: true, atom.Section: true, atom.Figure: true, atom.Details: true, atom.Dl: true,
}

// nodeSize returns an estimate of the size of the content of the node, in
// runes of text, with fixed sizes for elements, larger for media.
func nodeSize(n *html.Node) int {
	size := 0
	switch n.Type {
	case html.TextNode:
		size = len(strings.TrimSpace(n.Data))
	case html.ElementNode:
		size = 40
		switch n.DataAtom {
		case atom.Img, atom.Video, atom.Iframe, atom.Svg, atom.Embed, atom.Object, atom.Canvas:
			size = 2000
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		size += nodeSize(c)
	}
	return size
}

// NewContext returns a new [Context] with basic defaults.
func NewContext() *Context {
	return &Context{
		styles:            map[*html.Node][]*css.Rule{},
		OpenURL:           func(url string, e events.Event) { system.TheApp.OpenURL(url) },
		GetURL:            http.Get,
		ElementHandlers:   map[string]func(ctx *Context) bool{},
		AttributeHandlers: map[string]func(ctx *Context, w io.Writer, node ast.Node, entering bool, tag, value string) bool{},
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
			wb.SetProperty("id", attr.Val)
		case "style":
			c.SetStyleAttr(c.Node, attr.Val)
		default:
			wb.SetProperty(attr.Key, attr.Val)
		}
	}
	wb.SetProperty("tag", c.Node.Data) // this is needed by handleWidget in general
}

func (c *Context) SetStyleAttr(node *html.Node, style string) error {
	// our CSS parser is strict about semicolons, but
	// they aren't needed in normal inline styles in HTML
	if !strings.HasSuffix(style, ";") {
		style += ";"
	}
	decls, err := parser.ParseDeclarations(style)
	if errors.Log(err) != nil {
		return err
	}
	rule := &css.Rule{Declarations: decls}
	if c.styles == nil {
		c.styles = map[*html.Node][]*css.Rule{}
	}
	c.styles[c.Node] = append(c.styles[c.Node], rule)
	return nil
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
		for _, match := range selectRule(root, rule) {
			c.styles[match] = append(c.styles[match], rule)
		}
	}
}

// selectRule returns the nodes under root that the selectors of the rule
// select, once for each selector that selects them.
func selectRule(root *html.Node, rule *css.Rule) []*html.Node {
	// Selectors of element names, such as the user agent styles have, select
	// the elements with those names, which one walk of the tree finds; the
	// selector library walks the tree for each selector.
	if names := selectorAtoms(rule.Selectors); names != nil {
		var matches []*html.Node
		var walk func(n *html.Node)
		walk = func(n *html.Node) {
			for range names[n.DataAtom] {
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

	// Other selectors are matched by the selector library.
	sel := &selcss.Selector{}
	if len(rule.Selectors) > 0 {
		s, err := selcss.Parse(strings.Join(rule.Selectors, ","))
		if errors.Log(err) == nil {
			sel = s
		}
	}
	return sel.Select(root)
}

// selectorAtoms returns the number of selectors of each element name, when
// all of the selectors are lowercase element names, and otherwise nil.
func selectorAtoms(selectors []string) map[atom.Atom]int {
	if len(selectors) == 0 {
		return nil
	}
	names := make(map[atom.Atom]int, len(selectors))
	for _, s := range selectors {
		s = strings.TrimSpace(s)
		a := atom.Lookup([]byte(s))
		if a == 0 || a.String() != s {
			return nil
		}
		names[a]++
	}
	return names
}

// LinkButton is a helper function that makes the given button
// open the given link when clicked on, using [Context.OpenURL].
// The advantage of using this is that it does [tree.NodeBase.SetProperty]
// of "href" to the given url, allowing generatehtml to create an <a> element
// for HTML preview and SEO purposes. It also sets the tooltip to the URL.
//
// See also [Context.LinkButtonUpdating] for a dynamic version.
func (c *Context) LinkButton(bt *core.Button, url string) *core.Button {
	bt.SetProperty("tag", "a")
	bt.SetProperty("href", url)
	bt.SetTooltip(url)
	bt.OnClick(func(e events.Event) {
		c.OpenURL(url, e)
	})
	return bt
}

// LinkButtonUpdating is a version of [Context.LinkButton] that is robust to a changing/dynamic
// URL, using an Updater and a URL function instead of a variable.
func (c *Context) LinkButtonUpdating(bt *core.Button, url func() string) *core.Button {
	bt.SetProperty("tag", "a")
	bt.Updater(func() {
		u := url()
		bt.SetProperty("href", u)
		bt.SetTooltip(u)
	})
	bt.OnClick(func(e events.Event) {
		c.OpenURL(url(), e)
	})
	return bt
}

// AddWidgetHandler adds given widget handler function
func (c *Context) AddWidgetHandler(f func(w core.Widget)) {
	c.WidgetHandlers = append(c.WidgetHandlers, f)
}

func (c *Context) applyStyleRules(node *html.Node, w core.Widget) {
	wb := w.AsWidget()
	rules := c.styles[c.Node]
	wb.Styler(func(s *styles.Style) {
		for _, rule := range rules {
			for _, decl := range rule.Declarations {
				if strings.HasPrefix(decl.Property, "paginate-") {
					wb.SetProperty(decl.Property, decl.Value)
					continue
				}
				// TODO(kai/styproperties): parent style and context
				s.FromProperty(s, decl.Property, decl.Value, colors.BaseContext(colors.ToUniform(s.Color)))
			}
		}
	})
}

// handleWidget applies accumulated style rules,
// and calls WidgetHandlers functions on given widget,
// in order added so last one has override priority.
func (c *Context) handleWidget(node *html.Node, w core.Widget) {
	c.applyStyleRules(node, w)
	for _, f := range c.WidgetHandlers {
		f(w)
	}
}
