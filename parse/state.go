// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"fmt"

	"github.com/goki/pi/lex"
	"github.com/goki/pi/syms"
	"github.com/goki/pi/token"
)

// parse.State is the state maintained for parsing
type State struct {
	Src        *lex.File      `view:"no-inline" desc:"source and lexed version of source we're parsing"`
	Trace      TraceOpts      `desc:"tracing for this parser"`
	Ast        *Ast           `desc:"root of the Ast abstract syntax tree we're updating"`
	Syms       syms.SymMap    `desc:"symbol map that everything gets added to from current file of parsing -- typically best for subsequent management to just have a single outer-most scoping symbol here (e.g., in Go it is the package), and then everything is a child under that"`
	Scopes     syms.SymStack  `desc:"stack of scope(s) added to FileSyms e.g., package, library, module-level elements of which this file is a part -- these are reset at the start and must be added by parsing actions within the file itself"`
	Pos        lex.Pos        `desc:"the current lex token position"`
	Errs       lex.ErrorList  `view:"no-inline" desc:"any error messages accumulated during parsing specifically"`
	Matches    [][]MatchStack `view:"no-inline" desc:"rules that matched and ran at each point, in 1-to-1 correspondence with the Src.Lex tokens for the lines and char pos dims"`
	NonMatches ScopeRuleSet   `view:"no-inline" desc:"rules that did NOT match -- represented as a map by scope of a RuleSet"`
	Stack      lex.Stack      `view:"no-inline" desc:"stack for context-sensitive rules"`
}

// Init initializes the state at start of parsing
func (ps *State) Init(src *lex.File, ast *Ast) {
	ps.Src = src
	ps.Ast = ast
	ps.Ast.DeleteChildren(true)
	ps.Syms.Reset()
	ps.Scopes.Reset()
	ps.Stack.Reset()
	ps.Pos, _ = ps.Src.ValidTokenPos(lex.PosZero)
	ps.Errs.Reset()
	ps.Trace.Init()
	ps.AllocRules()
}

// AllocRules allocate the match, nonmatch rule state in correspondence with the src state
func (ps *State) AllocRules() {
	nlines := ps.Src.NLines()
	if nlines == 0 {
		return
	}
	if len(ps.Src.Lexs) != nlines {
		return
	}
	ps.Matches = make([][]MatchStack, nlines)
	ntot := 0
	for ln := 0; ln < nlines; ln++ {
		sz := len(ps.Src.Lexs[ln])
		if sz > 0 {
			ps.Matches[ln] = make([]MatchStack, sz)
			ntot += sz
		}
	}
	ps.NonMatches = make(ScopeRuleSet, ntot*10)
}

// Error adds a parsing error at given lex token position
func (ps *State) Error(pos lex.Pos, msg string, rule *Rule) {
	if pos != lex.PosZero {
		pos = ps.Src.TokenSrcPos(pos).St
	}
	e := ps.Errs.Add(pos, ps.Src.Filename, msg, ps.Src.SrcLine(pos.Ln), rule)
	if GuiActive {
		erstr := e.Report(ps.Src.BasePath, true, true)
		fmt.Fprintln(ps.Trace.OutWrite, "ERROR: "+erstr)
	}
}

// AtEof returns true if current position is at end of file -- this includes
// common situation where it is just at the very last token
func (ps *State) AtEof() bool {
	if ps.Pos.Ln >= ps.Src.NLines() {
		return true
	}
	_, ok := ps.Src.ValidTokenPos(ps.Pos)
	if !ok {
		return true
	}
	return false
}

// AtEofNext returns true if current OR NEXT position is at end of file -- this includes
// common situation where it is just at the very last token
func (ps *State) AtEofNext() bool {
	if ps.AtEof() {
		return true
	}
	if ps.Pos.Ln == ps.Src.NLines()-1 {
		return true
	}
	return false
}

// GotoEof sets current position at EOF
func (ps *State) GotoEof() {
	ps.Pos.Ln = ps.Src.NLines()
	ps.Pos.Ch = 0
}

// NextSrcLine returns the next line of text
func (ps *State) NextSrcLine() string {
	sp, ok := ps.Src.ValidTokenPos(ps.Pos)
	if !ok {
		return ""
	}
	ep := sp
	ep.Ch = ps.Src.NTokens(ep.Ln)
	if ep.Ch == sp.Ch+1 { // only one
		nep, ok := ps.Src.ValidTokenPos(ep)
		if ok {
			ep = nep
			ep.Ch = ps.Src.NTokens(ep.Ln)
		}
	}
	reg := lex.Reg{St: sp, Ed: ep}
	return ps.Src.TokenRegSrc(reg)
}

