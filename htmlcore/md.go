// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"io"
	"regexp"
	"strings"
	"unicode"

	"cogentcore.org/core/core"
	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/extension"
	east "github.com/yuin/goldmark/v2/extension/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/renderer"
	"github.com/yuin/goldmark/v2/renderer/html"
	"github.com/yuin/goldmark/v2/text"
	"github.com/yuin/goldmark/v2/util"
)

// Markdown is a goldmark markdown parser and HTML renderer.
type Markdown struct {
	Parser   parser.Parser
	Renderer html.Renderer
}

// Convert converts the given markdown to HTML, written to w.
func (md *Markdown) Convert(source []byte, w io.Writer) error {
	return md.Renderer.Render(w, source, md.Parser.Parse(source))
}

// NewMarkdown returns a goldmark converter with the extensions used for
// Cogent Core content: GitHub flavored markdown, heading ids, raw HTML,
// block attributes in a {...} line before a block, links opening in a new
// tab, and, if ctx is non-nil, wikilinks and its AttributeHandlers.
func NewMarkdown(ctx *Context) *Markdown {
	// The parser has the GFM syntax, heading ids, and block attributes.
	popts := []parser.Option{
		parser.WithExtensions(extension.NewGFMParser(), extension.NewTypographerParser()),
		parser.WithAutoHeadingID(),
		parser.WithAttribute(),
		parser.WithIDGenerator(headingIDs{}),
		parser.WithBlockParsers(util.Prioritized[parser.BlockParser](attrBlockParser{}, 50), util.Prioritized[parser.BlockParser](mathBlockParser{}, 60)),
		parser.WithASTTransformers(util.Prioritized[parser.ASTTransformer](attrTransformer{}, 100)),
		parser.WithInlineParsers(util.Prioritized[parser.InlineParser](mathParser{}, 500)),
	}
	if ctx != nil {
		popts = append(popts, parser.WithInlineParsers(util.Prioritized[parser.InlineParser](wikilinkParser{ctx}, 199)))
	}

	// The renderer writes raw HTML, and the attribute handlers run before
	// the default renderers of blocks.
	r := &hookRenderer{ctx: ctx}
	decorators := map[ast.NodeKind]html.NodeRendererDecorator{}
	for _, k := range hookKinds {
		decorators[k] = r.decorate
	}
	ropts := []html.Option{
		html.WithUnsafe(),
		html.WithExtensions(extension.NewGFMHTMLRenderer()),
		html.WithNodeRenderers(map[ast.NodeKind]html.NodeRenderer{
			ast.KindCodeBlock: html.NodeRendererFunc(renderCode),
			ast.KindImage:     html.NodeRendererFunc(renderImage),
			ast.KindLink:      html.NodeRendererFunc(renderLink),
			kindMath:          html.NodeRendererFunc(renderMath),
			kindMathBlock:     html.NodeRendererFunc(renderMathBlock),
		}),
		html.WithNodeRendererDecorators(decorators),
	}
	return &Markdown{Parser: parser.New(popts...), Renderer: html.New(ropts...)}
}

// MDToHTML converts the given markdown to HTML, using the given context
// for wikilinks and attribute handlers.
func MDToHTML(ctx *Context, md []byte) []byte {
	var buf bytes.Buffer
	md = escapeTableMath(md)
	if err := ctx.converter().Convert(md, &buf); err != nil {
		core.ErrorSnackbar(nil, err)
	}
	htm := buf.Bytes()
	if bytes.Contains(htm, []byte("><img ")) {
		// The attributes of a paragraph that is just an image, as
		// {id="figure_one" style="height:10em"} before it, are of the image.
		htm = imageParagraph.ReplaceAll(htm, []byte("<p><img${1}${2}/></p>"))
	}
	return htm
}

// converter returns the goldmark converter of the context, made once on
// first use, or a new one without Cogent Core extensions for a nil context.
func (ctx *Context) converter() *Markdown {
	if ctx == nil {
		return NewMarkdown(nil)
	}
	ctx.markdownOnce.Do(func() { ctx.markdown = NewMarkdown(ctx) })
	return ctx.markdown
}

