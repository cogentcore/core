// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package parser does the parsing stage after lexing, using a top-down recursive-descent
// (TDRD) strategy, with a special reverse mode to deal with left-associative binary expressions
// which otherwise end up being right-associative for TDRD parsing.
// Higher-level rules provide scope to lower-level ones, with a special EOS end-of-statement
// scope recognized for
package parser

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"cogentcore.org/core/base/indent"
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/parse/syms"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
	"cogentcore.org/core/tree"
)

// Set GUIActive to true if the gui (parseview) is active -- ensures that the
// AST tree is updated when nodes are swapped in reverse mode, and maybe
// other things
var GUIActive = false

// DepthLimit is the infinite recursion prevention cutoff
var DepthLimit = 10000

// parser.Rule operates on the lexically tokenized input, not the raw source.
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
type Rule struct {
	tree.NodeBase

	// disable this rule -- useful for testing and exploration
	Off bool `json:",omitempty"`

	// description / comments about this rule
	Desc string `json:",omitempty"`

	// the rule as a space-separated list of rule names and token(s) -- use single quotes around 'tokens' (using token.Tokens names or symbols). For keywords use 'key:keyword'.  All tokens are matched at the same nesting depth as the start of the scope of this rule, unless they have a +D relative depth value differential before the token.  Use @ prefix for a sub-rule to require that rule to match -- by default explicit tokens are used if available, and then only the first sub-rule failing that.  Use ! by itself to define start of an exclusionary rule -- doesn't match when those rule elements DO match.  Use : prefix for a special group node that matches a single token at start of scope, and then defers to the child rules to perform full match -- this is used for FirstTokenMap when there are multiple versions of a given keyword rule.  Use - prefix for tokens anchored by the end (next token) instead of the previous one -- typically just for token prior to 'EOS' but also a block of tokens that need to go backward in the middle of a sequence to avoid ambiguity can be marked with -
	Rule string

	// if present, this rule only fires if stack has this on it
	StackMatch string `json:",omitempty"`

	// what action should be take for this node when it matches
	AST ASTActs

	// actions to perform based on parsed AST tree data, when this rule is done executing
	Acts Acts `json:",omitempty"`

	// for group-level rules having lots of children and lots of recursiveness, and also of high-frequency, when we first encounter such a rule, make a map of all the tokens in the entire scope, and use that for a first-pass rejection on matching tokens
	OptTokenMap bool `json:",omitempty"`

	// for group-level rules with a number of rules that match based on first tokens / keywords, build map to directly go to that rule -- must also organize all of these rules sequentially from the start -- if no match, goes directly to first non-lookup case
	FirstTokenMap bool `json:",omitempty"`

	// rule elements compiled from Rule string
	Rules RuleList `json:"-" xml:"-"`

	// strategic matching order for matching the rules
	Order []int `edit:"-" json:"-" xml:"-"`

	// map from first tokens / keywords to rules for FirstTokenMap case
	FiTokenMap map[string]*Rule `edit:"-" json:"-" xml:"-" set:"-"`

	// for FirstTokenMap, the start of the else cases not covered by the map
	FiTokenElseIndex int `edit:"-" json:"-" xml:"-" set:"-"`

	// exclusionary key index -- this is the token in Rules that we need to exclude matches for using ExclFwd and ExclRev rules
	ExclKeyIndex int `edit:"-" json:"-" xml:"-" set:"-"`

	// exclusionary forward-search rule elements compiled from Rule string
	ExclFwd RuleList `edit:"-" json:"-" xml:"-" set:"-"`

	// exclusionary reverse-search rule elements compiled from Rule string
	ExclRev RuleList `edit:"-" json:"-" xml:"-" set:"-"`

	// Bool flags:

	// setsScope means that this rule sets its own scope, because it ends with EOS
	setsScope bool

	// reverse means that this rule runs in reverse (starts with - sign) -- for arithmetic
	// binary expressions only: this is needed to produce proper associativity result for
	// mathematical expressions in the recursive descent parser.
	// Only for rules of form: Expr '+' Expr -- two sub-rules with a token operator
	// in the middle.
	reverse bool

	// noTokens means that this rule doesn't have any explicit tokens -- only refers to
	// other rules
	noTokens bool

	// onlyTokens means that this rule only has explicit tokens for matching -- can be
	// optimized
	onlyTokens bool

	// tokenMatchGroup is a group node that also has a single token match, so it can
	// be used in a FirstTokenMap to optimize lookup of rules
	tokenMatchGroup bool
}

