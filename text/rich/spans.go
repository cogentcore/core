// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rich

import "slices"

// Spans is the basic rich text representation, with spans of []rune unicode characters
// that share a common set of text styling properties, which are represented
// by the first rune(s) in each span. If custom colors are used, they are encoded
// after the first style and size runes.
// This compact and efficient representation can be Join'd back into the raw
// unicode source, and indexing by rune index in the original is fast.
// It provides a GPU-compatible representation, and is the text equivalent of
// the [ppath.Path] encoding.
type Spans [][]rune

// NewSpans returns a new spans starting with given style and runes string,
// which can be empty.
func NewSpans(s *Style, r ...rune) Spans {
	sp := Spans{}
	sp.Add(s, r)
	return sp
}

// Index represents the [Span][Rune] index of a given rune.
// The Rune index can be either the actual index for [Spans], taking
// into account the leading style rune(s), or the logical index
// into a [][]rune type with no style runes, depending on the context.
type Index struct { //types:add
	Span, Rune int
}

// NStyleRunes specifies the base number of style runes at the start
// of each span: style + size.
const NStyleRunes = 2

// NumSpans returns the number of spans in this Spans.
func (sp Spans) NumSpans() int {
	return len(sp)
}

// Len returns the total number of runes in this Spans.
func (sp Spans) Len() int {
	n := 0
	for _, s := range sp {
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
func (sp Spans) Range(span int) (start, end int) {
	ci := 0
	for si, s := range sp {
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
func (sp Spans) Index(li int) Index {
	ci := 0
	for si, s := range sp {
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
func (sp Spans) At(li int) rune {
	i := sp.Index(li)
	if i.Span < 0 {
		return 0
	}
	return sp[i.Span][i.Rune]
}

// AtTry returns the rune at given logical index, as in the original
// source rune slice without any styling elements. Returns 0
// and false if index is invalid.
func (sp Spans) AtTry(li int) (rune, bool) {
	i := sp.Index(li)
	if i.Span < 0 {
		return 0, false
	}
	return sp[i.Span][i.Rune], true
}

// Split returns the raw rune spans without any styles.
// The rune span slices here point directly into the Spans rune slices.
// See SplitCopy for a version that makes a copy instead.
func (sp Spans) Split() [][]rune {
	rn := make([][]rune, 0, len(sp))
	for _, s := range sp {
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
// The rune span slices here are new copies; see also [Spans.Split].
func (sp Spans) SplitCopy() [][]rune {
	rn := make([][]rune, 0, len(sp))
	for _, s := range sp {
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
func (sp Spans) Join() []rune {
	rn := make([]rune, 0, sp.Len())
	for _, s := range sp {
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

// Add adds a span to the Spans using the given Style and runes.
func (sp *Spans) Add(s *Style, r []rune) *Spans {
	nr := s.ToRunes()
	nr = append(nr, r...)
	*sp = append(*sp, nr)
	return sp
}

func (sp Spans) String() string {
	str := ""
	for _, rs := range sp {
		s := &Style{}
		ss := s.FromRunes(rs)
		sstr := s.String()
		str += "[" + sstr + "]: " + string(ss) + "\n"
	}
	return str
}
