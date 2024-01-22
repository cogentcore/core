// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package coredom

import (
	"bytes"
	"fmt"

	"cogentcore.org/core/gi"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/wikilink"
)

// ReadMD reads MD (markdown) from the given bytes and adds corresponding
// Cogent Core widgets to the given [gi.Widget], using the given context.
func ReadMD(ctx *Context, par gi.Widget, b []byte) error {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			&wikilink.Extender{WikilinkResolver{}},
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
	return ReadHTML(ctx, par, &buf)
}

// ReadMDString reads MD (markdown) from the given string and adds
// corresponding Cogent Core widgets to the given [gi.Widget], using the given context.
func ReadMDString(ctx *Context, par gi.Widget, s string) error {
	return ReadMD(ctx, par, []byte(s))
}

// WikilinkResolver implements [wikilink.Resolver] by using pkg.go.dev.
type WikilinkResolver struct{}

func (wr WikilinkResolver) ResolveWikilink(n *wikilink.Node) (destination []byte, err error) {
	return append([]byte("https://pkg.go.dev/cogentcore.org/core/"), n.Target...), nil
}
