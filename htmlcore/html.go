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

	if ctx.NewParent != nil && n.FirstChild != nil {
		readHTMLNode(ctx, ctx.NewParent, n.FirstChild)
	}
}

// rootNode returns the root node of the given node.
func rootNode(n *html.Node) *html.Node {
	for n.Parent != nil {
		n = n.Parent
	}
	return n
}
