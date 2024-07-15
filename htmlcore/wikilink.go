// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
)

// WikilinkBaseURL is the base URL to use for wiki links
var WikilinkBaseURL = "cogentcore.org/core"

// note: this is from: https://github.com/kensanata/oddmu/blob/main/parser.go

// wikiLink returns an inline parser function. This indirection is
// required because we want to call the previous definition in case
// this is not a wikiLink.
func wikiLink(fn func(p *parser.Parser, data []byte, offset int) (int, ast.Node)) func(p *parser.Parser, data []byte, offset int) (int, ast.Node) {
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
		// pkg.go.dev uses fragments for first dot within package
		t := bytes.Replace(text, []byte{'.'}, []byte{'#'}, 1)
		dest := append([]byte("https://pkg.go.dev/"+WikilinkBaseURL+"/"), t...)
		link := &ast.Link{
			Destination: dest,
		}
		ast.AppendChild(link, &ast.Text{Leaf: ast.Leaf{Literal: text}})
		return i + 3, link
	}
}
