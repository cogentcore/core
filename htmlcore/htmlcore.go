// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package htmlcore converts HTML and Markdown into Cogent Core widgets.
//
// It is the API for apps and content sites: make a [Context] with
// [NewContext], set it up with its handlers and settings, and read HTML or
// Markdown into a parent widget with [ReadHTML], [ReadHTMLString], [ReadMD]
// or [ReadMDString].
//
// The engine that does the reading (the CSS cascade, layout of the
// elements, re-reading and restyling changed documents) is package
// [htmlengine], which apps that show web pages, as a browser does, use
// directly.
package htmlcore

import (
	"io"
	"io/fs"
	"net/http"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore/htmlengine"
	"github.com/yuin/goldmark/v2/ast"
	"golang.org/x/net/html"
)

// Context is the context for reading HTML and Markdown into widgets: its
// settings, its handlers of elements, attributes, wikilinks and widgets,
// and, in the handlers, the state of the current read (the current
// [Context.Node] and the parents that its widgets are added to).
type Context struct {
	// e is the engine context that reads.
	e *htmlengine.Context
}

// NewContext returns a new [Context] with basic defaults.
func NewContext() *Context {
	return &Context{e: htmlengine.NewContext()}
}

// wrap returns the Context for the given engine context, as for the
// handlers, which are called with the engine context that reads.
func wrap(e *htmlengine.Context) *Context {
	return &Context{e: e}
}

// Node returns the node that is currently being read.
func (c *Context) Node() *html.Node { return c.e.Node }

// SetNode sets the node that is currently being read, as for
// [ExtractText] of another node in an element handler.
func (c *Context) SetNode(n *html.Node) *Context {
	c.e.Node = n
	return c
}

// Parent returns the current parent widget that a widget associated with
// the current node should be added to. It may make changes to the widget
// tree, so the widget must be added to the resulting parent immediately.
func (c *Context) Parent() core.Widget { return c.e.Parent() }

// BlockParent returns the current parent widget that block-level
// (non-inline) elements are added to, as for the widgets that an element
// handler makes.
func (c *Context) BlockParent() core.Widget { return c.e.BlockParent }

// PageURL returns the URL of the current page, which relative URLs are
// relative to, or "" if there is none.
func (c *Context) PageURL() string { return c.e.PageURL }

// SetPageURL sets the URL of the current page (see [Context.PageURL]).
func (c *Context) SetPageURL(url string) *Context {
	c.e.PageURL = url
	return c
}

// SetOpenURL sets the function used to open URLs, as of links, which
// defaults to [system.App.OpenURL].
func (c *Context) SetOpenURL(f func(url string, e events.Event)) *Context {
	c.e.OpenURL = f
	return c
}

// GetURL returns the function used to get resources from URLs (see
// [Context.SetGetURL]), as for one that gets some URLs itself and the
// others with it.
func (c *Context) GetURL() func(url string) (*http.Response, error) { return c.e.GetURL }

// SetGetURL sets the function used to get resources from URLs, as images
// and style sheets, which defaults to [http.Get].
func (c *Context) SetGetURL(f func(url string) (*http.Response, error)) *Context {
	c.e.GetURL = f
	return c
}

// SetStyle sets CSS that is added to the style sheets of each page that is
// read, before those of the page, as for the margins of the pages of an app.
func (c *Context) SetStyle(style string) *Context {
	c.e.Style = style
	return c
}

// SetDelayedImageLoad sets whether images are loaded with a delay,
// rather than immediately.
func (c *Context) SetDelayedImageLoad(delayed bool) *Context {
	c.e.DelayedImageLoad = delayed
	return c
}

// SetStageSize sets the size of the content of a page (in runes of text,
// with estimates for other elements) after which [ReadHTML] stops adding
// the widgets of its top-level elements, leaving the rest for
// [Context.ReadRest], so that the first part of a long page is shown
// sooner. 0 reads all of it.
func (c *Context) SetStageSize(size int) *Context {
	c.e.StageSize = size
	return c
}

