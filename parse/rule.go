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
	"io"
	"reflect"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/goki/ki"
	"github.com/goki/ki/indent"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/syms"
	"github.com/goki/pi/token"
)

// Set GuiActive to true if the gui (piview) is active -- ensures that the
// Ast tree is updated when nodes are swapped in reverse mode, and maybe
// other things
var GuiActive = false

// DepthLimit is the infinite recursion prevention cutoff
var DepthLimit = 1000

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
	Off          bool             `desc:"disable this rule -- useful for testing and exploration"`
	Desc         string           `desc:"description / comments about this rule"`
	Rule         string           `desc:"the rule as a space-separated list of rule names and token(s) -- use single quotes around 'tokens' (using token.Tokens names or symbols). For keywords use 'key:keyword'.  All tokens are matched at the same nesting depth as the start of the scope of this rule, unless they have a +D relative depth value differential before the token.  Use @ prefix for a sub-rule to require that rule to match -- by default explicit tokens are used if available, and then only the first sub-rule failing that.  Use ! by itself to define start of an exclusionary rule -- doesn't match when those rule elements DO match.  Use : prefix for a special group node that matches a single token at start of scope, and then defers to the child rules to perform full match -- this is used for FirstTokMap when there are multiple versions of a given keyword rule."`
	StackMatch   string           `desc:"if present, this rule only fires if stack has this on it"`
	Ast          AstActs          `desc:"what action should be take for this node when it matches"`
	Acts         Acts             `desc:"actions to perform based on parsed Ast tree data, when this rule is done executing"`
	OptTokMap    bool             `desc:"for group-level rules having lots of children and lots of recursiveness, and also of high-frequency, when we first encounter such a rule, make a map of all the tokens in the entire scope, and use that for a first-pass rejection on matching tokens"`
	FirstTokMap  bool             `desc:"for group-level rules with a number of rules that match based on first tokens / keywords, build map to directly go to that rule -- must also organize all of these rules sequentially from the start -- if no match, goes directly to first non-lookup case"`
	Rules        RuleList         `json:"-" xml:"-" desc:"rule elements compiled from Rule string"`
	Order        []int            `json:"-" xml:"-" desc:"strategic matching order for matching the rules"`
	FiTokMap     map[string]*Rule `inactive:"+" json:"-" xml:"-" desc:"map from first tokens / keywords to rules for FirstTokMap case"`
	FiTokElseIdx int              `inactive:"+" json:"-" xml:"-" desc:"for FirstTokMap, the start of the else cases not covered by the map"`
	ExclKeyIdx   int              `inactive:"+" json:"-" xml:"-" desc:"exclusionary key index -- this is the token in Rules that we need to exclude matches for using ExclFwd and ExclRev rules"`
	ExclFwd      RuleList         `inactive:"+" json:"-" xml:"-" desc:"exclusionary forward-search rule elements compiled from Rule string"`
	ExclRev      RuleList         `inactive:"+" json:"-" xml:"-" desc:"exclusionary reverse-search rule elements compiled from Rule string"`
}

var KiT_Rule = kit.Types.AddType(&Rule{}, RuleProps)

// RuleFlags define bitflags for rule options compiled from rule syntax
type RuleFlags int

const (
	// SetsScope means that this rule sets its own scope, because it ends with EOS
	SetsScope RuleFlags = RuleFlags(ki.FlagsN) + iota

	// Reverse means that this rule runs in reverse (starts with - sign) -- for arithmetic
	// binary expressions only: this is needed to produce proper associativity result for
	// mathematical expressions in the recursive descent parser.
	// Only for rules of form: Expr '+' Expr -- two sub-rules with a token operator
	// in the middle.
	Reverse

	// NoToks means that this rule doesn't have any explicit tokens -- only refers to
	// other rules
	NoToks

	// OnlyToks means that this rule only has explicit tokens for matching -- can be
	// optimized
	OnlyToks

	// MatchEOS means that the rule ends with a *matched* EOS with StInc = 1.
	// SetsScope applies for optional and matching EOS rules alike.
	MatchEOS

	// MultiEOS means that the rule has multiple EOS tokens within it --
	// changes some of the logic
	MultiEOS

	// TokMatchGroup is a group node that also has a single token match, so it can
	// be used in a FirstTokMap to optimize lookup of rules
	TokMatchGroup
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
	Parse(ps *State, par *Rule, ast *Ast, scope lex.Reg, optMap lex.TokenMap, depth int) *Rule

	// AsParseRule returns object as a parse.Rule
	AsParseRule() *Rule
}

// check that Rule implements Parser interface
var _ Parser = (*Rule)(nil)

