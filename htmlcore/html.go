// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package htmlcore converts HTML and MD into Cogent Core widget trees.
package htmlcore

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"cogentcore.org/core/core"
	"golang.org/x/net/html"
)

// ReadHTML reads HTML from the given [io.Reader] and adds corresponding
// Cogent Core widgets to the given [core.Widget], using the given context.
func ReadHTML(ctx *Context, parent core.Widget, r io.Reader) error {
	// In browser style, the HTML is parsed as browsers do without
	// JavaScript (which htmlcore does not run), so that the content
	// of noscript elements is shown.
	n, err := html.ParseWithOptions(r, html.ParseOptionEnableScripting(!ctx.BrowserStyle))
	if err != nil {
		return fmt.Errorf("error parsing HTML: %w", err)
	}
	if ctx.BrowserStyle {
		closeDetails(n)
	}

	// HTML that an element handler reads while HTML is being read, such as
	// the page that content embeds for an <embed-page>, is read whole, and
	// the state of the outer read, including its staging, is kept.
	if ctx.reading {
		outer, stageSize := ctx.readState, ctx.StageSize
		ctx.readState, ctx.StageSize = readState{}, 0
		defer func() { ctx.readState, ctx.StageSize = outer, stageSize }()
		return readHTMLNode(ctx, parent, n)
	}
	ctx.readState = readState{}
	ctx.reading = true
	defer func() { ctx.reading = false }()
	return readHTMLNode(ctx, parent, n)
}

// ReadHTMLString reads HTML from the given string and adds corresponding
// Cogent Core widgets to the given [core.Widget], using the given context.
func ReadHTMLString(ctx *Context, parent core.Widget, s string) error {
	b := bytes.NewBufferString(s)
	return ReadHTML(ctx, parent, b)
}

// readHTMLNode reads HTML from the given [*html.Node] and its next siblings
// and adds corresponding Cogent Core widgets to the given [core.Widget],
// using the given context.
func readHTMLNode(ctx *Context, parent core.Widget, n *html.Node) error {
	for ; n != nil; n = n.NextSibling {
		// The rest of a staged body is left for later.
		if ctx.stageAt(n) {
			ctx.rest, ctx.restParent = n, parent
			return nil
		}
		if ctx.BrowserStyle && n.Parent != nil && ctx.isInlineLevel(n) {
			last := n
			for last.NextSibling != nil && ctx.isInlineLevel(last.NextSibling) && !ctx.stageAt(last.NextSibling) {
				last = last.NextSibling
			}
			ctx.readInlineRun(parent, n, last)
			n = last
			continue
		}
		readOneNode(ctx, parent, n)
	}
	return nil
}

// readOneNode reads the given node and its children (but not its
// siblings), adding the widgets for it to the given parent.
func readOneNode(ctx *Context, parent core.Widget, n *html.Node) {
	// nil parent means we are root, so we add user agent styles here
	if n.Parent == nil {
		ctx.Node = n
		if ctx.BrowserStyle {
			ctx.addStyleOrigin(browserStyles, originUserAgent)
			ctx.addStyleSheets(n)
			ctx.rootFont = 0
			ctx.rootFontSize() // of this document, while ctx.Node is it
		} else {
			ctx.addStyleOrigin(userAgentStyles, originUserAgent)
		}
	}

	switch n.Type {
	case html.TextNode:
		str := strings.TrimSpace(n.Data)
		if str != "" {
			ctx.handleWidget(n, New[core.Text](ctx).SetText(str))
		}
	case html.ElementNode:
		ctx.Node = n
		ctx.BlockParent = parent
		ctx.NewParent = nil

		handleElement(ctx)
	default:
		ctx.NewParent = parent
	}

	if np := ctx.NewParent; np != nil && n.FirstChild != nil {
		readHTMLNode(ctx, np, n.FirstChild)
		if ctx.BrowserStyle && n.Type == html.ElementNode && ctx.rest == nil {
			if fr, ok := np.(*core.Frame); ok && fr.Property("node") == n {
				switch ctx.displayOf(n) {
				case "grid", "inline-grid":
					ctx.arrangeGrid(n, fr)
				case "block", "list-item", "flow-root", "inline-block":
					ctx.arrangeColumns(n, fr)
				}
			}
		}
	}
}

// rootNode returns the root node of the given node.
func rootNode(n *html.Node) *html.Node {
	for n.Parent != nil {
		n = n.Parent
	}
	return n
}

// closeDetails removes the content of the details elements under n that
// are closed (without an open attribute), other than their summary, as
// browsers only show the summary of those.
func closeDetails(n *html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		closeDetails(c)
	}
	if n.Type != html.ElementNode || n.Data != "details" {
		return
	}
	for _, a := range n.Attr {
		if a.Key == "open" {
			return
		}
	}
	summary := false
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		if c.Type == html.ElementNode && c.Data == "summary" && !summary {
			summary = true
		} else {
			n.RemoveChild(c)
		}
		c = next
	}
}
