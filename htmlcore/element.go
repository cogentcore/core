// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"slices"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/textcore"
	"golang.org/x/net/html"
)

// The widgets of an element are made by the builder of its kind, in
// handleElement: the document and metadata elements, blocks of text,
// list items (list.go), form controls (forms.go), replaced elements
// (images and svg, image.go), tables (table.go), inline elements of text
// (inline.go), and boxes for the other elements by their display. The
// children of an element are then read into the widget that its builder
// sets as [Context.NewParent], and the formatting contexts that arrange
// the widgets of the children (grid, multi-column, details) do so after
// they are read (see [Context.arrange]).

// documentTags are the elements of the document and its metadata, which
// are handled whatever their display (as head, which is not displayed).
var documentTags = map[string]bool{"style": true, "link": true, "script": true, "title": true,
	"meta": true, "head": true, "html": true}

// boxTags are the elements that have a box even with a display of
// contents (as replaced elements and form controls).
var boxTags = map[string]bool{"html": true, "head": true, "body": true, "img": true, "input": true,
	"button": true, "select": true, "textarea": true, "svg": true, "video": true, "iframe": true,
	"canvas": true, "br": true}

// textBlockTags are the elements that are blocks of text, with
// a frame for their text and any other content.
var textBlockTags = map[string]bool{"pre": true, "p": true, "h1": true, "h2": true, "h3": true,
	"h4": true, "h5": true, "h6": true, "hr": true, "dt": true, "dd": true, "figcaption": true,
	"caption": true, "summary": true, "legend": true}

// handleElement makes the widgets of the current node with the handler in
// [Context.ElementHandlers] for its tag, if any, or else with the builder
// of its kind.
func handleElement(ctx *Context) {
	tag := ctx.Node.Data
	if h, ok := ctx.ElementHandlers[tag]; ok && h(ctx) {
		return
	}
	display := ctx.displayOf(ctx.Node)
	if display == "none" && !documentTags[tag] {
		return // not rendered, and neither are its children
	}
	if display == "contents" && !boxTags[tag] {
		// it has no box, and its children are in its parent
		ctx.NewParent = ctx.BlockParent
		ctx.inlineParent = nil
		return
	}

	var newWidget core.Widget
	switch tag {
	case "html", "head":
		ctx.NewParent = ctx.BlockParent
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
	case "li":
		handleListItem(ctx)
	case "select":
		handleSelect(ctx)
	case "button":
		handleButton(ctx)
	case "input":
		if typ := strings.ToLower(GetAttr(ctx.Node, "type")); typ == "submit" || typ == "button" || typ == "reset" {
			handleInputButton(ctx, typ)
		} else {
			newWidget = handleInput(ctx)
		}
	case "textarea":
		buf := lines.NewLines()
		buf.SetText([]byte(ExtractText(ctx)))
		newWidget = New[textcore.Editor](ctx).SetLines(buf)
	case "svg":
		handleSVG(ctx)
	case "img":
		newWidget = handleImage(ctx, tag)
	case "table":
		newWidget = handleTable(ctx)
	case "th", "td":
		newWidget = handleCell(ctx, tag)
	case "thead", "tbody", "tfoot", "tr":
		ctx.NewParent = ctx.tableFor(ctx.Node)
	default:
		switch {
		case textBlockTags[tag]:
			handleTextBlock(ctx, tag)
		case slices.Contains(textTags, tag): // not in a run of inline content
			handleInlineElement(ctx, tag, display)
		default:
			newWidget = handleBox(ctx, tag, display)
		}
	}
	if newWidget != nil {
		ctx.handleWidget(ctx.Node, newWidget)
	}
}