// RuleEl is an element of a parsing rule -- either a pointer to another rule or a token
type RuleEl struct {
	Rule  *Rule          `desc:"sub-rule for this position -- nil if token"`
	Tok   token.KeyToken `desc:"token, None if rule"`
	StInc int            `desc:"start increment for matching -- this is the number of non-optional, non-match items between (start | last match) and this item -- increments start region for matching"`
	Match bool           `desc:"if true, this rule must match for rule to fire -- by default only tokens and, failing that, the first sub-rule is used for matching -- use @ to require a match"`
	Opt   bool           `desc:"this rule is optional -- will absorb tokens if they exist -- indicated with ? prefix"`
	FmEnd bool           `desc:"match this rule working backward from the end -- triggered by - (minus) prefix and optimizes cases where there can be a lot of tokens going forward but few going from end -- must be anchored by a terminal EOS or other reverse-search elements and is ignored if at the very end"`
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

// Matches encodes the regions of each match, Err for no match
type Matches []lex.Reg

// StartEnd returns the first and last non-zero positions in the Matches list as a region
func (mm Matches) StartEnd() lex.Reg {
	reg := lex.RegZero
	for _, mp := range mm {
		if mp.St != lex.PosZero {
			if reg.St == lex.PosZero {
				reg.St = mp.St
			}
			reg.Ed = mp.Ed
		}
	}
	return reg
}

///////////////////////////////////////////////////////////////////////
//  Rule

func (pr *Rule) BaseIface() reflect.Type {
	return reflect.TypeOf((*Parser)(nil)).Elem()
}

func (pr *Rule) AsParseRule() *Rule {
	return pr.This().(*Rule)
}

// IsGroup returns true if this node is a group, else it should have rules
func (pr *Rule) IsGroup() bool {
	return pr.HasChildren()
}

// SetRuleMap is called on the top-level Rule and initializes the RuleMap
func (pr *Rule) SetRuleMap(ps *State) {
	RuleMap = map[string]*Rule{}
	pr.FuncDownMeFirst(0, pr.This(), func(k ki.Ki, level int, d interface{}) bool {
		pri := k.(*Rule)
		if epr, has := RuleMap[pri.Nm]; has {
			ps.Error(lex.PosZero, fmt.Sprintf("Parser Compile: multiple rules with same name: %v and %v", pri.PathUnique(), epr.PathUnique()), pri)
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
		pri := k.(*Rule)
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
	if pr.Off {
		pr.SetProp("inactive", true)
	} else {
		pr.DeleteProp("inactive")
	}
	if pr.Rule == "" { // parent
		pr.Rules = nil
		pr.ClearFlag(int(SetsScope))
		return true
	}
	valid := true
	rstr := pr.Rule
	if pr.Rule[0] == '-' {
		rstr = rstr[1:]
		pr.SetFlag(int(Reverse))
	} else {
		pr.ClearFlag(int(Reverse))
	}
	rs := strings.Split(rstr, " ")
	nr := len(rs)

	pr.Rules = make(RuleList, nr)
	pr.ExclFwd = nil
	pr.ExclRev = nil
	pr.ClearFlag(int(NoToks))
	pr.SetFlag(int(OnlyToks)) // default is this..
	pr.ClearFlag(int(SetsScope))
	pr.ClearFlag(int(MatchEOS))
	pr.ClearFlag(int(MultiEOS))
	pr.ClearFlag(int(TokMatchGroup))
	pr.Order = nil
	nmatch := 0
	ntok := 0
	curStInc := 0
	eoses := 0
	for ri := range rs {
		rn := strings.TrimSpace(rs[ri])
		if len(rn) == 0 {
			ps.Error(lex.PosZero, "Compile: Rules has empty string -- make sure there is only one space between rule elements", pr)
			valid = false
			break
		}
		if rn == "!" { // exclusionary rule
			nr = ri
			pr.Rules = pr.Rules[:ri]
			pr.CompileExcl(ps, rs, ri+1)
			break
		}
		if rn[0] == ':' {
			pr.SetFlag(int(TokMatchGroup))
		}
		rr := &pr.Rules[ri]
		tokst := strings.Index(rn, "'")
		if tokst >= 0 {
			if rn[0] == '?' {
				rr.Opt = true
			} else {
				rr.StInc = curStInc
				rr.Match = true // all tokens match by default
				pr.Order = append(pr.Order, ri)
				nmatch++
				ntok++
				curStInc = 0
			}
			sz := len(rn)
			if rn[0] == '+' {
				td, _ := strconv.ParseInt(rn[1:tokst], 10, 64)
				rr.Tok.Depth = int(td)
			} else if rn[0] == '-' {
				rr.FmEnd = true
			}
			tn := rn[tokst+1 : sz-1]
			if len(tn) > 4 && tn[:4] == "key:" {
				rr.Tok.Tok = token.Keyword
				rr.Tok.Key = tn[4:]
			} else {
				if pmt, has := token.OpPunctMap[tn]; has {
					rr.Tok.Tok = pmt
				} else {
					err := rr.Tok.Tok.FromString(tn)
					if err != nil {
						ps.Error(lex.PosZero, fmt.Sprintf("Compile: token convert error: %v", err.Error()), pr)
						valid = false
					}
				}
			}
			if rr.Tok.Tok == token.EOS {
				eoses++
				if eoses > 1 {
					pr.SetFlag(int(MultiEOS))
				}
				if ri == nr-1 {
					rr.StInc = eoses
					pr.SetFlag(int(SetsScope))
					if rr.Match && eoses == 1 {
						pr.SetFlag(int(MatchEOS))
					}
				}
			}
		} else {
			st := 0
			if rn[0] == '?' {
				st = 1
				rr.Opt = true
			} else if rn[0] == '@' {
				st = 1
				rr.Match = true
				pr.ClearFlag(int(OnlyToks))
				pr.Order = append(pr.Order, ri)
				nmatch++
			} else {
				curStInc++
			}
			rp, ok := RuleMap[rn[st:]]
			if !ok {
				ps.Error(lex.PosZero, fmt.Sprintf("Compile: refers to rule %v not found", rn), pr)
				valid = false
			} else {
				rr.Rule = rp
			}
		}
	}
	if pr.HasFlag(int(Reverse)) {
		pr.Ast = AnchorAst // must be
	}
	if ntok == 0 && nmatch == 0 {
		pr.Rules[0].Match = true
		pr.Order = append(pr.Order, 0)
		pr.SetFlag(int(NoToks))
	} else {
		pr.OptimizeOrder(ps)
	}
	return valid
}

// OptimizeOrder optimizes the order of processing rule elements, including:
// * A block of reversed elements that match backward
func (pr *Rule) OptimizeOrder(ps *State) {
	osz := len(pr.Order)
	if osz == 0 {
		return
	}
	nfmend := 0
	fmeSt := -1
	fmeEd := -1
	lastwas := false
	for oi := 0; oi < osz; oi++ {
		ri := pr.Order[oi]
		rr := &pr.Rules[ri]
		if rr.FmEnd {
			nfmend++
			if fmeSt < 0 {
				fmeSt = oi
			}
			if lastwas {
				fmeEd = oi // end of block
			}
			lastwas = true
		} else {
			lastwas = false
		}
	}
	if nfmend > 1 && fmeEd > 0 {
		nword := make([]int, osz)
		for oi := 0; oi < fmeSt; oi++ {
			nword[oi] = pr.Order[oi]
		}
		idx := fmeSt
		for oi := fmeEd - 1; oi >= fmeSt; oi-- {
			nword[idx] = pr.Order[oi]
			idx++
		}
		for oi := fmeEd; oi < osz; oi++ {
			nword[oi] = pr.Order[oi]
		}
		pr.Order = nword
	}
}

// CompileTokMap compiles first token map
func (pr *Rule) CompileTokMap(ps *State) bool {
	valid := true
	pr.FiTokMap = make(map[string]*Rule, len(pr.Kids))
	pr.FiTokElseIdx = len(pr.Kids)
	for i, kpri := range pr.Kids {
		kpr := kpri.(*Rule)
		if len(kpr.Rules) == 0 || !kpr.Rules[0].IsToken() {
			pr.FiTokElseIdx = i
			break
		}
		fr := kpr.Rules[0]
		skey := fr.Tok.StringKey()
		if _, has := pr.FiTokMap[skey]; has {
			ps.Error(lex.PosZero, fmt.Sprintf("CompileFirstTokMap: multiple rules have the same first token: %v -- must be unique -- use a :'tok' group to match that first token and put all the sub-rules as children of that node", fr.Tok), pr)
			pr.FiTokElseIdx = 0
			valid = false
		} else {
			pr.FiTokMap[skey] = kpr
		}
	}
	return valid
}

// CompileExcl compiles exclusionary rules starting at given point
// currently only working for single-token matching rule
func (pr *Rule) CompileExcl(ps *State, rs []string, rist int) bool {
	valid := true
	nr := len(rs)
	var ktok token.KeyToken

	ktoki := -1
	for ri := 0; ri < rist; ri++ {
		rr := &pr.Rules[ri]
		if !rr.IsToken() {
			continue
		}
		ktok = rr.Tok
		ktoki = ri
		break
	}

	if ktoki < 0 {
		ps.Error(lex.PosZero, "CompileExcl: no token found for matching exclusion rules", pr)
		return false
	}

	pr.ExclRev = make(RuleList, nr-rist)
	ki := -1
	for ri := rist; ri < nr; ri++ {
		rn := strings.TrimSpace(rs[ri])
		rr := &pr.ExclRev[ri-rist]
		if rn[0] == '?' {
			rr.Opt = true
		}
		tokst := strings.Index(rn, "'")
		if tokst < 0 {
			continue // pure optional
		}
		if !rr.Opt {
			rr.Match = true // all tokens match by default
		}
		sz := len(rn)
		if rn[0] == '+' {
			td, _ := strconv.ParseInt(rn[1:tokst], 10, 64)
			rr.Tok.Depth = int(td)
		}
		tn := rn[tokst+1 : sz-1]
		if len(tn) > 4 && tn[:4] == "key:" {
			rr.Tok.Tok = token.Keyword
			rr.Tok.Key = tn[4:]
		} else {
			if pmt, has := token.OpPunctMap[tn]; has {
				rr.Tok.Tok = pmt
			} else {
				err := rr.Tok.Tok.FromString(tn)
				if err != nil {
					ps.Error(lex.PosZero, fmt.Sprintf("CompileExcl: token convert error: %v", err.Error()), pr)
					valid = false
				}
			}
		}
		if rr.Tok.Equal(ktok) {
			ki = ri
		}
	}
	if ki < 0 {
		ps.Error(lex.PosZero, fmt.Sprintf("CompileExcl: key token: %v not found in exclusion rule", ktok), pr)
		valid = false
		return valid
	}
	pr.ExclKeyIdx = ktoki
	pr.ExclFwd = pr.ExclRev[ki+1-rist:]
	pr.ExclRev = pr.ExclRev[:ki-rist]
	return valid
}

// Validate checks for any errors in the rules and issues warnings,
// returns true if valid (no err) and false if invalid (errs)
func (pr *Rule) Validate(ps *State) bool {
	valid := true

	// do this here so everything else is compiled
	if len(pr.Rules) == 0 && pr.FirstTokMap {
		pr.CompileTokMap(ps)
	}

	if len(pr.Rules) == 0 && !pr.HasChildren() && !pr.IsRoot() {
		ps.Error(lex.PosZero, "Validate: rule has no rules and no children", pr)
		valid = false
	}
	if !pr.HasFlag(int(TokMatchGroup)) && len(pr.Rules) > 0 && pr.HasChildren() {
		ps.Error(lex.PosZero, "Validate: rule has both rules and children -- should be either-or", pr)
		valid = false
	}
	if pr.HasFlag(int(Reverse)) {
		if len(pr.Rules) != 3 {
			ps.Error(lex.PosZero, "Validate: a Reverse (-) rule must have 3 children -- for binary operator expressions only", pr)
			valid = false
		} else {
			if !pr.Rules[1].IsToken() {
				ps.Error(lex.PosZero, "Validate: a Reverse (-) rule must have a token to be recognized in the middle of two rules -- for binary operator expressions only", pr)
			}
		}
	}

	if len(pr.Rules) > 0 {
		if pr.Rules[0].IsRule() && (pr.Rules[0].Rule == pr || pr.ParentLevel(pr.Rules[0].Rule) >= 0) { // left recursive
			if pr.Rules[0].Match {
				ps.Error(lex.PosZero, fmt.Sprintf("Validate: rule refers to itself recursively in first sub-rule: %v and that sub-rule is marked as a Match -- this is infinite recursion and is not allowed!  Must use distinctive tokens in rule to match this rule, and then left-recursive elements will be filled in when the rule runs, but they cannot be used for matching rule.", pr.Rules[0].Rule.Name()), pr)
				valid = false
			}
			ntok := 0
			for _, rr := range pr.Rules {
				if rr.IsToken() {
					ntok++
				}
			}
			if ntok == 0 {
				ps.Error(lex.PosZero, fmt.Sprintf("Validate: rule refers to itself recursively in first sub-rule: %v, and does not have any tokens in the rule -- MUST promote tokens to this rule to disambiguate match, otherwise will just do infinite recursion!", pr.Rules[0].Rule.Name()), pr)
				valid = false
			}
		}
	}

	// now we iterate over our kids
	for _, kpri := range pr.Kids {
		kpr := kpri.(*Rule)
		if !kpr.Validate(ps) {
			valid = false
		}
	}
	return valid
}

// StartParse is called on the root of the parse rule tree to start the parsing process
func (pr *Rule) StartParse(ps *State) *Rule {
	if ps.AtEofNext() || !pr.HasChildren() {
		ps.GotoEof()
		return nil
	}
	kpr := pr.Kids[0].(*Rule) // first rule is special set of valid top-level matches
	var parAst *Ast
	scope := lex.Reg{St: ps.Pos}
	if ps.Ast.HasChildren() {
		parAst = ps.Ast.KnownChild(0).(*Ast)
	} else {
		parAst = ps.Ast.AddNewChild(KiT_Ast, kpr.Name()).(*Ast)
		ok := false
		scope.St, ok = ps.Src.ValidTokenPos(scope.St)
		if !ok {
			ps.GotoEof()
			return nil
		}
		ps.Pos = scope.St
	}
	didErr := false
	for {
		cpos := ps.Pos
		mrule := kpr.Parse(ps, pr, parAst, scope, nil, 0)
		ps.ResetNonMatches()
		if ps.AtEof() {
			return nil
		}
		if cpos == ps.Pos {
			if !didErr {
				ps.Error(cpos, "did not advance position -- need more rules to match current input -- skipping to next EOS", pr)
				didErr = true
			}
			cp, ok := ps.Src.NextTokenPos(ps.Pos)
			if !ok {
				ps.GotoEof()
				return nil
			}
			ep, ok := ps.Src.NextEosAnyDepth(cp)
			if !ok {
				ps.GotoEof()
				return nil
			}
			ps.Pos = ep
		} else {
			return mrule
		}
	}
}

// Parse tries to apply rule to given input state, returns rule that matched or nil
// par is the parent rule that we're being called from.
// parAst is the current ast node that we add to.
// scope is the region to search within, defined by parent or EOS if we have a terminal
// one
func (pr *Rule) Parse(ps *State, par *Rule, parAst *Ast, scope lex.Reg, optMap lex.TokenMap, depth int) *Rule {
	if pr.Off {
		return nil
	}

	if depth >= DepthLimit {
		ps.Error(scope.St, "depth limit exceeded -- parser rules error -- look for recursive cases", pr)
		return nil
	}

	nr := len(pr.Rules)
	if !pr.HasFlag(int(TokMatchGroup)) && nr > 0 {
		return pr.ParseRules(ps, par, parAst, scope, optMap, depth)
	}

	if optMap == nil && pr.OptTokMap {
		optMap = ps.Src.TokenMapReg(scope)
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, Run, scope.St, scope, parAst, fmt.Sprintf("made optmap of size: %d", len(optMap)))
		}
	}

	// pure group types just iterate over kids
	for _, kpri := range pr.Kids {
		kpr := kpri.(*Rule)
		if mrule := kpr.Parse(ps, pr, parAst, scope, optMap, depth+1); mrule != nil {
			return mrule
		}
	}
	return nil
}

// ParseRules parses rules and returns this rule if it matches, nil if not
func (pr *Rule) ParseRules(ps *State, par *Rule, parAst *Ast, scope lex.Reg, optMap lex.TokenMap, depth int) *Rule {
	ok := false
	if pr.HasFlag(int(SetsScope)) {
		scope, ok = pr.Scope(ps, parAst, scope)
		if !ok {
			return nil
		}
	} else if GuiActive {
		if scope == lex.RegZero {
			ps.Error(scope.St, "scope is empty and no EOS in rule -- invalid rules -- starting rules must all have EOS", pr)
			return nil
		}
	}
	match, nscope, mpos := pr.Match(ps, parAst, scope, 0, optMap)
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
	valid := pr.DoRules(ps, par, parAst, nscope, mpos, optMap, depth) // returns validity but we don't care once matched..
	if !valid {
		return nil
	}
	return pr
}

// Scope finds the potential scope region for looking for tokens -- either from
// EOS position or State ScopeStack pushed from parents.
// Returns new scope and false if no valid scope found.
func (pr *Rule) Scope(ps *State, parAst *Ast, scope lex.Reg) (lex.Reg, bool) {
	// prf := prof.Start("Scope")
	// defer prf.End()

	nscope := scope
	creg := scope
	lr := pr.Rules.Last()
	for ei := 0; ei < lr.StInc; ei++ {
		stlx := ps.Src.LexAt(creg.St)
		ep, ok := ps.Src.NextEos(creg.St, stlx.Tok.Depth)
		if !ok {
			// ps.Error(creg.St, "could not find EOS at target nesting depth -- parens / bracket / brace mismatch?", pr)
			return nscope, false
		}
		if scope.Ed != lex.PosZero && lr.Opt && scope.Ed.IsLess(ep) {
			// optional tokens can't take us out of scope
			return scope, true
		}
		if ei == lr.StInc-1 {
			nscope.Ed = ep
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, SubMatch, nscope.St, nscope, parAst, fmt.Sprintf("from EOS: starting scope: %v new scope: %v end pos: %v depth: %v", scope, nscope, ep, stlx.Tok.Depth))
			}
		} else {
			creg.St, ok = ps.Src.NextTokenPos(ep) // advance
			if !ok {
				// ps.Error(scope.St, "end of file looking for EOS tokens -- premature file end?", pr)
				return nscope, false
			}
		}
	}
	return nscope, true
}

