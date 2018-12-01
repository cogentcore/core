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
	"strconv"
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

// parse.Rule operates on the lexically-tokenized input, not the raw source.
//
// The overall strategy is pragmatically based on the current known form of
// most languages, which are organized around a sequence of statements having
// a clear scoping defined by the EOS (end of statement), which is identified
// in a first pass through tokenized output in PassTwo.
//
// We use a top-down, recursive-descent style parsing, with flexible lookahead
// based on scoping provided by the EOS tags.  Each rule progressively scopes
// down the space, using token matches etc to bracket the space for flexible
// elements.
//
// There are two different rule types:
// 1. Parents with multiple children (i.e. Groups), which are all the different
// variations for satisfying that rule, with precedence encoded directly in the
// ordering of the children.  These have empty "Rule" string and Rules.
// 2. Explicit rules specified in the Rule string.

// The first step is matching which searches in order for matches within the
// children of parent nodes, and for explicit rule nodes, it looks first
// through all the explicit tokens in the rule.  If there are no explicit tokens
// then matching defers to ONLY the first node listed by default -- you can
// add a @ prefix to indicate a rule that is also essential to match.
//
// After a rule matches, it then proceeds through the rules narrowing the scope
// and calling the sub-nodes..
//
type Rule struct {
	ki.Node
	Desc      string   `desc:"description / comments about this rule"`
	Rule      string   `desc:"the rule as a space-separated list of rule names and token(s) -- use single quotes around 'tokens' (using token.Tokens names or symbols). For keywords use 'key:keyword'.  All tokens are matched at the same nesting depth as the start of the scope of this rule, unless they have a +D relative depth value differential before the token.  Use @ prefix for a sub-rule to require that rule to match -- by default explicit tokens are used if available, and then only the first sub-rule failing that."`
	Ast       AstActs  `desc:"what action should be take for this node when it matches"`
	OptTokMap bool     `desc:"for group-level rules having lots of children and lots of recursiveness, and also of high-frequency, when we first encounter such a rule, make a map of all the tokens in the entire scope, and use that for a first-pass rejection on matching tokens"`
	Rules     RuleList `json:"-" xml:"-" desc:"rule elements compiled from Rule string"`
	Reverse   bool     `inactive:"+" json:"-" xml:"-" desc:"use a reverse parsing direction for binary operator expressions -- this is needed to produce proper associativity result for mathematical expressions in the recursive descent parser, triggered by a '-' at the start of the rule -- only for rules of form: Expr '+' Expr -- two sub-rules with a token operator in the middle"`
}

var KiT_Rule = kit.Types.AddType(&Rule{}, RuleProps)

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
	Rule  *Rule          `desc:"sub-rule for this position -- nil if token"`
	Tok   token.KeyToken `desc:"token, None if rule"`
	Match bool           `desc:"if true, this rule must match for rule to fire -- by default only tokens and, failing that, the first sub-rule is used for matching -- use @ to require a match"`
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

// Last returns the last rule -- only used in cases where there are rules
func (rl RuleList) Last() *RuleEl {
	return &rl[len(rl)-1]
}

// RuleMap is a map of all the rule names, for quick lookup
var RuleMap map[string]*Rule

// Matches encodes the positions of each match, Err for no match
type Matches []lex.Pos

///////////////////////////////////////////////////////////////////////
//  Rule

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
// it calls SetRuleMap first.
// Returns true if everything is ok, false if there were compile errors
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

