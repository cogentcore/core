// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"go/scanner"
	"go/token"
)

// Token provides full data for one token
type Token struct {
	// Go token classification
	Tok token.Token

	// Literal string
	Str string

	// position in the original string.
	// this is only set for the original parse,
	// not for transpiled additions.
	Pos token.Pos
}

// Tokens is a slice of Token
type Tokens []*Token

// NewToken returns a new token, for generated tokens without Pos
func NewToken(tok token.Token, str string) *Token {
	return &Token{Tok: tok, Str: str}
}

// Add adds a new token, for generated tokens without Pos
func (tk *Tokens) Add(tok token.Token, str string) *Token {
	nt := NewToken(tok, str)
	*tk = append(*tk, nt)
	return nt
}

// AddTokens adds given tokens to our list
func (tk *Tokens) AddTokens(toks Tokens) *Tokens {
	*tk = append(*tk, toks...)
	return tk
}

// Last returns the final token in the list
func (tk Tokens) Last() *Token {
	sz := len(tk)
	if sz == 0 {
		return nil
	}
	return tk[sz-1]
}

// DeleteLastComma removes any final Comma.
// easier to generate and delete at the end
func (tk *Tokens) DeleteLastComma() {
	lt := tk.Last()
	if lt == nil {
		return
	}
	if lt.Tok == token.COMMA {
		*tk = (*tk)[:len(*tk)-1]
	}
}

// String returns the string for the token
func (tk *Token) String() string {
	if tk.Str != "" {
		return tk.Str
	}
	return tk.Tok.String()
}

// IsBacktickString returns true if the given STRING uses backticks
func (tk *Token) IsBacktickString() bool {
	if tk.Tok != token.STRING {
		return false
	}
	return (tk.Str[0] == '`')
}

// IsGo returns true if the given token is a Go Keyword or Comment
func (tk *Token) IsGo() bool {
	if tk.Tok >= token.BREAK && tk.Tok <= token.VAR {
		return true
	}
	if tk.Tok == token.COMMENT {
		return true
	}
	return false
}

// String is the stringer version which includes the token ID
// in addition to the string literal
func (tk Tokens) String() string {
	str := ""
	for _, tok := range tk {
		str += "[" + tok.Tok.String() + "] " + tok.String() + " "
	}
	if len(str) == 0 {
		return str
	}
	return str[:len(str)-1] // remove trailing space
}

// Code returns concatenated Str values of the tokens,
// to generate a surface-valid code string.
func (tk Tokens) Code() string {
	str := ""
	for _, tok := range tk {
		// todo: detect . ( ) etc and manage spaces properly first-pass
		str += tok.String() + " "
	}
	if len(str) == 0 {
		return str
	}
	return str[:len(str)-1] // remove trailing space
}

func (sh *Shell) Tokens(ln string) Tokens {
	fset := token.NewFileSet()
	f := fset.AddFile("", fset.Base(), len(ln))
	var sc scanner.Scanner
	sc.Init(f, []byte(ln), sh.errHandler, scanner.ScanComments|2) // 2 is non-exported dontInsertSemis
	// note to Go team: just export this stuff.  seriously.

	var toks Tokens
	for {
		pos, tok, lit := sc.Scan()
		if tok == token.EOF {
			break
		}
		sh.DebugPrintf(2, "	token: %s\t%s\t%q\n", fset.Position(pos), tok, lit)
		toks = append(toks, &Token{Tok: tok, Pos: pos, Str: lit})
	}
	return toks
}

func (sh *Shell) errHandler(pos token.Position, msg string) {
	sh.DebugPrintln(1, "Scan Error:", pos, msg)
}
