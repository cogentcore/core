// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rich

import "slices"

// Text is the basic rich text representation, with spans of []rune unicode characters
// that share a common set of text styling properties, which are represented
// by the first rune(s) in each span. If custom colors are used, they are encoded
// after the first style and size runes.
// This compact and efficient representation can be Join'd back into the raw
// unicode source, and indexing by rune index in the original is fast.
// It provides a GPU-compatible representation, and is the text equivalent of
// the [ppath.Path] encoding.
type Text [][]rune

// NewText returns a new [Text] starting with given style and runes string,
// which can be empty.
func NewText(s *Style, r []rune) Text {
	tx := Text{}
	tx.AddSpan(s, r)
	return tx
}

// Index represents the [Span][Rune] index of a given rune.
// The Rune index can be either the actual index for [Text], taking
// into account the leading style rune(s), or the logical index
// into a [][]rune type with no style runes, depending on the context.
type Index struct { //types:add
	Span, Rune int
}

// NStyleRunes specifies the base number of style runes at the start
// of each span: style + size.
const NStyleRunes = 2

// NumText returns the number of spans in this Text.
func (tx Text) NumText() int {
	return len(tx)
}

// Len returns the total number of runes in this Text.
func (tx Text) Len() int {
	n := 0
	for _, s := range tx {
		sn := len(s)
		rs := s[0]
		nc := NumColors(rs)
		ns := max(0, sn-(NStyleRunes+nc))
		n += ns
	}
	return n
}

// Range returns the start, end range of indexes into original source
// for given span index.
func (tx Text) Range(span int) (start, end int) {
	ci := 0
	for si, s := range tx {
		sn := len(s)
		rs := s[0]
		nc := NumColors(rs)
		ns := max(0, sn-(NStyleRunes+nc))
		if si == span {
			return ci, ci + ns
		}
		ci += ns
	}
	return -1, -1
}

// Index returns the span, rune slice [Index] for the given logical
// index, as in the original source rune slice without spans or styling elements.
// If the logical index is invalid for the text, the returned index is -1,-1.
func (tx Text) Index(li int) Index {
	ci := 0
	for si, s := range tx {
		sn := len(s)
		if sn == 0 {
			continue
		}
		rs := s[0]
		nc := NumColors(rs)
		ns := max(0, sn-(NStyleRunes+nc))
		if li >= ci && li < ci+ns {
			return Index{Span: si, Rune: NStyleRunes + nc + (li - ci)}
		}
		ci += ns
	}
	return Index{Span: -1, Rune: -1}
}

// At returns the rune at given logical index, as in the original
// source rune slice without any styling elements. Returns 0
// if index is invalid. See AtTry for a version that also returns a bool
// indicating whether the index is valid.
func (tx Text) At(li int) rune {
	i := tx.Index(li)
	if i.Span < 0 {
		return 0
	}
	return tx[i.Span][i.Rune]
}

// AtTry returns the rune at given logical index, as in the original
// source rune slice without any styling elements. Returns 0
// and false if index is invalid.
func (tx Text) AtTry(li int) (rune, bool) {
	i := tx.Index(li)
	if i.Span < 0 {
		return 0, false
	}
	return tx[i.Span][i.Rune], true
}

// Split returns the raw rune spans without any styles.
// The rune span slices here point directly into the Text rune slices.
// See SplitCopy for a version that makes a copy instead.
func (tx Text) Split() [][]rune {
	rn := make([][]rune, 0, len(tx))
	for _, s := range tx {
		sn := len(s)
		if sn == 0 {
			continue
		}
		rs := s[0]
		nc := NumColors(rs)
		rn = append(rn, s[NStyleRunes+nc:])
	}
	return rn
}

// SplitCopy returns the raw rune spans without any styles.
// The rune span slices here are new copies; see also [Text.Split].
func (tx Text) SplitCopy() [][]rune {
	rn := make([][]rune, 0, len(tx))
	for _, s := range tx {
		sn := len(s)
		if sn == 0 {
			continue
		}
		rs := s[0]
		nc := NumColors(rs)
		rn = append(rn, slices.Clone(s[NStyleRunes+nc:]))
	}
	return rn
}

// Join returns a single slice of runes with the contents of all span runes.
func (tx Text) Join() []rune {
	rn := make([]rune, 0, tx.Len())
	for _, s := range tx {
		sn := len(s)
		if sn == 0 {
			continue
		}
		rs := s[0]
		nc := NumColors(rs)
		rn = append(rn, s[NStyleRunes+nc:]...)
	}
	return rn
}

// AddSpan adds a span to the Text using the given Style and runes.
func (tx *Text) AddSpan(s *Style, r []rune) *Text {
	nr := s.ToRunes()
	nr = append(nr, r...)
	*tx = append(*tx, nr)
	return tx
}

// AddRunes adds given runes to current span.
// If no existing span, then a new default one is made.
func (tx *Text) AddRunes(r []rune) *Text {
	n := len(*tx)
	if n == 0 {
		return tx.AddSpan(NewStyle(), r)
	}
	(*tx)[n-1] = append((*tx)[n-1], r...)
	return tx
}

func (tx Text) String() string {
	str := ""
	for _, rs := range tx {
		s := &Style{}
		ss := s.FromRunes(rs)
		sstr := s.String()
		str += "[" + sstr + "]: " + string(ss) + "\n"
	}
	return str
}

// Join joins multiple texts into one text. Just appends the spans.
func Join(txts ...Text) Text {
	nt := Text{}
	for _, tx := range txts {
		nt = append(nt, tx...)
	}
	return nt
}