// ImageCache returns the cache of the decoded images, if any.
func (c *Context) ImageCache() *ImageCache { return c.e.ImageCache }

// SetImageCache sets the cache of the decoded images of img elements, by
// their URL, so that reading a page again does not decode them again (see
// [NewImageCache]). It must only be set when the GetURL function returns
// the same content for the same URL until the cache is reset.
func (c *Context) SetImageCache(ic *ImageCache) *Context {
	c.e.ImageCache = ic
	return c
}

// AddElementHandler adds a handler of the elements with the given tag
// (as for custom elements), which makes their widgets, as in the
// [Context.BlockParent], in place of those that would be made for them. If
// it returns false, the default widgets of the element are made.
func (c *Context) AddElementHandler(tag string, f func(ctx *Context) bool) *Context {
	c.e.ElementHandlers[tag] = func(e *htmlengine.Context) bool { return f(wrap(e)) }
	return c
}

// AddAttributeHandler adds a handler of the given tag of the custom
// {tag: value} attributes of Markdown elements, which is called with the
// Markdown node as it is converted to HTML. It can write alternative or
// additional HTML to the given writer; if it returns true, the default HTML
// of the node is not written.
func (c *Context) AddAttributeHandler(tag string, f func(ctx *Context, w io.Writer, node ast.Node, entering bool, tag, value string) bool) *Context {
	c.e.AttributeHandlers[tag] = func(e *htmlengine.Context, w io.Writer, node ast.Node, entering bool, tag, value string) bool {
		return f(wrap(e), w, node, entering, tag, value)
	}
	return c
}

// AddWikilinkHandler adds a handler of wikilinks. If it returns "", "",
// the handlers added before it are tried.
func (c *Context) AddWikilinkHandler(h WikilinkHandler) *Context {
	c.e.AddWikilinkHandler(h)
	return c
}

// AddWidgetHandler adds a function that is called for each widget that is
// made, after it is configured, as for styling widgets of a type.
func (c *Context) AddWidgetHandler(f func(w core.Widget)) *Context {
	c.e.AddWidgetHandler(f)
	return c
}

// HasRest returns whether [ReadHTML] has left elements to read with
// [Context.ReadRest] (see [Context.SetStageSize]).
func (c *Context) HasRest() bool { return c.e.HasRest() }

// ReadRest adds the widgets of the elements that [ReadHTML] left (see
// [Context.SetStageSize]), and returns the widget that they are added to
// and the index of the first child added, which need to be updated.
// The context must not have been used to read other HTML since.
func (c *Context) ReadRest() (core.Widget, int, error) { return c.e.ReadRest() }

// LinkButton makes the given button open the given URL when it is
// clicked, with the OpenURL function of the context, and sets its
// tooltip to the URL. The URL is also the "href" property of the button,
// so that the HTML generated for the app has a link for it.
func (c *Context) LinkButton(bt *core.Button, url string) *core.Button {
	return c.e.LinkButton(bt, url)
}

// LinkButtonUpdating is a version of [Context.LinkButton] for a URL that
// changes, from the given function.
func (c *Context) LinkButtonUpdating(bt *core.Button, url func() string) *core.Button {
	return c.e.LinkButtonUpdating(bt, url)
}

// ReadHTML reads HTML from the given reader and adds the widgets of its
// elements to the given parent widget.
func ReadHTML(ctx *Context, parent core.Widget, r io.Reader) error {
	return htmlengine.ReadHTML(ctx.e, parent, r)
}

// ReadHTMLString reads HTML from the given string and adds the widgets of
// its elements to the given parent widget.
func ReadHTMLString(ctx *Context, parent core.Widget, s string) error {
	return htmlengine.ReadHTMLString(ctx.e, parent, s)
}

// ParseHTML parses HTML from the given reader into a document of nodes,
// as browsers parse it, which [ReadDocument] reads.
func ParseHTML(ctx *Context, r io.Reader) (*html.Node, error) {
	return htmlengine.ParseHTML(ctx.e, r)
}

// ReadDocument adds the widgets of the elements of the given parsed
// document (see [ParseHTML]) to the given parent widget.
func ReadDocument(ctx *Context, parent core.Widget, n *html.Node) error {
	return htmlengine.ReadDocument(ctx.e, parent, n)
}

