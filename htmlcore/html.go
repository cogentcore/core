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
	n, err := ParseHTML(ctx, r)
	if err != nil {
		return err
	}
	return ReadDocument(ctx, parent, n)
}

// ParseHTML parses the given HTML as browsers do: with scripting enabled
// if [Context.Scripting] is set (so that the content of noscript elements
// is their text, which is not shown), and otherwise as browsers do without
// JavaScript, so that the content of noscript elements is shown.
func ParseHTML(ctx *Context, r io.Reader) (*html.Node, error) {
	n, err := html.ParseWithOptions(r, html.ParseOptionEnableScripting(ctx.Scripting))
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %w", err)
	}
	return n, nil
}

// ReadDocument adds the Cogent Core widgets for the given parsed HTML
// document (or other node, with its next siblings) to the given widget,
// using the given context, as [ReadHTML] does after it parses its HTML.
// It allows the document to be read again after it is changed, as by the
// scripts of a page (which need its nodes), with a new context for each
// read (see [Context.NewPage]); the nodes of the document are those of the
// widgets (see [NodeOf]).
func ReadDocument(ctx *Context, parent core.Widget, n *html.Node) error {
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
	if ctx.ImageCache != nil {
		ctx.ImageCache.NewRead() // (its images are kept over the others)
	}
	// the style sheets of the new page are imported again, and it has its title
	ctx.cssImports = nil
	ctx.mediaUses, ctx.columnUses = nil, nil // of the new page
	ctx.Title = ""
	ctx.reading = true
	defer func() { ctx.reading = false }()
	err := readHTMLNode(ctx, parent, n)
	ctx.lockStylesOf(parent)
	return err
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
	// the text of the ::before and ::after of the element, as texts of
	// their own when its first and last children are not in a run of
	// inline content (which has them; see [Context.readInlineRun])
	el := n.Parent
	if el != nil && el.Type == html.ElementNode && el.FirstChild == n && !ctx.isInlineLevel(n) {
		ctx.pseudoText(parent, el, "before")
	}
	for ; n != nil; n = n.NextSibling {
		// The rest of a staged body is left for later.
		if ctx.stageAt(n) {
			ctx.rest, ctx.restParent = n, parent
			return nil
		}
		if n.NextSibling == nil && el != nil && el.Type == html.ElementNode && el.LastChild == n && !ctx.isInlineLevel(n) {
			defer ctx.pseudoText(parent, el, "after")
		}
		if n.Parent != nil && ctx.isInlineLevel(n) {
			last := n
			for last.NextSibling != nil && ctx.isInlineLevel(last.NextSibling) && !ctx.stageAt(last.NextSibling) {
				last = last.NextSibling
			}
			ctx.readInlineRun(parent, n, last)
			n = last
			continue
		}
		if ctx.keepRead[n] != nil {
			keptPlaceholder(parent, n) // kept as it is (see Context.keptElements)
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
		prev := ctx.reuseFrom
		ctx.reuseFrom = nil // (not kept, with all of its widgets)
		reuse := prev != nil && prev.snaps != nil && prev.snapRoot == n && prev.matchedRules == len(prev.rules)
		ctx.deferMatch = reuse
		ctx.addStyleOrigin(browserStyles, originUserAgent)
		if ctx.Style != "" { // before the style sheets of the page, which override it
			ctx.addStyleOrigin(ctx.Style, originAuthor)
		}
		restore := ctx.setCheckedAttrs(n)
		keptMasks := ctx.keepAncestorMasks() // (as the rules are matched)
		ctx.addStyleSheets(n)
		ctx.deferMatch = false
		if reuse && !ctx.reuseMatches(n, prev) {
			ctx.applyRules()
		}
		if ctx.snaps == nil || !reuse {
			ctx.snapshot(n)
		}
		ctx.matchedRules = len(ctx.rules)
		ctx.matchedPseudo = len(ctx.pseudoRules)
		keptMasks()
		restore()
		ctx.rootFont = 0
		ctx.rootFontSize() // of this document, while ctx.Node is it
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

	np := ctx.NewParent
	// the boxes of the ::before and ::after of an element (see
	// [Context.pseudoBoxNode]) are its first and last children while they
	// are read, in its flow (as in a run of its inline content)
	if np != nil && n.Type == html.ElementNode && n.Parent != nil && ctx.pseudoStyles != nil {
		if before := ctx.pseudoBoxNode(n, "before"); before != nil {
			n.InsertBefore(before, n.FirstChild)
			defer n.RemoveChild(before)
		}
		if after := ctx.pseudoBoxNode(n, "after"); after != nil {
			n.AppendChild(after)
			defer n.RemoveChild(after)
		}
	}
	if np != nil && n.FirstChild == nil && n.Type == html.ElementNode && n.Parent != nil && ctx.pseudoStyles != nil {
		// the text of the ::before and ::after of an empty element (as the
		// markers of the lines of the diffs of Wikipedia, in empty cells)
		if t := strings.TrimSpace(ctx.pseudoContent(n, "before") + ctx.pseudoContent(n, "after")); t != "" {
			ctx.pseudoText(np, n, "before")
			ctx.pseudoText(np, n, "after")
		}
	}
	if np != nil && n.FirstChild != nil {
		readHTMLNode(ctx, np, n.FirstChild)
		if n.Type == html.ElementNode && ctx.rest == nil {
			if fr := frameOf(np); fr != nil && fr.Property("node") == n {
				ctx.arrange(n, fr)
			}
		}
	}
	if n.Type == html.ElementNode && n.Data == "tr" && ctx.rest == nil && ctx.displayOf(n) != "none" {
		ctx.endTableRow(n) // the next row starts in the next row of the grid
	}
}

// rootNode returns the root node of the given node.
func rootNode(n *html.Node) *html.Node {
	for n.Parent != nil {
		n = n.Parent
	}
	return n
}
