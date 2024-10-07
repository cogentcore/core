// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package transpile

import (
	"fmt"
	"go/token"
	"strings"

	"cogentcore.org/core/base/logx"
)

// TranspileLine is the main function for parsing a single line of goal input,
// returning a new transpiled line of code that converts Exec code into corresponding
// Go function calls.
func (st *State) TranspileLine(code string) string {
	if len(code) == 0 {
		return code
	}
	if strings.HasPrefix(code, "#!") {
		return ""
	}
	toks := st.TranspileLineTokens(code)
	paren, brace, brack := toks.BracketDepths()
	st.ParenDepth += paren
	st.BraceDepth += brace
	st.BrackDepth += brack
	if st.TypeDepth > 0 && st.BraceDepth == 0 {
		st.TypeDepth = 0
	}
	if st.DeclDepth > 0 && st.ParenDepth == 0 {
		st.DeclDepth = 0
	}
	// logx.PrintlnDebug("depths: ", sh.ParenDepth, sh.BraceDepth, sh.BrackDepth)
	return toks.Code()
}

// TranspileLineTokens returns the tokens for the full line
func (st *State) TranspileLineTokens(code string) Tokens {
	if code == "" {
		return nil
	}
	toks := TokensFromString(code)
	n := len(toks)
	if n == 0 {
		return toks
	}
	if st.MathMode {
		if len(toks) >= 2 {
			if toks[0].Tok == token.ILLEGAL && toks[0].Str == "#" && toks[1].Tok == token.ILLEGAL && toks[1].Str == "#" {
				st.MathMode = false
				return nil
			}
		}
		return st.TranspileMath(toks, code, true)
	}
	ewords, err := ExecWords(code)
	if err != nil {
		st.AddError(err)
		return nil
	}
	logx.PrintlnDebug("\n########## line:\n", code, "\nTokens:\n", toks.String(), "\nWords:\n", ewords)

	if toks[0].Tok == token.TYPE {
		st.TypeDepth++
	}
	if toks[0].Tok == token.IMPORT || toks[0].Tok == token.VAR || toks[0].Tok == token.CONST {
		st.DeclDepth++
	}

	if st.TypeDepth > 0 || st.DeclDepth > 0 {
		logx.PrintlnDebug("go:   type / decl defn")
		return st.TranspileGo(toks, code)
	}

	t0 := toks[0]
	_, t0pn := toks.Path(true) // true = first position
	en := len(ewords)

	f0exec := (t0.Tok == token.IDENT && ExecWordIsCommand(ewords[0]))

	switch {
	case t0.Tok == token.ILLEGAL:
		if t0.Str == "#" {
			logx.PrintlnDebug("math #")
			if toks[1].Tok == token.ILLEGAL && toks[1].Str == "#" {
				st.MathMode = true
				return nil
			}
			return st.TranspileMath(toks[1:], code, true)
		}
		return st.TranspileExec(ewords, false)
	case t0.Tok == token.LBRACE:
		logx.PrintlnDebug("go:   { } line")
		return st.TranspileGo(toks[1:n-1], code[toks[1].Pos-1:toks[n-1].Pos-1])
	case t0.Tok == token.LBRACK:
		logx.PrintlnDebug("exec: [ ] line")
		return st.TranspileExec(ewords, false) // it processes the [ ]
	case t0.Tok == token.IDENT && t0.Str == "command":
		st.lastCommand = toks[1].Str // 1 is the name -- triggers AddCommand
		toks = toks[2:]              // get rid of first
		toks.Insert(0, token.IDENT, "goal.AddCommand")
		toks.Insert(1, token.LPAREN)
		toks.Insert(2, token.STRING, `"`+st.lastCommand+`"`)
		toks.Insert(3, token.COMMA)
		toks.Insert(4, token.FUNC)
		toks.Insert(5, token.LPAREN)
		toks.Insert(6, token.IDENT, "args")
		toks.Insert(7, token.ELLIPSIS)
		toks.Insert(8, token.IDENT, "string")
		toks.Insert(9, token.RPAREN)
		toks.AddTokens(st.TranspileGo(toks[11:], code)...)
	case t0.IsGo():
		if t0.Tok == token.GO {
			if !toks.Contains(token.LPAREN) {
				logx.PrintlnDebug("exec: go command")
				return st.TranspileExec(ewords, false)
			}
		}
		logx.PrintlnDebug("go    keyword")
		return st.TranspileGo(toks, code)
	case toks[n-1].Tok == token.INC:
		return st.TranspileGo(toks, code)
	case t0pn > 0: // path expr
		logx.PrintlnDebug("exec: path...")
		return st.TranspileExec(ewords, false)
	case t0.Tok == token.STRING:
		logx.PrintlnDebug("exec: string...")
		return st.TranspileExec(ewords, false)
	case f0exec && en == 1:
		logx.PrintlnDebug("exec: 1 word")
		return st.TranspileExec(ewords, false)
	case !f0exec: // exec must be IDENT
		logx.PrintlnDebug("go:   not ident")
		return st.TranspileGo(toks, code)
	case f0exec && en > 1 && (ewords[1][0] == '=' || ewords[1][0] == ':' || ewords[1][0] == '+' || toks[1].Tok == token.COMMA):
		logx.PrintlnDebug("go:   assignment or defn")
		return st.TranspileGo(toks, code)
	case f0exec: // now any ident
		logx.PrintlnDebug("exec: ident..")
		return st.TranspileExec(ewords, false)
	default:
		logx.PrintlnDebug("go:   default")
		return st.TranspileGo(toks, code)
	}
	return toks
}

