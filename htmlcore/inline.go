// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
	xhtml "golang.org/x/net/html"
)

// The inline content of block elements is read
// in runs of consecutive inline-level nodes: the text of a run (its text
// nodes and the elements of text, such as span and b) is made into one
// [core.Text], with the inline elements as rich text markup, and a run that
// also has inline widgets (such as img and button) is put in a row that
// wraps, an inline container.

// anonymousInline is the display of an inline container.
const anonymousInline = "anonymous-inline"

// isInlineLevel returns whether the given node is inline-level content,
// which is read in a run of inline content.
func (c *Context) isInlineLevel(n *xhtml.Node) bool {
	switch n.Type {
	case xhtml.TextNode, xhtml.CommentNode:
		return true
	case xhtml.ElementNode:
		if _, ok := c.ElementHandlers[n.Data]; ok {
			return false
		}
		if n.Data == "colgroup" || n.Data == "col" {
			return false // the columns of a table, which have no boxes
		}
		if c.absoluteInRun(n) {
			// out of the flow, in the run where it is, which it does not
			// break into lines (as the reply info of the postings of
			// craigslist.org, between the reply button and the actions)
			return true
		}
		switch c.displayOf(n) {
		case "", "inline", "inline-block", "inline-flex", "inline-grid", "inline-table":
			return true
		}
	}
	return false
}

// isInlineBox returns whether the given inline element is a box of its
// own in its line, which is not all text in the markup of a [core.Text]
// (which has no borders or padding): an element with a border that does
// not wrap (as the tags of lobste.rs, with rounded corners), or with a
// border or horizontal padding and no text beside it (as the padded links
// of the footer of lobste.rs, in a nav). Elements in text that wraps (as
// links with a bottom border, or code with padding) are text.
func (c *Context) isInlineBox(n *xhtml.Node) bool {
	// only an element of text, which is text otherwise (with no images)
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		if !c.isTextElement(ch) {
			return false
		}
	}
	border := false
	for _, prop := range []string{"border", "border-top", "border-right", "border-bottom", "border-left", "border-width", "border-style"} {
		if v := strings.ToLower(strings.TrimSpace(c.declaration(n, prop))); !noLength(v) {
			border = true
			break
		}
	}
	if ws := strings.TrimSpace(c.declaration(n, "white-space")); border && (ws == "nowrap" || ws == "pre") {
		return true
	}
	padding := border
	for _, prop := range []string{"padding-left", "padding-right", "padding-inline", "padding-inline-start", "padding-inline-end"} {
		if !noLength(strings.TrimSpace(c.declaration(n, prop))) {
			padding = true
		}
	}
	if v := strings.Fields(c.declaration(n, "padding")); len(v) > 0 {
		// the horizontal padding is the second value, or the only one
		h := v[0]
		if len(v) > 1 {
			h = v[1]
		}
		padding = padding || !noLength(h)
	}
	if !padding || n.Parent == nil || n.Parent.Type != xhtml.ElementNode {
		return false
	}
	// in a block, not in an inline element (whose margins would be lost,
	// as the links of the navigation of w3.org, in inline items)
	switch strings.TrimSpace(c.declaration(n.Parent, "display")) {
	case "", "inline", "contents":
		return false
	}
	for s := n.Parent.FirstChild; s != nil; s = s.NextSibling {
		if s.Type == xhtml.TextNode && strings.TrimSpace(s.Data) != "" {
			return false
		}
	}
	return true
}

// noLength returns whether the given CSS value is no length (or none).
func noLength(v string) bool {
	v = strings.ToLower(v)
	return v == "" || strings.Contains(v, "none") || strings.Contains(v, "hidden") || v == "0" || strings.HasPrefix(v, "0 ") || strings.HasPrefix(v, "0px")
}

