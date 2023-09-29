// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/tabwriter"
	"unicode"

	"goki.dev/glop/indent"
	"goki.dev/ki/v2"
	"goki.dev/pi/v2/token"
)

// Lexer is the interface type for lexers -- likely not necessary except is essential
// for defining the BaseIface for gui in making new nodes
type Lexer interface {
	ki.Ki

	// Compile performs any one-time compilation steps on the rule
	Compile(ls *State) bool

	// Validate checks for any errors in the rules and issues warnings,
	// returns true if valid (no err) and false if invalid (errs)
	Validate(ls *State) bool

	// Lex tries to apply rule to given input state, returns true if matched, false if not
	Lex(ls *State) *Rule

	// AsLexRule returns object as a lex.Rule
	AsLexRule() *Rule
}

// lex.Rule operates on the text input to produce the lexical tokens.
//
// Lexing is done line-by-line -- you must push and pop states to
// coordinate across multiple lines, e.g., for multi-line comments.
//
// There is full access to entire line and you can decide based on future
// (offset) characters.
//
// In general it is best to keep lexing as simple as possible and
// leave the more complex things for the parsing step.
type Rule struct {
	ki.Node

	// disable this rule -- useful for testing and exploration
	Off bool `desc:"disable this rule -- useful for testing and exploration"`

	// description / comments about this rule
	Desc string `desc:"description / comments about this rule"`

	// the token value that this rule generates -- use None for non-terminals
	Token token.Tokens `desc:"the token value that this rule generates -- use None for non-terminals"`

	// the lexical match that we look for to engage this rule
	Match Matches `desc:"the lexical match that we look for to engage this rule"`

	// position where match can occur
	Pos MatchPos `desc:"position where match can occur"`

	// if action is LexMatch, this is the string we match
	String string `desc:"if action is LexMatch, this is the string we match"`

	// offset into the input to look for a match: 0 = current char, 1 = next one, etc
	Offset int `desc:"offset into the input to look for a match: 0 = current char, 1 = next one, etc"`

	// adjusts the size of the region (plus or minus) that is processed for the Next action -- allows broader and narrower matching relative to tagging
	SizeAdj int `desc:"adjusts the size of the region (plus or minus) that is processed for the Next action -- allows broader and narrower matching relative to tagging"`

	// the action(s) to perform, in order, if there is a match -- these are performed prior to iterating over child nodes
	Acts []Actions `desc:"the action(s) to perform, in order, if there is a match -- these are performed prior to iterating over child nodes"`

	// string(s) for ReadUntil action -- will read until any of these strings are found -- separate different options with | -- if you need to read until a literal | just put two || in a row and that will show up as a blank, which is interpreted as a literal |
	Until string `desc:"string(s) for ReadUntil action -- will read until any of these strings are found -- separate different options with | -- if you need to read until a literal | just put two || in a row and that will show up as a blank, which is interpreted as a literal |"`

	// the state to push if our action is PushState -- note that State matching is on String, not this value
	PushState string `desc:"the state to push if our action is PushState -- note that State matching is on String, not this value"`

	// create an optimization map for this rule, which must be a parent with children that all match against a Name string -- this reads the Name and directly activates the associated rule with that String, without having to iterate through them -- use this for keywords etc -- produces a SIGNIFICANT speedup for long lists of keywords.
	NameMap bool `desc:"create an optimization map for this rule, which must be a parent with children that all match against a Name string -- this reads the Name and directly activates the associated rule with that String, without having to iterate through them -- use this for keywords etc -- produces a SIGNIFICANT speedup for long lists of keywords."`

	// [view: -] length of source that matched -- if Next is called, this is what will be skipped to
	MatchLen int `view:"-" json:"-" xml:"-" desc:"length of source that matched -- if Next is called, this is what will be skipped to"`

	// NameMap lookup map -- created during Compile
	NmMap map[string]*Rule `inactive:"+" json:"-" xml:"-" desc:"NameMap lookup map -- created during Compile"`
}

func (lr *Rule) BaseIface() reflect.Type {
	return reflect.TypeOf((*Lexer)(nil)).Elem()
}

func (lr *Rule) AsLexRule() *Rule {
	return lr.This().(*Rule)
}

