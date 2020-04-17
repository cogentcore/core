// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textbuf

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"regexp"
	"unicode/utf8"

	"github.com/goki/ki/ints"
	"github.com/goki/ki/runes"
	"github.com/goki/pi/lex"
)

// Match records one match for search within file, positions in runes
type Match struct {
	Reg  Region `desc:"region surrounding the match -- column positions are in runes, not bytes"`
	Text []byte `desc:"text surrounding the match, at most FileSearchContext on either side (within a single line)"`
}

// SearchContext is how much text to include on either side of the search match
var SearchContext = 30

var mst = []byte("<mark>")
var mstsz = len(mst)
var med = []byte("</mark>")
var medsz = len(med)

// NewMatch returns a new Match entry for given rune line with match starting
// at st and ending before ed, on given line
func NewMatch(rn []rune, st, ed, ln int) Match {
	sz := len(rn)
	reg := NewRegion(ln, st, ln, ed)
	cist := ints.MaxInt(st-SearchContext, 0)
	cied := ints.MinInt(ed+SearchContext, sz)
	sctx := []byte(string(rn[cist:st]))
	fstr := []byte(string(rn[st:ed]))
	ectx := []byte(string(rn[ed:cied]))
	tlen := mstsz + medsz + len(sctx) + len(fstr) + len(ectx)
	txt := make([]byte, tlen)
	copy(txt, sctx)
	ti := st - cist
	copy(txt[ti:], mst)
	ti += mstsz
	copy(txt[ti:], fstr)
	ti += len(fstr)
	copy(txt[ti:], med)
	ti += medsz
	copy(txt[ti:], ectx)
	return Match{Reg: reg, Text: txt}
}

const (
	// IgnoreCase is passed to search functions to indicate case should be ignored
	IgnoreCase = true

	// UseCase is passed to search functions to indicate case is relevant
	UseCase = false
)

// SearchRuneLines looks for a string (no regexp) within lines of runes,
// with given case-sensitivity returning number of occurrences
// and specific match position list.  Column positions are in runes.
func SearchRuneLines(src [][]rune, find []byte, ignoreCase bool) (int, []Match) {
	fr := bytes.Runes(find)
	fsz := len(fr)
	if fsz == 0 {
		return 0, nil
	}
	cnt := 0
	var matches []Match
	for ln, rn := range src {
		sz := len(rn)
		ci := 0
		for ci < sz {
			var i int
			if ignoreCase {
				i = runes.IndexFold(rn[ci:], fr)
			} else {
				i = runes.Index(rn[ci:], fr)
			}
			if i < 0 {
				break
			}
			i += ci
			ci = i + fsz
			mat := NewMatch(rn, i, ci, ln)
			matches = append(matches, mat)
			cnt++
		}
	}
	return cnt, matches
}

// SearchLexItems looks for a string (no regexp),
// as entire lexically tagged items,
// with given case-sensitivity returning number of occurrences
// and specific match position list.  Column positions are in runes.
func SearchLexItems(src [][]rune, lexs []lex.Line, find []byte, ignoreCase bool) (int, []Match) {
	fr := bytes.Runes(find)
	fsz := len(fr)
	if fsz == 0 {
		return 0, nil
	}
	cnt := 0
	var matches []Match
	mx := ints.MinInt(len(src), len(lexs))
	for ln := 0; ln < mx; ln++ {
		rln := src[ln]
		lxln := lexs[ln]
		for _, lx := range lxln {
			sz := lx.Ed - lx.St
			if sz != fsz {
				continue
			}
			rn := rln[lx.St:lx.Ed]
			var i int
			if ignoreCase {
				i = runes.IndexFold(rn, fr)
			} else {
				i = runes.Index(rn, fr)
			}
			if i < 0 {
				continue
			}
			mat := NewMatch(rln, lx.St, lx.Ed, ln)
			matches = append(matches, mat)
			cnt++
		}
	}
	return cnt, matches
}