// RuleEl is an element of a parsing rule -- either a pointer to another rule or a token
type RuleEl struct {

	// sub-rule for this position -- nil if token
	Rule *Rule

	// token, None if rule
	Token token.KeyToken

	// start increment for matching -- this is the number of non-optional, non-match items between (start | last match) and this item -- increments start region for matching
	StInc int

	// if true, this rule must match for rule to fire -- by default only tokens and, failing that, the first sub-rule is used for matching -- use @ to require a match
	Match bool

	// this rule is optional -- will absorb tokens if they exist -- indicated with ? prefix
	Opt bool

	// match this rule working backward from the next token -- triggered by - (minus) prefix and optimizes cases where there can be a lot of tokens going forward but few going from end -- must be anchored by a terminal EOS or other FromNext elements and is ignored if at the very end
	FromNext bool
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
type Matches []textpos.Region

// StartEnd returns the first and last non-zero positions in the Matches list as a region
func (mm Matches) StartEnd() textpos.Region {
	reg := textpos.RegionZero
	for _, mp := range mm {
		if mp.Start != textpos.PosZero {
			if reg.Start == textpos.PosZero {
				reg.Start = mp.Start
			}
			reg.End = mp.End
		}
	}
	return reg
}

// StartEndExcl returns the first and last non-zero positions in the Matches list as a region
// moves the end to next toke to make it the usual exclusive end pos
func (mm Matches) StartEndExcl(ps *State) textpos.Region {
	reg := mm.StartEnd()
	reg.End, _ = ps.Src.NextTokenPos(reg.End)
	return reg
}

///////////////////////////////////////////////////////////////////////
//  Rule

// IsGroup returns true if this node is a group, else it should have rules
func (pr *Rule) IsGroup() bool {
	return pr.HasChildren()
}

// SetRuleMap is called on the top-level Rule and initializes the RuleMap
func (pr *Rule) SetRuleMap(ps *State) {
	RuleMap = map[string]*Rule{}
	pr.WalkDown(func(k tree.Node) bool {
		pri := k.(*Rule)
		if epr, has := RuleMap[pri.Name]; has {
			ps.Error(textpos.PosZero, fmt.Sprintf("Parser Compile: multiple rules with same name: %v and %v", pri.Path(), epr.Path()), pri)
		} else {
			RuleMap[pri.Name] = pri
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
	pr.WalkDown(func(k tree.Node) bool {
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
		pr.SetProperty("inactive", true)
	} else {
		pr.DeleteProperty("inactive")
	}
	if pr.Rule == "" { // parent
		pr.Rules = nil
		pr.setsScope = false
		return true
	}
	valid := true
	rstr := pr.Rule
	if pr.Rule[0] == '-' {
		rstr = rstr[1:]
		pr.reverse = true
	} else {
		pr.reverse = false
	}
	rs := strings.Split(rstr, " ")
	nr := len(rs)

	pr.Rules = make(RuleList, nr)
	pr.ExclFwd = nil
	pr.ExclRev = nil
	pr.noTokens = false
	pr.onlyTokens = true // default is this..
	pr.setsScope = false
	pr.tokenMatchGroup = false
	pr.Order = nil
	nmatch := 0
	ntok := 0
	curStInc := 0
	eoses := 0
	for ri := range rs {
		rn := strings.TrimSpace(rs[ri])
		if len(rn) == 0 {
			ps.Error(textpos.PosZero, "Compile: Rules has empty string -- make sure there is only one space between rule elements", pr)
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
			pr.tokenMatchGroup = true
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
				rr.Token.Depth = int(td)
			} else if rn[0] == '-' {
				rr.FromNext = true
			}
			tn := rn[tokst+1 : sz-1]
			if len(tn) > 4 && tn[:4] == "key:" {
				rr.Token.Token = token.Keyword
				rr.Token.Key = tn[4:]
			} else {
				if pmt, has := token.OpPunctMap[tn]; has {
					rr.Token.Token = pmt
				} else {
					err := rr.Token.Token.SetString(tn)
					if err != nil {
						ps.Error(textpos.PosZero, fmt.Sprintf("Compile: token convert error: %v", err.Error()), pr)
						valid = false
					}
				}
			}
			if rr.Token.Token == token.EOS {
				eoses++
				if ri == nr-1 {
					rr.StInc = eoses
					pr.setsScope = true
				}
			}
		} else {
			st := 0
			if rn[:2] == "?@" || rn[:2] == "@?" {
				st = 2
				rr.Opt = true
				rr.Match = true
			} else if rn[0] == '?' {
				st = 1
				rr.Opt = true
			} else if rn[0] == '@' {
				st = 1
				rr.Match = true
				pr.onlyTokens = false
				pr.Order = append(pr.Order, ri)
				nmatch++
			} else {
				curStInc++
			}
			rp, ok := RuleMap[rn[st:]]
			if !ok {
				ps.Error(textpos.PosZero, fmt.Sprintf("Compile: refers to rule %v not found", rn), pr)
				valid = false
			} else {
				rr.Rule = rp
			}
		}
	}
	if pr.reverse {
		pr.AST = AnchorAST // must be
	}
	if ntok == 0 && nmatch == 0 {
		pr.Rules[0].Match = true
		pr.Order = append(pr.Order, 0)
		pr.noTokens = true
	} else {
		pr.OptimizeOrder(ps)
	}
	return valid
}

// OptimizeOrder optimizes the order of processing rule elements, including:
// * A block of reversed elements that match from next
func (pr *Rule) OptimizeOrder(ps *State) {
	osz := len(pr.Order)
	if osz == 0 {
		return
	}
	nfmnxt := 0
	fmnSt := -1
	fmnEd := -1
	lastwas := false
	for oi := 0; oi < osz; oi++ {
		ri := pr.Order[oi]
		rr := &pr.Rules[ri]
		if rr.FromNext {
			nfmnxt++
			if fmnSt < 0 {
				fmnSt = oi
			}
			if lastwas {
				fmnEd = oi // end of block
			}
			lastwas = true
		} else {
			lastwas = false
		}
	}
	if nfmnxt > 1 && fmnEd > 0 {
		nword := make([]int, osz)
		for oi := 0; oi < fmnSt; oi++ {
			nword[oi] = pr.Order[oi]
		}
		idx := fmnSt
		for oi := fmnEd - 1; oi >= fmnSt; oi-- {
			nword[idx] = pr.Order[oi]
			idx++
		}
		for oi := fmnEd; oi < osz; oi++ {
			nword[oi] = pr.Order[oi]
		}
		pr.Order = nword
	}
}

// CompileTokMap compiles first token map
func (pr *Rule) CompileTokMap(ps *State) bool {
	valid := true
	pr.FiTokenMap = make(map[string]*Rule, len(pr.Children))
	pr.FiTokenElseIndex = len(pr.Children)
	for i, kpri := range pr.Children {
		kpr := kpri.(*Rule)
		if len(kpr.Rules) == 0 || !kpr.Rules[0].IsToken() {
			pr.FiTokenElseIndex = i
			break
		}
		fr := kpr.Rules[0]
		skey := fr.Token.StringKey()
		if _, has := pr.FiTokenMap[skey]; has {
			ps.Error(textpos.PosZero, fmt.Sprintf("CompileFirstTokenMap: multiple rules have the same first token: %v -- must be unique -- use a :'tok' group to match that first token and put all the sub-rules as children of that node", fr.Token), pr)
			pr.FiTokenElseIndex = 0
			valid = false
		} else {
			pr.FiTokenMap[skey] = kpr
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
		ktok = rr.Token
		ktoki = ri
		break
	}

	if ktoki < 0 {
		ps.Error(textpos.PosZero, "CompileExcl: no token found for matching exclusion rules", pr)
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
			rr.Token.Depth = int(td)
		}
		tn := rn[tokst+1 : sz-1]
		if len(tn) > 4 && tn[:4] == "key:" {
			rr.Token.Token = token.Keyword
			rr.Token.Key = tn[4:]
		} else {
			if pmt, has := token.OpPunctMap[tn]; has {
				rr.Token.Token = pmt
			} else {
				err := rr.Token.Token.SetString(tn)
				if err != nil {
					ps.Error(textpos.PosZero, fmt.Sprintf("CompileExcl: token convert error: %v", err.Error()), pr)
					valid = false
				}
			}
		}
		if rr.Token.Equal(ktok) {
			ki = ri
		}
	}
	if ki < 0 {
		ps.Error(textpos.PosZero, fmt.Sprintf("CompileExcl: key token: %v not found in exclusion rule", ktok), pr)
		valid = false
		return valid
	}
	pr.ExclKeyIndex = ktoki
	pr.ExclFwd = pr.ExclRev[ki+1-rist:]
	pr.ExclRev = pr.ExclRev[:ki-rist]
	return valid
}

// Validate checks for any errors in the rules and issues warnings,
// returns true if valid (no err) and false if invalid (errs)
func (pr *Rule) Validate(ps *State) bool {
	valid := true

	// do this here so everything else is compiled
	if len(pr.Rules) == 0 && pr.FirstTokenMap {
		pr.CompileTokMap(ps)
	}

	if len(pr.Rules) == 0 && !pr.HasChildren() && !tree.IsRoot(pr) {
		ps.Error(textpos.PosZero, "Validate: rule has no rules and no children", pr)
		valid = false
	}
	if !pr.tokenMatchGroup && len(pr.Rules) > 0 && pr.HasChildren() {
		ps.Error(textpos.PosZero, "Validate: rule has both rules and children -- should be either-or", pr)
		valid = false
	}
	if pr.reverse {
		if len(pr.Rules) != 3 {
			ps.Error(textpos.PosZero, "Validate: a Reverse (-) rule must have 3 children -- for binary operator expressions only", pr)
			valid = false
		} else {
			if !pr.Rules[1].IsToken() {
				ps.Error(textpos.PosZero, "Validate: a Reverse (-) rule must have a token to be recognized in the middle of two rules -- for binary operator expressions only", pr)
			}
		}
	}

	if len(pr.Rules) > 0 {
		if pr.Rules[0].IsRule() && (pr.Rules[0].Rule == pr || pr.ParentLevel(pr.Rules[0].Rule) >= 0) { // left recursive
			if pr.Rules[0].Match {
				ps.Error(textpos.PosZero, fmt.Sprintf("Validate: rule refers to itself recursively in first sub-rule: %v and that sub-rule is marked as a Match -- this is infinite recursion and is not allowed!  Must use distinctive tokens in rule to match this rule, and then left-recursive elements will be filled in when the rule runs, but they cannot be used for matching rule.", pr.Rules[0].Rule.Name), pr)
				valid = false
			}
			ntok := 0
			for _, rr := range pr.Rules {
				if rr.IsToken() {
					ntok++
				}
			}
			if ntok == 0 {
				ps.Error(textpos.PosZero, fmt.Sprintf("Validate: rule refers to itself recursively in first sub-rule: %v, and does not have any tokens in the rule -- MUST promote tokens to this rule to disambiguate match, otherwise will just do infinite recursion!", pr.Rules[0].Rule.Name), pr)
				valid = false
			}
		}
	}

	// now we iterate over our kids
	for _, kpri := range pr.Children {
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
	kpr := pr.Children[0].(*Rule) // first rule is special set of valid top-level matches
	var parAST *AST
	scope := textpos.Region{Start: ps.Pos}
	if ps.AST.HasChildren() {
		parAST = ps.AST.ChildAST(0)
	} else {
		parAST = NewAST(ps.AST)
		parAST.SetName(kpr.Name)
		ok := false
		scope.Start, ok = ps.Src.ValidTokenPos(scope.Start)
		if !ok {
			ps.GotoEof()
			return nil
		}
		ps.Pos = scope.Start
	}
	didErr := false
	for {
		cpos := ps.Pos
		mrule := kpr.Parse(ps, pr, parAST, scope, nil, 0)
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
// parent is the parent rule that we're being called from.
// parAST is the current ast node that we add to.
// scope is the region to search within, defined by parent or EOS if we have a terminal
// one
func (pr *Rule) Parse(ps *State, parent *Rule, parAST *AST, scope textpos.Region, optMap lexer.TokenMap, depth int) *Rule {
	if pr.Off {
		return nil
	}

	if depth >= DepthLimit {
		ps.Error(scope.Start, "depth limit exceeded -- parser rules error -- look for recursive cases", pr)
		return nil
	}

	nr := len(pr.Rules)
	if !pr.tokenMatchGroup && nr > 0 {
		return pr.ParseRules(ps, parent, parAST, scope, optMap, depth)
	}

	if optMap == nil && pr.OptTokenMap {
		optMap = ps.Src.TokenMapReg(scope)
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, Run, scope.Start, scope, parAST, fmt.Sprintf("made optmap of size: %d", len(optMap)))
		}
	}

	// pure group types just iterate over kids
	for _, kpri := range pr.Children {
		kpr := kpri.(*Rule)
		if mrule := kpr.Parse(ps, pr, parAST, scope, optMap, depth+1); mrule != nil {
			return mrule
		}
	}
	return nil
}

// ParseRules parses rules and returns this rule if it matches, nil if not
func (pr *Rule) ParseRules(ps *State, parent *Rule, parAST *AST, scope textpos.Region, optMap lexer.TokenMap, depth int) *Rule {
	ok := false
	if pr.setsScope {
		scope, ok = pr.Scope(ps, parAST, scope)
		if !ok {
			return nil
		}
	} else if GUIActive {
		if scope == textpos.RegionZero {
			ps.Error(scope.Start, "scope is empty and no EOS in rule -- invalid rules -- starting rules must all have EOS", pr)
			return nil
		}
	}
	match, nscope, mpos := pr.Match(ps, parAST, scope, 0, optMap)
	if !match {
		return nil
	}

	rparent := parent.Parent.(*Rule)

	if parent.AST != NoAST && parent.IsGroup() {
		if parAST.Name != parent.Name {
			mreg := mpos.StartEndExcl(ps)
			newAST := ps.AddAST(parAST, parent.Name, mreg)
			if parent.AST == AnchorAST {
				parAST = newAST
			}
		}
	} else if parent.IsGroup() && rparent.AST != NoAST && rparent.IsGroup() { // two-level group...
		if parAST.Name != rparent.Name {
			mreg := mpos.StartEndExcl(ps)
			newAST := ps.AddAST(parAST, rparent.Name, mreg)
			if rparent.AST == AnchorAST {
				parAST = newAST
			}
		}
	}
	valid := pr.DoRules(ps, parent, parAST, nscope, mpos, optMap, depth) // returns validity but we don't care once matched..
	if !valid {
		return nil
	}
	return pr
}

// Scope finds the potential scope region for looking for tokens -- either from
// EOS position or State ScopeStack pushed from parents.
// Returns new scope and false if no valid scope found.
func (pr *Rule) Scope(ps *State, parAST *AST, scope textpos.Region) (textpos.Region, bool) {
	// prf := profile.Start("Scope")
	// defer prf.End()

	nscope := scope
	creg := scope
	lr := pr.Rules.Last()
	for ei := 0; ei < lr.StInc; ei++ {
		stlx := ps.Src.LexAt(creg.Start)
		ep, ok := ps.Src.NextEos(creg.Start, stlx.Token.Depth)
		if !ok {
			// ps.Error(creg.Start, "could not find EOS at target nesting depth -- parens / bracket / brace mismatch?", pr)
			return nscope, false
		}
		if scope.End != textpos.PosZero && lr.Opt && scope.End.IsLess(ep) {
			// optional tokens can't take us out of scope
			return scope, true
		}
		if ei == lr.StInc-1 {
			nscope.End = ep
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, SubMatch, nscope.Start, nscope, parAST, fmt.Sprintf("from EOS: starting scope: %v new scope: %v end pos: %v depth: %v", scope, nscope, ep, stlx.Token.Depth))
			}
		} else {
			creg.Start, ok = ps.Src.NextTokenPos(ep) // advance
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
func (pr *Rule) Match(ps *State, parAST *AST, scope textpos.Region, depth int, optMap lexer.TokenMap) (bool, textpos.Region, Matches) {
	if pr.Off {
		return false, scope, nil
	}

	if depth > DepthLimit {
		ps.Error(scope.Start, "depth limit exceeded -- parser rules error -- look for recursive cases", pr)
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

	// mprf := profile.Start("Match")
	// defer mprf.End()
	// Note: uncomment the following to see which rules are taking the most
	// time -- very helpful for focusing effort on optimizing those rules.
	// prf := profile.Start(pr.Nm)
	// defer prf.End()

	nr := len(pr.Rules)
	if pr.tokenMatchGroup || nr == 0 { // Group
		return pr.MatchGroup(ps, parAST, scope, depth, optMap)
	}

	// prf := profile.Start("IsMatch")
	if mst, match := ps.IsMatch(pr, scope); match {
		// prf.End()
		return true, scope, mst.Regs
	}
	// prf.End()

	var mpos Matches
	match := false

	if pr.noTokens {
		match, mpos = pr.MatchNoToks(ps, parAST, scope, depth, optMap)
	} else if pr.onlyTokens {
		match, mpos = pr.MatchOnlyToks(ps, parAST, scope, depth, optMap)
	} else {
		match, mpos = pr.MatchMixed(ps, parAST, scope, depth, optMap)
	}
	if !match {
		ps.AddNonMatch(scope, pr)
		return false, scope, nil
	}

	if len(pr.ExclFwd) > 0 || len(pr.ExclRev) > 0 {
		ktpos := mpos[pr.ExclKeyIndex]
		if pr.MatchExclude(ps, scope, ktpos, depth, optMap) {
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, NoMatch, ktpos.Start, scope, parAST, "Exclude criteria matched")
			}
			ps.AddNonMatch(scope, pr)
			return false, scope, nil
		}
	}

	mreg := mpos.StartEnd()
	ps.AddMatch(pr, scope, mpos)
	if ps.Trace.On {
		ps.Trace.Out(ps, pr, Match, mreg.Start, scope, parAST, fmt.Sprintf("Full Match reg: %v", mreg))
	}
	return true, scope, mpos
}

// MatchOnlyToks matches rules having only tokens
func (pr *Rule) MatchOnlyToks(ps *State, parAST *AST, scope textpos.Region, depth int, optMap lexer.TokenMap) (bool, Matches) {
	nr := len(pr.Rules)

	var mpos Matches

	scstlx := ps.Src.LexAt(scope.Start) // scope starting lex
	scstDepth := scstlx.Token.Depth

	creg := scope
	osz := len(pr.Order)
	for oi := 0; oi < osz; oi++ {
		ri := pr.Order[oi]
		rr := &pr.Rules[ri]
		kt := rr.Token
		if optMap != nil && !optMap.Has(kt.Token) { // not even a possibility
			return false, nil
		}
		if rr.FromNext {
			if mpos == nil {
				mpos = make(Matches, nr) // make on demand -- cuts out a lot of allocations!
			}
			mpos[nr-1] = textpos.Region{Start: scope.End, End: scope.End}
		}
		kt.Depth += scstDepth // always use starting scope depth
		match, tpos := pr.MatchToken(ps, rr, ri, kt, &creg, mpos, parAST, scope, depth, optMap)
		if !match {
			if ps.Trace.On {
				if tpos != textpos.PosZero {
					tlx := ps.Src.LexAt(tpos)
					ps.Trace.Out(ps, pr, NoMatch, creg.Start, creg, parAST, fmt.Sprintf("%v token: %v, was: %v", ri, kt.String(), tlx.String()))
				} else {
					ps.Trace.Out(ps, pr, NoMatch, creg.Start, creg, parAST, fmt.Sprintf("%v token: %v, nil region", ri, kt.String()))
				}
			}
			return false, nil
		}
		if mpos == nil {
			mpos = make(Matches, nr) // make on demand -- cuts out a lot of allocations!
		}
		mpos[ri] = textpos.Region{Start: tpos, End: tpos}
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, SubMatch, creg.Start, creg, parAST, fmt.Sprintf("%v token: %v", ri, kt.String()))
		}
	}

	return true, mpos
}

// MatchToken matches one token sub-rule -- returns true for match and
// false if no match -- and the position where it was / should have been
func (pr *Rule) MatchToken(ps *State, rr *RuleEl, ri int, kt token.KeyToken, creg *textpos.Region, mpos Matches, parAST *AST, scope textpos.Region, depth int, optMap lexer.TokenMap) (bool, textpos.Pos) {
	nr := len(pr.Rules)
	ok := false
	matchst := false // match start of creg
	matched := false // match end of creg
	var tpos textpos.Pos
	if ri == 0 {
		matchst = true
	} else if mpos != nil {
		lpos := mpos[ri-1].End
		if lpos != textpos.PosZero { // previous has matched
			matchst = true
		} else if ri < nr-1 && rr.FromNext {
			lpos := mpos[ri+1].Start
			if lpos != textpos.PosZero { // previous has matched
				creg.End, _ = ps.Src.PrevTokenPos(lpos)
				matched = true
			}
		}
	}
	for stinc := 0; stinc < rr.StInc; stinc++ {
		creg.Start, _ = ps.Src.NextTokenPos(creg.Start)
	}
	if ri == nr-1 && rr.Token.Token == token.EOS {
		return true, scope.End
	}
	if creg.IsNil() && !matched {
		return false, tpos
	}

	if matchst { // start token must be right here
		if !ps.MatchToken(kt, creg.Start) {
			return false, creg.Start
		}
		tpos = creg.Start
	} else if matched {
		if !ps.MatchToken(kt, creg.End) {
			return false, creg.End
		}
		tpos = creg.End
	} else {
		// prf := profile.Start("FindToken")
		if pr.reverse {
			tpos, ok = ps.FindTokenReverse(kt, *creg)
		} else {
			tpos, ok = ps.FindToken(kt, *creg)
		}
		// prf.End()
		if !ok {
			return false, tpos
		}
	}
	creg.Start, _ = ps.Src.NextTokenPos(tpos) // always ratchet up
	return true, tpos
}

// MatchMixed matches mixed tokens and non-tokens
func (pr *Rule) MatchMixed(ps *State, parAST *AST, scope textpos.Region, depth int, optMap lexer.TokenMap) (bool, Matches) {
	nr := len(pr.Rules)
	var mpos Matches

	scstlx := ps.Src.LexAt(scope.Start) // scope starting lex
	scstDepth := scstlx.Token.Depth

	creg := scope
	osz := len(pr.Order)

	// 	first pass filter on tokens
	if optMap != nil {
		for oi := 0; oi < osz; oi++ {
			ri := pr.Order[oi]
			rr := &pr.Rules[ri]
			if rr.IsToken() {
				kt := rr.Token
				if !optMap.Has(kt.Token) { // not even a possibility
					return false, nil
				}
			}
		}
	}

	for oi := 0; oi < osz; oi++ {
		ri := pr.Order[oi]
		rr := &pr.Rules[ri]

		/////////////////////////////////////////////
		// Token
		if rr.IsToken() {
			kt := rr.Token
			if rr.FromNext {
				if mpos == nil {
					mpos = make(Matches, nr) // make on demand -- cuts out a lot of allocations!
				}
				mpos[nr-1] = textpos.Region{Start: scope.End, End: scope.End}
			}
			kt.Depth += scstDepth // always use starting scope depth
			match, tpos := pr.MatchToken(ps, rr, ri, kt, &creg, mpos, parAST, scope, depth, optMap)
			if !match {
				if ps.Trace.On {
					if tpos != textpos.PosZero {
						tlx := ps.Src.LexAt(tpos)
						ps.Trace.Out(ps, pr, NoMatch, creg.Start, creg, parAST, fmt.Sprintf("%v token: %v, was: %v", ri, kt.String(), tlx.String()))
					} else {
						ps.Trace.Out(ps, pr, NoMatch, creg.Start, creg, parAST, fmt.Sprintf("%v token: %v, nil region", ri, kt.String()))
					}
				}
				return false, nil
			}
			if mpos == nil {
				mpos = make(Matches, nr) // make on demand -- cuts out a lot of allocations!
			}
			mpos[ri] = textpos.Region{Start: tpos, End: tpos}
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, SubMatch, creg.Start, creg, parAST, fmt.Sprintf("%v token: %v", ri, kt.String()))
			}
			continue
		}

		//////////////////////////////////////////////
		// Sub-Rule

		if creg.IsNil() {
			ps.Trace.Out(ps, pr, NoMatch, creg.Start, creg, parAST, fmt.Sprintf("%v sub-rule: %v, nil region", ri, rr.Rule.Name))
			return false, nil
		}

		// first, limit region to same depth or greater as start of region -- prevents
		// overflow beyond natural boundaries
		stlx := ps.Src.LexAt(creg.Start) // scope starting lex
		cp, _ := ps.Src.NextTokenPos(creg.Start)
		stdp := stlx.Token.Depth
		for cp.IsLess(creg.End) {
			lx := ps.Src.LexAt(cp)
			if lx.Token.Depth < stdp {
				creg.End = cp
				break
			}
			cp, _ = ps.Src.NextTokenPos(cp)
		}

		if ps.Trace.On {
			ps.Trace.Out(ps, pr, SubMatch, creg.Start, creg, parAST, fmt.Sprintf("%v trying sub-rule: %v", ri, rr.Rule.Name))
		}
		match, _, smpos := rr.Rule.Match(ps, parAST, creg, depth+1, optMap)
		if !match {
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, NoMatch, creg.Start, creg, parAST, fmt.Sprintf("%v sub-rule: %v", ri, rr.Rule.Name))
			}
			return false, nil
		}
		creg.End = scope.End // back to full scope
		// look through smpos for last valid position -- use that as last match pos
		mreg := smpos.StartEnd()
		lmnpos, ok := ps.Src.NextTokenPos(mreg.End)
		if !ok && !(ri == nr-1 || (ri == nr-2 && pr.setsScope)) {
			// if at end, or ends in EOS, then ok..
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, NoMatch, creg.Start, creg, parAST, fmt.Sprintf("%v sub-rule: %v -- not at end and no tokens left", ri, rr.Rule.Name))
			}
			return false, nil
		}
		if mpos == nil {
			mpos = make(Matches, nr) // make on demand -- cuts out a lot of allocations!
		}
		mpos[ri] = mreg
		creg.Start = lmnpos
		if ps.Trace.On {
			msreg := mreg
			msreg.End = lmnpos
			ps.Trace.Out(ps, pr, SubMatch, mreg.Start, msreg, parAST, fmt.Sprintf("%v rule: %v reg: %v", ri, rr.Rule.Name, msreg))
		}
	}

	return true, mpos
}

