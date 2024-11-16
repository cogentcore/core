// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strings"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
)

// WikilinkHandler is a function that converts wikilink text to
// a corresponding URL and label text. If it returns "", "", the
// handler will be skipped in favor of the next possible handlers.
// Wikilinks are of the form [[wikilink text]]. Only the text inside
// of the brackets is passed to the handler.
type WikilinkHandler func(text string) (url string, label string)

// AddWikilinkHandler adds a new [WikilinkHandler] to [Context.WikilinkHandlers].
// If it returns "", "", the next handlers will be tried instead.
// The functions are tried in sequential ascending order.
func (c *Context) AddWikilinkHandler(f WikilinkHandler) {
	c.WikilinkHandlers = append(c.WikilinkHandlers, f)
}

// GoDocWikilink returns a [WikilinkHandler] that converts wikilinks of the form
// [[prefix:identifier]] to a pkg.go.dev URL starting at base. For example, with
// base="cogentcore.org/core" and prefix="doc", the wikilink [[doc:core.Button]] will
// result in the URL "https://pkg.go.dev/cogentcore.org/core/core#Button".
func GoDocWikilink(base string, prefix string) WikilinkHandler {
	return func(text string) (url string, label string) {
		if !strings.HasPrefix(text, prefix+":") {
			return "", ""
		}
		text = strings.TrimPrefix(text, prefix+":")
		// pkg.go.dev uses fragments for first dot within package
		t := strings.Replace(text, ".", "#", 1)
		url = "https://pkg.go.dev/" + base + "/" + t
		return url, text
	}
}

// note: this is from: https://github.com/kensanata/oddmu/blob/main/parser.go

// wikilink returns an inline parser function. This indirection is
// required because we want to call the previous definition in case
// this is not a wikilink.
func wikilink(ctx *Context, fn func(p *parser.Parser, data []byte, offset int) (int, ast.Node)) func(p *parser.Parser, data []byte, offset int) (int, ast.Node) {
	return func(p *parser.Parser, original []byte, offset int) (int, ast.Node) {
		data := original[offset:]
		n := len(data)
		// minimum: [[X]]
		if n < 5 || data[1] != '[' {
			return fn(p, original, offset)
		}
		i := 2
		for i+1 < n && data[i] != ']' && data[i+1] != ']' {
			i++
		}
		text := data[2 : i+1]
		url, label := "", ""
		for _, h := range ctx.WikilinkHandlers {
			u, l := h(string(text))
			if u == "" && l == "" {
				continue
			}
			url, label = u, l
			break
		}
		if url == "" && label == "" {
			return fn(p, original, offset)
		}
		link := &ast.Link{
			Destination: []byte(url),
		}
		ast.AppendChild(link, &ast.Text{Leaf: ast.Leaf{Literal: []byte(label)}})
		return i + 3, link
	}
}
