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
	n, err := html.Parse(r)
	if err != nil {
		return fmt.Errorf("error parsing HTML: %w", err)
	}
	fmt.Println("parsed html")
	return readHTMLNode(ctx, parent, n)
}

// ReadHTMLString reads HTML from the given string and adds corresponding
// Cogent Core widgets to the given [core.Widget], using the given context.
func ReadHTMLString(ctx *Context, parent core.Widget, s string) error {
	b := bytes.NewBufferString(s)
	return ReadHTML(ctx, parent, b)
}

// readHTMLNode reads HTML from the given [*html.Node] and adds corresponding
// Cogent Core widgets to the given [core.Widget], using the given context.
func readHTMLNode(ctx *Context, parent core.Widget, n *html.Node) error {
	fmt.Println("read node", n)
	// nil parent means we are root, so we add user agent styles here
	if n.Parent == nil {
		ctx.Node = n
		ctx.addStyle(userAgentStyles)
	}

	switch n.Type {
	case html.TextNode:
		str := strings.TrimSpace(n.Data)
		if str != "" {
			New[core.Text](ctx).SetText(str)
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

	if n.NextSibling != nil {
		readHTMLNode(ctx, parent, n.NextSibling)
	}
	return nil
}

// rootNode returns the root node of the given node.
func rootNode(n *html.Node) *html.Node {
	for n.Parent != nil {
		n = n.Parent
	}
	return n
}
