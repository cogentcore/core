// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rich

import (
	"fmt"
	"slices"
	"unicode"

	"cogentcore.org/core/text/textpos"
)

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

// NewPlainText returns a new [Text] starting with default style and runes string,
// which can be empty.
func NewPlainText(r []rune) Text {
	return NewText(NewStyle(), r)
}

// NumSpans returns the number of spans in this Text.
func (tx Text) NumSpans() int {
	return len(tx)
}

// Len returns the total number of runes in this Text.
func (tx Text) Len() int {
	n := 0
	for _, s := range tx {
		_, rn := SpanLen(s)
		n += rn
	}
	return n
}

// Range returns the start, end range of indexes into original source
// for given span index.
func (tx Text) Range(span int) (start, end int) {
	ci := 0
	for si, s := range tx {
		_, rn := SpanLen(s)
		if si == span {
			return ci, ci + rn
		}
		ci += rn
	}
	return -1, -1
}

// Index returns the span index, number of style runes at start of span,
// and index into actual runes within the span after style runes,
// for the given logical index into the original source rune slice
// without spans or styling elements.
// If the logical index is invalid for the text returns -1,-1,-1.
func (tx Text) Index(li int) (span, stylen, ridx int) {
	ci := 0
	for si, s := range tx {
		sn, rn := SpanLen(s)
		if li >= ci && li < ci+rn {
			return si, sn, sn + (li - ci)
		}
		ci += rn
	}
	return -1, -1, -1
}

// AtTry returns the rune at given logical index, as in the original
// source rune slice without any styling elements. Returns 0
// and false if index is invalid.
func (tx Text) AtTry(li int) (rune, bool) {
	ci := 0
	for _, s := range tx {
		sn, rn := SpanLen(s)
		if li >= ci && li < ci+rn {
			return s[sn+(li-ci)], true
		}
		ci += rn
	}
	return -1, false
}

// At returns the rune at given logical index into the original
// source rune slice without any styling elements. Returns 0
// if index is invalid. See AtTry for a version that also returns a bool
// indicating whether the index is valid.
func (tx Text) At(li int) rune {
	r, _ := tx.AtTry(li)
	return r
}

// Split returns the raw rune spans without any styles.
// The rune span slices here point directly into the Text rune slices.
// See SplitCopy for a version that makes a copy instead.
func (tx Text) Split() [][]rune {
	ss := make([][]rune, 0, len(tx))
	for _, s := range tx {
		sn, _ := SpanLen(s)
		ss = append(ss, s[sn:])
	}
	return ss
}

// SplitCopy returns the raw rune spans without any styles.
// The rune span slices here are new copies; see also [Text.Split].
func (tx Text) SplitCopy() [][]rune {
	ss := make([][]rune, 0, len(tx))
	for _, s := range tx {
		sn, _ := SpanLen(s)
		ss = append(ss, slices.Clone(s[sn:]))
	}
	return ss
}

// Join returns a single slice of runes with the contents of all span runes.
func (tx Text) Join() []rune {
	ss := make([]rune, 0, tx.Len())
	for _, s := range tx {
		sn, _ := SpanLen(s)
		if sn < len(s) {
			ss = append(ss, s[sn:]...)
		}
	}
	return ss
}

// Span returns the [Style] and []rune content for given span index.
// Returns nil if out of range.
func (tx Text) Span(si int) (*Style, []rune) {
	n := len(tx)
	if si < 0 || si >= n || len(tx[si]) == 0 {
		return nil, nil
	}
	return NewStyleFromRunes(tx[si])
}

// SetSpanStyle sets the style for given span, updating the runes to encode it.
func (tx *Text) SetSpanStyle(si int, nsty *Style) *Text {
	sty, r := tx.Span(si)
	*sty = *nsty
	nr := sty.ToRunes()
	nr = append(nr, r...)
	(*tx)[si] = nr
	return tx
}

// SetSpanRunes sets the runes for given span.
func (tx *Text) SetSpanRunes(si int, r []rune) *Text {
	sty, _ := tx.Span(si)
	nr := sty.ToRunes()
	nr = append(nr, r...)
	(*tx)[si] = nr
	return tx
}

// AddSpan adds a span to the Text using the given Style and runes.
// The Text is modified for convenience in the high-frequency use-case.
// Clone first to avoid changing the original.
func (tx *Text) AddSpan(s *Style, r []rune) *Text {
	nr := s.ToRunes()
	nr = append(nr, r...)
	*tx = append(*tx, nr)
	return tx
}

// AddSpanString adds a span to the Text using the given Style and string content.
// The Text is modified for convenience in the high-frequency use-case.
// Clone first to avoid changing the original.
func (tx *Text) AddSpanString(s *Style, r string) *Text {
	nr := s.ToRunes()
	nr = append(nr, []rune(r)...)
	*tx = append(*tx, nr)
	return tx
}

// InsertSpan inserts a span to the Text at given span index,
// using the given Style and runes.
// The Text is modified for convenience in the high-frequency use-case.
// Clone first to avoid changing the original.
func (tx *Text) InsertSpan(at int, s *Style, r []rune) *Text {
	nr := s.ToRunes()
	nr = append(nr, r...)
	*tx = slices.Insert(*tx, at, nr)
	return tx
}

