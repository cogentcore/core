// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pi

import (
	"github.com/goki/pi/complete"
	"github.com/goki/pi/filecat"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/pi"
	"github.com/goki/pi/syms"
)

// TexLang implements the Lang interface for the Tex / LaTeX language
type TexLang struct {
	Pr *pi.Parser
}

// TheTexLang is the instance variable providing support for the Go language
var TheTexLang = TexLang{}

func (tl *TexLang) Parser() *pi.Parser {
	if tl.Pr != nil {
		return tl.Pr
	}
	lp, _ := pi.LangSupport.Props(filecat.TeX)
	if lp.Parser == nil {
		pi.LangSupport.OpenStd()
	}
	tl.Pr = lp.Parser
	if tl.Pr == nil {
		return nil
	}
	return tl.Pr
}

func (tl *TexLang) ParseFile(fss *pi.FileStates, txt []byte) {
	pr := tl.Parser()
	if pr == nil {
		return
	}
	pfs := fss.StartProc(txt) // current processing one
	pr.LexAll(pfs)
	fss.EndProc() // now done
	// no parser
}

func (tl *TexLang) LexLine(fs *pi.FileState, line int) lex.Line {
	pr := tl.Parser()
	if pr == nil {
		return nil
	}
	return pr.LexLine(fs, line)
}

func (tl *TexLang) ParseLine(fs *pi.FileState, line int) *pi.FileState {
	// n/a
	return nil
}

func (tl *TexLang) HiLine(fss *pi.FileStates, line int) lex.Line {
	fs := fss.Done()
	return tl.LexLine(fs, line)
}

func (tl *TexLang) CompleteLine(fss *pi.FileStates, str string, pos lex.Pos) (md complete.Matches) {
	// n/a
	return md
}

// Lookup is the main api called by completion code in giv/complete.go to lookup item
func (gl *TexLang) Lookup(fss *pi.FileStates, str string, pos lex.Pos) (ld complete.Lookup) {
	return
}

func (tl *TexLang) CompleteEdit(fss *pi.FileStates, text string, cp int, comp complete.Completion, seed string) (ed complete.Edit) {
	// n/a
	return ed
}

func (tl *TexLang) ParseDir(path string, opts pi.LangDirOpts) *syms.Symbol {
	// n/a
	return nil
}
