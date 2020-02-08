// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textbuf

import (
	"github.com/goki/ki/indent"
	"github.com/goki/pi/filecat"
	"github.com/goki/pi/pi"
)

// Opts contains options for TextBufs -- contains everything necessary to
// conditionalize editing of a given text file
type Opts struct {
	SpaceIndent  bool   `desc:"use spaces, not tabs, for indentation -- tab-size property in TextStyle has the tab size, used for either tabs or spaces"`
	TabSize      int    `desc:"size of a tab, in chars -- also determines indent level for space indent"`
	AutoIndent   bool   `desc:"auto-indent on newline (enter) or tab"`
	LineNos      bool   `desc:"show line numbers at left end of editor"`
	Completion   bool   `desc:"use the completion system to suggest options while typing"`
	SpellCorrect bool   `desc:"use spell checking to suggest corrections while typing"`
	EmacsUndo    bool   `desc:"use emacs-style undo, where after a non-undo command, all the current undo actions are added to the undo stack, such that a subsequent undo is actually a redo"`
	DepthColor   bool   `desc:"colorize the background according to nesting depth"`
	CommentLn    string `desc:"character(s) that start a single-line comment -- if empty then multi-line comment syntax will be used"`
	CommentSt    string `desc:"character(s) that start a multi-line comment or one that requires both start and end"`
	CommentEd    string `desc:"character(s) that end a multi-line comment or one that requires both start and end"`
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

// ConfigSupported configures options based on the supported language info in GoPi
// returns true if supported
func (tb *Opts) ConfigSupported(sup filecat.Supported) bool {
	if sup == filecat.NoSupport {
		return false
	}
	lp, ok := pi.StdLangProps[sup]
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

// SupportedComments returns the comment strings for supported file types,
// and returns the standard C-style comments otherwise.
func SupportedComments(fpath string) (comLn, comSt, comEd string) {
	comLn = "//"
	comSt = "/*"
	comEd = "*/"
	mtyp, _, err := filecat.MimeFromFile(fpath)
	if err != nil {
		return
	}
	sup := filecat.MimeSupported(mtyp)
	if sup == filecat.NoSupport {
		return
	}
	lp, ok := pi.StdLangProps[sup]
	if !ok {
		return
	}
	comLn = lp.CommentLn
	comSt = lp.CommentSt
	comEd = lp.CommentEd
	return
}
