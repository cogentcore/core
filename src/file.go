// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package src provides source file structures
package src

import "github.com/goki/pi/lex"

// Pos is a position within the source file
type Pos struct {
	Ln int
	Ch int
}

// Reg is a contiguous region within the source file
type Reg struct {
	St Pos `desc:"starting position of region"`
	Ed Pos `desc:"ending position of region"`
}

// File contains the contents of the file being parsed -- all kept in
// memory, and represented by Line as runes, so that positions in
// the file are directly convertible to indexes in Lines structure
type File struct {
	Lines [][]rune   `desc:"contents of the file as lines of runes"`
	Lexs  []lex.Line `desc:"lex'd version of the lines -- allocated to size of Lines"`
}

// SetSrc sets the source to given content, and alloc Lexs
func (fl *File) SetSrc(src [][]rune) {
	fl.Lines = src
	fl.Lexs = make([]lex.Line, len(src))
}

// NLines returns the number of lines in source
func (fl *File) NLines() int {
	return len(fl.Lines)
}

// SetLexs sets the lex output for given line -- does a copy
func (fl *File) SetLexs(ln int, lexs lex.Line) {
	fl.Lexs[ln] = lexs.Clone()
}
