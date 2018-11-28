// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"fmt"
	"unicode"

	"github.com/goki/pi/token"
)

// lex.State is the state maintained for lexing
type State struct {
	Filename     string    `desc:"the current file being lex'd"`
	KeepWS       bool      `desc:"if true, record whitespace tokens -- else ignore"`
	KeepComments bool      `desc:"if true, record comment tokens -- else ignore"`
	Src          []rune    `desc:"the current line of source being processed"`
	Lex          Line      `desc:"the lex output for this line"`
	Pos          int       `desc:"the current rune char position within the line"`
	Ln           int       `desc:"the line within overall source that we're operating on (0 indexed)"`
	Ch           rune      `desc:"the current rune read by NextRune"`
	State        []string  `desc:"state stack"`
	Errs         ErrorList `desc:"any error messages accumulated during lexing specifically"`
}

// Init initializes the state at start of parsing
func (ls *State) Init() {
	ls.State = nil
	ls.Ln = 0
	ls.SetLine(nil)
	ls.Errs.Reset()
}

// SetLine sets a new line for parsing and initializes the lex output and pos
func (ls *State) SetLine(src []rune) {
	ls.Src = src
	ls.Lex = nil
	ls.Pos = 0
}

// LineOut returns the current lex output as tagged source
func (ls *State) LineOut() string {
	return fmt.Sprintf("[%v,%v]: %v", ls.Ln, ls.Pos, ls.Lex.TagSrc(ls.Src))
}

// Error adds a lexing error at given position
func (ls *State) Error(pos int, msg string) {
	ls.Errs.Add(Pos{ls.Ln, pos}, ls.Filename, "Lexer: "+msg)
}

// AtEol returns true if current position is at end of line
func (ls *State) AtEol() bool {
	sz := len(ls.Src)
	return ls.Pos >= sz
}

// String gets the string at given offset and length from current position, returns false if out of range
func (ls *State) String(off, sz int) (string, bool) {
	idx := ls.Pos + off
	ei := idx + sz
	if ei > len(ls.Src) {
		return "", false
	}
	return string(ls.Src[idx:ei]), true
}

// Rune gets the rune at given offset from current position, returns false if out of range
func (ls *State) Rune(off int) (rune, bool) {
	idx := ls.Pos + off
	if idx >= len(ls.Src) {
		return 0, false
	}
	return ls.Src[idx], true
}

// Next moves to next position using given increment in source line -- returns false if at end
func (ls *State) Next(inc int) bool {
	sz := len(ls.Src)
	ls.Pos += inc
	if ls.Pos >= sz {
		ls.Pos = sz
		return false
	}
	return true
}

// NextRune reads the next rune into Ch and returns false if at end of line
func (ls *State) NextRune() bool {
	sz := len(ls.Src)
	ls.Pos++
	if ls.Pos >= sz {
		ls.Pos = sz
		return false
	}
	ls.Ch = ls.Src[ls.Pos]
	return true
}

// CurRune reads the current rune into Ch and returns false if at end of line
func (ls *State) CurRune() bool {
	sz := len(ls.Src)
	if ls.Pos >= sz {
		ls.Pos = sz
		return false
	}
	ls.Ch = ls.Src[ls.Pos]
	return true
}

// Add adds a lex token for given region -- merges with prior if same
func (ls *State) Add(tok token.Tokens, st, ed int) {
	if tok == token.TextWhitespace && !ls.KeepWS {
		return
	}
	if tok.Cat() == token.Comment && !ls.KeepComments {
		return
	}
	sz := len(ls.Lex)
	if sz > 0 && tok.CombineRepeats() {
		lst := &ls.Lex[sz-1]
		if lst.Tok == tok && lst.Ed == st {
			lst.Ed = ed
			return
		}
	}
	ls.Lex.Add(Lex{tok, 0, st, ed})
}

func (ls *State) PushState(st string) {
	ls.State = append(ls.State, st)
}

func (ls *State) CurState() string {
	sz := len(ls.State)
	if sz == 0 {
		return ""
	}
	return ls.State[sz-1]
}

func (ls *State) PopState() string {
	sz := len(ls.State)
	if sz == 0 {
		return ""
	}
	st := ls.CurState()
	ls.State = ls.State[:sz-1]
	return st
}

func (ls *State) ReadName() {
	sz := len(ls.Src)
	for ls.Pos < sz {
		rn := ls.Src[ls.Pos]
		if IsLetter(rn) || IsDigit(rn) {
			ls.Pos++
		} else {
			break
		}
	}
}

