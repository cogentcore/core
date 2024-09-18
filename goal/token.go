// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goal

import (
	"go/scanner"
	"go/token"
	"log/slog"
	"slices"
	"strings"

	"cogentcore.org/core/base/logx"
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
func NewToken(tok token.Token, str ...string) *Token {
	tk := &Token{Tok: tok}
	if len(str) > 0 {
		tk.Str = str[0]
	}
	return tk
}

// Add adds a new token, for generated tokens without Pos
func (tk *Tokens) Add(tok token.Token, str ...string) *Token {
	nt := NewToken(tok, str...)
	*tk = append(*tk, nt)
	return nt
}

// AddTokens adds given tokens to our list
func (tk *Tokens) AddTokens(toks ...*Token) *Tokens {
	*tk = append(*tk, toks...)
	return tk
}

// Insert inserts a new token at given position
func (tk *Tokens) Insert(i int, tok token.Token, str ...string) *Token {
	nt := NewToken(tok, str...)
	*tk = slices.Insert(*tk, i, nt)
	return nt
}

// Last returns the final token in the list
func (tk Tokens) Last() *Token {
	n := len(tk)
	if n == 0 {
		return nil
	}
	return tk[n-1]
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

// IsValidExecIdent returns true if the given token is a valid component
// of an Exec mode identifier
func (tk *Token) IsValidExecIdent() bool {
	return (tk.IsGo() || tk.Tok == token.IDENT || tk.Tok == token.SUB || tk.Tok == token.DEC || tk.Tok == token.INT || tk.Tok == token.FLOAT || tk.Tok == token.ASSIGN)
}

// String is the stringer version which includes the token ID
// in addition to the string literal
func (tk Tokens) String() string {
	str := ""
	for _, tok := range tk {
		str += "[" + tok.Tok.String() + "] "
		if tok.Str != "" {
			str += tok.Str + " "
		}
	}
	if len(str) == 0 {
		return str
	}
	return str[:len(str)-1] // remove trailing space
}

// Code returns concatenated Str values of the tokens,
// to generate a surface-valid code string.
func (tk Tokens) Code() string {
	n := len(tk)
	if n == 0 {
		return ""
	}
	str := ""
	prvIdent := false
	for _, tok := range tk {
		switch {
		case tok.IsOp():
			if tok.Tok == token.INC || tok.Tok == token.DEC {
				str += tok.String() + " "
			} else if tok.Tok == token.MUL {
				str += " " + tok.String()
			} else {
				str += " " + tok.String() + " "
			}
			prvIdent = false
		case tok.Tok == token.ELLIPSIS:
			str += " " + tok.String()
			prvIdent = false
		case tok.IsBracket() || tok.Tok == token.PERIOD:
			if tok.Tok == token.RBRACE || tok.Tok == token.LBRACE {
				if len(str) > 0 && str[len(str)-1] != ' ' {
					str += " "
				}
				str += tok.String() + " "
			} else {
				str += tok.String()
			}
			prvIdent = false
		case tok.Tok == token.COMMA || tok.Tok == token.COLON || tok.Tok == token.SEMICOLON:
			str += tok.String() + " "
			prvIdent = false
		case tok.Tok == token.STRUCT:
			str += " " + tok.String() + " "
		case tok.Tok == token.FUNC:
			if prvIdent {
				str += " "
			}
			str += tok.String()
			prvIdent = true
		case tok.IsGo():
			if prvIdent {
				str += " "
			}
			str += tok.String()
			if tok.Tok != token.MAP {
				str += " "
			}
			prvIdent = false
		case tok.Tok == token.IDENT || tok.Tok == token.STRING:
			if prvIdent {
				str += " "
			}
			str += tok.String()
			prvIdent = true
		default:
			str += tok.String()
			prvIdent = false
		}
	}
	if len(str) == 0 {
		return str
	}
	if str[len(str)-1] == ' ' {
		return str[:len(str)-1]
	}
	return str
}

// IsOp returns true if the given token is an operator
func (tk *Token) IsOp() bool {
	if tk.Tok >= token.ADD && tk.Tok <= token.DEFINE {
		return true
	}
	return false
}

// Contains returns true if the token string contains any of the given token(s)
func (tk Tokens) Contains(toks ...token.Token) bool {
	if len(toks) == 0 {
		slog.Error("programmer error: tokens.Contains with no args")
		return false
	}
	for _, t := range tk {
		for _, st := range toks {
			if t.Tok == st {
				return true
			}
		}
	}
	return false
}

// EscapeQuotes replaces any " with \"
func EscapeQuotes(str string) string {
	return strings.ReplaceAll(str, `"`, `\"`)
}

// AddQuotes surrounds given string with quotes,
// also escaping any contained quotes
func AddQuotes(str string) string {
	return `"` + EscapeQuotes(str) + `"`
}

// IsBracket returns true if the given token is a bracket delimiter:
// paren, brace, bracket
func (tk *Token) IsBracket() bool {
	if (tk.Tok >= token.LPAREN && tk.Tok <= token.LBRACE) || (tk.Tok >= token.RPAREN && tk.Tok <= token.RBRACE) {
		return true
	}
	return false
}

// RightMatching returns the position (or -1 if not found) for the
// right matching [paren, bracket, brace] given the left one that
// is at the 0 position of the current set of tokens.
func (tk Tokens) RightMatching() int {
	n := len(tk)
	if n == 0 {
		return -1
	}
	rb := token.RPAREN
	lb := tk[0].Tok
	switch lb {
	case token.LPAREN:
		rb = token.RPAREN
	case token.LBRACK:
		rb = token.RBRACK
	case token.LBRACE:
		rb = token.RBRACE
	}
	depth := 0
	for i := 1; i < n; i++ {
		tok := tk[i].Tok
		switch tok {
		case rb:
			if depth <= 0 {
				return i
			}
			depth--
		case lb:
			depth++
		}
	}
	return -1
}

// BracketDepths returns the depths for the three bracket delimiters
// [paren, bracket, brace], based on unmatched right versions.
func (tk Tokens) BracketDepths() (paren, brace, brack int) {
	n := len(tk)
	if n == 0 {
		return
	}
	for i := 0; i < n; i++ {
		tok := tk[i].Tok
		switch tok {
		case token.LPAREN:
			paren++
		case token.LBRACE:
			brace++
		case token.LBRACK:
			brack++
		case token.RPAREN:
			paren--
		case token.RBRACE:
			brace--
		case token.RBRACK:
			brack--
		}
	}
	return
}

// ModeEnd returns the position (or -1 if not found) for the
// next ILLEGAL mode token ($ or #) given the starting one that
// is at the 0 position of the current set of tokens.
func (tk Tokens) ModeEnd() int {
	n := len(tk)
	if n == 0 {
		return -1
	}
	st := tk[0].Str
	for i := 1; i < n; i++ {
		if tk[i].Tok != token.ILLEGAL {
			continue
		}
		if tk[i].Str == st {
			return i
		}
	}
	return -1
}

// Tokens converts the string into tokens
func (gl *Goal) Tokens(ln string) Tokens {
	fset := token.NewFileSet()
	f := fset.AddFile("", fset.Base(), len(ln))
	var sc scanner.Scanner
	sc.Init(f, []byte(ln), gl.errHandler, scanner.ScanComments|2) // 2 is non-exported dontInsertSemis
	// note to Go team: just export this stuff.  seriously.

	var toks Tokens
	for {
		pos, tok, lit := sc.Scan()
		if tok == token.EOF {
			break
		}
		// logx.PrintfDebug("	token: %s\t%s\t%q\n", fset.Position(pos), tok, lit)
		toks = append(toks, &Token{Tok: tok, Pos: pos, Str: lit})
	}
	return toks
}

func (gl *Goal) errHandler(pos token.Position, msg string) {
	logx.PrintlnDebug("Scan Error:", pos, msg)
}