// CompileAll is called on the top-level Rule to compile all nodes.
// returns true if everything is ok
func (lr *Rule) CompileAll(ls *State) bool {
	allok := false
	lr.WalkPre(func(k ki.Ki) bool {
		lri := k.(*Rule)
		ok := lri.Compile(ls)
		if !ok {
			allok = false
		}
		return true
	})
	return allok
}

// Compile performs any one-time compilation steps on the rule
// returns false if there are any problems.
func (lr *Rule) Compile(ls *State) bool {
	if lr.Off {
		lr.SetProp("inactive", true)
	} else {
		lr.DeleteProp("inactive")
	}
	valid := true
	lr.ComputeMatchLen(ls)
	if lr.NameMap {
		if !lr.CompileNameMap(ls) {
			valid = false
		}
	}
	return valid
}

// CompileNameMap compiles name map -- returns false if there are problems.
func (lr *Rule) CompileNameMap(ls *State) bool {
	valid := true
	lr.NmMap = make(map[string]*Rule, len(lr.Kids))
	for _, klri := range lr.Kids {
		klr := klri.(*Rule)
		if !klr.Validate(ls) {
			valid = false
		}
		if klr.String == "" {
			ls.Error(0, "CompileNameMap: must have non-empty String to match", lr)
			valid = false
			continue
		}
		if _, has := lr.NmMap[klr.String]; has {
			ls.Error(0, fmt.Sprintf("CompileNameMap: multiple rules have the same string name: %v -- must be unique!", klr.String), lr)
			valid = false
		} else {
			lr.NmMap[klr.String] = klr
		}
	}
	return valid
}

// Validate checks for any errors in the rules and issues warnings,
// returns true if valid (no err) and false if invalid (errs)
func (lr *Rule) Validate(ls *State) bool {
	valid := true
	if !ki.IsRoot(lr) {
		switch lr.Match {
		case StrName:
			fallthrough
		case String:
			if len(lr.String) == 0 {
				valid = false
				ls.Error(0, "match = String or StrName but String is empty", lr)
			}
		case CurState:
			for _, act := range lr.Acts {
				if act == Next {
					valid = false
					ls.Error(0, "match = CurState cannot have Action = Next -- no src match", lr)
				}
			}
			if len(lr.String) == 0 {
				ls.Error(0, "match = CurState must have state to match in String -- is empty", lr)
			}
			if len(lr.PushState) > 0 {
				ls.Error(0, "match = CurState has non-empty PushState -- must have state to match in String instead", lr)
			}
		}
	}

	if !lr.HasChildren() && len(lr.Acts) == 0 {
		valid = false
		ls.Error(0, "rule has no children and no action -- does nothing", lr)
	}

	hasPos := false
	for _, act := range lr.Acts {
		if act >= Name && act <= EOL {
			hasPos = true
		}
		if act == Next && hasPos {
			valid = false
			ls.Error(0, "action = Next incompatible with action that reads item such as Name, Number, Quoted", lr)
		}
	}

	if lr.Token.Cat() == token.Keyword && lr.Match != StrName {
		valid = false
		ls.Error(0, "Keyword token must use StrName to match entire name", lr)
	}

	// now we iterate over our kids
	for _, klri := range lr.Kids {
		klr := klri.(*Rule)
		if !klr.Validate(ls) {
			valid = false
		}
	}
	return valid
}

// ComputeMatchLen computes MatchLen based on match type
func (lr *Rule) ComputeMatchLen(ls *State) {
	switch lr.Match {
	case String:
		sz := len(lr.String)
		lr.MatchLen = lr.Offset + sz + lr.SizeAdj
	case StrName:
		sz := len(lr.String)
		lr.MatchLen = lr.Offset + sz + lr.SizeAdj
	case Letter:
		lr.MatchLen = lr.Offset + 1 + lr.SizeAdj
	case Digit:
		lr.MatchLen = lr.Offset + 1 + lr.SizeAdj
	case WhiteSpace:
		lr.MatchLen = lr.Offset + 1 + lr.SizeAdj
	case CurState:
		lr.MatchLen = 0
	case AnyRune:
		lr.MatchLen = lr.Offset + 1 + lr.SizeAdj
	}
}

