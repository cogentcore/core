// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"errors"
	"io/fs"
	"os"

	"cogentcore.org/core/content/bcontent"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// readIndexPage returns the HTML for the optional custom index page of the app,
// which is an index.html file or an index.md file (converted from markdown) in
// the app directory, with index.html taking precedence. It replaces the
// automatically pre-rendered HTML preview of the root page in the generated
// index.html file (see [makeIndexHTML]). The result is an HTML fragment that is
// placed inside of the app loader element, not a complete HTML document, so that
// the app keeps loading in the background while the user reads it. It returns ""
// if the app does not have a custom index page.
func readIndexPage() (string, error) {
	b, err := os.ReadFile("index.html")
	if err == nil {
		return string(b), nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	// [bcontent.Page.ReadContent] removes any TOML front matter, which is
	// ignored here but supported so that content pages can be used directly.
	pg := &bcontent.Page{Source: os.DirFS("."), Filename: "index.md"}
	b, err = pg.ReadContent(nil)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(mdToHTML(b)), nil
}

// mdToHTML converts the given markdown to HTML for a custom index page (see
// [readIndexPage]). Unlike [htmlcore.MDToHTML], this is a plain markdown
// conversion without support for Cogent Core extensions such as wikilinks,
// since those require the GUI packages, which the build tool can not import.
func mdToHTML(md []byte) []byte {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock | parser.Attributes | parser.Mmark
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)
	opts := html.RendererOptions{Flags: html.CommonFlags | html.HrefTargetBlank}
	return markdown.Render(doc, html.NewRenderer(opts))
}