// handleTextBlock handles an element that is a block of text (as p).
func handleTextBlock(ctx *Context, tag string) {
	w := New[core.Frame](ctx)
	ctx.NewParent = w
	ctx.handleWidget(ctx.Node, w)
	if p := ctx.Node.Parent; tag == "caption" && p != nil && p.Data == "table" {
		// the caption is above the cells, across all of the columns
		span := ctx.tableColumns(p)
		w.FinalStyler(func(s *styles.Style) { s.ColSpan = span })
	}
	if p := ctx.Node.Parent; p != nil && p.Data != "table" && ctx.displayOf(ctx.Node) == "table-caption" {
		// The caption of an element displayed as a table wraps in the
		// width of the table without widening it, which for a figure is
		// that of its image when it has a width (as for a thumbnail).
		if iw := tableImageWidth(p); iw > 0 {
			w.FinalStyler(func(s *styles.Style) { s.Max.X = units.Dp(iw); s.BoxSizing = styles.BoxBorderBox })
		}
	}
}

// handleInlineElement handles an inline element of text (as a or span)
// that is not in a run of inline content, with the given display.
func handleInlineElement(ctx *Context, tag, display string) {
	switch display {
	case "none":
		return
	case "", "inline":
		if !ctx.isTextElement(ctx.Node) {
			// an inline element with widgets in it, which are in the run
			// of inline content that it is in
			ctx.NewParent = ctx.Parent()
			return
		}
		ctx.handleWidget(ctx.Node, handleTextTag(ctx))
		return
	}
	w := New[core.Frame](ctx)
	ctx.NewParent = w
	ctx.handleWidget(ctx.Node, w)
	if href := GetAttr(ctx.Node, "href"); tag == "a" && href != "" {
		ctx.linkFrame(w, href)
	}
}

// handleBox handles the other elements by their display: a block (or
// inline block) has a frame for its children, and the children of an
// inline element are in the run of inline content that it is in.
// It returns the new widget, if any.
func handleBox(ctx *Context, tag, display string) core.Widget {
	switch {
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
		ctx.NewParent = w
		return w
	default:
		ctx.NewParent = ctx.Parent()
	}
	return nil
}

// arrange arranges the widgets of the children of the given element in
// the given frame of it, after they are read, for the formatting contexts
// that place them other than in order: grid areas, multiple columns, and
// the content of details.
func (c *Context) arrange(n *html.Node, fr *core.Frame) {
	switch c.displayOf(n) {
	case "grid", "inline-grid":
		c.arrangeGrid(n, fr)
	case "block", "list-item", "flow-root", "inline-block":
		c.arrangeColumns(n, fr)
	}
	if n.Data == "details" {
		c.arrangeDetails(n, fr)
	}
}

// linkFrame makes the given frame of an a element open the given URL.
func (c *Context) linkFrame(w *core.Frame, href string) {
	if c.linkFrames == nil {
		c.linkFrames = map[*html.Node]string{}
	}
	c.linkFrames[c.Node] = href
	w.SetProperty("href", href)
	w.SetTooltip(href)
	w.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Clickable)
		s.Cursor = cursors.Pointer
	})
	w.OnClick(func(e events.Event) {
		e.SetHandled()
		c.OpenURL(href, e)
	})
}

// handleInLink makes the given widget of the given node, if it is in a
// link that is a frame (see linkFrame), open the link when clicked on, as
// the frame does: the widgets in it that can be pressed, as its text,
// otherwise get the clicks instead of the frame.
func (c *Context) handleInLink(node *html.Node, w core.Widget) {
	if len(c.linkFrames) == 0 {
		return
	}
	for n := node; n != nil; n = n.Parent {
		href, ok := c.linkFrames[n]
		if !ok {
			continue
		}
		if n == node {
			if _, isFrame := w.(*core.Frame); isFrame {
				return // the link frame itself
			}
		}
		wb := w.AsWidget()
		wb.Styler(func(s *styles.Style) { s.Cursor = cursors.Pointer })
		wb.OnClick(func(e events.Event) {
			if e.IsHandled() {
				return
			}
			e.SetHandled()
			c.OpenURL(href, e)
		})
		return
	}
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
