// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textbuf

import (
	"github.com/goki/gi/gi"
	"github.com/goki/ki/indent"
	"github.com/goki/pi/filecat"
	"github.com/goki/pi/pi"
)

// Opts contains options for TextBufs -- contains everything necessary to
// conditionalize editing of a given text file
type Opts struct {
	gi.EditorPrefs `desc:"editor prefs from gogi prefs"`
	CommentLn      string `desc:"character(s) that start a single-line comment -- if empty then multi-line comment syntax will be used"`
	CommentSt      string `desc:"character(s) that start a multi-line comment or one that requires both start and end"`
	CommentEd      string `desc:"character(s) that end a multi-line comment or one that requires both start and end"`
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