// MatchNoToks matches NoToks case -- just does single sub-rule match
func (pr *Rule) MatchNoToks(ps *State, parAST *AST, scope textpos.Region, depth int, optMap lexer.TokenMap) (bool, Matches) {
	creg := scope
	ri := 0
	rr := &pr.Rules[0]
	if ps.Trace.On {
		ps.Trace.Out(ps, pr, SubMatch, creg.Start, creg, parAST, fmt.Sprintf("%v trying sub-rule: %v", ri, rr.Rule.Name))
	}
	match, _, smpos := rr.Rule.Match(ps, parAST, creg, depth+1, optMap)
	if !match {
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, NoMatch, creg.Start, creg, parAST, fmt.Sprintf("%v sub-rule: %v", ri, rr.Rule.Name))
		}
		return false, nil
	}
	if ps.Trace.On {
		mreg := smpos.StartEnd() // todo: should this include creg start instead?
		ps.Trace.Out(ps, pr, SubMatch, mreg.Start, mreg, parAST, fmt.Sprintf("%v rule: %v reg: %v", ri, rr.Rule.Name, mreg))
	}
	return true, smpos
}

// MatchGroup does matching for Group rules
func (pr *Rule) MatchGroup(ps *State, parAST *AST, scope textpos.Region, depth int, optMap lexer.TokenMap) (bool, textpos.Region, Matches) {
	// prf := profile.Start("SubMatch")
	if mst, match := ps.IsMatch(pr, scope); match {
		// 	prf.End()
		return true, scope, mst.Regs
	}
	// prf.End()
	sti := 0
	nk := len(pr.Children)
	if pr.FirstTokenMap {
		stlx := ps.Src.LexAt(scope.Start)
		if kpr, has := pr.FiTokenMap[stlx.Token.StringKey()]; has {
			match, nscope, mpos := kpr.Match(ps, parAST, scope, depth+1, optMap)
			if match {
				if ps.Trace.On {
					ps.Trace.Out(ps, pr, SubMatch, scope.Start, scope, parAST, fmt.Sprintf("first token group child: %v", kpr.Name))
				}
				ps.AddMatch(pr, scope, mpos)
				return true, nscope, mpos
			}
		}
		sti = pr.FiTokenElseIndex
	}

	for i := sti; i < nk; i++ {
		kpri := pr.Children[i]
		kpr := kpri.(*Rule)
		match, nscope, mpos := kpr.Match(ps, parAST, scope, depth+1, optMap)
		if match {
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, SubMatch, scope.Start, scope, parAST, fmt.Sprintf("group child: %v", kpr.Name))
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
func (pr *Rule) MatchExclude(ps *State, scope textpos.Region, ktpos textpos.Region, depth int, optMap lexer.TokenMap) bool {
	nf := len(pr.ExclFwd)
	nr := len(pr.ExclRev)
	scstlx := ps.Src.LexAt(scope.Start) // scope starting lex
	scstDepth := scstlx.Token.Depth
	if nf > 0 {
		cp, ok := ps.Src.NextTokenPos(ktpos.Start)
		if !ok {
			return false
		}
		prevAny := false
		for ri := 0; ri < nf; ri++ {
			rr := pr.ExclFwd[ri]
			kt := rr.Token
			kt.Depth += scstDepth // always use starting scope depth
			if kt.Token == token.None {
				prevAny = true // wild card
				continue
			}
			if prevAny {
				creg := scope
				creg.Start = cp
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
					if lx.Token.Depth != kt.Depth {
						break
					}
					// ok, keep going -- no info..
				}
			}
			cp, ok = ps.Src.NextTokenPos(cp)
			if !ok && ri < nf-1 {
				return false
			}
			if scope.End == cp || scope.End.IsLess(cp) { // out of scope -- if non-opt left, nomatch
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
		cp, ok := ps.Src.PrevTokenPos(ktpos.Start)
		if !ok {
			return false
		}
		prevAny := false
		for ri := nr - 1; ri >= 0; ri-- {
			rr := pr.ExclRev[ri]
			kt := rr.Token
			kt.Depth += scstDepth // always use starting scope depth
			if kt.Token == token.None {
				prevAny = true // wild card
				continue
			}
			if prevAny {
				creg := scope
				creg.End = cp
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
					if lx.Token.Depth != kt.Depth {
						break
					}
					// ok, keep going -- no info..
				}
			}
			cp, ok = ps.Src.PrevTokenPos(cp)
			if !ok && ri > 0 {
				return false
			}
			if cp.IsLess(scope.Start) {
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
func (pr *Rule) DoRules(ps *State, parent *Rule, parentAST *AST, scope textpos.Region, mpos Matches, optMap lexer.TokenMap, depth int) bool {
	trcAST := parentAST
	var ourAST *AST
	anchorFirst := (pr.AST == AnchorFirstAST && parentAST.Name != pr.Name)

	if pr.AST != NoAST {
		// prf := profile.Start("AddAST")
		ourAST = ps.AddAST(parentAST, pr.Name, scope)
		// prf.End()
		trcAST = ourAST
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, Run, scope.Start, scope, trcAST, fmt.Sprintf("running with new ast: %v", trcAST.Path()))
		}
	} else {
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, Run, scope.Start, scope, trcAST, fmt.Sprintf("running with parent ast: %v", trcAST.Path()))
		}
	}

	if pr.reverse {
		return pr.DoRulesRevBinExp(ps, parent, parentAST, scope, mpos, ourAST, optMap, depth)
	}

	nr := len(pr.Rules)
	valid := true
	creg := scope
	for ri := 0; ri < nr; ri++ {
		pr.DoActs(ps, ri, parent, ourAST, parentAST)
		rr := &pr.Rules[ri]
		if rr.IsToken() && !rr.Opt {
			mp := mpos[ri].Start
			if mp == ps.Pos {
				ps.Pos, _ = ps.Src.NextTokenPos(ps.Pos) // already matched -- move past
				if ps.Trace.On {
					ps.Trace.Out(ps, pr, Run, mp, scope, trcAST, fmt.Sprintf("%v: token at expected pos: %v", ri, rr.Token))
				}
			} else if mp.IsLess(ps.Pos) {
				// ps.Pos has moved beyond our expected token -- sub-rule has eaten more than expected!
				if rr.Token.Token == token.EOS {
					if ps.Trace.On {
						ps.Trace.Out(ps, pr, Run, mp, scope, trcAST, fmt.Sprintf("%v: EOS token consumed by sub-rule: %v", ri, rr.Token))
					}
				} else {
					ps.Error(mp, fmt.Sprintf("expected token: %v (at rule index: %v) was consumed by prior sub-rule(s)", rr.Token, ri), pr)
				}
			} else if ri == nr-1 && rr.Token.Token == token.EOS {
				ps.ResetNonMatches() // passed this chunk of inputs -- don't need those nonmatches
			} else {
				ps.Error(mp, fmt.Sprintf("token: %v (at rule index: %v) has extra preceding input inconsistent with grammar", rr.Token, ri), pr)
				ps.Pos, _ = ps.Src.NextTokenPos(mp) // move to token for more robustness
			}
			if ourAST != nil {
				ourAST.SetTokRegEnd(ps.Pos, ps.Src) // update our end to any tokens that match
			}
			continue
		}
		creg.Start = ps.Pos
		creg.End = scope.End
		if !pr.noTokens {
			for mi := ri + 1; mi < nr; mi++ {
				if mpos[mi].Start != textpos.PosZero {
					creg.End = mpos[mi].Start // only look up to point of next matching token
					break
				}
			}
		}
		if rr.IsToken() { // opt by definition here
			if creg.IsNil() { // no tokens left..
				if ps.Trace.On {
					ps.Trace.Out(ps, pr, Run, creg.Start, scope, trcAST, fmt.Sprintf("%v: opt token: %v no more src", ri, rr.Token))
				}
				continue
			}
			stlx := ps.Src.LexAt(creg.Start)
			kt := rr.Token
			kt.Depth += stlx.Token.Depth
			pos, ok := ps.FindToken(kt, creg)
			if !ok {
				if ps.Trace.On {
					ps.Trace.Out(ps, pr, NoMatch, creg.Start, creg, parentAST, fmt.Sprintf("%v token: %v", ri, kt.String()))
				}
				continue
			}
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, Match, pos, creg, parentAST, fmt.Sprintf("%v token: %v", ri, kt))
			}
			ps.Pos, _ = ps.Src.NextTokenPos(pos)
			continue
		}

		////////////////////////////////////////////////////
		//  Below here is a Sub-Rule

		if creg.IsNil() { // no tokens left..
			if rr.Opt {
				if ps.Trace.On {
					ps.Trace.Out(ps, pr, Run, creg.Start, scope, trcAST, fmt.Sprintf("%v: opt rule: %v no more src", ri, rr.Rule.Name))
				}
				continue
			}
			ps.Error(creg.Start, fmt.Sprintf("missing expected input for: %v", rr.Rule.Name), pr)
			valid = false
			break // no point in continuing
		}
		useAST := parentAST
		if pr.AST == AnchorAST || anchorFirst || (pr.AST == SubAST && ri < nr-1) {
			useAST = ourAST
		}
		// NOTE: we can't use anything about the previous match here, because it could have
		// come from a sub-sub-rule and in any case is not where you want to start
		// because is could have been a token in the middle.
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, Run, creg.Start, creg, trcAST, fmt.Sprintf("%v: trying rule: %v", ri, rr.Rule.Name))
		}
		subm := rr.Rule.Parse(ps, pr, useAST, creg, optMap, depth+1)
		if subm == nil {
			if !rr.Opt {
				ps.Error(creg.Start, fmt.Sprintf("required element: %v did not match input", rr.Rule.Name), pr)
				valid = false
				break
			}
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, Run, creg.Start, creg, trcAST, fmt.Sprintf("%v: optional rule: %v failed", ri, rr.Rule.Name))
			}
		}
		if !rr.Opt && ourAST != nil {
			ourAST.SetTokRegEnd(ps.Pos, ps.Src) // update our end to include non-optional elements
		}
	}
	if valid {
		pr.DoActs(ps, -1, parent, ourAST, parentAST)
	}
	return valid
}

