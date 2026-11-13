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
}

func (n *mathNode) Kind() ast.NodeKind { return kindMath }

func (n *mathNode) Dump(source []byte, level int) { ast.DumpHelper(n, source, level, nil, nil) }

// mathParser parses $inline$ math, on one line, and $$display$$ math,
// which may span lines of a paragraph.
type mathParser struct{}

func (mathParser) Trigger() []byte { return []byte{'$'} }

func (mathParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()
	if len(line) > 1 && line[1] == '$' {
		return parseDisplayMath(block)
	}
	// Inline math has no space just inside of the dollar signs.
	rest := line[1:]
	i := bytes.IndexByte(rest, '$')
	if i <= 0 || rest[0] == ' ' || rest[i-1] == ' ' {
		return nil
	}
	block.Advance(i + 2)
	return &mathNode{Value: append([]byte{}, rest[:i]...)}
}

// parseDisplayMath parses $$display$$ math up to the next $$.
func parseDisplayMath(block text.Reader) ast.Node {
	l, pos := block.Position()
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
			block.Advance(i + 2)
			return &mathNode{Value: buf, Display: true}
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
	if n.Display {
		w.WriteString(`<span class="math display">\[`)
		w.Write(util.EscapeHTML(n.Value))
		w.WriteString(`\]</span>`)
	} else {
		w.WriteString(`<span class="math inline">\(`)
		w.Write(util.EscapeHTML(n.Value))
		w.WriteString(`\)</span>`)
	}
	return ast.WalkContinue, nil
}