// imageParagraph matches a paragraph with attributes that is just an image.
var imageParagraph = regexp.MustCompile(`<p((?: [a-z-]+="[^"]*")+)><img((?: [a-z-]+="[^"]*")*) ?/?></p>`)

// ReadMD reads MD (markdown) from the given bytes and adds corresponding
// Cogent Core widgets to the given [core.Widget], using the given context.
func ReadMD(ctx *Context, parent core.Widget, b []byte) error {
	htm := MDToHTML(ctx, b)
	// os.WriteFile("htmlcore_tmp.html", htm, 0666) // note: keep here, needed for debugging
	buf := bytes.NewBuffer(htm)
	return ReadHTML(ctx, parent, buf)
}

// ReadMDString reads MD (markdown) from the given string and adds
// corresponding Cogent Core widgets to the given [core.Widget], using the given context.
func ReadMDString(ctx *Context, parent core.Widget, s string) error {
	return ReadMD(ctx, parent, []byte(s))
}

// headingIDs generates heading ids as gomarkdown did: lowercase letters
// and numbers, with a dash for each run of other characters between them.
// The parser adds -1, -2, ... to repeated ids.
type headingIDs struct{}

func (headingIDs) Generate(value []byte, kind ast.NodeKind) []byte {
	var id []rune
	dash := false
	for _, r := range string(value) {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			dash = true
			continue
		}
		if dash && len(id) > 0 {
			id = append(id, '-')
		}
		dash = false
		id = append(id, unicode.ToLower(r))
	}
	if len(id) == 0 {
		return []byte("heading")
	}
	return []byte(string(id))
}

// kindAttrBlock is the kind of an [attrBlock].
var kindAttrBlock = ast.NewNodeKind("AttrBlock")

// attrBlock is a line with only {...} attributes, which apply to the
// next block (the mmark / gomarkdown block attribute syntax).
type attrBlock struct {
	ast.BaseBlock
}

func (n *attrBlock) Kind() ast.NodeKind { return kindAttrBlock }

func (n *attrBlock) Dump(source []byte) *ast.NodeDump { return ast.NewNodeDump(n, nil) }

// attrBlockParser parses a line of only {...} attributes into an [attrBlock].
type attrBlockParser struct{}

func (attrBlockParser) Trigger() []byte { return []byte{'{'} }

func (attrBlockParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	// The line must be the attributes alone.
	line, _ := reader.PeekLine()
	trim := bytes.TrimSpace(line)
	if len(trim) < 2 || trim[len(trim)-1] != '}' {
		return nil, parser.NoChildren
	}
	l, pos := reader.Position()
	attrs, ok := parser.ParseAttributes(reader)
	rest, _ := reader.PeekLine()
	if !ok || len(util.TrimRightSpace(rest)) != 0 {
		reader.SetPosition(l, pos)
		return nil, parser.NoChildren
	}
	reader.Advance(len(rest) - 1) // the parser advances past the newline
	n := &attrBlock{}
	n.Init(n)
	for _, a := range attrs {
		n.SetAttribute(a.Name, a.Value)
	}
	return n, parser.Close
}

func (attrBlockParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return parser.Close
}

func (attrBlockParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {}

func (attrBlockParser) CanInterruptParagraph() bool { return false }

func (attrBlockParser) CanAcceptIndentedLine() bool { return false }

// attrTransformer moves the attributes of each [attrBlock] to the next
// block. It also makes the values of all attributes strings, so that they
// do not need the source (for [MDGetAttr] and the attribute handlers).
type attrTransformer struct{}

func (attrTransformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	source := reader.Source()
	var blocks []ast.Node
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if n.Kind() == kindAttrBlock {
			blocks = append(blocks, n)
		}
		for _, a := range n.Attributes() {
			n.SetAttribute(a.Name, text.NewMultiLineValueFromString(a.Value.Value(source), text.IdentityDecoder))
		}
		return ast.WalkContinue, nil
	})
	for _, n := range blocks {
		if next := n.NextSibling(); next != nil {
			for _, a := range n.Attributes() {
				next.SetAttribute(a.Name, a.Value)
			}
		}
		n.Parent().RemoveChild(n)
	}
}

