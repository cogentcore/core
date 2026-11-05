// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"io"
	"log/slog"
	"net/http"
	neturl "net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/tree"
	"golang.org/x/net/html"
)

// New adds a new widget of the given type to the context parent.
// It automatically calls [Context.config] on the resulting widget.
func New[T tree.NodeValue](ctx *Context) *T {
	parent := ctx.Parent()
	w := tree.New[T](parent)
	ctx.config(any(w).(core.Widget))
	return w
}

// handleElement calls the handler in [Context.ElementHandlers] associated with the current node
// using the given context. If there is no handler associated with it, it uses default
// hardcoded configuration code.
func handleElement(ctx *Context) {
	tag := ctx.Node.Data
	h, ok := ctx.ElementHandlers[tag]
	if ok {
		if h(ctx) {
			return
		}
	}

	if handleBrowserElement(ctx, tag) {
		return
	}

	var newWidget core.Widget

	switch tag {
	case "script", "link", "style":
		// nothing is rendered, and the style sheets are added first
	case "title":
		// it is not rendered, but it is the title of the page (the first one)
		if ctx.Title == "" {
			ctx.Title = strings.Join(strings.Fields(ExtractText(ctx)), " ")
		}
	case "meta":
		if strings.EqualFold(GetAttr(ctx.Node, "http-equiv"), "refresh") && ctx.RefreshURL == "" {
			// content is "delay" or "delay;url=URL"
			_, u, _ := strings.Cut(GetAttr(ctx.Node, "content"), ";")
			u = strings.TrimSpace(u)
			if len(u) > 4 && strings.EqualFold(u[:4], "url=") {
				ctx.RefreshURL = strings.Trim(strings.TrimSpace(u[4:]), `'"`)
			}
		}
	case "img":
		newWidget = handleImage(ctx, tag)
	case "input":
		newWidget = handleInput(ctx)
	case "textarea":
		buf := lines.NewLines()
		buf.SetText([]byte(ExtractText(ctx)))
		newWidget = New[textcore.Editor](ctx).SetLines(buf)
	case "table":
		newWidget = handleTable(ctx)
	case "th", "td":
		newWidget = handleCell(ctx, tag)
	case "thead", "tbody", "tfoot", "tr":
		ctx.NewParent = ctx.tableFor(ctx.Node)
	default:
		switch display := ctx.displayOf(ctx.Node); {
		case display == "none":
			// not rendered, and neither are its children
		case display == "contents" && tag != "body":
			// it has no box, and its children are in its parent
			ctx.NewParent = ctx.BlockParent
			ctx.inlineParent = nil
		case display == "block" && ctx.flattensBlock(ctx.Node):
			// its children are in its parent, as it has no box of its own
			ctx.NewParent = ctx.BlockParent
			ctx.inlineParent = nil
		case isBlockDisplay(display) || tag == "body" || strings.HasPrefix(display, "inline-"):
			w := New[core.Frame](ctx)
			newWidget = w
			ctx.NewParent = w
		default:
			ctx.NewParent = ctx.Parent()
		}
	}
	if newWidget != nil {
		ctx.handleWidget(ctx.Node, newWidget)
	}
}

// handleInput handles an input element.
func handleInput(ctx *Context) core.Widget {
	var newWidget core.Widget
	{
		ityp := GetAttr(ctx.Node, "type")
		val := GetAttr(ctx.Node, "value")
		switch ityp {
		case "number":
			fval := float32(errors.Log1(strconv.ParseFloat(val, 32)))
			newWidget = New[core.Spinner](ctx).SetValue(fval)
		case "checkbox":
			newWidget = New[core.Switch](ctx).SetType(core.SwitchCheckbox).
				SetState(HasAttr(ctx.Node, "checked"), states.Checked)
		case "radio":
			newWidget = New[core.Switch](ctx).SetType(core.SwitchRadioButton).
				SetState(HasAttr(ctx.Node, "checked"), states.Checked)
		case "range":
			fval := float32(errors.Log1(strconv.ParseFloat(val, 32)))
			newWidget = New[core.Slider](ctx).SetValue(fval)
		case "button", "submit":
			newWidget = New[core.Button](ctx).SetText(val)
		case "color":
			newWidget = core.Bind(val, New[core.ColorButton](ctx))
		case "datetime":
			newWidget = core.Bind(val, New[core.TimeInput](ctx))
		case "file":
			newWidget = core.Bind(val, New[core.FileButton](ctx))
		default:
			newWidget = New[core.TextField](ctx).SetText(val)
		}
	}
	return newWidget
}

