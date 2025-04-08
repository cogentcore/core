// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"io"
	"regexp"

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
	// this allows div to work properly:
	// https://github.com/gomarkdown/markdown/issues/5
	md = bytes.ReplaceAll(md, []byte("</div>"), []byte("</div><!-- dummy -->"))
	doc := p.Parse(md)

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags, RenderNodeHook: ctx.mdRenderHook}
	renderer := html.NewRenderer(opts)

	htm := markdown.Render(doc, renderer)
	htm = bytes.ReplaceAll(htm, []byte("<p></div><!-- dummy --></p>"), []byte("</div>"))
	divr := regexp.MustCompile("<p(.*)><div></p>")
	htm = divr.ReplaceAll(htm, []byte("<div${1}>"))
	return htm
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
	leaf := n.AsLeaf()
	if cont != nil {
		if cont.Attribute != nil {
			res = string(cont.Attribute.Attrs[attr])
		}
	} else if leaf != nil {
		if leaf.Attribute != nil {
			res = string(leaf.Attribute.Attrs[attr])
		}
	}
	return res
}

// MDSetAttr sets the given attribute on the given markdown node
func MDSetAttr(n ast.Node, attr, value string) {
	var attrs *ast.Attribute
	cont := n.AsContainer()
	leaf := n.AsLeaf()
	if cont != nil {
		attrs = cont.Attribute
	} else if leaf != nil {
		attrs = leaf.Attribute
	}
	if attrs == nil {
		attrs = &ast.Attribute{}
	}
	if attrs.Attrs == nil {
		attrs.Attrs = make(map[string][]byte)
	}
	attrs.Attrs[attr] = []byte(value)
	if cont != nil {
		cont.Attribute = attrs
	} else if leaf != nil {
		leaf.Attribute = attrs
	}
}