// absoluteInRun returns whether the given element is absolutely positioned
// (or fixed) between inline-level content in a block container, which it is
// in the run of, as it is out of the flow (in CSS, it is blockified, but it
// does not break the line that it is in).
func (c *Context) absoluteInRun(n *xhtml.Node) bool {
	pos := strings.TrimSpace(c.declaration(n, "position"))
	if (pos != "absolute" && pos != "fixed") || n.Parent == nil || c.displayOf(n) == "none" {
		return false
	}
	switch c.displayOf(n.Parent) {
	case "flex", "inline-flex", "grid", "inline-grid":
		return false
	}
	// the element (or text) before and after it, which are inline-level
	sibling := func(next bool) *xhtml.Node {
		s := n.PrevSibling
		if next {
			s = n.NextSibling
		}
		for s != nil {
			if s.Type == xhtml.ElementNode || (s.Type == xhtml.TextNode && strings.TrimSpace(s.Data) != "") {
				return s
			}
			if next {
				s = s.NextSibling
			} else {
				s = s.PrevSibling
			}
		}
		return nil
	}
	inline := func(s *xhtml.Node) bool {
		if s == nil {
			return false
		}
		if s.Type == xhtml.TextNode {
			return true
		}
		if p := strings.TrimSpace(c.declaration(s, "position")); p == "absolute" || p == "fixed" {
			return false
		}
		return c.isInlineLevel(s)
	}
	return inline(sibling(false)) && inline(sibling(true))
}

// isTextElement returns whether the given element is all text, which is
// represented in the markup of a [core.Text]: an element of text (such as
// span and b) that is displayed inline, with only text in it.
func (c *Context) isTextElement(n *xhtml.Node) bool {
	switch n.Type {
	case xhtml.TextNode, xhtml.CommentNode:
		return true
	case xhtml.ElementNode:
		if _, ok := c.ElementHandlers[n.Data]; ok {
			return false
		}
		switch n.Data {
		case "img", "button", "input", "select", "textarea", "svg", "video", "iframe", "canvas", "object", "embed", "math":
			return false
		}
		switch c.displayOf(n) {
		case "", "inline", "none":
		default:
			return false
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			if !c.isTextElement(ch) {
				return false
			}
		}
		return true
	}
	return false
}

// readInlineRun reads the run of inline content from first to last.
func (c *Context) readInlineRun(parent core.Widget, first, last *xhtml.Node) {
	hasWidget := false
	for n := first; ; n = n.NextSibling {
		if n.Type == xhtml.ElementNode && !c.isTextElement(n) && c.displayOf(n) != "none" {
			hasWidget = true
		}
		if n == last {
			break
		}
	}
	pre := c.isPre(first.Parent)
	container := parent
	newContainer := func() core.Widget {
		fr := core.NewFrame(parent)
		fr.SetName("inline-container")
		tree.SetUniqueName(fr)
		c.Node = nil
		c.browserStyler(nil, fr, nil)
		c.blocks[fr].display = anonymousInline
		return fr
	}
	if hasWidget && c.display(parent) != anonymousInline {
		container = newContainer()
	}

	var buf strings.Builder
	start := true // at the start of the run
	// flush makes a text of the inline content so far; between is whether
	// it is between inline widgets (before one, after content)
	flush := func(between bool) {
		str := buf.String()
		buf.Reset()
		if pre && last.NextSibling == nil { // a last line break makes no line
			str = lastBreak.ReplaceAllString(str, "$1")
		}
		if str != "" && strings.TrimSpace(str) == "" && !pre && between && container != parent {
			// White space between inline widgets (as the words in inline
			// blocks of the heading of github.com) is a space, as in
			// browsers: a box as wide as one in the font, as a text of
			// only a no-break space has no width (as trailing space).
			sp := core.NewFrame(container)
			sp.SetName("space")
			tree.SetUniqueName(sp)
			c.handleWidget(nil, sp)
			sp.FinalStyler(func(s *styles.Style) {
				s.Min.X = units.Em(0.28)
				s.Grow.Set(0, 0)
			})
			c.blocks[sp].display = "inline"
			return
		}
		if str == "" || (strings.TrimSpace(str) == "" && !pre) {
			return
		}
		if !pre {
			// Spaces at the start and end of the run are not shown, but
			// those next to inline widgets are, as non-breaking spaces.
			lead, trail := "", ""
			if !start && strings.HasPrefix(str, " ") {
				lead = "\u00a0"
			}
			if strings.HasSuffix(str, " ") && container != parent {
				trail = "\u00a0"
			}
			str = lead + strings.TrimSpace(str) + trail
			// white space at the start of a line is not shown
			str = brSpace.ReplaceAllString(str, "$1")
		}
		tx := core.NewText(container).SetText(str)
		tx.HandleTextClick(func(tl *rich.Hyperlink, e events.Event) {
			c.OpenURL(tl.URL, e)
		})
		c.handleWidget(nil, tx)
		c.handleInLink(first.Parent, tx) // the element that the run is in
		if container != parent || c.display(parent) == anonymousInline {
			c.blocks[tx].display = "inline"
		} else {
			c.blocks[tx].display = "block"
		}
	}
	// the text of the ::before and ::after of the element that the run is
	// in, at its start and end (as for elements that are widgets, as the
	// edit links of Wikipedia, which are inline blocks)
	if el := first.Parent; el != nil && el.FirstChild == first {
		buf.WriteString(c.pseudoMarkup(el, "before"))
	}
	for n := first; ; n = n.NextSibling {
		if container != parent && n.Type == xhtml.ElementNode && n.Data == "br" {
			// a line break in a row of widgets ends the row, and the
			// widgets after it are in another row (each is aligned)
			flush(false)
			start = true
			container = newContainer()
			if n == last {
				break
			}
			continue
		}
		if n.Type == xhtml.ElementNode && !c.isTextElement(n) && c.displayOf(n) != "none" {
			flush(!start)
			start = false
			c.inlineParent = container
			c.BlockParent = container
			readOneNode(c, container, n)
		} else {
			buf.WriteString(c.inlineMarkup(n, pre))
		}
		if n == last {
			break
		}
	}
	if el := last.Parent; el != nil && el.LastChild == last {
		buf.WriteString(c.pseudoMarkup(el, "after"))
	}
	flush(false)
	c.inlineParent = nil
}

