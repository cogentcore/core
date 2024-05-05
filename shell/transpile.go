// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"go/token"
)

// TranspileLine is the main function for parsing a single line of shell input,
// returning a new transpiled line of code that converts Exec code into corresponding
// Go function calls.
func (sh *Shell) TranspileLine(ln string) string {
	if len(ln) == 0 {
		return ln
	}
	sh.DebugPrintln(1, "######### starting line:")
	sh.DebugPrintln(1, ln)

	toks := sh.Tokens(ln)
	if len(toks) == 0 {
		return ln
	}

	sh.DebugPrintln(1, toks.String())

	switch {
	case toks[0].IsBacktickString():
		sh.DebugPrintln(1, "exec: backquoted string")
		exe := sh.TranspileExecString(toks[0].Str)
		if len(toks) > 1 { // todo: is this an error?
			exe.AddTokens(sh.TranspileGo(toks[1:]))
		}
		return exe.Code()
	case toks[0].IsGo():
		sh.DebugPrintln(1, "go    keyword")
		return sh.TranspileGo(toks).Code()
	case len(toks) == 1 && toks[0].Tok == token.IDENT:
		sh.DebugPrintln(1, "exec: 1 word")
		return sh.TranspileExec(toks).Code()
	case toks[0].Tok == token.PERIOD: // path expr
		sh.DebugPrintln(1, "exec: .")
		return sh.TranspileExec(toks).Code()
	case toks[0].Tok != token.IDENT: // exec must be IDENT
		sh.DebugPrintln(1, "go:   not ident")
		return sh.TranspileGo(toks).Code()
	case len(toks) == 2 && toks[0].Tok == token.IDENT && toks[1].Tok == token.IDENT:
		sh.DebugPrintln(1, "exec: word word")
		return sh.TranspileExec(toks).Code()
	case toks[0].Tok == token.IDENT && toks[1].Tok == token.LBRACE:
		sh.DebugPrintln(1, "exec: word {")
		return sh.TranspileExec(toks).Code()
	default:
		sh.DebugPrintln(1, "go:   default")
		return sh.TranspileGo(toks).Code()
	}
}

// TranspileGo returns transpiled tokens assuming Go code.
// Unpacks any backtick encapsulated shell commands.
func (sh *Shell) TranspileGo(toks Tokens) Tokens {
	gtoks := make(Tokens, 0, len(toks)) // return tokens
	for _, tok := range toks {
		if tok.IsBacktickString() {
			gtoks = append(gtoks, sh.TranspileExecString(tok.Str)...)
		} else {
			gtoks = append(gtoks, tok)
		}
	}
	return gtoks
}

// TranspileExecString returns transpiled tokens assuming Exec code,
// from a backtick-encoded string.
func (sh *Shell) TranspileExecString(str string) Tokens {
	etoks := sh.Tokens(str[1 : len(str)-1]) // enclosed string
	return sh.TranspileExec(etoks)
}

// TranspileExec returns transpiled tokens assuming Exec code.
func (sh *Shell) TranspileExec(toks Tokens) Tokens {
	etoks := make(Tokens, 0, len(toks)*2+5) // return tokens
	etoks.Add(token.IDENT, "shell")
	etoks.Add(token.PERIOD, ".")
	etoks.Add(token.IDENT, "Exec")
	etoks.Add(token.IDENT, "(")
	sz := len(toks)
	for i := 0; i < sz; i++ {
		tok := toks[i]
		switch {
		case tok.Tok == token.LBRACE: // todo: find the closing brace
			etoks.AddTokens(sh.TranspileGo(toks[i+1:]))
			etoks.Add(token.COMMA, ",")
			continue
		case tok.Tok == token.SUB && i < sz-1: // option
			etoks.Add(token.STRING, `"-`+toks[i+1].Str+`"`)
			etoks.Add(token.COMMA, ",")
			i++
		default:
			etoks.Add(token.STRING, `"`+tok.Str+`"`)
			etoks.Add(token.COMMA, ",")
		}
	}
	etoks.DeleteLastComma()
	etoks.Add(token.IDENT, ")")
	return etoks
}