// Match attempts to match the rule, returns true if it matches, and the
// match positions, along with any update to the scope
func (pr *Rule) Match(ps *State, parAst *Ast, scope lex.Reg, depth int, optMap lex.TokenMap) (bool, lex.Reg, Matches) {
	if pr.Off {
		return false, scope, nil
	}

	if depth > DepthLimit {
		ps.Error(scope.St, "depth limit exceeded -- parser rules error -- look for recursive cases", pr)
		return false, scope, nil
	}

	if ps.IsNonMatch(scope, pr) {
		return false, scope, nil
	}

	if pr.StackMatch != "" {
		if ps.Stack.Top() != pr.StackMatch {
			return false, scope, nil
		}
	}

	// mprf := prof.Start("Match")
	// defer mprf.End()
	// Note: uncomment the following to see which rules are taking the most
	// time -- very helpful for focusing effort on optimizing those rules.
	// prf := prof.Start(pr.Nm)
	// defer prf.End()

	nr := len(pr.Rules)
	if pr.HasFlag(int(TokMatchGroup)) || nr == 0 { // Group
		return pr.MatchGroup(ps, parAst, scope, depth, optMap)
	}

	// prf := prof.Start("IsMatch")
	if mst, match := ps.IsMatch(pr, scope); match {
		// prf.End()
		return true, scope, mst.Regs
	}
	// prf.End()

	var mpos Matches
	match := false

	if pr.HasFlag(int(NoToks)) {
		match, mpos = pr.MatchNoToks(ps, parAst, scope, depth, optMap)
	} else if pr.HasFlag(int(OnlyToks)) {
		match, mpos = pr.MatchOnlyToks(ps, parAst, scope, depth, optMap)
	} else {
		match, mpos = pr.MatchMixed(ps, parAst, scope, depth, optMap)
	}
	if !match {
		ps.AddNonMatch(scope, pr)
		return false, scope, nil
	}

	if len(pr.ExclFwd) > 0 || len(pr.ExclRev) > 0 {
		ktpos := mpos[pr.ExclKeyIdx]
		if pr.MatchExclude(ps, scope, ktpos, depth, optMap) {
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, NoMatch, ktpos.St, scope, parAst, "Exclude critera matched")
			}
			ps.AddNonMatch(scope, pr)
			return false, scope, nil
		}
	}

	mreg := mpos.StartEnd()
	ps.AddMatch(pr, scope, mpos)
	if ps.Trace.On {
		ps.Trace.Out(ps, pr, Match, mreg.St, scope, parAst, fmt.Sprintf("Full Match reg: %v", mreg))
	}
	return true, scope, mpos
}