// isPre returns whether the white space of the content of the given
// element is preserved.
func (c *Context) isPre(n *xhtml.Node) bool {
	for ; n != nil; n = n.Parent {
		if n.Type != xhtml.ElementNode {
			continue
		}
		switch c.declaration(n, "white-space") {
		case "pre", "pre-wrap", "pre-line", "break-spaces":
			return true
		case "normal", "nowrap":
			return false
		}
	}
	return false
}

// brSpace matches a line break with the white space after it.
var brSpace = regexp.MustCompile(`(<br\s*/?>)[ \t\n\r\f]+`)

var spaceRun = regexp.MustCompile(`[ \t\n\r\f]+`)

// lastBreak matches a line break at the end of markup, before end tags.
var lastBreak = regexp.MustCompile(`\n((?:</[a-z0-9]+>)*)$`)

// inlineMarkup returns the rich text markup for the given node of text.
func (c *Context) inlineMarkup(n *xhtml.Node, pre bool) string {
	switch n.Type {
	case xhtml.TextNode:
		t := n.Data
		if !pre {
			t = spaceRun.ReplaceAllString(t, " ")
		}
		return html.EscapeString(t)
	case xhtml.ElementNode:
		if c.displayOf(n) == "none" {
			return ""
		}
		if n.Data == "br" {
			return "<br>"
		}
		var sb strings.Builder
		start, end := c.nodeMarkup(n, pre)
		sb.WriteString(start)
		sb.WriteString(c.pseudoMarkup(n, "before"))
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			sb.WriteString(c.inlineMarkup(ch, pre))
		}
		sb.WriteString(c.pseudoMarkup(n, "after"))
		sb.WriteString(end)
		return sb.String()
	}
	return ""
}

// cssStyledTags are the rich text tags that have styles of their own in
// rich text (as a background for code), which are spans in preformatted
// text (as of pre elements), so that they only have their CSS styles
// there, without boxes behind the code in the boxes of the pre elements.
// Elsewhere they have their styles of rich text (code is monospace on a
// background), as in Cogent Core.
var cssStyledTags = map[string]bool{"code": true, "kbd": true, "samp": true, "tt": true, "mark": true}

// richTags are the tags that rich text markup has, which other inline
// elements are made into spans in place of.
var richTags = map[string]bool{
	"a": true, "b": true, "strong": true, "i": true, "em": true, "var": true, "cite": true,
	"u": true, "ins": true, "s": true, "del": true, "strike": true, "small": true, "big": true,
	"sub": true, "sup": true, "code": true, "kbd": true, "samp": true, "tt": true, "q": true,
	"mark": true, "abbr": true, "acronym": true, "span": true, "br": true, "dfn": true, "bdo": true,
}

// inlineProperties are the CSS properties of inline elements
// that are given to the rich text markup.
var inlineProperties = map[string]bool{
	"color": true, "background-color": true, "background": true, "font-weight": true,
	"font-style": true, "font-size": true, "font-family": true, "text-decoration": true,
	"text-decoration-line": true,
}

