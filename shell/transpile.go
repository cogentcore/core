// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"fmt"
	"go/token"

	"cogentcore.org/core/base/logx"
)

// TranspileLine is the main function for parsing a single line of shell input,
// returning a new transpiled line of code that converts Exec code into corresponding
// Go function calls.
func (sh *Shell) TranspileLine(ln string) string {
	if len(ln) == 0 {
		return ln
	}
	toks := sh.TranspileLineTokens(ln)
	paren, brace, brack := toks.BracketDepths()
	sh.ParenDepth += paren
	sh.BraceDepth += brace
	sh.BrackDepth += brack
	// logx.PrintlnDebug("depths: ", sh.ParenDepth, sh.BraceDepth, sh.BrackDepth)
	return toks.Code()
}

// TranspileLineTokens returns the tokens for the full line
func (sh *Shell) TranspileLineTokens(ln string) Tokens {
	toks := sh.Tokens(ln)
	n := len(toks)
	if n == 0 {
		return toks
	}

	logx.PrintlnDebug("\n########## line:\n", ln, "\nTokens:\n", toks.String())

	t0path, t0pn := toks.Path(true) // true = first position
	_, t0in := toks.ExecIdent()

	t1idx := max(t0pn, t0in)
	t1pn := 0
	t1in := 0
	if n > 1 {
		if t1idx > 0 {
			end0 := int(toks[t1idx-1].Pos) + len(toks[t1idx-1].String())
			if t1idx < n && int(toks[t1idx].Pos) > end0 { // only if spaced
				_, t1pn = toks[t1idx:].Path(false)
				_, t1in = toks[t1idx:].ExecIdent()
			}
		}
	}

	switch {
	case toks[0].Tok == token.LBRACE:
		logx.PrintlnDebug("go:   { } line")
		return sh.TranspileGo(toks[1 : n-1])
	case toks[0].IsBacktickString():
		logx.PrintlnDebug("exec: backquoted string")
		exe := sh.TranspileExecString(toks[0].Str, false)
		if n > 1 { // todo: is this an error?
			exe.AddTokens(sh.TranspileGo(toks[1:]))
		}
		return exe
	case toks[0].IsGo():
		if toks[0].Tok == token.GO {
			if !toks.Contains(token.LPAREN) {
				logx.PrintlnDebug("exec: go command")
				return sh.TranspileExec(toks, false)
			}
		}
		logx.PrintlnDebug("go    keyword")
		return sh.TranspileGo(toks)
	case t0pn > 0: // path expr
		logx.PrintlnDebug("exec: path...")
		rtok := toks.ReplaceIdentAt(0, t0path, t0pn)
		return sh.TranspileExec(rtok, false)
	case toks[0].Tok == token.STRING:
		logx.PrintlnDebug("exec: string...")
		return sh.TranspileExec(toks, false)
	case n == t0in || t0in > 1:
		logx.PrintlnDebug("exec: 1 word or non-go word")
		return sh.TranspileExec(toks, false)
	case t0in == 0: // exec must be IDENT
		logx.PrintlnDebug("go:   not ident")
		return sh.TranspileGo(toks)
	case t0in > 0 && n > t0in && (t1in > 0 || t1pn > 0 || toks[t1idx].Tok == token.SUB || toks[t1idx].Tok == token.STRING || toks[t1idx].Tok == token.LBRACE):
		logx.PrintlnDebug("exec: word non-go...")
		return sh.TranspileExec(toks, false)
	default:
		logx.PrintlnDebug("go:   default")
		return sh.TranspileGo(toks)
	}
}

// TranspileGo returns transpiled tokens assuming Go code.
// Unpacks any backtick encapsulated shell commands.
func (sh *Shell) TranspileGo(toks Tokens) Tokens {
	gtoks := make(Tokens, 0, len(toks)) // return tokens
	for _, tok := range toks {
		if tok.IsBacktickString() {
			gtoks = append(gtoks, sh.TranspileExecString(tok.Str, true)...)
		} else {
			gtoks = append(gtoks, tok)
		}
	}
	return gtoks
}

// TranspileExecString returns transpiled tokens assuming Exec code,
// from a backtick-encoded string, with the given bool indicating
// whether [Output] is needed.
func (sh *Shell) TranspileExecString(str string, output bool) Tokens {
	etoks := sh.Tokens(str[1 : len(str)-1]) // enclosed string
	return sh.TranspileExec(etoks, output)
}

// TranspileExec returns transpiled tokens assuming Exec code,
// with the given bool indicating whether [Output] is needed.
func (sh *Shell) TranspileExec(toks Tokens, output bool) Tokens {
	etoks := make(Tokens, 0, len(toks)*2+5) // return tokens
	etoks.Add(token.IDENT, "shell")
	etoks.Add(token.PERIOD)
	if output {
		etoks.Add(token.IDENT, "Output")
	} else {
		etoks.Add(token.IDENT, "Exec")
	}
	etoks.Add(token.LPAREN)
	n := len(toks)
	for i := 0; i < n; i++ {
		tok := toks[i]
		tpath, tpn := toks[i:].Path(false)
		tid, tin := toks[i:].ExecIdent()
		switch {
		case tpn > 0:
			etoks.Add(token.STRING, `"`+tpath+`"`)
			i += tpn
		case tok.Tok == token.LBRACE:
			rb := toks[i:].RightMatching()
			if rb < 0 {
				sh.AddError(fmt.Errorf("no right brace found in exec command line"))
			} else {
				etoks.AddTokens(sh.TranspileGo(toks[i+1 : i+rb]))
				i += rb
			}
		case tok.Tok == token.SUB && i < n-1: // option
			nid, nin := toks[i+1:].ExecIdent()
			if nin > 0 {
				etoks.Add(token.STRING, `"-`+nid+`"`)
				i += nin
			} else {
				etoks.Add(token.STRING, `"-`+toks[i+1].Str+`"`)
				i++
			}
		case tin > 0:
			etoks.Add(token.STRING, `"`+tid+`"`)
			i += (tin - 1)
		case tok.Tok == token.STRING:
			etoks.Add(token.STRING, tok.Str)
		case tok.Tok == token.ASSIGN:
			etoks.Add(token.STRING, `"`+tok.String()+`"`)
		default:
			etoks.Add(token.STRING, `"`+tok.Str+`"`)
		}
		etoks.Add(token.COMMA)
	}
	etoks.DeleteLastComma()
	etoks.Add(token.RPAREN)
	return etoks
}