// MatchOnlyToks matches rules having only tokens
func (pr *Rule) MatchOnlyToks(ps *State, parAst *Ast, scope lex.Reg, depth int, optMap lex.TokenMap) (bool, Matches) {
	nr := len(pr.Rules)

	var mpos Matches

	scstlx := ps.Src.LexAt(scope.St) // scope starting lex
	scstDepth := scstlx.Tok.Depth

	creg := scope
	osz := len(pr.Order)
	for oi := 0; oi < osz; oi++ {
		ri := pr.Order[oi]
		rr := &pr.Rules[ri]
		kt := rr.Tok
		if optMap != nil && !optMap.Has(kt.Tok) { // not even a possibility
			return false, nil
		}
		if rr.FmEnd {
			if mpos == nil {
				mpos = make(Matches, nr) // make on demand -- cuts out a lot of allocations!
			}
			mpos[nr-1] = lex.Reg{scope.Ed, scope.Ed}
		}
		kt.Depth += scstDepth // always use starting scope depth
		match, tpos := pr.MatchToken(ps, rr, ri, kt, &creg, mpos, parAst, scope, depth, optMap)
		if !match {
			if ps.Trace.On {
				if tpos != lex.PosZero {
					tlx := ps.Src.LexAt(tpos)
					ps.Trace.Out(ps, pr, NoMatch, creg.St, creg, parAst, fmt.Sprintf("%v token: %v, was: %v", ri, kt.String(), tlx.String()))
				} else {
					ps.Trace.Out(ps, pr, NoMatch, creg.St, creg, parAst, fmt.Sprintf("%v token: %v, nil region", ri, kt.String()))
				}
			}
			return false, nil
		}
		if mpos == nil {
			mpos = make(Matches, nr) // make on demand -- cuts out a lot of allocations!
		}
		mpos[ri] = lex.Reg{tpos, tpos}
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, SubMatch, creg.St, creg, parAst, fmt.Sprintf("%v token: %v", ri, kt.String()))
		}
	}

	return true, mpos
}

// MatchToken matches one token sub-rule -- returns true for match and
// false if no match -- and the position where it was / should have been
func (pr *Rule) MatchToken(ps *State, rr *RuleEl, ri int, kt token.KeyToken, creg *lex.Reg, mpos Matches, parAst *Ast, scope lex.Reg, depth int, optMap lex.TokenMap) (bool, lex.Pos) {
	nr := len(pr.Rules)
	ok := false
	matchst := false // match start of creg
	matched := false // match end of creg
	var tpos lex.Pos
	if ri == 0 {
		matchst = true
	} else if mpos != nil {
		lpos := mpos[ri-1].Ed
		if lpos != lex.PosZero { // previous has matched
			matchst = true
		} else if ri < nr-1 && rr.FmEnd {
			lpos := mpos[ri+1].St
			if lpos != lex.PosZero { // previous has matched
				creg.Ed, _ = ps.Src.PrevTokenPos(lpos)
				matched = true
			}
		}
	}
	for stinc := 0; stinc < rr.StInc; stinc++ {
		creg.St, _ = ps.Src.NextTokenPos(creg.St)
	}
	if ri == nr-1 && rr.Tok.Tok == token.EOS {
		return true, scope.Ed
	}
	if creg.IsNil() && !matched {
		return false, tpos
	}

	if matchst { // start token must be right here
		if !ps.MatchToken(kt, creg.St) {
			return false, creg.St
		}
		tpos = creg.St
	} else if matched {
		if !ps.MatchToken(kt, creg.Ed) {
			return false, creg.Ed
		}
		tpos = creg.Ed
	} else {
		// prf := prof.Start("FindToken")
		if pr.HasFlag(int(Reverse)) {
			tpos, ok = ps.FindTokenReverse(kt, *creg)
		} else {
			tpos, ok = ps.FindToken(kt, *creg)
		}
		// prf.End()
		if !ok {
			return false, tpos
		}
	}
	creg.St, _ = ps.Src.NextTokenPos(tpos) // always ratchet up
	return true, tpos
}

