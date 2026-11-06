// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
	"github.com/gomarkdown/markdown/ast"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Context contains context information about the current state of a htmlcore
// reader and its surrounding context. It should be created with [NewContext].
type Context struct {

	// Node is the node that is currently being read.
	Node *html.Node

	// RefreshURL is the URL of the first meta element with an http-equiv
	// of refresh in the HTML read, if any, which an app that shows web pages
	// can open next, as browsers do (as for a page that redirects browsers
	// without JavaScript to another one). It must be reset before each read.
	RefreshURL string

	// Style is CSS that is added to the style sheets of each page that is
	// read, before those of the page (as a user
	// style sheet of a browser), as for the margins of an app's pages.
	Style string

	// Title is the text of the first title element in the HTML read,
	// if any, as for the title of a window or tab that shows the page.
	Title string

	// MediaWidth, if positive, is the width of the viewport in CSS px (dp)
	// that the @media rules of style sheets are for; otherwise, it is the
	// width of the screen.
	MediaWidth float32

	// MediaHeight, if positive, is the height of the viewport in CSS px (dp)
	// that vh lengths are relative to; otherwise, they are relative to the
	// height of the scene.
	MediaHeight float32

	// Zoom, if positive, is the zoom factor of the page, as in browsers:
	// all of its sizes are multiplied by it (see
	// [styles.Style.Zoom]). MediaWidth and MediaHeight are then the size of
	// the viewport divided by it, as its CSS px are larger (or smaller).
	Zoom float32

	// WebFonts is whether the web fonts of the @font-face rules of the
	// style sheets of pages are loaded (once for each URL, for all pages),
	// and used for the font families that they are for (see
	// [fonts.AddWebFont]). Fonts in the WOFF2 format need a [FontDecoder].
	WebFonts bool

	// Scripting is whether the scripts of the pages are run (by the app
	// that reads them, as htmlcore does not run them): the HTML is then
	// parsed with scripting enabled (see [ParseHTML]), and noscript
	// elements are not displayed, as in browsers with JavaScript.
	Scripting bool

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

	// PostURL, if not nil, is the function used to open URLs with a POST
	// request with the given body of the given content type, as for
	// submitting forms with the POST method. If it is nil, those forms
	// can not be submitted.
	PostURL func(url, contentType string, body []byte, e events.Event)

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

	// pageState is the state of the pages read with the context: their
	// style sheets and the state of their widgets.
	pageState

	// readState is the state of the current read of HTML.
	readState

	// reading is whether HTML is being read, so that HTML read by an
	// element handler meanwhile is read within that read (see [ReadHTML]).
	reading bool
}

// pageState is the state of the pages read with a [Context], which
// [Context.NewPage] starts anew: the style sheets and declarations of
// their elements, and the state of their widgets for styling them.
type pageState struct {
	// selIndex is the index of the nodes of the document for selecting
	// the nodes of style rules (see [Context.selectNodes]).
	selIndex *selectorIndex

	// styles are the CSS declarations for each node, which are
	// sorted in the order of the cascade by [Context.declarations].
	styles map[*html.Node][]*cssDecl

	// sorted records the nodes whose declarations are sorted,
	// and hinted those whose attributes have been added.
	sorted, hinted map[*html.Node]bool

	// cssOrder is the order of the last declaration added.
	cssOrder int

	// cssBase is the URL of the style sheet being added, if it is from a
	// URL, which the URLs in it are relative to, and cssImports has the
	// URLs of the style sheets that have been added (for cycles of @import).
	cssBase    string
	cssImports map[string]bool

	// nodeWidgets has the widgets of the controls of forms.
	nodeWidgets map[*html.Node]core.Widget

	// floatSiblings has whether the unstyled blocks in each element
	// have floats in them, for flattensBlock.
	floatSiblings map[*html.Node]bool

	// linkFrames has the URLs of the links that are frames (blocks), by
	// their nodes, for the widgets in them (see linkFrame).
	linkFrames map[*html.Node]string

	// flattened has the blocks that are flattened (see flattensBlock).
	flattened map[*html.Node]bool

	// rootFont is the font size of the html element in px, once it is known
	// (see rootFontSize).
	rootFont float32

	// blocks has the margin collapsing state of the block widgets.
	blocks map[core.Widget]*blockState

	// mediaUses has the media query lists evaluated in reading the pages,
	// with their results, and columnUses the numbers of columns estimated
	// from the width of the viewport, for [Context.MediaChanged], while
	// checkingMedia is whether it is evaluating them for another width.
	mediaUses     map[string]bool
	columnUses    map[*html.Node]int
	checkingMedia bool
}

