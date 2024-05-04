// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"go/scanner"
	"go/token"
)

func (sh *Shell) ErrHand(pos token.Position, msg string) {
	sh.DebugPrintln(1, "Scan Error:", pos, msg)
}

type Token struct {
	Tok token.Token
	Pos token.Pos
	Str string
}

func (sh *Shell) Tokens(ln string) []Token {
	fset := token.NewFileSet()
	f := fset.AddFile("", fset.Base(), len(ln))
	var sc scanner.Scanner
	sc.Init(f, []byte(ln), sh.ErrHand, scanner.ScanComments)

	var toks []Token
	for {
		pos, tok, lit := sc.Scan()
		if tok == token.EOF {
			break
		}
		sh.DebugPrintf(2, "	token: %s\t%s\t%q\n", fset.Position(pos), tok, lit)
		toks = append(toks, Token{Tok: tok, Pos: pos, Str: lit})
	}
	return toks
}

func (sh *Shell) HasOperator(toks []Token) bool {
	for _, t := range toks {
		if t.Tok == token.SEMICOLON { // automatic
			continue
		}
		if t.Tok >= token.ADD && t.Tok <= token.COLON {
			return true
		}
	}
	return false
}

func (sh *Shell) ParseLine(ln string) {
	if len(ln) == 0 {
		return
	}
	sh.DebugPrintln(1, "######### starting line:")
	sh.DebugPrintln(1, ln)

	sh.GetSymbols()
	toks := sh.Tokens(ln)
	if len(toks) == 0 {
		return
	}
	hasOpr := sh.HasOperator(toks)
	sh.DebugPrintln(1, "hasOpr:", hasOpr)

	isGo := false
	switch toks[0].Tok {
	case token.IDENT: // only tricky case: string identifier
		has, _ := sh.SymbolByName(toks[0].Str)
		if has || hasOpr {
			isGo = true
		}
	default:
		isGo = true
	}

	if isGo {
		sh.DebugPrintln(1, "go interp:", ln)
		val, err := sh.Interp.Eval(ln)
		sh.DebugPrintln(2, "result:", val, err)
	} else {
		sh.ExecLine(ln)
	}
}
