// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package parse does the parsing stage after lexing
package parse

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/goki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/token"
)

// Parser is the interface type for parsers -- likely not necessary except is essential
// for defining the BaseIface for gui in making new nodes
type Parser interface {
	ki.Ki

	// Compile compiles string rules into their runnable elements
	Compile(ps *State) bool

	// Validate checks for any errors in the rules and issues warnings,
	// returns true if valid (no err) and false if invalid (errs)
	Validate(ps *State) bool

	// Parse tries to apply rule to given input state, returns rule that matched or nil
	// par is the parent rule that we're being called from
	// ast is the current ast node that we add to
	Parse(ps *State, par *Rule, ast *Ast) *Rule

	// AsParseRule returns object as a parse.Rule
	AsParseRule() *Rule
}

// RuleEl is an element of a parsing rule -- either a pointer to another rule or a token
type RuleEl struct {
	Rule    *Rule        `desc:"rule -- nil if token"`
	Token   token.Tokens `desc:"token, None if rule"`
	Keyword string       `desc:"if keyword, this it"`
	Opt     bool         `desc:"this rule is optional -- will absorb tokens if they exist -- indicated with ? prefix"`
}

func (re RuleEl) IsRule() bool {
	return re.Rule != nil
}

func (re RuleEl) IsToken() bool {
	return re.Rule == nil
}

// RuleList is a list (slice) of rule elements
type RuleList []RuleEl

// RuleMap is a map of all the rule names, for quick lookup
var RuleMap map[string]*Rule

///////////////////////////////////////////////////////////////////////
//  Rule

// parse.Rule operates on the lexically-tokenized input, not the raw source.
//
// The overall strategy is very pragmatic and based on the current known form of
// most languages, which are organized around a sequence of statements having
// a clear scoping defined by the EOS (end of statement), which is identified
// in a first pass through tokenized output using Eoser.
//
// We use a top-down, recursive-descent style parsing, with flexible lookahead
// based on scoping provided by the EOS tags.
//
// Each rule is triggered by a single key token (KeyTok) which is the main distinctive
// token associated with this rule -- if this token is found within the well-defined scope
// (from EOS or parent matching), then the rule matches and sub-rules are then matched.
//
// See Rule description for how to flag the key token if there are multiple
//
// There are two different styles of rules: parents with multiple children
// and the children that specify various alternative forms of the parent category.
// Precedence is encoded directly in the ordering of the children.
type Rule struct {
	ki.Node
	Desc      string       `desc:"description / comments about this rule"`
	Rule      string       `desc:"the rule as a space-separated list of rule names and token(s) -- use single quotes around 'tokens' (using token.Tokens names) -- each rule must have at least one token, and if there are multiple, and the first one should NOT be the key matching token, flag that one with a * -- for keywords use 'key:keyword'"`
	Ast       AstActs      `desc:"what action should be take for this node when it matches"`
	Rules     RuleList     `json:"-" xml:"-" desc:"rule elements compiled from Rule string"`
	KeyTok    token.Tokens `json:"-" xml:"-" desc:"the key token value that this rule matches -- all rules must have one"`
	KeyTokIdx int          `json:"-" xml:"-" desc:"index in rules for the key token"`
	Keyword   string       `json:"-" xml:"-" desc:"if the token is Keyword, this is the specific token we match"`
	PushState string       `desc:"the state to push if our action is PushState -- note that State matching is on String, not this value"`
}

var KiT_Rule = kit.Types.AddType(&Rule{}, RuleProps)

func (pr *Rule) BaseIface() reflect.Type {
	return reflect.TypeOf((*Parser)(nil)).Elem()
}

func (pr *Rule) AsParseRule() *Rule {
	return pr.This().Embed(KiT_Rule).(*Rule)
}

// IsGroup returns true if this node is a group, else it should have rules
func (pr *Rule) IsGroup() bool {
	return pr.HasChildren()
}

