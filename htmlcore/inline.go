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
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
	xhtml "golang.org/x/net/html"
)

// The inline content of block elements, for [Context.BrowserStyle], is read
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
		switch c.displayOf(n) {
		case "", "inline", "inline-block", "inline-flex", "inline-grid", "inline-table":
			return true
		}
	}
	return false
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
	if hasWidget && c.display(parent) != anonymousInline {
		fr := core.NewFrame(parent)
		fr.SetName("inline-container")
		tree.SetUniqueName(fr)
		c.Node = nil
		c.browserStyler(nil, fr, nil)
		c.blocks[fr].display = anonymousInline
		container = fr
	}

	var buf strings.Builder
	start := true // at the start of the run
	flush := func() {
		str := buf.String()
		buf.Reset()
		if pre && last.NextSibling == nil { // a last line break makes no line
			str = lastBreak.ReplaceAllString(str, "$1")
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
		}
		tx := core.NewText(container).SetText(str)
		tx.HandleTextClick(func(tl *rich.Hyperlink, e events.Event) {
			c.OpenURL(tl.URL, e)
		})
		c.handleWidget(nil, tx)
		if container != parent || c.display(parent) == anonymousInline {
			c.blocks[tx].display = "inline"
		} else {
			c.blocks[tx].display = "block"
		}
	}
	for n := first; ; n = n.NextSibling {
		if n.Type == xhtml.ElementNode && !c.isTextElement(n) && c.displayOf(n) != "none" {
			flush()
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
	flush()
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
		start, end := c.nodeMarkup(n)
		sb.WriteString(start)
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			sb.WriteString(c.inlineMarkup(ch, pre))
		}
		sb.WriteString(end)
		return sb.String()
	}
	return ""
}

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
// element of text, with the CSS styles of the element in its style.
func (c *Context) nodeMarkup(n *xhtml.Node) (start, end string) {
	tag := n.Data
	if !richTags[tag] {
		tag = "span"
	}
	var sb strings.Builder
	sb.WriteString("<" + tag)
	for _, a := range n.Attr {
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
			val = fontFamily(val).String()
		case "color", "background-color":
			if cl, ok := cssColorFunction(val); ok { // as colors.FromString does not parse CSS alpha
				val = colors.AsHex(cl)
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
