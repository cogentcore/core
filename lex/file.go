// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package src provides source file structures
package lex

import (
	"fmt"
)

// Pos is a position within the source file -- it is recorded always in 0, 0
// offset positions, but is converted into 1,1 offset for public consumption
// Ch positions are always in runes, not bytes
type Pos struct {
	Ln int
	Ch int
}

// String satisfies the fmt.Stringer interferace
func (ps Pos) String() string {
	s := fmt.Sprintf("%d", ps.Ln+1)
	if ps.Ch != 0 {
		s += fmt.Sprintf(":%d", ps.Ch)
	}
	return s
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
	Filename string   `desc:"the current file being lex'd"`
	Lines    [][]rune `desc:"contents of the file as lines of runes"`
	Lexs     []Line   `desc:"lex'd version of the lines -- allocated to size of Lines"`
}

// SetSrc sets the source to given content, and alloc Lexs
func (fl *File) SetSrc(src [][]rune, fname string) {
	fl.Filename = fname
	fl.Lines = src
	fl.Lexs = make([]Line, len(src))
}

// NLines returns the number of lines in source
func (fl *File) NLines() int {
	return len(fl.Lines)
}

// SetLexs sets the lex output for given line -- does a copy
func (fl *File) SetLexs(ln int, lexs Line) {
	fl.Lexs[ln] = lexs.Clone()
}

// LexTagSrcLn returns the lex'd tagged source line for given line
func (fl *File) LexTagSrcLn(ln int) string {
	return fl.Lexs[ln].TagSrc(fl.Lines[ln])
}

// LexTagSrc returns the lex'd tagged source for entire source
func (fl *File) LexTagSrc() string {
	txt := ""
	nlines := fl.NLines()
	for ln := 0; ln < nlines; ln++ {
		txt += fl.LexTagSrcLn(ln) + "\n"
	}
	return txt
}