// readState is the state of a read of HTML by [ReadHTML].
type readState struct {

	// tables has the widget of each table element.
	tables map[*html.Node]*core.Frame

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
		pageState:         pageState{styles: map[*html.Node][]*cssDecl{}},
		OpenURL:           func(url string, e events.Event) { system.TheApp.OpenURL(url) },
		GetURL:            getURL,
		ElementHandlers:   map[string]func(ctx *Context) bool{},
		AttributeHandlers: map[string]func(ctx *Context, w io.Writer, node ast.Node, entering bool, tag, value string) bool{},
	}
}

// Parent returns the current parent widget that a widget
// associated with the current node should be added to.
// It may make changes to the widget tree, so the widget
// must be added to the resulting parent immediately.
func (c *Context) Parent() core.Widget {
	display := c.displayOf(c.Node)
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
		case "style": // in the cascade
		default:
			wb.SetProperty(attr.Key, attr.Val)
		}
	}
	wb.SetProperty("tag", c.Node.Data) // this is needed by handleWidget in general
	wb.SetProperty("node", c.Node)     // for the grid-area of a grid item
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
	c.addStyleOrigin(style, originAuthor)
}

// addStyleOrigin adds the given CSS style string with the given origin.
func (c *Context) addStyleOrigin(style string, origin cssOrigin) {
	c.addRules(rootNode(c.Node), parseCSS(style), origin)
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
	if node == nil || node.Type != html.ElementNode {
		node = nil // text nodes only inherit
	}
	wb := w.AsWidget()
	decls := c.declarations(node)
	for _, d := range decls {
		if strings.HasPrefix(d.Property, "paginate-") {
			wb.SetProperty(d.Property, d.Value)
		}
	}
	c.browserStyler(node, w, decls)
}

// handleWidget applies accumulated style rules,
// and calls WidgetHandlers functions on given widget,
// in order added so last one has override priority.
func (c *Context) handleWidget(node *html.Node, w core.Widget) {
	c.applyStyleRules(node, w)
	c.handleFormWidget(node, w)
	c.handleInLink(node, w)
	for _, f := range c.WidgetHandlers {
		f(w)
	}
}

// NewPage returns a new context for reading a new page, with the settings
// of this context (as for media, styles, URLs, and handlers), but none of
// the state from reading a page (as its style sheets). A context that is
// used to read pages one after another (as in a browser) should get a new
// one for each page with this, so that nothing of one page affects the next.
func (c *Context) NewPage() *Context {
	n := NewContext()
	n.MediaWidth, n.MediaHeight, n.Zoom = c.MediaWidth, c.MediaHeight, c.Zoom
	n.Scripting, n.WebFonts = c.Scripting, c.WebFonts
	n.PageURL = c.PageURL
	n.OpenURL, n.GetURL, n.PostURL = c.OpenURL, c.GetURL, c.PostURL
	n.ElementHandlers = c.ElementHandlers
	n.WikilinkHandlers = c.WikilinkHandlers
	n.AttributeHandlers = c.AttributeHandlers
	n.WidgetHandlers = c.WidgetHandlers
	n.DelayedImageLoad = c.DelayedImageLoad
	n.ImageCache = c.ImageCache
	n.StageSize = c.StageSize
	n.Style = c.Style
	return n
}

// UserAgent is the User-Agent header of the requests of the default
// [Context.GetURL] (see [SetRequestHeaders]). It has the form of the user
// agents of browsers, for the platform that the app runs on, naming
// Cogent Core: sites treat the "compatible; ...; +url" form of crawlers
// (and the default of the Go HTTP client) as bots. Scripts should report
// the same value (navigator.userAgent), since sites check that they match.
var UserAgent = "Mozilla/5.0 (" + userAgentPlatform() + ") AppleWebKit/537.36 (KHTML, like Gecko) CogentCore/1.0 Safari/537.36"

// userAgentPlatform returns the platform part of the [UserAgent], as
// browsers report it for the platform that the app runs on.
func userAgentPlatform() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows NT 10.0; Win64; x64"
	case "linux":
		return "X11; Linux x86_64"
	case "android":
		return "Linux; Android 10; K"
	case "ios":
		return "iPhone; CPU iPhone OS 17_0 like Mac OS X"
	default:
		return "Macintosh; Intel Mac OS X 10_15_7"
	}
}

// AcceptLanguage is the Accept-Language header of the requests of the
// default [Context.GetURL] (see [SetRequestHeaders]), from the locale of
// the system (the LC_ALL, LC_MESSAGES, or LANG environment variable) or
// US English. Some sites send reduced pages to requests without it.
var AcceptLanguage = acceptLanguage(firstEnv("LC_ALL", "LC_MESSAGES", "LANG"))

// firstEnv returns the value of the first of the given environment
// variables that is set.
func firstEnv(names ...string) string {
	for _, n := range names {
		if v := os.Getenv(n); v != "" {
			return v
		}
	}
	return ""
}