// handleTable handles a table element.
func handleTable(ctx *Context) core.Widget {
	w := New[core.Frame](ctx)
	ctx.NewParent = w
	if ctx.tables == nil {
		ctx.tables = map[*html.Node]*core.Frame{}
	}
	ctx.tables[ctx.Node] = w
	ctx.TableParent = w
	cols := ctx.tableColumns(ctx.Node)
	w.SetProperty("cols", cols)
	if ctx.declaration(ctx.Node, "width") == "" && ctx.hasPercentColumns(ctx.Node) {
		// An auto-width table with percentage columns is as wide as it needs
		// to be to give them their shares, which is approximated by filling
		// the width that it has (in which they grow up to their shares).
		w.FinalStyler(func(s *styles.Style) { s.Grow.X = 1 })
	}
	w.Styler(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = cols
	})
	return w
}

// tableFor returns the widget of the table that the given node is in.
func (c *Context) tableFor(n *html.Node) core.Widget {
	if t := ancestorTable(n); t != nil {
		if w := c.tables[t]; w != nil {
			return w
		}
	}
	if c.TableParent != nil {
		return c.TableParent
	}
	return c.Parent()
}

// hasPercentColumns returns whether the given table has cells with
// percentage widths (in its rows, not in tables in it).
func (ctx *Context) hasPercentColumns(table *html.Node) bool {
	var rows func(n *html.Node) bool
	rows = func(n *html.Node) bool {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			switch c.Data {
			case "thead", "tbody", "tfoot":
				if rows(c) {
					return true
				}
			case "tr":
				for cell := c.FirstChild; cell != nil; cell = cell.NextSibling {
					if (cell.Data == "td" || cell.Data == "th") && strings.HasSuffix(strings.TrimSpace(ctx.declaration(cell, "width")), "%") {
						return true
					}
				}
			}
		}
		return false
	}
	return rows(table)
}

// tableColumns returns the number of columns of the given table, the
// largest number of cells (with their column spans) in any of its rows,
// not counting those that are not displayed.
func (ctx *Context) tableColumns(table *html.Node) int {
	cols := 0
	var rows func(n *html.Node)
	rows = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			switch c.Data {
			case "thead", "tbody", "tfoot":
				rows(c)
			case "tr":
				n := 0
				for cell := c.FirstChild; cell != nil; cell = cell.NextSibling {
					if (cell.Data == "td" || cell.Data == "th") && ctx.displayOf(cell) != "none" {
						span, err := strconv.Atoi(GetAttr(cell, "colspan"))
						if err != nil || span < 1 {
							span = 1
						}
						n += span
					}
				}
				cols = max(cols, n)
			}
		}
	}
	rows(table)
	return max(cols, 1)
}

// handleCell handles a th or td element.
func handleCell(ctx *Context, tag string) core.Widget {
	ctx.BlockParent = ctx.tableFor(ctx.Node)
	ctx.inlineParent = nil
	span, err := strconv.Atoi(GetAttr(ctx.Node, "colspan"))
	if err != nil || span < 1 {
		span = 1
	}
	w := New[core.Frame](ctx)
	ctx.NewParent = w
	// Columns grow by the size of their content, as in browsers, for
	// which the length of the text is a proxy, only when the table
	// has a width to grow into.
	weight := float32(max(len([]rune(strings.TrimSpace(nodeText(ctx.Node)))), 1))
	rspan, err := strconv.Atoi(GetAttr(ctx.Node, "rowspan"))
	if err != nil || rspan < 1 {
		rspan = 1
	}
	// a percentage width of a cell in a table that fills its width
	// (see handleTable) is the width of its column
	pct := float32(0)
	if table := ancestorTable(ctx.Node); table != nil && ctx.declaration(table, "width") == "" && ctx.hasPercentColumns(table) {
		if v, ok := strings.CutSuffix(strings.TrimSpace(ctx.declaration(ctx.Node, "width")), "%"); ok {
			if f, err := strconv.ParseFloat(v, 32); err == nil {
				pct = float32(f)
			}
		}
	}
	w.FinalStyler(func(s *styles.Style) {
		if pct > 0 {
			s.Min.X, s.Max.X = units.Pw(pct), units.Pw(pct)
			s.BoxSizing = styles.BoxBorderBox
		}
		if span > 1 {
			s.ColSpan = span
		}
		if rspan > 1 {
			s.RowSpan = rspan
		}
		if s.Grow.X > 0 {
			s.Grow.X = weight / float32(span)
		}
	})
	return w
}

