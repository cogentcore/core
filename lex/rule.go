// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"fmt"
	"reflect"

	"github.com/goki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/token"
)

// Lexer is the interface type for lexers -- likely not necessary except is essential
// for defining the BaseIface for gui in making new nodes
type Lexer interface {
	ki.Ki

	// Validate checks for any errors in the rules and issues warnings,
	// returns true if valid (no err) and false if invalid (errs)
	Validate() bool

	// Lex tries to apply rule to given input state, returns true if matched, false if not
	Lex(ls *State) *Rule

	// AsLexRule returns object as a lex.Rule
	AsLexRule() *Rule
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
	Desc      string       `desc:"description / comments about this rule"`
	Token     token.Tokens `desc:"the token value that this rule generates -- use None for non-terminals"`
	Match     Matches      `desc:"the lexical match that we look for to engage this rule"`
	String    string       `desc:"if action is LexMatch, this is the string we match"`
	Off       int          `desc:"offset into the input to look for a match: 0 = current char, 1 = next one, etc"`
	Acts      []Actions    `desc:"the action(s) to perform, in order, if there is a match -- these are performed prior to iterating over child nodes"`
	PushState string       `desc:"the state to push if our action is PushState -- note that State matching is on String, not this value"`
	TokEff    token.Tokens `view:"-" json:"-" desc:"effective token based on input -- e.g., for number is the type of number"`
	MatchLen  int          `view:"-" json:"-" desc:"length of source that matched -- if Next is called, this is what will be skipped to"`
}

var KiT_Rule = kit.Types.AddType(&Rule{}, RuleProps)

func (lr *Rule) BaseIface() reflect.Type {
	return reflect.TypeOf((*Lexer)(nil)).Elem()
}

func (lr *Rule) AsLexRule() *Rule {
	return lr.This().Embed(KiT_Rule).(*Rule)
}

// Validate checks for any errors in the rules and issues warnings,
// returns true if valid (no err) and false if invalid (errs)
func (lr *Rule) Validate() bool {
	hasErr := false
	if !lr.IsRoot() {
		switch lr.Match {
		case String:
			if len(lr.String) == 0 {
				hasErr = true
				fmt.Printf("lex.Rule: match = String but String is empty, in: %v\n", lr.PathUnique())
			}
		case CurState:
			for _, act := range lr.Acts {
				if act == Next {
					hasErr = true
					fmt.Printf("lex.Rule: match = CurState cannot have Action = Next -- no src match, in: %v\n", lr.PathUnique())
				}
			}
			if len(lr.String) == 0 {
				fmt.Printf("lex.Rule: match = CurState must have state to match in String -- is empty, in: %v\n", lr.PathUnique())
			}
			if len(lr.PushState) > 0 {
				fmt.Printf("lex.Rule: match = CurState has non-empty PushState -- must have state to match in String instead, in: %v\n", lr.PathUnique())
			}
		}
	}

	if !lr.HasChildren() && len(lr.Acts) == 0 {
		hasErr = true
		fmt.Printf("lex.Rule: has no children and no action -- does nothing, in: %v\n", lr.PathUnique())
	}

	hasPos := false
	for _, act := range lr.Acts {
		if act >= Name && act <= EOL {
			hasPos = true
		}
		if act == Next && hasPos {
			hasErr = true
			fmt.Printf("lex.Rule: action = Next incompatible with action that reads item such as Name, Number, Quoted, in: %v\n", lr.PathUnique())
		}
	}
	// now we iterate over our kids
	for _, klri := range lr.Kids {
		klr := klri.Embed(KiT_Rule).(*Rule)
		if !klr.Validate() {
			hasErr = true
		}
	}
	return hasErr
}

// Lex tries to apply rule to given input state, returns lowest-level rule that matched, nil if none
func (lr *Rule) Lex(ls *State) *Rule {
	if !lr.IsMatch(ls) {
		return nil
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
		return lr
	}

	// now we iterate over our kids
	for _, klri := range lr.Kids {
		klr := klri.Embed(KiT_Rule).(*Rule)
		if mrule := klr.Lex(ls); mrule != nil { // first to match takes it -- order matters!
			return mrule
		}
	}

	// if kids don't match and we don't have any actions, we are just a grouper
	// and thus we depend entirely on kids matching
	if len(lr.Acts) == 0 {
		return nil
	}

	return lr
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
		lr.MatchLen = lr.Off + sz
		return true
	case Letter:
		rn, ok := ls.Rune(lr.Off)
		if !ok {
			return false
		}
		if IsLetter(rn) {
			lr.MatchLen = lr.Off + 1
			return true
		}
		return false
	case Digit:
		rn, ok := ls.Rune(lr.Off)
		if !ok {
			return false
		}
		if IsDigit(rn) {
			lr.MatchLen = lr.Off + 1
			return true
		}
		return false
	case WhiteSpace:
		rn, ok := ls.Rune(lr.Off)
		if !ok {
			return false
		}
		if IsWhiteSpace(rn) {
			lr.MatchLen = lr.Off + 1
			return true
		}
		return false
	case CurState:
		if ls.CurState() == lr.String {
			lr.MatchLen = 0
			return true
		}
		return false
	case AnyRune:
		_, ok := ls.Rune(lr.Off)
		if !ok {
			return false
		}
		lr.MatchLen = lr.Off + 1
		return true
	}
	return false
}

// DoAct performs given action
func (lr *Rule) DoAct(ls *State, act Actions) {
	switch act {
	case Next:
		ls.Next(lr.MatchLen)
	case Name:
		ls.ReadName()
	case Number:
		lr.TokEff = ls.ReadNumber()
	case Quoted:
		ls.ReadQuoted()
	case QuotedRaw:
		ls.ReadQuoted() // todo: raw!
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
