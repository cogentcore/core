// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pi

import (
	"github.com/goki/gi/complete"
	"github.com/goki/gi/filecat"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/pi"
	"github.com/goki/pi/syms"
)

// MarkdownLang implements the Lang interface for the Markdown language
type MarkdownLang struct {
	Pr *pi.Parser
}

// TheMarkdownLang is the instance variable providing support for the Markdown language
var TheMarkdownLang = MarkdownLang{}

func init() {
	pi.StdLangProps[filecat.Markdown].Lang = &TheMarkdownLang
}

func (ml *MarkdownLang) Parser() *pi.Parser {
	if ml.Pr != nil {
		return ml.Pr
	}
	lp, _ := pi.LangSupport.Props(filecat.Markdown)
	if lp.Parser == nil {
		pi.LangSupport.OpenStd()
	}
	ml.Pr = lp.Parser
	if ml.Pr == nil {
		return nil
	}
	return ml.Pr
}

func (ml *MarkdownLang) ParseFile(fs *pi.FileState) {
	pr := ml.Parser()
	if pr == nil {
		return
	}
	pr.LexAll(fs)
	// no parser
}

func (ml *MarkdownLang) LexLine(fs *pi.FileState, line int) lex.Line {
	pr := ml.Parser()
	if pr == nil {
		return nil
	}
	return pr.LexLine(fs, line)
}

func (ml *MarkdownLang) ParseLine(fs *pi.FileState, line int) *pi.FileState {
	// n/a
	return nil
}

func (ml *MarkdownLang) HiLine(fs *pi.FileState, line int) lex.Line {
	return ml.LexLine(fs, line)
}

func (ml *MarkdownLang) CompleteLine(fs *pi.FileState, str string, pos lex.Pos) (md complete.MatchData) {
	// n/a
	return md
}

func (ml *MarkdownLang) CompleteEdit(fs *pi.FileState, text string, cp int, comp complete.Completion, seed string) (ed complete.EditData) {
	// n/a
	return ed
}

func (ml *MarkdownLang) ParseDir(path string, opts pi.LangDirOpts) *syms.Symbol {
	// n/a
	return nil
}
