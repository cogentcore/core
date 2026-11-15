// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copied and only lightly modified from:
// https://github.com/nickng/bibtex
// Licensed under an Apache-2.0 license
// and presumably Copyright (c) 2017 by Nick Ng

package bibtex

import (
	"bufio"
	"bytes"
	"io"
	"strconv"
	"strings"
)

// Scanner is a lexical scanner
type Scanner struct {
	r   *bufio.Reader
	pos TokenPos

	// parseField is whether a field value is being scanned (after an =),
	// in which a brace starts a braced string.
	parseField bool

	// depth is the depth of the braces or parentheses of the current entry:
	// the text outside of entries (at depth 0, before an @) is ignored,
	// as comments, as in BibTeX.
	depth int

	// inEntry is whether an entry has been started with an @.
	inEntry bool

	// lastTok is the last token returned.
	lastTok Token

	// pending are tokens to return before scanning more.
	pending []pendingToken
}

// pendingToken is a token with its literal value.
type pendingToken struct {
	tok Token
	lit string
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
	tok, lit = s.scan()
	s.lastTok = tok
	return
}

// scan returns the next token and literal value (see [Scanner.Scan]).
func (s *Scanner) scan() (tok Token, lit string) {
	if len(s.pending) > 0 {
		p := s.pending[0]
		s.pending = s.pending[1:]
		return p.tok, p.lit
	}
	ch := s.read()
	if !s.inEntry { // text outside of entries is ignored, up to an @
		for ch != '@' && ch != eof {
			ch = s.read()
		}
	}
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
		s.inEntry = true
		return ATSIGN, string(ch)
	case ':':
		return COLON, string(ch)
	case ',':
		s.parseField = false // reset parseField if reached end of field.
		return COMMA, string(ch)
	case '=':
		s.parseField = true // set parseField if = sign outside quoted or ident.
		return EQUAL, string(ch)
	case '"':
		return s.scanQuoted()
	case '{', '(':
		if ch == '{' && s.parseField {
			return s.scanBraced()
		}
		open, close := Token(LBRACE), Token(RBRACE)
		if ch == '(' {
			open, close = LPAREN, RPAREN
		}
		if s.lastTok == COMMENT { // the text of a comment, as it is
			text := s.scanUntilClose(ch)
			s.pending = append(s.pending, pendingToken{IDENT, text}, pendingToken{close, ""})
			s.inEntry = false
			return open, string(ch)
		}
		s.depth++
		if s.lastTok == PREAMBLE { // its content is a value, as after an =
			s.parseField = true
		}
		return open, string(ch)
	case '}', ')':
		s.parseField = false // reset parseField if reached end of entry.
		s.depth--
		if s.depth <= 0 {
			s.depth = 0
			s.inEntry = false
		}
		if ch == ')' {
			return RPAREN, string(ch)
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
		ch := s.read()
		if ch == eof {
			break
		}
		if !isAlphanum(ch) && !isBareSymbol(ch) || isWhitespace(ch) {
			s.unread()
			break
		}
		_, _ = buf.WriteRune(ch)
	}
	str := buf.String()
	if strings.ToLower(str) == "comment" {
		return COMMENT, str
	} else if strings.ToLower(str) == "preamble" {
		return PREAMBLE, str
	} else if strings.ToLower(str) == "string" {
		return STRING, str
	} else if _, err := strconv.Atoi(str); err == nil && s.parseField { // Special case for numeric
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

// scanUntilClose returns the text up to the given open brace or
// parenthesis that matches it, which is read too.
func (s *Scanner) scanUntilClose(open rune) string {
	close := '}'
	if open == '(' {
		close = ')'
	}
	var buf bytes.Buffer
	depth := 1
	for {
		ch := s.read()
		if ch == eof {
			break
		}
		if ch == open {
			depth++
		} else if ch == close {
			depth--
			if depth == 0 {
				break
			}
		}
		buf.WriteRune(ch)
	}
	return buf.String()
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