func handleImage(ctx *Context, tag string) core.Widget {
	if strings.TrimSpace(GetAttr(ctx.Node, "src")) == "" {
		return nil // an image without a source is not shown
	}
	n := ctx.Node
	src := GetAttr(n, "src")
	alt := GetAttr(n, "alt")
	isSVG := strings.HasSuffix(src, ".svg") || strings.HasPrefix(src, "data:image/svg")
	var newWidget core.Widget
	var resp *http.Response
	var err error
	// imageError handles an error getting the image, which is only logged
	// at the debug level, as browsers just do not show images that are
	// missing or not supported.
	imageError := func(err error) bool {
		if err == nil {
			return false
		}
		slog.Debug("error loading image", "url", src, "err", err)
		return true
	}
	var svgData []byte
	if !ctx.DelayedImageLoad && isSVG {
		resp, err = Get(ctx, src)
		if imageError(err) {
			return nil
		}
		defer resp.Body.Close()
	}
	var im image.Image
	if !ctx.DelayedImageLoad && !isSVG && ctx.ImageCache == nil {
		// An image is an svg by its type (not only the extension of its URL).
		resp, err = Get(ctx, src)
		if imageError(err) {
			return nil
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if imageError(err) {
			return nil
		}
		if strings.Contains(resp.Header.Get("Content-Type"), "svg") || looksLikeSVG(data) {
			isSVG, svgData = true, data
		} else if im, _, err = imagex.Read(bytes.NewReader(data)); imageError(err) {
			return nil
		}
	} else if !ctx.DelayedImageLoad && !isSVG {
		im, err = readImage(ctx, src)
		if imageError(err) {
			return nil
		}
	}

	// Can be either image or svg.
	var img *core.Image
	var svg *core.SVG
	if isSVG {
		svg = New[core.SVG](ctx)
		svg.SetTooltip(alt)
		if !ctx.DelayedImageLoad {
			data, err := svgData, error(nil)
			if data == nil {
				data, err = io.ReadAll(resp.Body)
			}
			if errors.Log(err) == nil {
				errors.Log(svg.Read(bytes.NewReader(data)))
				if w, h, ok := svgSize(data); ok { // its intrinsic size
					svg.SetProperty("svg-size", math32.Vec2(w, h))
				}
			}
		}
		newWidget = svg
	} else {
		img = New[core.Image](ctx)
		img.SetTooltip(alt)
		img.SetProperty("src", src) // for generatehtml
		img.SetImage(im)
		newWidget = img
	}

	if !ctx.DelayedImageLoad {
		return newWidget
	}
	go func() {
		if svg == nil {
			im, err := readImage(ctx, src)
			if errors.Log(err) != nil {
				return
			}
			img.AsyncLock()
			img.SetImage(im)
			img.Update()
			img.AsyncUnlock()
			return
		}
		resp, err := Get(ctx, src)
		if errors.Log(err) != nil {
			return
		}
		defer resp.Body.Close()
		svg.AsyncLock()
		errors.Log(svg.Read(resp.Body))
		svg.Update()
		svg.AsyncUnlock()
	}()
	return newWidget
}

// readImage returns the decoded image at the given URL, using
// [Context.ImageCache] if it is set. A decoding error is logged
// and gives a nil image without an error, as the image widget is
// still made then.
func readImage(ctx *Context, src string) (image.Image, error) {
	cache := ctx.ImageCache
	key := ""
	if cache != nil {
		u, err := parseRelativeURL(src, ctx.PageURL)
		if err != nil {
			return nil, err
		}
		key = u.String()
		if im := cache.Get(key); im != nil {
			return im, nil
		}
	}
	resp, err := Get(ctx, src)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	im, _, err := imagex.Read(resp.Body)
	if err != nil {
		// browsers just do not show images that are not supported
		slog.Debug("error loading image", "url", src, "err", err)
		return nil, nil
	}
	if cache != nil {
		cache.Add(key, im)
	}
	return im, nil
}

// handleTextTag creates a new [core.Text] from the given information, setting the text and
// the text click function so that URLs are opened according to [Context.OpenURL]. Also,
// it wraps the text with the [nodeString] of the given node, meaning that it
// should be used for standalone elements that are meant to only exist in text
// (eg: a, span, b, code, etc).
func handleTextTag(ctx *Context) *core.Text {
	start, end := nodeString(ctx.Node)
	str := start + ExtractText(ctx) + end
	tx := New[core.Text](ctx).SetText(str)
	tx.HandleTextClick(func(tl *rich.Hyperlink, e events.Event) {
		ctx.OpenURL(tl.URL, e)
	})
	return tx
}

// GetAttr gets the given attribute from the given node, returning ""
// if the attribute is not found.
func GetAttr(n *html.Node, attr string) string {
	res := ""
	for _, a := range n.Attr {
		if a.Key == attr {
			res = a.Val
		}
	}
	return res
}

// HasAttr returns whether the given node has the given attribute defined.
func HasAttr(n *html.Node, attr string) bool {
	return slices.ContainsFunc(n.Attr, func(a html.Attribute) bool {
		return a.Key == attr
	})
}

// GetLanguage returns the 'x' in a `language-x` class from the given
// string of class(es).
func GetLanguage(class string) string {
	fields := strings.Fields(class)
	for _, field := range fields {
		if strings.HasPrefix(field, "language-") {
			return strings.TrimPrefix(field, "language-")
		}
	}
	return ""
}

// Get is a helper function that calls [Context.GetURL] with the given URL, parsed
// relative to the page URL of the given context. It also checks the status
// code of the response and closes the response body and returns an error if
// it is not [http.StatusOK]. If the error is nil, then the response body is
// not closed and must be closed by the caller.
func Get(ctx *Context, url string) (*http.Response, error) {
	if strings.HasPrefix(url, "data:") {
		return dataURL(url)
	}
	u, err := parseRelativeURL(url, ctx.PageURL)
	if err != nil {
		return nil, err
	}
	resp, err := ctx.GetURL(u.String())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return resp, fmt.Errorf("got error status %q (code %d)", resp.Status, resp.StatusCode)
	}
	return resp, nil
}

