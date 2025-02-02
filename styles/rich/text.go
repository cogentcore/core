// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rich

import "slices"

// Text is a rich text representation, with spans of []rune unicode characters
// that share a common set of text styling properties, which are represented
// by the first rune(s) in each span. If custom colors are used, they are encoded
// after the first style rune.
// This compact and efficient representation can be Join'd back into the raw
// unicode source, and indexing by rune index in the original is fast and efficient.
type Text [][]rune

// Index represents the [Span][Rune] index of a given rune.
// The Rune index can be either the actual index for [Text], taking
// into account the leading style rune(s), or the logical index
// into a [][]rune type with no style runes, depending on the context.
type Index struct { //types:add
	Span, Rune int
}

// NumSpans returns the number of spans in this Text.
func (t Text) NumSpans() int {
	return len(t)
}

// Len returns the total number of runes in this Text.
func (t Text) Len() int {
	n := 0
	for _, s := range t {
		sn := len(s)
		if sn == 0 {
			continue
		}
		rs := s[0]
		nc := RuneToDecoration(rs).NumColors()
		ns := max(0, sn-(1+nc))
		n += ns
	}
	return n
}

// Index returns the span, rune slice [Index] for the given logical
// index, as in the original source rune slice without spans or styling elements.
// If the logical index is invalid for the text, the returned index is -1,-1.
func (t Text) Index(li int) Index {
	ci := 0
	for si, s := range t {
		sn := len(s)
		if sn == 0 {
			continue
		}
		rs := s[0]
		nc := RuneToDecoration(rs).NumColors()
		ns := max(0, sn-(1+nc))
		if li >= ci && li < ci+ns {
			return Index{Span: si, Rune: 1 + nc + (li - ci)}
		}
		ci += ns
	}
	return Index{Span: -1, Rune: -1}
}

// At returns the rune at given logical index, as in the original
// source rune slice without any styling elements. Returns 0
// if index is invalid. See AtTry for a version that also returns a bool
// indicating whether the index is valid.
func (t Text) At(li int) rune {
	i := t.Index(li)
	if i.Span < 0 {
		return 0
	}
	return t[i.Span][i.Rune]
}

// AtTry returns the rune at given logical index, as in the original
// source rune slice without any styling elements. Returns 0
// and false if index is invalid.
func (t Text) AtTry(li int) (rune, bool) {
	i := t.Index(li)
	if i.Span < 0 {
		return 0, false
	}
	return t[i.Span][i.Rune], true
}

// Split returns the raw rune spans without any styles.
// The rune span slices here point directly into the Text rune slices.
// See SplitCopy for a version that makes a copy instead.
func (t Text) Split() [][]rune {
	sp := make([][]rune, 0, len(t))
	for _, s := range t {
		sn := len(s)
		if sn == 0 {
			continue
		}
		rs := s[0]
		nc := NumColors(rs)
		sp = append(sp, s[1+nc:])
	}
	return sp
}

// SplitCopy returns the raw rune spans without any styles.
// The rune span slices here are new copies; see also [Text.Split].
func (t Text) SplitCopy() [][]rune {
	sp := make([][]rune, 0, len(t))
	for _, s := range t {
		sn := len(s)
		if sn == 0 {
			continue
		}
		rs := s[0]
		nc := NumColors(rs)
		sp = append(sp, slices.Clone(s[1+nc:]))
	}
	return sp
}

// Join returns a single slice of runes with the contents of all span runes.
func (t Text) Join() []rune {
	sp := make([]rune, 0, t.Len())
	for _, s := range t {
		sn := len(s)
		if sn == 0 {
			continue
		}
		rs := s[0]
		nc := NumColors(rs)
		sp = append(sp, s[1+nc:]...)
	}
	return sp
}

// Add adds a span to the Text using the given Style and runes.
func (t *Text) Add(s *Style, r []rune) {
	nr := s.ToRunes()
	nr = append(nr, r...)
	*t = append(*t, nr)
}
