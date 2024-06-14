// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"fmt"

	"cogentcore.org/core/core"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/goldmark/wikilink"
)

// ReadMD reads MD (markdown) from the given bytes and adds corresponding
// Cogent Core widgets to the given [core.Widget], using the given context.
func ReadMD(ctx *Context, parent core.Widget, b []byte) error {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			&wikilink.Extender{ctx},
			highlighting.NewHighlighting(
				highlighting.WithStyle(string(core.AppearanceSettings.HiStyle)),
				highlighting.WithWrapperRenderer(HighlightingWrapperRenderer),
			),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
	var buf bytes.Buffer
	err := md.Convert(b, &buf)
	if err != nil {
		return fmt.Errorf("error parsing MD (markdown): %w", err)
	}
	return ReadHTML(ctx, parent, &buf)
}

// ReadMDString reads MD (markdown) from the given string and adds
// corresponding Cogent Core widgets to the given [core.Widget], using the given context.
func ReadMDString(ctx *Context, parent core.Widget, s string) error {
	return ReadMD(ctx, parent, []byte(s))
}

// HighlightingWrapperRenderer is the [highlighting.WrapperRenderer] for markdown rendering
// that enables pages runnable Go code examples to work correctly.
func HighlightingWrapperRenderer(w util.BufWriter, context highlighting.CodeBlockContext, entering bool) {
	lang, ok := context.Language()
	if !ok || string(lang) != "Go" {
		return
	}
	if entering {
		w.WriteString("<pages-example>")
	} else {
		w.WriteString("</pages-example>")
	}
}