// nodeText returns all of the text in the given node.
func nodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(nodeText(c))
	}
	return sb.String()
}

// dataURL returns a response with the data of the given data URL.
func dataURL(u string) (*http.Response, error) {
	meta, data, ok := strings.Cut(strings.TrimPrefix(u, "data:"), ",")
	if !ok {
		return nil, fmt.Errorf("invalid data URL")
	}
	var b []byte
	if strings.HasSuffix(meta, ";base64") {
		var err error
		b, err = base64.StdEncoding.DecodeString(data)
		if err != nil {
			b, err = base64.RawStdEncoding.DecodeString(data)
			if err != nil {
				return nil, err
			}
		}
	} else {
		d, err := neturl.PathUnescape(data)
		if err != nil {
			return nil, err
		}
		b = []byte(d)
	}
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(bytes.NewReader(b)),
		Header: http.Header{"Content-Type": {strings.Split(meta, ";")[0]}}}, nil
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
				if c.mediaMatches(GetAttr(n, "media")) {
					c.addStyleOrigin(nodeText(n), originAuthor)
				}
				return
			case "link":
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
	if errors.Log(err) != nil {
		return
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if errors.Log(err) != nil {
		return
	}
	base := c.cssBase
	c.cssBase = u.String()
	c.addStyleOrigin(string(b), originAuthor)
	c.cssBase = base
}

// cssBaseURL returns the URL that the URLs in the current style sheet are
// relative to: the URL of the style sheet, or that of the page.
func (c *Context) cssBaseURL() string {
	if c.cssBase != "" {
		return c.cssBase
	}
	return c.PageURL
}