// ReadMD reads Markdown from the given bytes and adds the widgets of its
// elements to the given parent widget.
func ReadMD(ctx *Context, parent core.Widget, b []byte) error {
	return htmlengine.ReadMD(ctx.e, parent, b)
}

// ReadMDString reads Markdown from the given string and adds the widgets
// of its elements to the given parent widget.
func ReadMDString(ctx *Context, parent core.Widget, s string) error {
	return htmlengine.ReadMDString(ctx.e, parent, s)
}

// MDToHTML converts the given Markdown to HTML, with the attribute and
// wikilink handlers of the context.
func MDToHTML(ctx *Context, md []byte) []byte {
	return htmlengine.MDToHTML(ctx.e, md)
}

// ExtractText returns the text of the children of the current node of the
// context (see [Context.SetNode]), with inline markup for formatted text.
func ExtractText(ctx *Context) string {
	return htmlengine.ExtractText(ctx.e)
}

// WikilinkHandler is a function that converts the text of a wikilink
// ([[text]]) to the URL and label of a link, or returns "", "" if it does
// not handle it.
type WikilinkHandler = htmlengine.WikilinkHandler

// GoDocWikilink returns a [WikilinkHandler] that converts wikilinks of the
// form [[prefix:identifier]] to a pkg.go.dev URL starting at pkg. For
// example, with prefix="doc" and pkg="cogentcore.org/core", the wikilink
// [[doc:core.Button]] links to https://pkg.go.dev/cogentcore.org/core/core#Button.
func GoDocWikilink(prefix string, pkg string) WikilinkHandler {
	return htmlengine.GoDocWikilink(prefix, pkg)
}

// ImageCache caches the decoded images of img elements by their URL
// (see [Context.SetImageCache]).
type ImageCache = htmlengine.ImageCache

// NewImageCache returns a new [ImageCache] of at most the given number of
// bytes of decoded images.
func NewImageCache(maxBytes int) *ImageCache {
	return htmlengine.NewImageCache(maxBytes)
}

// GetURLFromFS returns the response for the given URL from the given
// filesystem, as for a GetURL function of the files of an app.
func GetURLFromFS(fsys fs.FS, rawURL string) (*http.Response, error) {
	return htmlengine.GetURLFromFS(fsys, rawURL)
}

// NodeOf returns the HTML node of the element that the given widget is
// for, or that of the nearest widget that it is in that has one (as for the
// text of an element), or nil if there is none.
func NodeOf(w core.Widget) *html.Node { return htmlengine.NodeOf(w) }

// GetAttr returns the value of the given attribute of the given node,
// or "" if it has none.
func GetAttr(n *html.Node, attr string) string { return htmlengine.GetAttr(n, attr) }

// HasAttr returns whether the given node has the given attribute.
func HasAttr(n *html.Node, attr string) bool { return htmlengine.HasAttr(n, attr) }

// GetLanguage returns the 'x' in a `language-x` class from the given
// string of class(es).
func GetLanguage(class string) string { return htmlengine.GetLanguage(class) }

// MDGetAttr returns the value of the given attribute of the given Markdown
// node, or "" if it has none.
func MDGetAttr(n ast.Node, attr string) string { return htmlengine.MDGetAttr(n, attr) }

// MDSetAttr sets the given attribute of the given Markdown node.
func MDSetAttr(n ast.Node, attr, value string) { htmlengine.MDSetAttr(n, attr, value) }

// TextTag returns the name of the HTML element that the given text is the
// text of, such as "h2" for the text of a heading, or "" if there is none.
func TextTag(tx *core.Text) string { return htmlengine.TextTag(tx) }

// TagTexts gives each text in the given widget the "tag" and "id"
// properties of the HTML element that it is the text of, if it does not
// have its own, as for printing it apart from the widget of its element (as
// in a PDF, which styles headings by their tags, and links to their ids).
func TagTexts(w core.Widget) { htmlengine.TagTexts(w) }
