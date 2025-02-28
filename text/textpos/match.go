// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textpos

// Match records one match for search within file, positions in runes.
type Match struct {

	// Region of the match. Column positions are in runes.
	Region Region

	// Text surrounding the match, at most MatchContext on either side
	// (within a single line).
	Text []rune

	// TextMatch has the Range within the Text where the match is.
	TextMatch Range
}

func (m *Match) String() string {
	return m.Region.String() + ": " + string(m.Text)
}

// MatchContext is how much text to include on either side of the match.
var MatchContext = 30

// NewMatch returns a new Match entry for given rune line with match starting
// at st and ending before ed, on given line
func NewMatch(rn []rune, st, ed, ln int) Match {
	sz := len(rn)
	reg := NewRegion(ln, st, ln, ed)
	cist := max(st-MatchContext, 0)
	cied := min(ed+MatchContext, sz)
	sctx := rn[cist:st]
	fstr := rn[st:ed]
	ectx := rn[ed:cied]
	tlen := len(sctx) + len(fstr) + len(ectx)
	txt := make([]rune, tlen)
	copy(txt, sctx)
	ti := st - cist
	copy(txt[ti:], fstr)
	ti += len(fstr)
	copy(txt[ti:], ectx)
	return Match{Region: reg, Text: txt, TextMatch: Range{Start: len(sctx), End: len(sctx) + len(fstr)}}
}

const (
	// IgnoreCase is passed to search functions to indicate case should be ignored
	IgnoreCase = true

	// UseCase is passed to search functions to indicate case is relevant
	UseCase = false
)
