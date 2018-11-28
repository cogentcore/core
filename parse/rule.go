// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package parse does the parsing stage after lexing, using a top-down recursive-descent
// (TDRD) strategy, with a special reverse mode to deal with left-associative binary expressions
// which otherwise end up being right-associative for TDRD parsing.
// Higher-level rules provide scope to lower-level ones, with a special EOS end-of-statement
// scope recognized for
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

// Set GuiActive to true if the gui (piview) is active -- ensures that the
// Ast tree is updated when nodes are swapped in reverse mode, and maybe
// other things
var GuiActive = false

// Set Trace to true to get print statement trace of parse steps
var Trace = true

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
	Rule  *Rule          `desc:"rule -- nil if token"`
	Token token.KeyToken `desc:"token, None if rule"`
	Opt   bool           `desc:"this rule is optional -- will absorb tokens if they exist -- indicated with ? prefix"`
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
	Desc      string         `desc:"description / comments about this rule"`
	Rule      string         `desc:"the rule as a space-separated list of rule names and token(s) -- use single quotes around 'tokens' (using token.Tokens names) -- each rule must have at least one token, and if there are multiple, and the first one should NOT be the key matching token, flag that one with a * -- for keywords use 'key:keyword'"`
	Ast       AstActs        `desc:"what action should be take for this node when it matches"`
	Rules     RuleList       `json:"-" xml:"-" desc:"rule elements compiled from Rule string"`
	KeyTok    token.KeyToken `json:"-" xml:"-" desc:"the key token value that this rule matches -- all rules must have one"`
	KeyTokIdx int            `json:"-" xml:"-" desc:"index in rules for the key token"`
	Reverse   bool           `json:"-" xml:"-" desc:"use a reverse parsing direction for binary operator expressions -- this is needed to produce proper associativity result for mathematical expressions in the recursive descent parser, triggered by a '-' at the start of the rule -- only for rules of form: Expr '+' Expr -- two sub-rules with a token operator in the middle"`
	EndTok    token.KeyToken `json:"-" xml:"-" desc:"ending token -- if rule ends with an EOS, paren, brace or bracket token then we first search for it to establish the scope of our rule -- automatically deals with embedded ones"`
	PushState string         `desc:"the state to push if our action is PushState -- note that State matching is on String, not this value"`
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
	rstr := pr.Rule
	if pr.Rule[0] == '-' {
		rstr = rstr[1:]
		pr.Reverse = true
	} else {
		pr.Reverse = false
	}
	rs := strings.Split(rstr, " ")
	pr.Rules = make(RuleList, len(rs))
	gotTok := false
	pr.KeyTok = token.KeyTokenZero
	pr.KeyTokIdx = -1
	pr.EndTok = token.KeyTokenZero
	nr := len(rs)
	for i := range rs {
		rn := strings.TrimSpace(rs[i])
		if len(rn) == 0 {
			ps.Error(lex.PosZero, fmt.Sprintf("Compile: rule %v: empty string -- make sure there is only one space between rule elements", pr.Nm))
			valid = false
			break
		}
		re := &pr.Rules[i]
		if rn[0] == '\'' || rn[0] == '*' {
			sz := len(rn)
			st := 1
			if rn[0] == '*' { // flag as key token
				st = 2
			}
			tn := rn[st : sz-1]
			if len(tn) > 4 && tn[:4] == "key:" {
				re.Token.Tok = token.Keyword
				re.Token.Key = tn[4:]
			} else {
				if pmt, has := token.OpPunctMap[tn]; has {
					re.Token.Tok = pmt
				} else {
					err := re.Token.Tok.FromString(tn)
					if err != nil {
						ps.Error(lex.PosZero, fmt.Sprintf("Compile: rule %v: %v", pr.Nm, err.Error()))
						valid = false
					}
				}
			}
			if !gotTok || st == 2 {
				pr.KeyTok = re.Token
				pr.KeyTokIdx = i
				gotTok = true
			}
			if i == nr-1 && pr.KeyTokIdx != i {
				if re.Token.Tok.SubCat() == token.PunctGp || re.Token.Tok == token.EOS {
					pr.EndTok = re.Token // scoping end token
				}
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
	if pr.Reverse {
		pr.Ast = AnchorAst // must be
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
	if pr.Reverse {
		if len(pr.Rules) != 3 {
			ps.Error(lex.PosZero, fmt.Sprintf("Validate: rule %v: is a Reverse (-) rule: must have 3 children -- for binary operator expressions only", pr.Nm))
			valid = false
		} else {
			if pr.KeyTokIdx != 1 {
				ps.Error(lex.PosZero, fmt.Sprintf("Validate: rule %v: is a Reverse (-) rule: must have a token to be recognized in the middle of two rules -- for binary operator expressions only", pr.Nm))
			}
		}
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
	mpos := lex.Pos{}
	match := false
	match, mpos, scope = pr.Match(ps, scope)
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

// HasEos returns true if our rule ends in EOS token and there are Eos tokens left in input
func (pr *Rule) HasEos(ps *State) bool {
	if pr.EndTok.Tok == token.EOS && len(ps.EosPos) > ps.EosIdx {
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
// match position, along with any update to the scope
func (pr *Rule) Match(ps *State, scope lex.Reg) (bool, lex.Pos, lex.Reg) {
	mpos := lex.Pos{} // match pos
	nr := len(pr.Rules)
	if nr == 0 {
		for _, kpri := range pr.Kids {
			kpr := kpri.Embed(KiT_Rule).(*Rule)
			match, mpos, nscp := kpr.Match(ps, scope)
			if match {
				if Trace {
					fmt.Printf("\trule %v: matched based on kid: %v at: %v\n", pr.Name(), kpr.Name(), mpos)
				}
				return true, mpos, nscp
			}
		}
		return false, mpos, scope
	}

	if pr.KeyTok.Tok == token.None { // no tokens, use first one to match
		rr := pr.Rules[0]
		if pr.Reverse {
			rr = pr.Rules[nr-1]
		}
		if rr.IsRule() {
			match, mpos, nscp := rr.Rule.Match(ps, scope)
			if Trace {
				if match {
					fmt.Printf("\trule %v: matched based on first rule: %v at: %v\n", pr.Name(), rr.Rule.Name(), mpos)
				} else {
					fmt.Printf("\trule %v: failed to match based on first rule: %v\n", pr.Name(), rr.Rule.Name())
				}
			}
			return match, mpos, nscp
		}
		// else already flagged in compile
		return false, mpos, scope
	}

	if pr.KeyTokIdx == 0 { // key constraint: must be at start
		if !ps.MatchToken(pr.KeyTok, scope.St) {
			return false, mpos, scope
		}
		mpos = scope.St
	} else {
		got := false
		if pr.Reverse {
			mpos, got = ps.FindTokenReverse(pr.KeyTok, scope)
		} else {
			mpos, got = ps.FindToken(pr.KeyTok, scope)
		}
		if !got {
			return false, mpos, scope
		}
	}

	if pr.EndTok.Tok != token.None && pr.EndTok.Tok != token.EOS {
		// end MUST match exactly with the scope we have -- that is our matching criterion
		trgep, ok := ps.Src.PrevTokenPos(scope.Ed)
		ep, ok := ps.Src.FindPunctGpMatch(pr.EndTok.Tok.PunctGpMatch(), lex.Reg{St: mpos, Ed: scope.Ed})
		if !ok || trgep != ep {
			return false, mpos, scope
		}
		if Trace {
			fmt.Printf("\trule %v: matched ending token %v at %v\n", pr.Name(), pr.EndTok, ep)
		}
		scope.Ed = ep // regular rule does not have logic to restrict end to before end tok..
	}

	if Trace {
		fmt.Printf("\trule %v: matched based on key token: %v at: %v\n", pr.Name(), pr.KeyTok, mpos)
	}
	return true, mpos, scope
}

// MatchRevGpToks is match for reverse group token
func (pr *Rule) MatchRevGpToks(ps *State, scope lex.Reg) (bool, lex.Pos, lex.Reg) {
	// match key token at end of scope..
	mpos, ok := ps.Src.PrevTokenPos(scope.Ed)
	if !ok || !ps.MatchToken(pr.KeyTok, mpos) {
		return false, mpos, scope
	}
	scope.Ed = mpos
	ep, ok := ps.Src.FindPunctGpMatch(pr.EndTok.Tok.PunctGpMatch(), lex.Reg{St: scope.St, Ed: mpos})
	if !ok {
		ps.Error(scope.St, fmt.Sprintf("rule %v: failed to find ending token: %v", pr.Nm, pr.EndTok))
	} else {
		scope.St, _ = ps.Src.NextTokenPos(ep)
		if Trace {
			fmt.Printf("\trule %v: found ending token %v at %v\n", pr.Name(), pr.EndTok, ep)
		}
	}
	if Trace {
		fmt.Printf("\trule %v: matched based on key token: %v at: %v\n", pr.Name(), pr.KeyTok, mpos)
	}
	return true, mpos, scope
}

// DoRules after we have matched, goes through rest of the rules -- returns false if
// there were any issues encountered
func (pr *Rule) DoRules(ps *State, par *Rule, parAst *Ast, scope lex.Reg, mpos lex.Pos) bool {
	var ourAst *Ast
	if pr.Ast != NoAst {
		ourAst = ps.AddAst(parAst, pr.Name(), scope)
		if Trace {
			fmt.Printf("rule: %v doing with new ast: %v par ast %v\n", pr.Name(), ourAst.PathUnique(), parAst.PathUnique())
		}
	} else {
		if Trace {
			fmt.Printf("rule: %v no new ast, par ast %v\n", pr.Name(), parAst.PathUnique())
		}
	}

	if pr.Reverse {
		return pr.DoRulesRevBinExp(ps, par, parAst, scope, mpos, ourAst)
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
			if Trace {
				fmt.Printf("\trule %v: matched key token: %v %v at: %v advanced pos to: %v\n", pr.Nm, rr.Token, mpos, ps.Pos)
			}
			if ourAst != nil {
				ourAst.SetTokRegEnd(ps.Pos, ps.Src) // update our end to any tokens that match
			}
			continue
		}
		if rr.IsToken() {
			if i == nr-1 && pr.EndTok.Tok != token.None { // already matched
				if pr.HasEos(ps) {
					ps.EosIdx++
				}
				ps.Pos, _ = ps.Src.NextTokenPos(scope.Ed) // consume end  -- todo for EOS not so clear
				if ourAst != nil {
					ourAst.SetTokRegEnd(ps.Pos, ps.Src) // update our end to any tokens that match
				}
				break
			}
			tp, got := ps.FindToken(rr.Token, creg)
			if !got {
				ps.Error(creg.St, fmt.Sprintf("rule %v: required token %v %v not found", pr.Nm, rr.Token))
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
				if Trace {
					fmt.Printf("\trule %v: matched token: %v %v at: %v advanced pos to: %v\n", pr.Nm, rr.Token, tp, ps.Pos)
				}
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
			if Trace {
				fmt.Printf("\trule %v: trying sub-rule: %v within region: %v\n", pr.Nm, rr.Rule.Name(), creg)
			}
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
	return valid
}

// DoRulesRevBinExp reverse version of do rules for binary
// expression rule with one key token in the middle -- we just pay attention
// to scoping rest of sub-rules relative to that, and don't otherwise adjust scope
// or position.  In particular all the position updating taking place in sup-rules
// is then just ignored and we set the position to the end position matched by the
// "last" rule (which was the first processed)
func (pr *Rule) DoRulesRevBinExp(ps *State, par *Rule, parAst *Ast, scope lex.Reg, mpos lex.Pos, ourAst *Ast) bool {
	nr := len(pr.Rules)
	valid := true
	creg := scope

	aftMpos, ok := ps.Src.NextTokenPos(mpos)
	if !ok {
		ps.Error(mpos, fmt.Sprintf("rule %v: end-of-tokens with more to match", pr.Nm))
		return false
	}

	epos := scope.Ed
	for i := nr - 1; i >= 0; i-- {
		rr := pr.Rules[i]
		if i > pr.KeyTokIdx {
			creg.St = aftMpos // end expr is in region from key token to end of scope
			ps.Pos = creg.St  // only works for a single rule after key token -- sub-rules not necc reverse
			creg.Ed = scope.Ed
		} else if i == pr.KeyTokIdx {
			if Trace {
				fmt.Printf("\trule %v: matched key token: %v %v at: %v", pr.Nm, rr.Token, mpos)
			}
			continue
		} else { // start
			creg.St = scope.St
			ps.Pos = creg.St
			creg.Ed = mpos
		}
		if rr.IsRule() { // non-key tokens ignored
			if creg.IsNil() { // no tokens left..
				ps.Error(creg.St, fmt.Sprintf("rule %v: non-optional rule element has no tokens", pr.Nm, rr.Rule.Name()))
				valid = false
				continue
			}
			useAst := parAst
			if pr.Ast == AnchorAst {
				useAst = ourAst
			}
			if Trace {
				fmt.Printf("\trule %v: trying sub-rule: %v within region: %v\n", pr.Nm, rr.Rule.Name(), creg)
			}
			ps.PushScope(creg)
			subm := rr.Rule.Parse(ps, pr, useAst)
			ps.PopScope()
			if subm == nil {
				if !rr.Opt {
					ps.Error(creg.St, fmt.Sprintf("rule %v: required sub-rule %v not matched", pr.Nm, rr.Rule.Name()))
					valid = false
				}
			}
			// if i == nr-1 {
			// 	// epos = ps.Pos // this is where the "last" sub-rule got to -- we end here
			// 	// if ourAst != nil {
			// 	// 	ourAst.SetTokRegEnd(ps.Pos, ps.Src) // update our end
			// 	// }
			// }
		}
	}
	// our AST is now backwards -- need to swap them
	if len(ourAst.Kids) == 2 {
		ourAst.SwapChildren(0, 1)
		if GuiActive {
			// we have a very strange situation here: the tree view of the Ast will typically
			// have two children, named identically (e.g., Expr, Expr) and it will not update
			// after our swap.  If we could use UniqNames then it would be ok, but that doesn't
			// work for treeview names.. really need an option that supports uniqname AND reg names
			// https://github.com/goki/ki/issues/2
			ourAst.AddNewChild(KiT_Ast, "Dummy")
			ourAst.DeleteChildAtIndex(2, true)
		}
	}

	ps.Pos = epos
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