// MatchLex is our optimized matcher method, matching tkey depth as well
func (ps *State) MatchLex(lx *lex.Lex, tkey token.KeyToken, isCat, isSubCat bool, cp lex.Pos) bool {
	if lx.Tok.Depth != tkey.Depth {
		return false
	}
	if !(lx.Tok.Tok == tkey.Tok || (isCat && lx.Tok.Tok.Cat() == tkey.Tok) || (isSubCat && lx.Tok.Tok.SubCat() == tkey.Tok)) {
		return false
	}
	if tkey.Key == "" {
		return true
	}
	return tkey.Key == lx.Tok.Key
}

// FindToken looks for token in given region, returns position where found, false if not.
// Only matches when depth is same as at reg.St start at the start of the search.
// All positions in token indexes.
func (ps *State) FindToken(tkey token.KeyToken, reg lex.Reg) (lex.Pos, bool) {
	// prf := prof.Start("FindToken")
	// defer prf.End()
	cp, ok := ps.Src.ValidTokenPos(reg.St)
	if !ok {
		return cp, false
	}
	tok := tkey.Tok
	isCat := tok.Cat() == tok
	isSubCat := tok.SubCat() == tok
	for cp.IsLess(reg.Ed) {
		lx := ps.Src.LexAt(cp)
		if ps.MatchLex(lx, tkey, isCat, isSubCat, cp) {
			return cp, true
		}
		cp, ok = ps.Src.NextTokenPos(cp)
		if !ok {
			return cp, false
		}
	}
	return cp, false
}

// MatchToken returns true if token matches at given position -- must be
// a valid position!
func (ps *State) MatchToken(tkey token.KeyToken, pos lex.Pos) bool {
	tok := tkey.Tok
	isCat := tok.Cat() == tok
	isSubCat := tok.SubCat() == tok
	lx := ps.Src.LexAt(pos)
	tkey.Depth = lx.Tok.Depth
	return ps.MatchLex(lx, tkey, isCat, isSubCat, pos)
}

// FindTokenReverse looks *backwards* for token in given region, with same depth as reg.Ed-1 end
// where the search starts. Returns position where found, false if not.
// Automatically deals with possible confusion with unary operators -- if there are two
// ambiguous operators in a row, automatically gets the first one.  This is mainly / only used for
// binary operator expressions (mathematical binary operators).
// All positions are in token indexes.
func (ps *State) FindTokenReverse(tkey token.KeyToken, reg lex.Reg) (lex.Pos, bool) {
	// prf := prof.Start("FindTokenReverse")
	// defer prf.End()
	cp, ok := ps.Src.PrevTokenPos(reg.Ed)
	if !ok {
		return cp, false
	}
	tok := tkey.Tok
	isCat := tok.Cat() == tok
	isSubCat := tok.SubCat() == tok
	isAmbigUnary := tok.IsAmbigUnaryOp()
	for reg.St.IsLess(cp) || cp == reg.St {
		lx := ps.Src.LexAt(cp)
		if ps.MatchLex(lx, tkey, isCat, isSubCat, cp) {
			if isAmbigUnary { // make sure immed prior is not also!
				pp, ok := ps.Src.PrevTokenPos(cp)
				if ok {
					pt := ps.Src.Token(pp)
					if tok == token.OpMathMul {
						if !pt.Tok.IsUnaryOp() {
							return cp, true
						}
					} else {
						if !pt.Tok.IsAmbigUnaryOp() {
							return cp, true
						}
					}
					// otherwise we don't match -- cannot match second opr
				} else {
					return cp, true
				}
			} else {
				return cp, true
			}
		}
		ok := false
		cp, ok = ps.Src.PrevTokenPos(cp)
		if !ok {
			return cp, false
		}
	}
	return cp, false
}

// AddAst adds a child Ast node to given parent Ast node
func (ps *State) AddAst(parAst *Ast, rule string, reg lex.Reg) *Ast {
	chAst := &Ast{}
	chAst.InitName(chAst, rule)
	parAst.AddChild(chAst)
	chAst.SetTokReg(reg, ps.Src)
	return chAst
}

///////////////////////////////////////////////////////////////////////////
//  Match State, Stack

// MatchState holds state info for rules that matched, recorded at starting position of match
type MatchState struct {
	Rule  *Rule   `desc:"rule that either matched or ran here"`
	Scope lex.Reg `desc:"scope for match"`
	Regs  Matches `desc:"regions of match for each sub-region"`
}

// String is fmt.Stringer
func (rs MatchState) String() string {
	if rs.Rule == nil {
		return ""
	}
	return fmt.Sprintf("%v%v", rs.Rule.Name(), rs.Scope)
}

