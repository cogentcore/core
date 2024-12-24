// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"fmt"

	"cogentcore.org/core/core"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

func mdToHTML(md []byte) []byte {
	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	prev := p.RegisterInline('[', nil)
	p.RegisterInline('[', wikiLink(prev))
	doc := p.Parse(md)

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}

// ReadMD reads MD (markdown) from the given bytes and adds corresponding
// Cogent Core widgets to the given [core.Widget], using the given context.
func ReadMD(ctx *Context, parent core.Widget, b []byte) error {
	htm := mdToHTML(b)
	buf := bytes.NewBuffer(htm)
	fmt.Println("got html", htm)
	return ReadHTML(ctx, parent, buf)
}

// ReadMDString reads MD (markdown) from the given string and adds
// corresponding Cogent Core widgets to the given [core.Widget], using the given context.
func ReadMDString(ctx *Context, parent core.Widget, s string) error {
	return ReadMD(ctx, parent, []byte(s))
}
