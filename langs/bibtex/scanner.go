// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copied and only lightly modified from:
// https://github.com/nickng/bibtex
// Licenced under an Apache-2.0 licence
// and presumably Copyright (c) 2017 by Nick Ng

package bibtex

import (
	"bufio"
	"bytes"
	"io"
	"strconv"
	"strings"
)

var parseField bool

// Scanner is a lexical scanner
type Scanner struct {
	r   *bufio.Reader
	pos TokenPos
}

// NewScanner returns a new instance of Scanner.
func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: bufio.NewReader(r), pos: TokenPos{Char: 0, Lines: []int{}}}
}

// read reads the next rune from the buffered reader.
// Returns the rune(0) if an error occurs (or io.eof is returned).
func (s *Scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	if ch == '\n' {
		s.pos.Lines = append(s.pos.Lines, s.pos.Char)
		s.pos.Char = 0
	} else {
		s.pos.Char++
	}
	return ch
}

// unread places the previously read rune back on the reader.
func (s *Scanner) unread() {
	_ = s.r.UnreadRune()
	if s.pos.Char == 0 {
		s.pos.Char = s.pos.Lines[len(s.pos.Lines)-1]
		s.pos.Lines = s.pos.Lines[:len(s.pos.Lines)-1]
	} else {
		s.pos.Char--
	}
}

// Scan returns the next token and literal value.
func (s *Scanner) Scan() (tok Token, lit string) {
	ch := s.read()
	if isWhitespace(ch) {
		s.ignoreWhitespace()
		ch = s.read()
	}
	if isAlphanum(ch) {
		s.unread()
		return s.scanIdent()
	}
	switch ch {
	case eof:
		return 0, ""
	case '@':
		return ATSIGN, string(ch)
	case ':':
		return COLON, string(ch)
	case ',':
		parseField = false // reset parseField if reached end of field.
		return COMMA, string(ch)
	case '=':
		parseField = true // set parseField if = sign outside quoted or ident.
		return EQUAL, string(ch)
	case '"':
		return s.scanQuoted()
	case '{':
		if parseField {
			return s.scanBraced()
		}
		return LBRACE, string(ch)
	case '}':
		if parseField { // reset parseField if reached end of entry.
			parseField = false
		}
		return RBRACE, string(ch)
	case '#':
		return POUND, string(ch)
	case ' ':
		s.ignoreWhitespace()
	}
	return ILLEGAL, string(ch)
}

// scanIdent categorises a string to one of three categories.
func (s *Scanner) scanIdent() (tok Token, lit string) {
	switch ch := s.read(); ch {
	case '"':
		return s.scanQuoted()
	case '{':
		return s.scanBraced()
	default:
		s.unread() // Not open quote/brace.
		return s.scanBare()
	}
}

func (s *Scanner) scanBare() (Token, string) {
	var buf bytes.Buffer
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isAlphanum(ch) && !isBareSymbol(ch) || isWhitespace(ch) {
			s.unread()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}
	str := buf.String()
	if strings.ToLower(str) == "comment" {
		return COMMENT, str
	} else if strings.ToLower(str) == "preamble" {
		return PREAMBLE, str
	} else if strings.ToLower(str) == "string" {
		return STRING, str
	} else if _, err := strconv.Atoi(str); err == nil && parseField { // Special case for numeric
		return IDENT, str
	}
	return BAREIDENT, str
}

// scanBraced parses a braced string, like {this}.
func (s *Scanner) scanBraced() (Token, string) {
	var buf bytes.Buffer
	var macro bool
	brace := 1
	for {
		if ch := s.read(); ch == eof {
			break
		} else if ch == '\\' {
			_, _ = buf.WriteRune(ch)
			macro = true
		} else if ch == '{' {
			_, _ = buf.WriteRune(ch)
			brace++
		} else if ch == '}' {
			brace--
			macro = false
			if brace == 0 { // Balances open brace.
				return IDENT, buf.String()
			}
			_, _ = buf.WriteRune(ch)
		} else if ch == '@' {
			if macro {
				_, _ = buf.WriteRune(ch)
				// } else {
				// 	log.Printf("%s: %s", ErrUnexpectedAtsign, buf.String())
			}
		} else if isWhitespace(ch) {
			_, _ = buf.WriteRune(ch)
			macro = false
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}
	return ILLEGAL, buf.String()
}

// scanQuoted parses a quoted string, like "this".
func (s *Scanner) scanQuoted() (Token, string) {
	var buf bytes.Buffer
	brace := 0
	for {
		if ch := s.read(); ch == eof {
			break
		} else if ch == '{' {
			brace++
		} else if ch == '}' {
			brace--
		} else if ch == '"' {
			if brace == 0 { // Matches open quote, unescaped
				return IDENT, buf.String()
			}
			_, _ = buf.WriteRune(ch)
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}
	return ILLEGAL, buf.String()
}

// ignoreWhitespace consumes the current rune and all contiguous whitespace.
func (s *Scanner) ignoreWhitespace() {
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isWhitespace(ch) {
			s.unread()
			break
		}
	}
}
