// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"regexp"
	"strconv"
	"unicode"

	"cogentcore.org/core/core"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// NewMarkdown returns a goldmark converter with the extensions used for
// Cogent Core content: GitHub flavored markdown, heading ids, raw HTML,
// block attributes in a {...} line before a block, links opening in a new
// tab, and, if ctx is non-nil, wikilinks and its AttributeHandlers.
func NewMarkdown(ctx *Context) goldmark.Markdown {
	// The parser has the GFM syntax, heading ids, and block attributes.
	popts := []parser.Option{
		parser.WithAutoHeadingID(),
		parser.WithAttribute(),
		parser.WithBlockParsers(util.Prioritized(attrBlockParser{}, 50), util.Prioritized(mathBlockParser{}, 60)),
		parser.WithASTTransformers(util.Prioritized(attrTransformer{}, 100)),
		parser.WithInlineParsers(util.Prioritized(mathParser{}, 500)),
	}
	if ctx != nil {
		popts = append(popts, parser.WithInlineParsers(util.Prioritized(wikilinkParser{ctx}, 199)))
	}

	// The renderer writes raw HTML and the attributes of all blocks.
	ropts := []renderer.Option{
		html.WithUnsafe(),
		renderer.WithNodeRenderers(util.Prioritized(&hookRenderer{ctx: ctx}, 100)),
	}
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Typographer),
		goldmark.WithParserOptions(popts...),
		goldmark.WithRendererOptions(ropts...),
	)
}

