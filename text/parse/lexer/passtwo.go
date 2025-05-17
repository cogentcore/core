// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer

import (
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
)

// PassTwo performs second pass(s) through the lexicalized version of the source,
// computing nesting depth for every token once and for all -- this is essential for
// properly matching tokens and also for colorization in syntax highlighting.
// Optionally, a subsequent pass finds end-of-statement (EOS) tokens, which are essential
// for parsing to first break the source down into statement-sized chunks.  A separate
// list of EOS token positions is maintained for very fast access.
type PassTwo struct {

	// should we perform EOS detection on this type of file?
	DoEos bool

	// use end-of-line as a default EOS, if nesting depth is same as start of line (python) -- see also EolToks
	Eol bool

	// replace all semicolons with EOS to keep it consistent (C, Go..)
	Semi bool

	// use backslash as a line continuer (python)
	Backslash bool

	// if a right-brace } is detected anywhere in the line, insert an EOS *before* RBrace AND after it (needed for Go) -- do not include RBrace in EolToks in this case
	RBraceEos bool

	// specific tokens to recognize at the end of a line that trigger an EOS (Go)
	EolToks token.KeyTokenList
}

// TwoState is the state maintained for the PassTwo process
type TwoState struct {

	// position in lex tokens we're on
	Pos textpos.Pos

	// file that we're operating on
	Src *File

	// stack of nesting tokens
	NestStack []token.Tokens

	// any error messages accumulated during lexing specifically
	Errs ErrorList
}

// Init initializes state for a new pass -- called at start of NestDepth
func (ts *TwoState) Init() {
	ts.Pos = textpos.PosZero
	ts.NestStack = ts.NestStack[0:0]
}

// SetSrc sets the source we're operating on
func (ts *TwoState) SetSrc(src *File) {
	ts.Src = src
}

// NextLine advances to next line
func (ts *TwoState) NextLine() {
	ts.Pos.Line++
	ts.Pos.Char = 0
}

// Error adds an passtwo error at current position
func (ts *TwoState) Error(msg string) {
	ppos := ts.Pos
	ppos.Char--
	clex := ts.Src.LexAtSafe(ppos)
	ts.Errs.Add(textpos.Pos{ts.Pos.Line, clex.Start}, ts.Src.Filename, "PassTwo: "+msg, ts.Src.SrcLine(ts.Pos.Line), nil)
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
	for ts.Pos.Line < nlines {
		sz := len(ts.Src.Lexs[ts.Pos.Line])
		if sz == 0 {
			ts.NextLine()
			continue
		}
		lx := ts.Src.LexAt(ts.Pos)
		tok := lx.Token.Token
		if tok.IsPunctGpLeft() {
			lx.Token.Depth = len(ts.NestStack) // depth increments AFTER -- this turns out to be ESSENTIAL!
			pt.PushNest(ts, tok)
		} else if tok.IsPunctGpRight() {
			pt.PopNest(ts, tok)
			lx.Token.Depth = len(ts.NestStack) // end has same depth as start, which is same as SURROUND
		} else {
			lx.Token.Depth = len(ts.NestStack)
		}
		ts.Pos.Char++
		if ts.Pos.Char >= sz {
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
		tok := lx.Token.Token
		if tok.IsPunctGpLeft() {
			lx.Token.Depth = depth
			depth++
		} else if tok.IsPunctGpRight() {
			depth--
			lx.Token.Depth = depth
		} else {
			lx.Token.Depth = depth
		}
	}
}

// Perform EOS detection
func (pt *PassTwo) EosDetect(ts *TwoState) {
	nlines := ts.Src.NLines()
	pt.EosDetectPos(ts, textpos.PosZero, nlines)
}

// Perform EOS detection at given starting position, for given number of lines
func (pt *PassTwo) EosDetectPos(ts *TwoState, pos textpos.Pos, nln int) {
	ts.Pos = pos
	nlines := ts.Src.NLines()
	ok := false
	for lc := 0; ts.Pos.Line < nlines && lc < nln; lc++ {
		sz := len(ts.Src.Lexs[ts.Pos.Line])
		if sz == 0 {
			ts.NextLine()
			continue
		}
		if pt.Semi {
			for ts.Pos.Char = 0; ts.Pos.Char < sz; ts.Pos.Char++ {
				lx := ts.Src.LexAt(ts.Pos)
				if lx.Token.Token == token.PunctSepSemicolon {
					ts.Src.ReplaceEos(ts.Pos)
				}
			}
		}
		if pt.RBraceEos {
			skip := false
			for ci := 0; ci < sz; ci++ {
				lx := ts.Src.LexAt(textpos.Pos{ts.Pos.Line, ci})
				if lx.Token.Token == token.PunctGpRBrace {
					if ci == 0 {
						ip := textpos.Pos{ts.Pos.Line, 0}
						ip, ok = ts.Src.PrevTokenPos(ip)
						if ok {
							ilx := ts.Src.LexAt(ip)
							if ilx.Token.Token != token.PunctGpLBrace && ilx.Token.Token != token.EOS {
								ts.Src.InsertEos(ip)
							}
						}
					} else {
						ip := textpos.Pos{ts.Pos.Line, ci - 1}
						ilx := ts.Src.LexAt(ip)
						if ilx.Token.Token != token.PunctGpLBrace {
							ts.Src.InsertEos(ip)
							ci++
							sz++
						}
					}
					if ci == sz-1 {
						ip := textpos.Pos{ts.Pos.Line, ci}
						ts.Src.InsertEos(ip)
						sz++
						skip = true
					}
				}
			}
			if skip {
				ts.NextLine()
				continue
			}
		}
		ep := textpos.Pos{ts.Pos.Line, sz - 1} // end of line token
		elx := ts.Src.LexAt(ep)
		if pt.Eol {
			sp := textpos.Pos{ts.Pos.Line, 0} // start of line token
			slx := ts.Src.LexAt(sp)
			if slx.Token.Depth == elx.Token.Depth {
				ts.Src.InsertEos(ep)
			}
		}
		if len(pt.EolToks) > 0 { // not depth specific
			if pt.EolToks.Match(elx.Token) {
				ts.Src.InsertEos(ep)
			}
		}
		ts.NextLine()
	}
}
