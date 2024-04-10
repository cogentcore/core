// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textbuf

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/fileinfo"
	"cogentcore.org/core/pi"
	"cogentcore.org/core/xgo/indent"
)

// Opts contains options for textview.Bufs
// Contains everything necessary to conditionalize editing
// of a given text file.
type Opts struct {

	// editor settings from core settings
	core.EditorSettings

	// character(s) that start a single-line comment -- if empty then multi-line comment syntax will be used
	CommentLn string

	// character(s) that start a multi-line comment or one that requires both start and end
	CommentSt string

	// character(s) that end a multi-line comment or one that requires both start and end
	CommentEd string
}

// CommentStrs returns the comment start and end strings, using line-based CommentLn first if set
// and falling back on multi-line / general purpose start / end syntax
func (tb *Opts) CommentStrs() (comst, comed string) {
	comst = tb.CommentLn
	if comst == "" {
		comst = tb.CommentSt
		comed = tb.CommentEd
	}
	return
}

// IndentChar returns the indent character based on SpaceIndent option
func (tb *Opts) IndentChar() indent.Char {
	if tb.SpaceIndent {
		return indent.Space
	}
	return indent.Tab
}

// ConfigKnown configures options based on the supported language info in GoPi
// returns true if supported
func (tb *Opts) ConfigKnown(sup fileinfo.Known) bool {
	if sup == fileinfo.Unknown {
		return false
	}
	lp, ok := pi.StandardLangProperties[sup]
	if !ok {
		return false
	}
	tb.CommentLn = lp.CommentLn
	tb.CommentSt = lp.CommentSt
	tb.CommentEd = lp.CommentEd
	for _, flg := range lp.Flags {
		switch flg {
		case pi.IndentSpace:
			tb.SpaceIndent = true
		case pi.IndentTab:
			tb.SpaceIndent = false
		}
	}
	return true
}

// KnownComments returns the comment strings for supported file types,
// and returns the standard C-style comments otherwise.
func KnownComments(fpath string) (comLn, comSt, comEd string) {
	comLn = "//"
	comSt = "/*"
	comEd = "*/"
	mtyp, _, err := fileinfo.MimeFromFile(fpath)
	if err != nil {
		return
	}
	sup := fileinfo.MimeKnown(mtyp)
	if sup == fileinfo.Unknown {
		return
	}
	lp, ok := pi.StandardLangProperties[sup]
	if !ok {
		return
	}
	comLn = lp.CommentLn
	comSt = lp.CommentSt
	comEd = lp.CommentEd
	return
}