// Search looks for a string (no regexp) from an io.Reader input stream,
// using given case-sensitivity.
// Returns number of occurrences and specific match position list.
// Column positions are in runes.
func Search(reader io.Reader, find []byte, ignoreCase bool) (int, []Match) {
	fsz := len(find)
	if fsz == 0 {
		return 0, nil
	}
	fr := bytes.Runes(find)
	cnt := 0
	var matches []Match
	scan := bufio.NewScanner(reader)
	ln := 0
	for scan.Scan() {
		rn := bytes.Runes(scan.Bytes()) // note: temp -- must copy -- convert to runes anyway
		sz := len(rn)
		ci := 0
		for ci < sz {
			var i int
			if ignoreCase {
				i = runes.IndexFold(rn[ci:], fr)
			} else {
				i = runes.Index(rn[ci:], fr)
			}
			if i < 0 {
				break
			}
			i += ci
			ci = i + fsz
			mat := NewMatch(rn, i, ci, ln)
			matches = append(matches, mat)
			cnt++
		}
		ln++
	}
	if err := scan.Err(); err != nil {
		// note: we expect: bufio.Scanner: token too long  when reading binary files
		// not worth printing here.  otherwise is very reliable.
		// log.Printf("giv.FileSearch error: %v\n", err)
	}
	return cnt, matches
}

// SearchFile looks for a string (no regexp) within a file, in a
// case-sensitive way, returning number of occurrences and specific match
// position list -- column positions are in runes.
func SearchFile(filename string, find []byte, ignoreCase bool) (int, []Match) {
	fp, err := os.Open(filename)
	if err != nil {
		log.Printf("textbuf.SearchFile: open error: %v\n", err)
		return 0, nil
	}
	defer fp.Close()
	return Search(fp, find, ignoreCase)
}

// SearchRegexp looks for a string (using regexp) from an io.Reader input stream.
// Returns number of occurrences and specific match position list.
// Column positions are in runes.
func SearchRegexp(reader io.Reader, re *regexp.Regexp) (int, []Match) {
	cnt := 0
	var matches []Match
	scan := bufio.NewScanner(reader)
	ln := 0
	for scan.Scan() {
		b := scan.Bytes() // note: temp -- must copy -- convert to runes anyway
		fi := re.FindAllIndex(b, -1)
		if fi == nil {
			ln++
			continue
		}
		sz := len(b)
		ri := make([]int, sz+1) // byte indexes to rune indexes
		rn := make([]rune, 0, sz)
		for i, w := 0, 0; i < sz; i += w {
			r, wd := utf8.DecodeRune(b[i:])
			w = wd
			ri[i] = len(rn)
			rn = append(rn, r)
		}
		ri[sz] = len(rn)
		for _, f := range fi {
			st := f[0]
			ed := f[1]
			mat := NewMatch(rn, ri[st], ri[ed], ln)
			matches = append(matches, mat)
			cnt++
		}
		ln++
	}
	if err := scan.Err(); err != nil {
		// note: we expect: bufio.Scanner: token too long  when reading binary files
		// not worth printing here.  otherwise is very reliable.
		// log.Printf("giv.FileSearch error: %v\n", err)
	}
	return cnt, matches
}

// SearchFileRegexp looks for a string (using regexp) within a file,
// returning number of occurrences and specific match
// position list -- column positions are in runes.
func SearchFileRegexp(filename string, re *regexp.Regexp) (int, []Match) {
	fp, err := os.Open(filename)
	if err != nil {
		log.Printf("textbuf.SearchFile: open error: %v\n", err)
		return 0, nil
	}
	defer fp.Close()
	return SearchRegexp(fp, re)
}

// SearchByteLinesRegexp looks for a regexp within lines of bytes,
// with given case-sensitivity returning number of occurrences
// and specific match position list.  Column positions are in runes.
func SearchByteLinesRegexp(src [][]byte, re *regexp.Regexp) (int, []Match) {
	cnt := 0
	var matches []Match
	for ln, b := range src {
		fi := re.FindAllIndex(b, -1)
		if fi == nil {
			continue
		}
		sz := len(b)
		ri := make([]int, sz+1) // byte indexes to rune indexes
		rn := make([]rune, 0, sz)
		for i, w := 0, 0; i < sz; i += w {
			r, wd := utf8.DecodeRune(b[i:])
			w = wd
			ri[i] = len(rn)
			rn = append(rn, r)
		}
		ri[sz] = len(rn)
		for _, f := range fi {
			st := f[0]
			ed := f[1]
			mat := NewMatch(rn, ri[st], ri[ed], ln)
			matches = append(matches, mat)
			cnt++
		}
	}
	return cnt, matches
}