// MatchStack is the stack of rules that matched or ran for each token point
type MatchStack []MatchState

// Add given rule to stack
func (rs *MatchStack) Add(pr *Rule, scope lex.Reg, regs Matches) {
	*rs = append(*rs, MatchState{Rule: pr, Scope: scope, Regs: regs})
}

// Find looks for given rule and scope on the stack
func (rs *MatchStack) Find(pr *Rule, scope lex.Reg) (*MatchState, bool) {
	for i := range *rs {
		r := &(*rs)[i]
		if r.Rule == pr && r.Scope == scope {
			return r, true
		}
	}
	return nil, false
}

// AddMatch adds given rule to rule stack at given scope
func (ps *State) AddMatch(pr *Rule, scope lex.Reg, regs Matches) {
	rs := &ps.Matches[scope.St.Ln][scope.St.Ch]
	rs.Add(pr, scope, regs)
}

// IsMatch looks for rule at given scope in list of matches, if found
// returns match state info
func (ps *State) IsMatch(pr *Rule, scope lex.Reg) (*MatchState, bool) {
	rs := &ps.Matches[scope.St.Ln][scope.St.Ch]
	sz := len(*rs)
	if sz == 0 {
		return nil, false
	}
	return rs.Find(pr, scope)
}

// RuleString returns the rule info for entire source -- if full
// then it includes the full stack at each point -- otherwise just the top
// of stack
func (ps *State) RuleString(full bool) string {
	txt := ""
	nlines := ps.Src.NLines()
	for ln := 0; ln < nlines; ln++ {
		sz := len(ps.Matches[ln])
		if sz == 0 {
			txt += "\n"
		} else {
			for ch := 0; ch < sz; ch++ {
				rs := ps.Matches[ln][ch]
				sd := len(rs)
				txt += ` "` + string(ps.Src.TokenSrc(lex.Pos{ln, ch})) + `"`
				if sd == 0 {
					txt += "-"
				} else {
					if !full {
						txt += rs[sd-1].String()
					} else {
						txt += fmt.Sprintf("[%v: ", sd)
						for i := 0; i < sd; i++ {
							txt += rs[i].String()
						}
						txt += "]"
					}
				}
			}
			txt += "\n"
		}
	}
	return txt
}

///////////////////////////////////////////////////////////////////////////
//  ScopeRuleSet and NonMatch

// ScopeRule is a scope and a rule, for storing matches / nonmatch
type ScopeRule struct {
	Scope lex.Reg
	Rule  *Rule
}

// ScopeRuleSet is a map by scope of RuleSets, for non-matching rules
type ScopeRuleSet map[ScopeRule]struct{}

// Add a rule to scope set, with auto-alloc
func (rs ScopeRuleSet) Add(scope lex.Reg, pr *Rule) {
	sr := ScopeRule{scope, pr}
	rs[sr] = struct{}{}
}

// Has checks if scope rule set has given scope, rule
func (rs ScopeRuleSet) Has(scope lex.Reg, pr *Rule) bool {
	sr := ScopeRule{scope, pr}
	_, has := rs[sr]
	return has
}

// AddNonMatch adds given rule to non-matching rule set for this scope
func (ps *State) AddNonMatch(scope lex.Reg, pr *Rule) {
	ps.NonMatches.Add(scope, pr)
}

// IsNonMatch looks for rule in nonmatch list at given scope
func (ps *State) IsNonMatch(scope lex.Reg, pr *Rule) bool {
	return ps.NonMatches.Has(scope, pr)
}

// ResetNonMatches resets the non-match map -- do after every EOS
func (ps *State) ResetNonMatches() {
	ps.NonMatches = make(ScopeRuleSet)
}

///////////////////////////////////////////////////////////////////////////
//  Symbol management

// FindNameScoped searches top-down in the stack for something with the given name
// in symbols that are of subcategory token.NameScope (i.e., namespace, module, package, library)
// also looks in ps.Syms if not found in Scope stack.
func (ps *State) FindNameScoped(nm string) (*syms.Symbol, bool) {
	sy, has := ps.Scopes.FindNameScoped(nm)
	if has {
		return sy, has
	}
	return ps.Syms.FindNameScoped(nm)
}

// FindNameTopScope searches only in top of current scope for something
//  with the given name in symbols
// also looks in ps.Syms if not found in Scope stack.
func (ps *State) FindNameTopScope(nm string) (*syms.Symbol, bool) {
	sy := ps.Scopes.Top()
	if sy == nil {
		return nil, false
	}
	chs, has := sy.Children[nm]
	return chs, has
}