// SetRuleMap is called on the top-level Rule and initializes the RuleMap
func (pr *Rule) SetRuleMap(ps *State) {
	RuleMap = map[string]*Rule{}
	pr.FuncDownMeFirst(0, pr.This(), func(k ki.Ki, level int, d interface{}) bool {
		pri := k.Embed(KiT_Rule).(*Rule)
		if epr, has := RuleMap[pri.Nm]; has {
			ps.Error(lex.PosZero, fmt.Sprintf("Parser Compile: multiple rules with same name: %v and %v", pri.PathUnique(), epr.PathUnique()))
		} else {
			RuleMap[pri.Nm] = pri
		}
		return true
	})
}

// CompileAll is called on the top-level Rule to compile all nodes
// it calls SetRuleMap first
// returns true if everything is ok, false if there were compile errors
func (pr *Rule) CompileAll(ps *State) bool {
	pr.SetRuleMap(ps)
	allok := true
	pr.FuncDownMeFirst(0, pr.This(), func(k ki.Ki, level int, d interface{}) bool {
		pri := k.Embed(KiT_Rule).(*Rule)
		ok := pri.Compile(ps)
		if !ok {
			allok = false
		}
		return true
	})
	return allok
}

// Compile compiles string rules into their runnable elements
// returns true if everything is ok, false if there were compile errors
func (pr *Rule) Compile(ps *State) bool {
	if pr.Rule == "" { // parent
		return true
	}
	valid := true
	rs := strings.Split(pr.Rule, " ")
	pr.Rules = make(RuleList, len(rs))
	gotTok := false
	pr.KeyTok = token.None
	pr.KeyTokIdx = -1
	pr.Keyword = ""
	for i := range rs {
		rn := rs[i]
		re := &pr.Rules[i]
		if rn[0] == '\'' || rn[0] == '*' {
			sz := len(rn)
			st := 1
			if rn[0] == '*' { // flag as key token
				st = 2
			}
			tn := rn[st : sz-1]
			if len(tn) > 4 && tn[:4] == "key:" {
				re.Token = token.Keyword
				re.Keyword = tn[4:]
			} else {
				if pmt, has := token.OpPunctMap[tn]; has {
					re.Token = pmt
				} else {
					err := re.Token.FromString(tn)
					if err != nil {
						ps.Error(lex.PosZero, fmt.Sprintf("Compile: rule %v: %v", pr.Nm, err.Error()))
						valid = false
					}
				}
			}
			if !gotTok || st == 2 {
				pr.KeyTok = re.Token
				pr.Keyword = re.Keyword
				pr.KeyTokIdx = i
				gotTok = true
			}
		} else {
			st := 0
			if rn[0] == '?' {
				st = 1
				re.Opt = true
			}
			rp, ok := RuleMap[rn[st:]]
			if !ok {
				ps.Error(lex.PosZero, fmt.Sprintf("Compile: rule %v: refers to rule %v not found", pr.Nm, rn))
				valid = false
			} else {
				re.Rule = rp
			}
		}
	}
	return valid
}

// Validate checks for any errors in the rules and issues warnings,
// returns true if valid (no err) and false if invalid (errs)
func (pr *Rule) Validate(ps *State) bool {
	valid := true
	if len(pr.Rules) == 0 && !pr.HasChildren() {
		ps.Error(lex.PosZero, fmt.Sprintf("Validate: rule %v: has no rules and no children", pr.Nm))
		valid = false
	}
	if len(pr.Rules) > 0 && pr.HasChildren() {
		ps.Error(lex.PosZero, fmt.Sprintf("Validate: rule %v: has both rules and children -- should be either-or", pr.Nm))
		valid = false
	}
	// now we iterate over our kids
	for _, kpri := range pr.Kids {
		kpr := kpri.Embed(KiT_Rule).(*Rule)
		if !kpr.Validate(ps) {
			valid = false
		}
	}
	return valid
}

