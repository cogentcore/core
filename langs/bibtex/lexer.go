// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copied and only lightly modified from:
// https://github.com/nickng/bibtex
// Licenced under an Apache-2.0 licence
// and presumably Copyright (c) 2017 by Nick Ng

//go:generate goyacc -p bibtex -o bibtex.y.go bibtex.y

package bibtex

import "io"

// Lexer for bibtex.
type Lexer struct {
	scanner *Scanner
	Errors  chan error
}

// NewLexer returns a new yacc-compatible lexer.
func NewLexer(r io.Reader) *Lexer {
	return &Lexer{scanner: NewScanner(r), Errors: make(chan error, 1)}
}

// Lex is provided for yacc-compatible parser.
func (l *Lexer) Lex(yylval *bibtexSymType) int {
	token, strval := l.scanner.Scan()
	yylval.strval = strval
	return int(token)
}

// Error handles error.
func (l *Lexer) Error(err string) {
	l.Errors <- &ErrParse{Err: err, Pos: l.scanner.pos}
}