// hookRenderer runs the [Context.AttributeHandlers] for blocks with
// attributes, before their renderers.
type hookRenderer struct {
	ctx *Context
}

// hookKinds are the kinds of blocks that the [hookRenderer] handles.
var hookKinds = []ast.NodeKind{ast.KindParagraph, ast.KindHeading, ast.KindCodeBlock, ast.KindBlockquote, ast.KindList, ast.KindThematicBreak, ast.KindHTMLBlock, east.KindTable, kindMathBlock}

// runHandlers runs the attribute handler for the first attribute of
// the node that has one, returning whether it handled the node.
func (r *hookRenderer) runHandlers(w io.Writer, n ast.Node, entering bool) bool {
	if r.ctx == nil || len(r.ctx.AttributeHandlers) == 0 {
		return false
	}
	for _, a := range n.Attributes() {
		if f, has := r.ctx.AttributeHandlers[a.Name]; has {
			return f(r.ctx, w, n, entering, a.Name, a.Value.Value(nil))
		}
	}
	return false
}

// decorate returns a renderer that runs the attribute handlers before next.
func (r *hookRenderer) decorate(next html.NodeRenderer) html.NodeRenderer {
	return html.NodeRendererFunc(func(w io.Writer, source []byte, n ast.Node, entering bool, rc renderer.Context) (ast.WalkStatus, error) {
		if n.Attributes() != nil && r.runHandlers(w, n, entering) {
			return ast.WalkContinue, nil
		}
		return next.Render(w, source, n, entering, rc)
	})
}

// renderCode renders a code block with all of its attributes on its code
// element, as gomarkdown did.
func renderCode(writer io.Writer, source []byte, node ast.Node, entering bool, rc renderer.Context) (ast.WalkStatus, error) {
	w := writer.(util.BufWriter)
	n := node.(*ast.CodeBlock)
	if !entering {
		w.WriteString("</code></pre>\n")
		return ast.WalkContinue, nil
	}
	w.WriteString("<pre><code")
	if lang, ok := n.Language(source); ok {
		w.WriteString(` class="language-`)
		html.ContextTextWriter(rc).WriteString(lang)
		w.WriteByte('"')
	}
	if n.Attributes() != nil {
		html.RenderAttributes(w, source, n, nil, rc)
	}
	w.WriteByte('>')
	n.Value.WriteTo(html.ContextTextWriter(rc), source)
	return ast.WalkContinue, nil
}

// renderImage renders an image with its alt text as written, including
// any markdown or HTML in it (the caption of a figure), as gomarkdown did.
func renderImage(writer io.Writer, source []byte, node ast.Node, entering bool, rc renderer.Context) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	w := writer.(util.BufWriter)
	n := node.(*ast.Image)
	w.WriteString(`<img src="`)
	html.ContextLinkURLWriter(rc).WriteString(n.Destination.Value(source))
	w.WriteString(`" alt="`)
	html.ContextTextWriter(rc).Write(smartPunct(imageAlt(source, n.Pos())))
	w.WriteByte('"')
	if !n.Title.IsEmpty() {
		w.WriteString(` title="`)
		n.Title.WriteTo(html.ContextTextWriter(rc), source)
		w.WriteByte('"')
	}
	if n.Attributes() != nil {
		html.RenderAttributes(w, source, n, html.ImageAttributeFilter, rc)
	}
	w.WriteString(">")
	return ast.WalkSkipChildren, nil
}