// DoRulesRevBinExp reverse version of do rules for binary expression rule with
// one key token in the middle -- we just pay attention to scoping rest of sub-rules
// relative to that, and don't otherwise adjust scope or position.  In particular all
// the position updating taking place in sup-rules is then just ignored and we set the
// position to the end position matched by the "last" rule (which was the first processed)
func (pr *Rule) DoRulesRevBinExp(ps *State, parent *Rule, parentAST *AST, scope textpos.Region, mpos Matches, ourAST *AST, optMap lexer.TokenMap, depth int) bool {
	nr := len(pr.Rules)
	valid := true
	creg := scope

	trcAST := parentAST
	if ourAST != nil {
		trcAST = ourAST
	}
	tokpos := mpos[1].Start
	aftMpos, ok := ps.Src.NextTokenPos(tokpos)
	if !ok {
		ps.Error(tokpos, "premature end of input", pr)
		return false
	}

	epos := scope.End
	for i := nr - 1; i >= 0; i-- {
		rr := &pr.Rules[i]
		if i > 1 {
			creg.Start = aftMpos // end expr is in region from key token to end of scope
			ps.Pos = creg.Start  // only works for a single rule after key token -- sub-rules not necc reverse
			creg.End = scope.End
		} else if i == 1 {
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, Run, tokpos, scope, trcAST, fmt.Sprintf("%v: key token: %v", i, rr.Token))
			}
			continue
		} else { // start
			creg.Start = scope.Start
			ps.Pos = creg.Start
			creg.End = tokpos
		}
		if rr.IsRule() { // non-key tokens ignored
			if creg.IsNil() { // no tokens left..
				ps.Error(creg.Start, fmt.Sprintf("missing expected input for: %v", rr.Rule.Name), pr)
				valid = false
				continue
			}
			useAST := parentAST
			if pr.AST == AnchorAST {
				useAST = ourAST
			}
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, Run, creg.Start, creg, trcAST, fmt.Sprintf("%v: trying rule: %v", i, rr.Rule.Name))
			}
			subm := rr.Rule.Parse(ps, pr, useAST, creg, optMap, depth+1)
			if subm == nil {
				if !rr.Opt {
					ps.Error(creg.Start, fmt.Sprintf("required element: %v did not match input", rr.Rule.Name), pr)
					valid = false
				}
			}
		}
	}
	// our AST is now backwards -- need to swap them
	if len(ourAST.Children) == 2 {
		slicesx.Swap(ourAST.Children, 0, 1)
		// if GuiActive {
		// we have a very strange situation here: the tree of the AST will typically
		// have two children, named identically (e.g., Expr, Expr) and it will not update
		// after our swap.  If we could use UniqNames then it would be ok, but that doesn't
		// work for tree names.. really need an option that supports uniqname AND reg names
		// https://cogentcore.org/core/ki/issues/2
		// ourAST.NewChild(ASTType, "Dummy")
		// ourAST.DeleteChildAt(2, true)
		// }
	}

	ps.Pos = epos
	return valid
}

