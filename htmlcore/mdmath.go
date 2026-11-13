// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// kindMath is the kind of a [mathNode].
var kindMath = ast.NewNodeKind("Math")

// mathNode is $inline$ or $$display$$ math, rendered as gomarkdown did:
// <span class="math inline">\(...\)</span> or class="math display" with \[...\].
type mathNode struct {
	ast.BaseInline
	Value   []byte
	Display bool

	// Segment is the source of the math, with its dollar signs.
	Segment text.Segment
}

func (n *mathNode) Kind() ast.NodeKind { return kindMath }

func (n *mathNode) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

// mathParser parses $inline$ math, on one line, and $$display$$ math,
// which may span lines of a paragraph.
type mathParser struct{}

func (mathParser) Trigger() []byte { return []byte{'$'} }

func (mathParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, seg := block.PeekLine()
	if len(line) > 1 && line[1] == '$' {
		return parseDisplayMath(block)
	}
	rest := line[1:]
	i := inlineMathEnd(rest)
	if i < 0 {
		return nil
	}
	block.Advance(i + 2)
	return &mathNode{Value: append([]byte{}, rest[:i]...), Segment: text.NewSegment(seg.Start, seg.Start+i+2)}
}

// inlineMathEnd returns the index in s, the text after an opening $, of
// the closing $ of inline math, or -1 if it is not math: the opening $ is
// not followed by a space, and the closing $ is not followed by a digit,
// so that "$5 and $6" is not math.
func inlineMathEnd(s []byte) int {
	if len(s) == 0 || s[0] == ' ' || s[0] == '\t' || s[0] == '$' {
		return -1
	}
	for i := 1; i < len(s); i++ {
		if s[i] == '\\' {
			i++
			continue
		}
		if s[i] == '$' {
			if i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {
				continue
			}
			return i
		}
	}
	return -1
}

// parseDisplayMath parses $$display$$ math up to the next $$.
func parseDisplayMath(block text.Reader) ast.Node {
	l, pos := block.Position()
	_, seg := block.PeekLine()
	block.Advance(2)
	var buf []byte
	for {
		line, _ := block.PeekLine()
		if line == nil {
			block.SetPosition(l, pos)
			return nil
		}
		if i := bytes.Index(line, []byte("$$")); i >= 0 {
			buf = append(buf, line[:i]...)
			_, end := block.PeekLine()
			block.Advance(i + 2)
			return &mathNode{Value: buf, Display: true, Segment: text.NewSegment(seg.Start, end.Start+i+2)}
		}
		buf = append(buf, line...)
		block.AdvanceLine()
	}
}

// renderMath renders a [mathNode].
func renderMath(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*mathNode)
	v := bytes.ReplaceAll(n.Value, []byte(mathPipe), []byte("|"))
	if n.Display {
		w.WriteString(`<span class="math display">\[`)
		w.Write(util.EscapeHTML(v))
		w.WriteString(`\]</span>`)
	} else {
		w.WriteString(`<span class="math inline">\(`)
		w.Write(util.EscapeHTML(v))
		w.WriteString(`\)</span>`)
	}
	return ast.WalkContinue, nil
}

// mathPipe stands for a | in the math of a table line (see [escapeTableMath]).
const mathPipe = "\uE000"

// escapeTableMath returns the markdown with each | inside of $math$ on a
// table line (starting with |) replaced by [mathPipe], so that a table cell
// can have math such as $|\psi\rangle$: GFM splits the cells of a row at
// every unescaped |, before it parses the inlines of the cells, and \| can
// not be used, since it is also LaTeX. [renderMath] restores the pipes.
// Lines of fenced code blocks are left as they are.
func escapeTableMath(md []byte) []byte {
	if !bytes.Contains(md, []byte("$")) || !bytes.Contains(md, []byte("|")) {
		return md
	}
	var out []byte
	fence := false
	for line := range bytes.Lines(md) {
		trim := bytes.TrimLeft(line, " \t")
		if bytes.HasPrefix(trim, []byte("```")) || bytes.HasPrefix(trim, []byte("~~~")) {
			fence = !fence
		}
		if fence || len(trim) == 0 || trim[0] != '|' || !bytes.Contains(line, []byte("$")) {
			out = append(out, line...)
			continue
		}
		out = append(out, escapeLineMath(line)...)
	}
	return out
}

