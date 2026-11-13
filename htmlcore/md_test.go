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
		{"A $5 and $ 6 fee.", `<p>A $5 and $ 6 fee.</p>`},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, strings.TrimSpace(string(MDToHTML(ctx, []byte(tt.md)))), tt.md)
	}

	// attribute handlers run for tables and other blocks
	MDToHTML(ctx, []byte("{id=\"t1\" label=\"Table\"}\n| a | b |\n|---|---|\n| 1 | 2 |\n\n{label=\"Para\"}\nText"))
	assert.Equal(t, []string{"Table:Table:t1", "Paragraph:Para:"}, handled)
}
