// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"testing"

	"cogentcore.org/core/base/fileinfo"
	_ "cogentcore.org/core/system/driver"
	"cogentcore.org/core/text/highlighting"
	"cogentcore.org/core/text/parse"
	"github.com/stretchr/testify/assert"
)

func TestMarkup(t *testing.T) {
	src := `func (ls *Lines) deleteTextRectImpl(st, ed textpos.Pos) *textpos.Edit {
	tbe := ls.regionRect(st, ed)
	if tbe == nil {
	return nil
	}
`

	lns := &Lines{}
	lns.Defaults()
	lns.width = 40

	fi, err := fileinfo.NewFileInfo("dummy.go")
	assert.Error(t, err)
	var pst parse.FileStates
	pst.SetSrc("dummy.go", "", fi.Known)
	pst.Done()

	lns.Highlighter.Init(fi, &pst)
	lns.Highlighter.SetStyle(highlighting.HighlightingName("emacs"))
	lns.Highlighter.Has = true
	assert.Equal(t, true, lns.Highlighter.UsingParse())

	lns.SetText([]byte(src))
	assert.Equal(t, src+"\n", string(lns.Bytes()))

	mu := `[monospace]: ""
[monospace bold fill-color]: "func "
[monospace]: " ("
[monospace]: "ls "
[monospace fill-color]: " *"
[monospace]: "Lines"
[monospace]: ") "
[monospace]: " deleteTextRectImpl"
[monospace]: "(
"
[monospace]: "st"
[monospace]: ", "
[monospace]: " ed "
[monospace]: " textpos"
[monospace]: "."
[monospace]: "Pos"
[monospace]: ") "
[monospace fill-color]: " *"
[monospace]: "textpos"
[monospace]: "."
[monospace]: "Edit "
[monospace]: " {"
`
	assert.Equal(t, 1, lns.nbreaks[0])
	assert.Equal(t, mu, lns.markup[0].String())
}
