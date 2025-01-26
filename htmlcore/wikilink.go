// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"log/slog"
	"strings"
	"unicode"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
)

// WikilinkHandler is a function that converts wikilink text to
// a corresponding URL and label text. If it returns "", "", the
// handler will be skipped in favor of the next possible handlers.
// Wikilinks are of the form [[wikilink text]]. Only the text inside
// of the brackets is passed to the handler. If there is additional
// text directly after the closing brackets without spaces or punctuation,
// it will be appended to the label text after the handler is run
// (ex: [[widget]]s).
type WikilinkHandler func(text string) (url string, label string)

// AddWikilinkHandler adds a new [WikilinkHandler] to [Context.WikilinkHandlers].
// If it returns "", "", the next handlers will be tried instead.
// The functions are tried in sequential ascending order.
func (c *Context) AddWikilinkHandler(h WikilinkHandler) {
	c.WikilinkHandlers = append(c.WikilinkHandlers, h)
}

// GoDocWikilink returns a [WikilinkHandler] that converts wikilinks of the form
// [[prefix:identifier]] to a pkg.go.dev URL starting at pkg. For example, with
// prefix="doc" and pkg="cogentcore.org/core", the wikilink [[doc:core.Button]] will
// result in the URL "https://pkg.go.dev/cogentcore.org/core/core#Button".
func GoDocWikilink(prefix string, pkg string) WikilinkHandler {
	return func(text string) (url string, label string) {
		if !strings.HasPrefix(text, prefix+":") {
			return "", ""
		}
		text = strings.TrimPrefix(text, prefix+":")
		// pkg.go.dev uses fragments for first dot within package
		t := strings.Replace(text, ".", "#", 1)
		url = "https://pkg.go.dev/" + pkg + "/" + t
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
		// minimum: [[X]]
		if len(data) < 5 || data[1] != '[' {
			return fn(p, original, offset)
		}
		inside, after := getWikilinkText(data)
		url, label := runWikilinkHandlers(ctx, inside)

		var node ast.Node
		if len(url) == 0 && len(label) == 0 {
			slog.Error("invalid wikilink", "link", string(inside))
			// TODO: we just treat broken wikilinks like plaintext for now, but we should
			// make red links instead at some point
			node = &ast.Text{Leaf: ast.Leaf{Literal: append(inside, after...)}}
		} else {
			node = &ast.Link{Destination: url}
			ast.AppendChild(node, &ast.Text{Leaf: ast.Leaf{Literal: append(label, after...)}})
		}
		return len(inside) + len(after) + 4, node
	}
}

// getWikilinkText gets the wikilink text from the given raw text data starting with [[.
// Inside contains the text inside the [[]], and after contains all of the text
// after the ]] until there is a space or punctuation.
func getWikilinkText(data []byte) (inside, after []byte) {
	i := 2
	for ; i < len(data); i++ {
		if data[i] == ']' && data[i-1] == ']' {
			inside = data[2 : i-1]
			continue
		}
		r := rune(data[i])
		// Space or punctuation after ]] means we are done.
		if inside != nil && (unicode.IsSpace(r) || unicode.IsPunct(r)) {
			break
		}
	}
	after = data[len(inside)+4 : i]
	return
}

// runWikilinkHandlers returns the first non-blank URL and label returned
// by [Context.WikilinkHandlers].
func runWikilinkHandlers(ctx *Context, text []byte) (url, label []byte) {
	for _, h := range ctx.WikilinkHandlers {
		u, l := h(string(text))
		if u == "" && l == "" {
			continue
		}
		url, label = []byte(u), []byte(l)
		break
	}
	return
}
