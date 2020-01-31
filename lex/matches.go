// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"unicode"
	"unicode/utf8"

	"github.com/goki/ki/kit"
)

// Matches are what kind of lexing matches to make
type Matches int

//go:generate stringer -type=Matches

var KiT_Matches = kit.Enums.AddEnum(MatchesN, kit.NotBitFlag, nil)

func (ev Matches) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Matches) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// Matching rules
const (
	// String means match a specific string as given in the rule
	// Note: this only looks for the string with no constraints on
	// what happens after this string -- use StrName to match entire names
	String Matches = iota

	// StrName means match a specific string that is a complete alpha-numeric
	// string (including underbar _) with some other char at the end
	// must use this for all keyword matches to ensure that it isn't just
	// the start of a longer name
	StrName

	// Match any letter, including underscore
	Letter

	// Match digit 0-9
	Digit

	// Match any white space (space, tab) -- input is already broken into lines
	WhiteSpace

	// CurState means match current state value set by a PushState action, using String value in rule
	// all CurState cases must generally be first in list of rules so they can preempt
	// other rules when the state is active
	CurState

	// AnyRune means match any rune -- use this as the last condition where other terminators
	// come first!
	AnyRune

	MatchesN
)

// MatchPos are special positions for a match to occur
type MatchPos int

//go:generate stringer -type=MatchPos

var KiT_MatchPos = kit.Enums.AddEnum(MatchPosN, kit.NotBitFlag, nil)

func (ev MatchPos) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *MatchPos) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// Matching position rules
const (
	// AnyPos matches at any position
	AnyPos MatchPos = iota

	// StartOfLine matches at start of line
	StartOfLine

	// EndOfLine matches at end of line
	EndOfLine

	// MiddleOfLine matches not at the start or end
	MiddleOfLine

	// StartOfWord matches at start of word
	StartOfWord

	// EndOfWord matches at end of word
	EndOfWord

	// MiddleOfWord matches not at the start or end
	MiddleOfWord

	MatchPosN
)

//////////////////////////////////////////////////////////////////////////////
// Match functions -- see also state for more complex ones

func IsLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}

func IsDigit(ch rune) bool {
	return '0' <= ch && ch <= '9' || ch >= utf8.RuneSelf && unicode.IsDigit(ch)
}

func IsLetterOrDigit(ch rune) bool {
	return IsLetter(ch) || IsDigit(ch)
}

func IsWhiteSpace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}
