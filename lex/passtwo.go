// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"github.com/goki/pi/token"
)

// PassTwo performs second pass(s) through the lexicalized version of the source,
// computing nesting depth for every token once and for all -- this is essential for
// properly matching tokens and also for colorization in syntax highlighting.
// Optionally, a subsequent pass finds end-of-statement (EOS) tokens, which are essential
// for parsing to first break the source down into statement-sized chunks.  A separate
// list of EOS token positions is maintained for very fast access.
type PassTwo struct {
	DoEos         bool               `desc:"should we perform EOS detection on this type of file?"`
	Eol           bool               `desc:"use end-of-line as a default EOS, if nesting depth is same as start of line (python) -- see also EolToks"`
	Semi          bool               `desc:"replace all semicolons with EOS to keep it consistent (C, Go..)"`
	Backslash     bool               `desc:"use backslash as a line continuer (python)"`
	RBraceOneLine bool               `desc:"if a right-brace } is detected at end of line, and there are multiple tokens on that line prior to right brace, insert an EOS *before* RBrace as well (needed for Go)"`
	EolToks       token.KeyTokenList `desc:"specific tokens to recognize at the end of a line that trigger an EOS (Go)"`
}

// TwoState is the state maintained for the PassTwo process
type TwoState struct {
	Pos       Pos            `desc:"position in lex tokens we're on"`
	Src       *File          `desc:"file that we're operating on"`
	NestStack []token.Tokens `desc:"stack of nesting tokens"`
	EosPos    []Pos          `desc:"positions *in token coordinates* of the EOS markers generated"`
	Errs      ErrorList      `desc:"any error messages accumulated during lexing specifically"`
}

// Init initializes state for a new pass -- called at start of NestDepth
func (ts *TwoState) Init() {
	ts.Pos = PosZero
	ts.NestStack = ts.NestStack[0:0]
	ts.EosPos = ts.EosPos[0:0]
}

// SetSrc sets the source we're operating on
func (ts *TwoState) SetSrc(src *File) {
	ts.Src = src
}

// NextLine advances to next line
func (ts *TwoState) NextLine() {
	ts.Pos.Ln++
	ts.Pos.Ch = 0
}

// InsertEOS inserts an EOS just after the given token position (e.g., cp = last token in line)
func (ts *TwoState) InsertEOS(cp Pos) {
	np := Pos{cp.Ln, cp.Ch + 1}
	elx := ts.Src.LexAt(cp)
	depth := elx.Depth
	ts.Src.Lexs[cp.Ln].Insert(np.Ch, Lex{Tok: token.EOS, Depth: depth, St: elx.Ed, Ed: elx.Ed})
	ts.EosPos = append(ts.EosPos, np)
}

// ReplaceEOS replaces given token with an EOS
func (ts *TwoState) ReplaceEOS(cp Pos) {
	clex := ts.Src.LexAt(cp)
	clex.Tok = token.EOS
	ts.EosPos = append(ts.EosPos, cp)
}

// Error adds an passtwo error at current position
func (ts *TwoState) Error(msg string) {
	ppos := ts.Pos
	ppos.Ch--
	clex := ts.Src.LexAtSafe(ppos)
	ts.Errs.Add(Pos{ts.Pos.Ln, clex.St}, ts.Src.Filename, "PassTwo: "+msg, ts.Src.SrcLine(ts.Pos.Ln), nil)
}

// NestStackStr returns the token stack as strings
func (ts *TwoState) NestStackStr() string {
	str := ""
	for _, tok := range ts.NestStack {
		switch tok {
		case token.PunctGpLParen:
			str += "paren ( "
		case token.PunctGpLBrack:
			str += "bracket [ "
		case token.PunctGpLBrace:
			str += "brace { "
		}
	}
	return str
}

/////////////////////////////////////////////////////////////////////
//  PassTwo

// Error adds an passtwo error at given position
func (pt *PassTwo) Error(ts *TwoState, msg string) {
	ts.Error(msg)
}

// HasErrs reports if there are errors in eosing process
func (pt *PassTwo) HasErrs(ts *TwoState) bool {
	return len(ts.Errs) > 0
}

// MismatchError reports a mismatch for given type of parentheses / bracket
func (pt *PassTwo) MismatchError(ts *TwoState, tok token.Tokens) {
	switch tok {
	case token.PunctGpRParen:
		pt.Error(ts, "mismatching parentheses -- right paren ')' without matching left paren '('")
	case token.PunctGpRBrack:
		pt.Error(ts, "mismatching square brackets -- right bracket ']' without matching left bracket '['")
	case token.PunctGpRBrace:
		pt.Error(ts, "mismatching curly braces -- right brace '}' without matching left bracket '{'")
	}
}

