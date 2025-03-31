// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"io"

	"cogentcore.org/core/core"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

func mdToHTML(ctx *Context, md []byte) []byte {
	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock | parser.Attributes | parser.Mmark
	p := parser.NewWithExtensions(extensions)
	prev := p.RegisterInline('[', nil)
	p.RegisterInline('[', wikilink(ctx, prev))
	doc := p.Parse(md)

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags, RenderNodeHook: ctx.mdRenderHook}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}

// ReadMD reads MD (markdown) from the given bytes and adds corresponding
// Cogent Core widgets to the given [core.Widget], using the given context.
func ReadMD(ctx *Context, parent core.Widget, b []byte) error {
	htm := mdToHTML(ctx, b)
	// os.WriteFile("htmlcore_tmp.html", htm, 0666)
	buf := bytes.NewBuffer(htm)
	return ReadHTML(ctx, parent, buf)
}

// ReadMDString reads MD (markdown) from the given string and adds
// corresponding Cogent Core widgets to the given [core.Widget], using the given context.
func ReadMDString(ctx *Context, parent core.Widget, s string) error {
	return ReadMD(ctx, parent, []byte(s))
}

func (ctx *Context) attrRenderHooks(attr *ast.Attribute, w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	for tag, val := range attr.Attrs {
		f, has := ctx.AttributeHandlers[tag]
		if has {
			b := f(ctx, w, node, entering, tag, string(val))
			return ast.GoToNext, b
		}
	}
	return ast.GoToNext, false
}

func (ctx *Context) mdRenderHook(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	cont := node.AsContainer()
	if cont != nil && cont.Attribute != nil {
		return ctx.attrRenderHooks(cont.Attribute, w, node, entering)
	}
	leaf := node.AsLeaf()
	if leaf != nil && leaf.Attribute != nil {
		return ctx.attrRenderHooks(leaf.Attribute, w, node, entering)
	}
	return ast.GoToNext, false
}

// MDGetAttr gets the given attribute from the given markdown node, returning ""
// if the attribute is not found.
func MDGetAttr(n ast.Node, attr string) string {
	res := ""
	cont := n.AsContainer()
	if cont != nil {
		if cont.Attribute != nil {
			res = string(cont.Attribute.Attrs[attr])
		}
	} else {
		leaf := n.AsLeaf()
		if leaf != nil {
			if leaf.Attribute != nil {
				res = string(leaf.Attribute.Attrs[attr])
			}
		}
	}
	return res
}
