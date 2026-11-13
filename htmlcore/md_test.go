// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"io"
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore/tohtml"
	"github.com/stretchr/testify/assert"
	"github.com/yuin/goldmark/ast"
)

func TestMD(t *testing.T) {
	tests := map[string]string{
		"h1":   `# Test`,
		"h2":   `## Test`,
		"p":    `Test`,
		`code`: "```go\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n```",
	}
	for nm, s := range tests {
		b := core.NewBody()
		assert.NoError(t, ReadMDString(NewContext(), b, s))
		b.AssertRender(t, "md/"+nm)
	}
}

func TestDoubleButtonBug(t *testing.T) {
	b := core.NewBody()
	assert.NoError(t, ReadMDString(NewContext(), b, `<button>A</button><button>B</button>`))
	b.AssertRender(t, "md/double-button-bug")
}

func TestExtraNewlineBug(t *testing.T) {
	b := core.NewBody()
	assert.NoError(t, ReadMDString(NewContext(), b, `A

<button>B</button>`))
	b.AssertRender(t, "md/extra-newline-bug")
}

func TestButtonInHeadingBug(t *testing.T) {
	// TODO: this does not work correctly due to a minor bug in [extractText]
	// (see the comment there).
	b := core.NewBody()
	assert.NoError(t, ReadMDString(NewContext(), b, `<h1><button>A</button>B</h1>`))
	b.AssertRender(t, "md/button-in-heading-bug")
}

// TestLineBreakHTML checks that the line breaks of a backslash at the end of a
// line and of <br> are kept in the text and in the HTML generated from it.
func TestLineBreakHTML(t *testing.T) {
	b := core.NewBody()
	assert.NoError(t, ReadMDString(NewContext(), b, "A\\\nB\\\nC\n\nD<br>E\n"))
	h, err := tohtml.Widget(b)
	assert.NoError(t, err)
	assert.Contains(t, string(h), ">A<br>B<br>C</p>")
	assert.Contains(t, string(h), ">D<br>E</p>")
}

// TestMDToHTML checks the HTML of the Cogent Core markdown extensions.
func TestMDToHTML(t *testing.T) {
	ctx := NewContext()
	ctx.AddWikilinkHandler(func(text string) (string, string) {
		if text == "bad" {
			return "", ""
		}
		return "page/" + text, text
	})
	var handled []string
	ctx.AttributeHandlers["label"] = func(ctx *Context, w io.Writer, node ast.Node, entering bool, tag, value string) bool {
		if entering {
			handled = append(handled, node.Kind().String()+":"+value+":"+MDGetAttr(node, "id"))
		}
		return false
	}
	tests := []struct{ md, want string }{
		// attributes before a code block are on its code element
		{"{collapsed=\"true\" title=\"T\"}\n```Go\nx := 1\n```", `<pre><code class="language-Go" collapsed="true" title="T">x := 1
</code></pre>`},
		// attributes of a paragraph that is just an image are of the image
		{"{id=\"fig\" style=\"height:10em\"}\n![A <i>cap</i>](a.png)", `<p><img id="fig" style="height:10em" src="a.png" alt="A &lt;i&gt;cap&lt;/i&gt;"/></p>`},
		{"See [[the page]]s and [[bad]].", `<p>See <a href="page/the page">the pages</a> and bad.</p>`},
		{"[a](https://x.org) [b](b)", `<p><a href="https://x.org" target="_blank">a</a> <a href="b">b</a></p>`},
		{"So $x^2$ and $$\ny = 1\n$$", `<p>So <span class="math inline">\(x^2\)</span> and <span class="math display">\[
y = 1
\]</span></p>`},
		{"### Control+Shift", `<h3 id="control-shift">Control+Shift</h3>`},
		{"A $5 and $6 fee.", `<p>A $5 and $6 fee.</p>`},
		{"Faces $k = \\pm 1 $ here.", `<p>Faces <span class="math inline">\(k = \pm 1 \)</span> here.</p>`},
		// figure captions get typographic punctuation, except in tags and math
		{"![A \"b\" -- it's ''c'' <a href=\"x\">d</a> $x'$](a.png)", "<p><img src=\"a.png\" alt=\"A \u201cb\u201d \u2013 it\u2019s \u201cc\u201d &lt;a href=&quot;x&quot;&gt;d&lt;/a&gt; $x'$\"></p>"},
		// pipes in math do not split table cells; currency does not start math
		{"| a | b |\n|---|---|\n| $|\\psi\\rangle$ | $\\|x\\|$ |\n| $5 | $6 |", `<table>
<thead>
<tr>
<th>a</th>
<th>b</th>
</tr>
</thead>
<tbody>
<tr>
<td><span class="math inline">\(|\psi\rangle\)</span></td>
<td><span class="math inline">\(\|x\|\)</span></td>
</tr>
<tr>
<td>$5</td>
<td>$6</td>
</tr>
</tbody>
</table>`},
		// a $$ block is raw math, without markdown
		{"Text\n$$\n- a_1 \\\\\n_b_ & c\n$$\nafter", "<p>Text</p>\n<p><span class=\"math display\">\\[\n- a_1 \\\\\n_b_ &amp; c\n\\]</span></p>\n<p>after</p>"},
		{"A:\n$$x = 1$$\nso b.", "<p>A:</p>\n<p><span class=\"math display\">\\[x = 1\\]</span></p>\n<p>so b.</p>"},
		// the attributes of an equation are only for its handlers
		{"{id=\"eq_a\" title=\"A\"}\n$$\nx = 1\n$$", `<p><span class="math display">\[
x = 1
\]</span></p>`},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, strings.TrimSpace(string(MDToHTML(ctx, []byte(tt.md)))), tt.md)
	}

	// attribute handlers run for tables and other blocks
	MDToHTML(ctx, []byte("{id=\"t1\" label=\"Table\"}\n| a | b |\n|---|---|\n| 1 | 2 |\n\n{label=\"Para\"}\nText"))
	assert.Equal(t, []string{"Table:Table:t1", "Paragraph:Para:"}, handled)
}