// PushNest pushes a nesting left paren / bracket onto stack
func (pt *PassTwo) PushNest(ts *TwoState, tok token.Tokens) {
	ts.NestStack = append(ts.NestStack, tok)
}

// PopNest attempts to pop given token off of nesting stack, generating error if it mismatches
func (pt *PassTwo) PopNest(ts *TwoState, tok token.Tokens) {
	sz := len(ts.NestStack)
	if sz == 0 {
		pt.MismatchError(ts, tok)
		return
	}
	cur := ts.NestStack[sz-1]
	ts.NestStack = ts.NestStack[:sz-1] // better to clear than keep even if err
	if cur != tok.PunctGpMatch() {
		pt.MismatchError(ts, tok)
	}
}

// Perform nesting depth computation
func (pt *PassTwo) NestDepth(ts *TwoState) {
	ts.Init()
	nlines := ts.Src.NLines()
	if nlines == 0 {
		return
	}
	// if len(ts.Src.Lexs[nlines-1]) > 0 { // last line ends with tokens -- parser needs empty last line..
	// 	ts.Src.Lexs = append(ts.Src.Lexs, Line{})
	// 	*ts.Src.Lines = append(*ts.Src.Lines, []rune{})
	// }
	for ts.Pos.Ln < nlines {
		sz := len(ts.Src.Lexs[ts.Pos.Ln])
		if sz == 0 {
			ts.NextLine()
			continue
		}
		lx := ts.Src.LexAt(ts.Pos)
		tok := lx.Tok
		if tok.IsPunctGpLeft() {
			lx.Depth = len(ts.NestStack) // depth increments AFTER -- this turns out to be ESSENTIAL!
			pt.PushNest(ts, tok)
		} else if tok.IsPunctGpRight() {
			pt.PopNest(ts, tok)
			lx.Depth = len(ts.NestStack) // end has same depth as start, which is same as SURROUND
		} else {
			lx.Depth = len(ts.NestStack)
		}
		ts.Pos.Ch++
		if ts.Pos.Ch >= sz {
			ts.NextLine()
		}
	}
	stsz := len(ts.NestStack)
	if stsz > 0 {
		pt.Error(ts, "mismatched grouping -- end of file with these left unmatched: "+ts.NestStackStr())
	}
}

// Perform nesting depth computation on only one line, starting at
// given initial depth -- updates the given line
func (pt *PassTwo) NestDepthLine(line Line, initDepth int) {
	sz := len(line)
	if sz == 0 {
		return
	}
	depth := initDepth
	for i := 0; i < sz; i++ {
		lx := &line[i]
		tok := lx.Tok
		if tok.IsPunctGpLeft() {
			lx.Depth = depth
			depth++
		} else if tok.IsPunctGpRight() {
			depth--
			lx.Depth = depth
		} else {
			lx.Depth = depth
		}
	}
}

// Perform EOS detection
func (pt *PassTwo) EosDetect(ts *TwoState) {
	ts.Pos = PosZero
	nlines := ts.Src.NLines()
	for ts.Pos.Ln < nlines {
		sz := len(ts.Src.Lexs[ts.Pos.Ln])
		if sz == 0 {
			ts.NextLine()
			continue
		}
		ep := Pos{ts.Pos.Ln, sz - 1} // end of line token
		elx := ts.Src.LexAt(ep)
		if pt.Eol {
			sp := Pos{ts.Pos.Ln, 0} // start of line token
			slx := ts.Src.LexAt(sp)
			if slx.Depth == elx.Depth {
				ts.InsertEOS(ep)
			}
		}
		if len(pt.EolToks) > 0 { // not depth specific
			etkey := token.KeyToken{Tok: elx.Tok}
			if elx.Tok.IsKeyword() {
				etkey.Key = string(ts.Src.TokenSrc(ep))
			}
			if pt.EolToks.Match(etkey) {
				ts.InsertEOS(ep)
				if pt.RBraceOneLine && elx.Tok == token.PunctGpRBrace && sz > 2 {
					ts.InsertEOS(Pos{ts.Pos.Ln, sz - 2})
				}
			}
		}
		if pt.Semi {
			for ts.Pos.Ch = 0; ts.Pos.Ch < sz; ts.Pos.Ch++ {
				lx := ts.Src.LexAt(ts.Pos)
				if lx.Tok == token.PunctSepSemicolon {
					ts.ReplaceEOS(ts.Pos)
				}
			}
		}
		ts.NextLine()
	}
}
