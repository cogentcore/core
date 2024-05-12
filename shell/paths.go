// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"go/token"
)

// ReplaceIdentAt replaces an identifier spanning n tokens
// starting at given index, with a single identifier with given string.
// This is used in Exec mode for dealing with identifiers and paths that are
// separately-parsed by Go.
func (tk Tokens) ReplaceIdentAt(at int, str string, n int) Tokens {
	ntk := append(tk[:at], &Token{Tok: token.IDENT, Str: str})
	ntk = append(ntk, tk[at+n:]...)
	return ntk
}

// Path extracts a standard path or URL expression from the current
// list of tokens (starting at index 0), returning the path string
// and the number of tokens included in the path.
// Restricts processing to contiguous elements with no spaces!
// If it is not a path, returns nil string, 0
func (tk Tokens) Path(idx0 bool) (string, int) {
	n := len(tk)
	if n == 0 {
		return "", 0
	}
	t0 := tk[0]
	ispath := (t0.IsPathDelim() || t0.Tok == token.TILDE)
	if n == 1 {
		if ispath {
			return t0.String(), 1
		}
		return "", 0
	}
	str := tk[0].String()
	lastEnd := int(tk[0].Pos) + len(str)
	ci := 1
	if !ispath {
		lastEnd = int(tk[0].Pos)
		ci = 0
		if t0.Tok != token.IDENT {
			return "", 0
		}
		tin := 1
		tid := t0.Str
		tindelim := tk[tin].IsPathDelim()
		if idx0 {
			tindelim = tk[tin].Tok == token.QUO
		}
		if (int(tk[tin].Pos) > lastEnd+len(tid)) || !(tk[tin].Tok == token.COLON || tindelim) {
			return "", 0
		}
		ci += tin + 1
		str = tid + tk[tin].String()
		lastEnd += len(str)
	}
	prevWasDelim := true
	for {
		if ci >= n || int(tk[ci].Pos) > lastEnd {
			return str, ci
		}
		ct := tk[ci]
		if ct.IsPathDelim() || ct.IsPathExtraDelim() {
			prevWasDelim = true
			str += ct.String()
			lastEnd += len(ct.String())
			ci++
			continue
		}
		if ct.Tok == token.STRING {
			prevWasDelim = true
			str += EscapeQuotes(ct.String())
			lastEnd += len(ct.String())
			ci++
			continue
		}
		if !prevWasDelim {
			if ct.Tok == token.ILLEGAL && ct.Str == `\` && ci+1 < n && int(tk[ci+1].Pos) == lastEnd+2 {
				prevWasDelim = true
				str += " "
				ci++
				lastEnd += 2
				continue
			}
			return str, ci
		}
		if ct.IsWord() {
			prevWasDelim = false
			str += ct.String()
			lastEnd += len(ct.String())
			ci++
			continue
		}
		return str, ci
	}
}

func (tk *Token) IsPathDelim() bool {
	return tk.Tok == token.PERIOD || tk.Tok == token.QUO
}

func (tk *Token) IsPathExtraDelim() bool {
	return tk.Tok == token.SUB || tk.Tok == token.ASSIGN || tk.Tok == token.REM || (tk.Tok == token.ILLEGAL && (tk.Str == "?" || tk.Str == "#"))
}

// IsWord returns true if the token is some kind of word-like entity,
// including IDENT, STRING, CHAR, or one of the Go keywords.
// This is for exec filtering.
func (tk *Token) IsWord() bool {
	return tk.Tok == token.IDENT || tk.IsGo() || tk.Tok == token.STRING || tk.Tok == token.CHAR
}