// MatchMixed matches mixed tokens and non-tokens
func (pr *Rule) MatchMixed(ps *State, parAst *Ast, scope lex.Reg, depth int, optMap lex.TokenMap) (bool, Matches) {
	nr := len(pr.Rules)
	var mpos Matches

	scstlx := ps.Src.LexAt(scope.St) // scope starting lex
	scstDepth := scstlx.Tok.Depth

	creg := scope
	osz := len(pr.Order)
	for oi := 0; oi < osz; oi++ {
		ri := pr.Order[oi]
		rr := &pr.Rules[ri]

		/////////////////////////////////////////////
		// Token
		if rr.IsToken() {
			kt := rr.Tok
			if optMap != nil && !optMap.Has(kt.Tok) { // not even a possibility
				return false, nil
			}
			if rr.FmEnd {
				if mpos == nil {
					mpos = make(Matches, nr) // make on demand -- cuts out a lot of allocations!
				}
				mpos[nr-1] = lex.Reg{scope.Ed, scope.Ed}
			}
			kt.Depth += scstDepth // always use starting scope depth
			match, tpos := pr.MatchToken(ps, rr, ri, kt, &creg, mpos, parAst, scope, depth, optMap)
			if !match {
				if ps.Trace.On {
					if tpos != lex.PosZero {
						tlx := ps.Src.LexAt(tpos)
						ps.Trace.Out(ps, pr, NoMatch, creg.St, creg, parAst, fmt.Sprintf("%v token: %v, was: %v", ri, kt.String(), tlx.String()))
					} else {
						ps.Trace.Out(ps, pr, NoMatch, creg.St, creg, parAst, fmt.Sprintf("%v token: %v, nil region", ri, kt.String()))
					}
				}
				return false, nil
			}
			if mpos == nil {
				mpos = make(Matches, nr) // make on demand -- cuts out a lot of allocations!
			}
			mpos[ri] = lex.Reg{tpos, tpos}
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, SubMatch, creg.St, creg, parAst, fmt.Sprintf("%v token: %v", ri, kt.String()))
			}
			continue
		}

		//////////////////////////////////////////////
		// Sub-Rule

		if creg.IsNil() {
			ps.Trace.Out(ps, pr, NoMatch, creg.St, creg, parAst, fmt.Sprintf("%v sub-rule: %v, nil region", ri, rr.Rule.Name()))
			return false, nil
		}

		// first, limit region to same depth or greater as start of region -- prevents
		// overflow beyond natural boundaries
		stlx := ps.Src.LexAt(creg.St) // scope starting lex
		cp, _ := ps.Src.NextTokenPos(creg.St)
		stdp := stlx.Tok.Depth
		for cp.IsLess(creg.Ed) {
			lx := ps.Src.LexAt(cp)
			if lx.Tok.Depth < stdp {
				creg.Ed = cp
				break
			}
			cp, _ = ps.Src.NextTokenPos(cp)
		}

		if ps.Trace.On {
			ps.Trace.Out(ps, pr, SubMatch, creg.St, creg, parAst, fmt.Sprintf("%v trying sub-rule: %v", ri, rr.Rule.Name()))
		}
		match, _, smpos := rr.Rule.Match(ps, parAst, creg, depth+1, optMap)
		if !match {
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, NoMatch, creg.St, creg, parAst, fmt.Sprintf("%v sub-rule: %v", ri, rr.Rule.Name()))
			}
			return false, nil
		}
		creg.Ed = scope.Ed // back to full scope
		// look through smpos for last valid position -- use that as last match pos
		mreg := smpos.StartEnd()
		lmnpos, ok := ps.Src.NextTokenPos(mreg.Ed)
		if !ok && !(ri == nr-1 || (ri == nr-2 && pr.HasFlag(int(SetsScope)))) {
			// if at end, or ends in EOS, then ok..
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, NoMatch, creg.St, creg, parAst, fmt.Sprintf("%v sub-rule: %v -- not at end and no tokens left", ri, rr.Rule.Name()))
			}
			return false, nil
		}
		if mpos == nil {
			mpos = make(Matches, nr) // make on demand -- cuts out a lot of allocations!
		}
		mpos[ri] = mreg
		creg.St = lmnpos
		if ps.Trace.On {
			msreg := mreg
			msreg.Ed = lmnpos
			ps.Trace.Out(ps, pr, SubMatch, mreg.St, msreg, parAst, fmt.Sprintf("%v rule: %v reg: %v", ri, rr.Rule.Name(), msreg))
		}
	}

	return true, mpos
}

// MatchNoToks matches NoToks case -- just does single sub-rule match
func (pr *Rule) MatchNoToks(ps *State, parAst *Ast, scope lex.Reg, depth int, optMap lex.TokenMap) (bool, Matches) {
	creg := scope
	ri := 0
	rr := &pr.Rules[0]
	if ps.Trace.On {
		ps.Trace.Out(ps, pr, SubMatch, creg.St, creg, parAst, fmt.Sprintf("%v trying sub-rule: %v", ri, rr.Rule.Name()))
	}
	match, _, smpos := rr.Rule.Match(ps, parAst, creg, depth+1, optMap)
	if !match {
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, NoMatch, creg.St, creg, parAst, fmt.Sprintf("%v sub-rule: %v", ri, rr.Rule.Name()))
		}
		return false, nil
	}
	if ps.Trace.On {
		mreg := smpos.StartEnd() // todo: should this include creg start instead?
		ps.Trace.Out(ps, pr, SubMatch, mreg.St, mreg, parAst, fmt.Sprintf("%v rule: %v reg: %v", ri, rr.Rule.Name(), mreg))
	}
	return true, smpos
}

// MatchGroup does matching for Group rules
func (pr *Rule) MatchGroup(ps *State, parAst *Ast, scope lex.Reg, depth int, optMap lex.TokenMap) (bool, lex.Reg, Matches) {
	// prf := prof.Start("SubMatch")
	if mst, match := ps.IsMatch(pr, scope); match {
		// 	prf.End()
		return true, scope, mst.Regs
	}
	// prf.End()
	sti := 0
	nk := len(pr.Kids)
	if pr.FirstTokMap {
		stlx := ps.Src.LexAt(scope.St)
		if kpr, has := pr.FiTokMap[stlx.Tok.StringKey()]; has {
			match, nscope, mpos := kpr.Match(ps, parAst, scope, depth+1, optMap)
			if match {
				if ps.Trace.On {
					ps.Trace.Out(ps, pr, SubMatch, scope.St, scope, parAst, fmt.Sprintf("first token group child: %v", kpr.Name()))
				}
				ps.AddMatch(pr, scope, mpos)
				return true, nscope, mpos
			}
		}
		sti = pr.FiTokElseIdx
	}

	for i := sti; i < nk; i++ {
		kpri := pr.Kids[i]
		kpr := kpri.(*Rule)
		match, nscope, mpos := kpr.Match(ps, parAst, scope, depth+1, optMap)
		if match {
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, SubMatch, scope.St, scope, parAst, fmt.Sprintf("group child: %v", kpr.Name()))
			}
			ps.AddMatch(pr, scope, mpos)
			return true, nscope, mpos
		}
	}
	ps.AddNonMatch(scope, pr)
	return false, scope, nil
}