// SplitSpan splits an existing span at the given logical source index,
// with the span containing that logical index truncated to contain runes
// just before the index, and a new span inserted starting at that index,
// with the remaining contents of the original containing span.
// If that logical index is already the start of a span, or the logical
// index is invalid, nothing happens. Returns the index of span,
// which will be negative if the logical index is out of range.
func (tx *Text) SplitSpan(li int) int {
	si, sn, ri := tx.Index(li)
	if si < 0 {
		return si
	}
	if sn == ri { // already the start
		return si
	}
	nr := slices.Clone((*tx)[si][:sn]) // style runes
	nr = append(nr, (*tx)[si][ri:]...)
	(*tx)[si] = (*tx)[si][:ri] // truncate
	*tx = slices.Insert(*tx, si+1, nr)
	return si + 1
}

// StartSpecial adds a Span of given Special type to the Text,
// using given style and rune text. This creates a new style
// with the special value set, to avoid accidentally repeating
// the start of new specials.
func (tx *Text) StartSpecial(s *Style, special Specials, r []rune) *Text {
	ss := *s
	ss.Special = special
	return tx.AddSpan(&ss, r)
}

// EndSpecial adds an [End] Special to the Text, to terminate the current
// Special. All [Specials] must be terminated with this empty end tag.
func (tx *Text) EndSpecial() *Text {
	s := NewStyle()
	s.Special = End
	return tx.AddSpan(s, nil)
}

// InsertEndSpecial inserts an [End] Special to the Text at given span
// index, to terminate the current Special. All [Specials] must be
// terminated with this empty end tag.
func (tx *Text) InsertEndSpecial(at int) *Text {
	s := NewStyle()
	s.Special = End
	return tx.InsertSpan(at, s, nil)
}

// SpecialRange returns the range of spans for the
// special starting at given span index. Returns -1 if span
// at given index is not a special.
func (tx Text) SpecialRange(si int) textpos.Range {
	sp := RuneToSpecial(tx[si][0])
	if sp == Nothing {
		return textpos.Range{-1, -1}
	}
	depth := 1
	n := len(tx)
	for j := si + 1; j < n; j++ {
		s := RuneToSpecial(tx[j][0])
		switch s {
		case End:
			depth--
			if depth == 0 {
				return textpos.Range{si, j}
			}
		default:
			depth++
		}
	}
	return textpos.Range{-1, -1}
}

// AddLink adds a [Link] special with given url and label text.
// This calls StartSpecial and EndSpecial for you. If the link requires
// further formatting, use those functions separately.
func (tx *Text) AddLink(s *Style, url, label string) *Text {
	ss := *s
	ss.URL = url
	tx.StartSpecial(&ss, Link, []rune(label))
	return tx.EndSpecial()
}

// AddSuper adds a [Super] special with given text.
// This calls StartSpecial and EndSpecial for you. If the Super requires
// further formatting, use those functions separately.
func (tx *Text) AddSuper(s *Style, text string) *Text {
	tx.StartSpecial(s, Super, []rune(text))
	return tx.EndSpecial()
}

// AddSub adds a [Sub] special with given text.
// This calls StartSpecial and EndSpecial for you. If the Sub requires
// further formatting, use those functions separately.
func (tx *Text) AddSub(s *Style, text string) *Text {
	tx.StartSpecial(s, Sub, []rune(text))
	return tx.EndSpecial()
}

// AddMathInline adds a [MathInline] special with given text.
// This calls StartSpecial and EndSpecial for you. If the Math requires
// further formatting, use those functions separately.
func (tx *Text) AddMathInline(s *Style, text string) *Text {
	tx.StartSpecial(s, MathInline, []rune(text))
	return tx.EndSpecial()
}

// AddMathDisplay adds a [MathDisplay] special with given text.
// This calls StartSpecial and EndSpecial for you. If the Math requires
// further formatting, use those functions separately.
func (tx *Text) AddMathDisplay(s *Style, text string) *Text {
	tx.StartSpecial(s, MathDisplay, []rune(text))
	return tx.EndSpecial()
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
		str += "[" + sstr + "]: \"" + string(ss) + "\"\n"
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

func (tx Text) DebugDump() {
	for i := range tx {
		s, r := tx.Span(i)
		fmt.Println(i, len(tx[i]), tx[i])
		fmt.Printf("style: %#v\n", s)
		fmt.Printf("chars: %q\n", string(r))
	}
}

// Clone returns a deep copy clone of the current text, safe for subsequent
// modification without affecting this one.
func (tx Text) Clone() Text {
	ct := make(Text, len(tx))
	for i := range tx {
		ct[i] = slices.Clone(tx[i])
	}
	return ct
}

// SplitSpaces splits this text after first unicode space after non-space.
func (tx *Text) SplitSpaces() {
	txt := tx.Join()
	if len(txt) == 0 {
		return
	}
	prevSp := unicode.IsSpace(txt[0])
	for i, r := range txt {
		isSp := unicode.IsSpace(r)
		if prevSp && isSp {
			continue
		}
		if isSp {
			prevSp = true
			tx.SplitSpan(i + 1) // already doesn't re-split
		} else {
			prevSp = false
		}
	}
}
