// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package highlighting

import (
	"fmt"
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/runes"
	"cogentcore.org/core/text/parse"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/token"
	"github.com/stretchr/testify/assert"
)

func TestRich(t *testing.T) {
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
	fmt.Println(lex)

	// this "avoid" is what drives the need for depth in styles
	oix := runes.Index(rsrc, []rune("avoid"))
	fmt.Println("oix:", oix)
	ot := []lexer.Lex{lexer.Lex{Token: token.KeyToken{Token: token.TextSpellErr, Depth: 1}, Start: oix, End: oix + 5}}

	sty := rich.NewStyle()
	sty.Family = rich.Monospace
	tx := MarkupLineRich(hi.style, sty, rsrc, lex, ot)
	fmt.Println(tx)
}