// MDToHTML converts the given markdown to HTML, using the given context
// for wikilinks and attribute handlers.
func MDToHTML(ctx *Context, md []byte) []byte {
	var buf bytes.Buffer
	pc := parser.NewContext(parser.WithIDs(&headingIDs{values: map[string]bool{}}))
	md = escapeTableMath(md)
	if err := ctx.converter().Convert(md, &buf, parser.WithContext(pc)); err != nil {
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
func (ctx *Context) converter() goldmark.Markdown {
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
type headingIDs struct {
	values map[string]bool
}

func (s *headingIDs) Generate(value []byte, kind ast.NodeKind) []byte {
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
	res := string(id)
	if res == "" {
		res = "heading"
	}
	for i, base := 1, res; s.values[res]; i++ {
		res = base + "-" + strconv.Itoa(i)
	}
	s.values[res] = true
	return []byte(res)
}

func (s *headingIDs) Put(value []byte) { s.values[string(value)] = true }

// kindAttrBlock is the kind of an [attrBlock].
var kindAttrBlock = ast.NewNodeKind("AttrBlock")

// attrBlock is a line with only {...} attributes, which apply to the
// next block (the mmark / gomarkdown block attribute syntax).
type attrBlock struct {
	ast.BaseBlock
}

func (n *attrBlock) Kind() ast.NodeKind { return kindAttrBlock }

func (n *attrBlock) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

// attrBlockParser parses a line of only {...} attributes into an [attrBlock].
type attrBlockParser struct{}

func (attrBlockParser) Trigger() []byte { return []byte{'{'} }

func (attrBlockParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	// The line must be the attributes alone.
	line, _ := reader.PeekLine()
	if len(bytes.TrimSpace(line)) < 2 || bytes.TrimSpace(line)[len(bytes.TrimSpace(line))-1] != '}' {
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

// attrTransformer moves the attributes of each [attrBlock] to the next block.
type attrTransformer struct{}

func (attrTransformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	var blocks []ast.Node
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && n.Kind() == kindAttrBlock {
			blocks = append(blocks, n)
		}
		return ast.WalkContinue, nil
	})
	for _, n := range blocks {
		if next := n.NextSibling(); next != nil {
			for _, a := range n.Attributes() {
				next.SetAttribute(a.Name, a.Value)
			}
		}
		n.Parent().RemoveChild(n.Parent(), n)
	}
}

// hookRenderer renders the blocks with attributes: it runs the
// [Context.AttributeHandlers], and it writes all of the attributes of
// code blocks on their code element, as gomarkdown did.
type hookRenderer struct {
	ctx  *Context
	base map[ast.NodeKind]renderer.NodeRendererFunc
}

// hookKinds are the kinds of blocks that the [hookRenderer] handles.
var hookKinds = []ast.NodeKind{ast.KindParagraph, ast.KindHeading, ast.KindFencedCodeBlock, ast.KindCodeBlock, ast.KindBlockquote, ast.KindList, ast.KindThematicBreak, ast.KindHTMLBlock}

// funcMap is a NodeRendererFuncRegisterer that records the functions.
type funcMap map[ast.NodeKind]renderer.NodeRendererFunc

func (m funcMap) Register(k ast.NodeKind, f renderer.NodeRendererFunc) { m[k] = f }

func (r *hookRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	// The default renderer functions are called for blocks without handlers.
	r.base = map[ast.NodeKind]renderer.NodeRendererFunc{}
	html.NewRenderer(html.WithUnsafe()).RegisterFuncs(funcMap(r.base))
	reg.Register(ast.KindFencedCodeBlock, r.renderCode)
	reg.Register(ast.KindImage, r.renderImage)
	reg.Register(ast.KindLink, r.renderLink)
	reg.Register(kindMath, renderMath)
	reg.Register(kindMathBlock, r.wrap(renderMathBlock))
	// Tables are rendered by the GFM table renderer.
	tr := map[ast.NodeKind]renderer.NodeRendererFunc{}
	extension.NewTableHTMLRenderer().RegisterFuncs(funcMap(tr))
	reg.Register(east.KindTable, r.wrap(tr[east.KindTable]))
	for _, k := range hookKinds {
		if k == ast.KindFencedCodeBlock {
			continue
		}
		reg.Register(k, r.wrap(r.base[k]))
	}
}

// runHandlers runs the attribute handler for the first attribute of
// the node that has one, returning whether it handled the node.
func (r *hookRenderer) runHandlers(w util.BufWriter, n ast.Node, entering bool) bool {
	if r.ctx == nil || len(r.ctx.AttributeHandlers) == 0 {
		return false
	}
	for _, a := range n.Attributes() {
		if f, has := r.ctx.AttributeHandlers[string(a.Name)]; has {
			return f(r.ctx, w, n, entering, string(a.Name), attrString(a.Value))
		}
	}
	return false
}

func (r *hookRenderer) wrap(f renderer.NodeRendererFunc) renderer.NodeRendererFunc {
	return func(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
		if n.Attributes() != nil && r.runHandlers(w, n, entering) {
			return ast.WalkContinue, nil
		}
		return f(w, source, n, entering)
	}
}

// renderCode renders a fenced code block with all of its attributes.
func (r *hookRenderer) renderCode(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if node.Attributes() != nil && r.runHandlers(w, node, entering) {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.FencedCodeBlock)
	if !entering {
		w.WriteString("</code></pre>\n")
		return ast.WalkContinue, nil
	}
	w.WriteString("<pre><code")
	if lang := n.Language(source); lang != nil {
		w.WriteString(` class="language-`)
		w.Write(util.EscapeHTML(lang))
		w.WriteByte('"')
	}
	if n.Attributes() != nil {
		html.RenderAttributes(w, n, nil)
	}
	w.WriteByte('>')
	for i := 0; i < n.Lines().Len(); i++ {
		line := n.Lines().At(i)
		w.Write(util.EscapeHTML(line.Value(source)))
	}
	return ast.WalkContinue, nil
}

// renderImage renders an image with its alt text as written, including
// any markdown or HTML in it (the caption of a figure), as gomarkdown did.
func (r *hookRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Image)
	w.WriteString(`<img src="`)
	w.Write(util.EscapeHTML(util.URLEscape(n.Destination, true)))
	w.WriteString(`" alt="`)
	start, stop := -1, -1
	ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		var segs []text.Segment
		switch c := c.(type) {
		case *ast.Text:
			segs = []text.Segment{c.Segment}
		case *ast.RawHTML:
			segs = c.Segments.Sliced(0, c.Segments.Len())
		case *mathNode:
			segs = []text.Segment{c.Segment}
		}
		for _, sg := range segs {
			if start < 0 || sg.Start < start {
				start = sg.Start
			}
			stop = max(stop, sg.Stop)
		}
		return ast.WalkContinue, nil
	})
	if start >= 0 {
		w.Write(util.EscapeHTML(smartPunct(source[start:stop])))
	}
	w.WriteByte('"')
	if n.Title != nil {
		w.WriteString(` title="`)
		w.Write(util.EscapeHTML(n.Title))
		w.WriteByte('"')
	}
	if n.Attributes() != nil {
		html.RenderAttributes(w, n, html.ImageAttributeFilter)
	}
	w.WriteString(">")
	return ast.WalkSkipChildren, nil
}

// renderLink renders a link, with target="_blank" for an absolute URL,
// as gomarkdown did with HrefTargetBlank. The destination is not URL
// escaped, since wikilink handlers return URLs with spaces.
func (r *hookRenderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Link)
	if !entering {
		w.WriteString("</a>")
		return ast.WalkContinue, nil
	}
	w.WriteString(`<a href="`)
	w.Write(util.EscapeHTML(n.Destination))
	w.WriteByte('"')
	if n.Title != nil {
		w.WriteString(` title="`)
		w.Write(util.EscapeHTML(n.Title))
		w.WriteByte('"')
	}
	if bytes.Contains(n.Destination, []byte("://")) {
		w.WriteString(` target="_blank"`)
	}
	if n.Attributes() != nil {
		html.RenderAttributes(w, n, html.LinkAttributeFilter)
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

// attrString returns the given attribute value as a string.
func attrString(v any) string {
	switch v := v.(type) {
	case []byte:
		return string(v)
	case string:
		return v
	}
	return ""
}

// MDGetAttr gets the given attribute from the given markdown node, returning ""
// if the attribute is not found.
func MDGetAttr(n ast.Node, attr string) string {
	v, ok := n.AttributeString(attr)
	if !ok {
		return ""
	}
	return attrString(v)
}

// MDSetAttr sets the given attribute on the given markdown node
func MDSetAttr(n ast.Node, attr, value string) {
	n.SetAttributeString(attr, []byte(value))
}
