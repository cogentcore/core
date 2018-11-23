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

var KiT_Matches = kit.Enums.AddEnum(MatchesN, false, nil)

func (ev Matches) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Matches) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// Matching rules
const (
	// String means match a specific string as given in the rule
	String Matches = iota

	// Match any letter, including underscore
	Letter

	// Match digit 0-9
	Digit

	// Match any white space (space, tab) -- input is already broken into lines
	WhiteSpace

	// CurState means match current state value set by a PushState action, using String value in rule
	CurState

	MatchesN
)

//////////////////////////////////////////////////////////////////////////////
// Match functions

func IsLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}

func IsDigit(ch rune) bool {
	return '0' <= ch && ch <= '9' || ch >= utf8.RuneSelf && unicode.IsDigit(ch)
}

func IsWhiteSpace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}
