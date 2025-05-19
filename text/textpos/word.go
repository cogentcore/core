// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textpos

import "unicode"

// RuneIsWordBreak returns true if given rune counts as a word break
// for the purposes of selecting words.
func RuneIsWordBreak(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsSymbol(r) || unicode.IsPunct(r)
}

// IsWordBreak defines what counts as a word break for the purposes of selecting words.
// r1 is the rune in question, r2 is the rune past r1 in the direction you are moving.
// Pass -1 for r2 if there is no rune past r1.
func IsWordBreak(r1, r2 rune) bool {
	if r2 == -1 {
		return RuneIsWordBreak(r1)
	}
	if unicode.IsSpace(r1) || unicode.IsSymbol(r1) {
		return true
	}
	if unicode.IsPunct(r1) && r1 != rune('\'') {
		return true
	}
	if unicode.IsPunct(r1) && r1 == rune('\'') {
		return unicode.IsSpace(r2) || unicode.IsSymbol(r2) || unicode.IsPunct(r2)
	}
	return false
}

// WordAt returns the range for a word within given text starting at given
// position index. If the current position is a word break then go to next
// break after the first non-break.
func WordAt(txt []rune, pos int) Range {
	var rg Range
	sz := len(txt)
	if sz == 0 {
		return rg
	}
	if pos < 0 {
		pos = 0
	}
	if pos >= sz {
		pos = sz - 1
	}
	rg.Start = pos
	if !RuneIsWordBreak(txt[rg.Start]) {
		for rg.Start > 0 {
			if RuneIsWordBreak(txt[rg.Start-1]) {
				break
			}
			rg.Start--
		}
		rg.End = pos + 1
		for rg.End < sz {
			if RuneIsWordBreak(txt[rg.End]) {
				break
			}
			rg.End++
		}
		return rg
	}
	// keep the space start -- go to next space..
	rg.End = pos + 1
	for rg.End < sz {
		if !RuneIsWordBreak(txt[rg.End]) {
			break
		}
		rg.End++
	}
	for rg.End < sz {
		if RuneIsWordBreak(txt[rg.End]) {
			break
		}
		rg.End++
	}
	return rg
}

// ForwardWord moves position index forward by words, for given
// number of steps. Returns the number of steps actually moved,
// given the amount of text available.
func ForwardWord(txt []rune, pos, steps int) (wpos, nstep int) {
	sz := len(txt)
	if sz == 0 {
		return 0, 0
	}
	if pos >= sz-1 {
		return sz - 1, 0
	}
	if pos < 0 {
		pos = 0
	}
	for range steps {
		if pos == sz-1 {
			break
		}
		ch := pos
		for ch < sz-1 { // if on a wb, go past
			if !IsWordBreak(txt[ch], txt[ch+1]) {
				break
			}
			ch++
		}
		for ch < sz-1 { // now go to next wb
			if IsWordBreak(txt[ch], txt[ch+1]) {
				break
			}
			ch++
		}
		pos = ch
		nstep++
	}
	return pos, nstep
}

// BackwardWord moves position index backward by words, for given
// number of steps. Returns the number of steps actually moved,
// given the amount of text available.
func BackwardWord(txt []rune, pos, steps int) (wpos, nstep int) {
	sz := len(txt)
	if sz == 0 {
		return 0, 0
	}
	if pos <= 0 {
		return 0, 0
	}
	if pos >= sz {
		pos = sz - 1
	}
	for range steps {
		if pos == 0 {
			break
		}
		ch := pos
		for ch > 0 { // if on a wb, go past
			if !IsWordBreak(txt[ch], txt[ch-1]) {
				break
			}
			ch--
		}
		for ch > 0 { // now go to next wb
			if IsWordBreak(txt[ch], txt[ch-1]) {
				break
			}
			ch--
		}
		pos = ch
		nstep++
	}
	return pos, nstep
}
