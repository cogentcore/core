// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"reflect"

	"github.com/goki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/token"
)

// Lexer is the interface type for lexers -- likely not necessary except is essential
// for defining the BaseIface for gui in making new nodes
type Lexer interface {
	ki.Ki
	// Lex tries to apply rule to given input state, returns true if matched, false if not
	Lex(ls *State) bool
}

// lex.Rule operates on the text input to produce the lexical tokens
// it is assembled into a lexical grammar structure to perform lexing
//
// Lexing is done line-by-line -- you must push and pop states to
// coordinate across multiple lines, e.g., for multi-line comments
//
// In general it is best to keep lexing as simple as possible and
// leave the more complex things for the parsing step.
type Rule struct {
	ki.Node
	Token     token.Tokens `desc:"the token value that this rule generates -- use None for non-terminals"`
	TokEff    token.Tokens `view:"-" json:"-" desc:"effective token based on input -- e.g., for number is the type of number"`
	Match     Matches      `desc:"the lexical match that we look for to engage this rule"`
	Off       int          `desc:"offset into the input to look for a match: 0 = current char, 1 = next one, etc"`
	String    string       `desc:"if action is LexMatch, this is the string we match"`
	Acts      []Actions    `desc:"the action(s) to perform, in order, if there is a match -- these are performed prior to iterating over child nodes"`
	PushState string       `desc:"the state to push if our action is PushState -- note that State matching is on String, not this value"`
}

var KiT_Rule = kit.Types.AddType(&Rule{}, RuleProps)

func (lr *Rule) BaseIface() reflect.Type {
	return reflect.TypeOf((*Lexer)(nil)).Elem()
}

// Lex tries to apply rule to given input state, returns true if matched, false if not
func (lr *Rule) Lex(ls *State) bool {
	if !lr.IsMatch(ls) {
		return false
	}
	st := ls.Pos // starting pos that we're consuming
	lr.TokEff = lr.Token
	for _, act := range lr.Acts {
		lr.DoAct(ls, act)
	}
	ed := ls.Pos // our ending state
	if ed > st {
		ls.Add(lr.TokEff, st, ed)
	}
	if !lr.HasChildren() {
		return true
	}

	// now we iterate over our kids
	for _, klri := range lr.Kids {
		klr := klri.Embed(KiT_Rule).(*Rule)
		if klr.Lex(ls) { // first to match takes it -- order matters!
			break
		}
	}
	return true // regardless of kids, we matched
}

// IsMatch tests if the rule matches for current input state, returns true if so, false if not
func (lr *Rule) IsMatch(ls *State) bool {
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
func (lr *Rule) DoAct(ls *State, act Actions) {
	switch act {
	case Next:
		ls.Next(len(lr.String))
	case Name:
		ls.ReadName()
	case Number:
		lr.TokEff = ls.ReadNumber()
	case StringQuote:
		ls.ReadString()
	case StringDblQuote:
		ls.ReadString()
	case StringBacktick:
		ls.ReadString() // todo: multi-line
	case EOL:
		ls.Pos = len(ls.Src)
	case PushState:
		ls.PushState(lr.PushState)
	case PopState:
		ls.PopState()
	}
}

var RuleProps = ki.Props{
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