// imageAlt returns the source of the alt text of the image at pos (the
// position of its "!["): the text up to the matching "]".
func imageAlt(source []byte, pos int) []byte {
	if pos < 0 || !bytes.HasPrefix(source[pos:], []byte("![")) {
		return nil
	}
	start := pos + 2
	depth := 0
	for i := start; i < len(source); i++ {
		switch source[i] {
		case '\\':
			i++
		case '`':
			// A code span is skipped up to its closing backtick.
			if e := bytes.IndexByte(source[i+1:], '`'); e >= 0 {
				i += e + 1
			}
		case '[':
			depth++
		case ']':
			if depth == 0 {
				return source[start:i]
			}
			depth--
		}
	}
	return nil
}

// renderLink renders a link, with target="_blank" for an absolute URL,
// as gomarkdown did with HrefTargetBlank. The destination is not URL
// escaped, since wikilink handlers return URLs with spaces.
func renderLink(writer io.Writer, source []byte, node ast.Node, entering bool, rc renderer.Context) (ast.WalkStatus, error) {
	w := writer.(util.BufWriter)
	n := node.(*ast.Link)
	if !entering {
		w.WriteString("</a>")
		return ast.WalkContinue, nil
	}
	dest := n.Destination.Value(source)
	w.WriteString(`<a href="`)
	html.ContextTextWriter(rc).WriteString(dest)
	w.WriteByte('"')
	if !n.Title.IsEmpty() {
		w.WriteString(` title="`)
		n.Title.WriteTo(html.ContextTextWriter(rc), source)
		w.WriteByte('"')
	}
	if strings.Contains(dest, "://") {
		w.WriteString(` target="_blank"`)
	}
	if n.Attributes() != nil {
		html.RenderAttributes(w, source, n, html.LinkAttributeFilter, rc)
	}
	w.WriteByte('>')
	return ast.WalkContinue, nil
}

// smartPunct returns the text with the typographic punctuation of the
// Typographer extension (as gomarkdown's Smartypants did for image alt
// text, the caption of a figure): dashes for -- and ---, an ellipsis, and
// curly quotes. HTML tags, `code`, and $math$ are left as they are.
func smartPunct(b []byte) []byte {
	var out []byte
	// quote returns the opening quote o or the closing one c for a quote at i.
	quote := func(i int, o, c string) string {
		if i == 0 || bytes.IndexByte([]byte(" \t\n([>"), b[i-1]) >= 0 {
			return o
		}
		return c
	}
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch c {
		case '<', '`', '$':
			// Tags, code, and math are copied as they are.
			end := map[byte]byte{'<': '>', '`': '`', '$': '$'}[c]
			if e := bytes.IndexByte(b[i+1:], end); e >= 0 {
				out = append(out, b[i:i+e+2]...)
				i += e + 1
				continue
			}
		case '-':
			if bytes.HasPrefix(b[i:], []byte("---")) {
				out = append(out, "\u2014"...)
				i += 2
				continue
			}
			if bytes.HasPrefix(b[i:], []byte("--")) {
				out = append(out, "\u2013"...)
				i++
				continue
			}
		case '.':
			if bytes.HasPrefix(b[i:], []byte("...")) {
				out = append(out, "\u2026"...)
				i += 2
				continue
			}
		case '"':
			out = append(out, quote(i, "\u201c", "\u201d")...)
			continue
		case '\'':
			if i+1 < len(b) && b[i+1] == '\'' { // '' is a double quote
				out = append(out, quote(i, "\u201c", "\u201d")...)
				i++
				continue
			}
			out = append(out, quote(i, "\u2018", "\u2019")...)
			continue
		}
		out = append(out, c)
	}
	return out
}

// MDGetAttr gets the given attribute from the given markdown node, returning ""
// if the attribute is not found.
func MDGetAttr(n ast.Node, attr string) string {
	v, ok := n.Attribute(attr)
	if !ok {
		return ""
	}
	return v.Value(nil)
}

// MDSetAttr sets the given attribute on the given markdown node
func MDSetAttr(n ast.Node, attr, value string) {
	n.SetAttribute(attr, text.NewMultiLineValueFromString(value, text.IdentityDecoder))
}