// escapeLineMath replaces each | inside of the $math$ of the given line,
// with the rules of [mathParser], skipping `code` spans.
func escapeLineMath(line []byte) []byte {
	var out []byte
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case c == '\\' && i+1 < len(line):
			out = append(out, c, line[i+1])
			i++
			continue
		case c == '`':
			// A code span is copied up to its closing backtick.
			if e := bytes.IndexByte(line[i+1:], '`'); e >= 0 {
				out = append(out, line[i:i+e+2]...)
				i += e + 1
				continue
			}
		case c == '$':
			delim := 1
			if i+1 < len(line) && line[i+1] == '$' {
				delim = 2
			}
			start := i + delim
			e := -1
			if delim == 2 {
				e = bytes.Index(line[start:], []byte("$$"))
			} else {
				e = inlineMathEnd(line[start:])
			}
			if e > 0 {
				out = append(out, line[i:start]...)
				out = append(out, bytes.ReplaceAll(line[start:start+e], []byte("|"), []byte(mathPipe))...)
				out = append(out, line[i:start]...)
				i = start + e + delim - 1
				continue
			}
		}
		out = append(out, c)
	}
	return out
}

// kindMathBlock is the kind of a [mathBlock].
var kindMathBlock = ast.NewNodeKind("MathBlock")

// mathBlock is a block of $$display$$ math that starts with $$ at the start
// of a line and ends with the next $$, the raw lines of which are not
// parsed as markdown, rendered as gomarkdown did:
// <p><span class="math display">\[...\]</span></p>.
type mathBlock struct {
	ast.BaseBlock

	// closed is whether the closing $$ has been read.
	closed bool
}

func (n *mathBlock) Kind() ast.NodeKind { return kindMathBlock }

func (n *mathBlock) IsRaw() bool { return true }

func (n *mathBlock) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

// mathBlockParser parses a [mathBlock].
type mathBlockParser struct{}

func (mathBlockParser) Trigger() []byte { return []byte{'$'} }

func (mathBlockParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, seg := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 || !bytes.HasPrefix(line[pos:], []byte("$$")) {
		return nil, parser.NoChildren
	}
	if pc.LastOpenedBlock().Node != nil {
		if open, ok := pc.LastOpenedBlock().Node.(*ast.Paragraph); ok && openDisplayMath(open, reader.Source()) {
			// This $$ closes the display math of the paragraph.
			return nil, parser.NoChildren
		}
	}
	start := pos + 2
	rest := line[start:]
	n := &mathBlock{}
	if e := bytes.Index(rest, []byte("$$")); e >= 0 {
		// The math is on this line, which must have nothing after it.
		if !util.IsBlank(rest[e+2:]) {
			return nil, parser.NoChildren
		}
		n.Lines().Append(text.NewSegment(seg.Start+start, seg.Start+start+e))
		n.closed = true
		reader.AdvanceToEOL()
		return n, parser.Close
	}
	// The newline of the first line is kept, as gomarkdown did.
	n.Lines().Append(text.NewSegment(seg.Start+start, seg.Stop))
	reader.AdvanceToEOL()
	return n, parser.NoChildren
}

// openDisplayMath returns whether the paragraph has $$display$$ math that
// is not closed yet: an odd number of $$.
func openDisplayMath(p *ast.Paragraph, source []byte) bool {
	n := 0
	lines := p.Lines()
	for i := range lines.Len() {
		seg := lines.At(i)
		n += bytes.Count(seg.Value(source), []byte("$$"))
	}
	return n%2 == 1
}

func (mathBlockParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	if node.(*mathBlock).closed {
		// goldmark continues a block that Open closed.
		return parser.Close
	}
	line, seg := reader.PeekLine()
	if e := bytes.Index(line, []byte("$$")); e >= 0 {
		if e > 0 {
			node.Lines().Append(text.NewSegment(seg.Start, seg.Start+e))
		}
		reader.AdvanceToEOL()
		return parser.Close
	}
	node.Lines().Append(seg)
	reader.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}

func (mathBlockParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {}

func (mathBlockParser) CanInterruptParagraph() bool { return true }

func (mathBlockParser) CanAcceptIndentedLine() bool { return false }

// renderMathBlock renders a [mathBlock]. Its attributes are only for the
// [Context.AttributeHandlers] (as the label of an equation).
func renderMathBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	w.WriteString(`<p><span class="math display">\[`)
	lines := node.Lines()
	for i := range lines.Len() {
		seg := lines.At(i)
		w.Write(util.EscapeHTML(seg.Value(source)))
	}
	w.WriteString("\\]</span></p>\n")
	return ast.WalkContinue, nil
}