// LexStart is called on the top-level lex node to start lexing process for one step
func (lr *Rule) LexStart(ls *State) *Rule {
	hasGuest := ls.GuestLex != nil
	cpos := ls.Pos
	lxsz := len(ls.Lex)
	mrule := lr
	for _, klri := range lr.Kids {
		klr := klri.(*Rule)
		if mrule = klr.Lex(ls); mrule != nil { // first to match takes it -- order matters!
			break
		}
	}
	if hasGuest && ls.GuestLex != nil && lr != ls.GuestLex {
		ls.Pos = cpos // backup and undo what the standard rule did, and redo with guest..
		// this is necessary to allow main lex to detect when to turn OFF the guest!
		if lxsz > 0 {
			ls.Lex = ls.Lex[:lxsz]
		} else {
			ls.Lex = nil
		}
		mrule = ls.GuestLex.LexStart(ls)
	}
	if !ls.AtEol() && cpos == ls.Pos {
		ls.Error(cpos, "did not advance position -- need more rules to match current input", lr)
		return nil
	}
	return mrule
}

// Lex tries to apply rule to given input state, returns lowest-level rule that matched, nil if none
func (lr *Rule) Lex(ls *State) *Rule {
	if lr.Off || !lr.IsMatch(ls) {
		return nil
	}
	st := ls.Pos // starting pos that we're consuming
	tok := token.KeyToken{Tok: lr.Token}
	for _, act := range lr.Acts {
		lr.DoAct(ls, act, &tok)
	}
	ed := ls.Pos // our ending state
	if ed > st {
		if tok.Tok.IsKeyword() {
			tok.Key = lr.String //  if we matched, this is it
		}
		ls.Add(tok, st, ed)
	}
	if !lr.HasChildren() {
		return lr
	}

	if lr.NameMap && lr.NmMap != nil {
		nm := ls.ReadNameTmp(lr.Offset)
		klr, ok := lr.NmMap[nm]
		if ok {
			if mrule := klr.Lex(ls); mrule != nil { // should!
				return mrule
			}
		}
	} else {
		// now we iterate over our kids
		for _, klri := range lr.Kids {
			klr := klri.(*Rule)
			if mrule := klr.Lex(ls); mrule != nil { // first to match takes it -- order matters!
				return mrule
			}
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
	if !lr.IsMatchPos(ls) {
		return false
	}

	switch lr.Match {
	case String:
		sz := len(lr.String)
		str, ok := ls.String(lr.Offset, sz)
		if !ok {
			return false
		}
		if str != lr.String {
			return false
		}
		return true
	case StrName:
		nm := ls.ReadNameTmp(lr.Offset)
		if nm != lr.String {
			return false
		}
		return true
	case Letter:
		rn, ok := ls.Rune(lr.Offset)
		if !ok {
			return false
		}
		if IsLetter(rn) {
			return true
		}
		return false
	case Digit:
		rn, ok := ls.Rune(lr.Offset)
		if !ok {
			return false
		}
		if IsDigit(rn) {
			return true
		}
		return false
	case WhiteSpace:
		rn, ok := ls.Rune(lr.Offset)
		if !ok {
			return false
		}
		if IsWhiteSpace(rn) {
			return true
		}
		return false
	case CurState:
		if ls.MatchState(lr.String) {
			return true
		}
		return false
	case AnyRune:
		_, ok := ls.Rune(lr.Offset)
		if !ok {
			return false
		}
		return true
	}
	return false
}

// IsMatchPos tests if the rule matches position
func (lr *Rule) IsMatchPos(ls *State) bool {
	lsz := len(ls.Src)
	switch lr.Pos {
	case AnyPos:
		return true
	case StartOfLine:
		return ls.Pos == 0
	case EndOfLine:
		tsz := lr.TargetLen(ls)
		return ls.Pos == lsz-1-tsz
	case MiddleOfLine:
		if ls.Pos == 0 {
			return false
		}
		tsz := lr.TargetLen(ls)
		return ls.Pos != lsz-1-tsz
	case StartOfWord:
		return ls.Pos == 0 || unicode.IsSpace(ls.Src[ls.Pos-1])
	case EndOfWord:
		tsz := lr.TargetLen(ls)
		ep := ls.Pos + tsz
		return ep == lsz || (ep+1 < lsz && unicode.IsSpace(ls.Src[ep+1]))
	case MiddleOfWord:
		if ls.Pos == 0 || unicode.IsSpace(ls.Src[ls.Pos-1]) {
			return false
		}
		tsz := lr.TargetLen(ls)
		ep := ls.Pos + tsz
		if ep == lsz || (ep+1 < lsz && unicode.IsSpace(ls.Src[ep+1])) {
			return false
		}
		return true
	}
	return true
}

// TargetLen returns the length of the target including offset
func (lr *Rule) TargetLen(ls *State) int {
	switch lr.Match {
	case StrName:
		fallthrough
	case String:
		sz := len(lr.String)
		return lr.Offset + sz
	case Letter:
		return lr.Offset + 1
	case Digit:
		return lr.Offset + 1
	case WhiteSpace:
		return lr.Offset + 1
	case AnyRune:
		return lr.Offset + 1
	case CurState:
		return 0
	}
	return 0
}

// DoAct performs given action
func (lr *Rule) DoAct(ls *State, act Actions, tok *token.KeyToken) {
	switch act {
	case Next:
		ls.Next(lr.MatchLen)
	case Name:
		ls.ReadName()
	case Number:
		tok.Tok = ls.ReadNumber()
	case Quoted:
		ls.ReadQuoted()
	case QuotedRaw:
		ls.ReadQuoted() // todo: raw!
	case EOL:
		ls.Pos = len(ls.Src)
	case ReadUntil:
		ls.ReadUntil(lr.Until)
		ls.Pos += lr.SizeAdj
	case PushState:
		ls.PushState(lr.PushState)
	case PopState:
		ls.PopState()
	case SetGuestLex:
		if ls.LastName == "" {
			ls.Error(ls.Pos, "SetGuestLex action requires prior Name action -- name is empty", lr)
		} else {
			lx := TheLangLexer.LexerByName(ls.LastName)
			if lx != nil {
				ls.GuestLex = lx
				ls.SaveStack = ls.Stack.Clone()
			}
		}
	case PopGuestLex:
		if ls.SaveStack != nil {
			ls.Stack = ls.SaveStack
			ls.SaveStack = nil
		}
		ls.GuestLex = nil
	}
}

///////////////////////////////////////////////////////////////////////
//  Non-lexing functions

// Find looks for rules in the tree that contain given string in String or Name fields
func (lr *Rule) Find(find string) []*Rule {
	var res []*Rule
	lr.WalkPre(func(k ki.Ki) bool {
		lri := k.(*Rule)
		if strings.Contains(lri.String, find) || strings.Contains(lri.Nm, find) {
			res = append(res, lri)
		}
		return true
	})
	return res
}

// WriteGrammar outputs the lexer rules as a formatted grammar in a BNF-like format
// it is called recursively
func (lr *Rule) WriteGrammar(writer io.Writer, depth int) {
	if ki.IsRoot(lr) {
		for _, k := range lr.Kids {
			lri := k.(*Rule)
			lri.WriteGrammar(writer, depth)
		}
	} else {
		ind := indent.Tabs(depth)
		gpstr := ""
		if lr.HasChildren() {
			gpstr = " {"
		}
		offstr := ""
		if lr.Pos != AnyPos {
			offstr += fmt.Sprintf("@%v:", lr.Pos)
		}
		if lr.Offset > 0 {
			offstr += fmt.Sprintf("+%d:", lr.Offset)
		}
		actstr := ""
		if len(lr.Acts) > 0 {
			actstr = "\t do: "
			for _, ac := range lr.Acts {
				actstr += ac.String()
				if ac == PushState {
					actstr += ": " + lr.PushState
				} else if ac == ReadUntil {
					actstr += ": \"" + lr.Until + "\""
				}
				actstr += "; "
			}
		}
		if lr.Desc != "" {
			fmt.Fprintf(writer, "%v// %v %v \n", ind, lr.Nm, lr.Desc)
		}
		if (lr.Match >= Letter && lr.Match <= WhiteSpace) || lr.Match == AnyRune {
			fmt.Fprintf(writer, "%v%v:\t\t %v\t\t if %v%v%v%v\n", ind, lr.Nm, lr.Token, offstr, lr.Match, actstr, gpstr)
		} else {
			fmt.Fprintf(writer, "%v%v:\t\t %v\t\t if %v%v == \"%v\"%v%v\n", ind, lr.Nm, lr.Token, offstr, lr.Match, lr.String, actstr, gpstr)
		}
		if lr.HasChildren() {
			w := tabwriter.NewWriter(writer, 4, 4, 2, ' ', 0)
			for _, k := range lr.Kids {
				lri := k.(*Rule)
				lri.WriteGrammar(w, depth+1)
			}
			w.Flush()
			fmt.Fprintf(writer, "%v}\n", ind)
		}
	}
}