// DoActs performs actions at given point in rule execution (ri = rule index, is -1 at end)
func (pr *Rule) DoActs(ps *State, ri int, parent *Rule, ourAST, parentAST *AST) bool {
	if len(pr.Acts) == 0 {
		return false
	}
	// prf := profile.Start("DoActs")
	// defer prf.End()
	valid := true
	for ai := range pr.Acts {
		act := &pr.Acts[ai]
		if act.RunIndex != ri {
			continue
		}
		if !pr.DoAct(ps, act, parent, ourAST, parentAST) {
			valid = false
		}
	}
	return valid
}

// DoAct performs one action after a rule executes
func (pr *Rule) DoAct(ps *State, act *Act, parent *Rule, ourAST, parAST *AST) bool {
	if act.Act == PushStack {
		ps.Stack.Push(act.Path)
		return true
	} else if act.Act == PopStack {
		ps.Stack.Pop()
		return true
	}

	useAST := ourAST
	if useAST == nil {
		useAST = parAST
	}
	apath := useAST.Path()
	var node tree.Node
	var adnl []tree.Node // additional nodes
	if act.Path == "" {
		node = useAST
	} else if andidx := strings.Index(act.Path, "&"); andidx >= 0 {
		pths := strings.Split(act.Path, "&")
		for _, p := range pths {
			findAll := false
			if strings.HasSuffix(p, "...") {
				findAll = true
				p = strings.TrimSuffix(p, "...")
			}
			var nd tree.Node
			if p[:3] == "../" {
				nd = parAST.FindPath(p[3:])
			} else {
				nd = useAST.FindPath(p)
			}
			if nd != nil {
				if node == nil {
					node = nd
				}
				if findAll {
					pn := nd.AsTree().Parent
					for _, pk := range pn.AsTree().Children {
						if pk != nd && pk.AsTree().Name == nd.AsTree().Name {
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
				node = parAST.FindPath(p[3:])
			} else {
				node = useAST.FindPath(p)
			}
			if node != nil {
				if findAll {
					pn := node.AsTree().Parent
					for _, pk := range pn.AsTree().Children {
						if pk != node && pk.AsTree().Name == node.AsTree().Name {
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
			ps.Trace.Out(ps, pr, RunAct, ps.Pos, textpos.RegionZero, useAST, fmt.Sprintf("Act %v: ERROR: node not found at path(s): %v in node: %v", act.Act, act.Path, apath))
		}
		return false
	}
	ast := node.(*AST)
	lx := ps.Src.LexAt(ast.TokReg.Start)
	useTok := lx.Token.Token
	if act.Token != token.None {
		useTok = act.Token
	}
	nm := ast.Src
	nms := strings.Split(nm, ",")
	if len(adnl) > 0 {
		for _, pk := range adnl {
			nast := pk.(*AST)
			if nast != ast {
				nms = append(nms, strings.Split(nast.Src, ",")...)
			}
		}
	}
	for i := range nms {
		nms[i] = strings.TrimSpace(nms[i])
	}
	switch act.Act {
	case ChangeToken:
		cp := ast.TokReg.Start
		for cp.IsLess(ast.TokReg.End) {
			tlx := ps.Src.LexAt(cp)
			act.ChangeToken(tlx)
			cp, _ = ps.Src.NextTokenPos(cp)
		}
		if len(adnl) > 0 {
			for _, pk := range adnl {
				nast := pk.(*AST)
				cp := nast.TokReg.Start
				for cp.IsLess(nast.TokReg.End) {
					tlx := ps.Src.LexAt(cp)
					act.ChangeToken(tlx)
					cp, _ = ps.Src.NextTokenPos(cp)
				}
			}
		}
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, RunAct, ast.TokReg.Start, ast.TokReg, ast, fmt.Sprintf("Act: Token set to: %v from path: %v = %v in node: %v", act.Token, act.Path, nm, apath))
		}
		return false
	case AddSymbol:
		for i := range nms {
			n := nms[i]
			if n == "" || n == "_" { // go special case..
				continue
			}
			sy, has := ps.FindNameTopScope(n) // only look in top scope
			added := false
			if has {
				sy.Region = ast.SrcReg
				sy.Kind = useTok
				if ps.Trace.On {
					ps.Trace.Out(ps, pr, RunAct, ast.TokReg.Start, ast.TokReg, ast, fmt.Sprintf("Act: Add sym already exists: %v from path: %v = %v in node: %v", sy.String(), act.Path, n, apath))
				}
			} else {
				sy = syms.NewSymbol(n, useTok, ps.Src.Filename, ast.SrcReg)
				added = sy.AddScopesStack(ps.Scopes)
				if !added {
					ps.Syms.Add(sy)
				}
			}
			useAST.Syms.Push(sy)
			sy.AST = useAST.This
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, RunAct, ast.TokReg.Start, ast.TokReg, ast, fmt.Sprintf("Act: Added sym: %v from path: %v = %v in node: %v", sy.String(), act.Path, n, apath))
			}
		}
	case PushScope:
		sy, has := ps.FindNameTopScope(nm) // Scoped(nm)
		if !has {
			sy = syms.NewSymbol(nm, useTok, ps.Src.Filename, ast.SrcReg) // textpos.RegionZero) // zero = tmp
			added := sy.AddScopesStack(ps.Scopes)
			if !added {
				ps.Syms.Add(sy)
			}
		}
		ps.Scopes.Push(sy)
		useAST.Syms.Push(sy)
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, RunAct, ast.TokReg.Start, ast.TokReg, ast, fmt.Sprintf("Act: Pushed Sym: %v from path: %v = %v in node: %v", sy.String(), act.Path, nm, apath))
		}
	case PushNewScope:
		// add plus push
		sy, has := ps.FindNameTopScope(nm) // Scoped(nm)
		if has {
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, RunAct, ast.TokReg.Start, ast.TokReg, ast, fmt.Sprintf("Act: Push New sym already exists: %v from path: %v = %v in node: %v", sy.String(), act.Path, nm, apath))
			}
		} else {
			sy = syms.NewSymbol(nm, useTok, ps.Src.Filename, ast.SrcReg)
			added := sy.AddScopesStack(ps.Scopes)
			if !added {
				ps.Syms.Add(sy)
			}
		}
		ps.Scopes.Push(sy) // key diff from add..
		useAST.Syms.Push(sy)
		sy.AST = useAST.This
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, RunAct, ast.TokReg.Start, ast.TokReg, ast, fmt.Sprintf("Act: Pushed New Sym: %v from path: %v = %v in node: %v", sy.String(), act.Path, nm, apath))
		}
	case PopScope:
		sy := ps.Scopes.Pop()
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, RunAct, ast.TokReg.Start, ast.TokReg, ast, fmt.Sprintf("Act: Popped Sym: %v in node: %v", sy.String(), apath))
		}
	case PopScopeReg:
		sy := ps.Scopes.Pop()
		sy.Region = ast.SrcReg // update source region to final -- select remains initial trigger one
		if ps.Trace.On {
			ps.Trace.Out(ps, pr, RunAct, ast.TokReg.Start, ast.TokReg, ast, fmt.Sprintf("Act: Popped Sym: %v in node: %v", sy.String(), apath))
		}
	case AddDetail:
		sy := useAST.Syms.Top()
		if sy != nil {
			if sy.Detail == "" {
				sy.Detail = nm
			} else {
				sy.Detail += " " + nm
			}
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, RunAct, ast.TokReg.Start, ast.TokReg, ast, fmt.Sprintf("Act: Added Detail: %v to Sym: %v in node: %v", nm, sy.String(), apath))
			}
		} else {
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, RunAct, ast.TokReg.Start, ast.TokReg, ast, fmt.Sprintf("Act: Add Detail: %v ERROR -- symbol not found in node: %v", nm, apath))
			}
		}
	case AddType:
		scp := ps.Scopes.Top()
		if scp == nil {
			ps.Trace.Out(ps, pr, RunAct, ast.TokReg.Start, ast.TokReg, ast, fmt.Sprintf("Act: Add Type: %v ERROR -- requires current scope -- none set in node: %v", nm, apath))
			return false
		}
		for i := range nms {
			n := nms[i]
			if n == "" || n == "_" { // go special case..
				continue
			}
			ty := syms.NewType(n, syms.Unknown)
			ty.Filename = ps.Src.Filename
			ty.Region = ast.SrcReg
			ty.AST = useAST.This
			ty.AddScopesStack(ps.Scopes)
			scp.Types.Add(ty)
			if ps.Trace.On {
				ps.Trace.Out(ps, pr, RunAct, ast.TokReg.Start, ast.TokReg, ast, fmt.Sprintf("Act: Added type: %v from path: %v = %v in node: %v", ty.String(), act.Path, n, apath))
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
	pr.WalkDown(func(k tree.Node) bool {
		pri := k.(*Rule)
		if strings.Contains(pri.Rule, find) || strings.Contains(pri.Name, find) {
			res = append(res, pri)
		}
		return true
	})
	return res
}

// WriteGrammar outputs the parser rules as a formatted grammar in a BNF-like format
// it is called recursively
func (pr *Rule) WriteGrammar(writer io.Writer, depth int) {
	if tree.IsRoot(pr) {
		for _, k := range pr.Children {
			pri := k.(*Rule)
			pri.WriteGrammar(writer, depth)
		}
	} else {
		ind := indent.Tabs(depth)
		nmstr := pr.Name
		if pr.Off {
			nmstr = "// OFF: " + nmstr
		}
		if pr.Desc != "" {
			fmt.Fprintf(writer, "%v// %v %v \n", ind, nmstr, pr.Desc)
		}
		if pr.IsGroup() {
			fmt.Fprintf(writer, "%v%v {\n", ind, nmstr)
			w := tabwriter.NewWriter(writer, 4, 4, 2, ' ', 0)
			for _, k := range pr.Children {
				pri := k.(*Rule)
				pri.WriteGrammar(w, depth+1)
			}
			w.Flush()
			fmt.Fprintf(writer, "%v}\n", ind)
		} else {
			astr := ""
			switch pr.AST {
			case AddAST:
				astr = "+AST"
			case SubAST:
				astr = "_AST"
			case AnchorAST:
				astr = ">AST"
			case AnchorFirstAST:
				astr = ">1AST"
			}
			fmt.Fprintf(writer, "%v%v:\t%v\t%v\n", ind, nmstr, pr.Rule, astr)
			if len(pr.Acts) > 0 {
				fmt.Fprintf(writer, "%v--->Acts:%v\n", ind, pr.Acts.String())
			}
		}
	}
}