// Parse tries to apply rule to given input state, returns rule that matched or nil
// par is the parent rule that we're being called from
// ast is the current ast node that we add to
func (pr *Rule) Parse(ps *State, par *Rule, parAst *Ast) *Rule {
	if pr.IsRoot() {
		kpr := pr.Kids[0].Embed(KiT_Rule).(*Rule) // first rule is special set of valid top-level matches
		if ps.Ast.HasChildren() {
			parAst = ps.Ast.KnownChild(0).(*Ast)
		} else {
			parAst = ps.Ast.AddNewChild(KiT_Ast, kpr.Name()).(*Ast)
		}
		return kpr.Parse(ps, par, parAst)
	}

	nr := len(pr.Rules)
	if nr > 0 {
		return pr.ParseRules(ps, par, parAst)
	}

	// pure group types just iterate over kids
	for _, kpri := range pr.Kids {
		kpr := kpri.Embed(KiT_Rule).(*Rule)
		if mrule := kpr.Parse(ps, pr, parAst); mrule != nil {
			return mrule
		}
	}
	return nil
}

// ParseRules parses rules and returns this rule if it matches, nil if not
func (pr *Rule) ParseRules(ps *State, par *Rule, parAst *Ast) *Rule {
	scope, ok := pr.Scope(ps)
	if !ok {
		return nil
	}
	match, mpos := pr.Match(ps, scope)
	if !match {
		return nil
	}
	if par.Ast != NoAst && par.IsGroup() {
		if parAst.Nm != par.Nm {
			newAst := ps.AddAst(parAst, par.Name(), scope)
			if par.Ast == AnchorAst {
				parAst = newAst
			}
		}
	}
	pr.DoRules(ps, par, parAst, scope, mpos) // returns validity but we don't care..
	return pr
}

// HasEos returns true if our rule ends in EOS token
func (pr *Rule) HasEos(ps *State) bool {
	nr := len(pr.Rules)
	er := pr.Rules[nr-1]
	if er.IsToken() && er.Token == token.EOS && len(ps.EosPos) > ps.EosIdx {
		return true
	}
	return false
}

// Scope finds the potential scope region for looking for tokens -- either from
// EOS position or State ScopeStack pushed from parents.
// returns false if no valid scope found
func (pr *Rule) Scope(ps *State) (lex.Reg, bool) {
	scope := lex.Reg{}
	scope.St = ps.Pos
	ok := false
	if pr.HasEos(ps) {
		scope.Ed = ps.EosPos[ps.EosIdx]
	} else {
		scope, ok = ps.CurScope()
		if !ok {
			ps.Error(scope.St, fmt.Sprintf("rule %v: no current scope -- parent must push a scope on stack", pr.Nm))
			return scope, false
		}
	}

	scope.St, ok = ps.Src.ValidTokenPos(scope.St)
	ps.Pos = scope.St
	if !ok {
		return scope, false
	}
	return scope, true
}

// Match attempts to match the rule, returns true if it matches, and the
// match position
func (pr *Rule) Match(ps *State, scope lex.Reg) (bool, lex.Pos) {
	mpos := lex.Pos{} // match pos

	nr := len(pr.Rules)
	if nr == 0 {
		for _, kpri := range pr.Kids {
			kpr := kpri.Embed(KiT_Rule).(*Rule)
			match, mpos := kpr.Match(ps, scope)
			if match {
				fmt.Printf("\trule %v: matched based on kid: %v at: %v\n", pr.Name(), kpr.Name(), mpos)
				return true, mpos
			}
		}
		return false, mpos
	}

	if pr.KeyTok == token.None { // no tokens, use first one to match
		rr := pr.Rules[0]
		if rr.IsRule() {
			match, mpos := rr.Rule.Match(ps, scope)
			if match {
				fmt.Printf("\trule %v: matched based on first rule: %v at: %v\n", pr.Name(), rr.Rule.Name(), mpos)
			} else {
				fmt.Printf("\trule %v: failed to match based on first rule: %v\n", pr.Name(), rr.Rule.Name())
			}
			return match, mpos
		}
		// else already flagged in compile
		return false, mpos
	}

	if pr.KeyTokIdx == 0 { // key constraint: must be at start
		if !ps.MatchToken(pr.KeyTok, pr.Keyword, scope.St) {
			return false, mpos
		}
		mpos = scope.St
	} else {
		got := false
		mpos, got = ps.FindToken(pr.KeyTok, pr.Keyword, scope)
		if !got {
			return false, mpos
		}
	}
	fmt.Printf("\trule %v: matched based on key token: %v at: %v\n", pr.Name(), pr.KeyTok, mpos)
	return true, mpos
}

