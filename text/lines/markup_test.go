// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"testing"

	_ "cogentcore.org/core/system/driver"
	"github.com/stretchr/testify/assert"
)

func TestMarkup(t *testing.T) {
	src := `func (ls *Lines) deleteTextRectImpl(st, ed textpos.Pos) *textpos.Edit {
	tbe := ls.regionRect(st, ed)
	if tbe == nil {
	return nil
	}
`

	lns, vid := NewLinesFromBytes("dummy.go", 40, []byte(src))
	vw := lns.view(vid)
	assert.Equal(t, src+"\n", lns.String())

	mu0 := `[monospace bold fill-color]: "func"
[monospace]: " ("
[monospace]: "ls"
[monospace fill-color]: " *"
[monospace]: "Lines"
[monospace]: ")"
[monospace]: " deleteTextRectImpl"
[monospace]: "("
[monospace]: "st"
[monospace]: ","
[monospace]: " "
`
	mu1 := `[monospace]: "ed"
[monospace]: " textpos"
[monospace]: "."
[monospace]: "Pos"
[monospace]: ")"
[monospace fill-color]: " *"
[monospace]: "textpos"
[monospace]: "."
[monospace]: "Edit"
[monospace]: " {"
`
	// fmt.Println(vw.markup[0])
	assert.Equal(t, mu0, vw.markup[0].String())

	// fmt.Println(vw.markup[1])
	assert.Equal(t, mu1, vw.markup[1].String())
}

func TestLineWrap(t *testing.T) {
	src := `The [rich.Text](http://rich.text.com) type is the standard representation for formatted text, used as the input to the "shaped" package for text layout and rendering. It is encoded purely using "[]rune" slices for each span, with the _style_ information **represented** with special rune values at the start of each span. This is an efficient and GPU-friendly pure-value format that avoids any issues of style struct pointer management etc.
`

	lns, vid := NewLinesFromBytes("dummy.md", 80, []byte(src))
	vw := lns.view(vid)
	assert.Equal(t, src+"\n", lns.String())

	tmu := []string{`[monospace]: "The "
[monospace fill-color]: "[rich.Text]"
[monospace fill-color]: "(http://rich.text.com)"
[monospace]: " type is the standard representation for "
`,

		`[monospace]: "formatted text, used as the input to the "
[monospace fill-color]: ""shaped""
[monospace]: " package for text layout and "
`,

		`[monospace]: "rendering. It is encoded purely using "
[monospace fill-color]: ""[]rune""
[monospace]: " slices for each span, with the"
[monospace italic]: " "
`,

		`[monospace italic]: "_style_"
[monospace]: " information"
[monospace bold]: " **represented**"
[monospace]: " with special rune values at the start of "
`,

		`[monospace]: "each span. This is an efficient and GPU-friendly pure-value format that avoids "
`,
		`[monospace]: "any issues of style struct pointer management etc."
`,
	}

	join := `The [rich.Text](http://rich.text.com) type is the standard representation for 
formatted text, used as the input to the "shaped" package for text layout and 
rendering. It is encoded purely using "[]rune" slices for each span, with the 
_style_ information **represented** with special rune values at the start of 
each span. This is an efficient and GPU-friendly pure-value format that avoids 
any issues of style struct pointer management etc.
`
	assert.Equal(t, 6, vw.viewLines)

	jtxt := ""
	for i := range vw.viewLines {
		trg := tmu[i]
		// fmt.Println(vw.markup[i])
		assert.Equal(t, trg, vw.markup[i].String())
		jtxt += string(vw.markup[i].Join()) + "\n"
	}
	// fmt.Println(jtxt)
	assert.Equal(t, join, jtxt)
}

func TestMarkupSpaces(t *testing.T) {
	src := `Name           string
`

	lns, vid := NewLinesFromBytes("dummy.go", 40, []byte(src))
	vw := lns.view(vid)
	assert.Equal(t, src+"\n", lns.String())

	mu0 := `[monospace]: "Name           "
[monospace bold fill-color]: "string"
`
	// fmt.Println(lns.markup[0])
	// fmt.Println(vw.markup[0])
	assert.Equal(t, mu0, vw.markup[0].String())
}