// MatchExclude looks for matches of exclusion tokens -- if found, they exclude this rule
// return is true if exclude matches and rule should be excluded
func (pr *Rule) MatchExclude(ps *State, scope lex.Reg, ktpos lex.Reg, depth int, optMap lex.TokenMap) bool {
	nf := len(pr.ExclFwd)
	nr := len(pr.ExclRev)
	scstlx := ps.Src.LexAt(scope.St) // scope starting lex
	scstDepth := scstlx.Tok.Depth
	if nf > 0 {
		cp, ok := ps.Src.NextTokenPos(ktpos.St)
		if !ok {
			return false
		}
		prevAny := false
		for ri := 0; ri < nf; ri++ {
			rr := pr.ExclFwd[ri]
			kt := rr.Tok
			kt.Depth += scstDepth // always use starting scope depth
			if kt.Tok == token.None {
				prevAny = true // wild card
				continue
			}
			if prevAny {
				creg := scope
				creg.St = cp
				pos, ok := ps.FindToken(kt, creg)
				if !ok {
					return false
				}
				cp = pos
			} else {
				if !ps.MatchToken(kt, cp) {
					if !rr.Opt {
						return false
					}
					lx := ps.Src.LexAt(cp)
					if lx.Tok.Depth != kt.Depth {
						break
					}
					// ok, keep going -- no info..
				}
			}
			cp, ok = ps.Src.NextTokenPos(cp)
			if !ok && ri < nf-1 {
				return false
			}
			if scope.Ed == cp || scope.Ed.IsLess(cp) { // out of scope -- if non-opt left, nomatch
				ri++
				for ; ri < nf; ri++ {
					rr := pr.ExclFwd[ri]
					if !rr.Opt {
						return false
					}
				}
				break
			}
			prevAny = false
		}
	}
	if nr > 0 {
		cp, ok := ps.Src.PrevTokenPos(ktpos.St)
		if !ok {
			return false
		}
		prevAny := false
		for ri := nr - 1; ri >= 0; ri-- {
			rr := pr.ExclRev[ri]
			kt := rr.Tok
			kt.Depth += scstDepth // always use starting scope depth
			if kt.Tok == token.None {
				prevAny = true // wild card
				continue
			}
			if prevAny {
				creg := scope
				creg.Ed = cp
				pos, ok := ps.FindTokenReverse(kt, creg)
				if !ok {
					return false
				}
				cp = pos
			} else {
				if !ps.MatchToken(kt, cp) {
					if !rr.Opt {
						return false
					}
					lx := ps.Src.LexAt(cp)
					if lx.Tok.Depth != kt.Depth {
						break
					}
					// ok, keep going -- no info..
				}
			}
			cp, ok = ps.Src.PrevTokenPos(cp)
			if !ok && ri > 0 {
				return false
			}
			if cp.IsLess(scope.St) {
				ri--
				for ; ri >= 0; ri-- {
					rr := pr.ExclRev[ri]
					if !rr.Opt {
						return false
					}
				}
				break
			}
			prevAny = false
		}
	}
	return true
}

// DoRules after we have matched, goes through rest of the rules -- returns false if
// there were any issues encountered
func (pr *Rule) DoRules(ps *State, par *Rule, parAst *Ast, scope lex.Reg, mpos Matches, optMap lex.TokenMap, depth int) bool {
	trcAst := parAst
	var ourAst *Ast
	anchorFirst := (pr.Ast == AnchorFirstAst && parAst.Nm != pr.Nm)

	if pr.Ast != NoAst {
		// prf := prof.Start("AddAst")
		ourAst = ps.AddAst(parAst, pr.Name(), scope)
		// prf.End()
		trcAst = ourAst
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, Run, scope.St, scope, trcAst, fmt.Sprintf("running with new ast: %v", trcAst.PathUnique()))
		}
	} else {
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, Run, scope.St, scope, trcAst, fmt.Sprintf("running with par ast: %v", trcAst.PathUnique()))
		}
	}

	if pr.HasFlag(int(Reverse)) {
		return pr.DoRulesRevBinExp(ps, par, parAst, scope, mpos, ourAst, optMap, depth)
	}

	nr := len(pr.Rules)
	valid := true
	creg := scope
	for ri := 0; ri < nr; ri++ {
		pr.DoActs(ps, ri, par, ourAst, parAst)
		rr := &pr.Rules[ri]
		if rr.IsToken() && !rr.Opt {
			mp := mpos[ri].St
			if mp == ps.Pos {
				ps.Pos, _ = ps.Src.NextTokenPos(ps.Pos) // already matched -- move past
				if ps.Trace.On {
					ps.Trace.Out(ps, pr, Run, mp, scope, trcAst, fmt.Sprintf("%v: token at expected pos: %v", ri, rr.Tok))
				}
			} else if mp.IsLess(ps.Pos) {
				// ps.Pos has moved beyond our expected token -- sub-rule has eaten more than expected!
				if rr.Tok.Tok == token.EOS {
					if ps.Trace.On {
						ps.Trace.Out(ps, pr, Run, mp, scope, trcAst, fmt.Sprintf("%v: EOS token consumed by sub-rule: %v", ri, rr.Tok))
					}
				} else {
					ps.Error(mp, fmt.Sprintf("expected token: %v (at rule index: %v) was consumed by prior sub-rule(s)", rr.Tok, ri), pr)
				}
			} else if ri == nr-1 && rr.Tok.Tok == token.EOS {
				ps.ResetNonMatches() // passed this chunk of inputs -- don't need those nonmatches
			} else {
				ps.Error(mp, fmt.Sprintf("token: %v (at rule index: %v) has extra preceeding input inconsistent with grammar", rr.Tok, ri), pr)
				ps.Pos, _ = ps.Src.NextTokenPos(mp) // move to token for more robustness
			}
			if ourAst != nil {
				ourAst.SetTokRegEnd(ps.Pos, ps.Src) // update our end to any tokens that match
			}
			continue
		}
		creg.St = ps.Pos
		creg.Ed = scope.Ed
		if !pr.HasFlag(int(NoToks)) {
			for mi := ri + 1; mi < nr; mi++ {
				if mpos[mi].St != lex.PosZero {
					creg.Ed = mpos[mi].St // only look up to point of next matching token
					break
				}
			}
		}
		if rr.IsToken() { // opt by definition here
			if creg.IsNil() { // no tokens left..
				if ps.Trace.On {
					ps.Trace.Out(ps, pr, Run, creg.St, scope, trcAst, fmt.Sprintf("%v: opt token: %v no more src", ri, rr.Tok))
				}
				continue
			}
			stlx := ps.Src.LexAt(creg.St)
			kt := rr.Tok
			kt.Depth += stlx.Tok.Depth
			pos, ok := ps.FindToken(kt, creg)
			if !ok {
				if ps.Trace.On {
					ps.Trace.Out(ps, pr, NoMatch, creg.St, creg, parAst, fmt.Sprintf("%v token: %v", ri, kt.String()))
				}
				continue
			}
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, Match, pos, creg, parAst, fmt.Sprintf("%v token: %v", ri, kt))
			}
			ps.Pos, _ = ps.Src.NextTokenPos(pos)
			continue
		}

		////////////////////////////////////////////////////
		//  Below here is a Sub-Rule

		if creg.IsNil() { // no tokens left..
			if rr.Opt {
				if ps.Trace.On {
					ps.Trace.Out(ps, pr, Run, creg.St, scope, trcAst, fmt.Sprintf("%v: opt rule: %v no more src", ri, rr.Rule.Name()))
				}
				continue
			}
			ps.Error(creg.St, fmt.Sprintf("missing expected input for: %v", rr.Rule.Name()), pr)
			valid = false
			break // no point in continuing
		}
		useAst := parAst
		if pr.Ast == AnchorAst || anchorFirst || (pr.Ast == SubAst && ri < nr-1) {
			useAst = ourAst
		}
		// NOTE: we can't use anything about the previous match here, because it could have
		// come from a sub-sub-rule and in any case is not where you want to start
		// because is could have been a token in the middle.
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, Run, creg.St, creg, trcAst, fmt.Sprintf("%v: trying rule: %v", ri, rr.Rule.Name()))
		}
		subm := rr.Rule.Parse(ps, pr, useAst, creg, optMap, depth+1)
		if subm == nil {
			if !rr.Opt {
				ps.Error(creg.St, fmt.Sprintf("required element: %v did not match input", rr.Rule.Name()), pr)
				valid = false
				break
			} else {
				if ps.Trace.On {
					ps.Trace.Out(ps, pr, Run, creg.St, creg, trcAst, fmt.Sprintf("%v: optional rule: %v failed", ri, rr.Rule.Name()))
				}
			}
		}
		if !rr.Opt && ourAst != nil {
			ourAst.SetTokRegEnd(ps.Pos, ps.Src) // update our end to include non-optional elements
		}
	}
	if valid {
		pr.DoActs(ps, -1, par, ourAst, parAst)
	}
	return valid
}

