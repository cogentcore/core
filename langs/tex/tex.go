// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tex

import (
	"github.com/goki/gi/complete"
	"github.com/goki/gi/filecat"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/pi"
	"github.com/goki/pi/syms"
)

// TexLang implements the Lang interface for the Tex / LaTeX language
type TexLang struct {
	Pr *pi.Parser
}

// TheTexLang is the instance variable providing support for the TeX language
var TheTexLang = TexLang{}

func init() {
	pi.StdLangProps[filecat.TeX].Lang = &TheTexLang
}

func (ml *TexLang) Parser() *pi.Parser {
	if ml.Pr != nil {
		return ml.Pr
	}
	lp, _ := pi.LangSupport.Props(filecat.TeX)
	if lp.Parser == nil {
		pi.LangSupport.OpenStd()
	}
	ml.Pr = lp.Parser
	if ml.Pr == nil {
		return nil
	}
	return ml.Pr
}

func (ml *TexLang) ParseFile(fs *pi.FileState) {
	pr := ml.Parser()
	if pr == nil {
		return
	}
	pr.LexAll(fs)
	// no parser
}

func (ml *TexLang) LexLine(fs *pi.FileState, line int) lex.Line {
	pr := ml.Parser()
	if pr == nil {
		return nil
	}
	return pr.LexLine(fs, line)
}

func (ml *TexLang) ParseLine(fs *pi.FileState, line int) *pi.FileState {
	// n/a
	return nil
}

func (ml *TexLang) HiLine(fs *pi.FileState, line int) lex.Line {
	return ml.LexLine(fs, line)
}

func (ml *TexLang) CompleteLine(fs *pi.FileState, str string, pos lex.Pos) (md complete.MatchData) {
	// n/a
	return md
}

func (ml *TexLang) ParseDir(path string, opts pi.LangDirOpts) *syms.Symbol {
	// n/a
	return nil
}