// acceptLanguage returns the Accept-Language header for the given POSIX
// locale (as "fr_CA.UTF-8"): "fr-CA,fr;q=0.9,en;q=0.8". It is US English
// for the C and POSIX locales and for an empty one.
func acceptLanguage(locale string) string {
	loc, _, _ := strings.Cut(locale, ".")
	loc, _, _ = strings.Cut(loc, "@")
	if loc == "" || loc == "C" || loc == "POSIX" {
		return "en-US,en;q=0.9"
	}
	tag := strings.ReplaceAll(loc, "_", "-")
	lang, _, _ := strings.Cut(tag, "-")
	h := tag
	if lang != tag {
		h += "," + lang + ";q=0.9"
	}
	if lang != "en" {
		h += ",en;q=0.8"
	}
	return h
}

// acceptHeader is the Accept header of requests, as browsers send it for
// pages (and accepted for their resources).
const acceptHeader = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8"

// SetRequestHeaders sets the headers of the given request that browsers
// send: the [UserAgent], [AcceptLanguage], and Accept. (Accept-Encoding is
// set by the Go HTTP client, which decompresses the gzip responses.)
func SetRequestHeaders(req *http.Request) {
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", acceptHeader)
	req.Header.Set("Accept-Language", AcceptLanguage)
}

// httpClient is the client of the default [Context.GetURL], with a timeout
// so that a server that does not respond does not stop the reading of a
// page (as a style sheet or image of the page).
var httpClient = &http.Client{Timeout: 15 * time.Second}

// getURL gets the given URL with an HTTP GET request, with the headers of
// [SetRequestHeaders].
func getURL(url string) (*http.Response, error) {
	if strings.HasPrefix(url, "file:") {
		return getFileURL(url)
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	SetRequestHeaders(req)
	return httpClient.Do(req)
}

// getFileURL gets the local file at the given file: URL, as a response
// with the status of a server: 404 Not Found if it does not exist.
func getFileURL(rawURL string) (*http.Response, error) {
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	path := u.Path
	if runtime.GOOS == "windows" {
		path = strings.TrimPrefix(path, "/") // file:///C:/dir
	}
	f, err := os.Open(path)
	if err != nil {
		return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found",
			Body: io.NopCloser(strings.NewReader("")), Request: &http.Request{URL: u}}, nil
	}
	if fi, err := f.Stat(); err == nil && fi.IsDir() { // its index.html, as servers do
		f.Close()
		if f, err = os.Open(filepath.Join(path, "index.html")); err != nil {
			return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found",
				Body: io.NopCloser(strings.NewReader("")), Request: &http.Request{URL: u}}, nil
		}
	}
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: f, Request: &http.Request{URL: u}}, nil
}

// TextElement returns the widget of the HTML element that the given text
// is the text of: the text itself if it has a "tag" property, or else the
// widget with a tag that it is in (possibly in an inline container of it),
// as the text of a paragraph or heading is in the frame of its element.
func TextElement(tx *core.Text) *core.WidgetBase {
	if _, ok := tx.Property("tag").(string); ok {
		return tx.AsWidget()
	}
	pw := core.AsWidget(tx.Parent)
	if pw != nil && strings.HasPrefix(pw.Name, "inline-container") {
		pw = core.AsWidget(pw.Parent)
	}
	if pw != nil {
		if _, ok := pw.Property("tag").(string); ok {
			return pw
		}
	}
	return tx.AsWidget()
}

// TextTag returns the name of the HTML element that the given text is the
// text of (see [TextElement]), such as "h2" for the text of a heading,
// or "" if there is none.
func TextTag(tx *core.Text) string {
	// note: checking ok, as a failed assertion of a nil value to a
	// string is null instead of "" under goscript (js)
	if tag, ok := TextElement(tx).Property("tag").(string); ok {
		return tag
	}
	return ""
}

// TagTexts gives each text in the given widget the "tag" and "id"
// properties of the HTML element that it is the text of (see
// [TextElement]), if it does not have its own, as for printing it apart
// from the widget of its element (as in a PDF, which styles headings by
// their tags, and links to their ids).
func TagTexts(w core.Widget) {
	w.AsWidget().WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		tx, ok := cw.(*core.Text)
		if !ok {
			return tree.Continue
		}
		if el := TextElement(tx); el != cwb {
			for _, k := range []string{"tag", "id"} {
				if v, ok := el.Property(k).(string); ok {
					cwb.SetProperty(k, v)
				}
			}
		}
		return tree.Continue
	})
}

// NodeOf returns the HTML node of the element that the given widget is
// for, or that of the nearest widget that it is in that has one (as for
// the text of an element), or nil if there is none. The nodes are those
// of the document that was read (see [ReadDocument]).
func NodeOf(w core.Widget) *html.Node {
	for wb := w.AsWidget(); wb != nil; wb = core.AsWidget(wb.Parent) {
		if n, ok := wb.Property("node").(*html.Node); ok && n != nil {
			return n
		}
	}
	return nil
}
