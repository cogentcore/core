// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"github.com/goki/pi/lex"
	"github.com/goki/pi/token"
)

// parse.State is the state maintained for parsing
type State struct {
	Src      *lex.File     `desc:"source and lexed version of source we're parsing"`
	Ast      *Ast          `desc:"root of the Ast abstract syntax tree we're updating"`
	EosPos   []lex.Pos     `desc:"positions *in token coordinates* of the EOS markers generated"`
	EosIdx   int           `desc:"index in list of Eos tokens that we're currently on"`
	Pos      lex.Pos       `desc:"the current lex token position"`
	RegStack []lex.Reg     `desc:"region *in token coordinates* of start / end positions for looking for tokens"`
	State    []string      `desc:"state stack"`
	Errs     lex.ErrorList `desc:"any error messages accumulated during parsing specifically"`
}

// Init initializes the state at start of parsing
func (ps *State) Init(src *lex.File, ast *Ast, eospos []lex.Pos) {
	ps.Src = src
	ps.Ast = ast
	ps.Ast.DeleteChildren(true)
	ps.EosPos = eospos
	ps.State = nil
	ps.Pos, _ = ps.Src.ValidTokenPos(lex.PosZero)
	ps.EosIdx = 0
	ps.Errs.Reset()
}

// Error adds a parsing error at given lex token position
func (ps *State) Error(pos lex.Pos, msg string) {
	if pos != lex.PosZero {
		pos = ps.Src.TokenSrcPos(pos).St
	}
	ps.Errs.Add(pos, ps.Src.Filename, "Parser: "+msg)
}

// AtEof returns true if current position is at end of file
func (ps *State) AtEof() bool {
	return ps.Pos.Ln >= ps.Src.NLines()
}

// FindToken looks for token in given region, returns position where found, false if not
// all positions in token indexes
func (ps *State) FindToken(tok token.Tokens, keyword string, reg lex.Reg) (lex.Pos, bool) {
	cp, ok := ps.Src.ValidTokenPos(reg.St)
	if !ok {
		return cp, false
	}
	isCat := tok.Cat() == tok
	isSubCat := tok.SubCat() == tok
	for cp.IsLess(reg.Ed) {
		tk := ps.Src.Token(cp)
		if tk == tok || (isCat && tk.Cat() == tok) || (isSubCat && tk.SubCat() == tok) {
			if keyword != "" {
				tksrc := string(ps.Src.TokenSrc(cp))
				if tksrc == keyword {
					return cp, true
				}
			} else {
				return cp, true
			}
		}
		ok := false
		cp, ok = ps.Src.NextTokenPos(cp)
		if !ok {
			return cp, false
		}
	}
	return cp, false
}

// AddAst adds a child Ast node to given Ast parent node -- if nil uses global parent
func (ps *State) AddAst(ast *Ast, par *Rule, rule string, reg lex.Reg) (parAst, chAst *Ast) {
	if ast == nil || ast.Name() != par.Name() {
		parAst = ps.Ast.AddNewChild(KiT_Ast, par.Name()).(*Ast)
		parAst.SetTokReg(reg, ps.Src)
		chAst = parAst.AddNewChild(KiT_Ast, rule).(*Ast)
		chAst.SetTokReg(reg, ps.Src)
		return
	}
	if ast.Name() == rule { // recursive
		parAst = ast.Par.(*Ast)
	} else {
		parAst = ast
	}
	chAst = parAst.AddNewChild(KiT_Ast, rule).(*Ast)
	chAst.SetTokReg(reg, ps.Src)
	return
}

func (ps *State) PushState(st string) {
	ps.State = append(ps.State, st)
}

func (ps *State) CurState() string {
	sz := len(ps.State)
	if sz == 0 {
		return ""
	}
	return ps.State[sz-1]
}

func (ps *State) PopState() string {
	sz := len(ps.State)
	if sz == 0 {
		return ""
	}
	st := ps.CurState()
	ps.State = ps.State[:sz-1]
	return st
}

func (ps *State) PushReg(reg lex.Reg) {
	ps.RegStack = append(ps.RegStack, reg)
}

func (ps *State) CurReg() (lex.Reg, bool) {
	sz := len(ps.RegStack)
	if sz == 0 {
		return lex.Reg{}, false
	}
	return ps.RegStack[sz-1], true
}

func (ps *State) PopReg() (lex.Reg, bool) {
	sz := len(ps.RegStack)
	if sz == 0 {
		return lex.Reg{}, false
	}
	st, _ := ps.CurReg()
	ps.RegStack = ps.RegStack[:sz-1]
	return st, true
}
