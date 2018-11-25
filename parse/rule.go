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

// SetRuleMap is called on the top-level Rule and initializes the RuleMap
func (pr *Rule) SetRuleMap() {
	RuleMap = map[string]*Rule{}
	pr.FuncDownMeFirst(0, pr.This(), func(k ki.Ki, level int, d interface{}) bool {
		pri := k.Embed(KiT_Rule).(*Rule)
		RuleMap[pri.Nm] = pri
		return true
	})
}

// CompileAll is called on the top-level Rule to compile all nodes
// it calls SetRuleMap first
// returns true if everything is ok, false if there were compile errors
func (pr *Rule) CompileAll(ps *State) bool {
	pr.SetRuleMap()
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
						ps.Error(lex.PosZero, fmt.Sprintf("Parser Compile: rule %v: %v", pr.Nm, err.Error()))
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
			rp, ok := RuleMap[rn]
			if !ok {
				ps.Error(lex.PosZero, fmt.Sprintf("Parser Compile: rule %v: refers to rule %v not found", pr.Nm, rn))
				valid = false
			} else {
				re.Rule = rp
			}
		}
	}
	if !gotTok {
		ps.Error(lex.PosZero, fmt.Sprintf("Parser Compile: rule %v: first token not found -- all rules must have at least one token in single quotes -- use token.Tokens names", pr.Nm))
		valid = false
	}
	return valid
}

// Validate checks for any errors in the rules and issues warnings,
// returns true if valid (no err) and false if invalid (errs)
func (pr *Rule) Validate(ps *State) bool {
	valid := true
	if len(pr.Rules) == 0 && !pr.HasChildren() {
		ps.Error(lex.PosZero, fmt.Sprintf("Parser Validate: rule %v: has no rules and no children", pr.Nm))
		valid = false
	}
	if len(pr.Rules) > 0 && pr.HasChildren() {
		ps.Error(lex.PosZero, fmt.Sprintf("Parser Validate: rule %v: has both rules and children -- should be either-or", pr.Nm))
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
func (pr *Rule) Parse(ps *State, par *Rule, ast *Ast) *Rule {
	if pr.IsRoot() {
		kpr := pr.Kids[0].Embed(KiT_Rule).(*Rule) // first rule is special set of valid top-level matches
		if ps.Ast.HasChildren() {
			ast = ps.Ast.KnownChild(0).(*Ast)
		}
		return kpr.Parse(ps, par, ast)
	}

	nr := len(pr.Rules)
	if nr > 0 {
		return pr.ParseRules(ps, par, ast)
	}

	// now we iterate over our kids
	for _, kpri := range pr.Kids {
		kpr := kpri.Embed(KiT_Rule).(*Rule)
		if mrule := kpr.Parse(ps, pr, ast); mrule != nil {
			return mrule
		}
	}
	return nil
}

// ParseRules parses rules and returns this rule if it matches, nil if not
func (pr *Rule) ParseRules(ps *State, par *Rule, ast *Ast) *Rule {
	nr := len(pr.Rules)
	er := pr.Rules[nr-1]
	reg := lex.Reg{}
	reg.St = ps.Pos
	gotEos := false
	if er.IsToken() && er.Token == token.EOS && len(ps.EosPos) > 0 {
		if ps.EosIdx < len(ps.EosPos) {
			reg.Ed = ps.EosPos[ps.EosIdx]
			gotEos = true
		}
	}
	if !gotEos {
		ok := true
		reg, ok = ps.CurReg()
		if !ok {
			ps.Error(reg.St, fmt.Sprintf("Parser rule %v: no current region -- parent must push a region on stack", pr.Nm))
			return nil
		}
	}

	mp, got := ps.FindToken(pr.KeyTok, pr.Keyword, reg)
	if !got {
		return nil
	}

	// we match, add Ast node, do sub-nodes
	parAst, ourAst := ps.AddAst(ast, par, pr.Name(), reg)
	fmt.Printf("matched: %v new ast: %v par ast %v\n", pr.Name(), ourAst.PathUnique(), parAst.PathUnique())

	ok := false
	creg := reg
	lastPos := ps.Pos
	for i := 0; i < nr; i++ {
		if i < pr.KeyTokIdx {
			creg.Ed = mp // only look before token
		} else if i == pr.KeyTokIdx {
			creg.St, ok = ps.Src.NextTokenPos(mp)
			if !ok {
				if nr > i+1 {
					ps.Error(mp, fmt.Sprintf("Parser rule %v: end-of-tokens with more to match", pr.Nm))
					break
				}
			}
			creg.Ed = reg.Ed // full scope
			ps.Pos = creg.St
			lastPos = ps.Pos
			continue
		}
		rr := pr.Rules[i]
		if rr.IsToken() {
			if i == nr-1 && rr.Token == token.EOS {
				break
			}
			tp, got := ps.FindToken(rr.Token, rr.Keyword, creg)
			if !got {
				ps.Error(creg.St, fmt.Sprintf("Parser rule %v: required token %v %v not found", pr.Nm, rr.Token, rr.Keyword))
			} else {
				creg.St, ok = ps.Src.NextTokenPos(tp)
				if !ok {
					if nr > i+1 {
						ps.Error(mp, fmt.Sprintf("Parser rule %v: end-of-tokens with more to match", pr.Nm))
						break
					}
				}
				ps.Pos = creg.St
				lastPos = ps.Pos
			}
		} else {
			ps.PushReg(creg)
			if rr.Rule == pr { // recursive
				nreg := ourAst.TokReg
				nreg.Ed = lastPos
				ourAst.SetTokReg(nreg, ps.Src)
			}
			subm := rr.Rule.Parse(ps, pr, ourAst)
			ps.PopReg()
			if subm == nil {
				ps.Error(creg.St, fmt.Sprintf("Parser rule %v: required sub-rule %v not matched", pr.Nm, rr.Rule.Name()))
			} else {
				nkids := len(ourAst.Kids)
				if nkids > 0 { // should..
					lstk := ourAst.Kids[nkids-1].(*Ast)
					creg.St, ok = ps.Src.NextTokenPos(lstk.TokReg.Ed)
					if !ok {
						if nr > i+1 {
							ps.Error(mp, fmt.Sprintf("Parser rule %v: end-of-tokens with more to match", pr.Nm))
							break
						}
					}
					ps.Pos = creg.St
					lastPos = ps.Pos
				}
			}
		}
	}

	if gotEos {
		ps.EosIdx++
		ps.Pos, _ = ps.Src.NextTokenPos(reg.Ed)
	}
	return pr
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
