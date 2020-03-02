// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copied and only lightly modified from:
// https://github.com/nickng/bibtex
// Licenced under an Apache-2.0 licence
// and presumably Copyright (c) 2017 by Nick Ng

package bibtex

import (
	"fmt"
	"strings"
)

// Lexer token.
type Token int

const (
	// ILLEGAL stands for an invalid token.
	ILLEGAL Token = iota
)

var eof = rune(0)

// TokenPos is a pair of coordinate to identify start of token.
type TokenPos struct {
	Char  int
	Lines []int
}

func (p TokenPos) String() string {
	return fmt.Sprintf("%d:%d", len(p.Lines)+1, p.Char)
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func isAlpha(ch rune) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z')
}

func isDigit(ch rune) bool {
	return ('0' <= ch && ch <= '9')
}

func isAlphanum(ch rune) bool {
	return isAlpha(ch) || isDigit(ch)
}

func isBareSymbol(ch rune) bool {
	return strings.ContainsRune("-_:./+", ch)
}

// isSymbol returns true if ch is a valid symbol
func isSymbol(ch rune) bool {
	return strings.ContainsRune("!?&*+-./:;<>[]^_`|~@", ch)
}

func isOpenQuote(ch rune) bool {
	return ch == '{' || ch == '"'
}