// DoRulesRevBinExp reverse version of do rules for binary expression rule with
// one key token in the middle -- we just pay attention to scoping rest of sub-rules
// relative to that, and don't otherwise adjust scope or position.  In particular all
// the position updating taking place in sup-rules is then just ignored and we set the
// position to the end position matched by the "last" rule (which was the first processed)
func (pr *Rule) DoRulesRevBinExp(ps *State, par *Rule, parAst *Ast, scope lex.Reg, mpos Matches, ourAst *Ast, optMap lex.TokenMap, depth int) bool {
	nr := len(pr.Rules)
	valid := true
	creg := scope

	trcAst := parAst
	if ourAst != nil {
		trcAst = ourAst
	}
	tokpos := mpos[1].St
	aftMpos, ok := ps.Src.NextTokenPos(tokpos)
	if !ok {
		ps.Error(tokpos, "premature end of input", pr)
		return false
	}

	epos := scope.Ed
	for i := nr - 1; i >= 0; i-- {
		rr := &pr.Rules[i]
		if i > 1 {
			creg.St = aftMpos // end expr is in region from key token to end of scope
			ps.Pos = creg.St  // only works for a single rule after key token -- sub-rules not necc reverse
			creg.Ed = scope.Ed
		} else if i == 1 {
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, Run, tokpos, scope, trcAst, fmt.Sprintf("%v: key token: %v", i, rr.Tok))
			}
			continue
		} else { // start
			creg.St = scope.St
			ps.Pos = creg.St
			creg.Ed = tokpos
		}
		if rr.IsRule() { // non-key tokens ignored
			if creg.IsNil() { // no tokens left..
				ps.Error(creg.St, fmt.Sprintf("missing expected input for: %v", rr.Rule.Name()), pr)
				valid = false
				continue
			}
			useAst := parAst
			if pr.Ast == AnchorAst {
				useAst = ourAst
			}
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, Run, creg.St, creg, trcAst, fmt.Sprintf("%v: trying rule: %v", i, rr.Rule.Name()))
			}
			subm := rr.Rule.Parse(ps, pr, useAst, creg, optMap, depth+1)
			if subm == nil {
				if !rr.Opt {
					ps.Error(creg.St, fmt.Sprintf("required element: %v did not match input", rr.Rule.Name()), pr)
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

// DoActs performs actions at given point in rule execution (ri = rule index, is -1 at end)
func (pr *Rule) DoActs(ps *State, ri int, par *Rule, ourAst, parAst *Ast) bool {
	if len(pr.Acts) == 0 {
		return false
	}
	// prf := prof.Start("DoActs")
	// defer prf.End()
	valid := true
	for ai := range pr.Acts {
		act := &pr.Acts[ai]
		if act.RunIdx != ri {
			continue
		}
		if !pr.DoAct(ps, act, par, ourAst, parAst) {
			valid = false
		}
	}
	return valid
}

// DoAct performs one action after a rule executes
func (pr *Rule) DoAct(ps *State, act *Act, par *Rule, ourAst, parAst *Ast) bool {
	if act.Act == PushStack {
		ps.Stack.Push(act.Path)
		return true
	} else if act.Act == PopStack {
		ps.Stack.Pop()
		return true
	}

	useAst := ourAst
	if useAst == nil {
		useAst = parAst
	}
	apath := useAst.PathUnique()
	var node ki.Ki
	var adnl []ki.Ki // additional nodes
	ok := false
	if act.Path == "" {
		node = useAst
		ok = true
	} else if andidx := strings.Index(act.Path, "&"); andidx >= 0 {
		pths := strings.Split(act.Path, "&")
		for _, p := range pths {
			findAll := false
			if strings.HasSuffix(p, "...") {
				findAll = true
				p = strings.TrimSuffix(p, "...")
			}
			var nd ki.Ki
			if p[:3] == "../" {
				nd, ok = parAst.FindPathUnique(p[3:])
			} else {
				nd, ok = useAst.FindPathUnique(p)
			}
			if ok {
				if node == nil {
					node = nd
				}
				if findAll {
					pn := nd.Parent()
					for _, pk := range *pn.Children() {
						if pk != nd && pk.Name() == nd.Name() {
							adnl = append(adnl, pk)
						}
					}
				} else if node != nd {
					adnl = append(adnl, nd)
				}
			}
		}
	} else {
		pths := strings.Split(act.Path, "|")
		for _, p := range pths {
			findAll := false
			if strings.HasSuffix(p, "...") {
				findAll = true
				p = strings.TrimSuffix(p, "...")
			}
			if p[:3] == "../" {
				node, ok = parAst.FindPathUnique(p[3:])
			} else {
				node, ok = useAst.FindPathUnique(p)
			}
			if ok {
				if findAll {
					pn := node.Parent()
					for _, pk := range *pn.Children() {
						if pk != node && pk.Name() == node.Name() {
							adnl = append(adnl, pk)
						}
					}
				}
				break
			}
		}
	}
	if node == nil {
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, RunAct, ps.Pos, lex.RegZero, useAst, fmt.Sprintf("Act %v: ERROR: node not found at path(s): %v in node: %v", act.Act, act.Path, apath))
		}
		return false
	}
	ast := node.(*Ast)
	lx := ps.Src.LexAt(ast.TokReg.St)
	useTok := lx.Tok.Tok
	if act.Tok != token.None {
		useTok = act.Tok
	}
	nm := ast.Src
	nms := strings.Split(nm, ",")
	if len(adnl) > 0 {
		for _, pk := range adnl {
			nast := pk.(*Ast)
			if nast != ast {
				nms = append(nms, strings.Split(nast.Src, ",")...)
			}
		}
	}
	for i := range nms {
		nms[i] = strings.TrimSpace(nms[i])
	}
	switch act.Act {
	case ChgToken:
		cp := ast.TokReg.St
		for cp.IsLess(ast.TokReg.Ed) {
			tlx := ps.Src.LexAt(cp)
			act.ChgTok(tlx)
			cp, _ = ps.Src.NextTokenPos(cp)
		}
		if len(adnl) > 0 {
			for _, pk := range adnl {
				nast := pk.(*Ast)
				cp := nast.TokReg.St
				for cp.IsLess(nast.TokReg.Ed) {
					tlx := ps.Src.LexAt(cp)
					act.ChgTok(tlx)
					cp, _ = ps.Src.NextTokenPos(cp)
				}
			}
		}
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, RunAct, ast.TokReg.St, ast.TokReg, ast, fmt.Sprintf("Act: Token set to: %v from path: %v = %v in node: %v", act.Tok, act.Path, nm, apath))
		}
		return false
	case AddSymbol:
		for i := range nms {
			n := nms[i]
			if n == "" || n == "_" { // go special case..
				continue
			}
			sy, has := ps.FindNameScoped(n)
			added := false
			if has {
				sy.Region = ast.SrcReg
				if ps.Trace.On {
					ps.Trace.Out(ps, pr, RunAct, ast.TokReg.St, ast.TokReg, ast, fmt.Sprintf("Act: Add sym already exists: %v from path: %v = %v in node: %v", sy.String(), act.Path, n, apath))
				}
			} else {
				sy = syms.NewSymbol(n, useTok, ps.Src.Filename, ast.SrcReg)
				added = sy.AddScopesStack(ps.Scopes)
				if !added {
					ps.Syms.Add(sy)
				}
			}
			useAst.Syms.Push(sy)
			sy.Ast = useAst.This()
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, RunAct, ast.TokReg.St, ast.TokReg, ast, fmt.Sprintf("Act: Added sym: %v from path: %v = %v in node: %v", sy.String(), act.Path, n, apath))
			}
		}
	case PushScope:
		sy, has := ps.FindNameScoped(nm)
		if !has {
			// tmps should be overwritten automatically?
			sy = syms.NewSymbol(nm, useTok, ps.Src.Filename, lex.RegZero) // zero = tmp
			ps.Syms.Add(sy)
		}
		ps.Scopes.Push(sy)
		useAst.Syms.Push(sy)
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, RunAct, ast.TokReg.St, ast.TokReg, ast, fmt.Sprintf("Act: Pushed Sym: %v from path: %v = %v in node: %v", sy.String(), act.Path, nm, apath))
		}
	case PushNewScope:
		// add plus push
		sy, has := ps.FindNameScoped(nm)
		added := false
		if has {
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, RunAct, ast.TokReg.St, ast.TokReg, ast, fmt.Sprintf("Act: Push New sym already exists: %v from path: %v = %v in node: %v", sy.String(), act.Path, nm, apath))
			}
		} else {
			sy = syms.NewSymbol(nm, useTok, ps.Src.Filename, ast.SrcReg)
			added = sy.AddScopesStack(ps.Scopes)
			if !added {
				ps.Syms.Add(sy)
			}
		}
		ps.Scopes.Push(sy) // key diff from add..
		useAst.Syms.Push(sy)
		sy.Ast = useAst.This()
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, RunAct, ast.TokReg.St, ast.TokReg, ast, fmt.Sprintf("Act: Pushed New Sym: %v from path: %v = %v in node: %v", sy.String(), act.Path, nm, apath))
		}
	case PopScope:
		sy := ps.Scopes.Pop()
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, RunAct, ast.TokReg.St, ast.TokReg, ast, fmt.Sprintf("Act: Popped Sym: %v in node: %v", sy.String(), apath))
		}
	case PopScopeReg:
		sy := ps.Scopes.Pop()
		sy.Region = ast.SrcReg // update source region to final -- select remains initial trigger one
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, RunAct, ast.TokReg.St, ast.TokReg, ast, fmt.Sprintf("Act: Popped Sym: %v in node: %v", sy.String(), apath))
		}
	case AddDetail:
		sy := useAst.Syms.Top()
		if sy != nil {
			if sy.Detail == "" {
				sy.Detail = nm
			} else {
				sy.Detail += " " + nm
			}
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, RunAct, ast.TokReg.St, ast.TokReg, ast, fmt.Sprintf("Act: Added Detail: %v to Sym: %v in node: %v", nm, sy.String(), apath))
			}
		} else {
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, RunAct, ast.TokReg.St, ast.TokReg, ast, fmt.Sprintf("Act: Add Detail: %v ERROR -- symbol not found in node: %v", nm, apath))
			}
		}
	case AddType:
		scp := ps.Scopes.Top()
		if scp == nil {
			ps.Trace.Out(ps, pr, RunAct, ast.TokReg.St, ast.TokReg, ast, fmt.Sprintf("Act: Add Type: %v ERROR -- requires current scope -- none set in node: %v", nm, apath))
			return false
		}
		for i := range nms {
			n := nms[i]
			if n == "" || n == "_" { // go special case..
				continue
			}
			ty := syms.NewType(n, syms.Unknown)
			ty.Ast = useAst.This()
			scp.Types.Add(ty)
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, RunAct, ast.TokReg.St, ast.TokReg, ast, fmt.Sprintf("Act: Added type: %v from path: %v = %v in node: %v", ty.String(), act.Path, n, apath))
			}
		}
	}
	return true
}