// TranspileGo returns transpiled tokens assuming Go code.
// Unpacks any encapsulated shell or math expressions.
func (st *State) TranspileGo(toks Tokens, code string) Tokens {
	n := len(toks)
	if n == 0 {
		return toks
	}
	if st.FuncToVar && toks[0].Tok == token.FUNC { // reorder as an assignment
		if len(toks) > 1 && toks[1].Tok == token.IDENT {
			toks[0] = toks[1]
			toks.Insert(1, token.DEFINE)
			toks[2] = &Token{Tok: token.FUNC}
			n = len(toks)
		}
	}
	gtoks := make(Tokens, 0, len(toks)) // return tokens
	for i := 0; i < n; i++ {
		tok := toks[i]
		switch {
		case tok.Tok == token.ILLEGAL:
			et := toks[i:].ModeEnd()
			if et > 0 {
				if tok.Str == "#" {
					gtoks.AddTokens(st.TranspileMath(toks[i+1:i+et], code, false)...)
				} else {
					gtoks.AddTokens(st.TranspileExecTokens(toks[i+1:i+et+1], code, true)...)
				}
				i += et
				continue
			} else {
				gtoks = append(gtoks, tok)
			}
		case tok.Tok == token.LBRACK && i > 0 && toks[i-1].Tok == token.IDENT: // index expr
			ixtoks := toks[i:]
			rm := ixtoks.RightMatching()
			if rm < 3 {
				gtoks = append(gtoks, tok)
				continue
			}
			idx := st.TranspileGoNDimIndex(toks, &gtoks, i-1, rm+i)
			if idx > 0 {
				i = idx
			} else {
				gtoks = append(gtoks, tok)
			}
		default:
			gtoks = append(gtoks, tok)
		}
	}
	return gtoks
}

// TranspileExecString returns transpiled tokens assuming Exec code,
// from a string, with the given bool indicating whether [Output] is needed.
func (st *State) TranspileExecString(str string, output bool) Tokens {
	if len(str) <= 1 {
		return nil
	}
	ewords, err := ExecWords(str)
	if err != nil {
		st.AddError(err)
	}
	return st.TranspileExec(ewords, output)
}

// TranspileExecTokens returns transpiled tokens assuming Exec code,
// from given tokens, with the given bool indicating
// whether [Output] is needed.
func (st *State) TranspileExecTokens(toks Tokens, code string, output bool) Tokens {
	nt := len(toks)
	if nt == 0 {
		return nil
	}
	str := code[toks[0].Pos-1 : toks[nt-1].Pos-1]
	return st.TranspileExecString(str, output)
}

