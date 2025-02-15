// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package highlighting

import (
	"fmt"
	"testing"

	"cogentcore.org/core/base/fileinfo"
	_ "cogentcore.org/core/system/driver"
	"cogentcore.org/core/text/parse"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/runes"
	"cogentcore.org/core/text/token"
	"github.com/stretchr/testify/assert"
)

func TestMarkup(t *testing.T) {

	src := `	if len(txt) > maxLineLen { // avoid overflow`
	rsrc := []rune(src)

	fi, err := fileinfo.NewFileInfo("dummy.go")
	assert.Error(t, err)

	var pst parse.FileStates
	pst.SetSrc("dummy.go", "", fi.Known)

	hi := Highlighter{}
	hi.Init(fi, &pst)
	hi.SetStyle(HighlightingName("emacs"))

	fs := pst.Done() // initialize
	fs.Src.SetBytes([]byte(src))

	lex, err := hi.MarkupTagsLine(0, rsrc)
	assert.NoError(t, err)

	hitrg := `[{NameFunction: if 1 3 {0 0}} {NameBuiltin 4 7 {0 0}} {PunctGpLParen 7 8 {0 0}} {+1:Name 8 11 {0 0}} {PunctGpRParen 11 12 {0 0}} {OpRelGreater 13 14 {0 0}} {Name 15 25 {0 0}} {PunctGpLBrace 26 27 {0 0}} {+1:EOS 27 27 {0 0}} {+1:Comment 28 45 {0 0}}]`
	assert.Equal(t, hitrg, fmt.Sprint(lex))
	// fmt.Println(lex)

	// this "avoid" is what drives the need for depth in styles
	// we're marking it as misspelled
	aix := runes.Index(rsrc, []rune("avoid"))
	ot := []lexer.Lex{lexer.Lex{Token: token.KeyToken{Token: token.TextSpellErr, Depth: 1}, Start: aix, End: aix + 5}}

	// todo: it doesn't detect the offset of the embedded avoid token here!

	sty := rich.NewStyle()
	sty.Family = rich.Monospace
	tx := MarkupLineRich(hi.Style, sty, rsrc, lex, ot)

	rtx := `[monospace]: "	"
[monospace fill-color]: "if "
[monospace fill-color]: " len"
[monospace]: "("
[monospace]: "txt"
[monospace]: ") "
[monospace fill-color]: " > "
[monospace]: " maxLineLen "
[monospace]: " {"
[monospace]: " "
[monospace italic fill-color]: " // "
[monospace italic fill-color]: "avoid"
[monospace italic fill-color]: " overflow"
`
	// fmt.Println(tx)
	assert.Equal(t, rtx, fmt.Sprint(tx))

	rht := `	<span class="nf">if</span> <span class="nb">len</span><span class="">(</span><span class="n">txt</span><span class="">)</span> <span class="">></span> <span class="n">maxLineLen</span> <span class="">{</span><span class="EOS"></span> <span class="c">// <span class="te">avoid</span> overflow</span>`

	b := MarkupLineHTML(rsrc, lex, ot, NoEscapeHTML)
	assert.Equal(t, rht, fmt.Sprint(string(b)))

}