///////////////////////////////////////////////////////////////////////
//  Non-parsing functions

// Find looks for rules in the tree that contain given string in Rule or Name fields
func (pr *Rule) Find(find string) []*Rule {
	var res []*Rule
	pr.FuncDownMeFirst(0, pr.This(), func(k ki.Ki, level int, d interface{}) bool {
		pri := k.(*Rule)
		if strings.Contains(pri.Rule, find) || strings.Contains(pri.Nm, find) {
			res = append(res, pri)
		}
		return true
	})
	return res
}

// WriteGrammar outputs the parser rules as a formatted grammar in a BNF-like format
// it is called recursively
func (pr *Rule) WriteGrammar(writer io.Writer, depth int) {
	if pr.IsRoot() {
		for _, k := range pr.Kids {
			pri := k.(*Rule)
			pri.WriteGrammar(writer, depth)
		}
	} else {
		ind := indent.Tabs(depth)
		nmstr := pr.Nm
		if pr.Off {
			nmstr = "// OFF: " + nmstr
		}
		if pr.Desc != "" {
			fmt.Fprintf(writer, "%v// %v %v \n", ind, nmstr, pr.Desc)
		}
		if pr.IsGroup() {
			fmt.Fprintf(writer, "%v%v {\n", ind, nmstr)
			w := tabwriter.NewWriter(writer, 4, 4, 2, ' ', 0)
			for _, k := range pr.Kids {
				pri := k.(*Rule)
				pri.WriteGrammar(w, depth+1)
			}
			w.Flush()
			fmt.Fprintf(writer, "%v}\n", ind)
		} else {
			astr := ""
			switch pr.Ast {
			case AddAst:
				astr = "+Ast"
			case SubAst:
				astr = "_Ast"
			case AnchorAst:
				astr = ">Ast"
			case AnchorFirstAst:
				astr = ">1Ast"
			}
			fmt.Fprintf(writer, "%v%v:\t%v\t%v\n", ind, nmstr, pr.Rule, astr)
			if len(pr.Acts) > 0 {
				fmt.Fprintf(writer, "%vActs:%v\n", ind, pr.Acts.String())
			}
		}
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