// TranspileExec returns transpiled tokens assuming Exec code,
// with the given bools indicating the type of run to execute.
func (st *State) TranspileExec(ewords []string, output bool) Tokens {
	n := len(ewords)
	if n == 0 {
		return nil
	}
	etoks := make(Tokens, 0, n+5) // return tokens
	var execTok *Token
	bgJob := false
	noStop := false
	if ewords[0] == "[" {
		ewords = ewords[1:]
		n--
		noStop = true
	}
	startExec := func() {
		bgJob = false
		etoks.Add(token.IDENT, "goal")
		etoks.Add(token.PERIOD)
		switch {
		case output && noStop:
			execTok = etoks.Add(token.IDENT, "OutputErrOK")
		case output && !noStop:
			execTok = etoks.Add(token.IDENT, "Output")
		case !output && noStop:
			execTok = etoks.Add(token.IDENT, "RunErrOK")
		case !output && !noStop:
			execTok = etoks.Add(token.IDENT, "Run")
		}
		etoks.Add(token.LPAREN)
	}
	endExec := func() {
		if bgJob {
			execTok.Str = "Start"
		}
		etoks.DeleteLastComma()
		etoks.Add(token.RPAREN)
	}

	startExec()

	for i := 0; i < n; i++ {
		f := ewords[i]
		switch {
		// case f == "#": // embedded math TODO
		case f == "{": // embedded go
			if n < i+3 {
				st.AddError(fmt.Errorf("goal: no matching right brace } found in exec command line"))
			} else {
				gstr := ewords[i+1]
				etoks.AddTokens(st.TranspileGo(TokensFromString(gstr), gstr)...)
				etoks.Add(token.COMMA)
				i += 2
			}
		case f == "[":
			noStop = true
		case f == "]": // solo is def end
			// just skip
			noStop = false
		case f == "&":
			bgJob = true
		case f[0] == '|':
			execTok.Str = "Start"
			etoks.Add(token.IDENT, AddQuotes(f))
			etoks.Add(token.COMMA)
			endExec()
			etoks.Add(token.SEMICOLON)
			etoks.AddTokens(st.TranspileExec(ewords[i+1:], output)...)
			return etoks
		case f == ";":
			endExec()
			etoks.Add(token.SEMICOLON)
			etoks.AddTokens(st.TranspileExec(ewords[i+1:], output)...)
			return etoks
		default:
			if f[0] == '"' || f[0] == '`' {
				etoks.Add(token.STRING, f)
			} else {
				etoks.Add(token.IDENT, AddQuotes(f)) // mark as an IDENT but add quotes!
			}
			etoks.Add(token.COMMA)
		}
	}
	endExec()
	return etoks
}

// TranspileGoNDimIndex processes an ident[*] sequence of tokens,
// translating it into a corresponding tensor Value or Set expression,
// if it is a multi-dimensional indexing expression which is not valid in Go,
// to support simple n-dimensional tensor indexing in Go (not math) mode.
// Gets the current sequence of toks tokens, where the ident starts at idIdx
// and the ] is at rbIdx. It puts the results in gtoks generated tokens.
// Returns a positive index to resume processing at, if it is actually an
// n-dimensional expr, and -1 if not, in which case the normal process resumes.
func (st *State) TranspileGoNDimIndex(toks Tokens, gtoks *Tokens, idIdx, rbIdx int) int {
	nc := 0
	for i := idIdx + 2; i < rbIdx; i++ {
		tk := toks[i]
		if tk.Tok == token.COMMA {
			nc++
			break
		}
		if tk.Tok == token.LPAREN || tk.Tok == token.LBRACE || tk.Tok == token.LBRACK {
			rp := toks[i:rbIdx].RightMatching()
			if rp > 0 {
				i += rp
			}
		}
	}
	if nc == 0 { // not multidim
		return -1
	}
	// now we need to determine if it is a Set based on what happens after rb
	isSet := false
	stok := token.ILLEGAL
	n := len(toks)
	if n-rbIdx > 1 {
		ntk := toks[rbIdx+1].Tok
		if ntk == token.ASSIGN || (ntk >= token.ADD_ASSIGN && ntk <= token.QUO_ASSIGN) {
			isSet = true
			stok = ntk
		}
	}
	fun := "Value"
	if isSet {
		fun = "Set"
		switch stok {
		case token.ADD_ASSIGN:
			fun += "Add"
		case token.SUB_ASSIGN:
			fun += "Sub"
		case token.MUL_ASSIGN:
			fun += "Mul"
		case token.QUO_ASSIGN:
			fun += "Div"
		}
	}
	gtoks.Add(token.PERIOD)
	gtoks.Add(token.IDENT, fun)
	gtoks.Add(token.LPAREN)
	if isSet {
		gtoks.AddTokens(toks[rbIdx+2:]...)
		gtoks.Add(token.COMMA)
	}
	gtoks.AddTokens(toks[idIdx+2 : rbIdx]...)
	gtoks.Add(token.RPAREN)
	if isSet {
		return n
	} else {
		return rbIdx
	}
}
