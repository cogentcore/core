// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package htmlview converts HTML and MD into Cogent Core widget trees.
package htmlview

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
	return ReadHTMLNode(ctx, parent, n)
}

// ReadHTMLString reads HTML from the given string and adds corresponding
// Cogent Core widgets to the given [core.Widget], using the given context.
func ReadHTMLString(ctx *Context, parent core.Widget, s string) error {
	b := bytes.NewBufferString(s)
	return ReadHTML(ctx, parent, b)
}

// ReadHTMLNode reads HTML from the given [*html.Node] and adds corresponding
// Cogent Core widgets to the given [core.Widget], using the given context.
func ReadHTMLNode(ctx *Context, parent core.Widget, n *html.Node) error {
	// nil parent means we are root, so we add user agent styles here
	if n.Parent == nil {
		ctx.Node = n
		ctx.AddStyle(UserAgentStyles)
	}

	switch n.Type {
	case html.TextNode:
		str := strings.TrimSpace(n.Data)
		if str != "" {
			New[*core.Label](ctx).SetText(str)
		}
	case html.ElementNode:
		ctx.Node = n
		ctx.BlockParent = parent
		ctx.NewParent = nil

		HandleElement(ctx)
	default:
		ctx.NewParent = parent
	}

	if ctx.NewParent != nil && n.FirstChild != nil {
		ReadHTMLNode(ctx, ctx.NewParent, n.FirstChild)
	}

	if n.NextSibling != nil {
		ReadHTMLNode(ctx, parent, n.NextSibling)
	}
	return nil
}

// RootNode returns the root node of the given node.
func RootNode(n *html.Node) *html.Node {
	for n.Parent != nil {
		n = n.Parent
	}
	return n
}
