// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package highlighting

import (
	"testing"

	"cogentcore.org/core/base/fileinfo"
	_ "cogentcore.org/core/text/parse/supportedlanguages"
	"github.com/stretchr/testify/assert"
)

// The tests use the parse languages for syntax highlighting,
// which their expected results were made with.
func init() {
	UseParse = true
}

func TestUseParseOff(t *testing.T) {
	// With UseParse off, Go code is highlighted with chroma even though
	// the parse languages are registered.
	UseParse = false
	defer func() { UseParse = true }()
	fi, err := fileinfo.NewFileInfo("highlighter.go")
	assert.NoError(t, err)
	hi := &Highlighter{}
	hi.SetStyle(HighlightingName("emacs"))
	hi.Init(fi, nil)
	assert.True(t, hi.Has)
	assert.False(t, hi.UsingParse())
	assert.Nil(t, ParseLanguage(fileinfo.Go))
}
