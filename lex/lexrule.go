// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/token"
)

// LexRule operates on the text input to produce the lexical tokens
// it is assembled into a lexical grammar structure to perform lexing
//
// Lexing is done line-by-line -- you must push and pop states to
// coordinate across multiple lines, e.g., for multi-line comments
//
// In general it is best to keep lexing as simple as possible and
// leave the more complex things for the parsing step.
type LexRule struct {
	ki.Node
	Token     token.Tokens `desc:"the token value that this rule generates -- use None for non-terminals"`
	Match     Matches      `desc:"the lexical match that we look for to engage this rule"`
	Off       int          `desc:"offset into the input to look for a match: 0 = current char, 1 = next one, etc"`
	String    string       `desc:"if action is LexMatch, this is the string we match"`
	Acts      []Actions    `desc:"the action(s) to perform, in order, if there is a match -- these are performed prior to iterating over child nodes"`
	PushState string       `desc:"the state to push if our action is PushState -- note that State matching is on String, not this value"`
}

var KiT_LexRule = kit.Types.AddType(&LexRule{}, LexRuleProps)

// Lex tries to apply rule to given input state, returns true if matched, false if not
func (lr *LexRule) Lex(ls *State) bool {
	if !lr.IsMatch(ls) {
		return false
	}
	st := ls.Pos // starting pos that we're consuming
	for _, act := range lr.Acts {
		lr.DoAct(ls, act)
	}
	ed := ls.Pos // our ending state
	if ed > st {
		ls.Add(lr.Token, st, ed)
	}
	if !lr.HasChildren() {
		return true
	}

	// now we iterate over our kids
	for _, klri := range lr.Kids {
		klr := klri.Embed(KiT_LexRule).(*LexRule)
		if klr.Lex(ls) { // first to match takes it -- order matters!
			break
		}
	}
	return true // regardless of kids, we matched
}

// IsMatch tests if the rule matches for current input state, returns true if so, false if not
func (lr *LexRule) IsMatch(ls *State) bool {
	if lr.IsRoot() { // root always matches
		return true
	}
	switch lr.Match {
	case String:
		sz := len(lr.String)
		str, ok := ls.String(lr.Off, sz)
		if !ok {
			return false
		}
		if str != lr.String {
			return false
		}
		return true
	case Letter:
		rn, ok := ls.Rune(lr.Off)
		if !ok {
			return false
		}
		return IsLetter(rn)
	case Digit:
		rn, ok := ls.Rune(lr.Off)
		if !ok {
			return false
		}
		return IsDigit(rn)
	case WhiteSpace:
		rn, ok := ls.Rune(lr.Off)
		if !ok {
			return false
		}
		return IsWhiteSpace(rn)
	case CurState:
		return ls.CurState() == lr.String
	}
	return false
}

// DoAct performs given action
func (lr *LexRule) DoAct(ls *State, act Actions) {
	switch act {
	case Next:
		ls.Next(len(lr.String))
	case Number:
		ls.Next(len(lr.String))
	case StringQuote:
		ls.Next(len(lr.String))
	case StringDblQuote:
		ls.Next(len(lr.String))
	case StringBacktick:
		ls.Next(len(lr.String))
	case EOL:
		ls.Pos = len(ls.Src)
	case PushState:
		ls.PushState(lr.PushState)
	case PopState:
		ls.PopState()
	}
}

//////////////////////////////////////////////////////////////////////////////
//  Matches, Actions

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

// Actions are lexing actions to perform
type Actions int

//go:generate stringer -type=Actions

var KiT_Actions = kit.Enums.AddEnum(ActionsN, false, nil)

func (ev Actions) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Actions) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// The lexical acts
const (
	// Next means advance input position to the next character(s) after the matched characters
	Next Actions = iota

	// Number means read in an entire number -- the token type will automatically be
	// set to the actual type of number that was read in, and position advanced to just after
	Number

	// StringQuote means read in an entire string enclosed in single-quotes,
	// with proper skipping of escaped, position advanced to just after
	StringQuote

	// StringDblQuote means read in an entire string enclosed in double-quotes,
	// with proper skipping of escaped, position advanced to just after
	StringDblQuote

	// StringBacktick means read in an entire string enclosed in backtick's
	// with proper skipping of escaped, position advanced to just after
	StringBacktick

	// EOL means read till the end of the line (e.g., for single-line comments)
	EOL

	// PushState means push the given state value onto the state stack
	PushState

	// PopState means pop given state value off the state stack
	PopState

	ActionsN
)

var LexRuleProps = ki.Props{
	// "CallMethods": ki.PropSlice{
	// 	{"SaveAs", ki.Props{
	// 		"Args": ki.PropSlice{
	// 			{"File Name", ki.Props{
	// 				"default-field": "Filename",
	// 			}},
	// 		},
	// 	}},
	// },
}
