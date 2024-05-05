// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"go/token"
	"log/slog"

	"cogentcore.org/core/base/logx"
)

// TranspileLine is the main function for parsing a single line of shell input,
// returning a new transpiled line of code that converts Exec code into corresponding
// Go function calls.
func (sh *Shell) TranspileLine(ln string) string {
	if len(ln) == 0 {
		return ln
	}
	logx.PrintlnInfo("######### starting line:")
	logx.PrintlnInfo(ln)

	toks := sh.Tokens(ln)
	if len(toks) == 0 {
		return ln
	}

	logx.PrintlnInfo(toks.String())

	switch {
	case toks[0].IsBacktickString():
		logx.PrintlnInfo("exec: backquoted string")
		exe := sh.TranspileExecString(toks[0].Str, false)
		if len(toks) > 1 { // todo: is this an error?
			exe.AddTokens(sh.TranspileGo(toks[1:]))
		}
		return exe.Code()
	case toks[0].IsGo():
		logx.PrintlnInfo("go    keyword")
		return sh.TranspileGo(toks).Code()
	case len(toks) == 1 && toks[0].Tok == token.IDENT:
		logx.PrintlnInfo("exec: 1 word")
		return sh.TranspileExec(toks, false).Code()
	case toks[0].Tok == token.PERIOD: // path expr
		logx.PrintlnInfo("exec: .")
		return sh.TranspileExec(toks, false).Code()
	case toks[0].Tok != token.IDENT: // exec must be IDENT
		logx.PrintlnInfo("go:   not ident")
		return sh.TranspileGo(toks).Code()
	case len(toks) == 2 && toks[0].Tok == token.IDENT && toks[1].Tok == token.IDENT:
		logx.PrintlnInfo("exec: word word")
		return sh.TranspileExec(toks, false).Code()
	case toks[0].Tok == token.IDENT && toks[1].Tok == token.LBRACE:
		logx.PrintlnInfo("exec: word {")
		return sh.TranspileExec(toks, false).Code()
	default:
		logx.PrintlnInfo("go:   default")
		return sh.TranspileGo(toks).Code()
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
	etoks.Add(token.PERIOD, ".")
	if output {
		etoks.Add(token.IDENT, "Output")
	} else {
		etoks.Add(token.IDENT, "Exec")
	}
	etoks.Add(token.IDENT, "(")
	sz := len(toks)
	for i := 0; i < sz; i++ {
		tok := toks[i]
		switch {
		case tok.Tok == token.LBRACE: // todo: find the closing brace
			rb := toks[i:].RightMatching()
			if rb < 0 {
				slog.Error("no right brace found in exec command line")
			} else {
				etoks.AddTokens(sh.TranspileGo(toks[i+1 : i+rb]))
				i += rb
			}
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
