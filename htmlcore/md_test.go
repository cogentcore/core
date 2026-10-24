// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"testing"

	"cogentcore.org/core/core"
	"github.com/stretchr/testify/assert"
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

// TestReplaceDivs tests that replaceDivs replaces as divRegex does.
func TestReplaceDivs(t *testing.T) {
	tests := []string{
		"",
		"<p><div></p>",
		`<p class="a"><div></p>x<p id="b"><div></p>`,
		"<p>a\n<p class=\"c\"><div></p>",
		"<pre>code</pre><p><div></p> and <p><div></p><div></p>",
		"<p\n><div></p>",
		"no paragraphs",
		"<p>only</p>",
		"<p><p><div></p></p>",
		"<path d='x'/><div></p>\n<p x><div></p>",
	}
	for _, s := range tests {
		want := divRegex.ReplaceAll([]byte(s), []byte("<div${1}>"))
		if got := replaceDivs([]byte(s)); string(got) != string(want) {
			t.Errorf("%q: got %q, want %q", s, got, want)
		}
	}
}

// TestLineBreakHTML checks that the line breaks of a backslash at the end of a
// line and of <br> are kept in the text and in the HTML generated from it.
func TestLineBreakHTML(t *testing.T) {
	b := core.NewBody()
	assert.NoError(t, ReadMDString(NewContext(), b, "A\\\nB\\\nC\n\nD<br>E\n"))
	h, err := core.ToHTML(b)
	assert.NoError(t, err)
	assert.Contains(t, string(h), ">A<br>B<br>C</p>")
	assert.Contains(t, string(h), ">D<br>E</p>")
}