// DoRules after we have matched, goes through rest of the rules -- returns false if
// there were any issues encountered
func (pr *Rule) DoRules(ps *State, par *Rule, parAst *Ast, scope lex.Reg, mpos lex.Pos) bool {
	var ourAst *Ast
	if pr.Ast != NoAst {
		ourAst = ps.AddAst(parAst, pr.Name(), scope)
		fmt.Printf("rule: %v doing with new ast: %v par ast %v\n", pr.Name(), ourAst.PathUnique(), parAst.PathUnique())
	} else {
		fmt.Printf("rule: %v no new ast, par ast %v\n", pr.Name(), parAst.PathUnique())
	}

	nr := len(pr.Rules)
	valid := true
	ok := false
	creg := scope
	for i := 0; i < nr; i++ {
		rr := pr.Rules[i]
		creg.St = ps.Pos
		if i < pr.KeyTokIdx {
			creg.Ed = mpos // only look before token
		} else if i == pr.KeyTokIdx {
			ps.Pos, ok = ps.Src.NextTokenPos(mpos)
			if !ok {
				if nr > i+1 {
					ps.Error(mpos, fmt.Sprintf("rule %v: end-of-tokens with more to match", pr.Nm))
					valid = false
					break
				}
			}
			creg.Ed = scope.Ed // full scope
			fmt.Printf("\trule %v: matched token: %v %v at: %v advanced pos to: %v\n", pr.Nm, rr.Token, rr.Keyword, mpos, ps.Pos)
			if ourAst != nil {
				ourAst.SetTokRegEnd(ps.Pos, ps.Src) // update our end to any tokens that match
			}
			continue
		}
		if rr.IsToken() {
			if i == nr-1 && rr.Token == token.EOS {
				break
			}
			tp, got := ps.FindToken(rr.Token, rr.Keyword, creg)
			if !got {
				ps.Error(creg.St, fmt.Sprintf("rule %v: required token %v %v not found", pr.Nm, rr.Token, rr.Keyword))
				valid = false
			} else {
				ps.Pos, ok = ps.Src.NextTokenPos(tp)
				if !ok {
					if nr > i+1 {
						ps.Error(mpos, fmt.Sprintf("rule %v: end-of-tokens with more to match", pr.Nm))
						valid = false
						break
					}
				}
				fmt.Printf("\trule %v: matched token: %v %v at: %v advanced pos to: %v\n", pr.Nm, rr.Token, rr.Keyword, tp, ps.Pos)
				if ourAst != nil {
					ourAst.SetTokRegEnd(ps.Pos, ps.Src) // update our end to any tokens that match
				}
			}
		} else {
			if creg.IsNil() { // no tokens left..
				if rr.Opt {
					continue
				}
				ps.Error(creg.St, fmt.Sprintf("rule %v: non-optional rule element has no tokens", pr.Nm, rr.Rule.Name()))
				valid = false
				continue
			}
			useAst := parAst
			if pr.Ast == AnchorAst {
				useAst = ourAst
			}
			fmt.Printf("\trule %v: trying sub-rule: %v within region: %v\n", pr.Nm, rr.Rule.Name(), creg)
			ps.PushScope(creg)
			subm := rr.Rule.Parse(ps, pr, useAst)
			ps.PopScope()
			if subm == nil {
				if !rr.Opt {
					ps.Error(creg.St, fmt.Sprintf("rule %v: required sub-rule %v not matched", pr.Nm, rr.Rule.Name()))
					valid = false
				}
			}
			if !rr.Opt && ourAst != nil {
				ourAst.SetTokRegEnd(ps.Pos, ps.Src) // update our end to include non-optional elements
			}
		}
	}
	if pr.HasEos(ps) {
		ps.EosIdx++
		ps.Pos, _ = ps.Src.NextTokenPos(scope.Ed)
	}
	return valid
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
