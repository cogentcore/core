// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tex

import (
	"time"

	"github.com/goki/pi/filecat"
	"github.com/goki/pi/langs/bibtex"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/pi"
	"github.com/goki/pi/syms"
)

// BibData contains the bibliography data
type BibData struct {
	File   string         `desc:"file name -- full path"`
	BibTex *bibtex.BibTex `desc:"bibtex data loaded from file"`
	Mod    time.Time      `desc:"mod time for loaded bibfile -- to detect updates"`
}

// TexLang implements the Lang interface for the Tex / LaTeX language
type TexLang struct {
	Pr   *pi.Parser
	Bibs map[string]*BibData `desc:"bibliography data that has been loaded, keyed by abs file path"`
}

// TheTexLang is the instance variable providing support for the Go language
var TheTexLang = TexLang{}

func init() {
	pi.StdLangProps[filecat.TeX].Lang = &TheTexLang
}

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
	tl.FindBibfile(fss, pfs)
	fss.EndProc() // now done
	// no parser
}

func (tl *TexLang) LexLine(fs *pi.FileState, line int, txt []rune) lex.Line {
	pr := tl.Parser()
	if pr == nil {
		return nil
	}
	return pr.LexLine(fs, line, txt)
}

func (tl *TexLang) ParseLine(fs *pi.FileState, line int) *pi.FileState {
	// n/a
	return nil
}

func (tl *TexLang) HiLine(fss *pi.FileStates, line int, txt []rune) lex.Line {
	fs := fss.Done()
	return tl.LexLine(fs, line, txt)
}

func (tl *TexLang) ParseDir(path string, opts pi.LangDirOpts) *syms.Symbol {
	// n/a
	return nil
}