func (ls *State) ReadNumber() token.Tokens {
	offs := ls.Pos
	tok := token.LitNumInteger
	ls.CurRune()
	if ls.Ch == '0' {
		// int or float
		offs := ls.Pos
		ls.NextRune()
		if ls.Ch == 'x' || ls.Ch == 'X' {
			// hexadecimal int
			ls.NextRune()
			ls.ScanMantissa(16)
			if ls.Pos-offs <= 2 {
				// only scanned "0x" or "0X"
				ls.Error(offs, "illegal hexadecimal number")
			}
		} else {
			// octal int or float
			seenDecimalDigit := false
			ls.ScanMantissa(8)
			if ls.Ch == '8' || ls.Ch == '9' {
				// illegal octal int or float
				seenDecimalDigit = true
				ls.ScanMantissa(10)
			}
			if ls.Ch == '.' || ls.Ch == 'e' || ls.Ch == 'E' || ls.Ch == 'i' {
				goto fraction
			}
			// octal int
			if seenDecimalDigit {
				ls.Error(offs, "illegal octal number")
			}
		}
		goto exit
	}

	// decimal int or float
	ls.ScanMantissa(10)

fraction:
	if ls.Ch == '.' {
		tok = token.LitNumFloat
		ls.NextRune()
		ls.ScanMantissa(10)
	}

	if ls.Ch == 'e' || ls.Ch == 'E' {
		tok = token.LitNumFloat
		ls.NextRune()
		if ls.Ch == '-' || ls.Ch == '+' {
			ls.NextRune()
		}
		if DigitVal(ls.Ch) < 10 {
			ls.ScanMantissa(10)
		} else {
			ls.Error(offs, "illegal floating-point exponent")
		}
	}

	if ls.Ch == 'i' {
		tok = token.LitNumImag
		ls.NextRune()
	}

exit:
	return tok
}

func DigitVal(ch rune) int {
	switch {
	case '0' <= ch && ch <= '9':
		return int(ch - '0')
	case 'a' <= ch && ch <= 'f':
		return int(ch - 'a' + 10)
	case 'A' <= ch && ch <= 'F':
		return int(ch - 'A' + 10)
	}
	return 16 // larger than any legal digit val
}

func (ls *State) ScanMantissa(base int) {
	for DigitVal(ls.Ch) < base {
		if !ls.NextRune() {
			break
		}
	}
}

func (ls *State) ReadQuoted() {
	delim, _ := ls.Rune(0)
	offs := ls.Pos
	ls.NextRune()
	for {
		ch := ls.Ch
		if ch == '\n' || ch < 0 {
			ls.Error(offs, "string literal not terminated")
			break
		}
		if ch == delim {
			ls.NextRune() // move past
			break
		}
		if ch == '\\' {
			ls.ReadEscape(delim)
		}
		if !ls.NextRune() {
			ls.Error(offs, "string literal not terminated")
			break
		}
	}
}

// ReadEscape parses an escape sequence where rune is the accepted
// escaped quote. In case of a syntax error, it stops at the offending
// character (without consuming it) and returns false. Otherwise
// it returns true.
func (ls *State) ReadEscape(quote rune) bool {
	offs := ls.Pos

	var n int
	var base, max uint32
	switch ls.Ch {
	case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', quote:
		ls.NextRune()
		return true
	case '0', '1', '2', '3', '4', '5', '6', '7':
		n, base, max = 3, 8, 255
	case 'x':
		ls.NextRune()
		n, base, max = 2, 16, 255
	case 'u':
		ls.NextRune()
		n, base, max = 4, 16, unicode.MaxRune
	case 'U':
		ls.NextRune()
		n, base, max = 8, 16, unicode.MaxRune
	default:
		msg := "unknown escape sequence"
		if ls.Ch < 0 {
			msg = "escape sequence not terminated"
		}
		ls.Error(offs, msg)
		return false
	}

	var x uint32
	for n > 0 {
		d := uint32(DigitVal(ls.Ch))
		if d >= base {
			msg := fmt.Sprintf("illegal character %#U in escape sequence", ls.Ch)
			if ls.Ch < 0 {
				msg = "escape sequence not terminated"
			}
			ls.Error(ls.Pos, msg)
			return false
		}
		x = x*base + d
		ls.NextRune()
		n--
	}

	if x > max || 0xD800 <= x && x < 0xE000 {
		ls.Error(ls.Pos, "escape sequence is invalid Unicode code point")
		return false
	}

	return true
}
