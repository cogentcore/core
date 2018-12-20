// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pi

import (
	"github.com/goki/gi/complete"
	"github.com/goki/gi/filecat"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/syms"
)

// MarkdownLang implements the Lang interface for the Markdown language
type MarkdownLang struct {
	Pr *Parser
}

// TheMarkdownLang is the instance variable providing support for the Go language
var TheMarkdownLang = MarkdownLang{}

func (ml *MarkdownLang) Parser() *Parser {
	if ml.Pr != nil {
		return ml.Pr
	}
	lp, _ := LangSupport.Props(filecat.Markdown)
	if lp.Parser == nil {
		LangSupport.OpenStd()
	}
	ml.Pr = lp.Parser
	if ml.Pr == nil {
		return nil
	}
	ml.Pr.InitAll()
	return ml.Pr
}

func (ml *MarkdownLang) ParseFile(fs *FileState) {
	pr := ml.Parser()
	if pr == nil {
		return
	}
	pr.LexAll(fs)
	// no parser
}

func (ml *MarkdownLang) LexLine(fs *FileState, line int) lex.Line {
	pr := ml.Parser()
	if pr == nil {
		return nil
	}
	return pr.LexLine(fs, line)
}

func (ml *MarkdownLang) ParseLine(fs *FileState, line int) *FileState {
	// n/a
	return nil
}

func (ml *MarkdownLang) HiLine(fs *FileState, line int) lex.Line {
	return ml.LexLine(fs, line)
}

func (ml *MarkdownLang) CompleteLine(fs *FileState, str string, pos lex.Pos) (md complete.MatchData) {
	// n/a
	return md
}

func (ml *MarkdownLang) ParseDir(path string, opts LangDirOpts) *syms.Symbol {
	// n/a
	return nil
}