// flattensBlock returns whether the given block element, in the flow of
// blocks of its parent, has no styles of its own (only its display), so
// that its children can be read into its parent. Then the floats in it
// are beside the blocks after it, as in CSS, where they are not limited
// to the block that they are in (as they are in core). Such blocks are
// common wrappers, as the section elements of Wikipedia.
func (ctx *Context) flattensBlock(n *html.Node) bool {
	switch n.Data {
	case "body", "html", "form", "a", "button", "label", "details", "dialog", "fieldset":
		return false // they have widgets with a function
	}
	p := n.Parent
	if p == nil || p.Type != html.ElementNode || p.Data == "html" {
		return false
	}
	switch ctx.displayOf(p) {
	case "block", "list-item", "flow-root", "table-cell":
	default:
		return false // a flex, grid, or table item
	}
	if !ctx.unstyledBlock(n) {
		return false
	}
	// Only the blocks among siblings that have floats in them are
	// flattened, so that the floats are beside all of their content.
	if ctx.floatSiblings == nil {
		ctx.floatSiblings = map[*html.Node]bool{}
	}
	if ctx.flattened[n] { // flattened already
		return true
	}
	has, ok := ctx.floatSiblings[p]
	if !ok && ctx.flattened[p] {
		has, ok = true, true // in a flattened block, as beside its floats
	}
	if !ok {
		for c := p.FirstChild; c != nil && !has; c = c.NextSibling {
			if c.Type == html.ElementNode && ctx.unstyledBlock(c) {
				has = ctx.hasFloatChild(c)
			}
		}
		ctx.floatSiblings[p] = has
	}
	if has {
		if ctx.flattened == nil {
			ctx.flattened = map[*html.Node]bool{}
		}
		ctx.flattened[n] = true
	}
	return has
}

// unstyledBlock returns whether the given element has no declarations
// other than its display (and custom properties).
func (ctx *Context) unstyledBlock(n *html.Node) bool {
	for _, d := range ctx.declarations(n) {
		switch d.Property {
		case "display", "unicode-bidi":
		default:
			if !strings.HasPrefix(d.Property, "--") {
				return false
			}
		}
	}
	return ctx.displayOf(n) == "block"
}

// hasFloatChild returns whether the given element has a child that floats.
func (ctx *Context) hasFloatChild(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		if f := ctx.declaration(c, "float"); f != "" && f != "none" && ctx.displayOf(c) != "none" {
			return true
		}
	}
	return false
}

// looksLikeSVG returns whether the given image data is an SVG document.
func looksLikeSVG(data []byte) bool {
	head := data[:min(len(data), 512)]
	return bytes.Contains(head, []byte("<svg")) || (bytes.HasPrefix(bytes.TrimSpace(head), []byte("<?xml")) && bytes.Contains(data[:min(len(data), 4096)], []byte("<svg")))
}

// svgRootTag matches the start tag of the root svg element.
var svgRootTag = regexp.MustCompile(`(?is)<svg\b[^>]*>`)

// svgAttr matches an attribute of a start tag.
var svgAttr = regexp.MustCompile(`(?is)\s(width|height|viewBox)\s*=\s*["']([^"']*)["']`)

// svgSize returns the intrinsic size of the given SVG document, in px,
// from the width and height of its root element, or the size of its view
// box for those that are not set, as for the size of an img of it.
func svgSize(data []byte) (w, h float32, ok bool) {
	calcEnvMu.Lock() // for parseLength (of absolute lengths)
	defer calcEnvMu.Unlock()
	tag := svgRootTag.Find(data)
	if tag == nil {
		return 0, 0, false
	}
	var vw, vh float32
	for _, m := range svgAttr.FindAllSubmatch(tag, -1) {
		v := string(m[2])
		switch strings.ToLower(string(m[1])) {
		case "width":
			if l, lok := parseLength(v, units.UnitPw); lok && l.Unit == units.UnitDp {
				w = l.Value
			}
		case "height":
			if l, lok := parseLength(v, units.UnitPh); lok && l.Unit == units.UnitDp {
				h = l.Value
			}
		case "viewbox":
			vb := strings.Fields(strings.ReplaceAll(v, ",", " "))
			if len(vb) == 4 {
				x, _ := strconv.ParseFloat(vb[2], 32)
				y, _ := strconv.ParseFloat(vb[3], 32)
				vw, vh = float32(x), float32(y)
			}
		}
	}
	switch {
	case w > 0 && h == 0 && vw > 0:
		h = w * vh / vw
	case h > 0 && w == 0 && vh > 0:
		w = h * vw / vh
	case w == 0 && h == 0:
		w, h = vw, vh
	}
	return w, h, w > 0 && h > 0
}