// Compile compiles string rules into their runnable elements.
// Returns true if everything is ok, false if there were compile errors.
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
	nr := len(rs)
	pr.Rules = make(RuleList, nr)
	nmatch := 0
	ntok := 0
	for i := range rs {
		rn := strings.TrimSpace(rs[i])
		if len(rn) == 0 {
			ps.Error(lex.PosZero, fmt.Sprintf("Compile: rule %v: empty string -- make sure there is only one space between rule elements", pr.Nm))
			valid = false
			break
		}
		re := &pr.Rules[i]
		tokst := strings.Index(rn, "'")
		if tokst >= 0 {
			if rn[0] == '?' {
				re.Opt = true
			} else {
				re.Match = true // all tokens match by default
				nmatch++
				ntok++
			}
			sz := len(rn)
			if rn[0] == '+' {
				td, _ := strconv.ParseInt(rn[1:tokst], 10, 64)
				re.Tok.Depth = int(td)
			}
			tn := rn[tokst+1 : sz-1]
			if len(tn) > 4 && tn[:4] == "key:" {
				re.Tok.Tok = token.Keyword
				re.Tok.Key = tn[4:]
			} else {
				if pmt, has := token.OpPunctMap[tn]; has {
					re.Tok.Tok = pmt
				} else {
					err := re.Tok.Tok.FromString(tn)
					if err != nil {
						ps.Error(lex.PosZero, fmt.Sprintf("Compile: rule %v: %v", pr.Nm, err.Error()))
						valid = false
					}
				}
			}
			// if i == nr-1 && pr.KeyTokIdx != i {
			// 	if re.Tok.Tok.SubCat() == token.PunctGp || re.Tok.Tok == token.EOS {
			// 		pr.EndTok = re.Tok // scoping end token
			// 		continue
			// 	}
		} else {
			st := 0
			if rn[0] == '?' {
				st = 1
				re.Opt = true
			} else if rn[0] == '@' {
				st = 1
				re.Match = true
				nmatch++
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
	if ntok == 0 {
		pr.Rules[0].Match = true
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
			if !pr.Rules[1].IsToken() {
				ps.Error(lex.PosZero, fmt.Sprintf("Validate: rule %v: is a Reverse (-) rule: must have a token to be recognized in the middle of two rules -- for binary operator expressions only", pr.Nm))
			}
		}
	} else {
		if len(pr.Rules) == 3 && pr.Rules[1].IsToken() && pr.Rules[0].IsRule() && pr.Rules[2].IsRule() {
			ktok := pr.Rules[1].Tok.Tok
			if ktok.Cat() == token.Operator && ktok.SubCat() != token.OpList {
				ps.Error(lex.PosZero, fmt.Sprintf("Validate: rule %v: is a binary operator expression -- should use reverse - operation order to produce correct associativity", pr.Nm))
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

// StartParse is called on the root of the parse rule tree to start the parsing process
func (pr *Rule) StartParse(ps *State) *Rule {
	kpr := pr.Kids[0].Embed(KiT_Rule).(*Rule) // first rule is special set of valid top-level matches
	var parAst *Ast
	if ps.Ast.HasChildren() {
		parAst = ps.Ast.KnownChild(0).(*Ast)
	} else {
		parAst = ps.Ast.AddNewChild(KiT_Ast, kpr.Name()).(*Ast)
	}
	return kpr.Parse(ps, pr, parAst, lex.RegZero, nil)
}

// Parse tries to apply rule to given input state, returns rule that matched or nil
// par is the parent rule that we're being called from.
// parAst is the current ast node that we add to.
// scope is the region to search within, defined by parent or EOS if we have a terminal
// one
func (pr *Rule) Parse(ps *State, par *Rule, parAst *Ast, scope lex.Reg, optMap lex.TokenMap) *Rule {
	nr := len(pr.Rules)
	if nr > 0 {
		return pr.ParseRules(ps, par, parAst, scope, optMap)
	}

	if optMap == nil && pr.OptTokMap {
		optMap = ps.Src.TokenMapReg(scope)
		Trace.Out(ps, pr, Run, scope.St, scope, parAst, fmt.Sprintf("made optmap of size: %d", len(optMap)))
	}

	// pure group types just iterate over kids
	for _, kpri := range pr.Kids {
		kpr := kpri.Embed(KiT_Rule).(*Rule)
		if mrule := kpr.Parse(ps, pr, parAst, scope, optMap); mrule != nil {
			return mrule
		}
	}
	return nil
}

// ParseRules parses rules and returns this rule if it matches, nil if not
func (pr *Rule) ParseRules(ps *State, par *Rule, parAst *Ast, scope lex.Reg, optMap lex.TokenMap) *Rule {
	scope, ok := pr.Scope(ps, parAst, scope)
	if !ok {
		return nil
	}
	match, nscope, mpos := pr.Match(ps, parAst, scope, optMap)
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
	pr.DoRules(ps, par, parAst, nscope, mpos, optMap) // returns validity but we don't care once matched..
	return pr
}

// Scope finds the potential scope region for looking for tokens -- either from
// EOS position or State ScopeStack pushed from parents.
// Returns new scope and false if no valid scope found.
func (pr *Rule) Scope(ps *State, parAst *Ast, scope lex.Reg) (lex.Reg, bool) {
	ok := false
	lr := pr.Rules.Last()
	if scope == lex.RegZero {
		scope.St = ps.Pos
	}
	scope.St, ok = ps.Src.ValidTokenPos(scope.St) // should have been done, but just in case
	if !ok {
		return scope, false
	}
	nscope := scope
	if lr.Tok.Tok == token.EOS {
		stlx := ps.Src.LexAt(scope.St)
		ep, eosIdx := ps.FindEos(scope.St, stlx.Depth+lr.Tok.Depth)
		if eosIdx < 0 {
			ps.Error(scope.St, fmt.Sprintf("rule %v: could not find EOS at target nesting depth -- parens / bracket / brace mismatch?", pr.Nm))
			return nscope, false
		}
		nscope.Ed = ep
	} else { // note: could conceivably have mode whre non-EOS tokens are used, but very expensive..
		if scope.IsNil() {
			ps.Error(scope.St, fmt.Sprintf("rule %v: scope is empty and no EOS in rule -- invalid rules -- must all start with EOS", pr.Nm))
			return nscope, false
		}
	}
	return nscope, true
}

// Match attempts to match the rule, returns true if it matches, and the
// match positions, along with any update to the scope
func (pr *Rule) Match(ps *State, parAst *Ast, scope lex.Reg, optMap lex.TokenMap) (bool, lex.Reg, Matches) {
	nr := len(pr.Rules)
	if nr == 0 { // Group
		for _, kpri := range pr.Kids {
			kpr := kpri.Embed(KiT_Rule).(*Rule)
			match, nscope, mpos := kpr.Match(ps, parAst, scope, optMap)
			if match {
				Trace.Out(ps, pr, Match, scope.St, scope, parAst, fmt.Sprintf("group child: %v", kpr.Name()))
				return true, nscope, mpos
			}
		}
		return false, scope, nil
	}

	mpos := make(Matches, len(pr.Rules))

	ok := false
	creg := scope
	lmnpos := lex.PosZero // last match next-pos
	pos := lex.PosZero
	for i := 0; i < nr; i++ {
		rr := pr.Rules[i]
		if pr.Reverse {
			rr = pr.Rules[nr-1-i]
		}
		if !rr.Match {
			continue
		}
		if lmnpos != lex.PosZero {
			creg.St = lmnpos
		}
		if i == nr-1 && rr.Tok.Tok == token.EOS {
			mpos[i] = scope.Ed
			break
		}
		if creg.IsNil() {
			return false, scope, nil
		}
		if rr.IsToken() {
			slx := ps.Src.LexAt(creg.St)
			kt := rr.Tok
			kt.Depth = slx.Depth
			if i == 0 { // start token must be right here
				if !ps.MatchToken(kt, scope.St) {
					Trace.Out(ps, pr, NoMatch, creg.St, creg, parAst, fmt.Sprintf("%v token: %v, was: %v", i, kt.String(), slx.String()))
					return false, scope, nil
				}
				Trace.Out(ps, pr, Match, creg.St, creg, parAst, fmt.Sprintf("%v token: %v", i, kt.String()))
				mpos[i] = creg.St
			} else { // look for token
				if optMap != nil && !optMap.Has(kt.Tok) { // not even a possibility
					return false, scope, nil
				}
				if pr.Reverse || i == nr-1 { // this is key to look in reverse for last item!
					pos, ok = ps.FindTokenReverse(kt, creg)
				} else {
					pos, ok = ps.FindToken(kt, creg)
				}
				if !ok {
					Trace.Out(ps, pr, NoMatch, creg.St, creg, parAst, fmt.Sprintf("%v token: %v", i, kt.String()))
					return false, scope, nil
				}
				Trace.Out(ps, pr, Match, pos, creg, parAst, fmt.Sprintf("%v token: %v", i, kt))
				mpos[i] = pos
				// note: used to use this code to enforce match at end of scope for last token:
				// if ever run into an issue, this is here..  todo: deleteme at some point!
				// et.Depth = slx.Depth
				// ep, _ := ps.Src.PrevTokenPos(scope.Ed)
				// elx := ps.Src.LexAt(ep)
				// if slx.Depth != elx.Depth || elx.Tok != pr.EndTok.Tok {
				// 	Trace.Out(ps, pr, NoMatch, ep, scope, parAst, fmt.Sprintf("end token: %v, was: %v", et.String(), elx.String()))
				// 	return false, mpos, scope
				// }
				// Trace.Out(ps, pr, Match, ep, scope, parAst, fmt.Sprintf("end token: %v", et.String()))
				// scope.Ed = ep // regular rule does not have logic to restrict end to before end tok..
			}
			lmnpos, ok = ps.Src.NextTokenPos(mpos[i])
			if !ok {
				return false, scope, nil
			}
		} else { // Sub-Rule
			match, _, smpos := rr.Rule.Match(ps, parAst, creg, optMap)
			if !match {
				Trace.Out(ps, pr, NoMatch, creg.St, creg, parAst, fmt.Sprintf("%v rule: %v", i, rr.Rule.Name()))
				return false, scope, nil
			}
			// look through smpos for last valid position -- use that as last match pos
			lpos := lex.PosZero
			fpos := lex.PosZero
			for _, mp := range smpos {
				if mp != lex.PosZero {
					if fpos == lex.PosZero {
						fpos = mp
					}
					lpos = mp
				}
			}
			mpos[i] = fpos // can use this during rule exec to start at right place
			Trace.Out(ps, pr, Match, fpos, creg, parAst, fmt.Sprintf("%v rule: %v", i, rr.Rule.Name()))
			lmnpos, ok = ps.Src.NextTokenPos(lpos)
			if !ok {
				return false, scope, nil
			}
		}
	}
	return true, scope, mpos
}

// DoRules after we have matched, goes through rest of the rules -- returns false if
// there were any issues encountered
func (pr *Rule) DoRules(ps *State, par *Rule, parAst *Ast, scope lex.Reg, mpos Matches, optMap lex.TokenMap) bool {
	trcAst := parAst
	var ourAst *Ast
	if pr.Ast != NoAst {
		ourAst = ps.AddAst(parAst, pr.Name(), scope)
		trcAst = ourAst
		Trace.Out(ps, pr, Run, scope.St, scope, trcAst, fmt.Sprintf("running with new ast: %v", trcAst.PathUnique()))
	} else {
		Trace.Out(ps, pr, Run, scope.St, scope, trcAst, fmt.Sprintf("running with par ast: %v", trcAst.PathUnique()))
	}

	if pr.Reverse {
		return pr.DoRulesRevBinExp(ps, par, parAst, scope, mpos, ourAst, optMap)
	}

	nr := len(pr.Rules)
	valid := true
	creg := scope
	for i := 0; i < nr; i++ {
		rr := pr.Rules[i]
		if rr.IsToken() && !rr.Opt {
			ps.Pos, _ = ps.Src.NextTokenPos(mpos[i]) // already matched -- move past
			Trace.Out(ps, pr, Run, mpos[i], scope, trcAst, fmt.Sprintf("%v: token: %v", i, rr.Tok))
			if ourAst != nil {
				ourAst.SetTokRegEnd(ps.Pos, ps.Src) // update our end to any tokens that match
			}
			continue
		}
		creg.St = ps.Pos
		creg.Ed = scope.Ed
		for mi := i + 1; mi < nr; mi++ {
			if mpos[mi] != lex.PosZero {
				creg.Ed = mpos[mi] // only look up to point of next matching token
				break
			}
		}
		if rr.IsToken() { // opt by definition here
			if creg.IsNil() { // no tokens left..
				Trace.Out(ps, pr, Run, creg.St, scope, trcAst, fmt.Sprintf("%v: opt token: %v no more src", i, rr.Tok))
				continue
			}
			slx := ps.Src.LexAt(creg.St)
			kt := rr.Tok
			kt.Depth = slx.Depth
			pos, ok := ps.FindToken(kt, creg)
			if !ok {
				Trace.Out(ps, pr, NoMatch, creg.St, creg, parAst, fmt.Sprintf("%v token: %v", i, kt.String()))
				continue
			}
			Trace.Out(ps, pr, Match, pos, creg, parAst, fmt.Sprintf("%v token: %v", i, kt))
			ps.Pos, _ = ps.Src.NextTokenPos(pos)
			continue
		}

		if creg.IsNil() { // no tokens left..
			if rr.Opt {
				Trace.Out(ps, pr, Run, creg.St, scope, trcAst, fmt.Sprintf("%v: opt rule: %v no more src", i, rr.Rule.Name()))
				continue
			}
			ps.Error(creg.St, fmt.Sprintf("rule %v: non-optional sub-rule has no tokens: %v scope: %v", pr.Nm, rr.Rule.Name(), creg))
			valid = false
			break // no point in continuing
		}
		useAst := parAst
		if pr.Ast == AnchorAst {
			useAst = ourAst
		}
		// NOTE: we can't use anything about the previous match here, because it could have
		// come from a sub-sub-rule and in any case is not where you want to start
		// because is could have been a token in the middle.
		Trace.Out(ps, pr, Run, creg.St, creg, trcAst, fmt.Sprintf("%v: trying rule: %v", i, rr.Rule.Name()))
		subm := rr.Rule.Parse(ps, pr, useAst, creg, optMap)
		if subm == nil {
			if !rr.Opt {
				ps.Error(creg.St, fmt.Sprintf("rule %v: required sub-rule %v not matched", pr.Nm, rr.Rule.Name()))
				valid = false
			} else {
				Trace.Out(ps, pr, Run, creg.St, creg, trcAst, fmt.Sprintf("%v: optional rule: %v failed", i, rr.Rule.Name()))
			}
		}
		if !rr.Opt && ourAst != nil {
			ourAst.SetTokRegEnd(ps.Pos, ps.Src) // update our end to include non-optional elements
		}
	}
	return valid
}

// DoRulesRevBinExp reverse version of do rules for binary expression rule with
// one key token in the middle -- we just pay attention to scoping rest of sub-rules
// relative to that, and don't otherwise adjust scope or position.  In particular all
// the position updating taking place in sup-rules is then just ignored and we set the
// position to the end position matched by the "last" rule (which was the first processed)
func (pr *Rule) DoRulesRevBinExp(ps *State, par *Rule, parAst *Ast, scope lex.Reg, mpos Matches, ourAst *Ast, optMap lex.TokenMap) bool {
	nr := len(pr.Rules)
	valid := true
	creg := scope

	trcAst := parAst
	if ourAst != nil {
		trcAst = ourAst
	}
	tokpos := mpos[1]
	aftMpos, ok := ps.Src.NextTokenPos(tokpos)
	if !ok {
		ps.Error(tokpos, fmt.Sprintf("rule %v: end-of-tokens with more to match", pr.Nm))
		return false
	}

	epos := scope.Ed
	for i := nr - 1; i >= 0; i-- {
		rr := pr.Rules[i]
		if i > 1 {
			creg.St = aftMpos // end expr is in region from key token to end of scope
			ps.Pos = creg.St  // only works for a single rule after key token -- sub-rules not necc reverse
			creg.Ed = scope.Ed
		} else if i == 1 {
			Trace.Out(ps, pr, Run, tokpos, scope, trcAst, fmt.Sprintf("%v: key token: %v", i, rr.Tok))
			continue
		} else { // start
			creg.St = scope.St
			ps.Pos = creg.St
			creg.Ed = tokpos
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
			Trace.Out(ps, pr, Run, creg.St, creg, trcAst, fmt.Sprintf("%v: trying rule: %v", i, rr.Rule.Name()))
			subm := rr.Rule.Parse(ps, pr, useAst, creg, optMap)
			if subm == nil {
				if !rr.Opt {
					ps.Error(creg.St, fmt.Sprintf("rule %v: required sub-rule %v not matched", pr.Nm, rr.Rule.Name()))
					valid = false
				}
			}
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