// nodeMarkup returns the start and end tags of the markup for the given
// element of text, with the CSS styles of the element in its style,
// in preformatted text if pre.
func (c *Context) nodeMarkup(n *xhtml.Node, pre bool) (start, end string) {
	tag := n.Data
	if !richTags[tag] || (pre && cssStyledTags[tag]) {
		tag = "span"
	}
	var sb strings.Builder
	sb.WriteString("<" + tag)
	for _, a := range n.Attr {
		if a.Key == "class" && (a.Val == "math inline" || a.Val == "math display") {
			// the math of markdown, which rich text renders
			fmt.Fprintf(&sb, ` class="%s"`, a.Val)
			continue
		}
		if a.Key == "style" || a.Key == "class" || a.Key == "id" || strings.HasPrefix(a.Key, "data-") {
			continue
		}
		fmt.Fprintf(&sb, ` %s="%s"`, a.Key, html.EscapeString(a.Val))
	}
	// The last declaration of each property is the one that applies,
	// and one that inherits is that of the parent, which the span has.
	var props []string
	vals := map[string]string{}
	for _, d := range c.declarations(n) {
		if !inlineProperties[d.Property] || d.origin == originUserAgent || d.invalid {
			continue
		}
		prop, val := d.Property, d.Value
		if prop == "background" {
			prop, val = "background-color", backgroundColor(val)
			if val == "" {
				continue
			}
		}
		if _, ok := vals[prop]; !ok {
			props = append(props, prop)
		}
		vals[prop] = val
	}
	var style strings.Builder
	for _, prop := range props {
		val := vals[prop]
		switch strings.ToLower(val) {
		case "inherit", "unset", "revert", "revert-layer":
			continue
		}
		switch prop {
		case "font-weight":
			if w, err := strconv.Atoi(val); err == nil {
				val = "normal"
				if w >= 600 {
					val = "bold"
				}
			}
		case "font-size":
			val = c.relativeFontSize(n)
			if val == "" {
				continue
			}
		case "font-family":
			if c.WebFonts && webFontFamily(val) != "" {
				// the custom font of the text (as the web font of the
				// element of the text, which runs cannot have their own)
				val = rich.Custom.String()
			} else {
				val = fontFamily(val).String()
			}
		case "color", "background-color":
			if cl, ok := cssColorFunction(val); ok { // as colors.FromString does not parse CSS alpha
				val = colors.AsHex(cl)
			} else if strings.Contains(val, "(") {
				continue // a function that is not supported (as color-mix())
			}
		}
		fmt.Fprintf(&style, "%s:%s;", prop, val)
	}
	if style.Len() > 0 {
		fmt.Fprintf(&sb, ` style="%s"`, html.EscapeString(style.String()))
	}
	sb.WriteString(">")
	return sb.String(), "</" + tag + ">"
}

// relativeFontSize returns the font size of the given inline element
// as a percentage of that of its parent, for rich text markup.
func (c *Context) relativeFontSize(n *xhtml.Node) string {
	if n.Parent == nil {
		return ""
	}
	f := c.fontSizePx(n) / c.fontSizePx(n.Parent)
	if f <= 0 {
		return ""
	}
	return strconv.FormatFloat(float64(100*f), 'f', 1, 32) + "%"
}

// fontSizePx returns the computed font size in px of the given element,
// from the declarations of it and its ancestors.
func (c *Context) fontSizePx(n *xhtml.Node) float32 {
	if n == nil {
		return 16
	}
	if n.Type != xhtml.ElementNode {
		return c.fontSizePx(n.Parent)
	}
	v := strings.ToLower(c.declaration(n, "font-size"))
	if v == "" || v == "inherit" {
		return c.fontSizePx(n.Parent)
	}
	if px, ok := cssFontSizes[v]; ok {
		return px
	}
	parent := c.fontSizePx(n.Parent)
	num := func(suffix string) (float32, bool) {
		f, err := strconv.ParseFloat(strings.TrimSuffix(v, suffix), 32)
		return float32(f), err == nil
	}
	switch {
	case v == "smaller":
		return parent / 1.2
	case v == "larger":
		return parent * 1.2
	case strings.HasSuffix(v, "%"):
		if f, ok := num("%"); ok {
			return parent * f / 100
		}
	case strings.HasSuffix(v, "rem"):
		if f, ok := num("rem"); ok {
			return 16 * f
		}
	case strings.HasSuffix(v, "em"):
		if f, ok := num("em"); ok {
			return parent * f
		}
	case strings.HasSuffix(v, "px"):
		if f, ok := num("px"); ok {
			return f
		}
	case strings.HasSuffix(v, "pt"):
		if f, ok := num("pt"); ok {
			return f * 96 / 72
		}
	}
	return parent
}
