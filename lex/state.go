// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/goki/ki/nptime"
	"github.com/goki/pi/token"
)

// LangLexer looks up lexer for given language -- impl in parent pi package
// so we need the interface
type LangLexer interface {
	// LexerByName returns the top-level lex.Rule for given language (case invariant lookup)
	LexerByName(lang string) *Rule
}

// TheLangLexer is the instance of LangLexer interface used to lookup lexers
// for languages -- is set in pi/langs.go
var TheLangLexer LangLexer

// lex.State is the state maintained for lexing
type State struct {
	Filename  string      `desc:"the current file being lex'd"`
	KeepWS    bool        `desc:"if true, record whitespace tokens -- else ignore"`
	Src       []rune      `desc:"the current line of source being processed"`
	Lex       Line        `desc:"the lex output for this line"`
	Comments  Line        `desc:"the comments output for this line -- kept separately"`
	Pos       int         `desc:"the current rune char position within the line"`
	Ln        int         `desc:"the line within overall source that we're operating on (0 indexed)"`
	Ch        rune        `desc:"the current rune read by NextRune"`
	Stack     Stack       `desc:"state stack"`
	LastName  string      `desc:"the last name that was read"`
	GuestLex  *Rule       `desc:"a guest lexer that can be installed for managing a different language type, e.g., quoted text in markdown files"`
	SaveStack Stack       `desc:"copy of stack at point when guest lexer was installed -- restore when popped"`
	Time      nptime.Time `desc:"time stamp for lexing -- set at start of new lex process"`
	Errs      ErrorList   `desc:"any error messages accumulated during lexing specifically"`
}

// Init initializes the state at start of parsing
func (ls *State) Init() {
	ls.GuestLex = nil
	ls.Stack.Reset()
	ls.Ln = 0
	ls.SetLine(nil)
	ls.SaveStack = nil
	ls.Errs.Reset()
}

// SetLine sets a new line for parsing and initializes the lex output and pos
func (ls *State) SetLine(src []rune) {
	ls.Src = src
	ls.Lex = nil
	ls.Comments = nil
	ls.Pos = 0
}

// LineString returns the current lex output as tagged source
func (ls *State) LineString() string {
	return fmt.Sprintf("[%v,%v]: %v", ls.Ln, ls.Pos, ls.Lex.TagSrc(ls.Src))
}

// Error adds a lexing error at given position
func (ls *State) Error(pos int, msg string, rule *Rule) {
	ls.Errs.Add(Pos{ls.Ln, pos}, ls.Filename, "Lexer: "+msg, string(ls.Src), rule)
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
func (ls *State) Add(tok token.KeyToken, st, ed int) {
	if tok.Tok == token.TextWhitespace && !ls.KeepWS {
		return
	}
	lxl := &ls.Lex
	if tok.Tok.Cat() == token.Comment {
		lxl = &ls.Comments
	}
	sz := len(*lxl)
	if sz > 0 && tok.Tok.CombineRepeats() {
		lst := &(*lxl)[sz-1]
		if lst.Tok.Tok == tok.Tok && lst.Ed == st {
			lst.Ed = ed
			return
		}
	}
	lx := (*lxl).AddLex(tok, st, ed)
	lx.Time = ls.Time
}

// PushState pushes state onto stack
func (ls *State) PushState(st string) {
	ls.Stack.Push(st)
}

// CurState returns the current state
func (ls *State) CurState() string {
	return ls.Stack.Top()
}

// PopState pops state off of stack
func (ls *State) PopState() string {
	return ls.Stack.Pop()
}

// MatchState returns true if the current state matches the string
func (ls *State) MatchState(st string) bool {
	sz := len(ls.Stack)
	if sz == 0 {
		return false
	}
	return ls.Stack[sz-1] == st
}

// ReadNameTmp reads a standard alpha-numeric_ name and returns it.
// Does not update the lexing position -- a "lookahead" name read
func (ls *State) ReadNameTmp(off int) string {
	cp := ls.Pos
	ls.Pos += off
	ls.ReadName()
	ls.Pos = cp
	return ls.LastName
}

// ReadName reads a standard alpha-numeric_ name -- saves in LastName
func (ls *State) ReadName() {
	str := ""
	sz := len(ls.Src)
	for ls.Pos < sz {
		rn := ls.Src[ls.Pos]
		if IsLetter(rn) || IsDigit(rn) {
			str += string(rn)
			ls.Pos++
		} else {
			break
		}
	}
	ls.LastName = str
}

// NextSrcLine returns the next line of text
func (ls *State) NextSrcLine() string {
	if ls.AtEol() {
		return "EOL"
	}
	return string(ls.Src[ls.Pos:])
}

// ReadUntil reads until given string(s) -- does do depth tracking if looking for a bracket
// open / close kind of symbol
// until string options are separated by | -- use two || in a row for a literal |
// terminates at end of line without error
func (ls *State) ReadUntil(until string) {
	ustrs := strings.Split(until, "|")
	sz := len(ls.Src)
	got := false
	depth := 0
	match := rune(0)
	if len(ustrs) == 1 && len(ustrs[0]) == 1 {
		switch ustrs[0][0] {
		case '}':
			match = '{'
		case ')':
			match = '('
		case ']':
			match = '['
		}
	}
	for ls.NextRune() {
		if match != 0 {
			if ls.Ch == match {
				depth++
				continue
			} else if ls.Ch == rune(ustrs[0][0]) {
				if depth > 0 {
					depth--
					continue
				}
			}
			if depth > 0 {
				continue
			}
		}
		for _, un := range ustrs {
			usz := len(un)
			if usz == 0 { // ||
				if ls.Ch == '|' {
					ls.NextRune() // move past
					break
				}
			} else {
				ep := ls.Pos + usz
				sm := string(ls.Src[ls.Pos:ep])
				if ep <= sz && sm == un {
					ls.Pos += usz
					got = true
					break
				}
			}
		}
		if got {
			break
		}
	}
}

// ReadNumber reads a number of any sort, returning the type of the number
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
				ls.Error(offs, "illegal hexadecimal number", nil)
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
				ls.Error(offs, "illegal octal number", nil)
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
			ls.Error(offs, "illegal floating-point exponent", nil)
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
		if ch == delim {
			ls.NextRune() // move past
			break
		}
		if ch == '\\' {
			ls.ReadEscape(delim)
		}
		if !ls.NextRune() {
			ls.Error(offs, "string literal not terminated", nil)
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
		ls.Error(offs, msg, nil)
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
			ls.Error(ls.Pos, msg, nil)
			return false
		}
		x = x*base + d
		ls.NextRune()
		n--
	}

	if x > max || 0xD800 <= x && x < 0xE000 {
		ls.Error(ls.Pos, "escape sequence is invalid Unicode code point", nil)
		return false
	}

	return true
}
