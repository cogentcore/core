// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"fmt"

	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/parse/syms"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
)

// parser.State is the state maintained for parsing
type State struct {

	// source and lexed version of source we're parsing
	Src *lexer.File `display:"no-inline"`

	// tracing for this parser
	Trace TraceOptions

	// root of the AST abstract syntax tree we're updating
	AST *AST

	// symbol map that everything gets added to from current file of parsing -- typically best for subsequent management to just have a single outer-most scoping symbol here (e.g., in Go it is the package), and then everything is a child under that
	Syms syms.SymMap

	// stack of scope(s) added to FileSyms e.g., package, library, module-level elements of which this file is a part -- these are reset at the start and must be added by parsing actions within the file itself
	Scopes syms.SymStack

	// the current lex token position
	Pos textpos.Pos

	// any error messages accumulated during parsing specifically
	Errs lexer.ErrorList `display:"no-inline"`

	// rules that matched and ran at each point, in 1-to-1 correspondence with the Src.Lex tokens for the lines and char pos dims
	Matches [][]MatchStack `display:"no-inline"`

	// rules that did NOT match -- represented as a map by scope of a RuleSet
	NonMatches ScopeRuleSet `display:"no-inline"`

	// stack for context-sensitive rules
	Stack lexer.Stack `display:"no-inline"`
}

// Init initializes the state at start of parsing
func (ps *State) Init(src *lexer.File, ast *AST) {
	// fmt.Println("in init")
	// if ps.Src != nil {
	// 	fmt.Println("was:", ps.Src.Filename)
	// }
	// if src != nil {
	// 	fmt.Println("new:", src.Filename)
	// }
	ps.Src = src
	if ps.AST != nil && ps.AST.This != nil {
		// fmt.Println("deleting old ast")
		ps.AST.DeleteChildren()
	}
	ps.AST = ast
	if ps.AST != nil && ps.AST.This != nil {
		// fmt.Println("deleting new ast")
		ps.AST.DeleteChildren()
	}
	ps.ClearAST()
	ps.Syms.Reset()
	ps.Scopes.Reset()
	ps.Stack.Reset()
	if ps.Src != nil {
		ps.Pos, _ = ps.Src.ValidTokenPos(textpos.PosZero)
	}
	ps.Errs.Reset()
	ps.Trace.Init()
	ps.AllocRules()
}

func (ps *State) ClearAST() {
	ps.Syms.ClearAST()
	ps.Scopes.ClearAST()
}

func (ps *State) Destroy() {
	if ps.AST != nil && ps.AST.This != nil {
		ps.AST.DeleteChildren()
	}
	ps.AST = nil
	ps.ClearAST()
	ps.Syms.Reset()
	ps.Scopes.Reset()
	ps.Stack.Reset()
	if ps.Src != nil {
		ps.Pos, _ = ps.Src.ValidTokenPos(textpos.PosZero)
	}
	ps.Errs.Reset()
	ps.Trace.Init()
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
func (ps *State) Error(pos textpos.Pos, msg string, rule *Rule) {
	if pos != textpos.PosZero {
		pos = ps.Src.TokenSrcPos(pos).Start
	}
	e := ps.Errs.Add(pos, ps.Src.Filename, msg, ps.Src.SrcLine(pos.Line), rule)
	if GUIActive {
		erstr := e.Report(ps.Src.BasePath, true, true)
		fmt.Fprintln(ps.Trace.OutWrite, "ERROR: "+erstr)
	}
}

// AtEof returns true if current position is at end of file -- this includes
// common situation where it is just at the very last token
func (ps *State) AtEof() bool {
	if ps.Pos.Line >= ps.Src.NLines() {
		return true
	}
	_, ok := ps.Src.ValidTokenPos(ps.Pos)
	return !ok
}

// AtEofNext returns true if current OR NEXT position is at end of file -- this includes
// common situation where it is just at the very last token
func (ps *State) AtEofNext() bool {
	if ps.AtEof() {
		return true
	}
	return ps.Pos.Line == ps.Src.NLines()-1
}

// GotoEof sets current position at EOF
func (ps *State) GotoEof() {
	ps.Pos.Line = ps.Src.NLines()
	ps.Pos.Char = 0
}

// NextSrcLine returns the next line of text
func (ps *State) NextSrcLine() string {
	sp, ok := ps.Src.ValidTokenPos(ps.Pos)
	if !ok {
		return ""
	}
	ep := sp
	ep.Char = ps.Src.NTokens(ep.Line)
	if ep.Char == sp.Char+1 { // only one
		nep, ok := ps.Src.ValidTokenPos(ep)
		if ok {
			ep = nep
			ep.Char = ps.Src.NTokens(ep.Line)
		}
	}
	reg := textpos.Region{Start: sp, End: ep}
	return ps.Src.TokenRegSrc(reg)
}

// MatchLex is our optimized matcher method, matching tkey depth as well
func (ps *State) MatchLex(lx *lexer.Lex, tkey token.KeyToken, isCat, isSubCat bool, cp textpos.Pos) bool {
	if lx.Token.Depth != tkey.Depth {
		return false
	}
	if !(lx.Token.Token == tkey.Token || (isCat && lx.Token.Token.Cat() == tkey.Token) || (isSubCat && lx.Token.Token.SubCat() == tkey.Token)) {
		return false
	}
	if tkey.Key == "" {
		return true
	}
	return tkey.Key == lx.Token.Key
}

// FindToken looks for token in given region, returns position where found, false if not.
// Only matches when depth is same as at reg.Start start at the start of the search.
// All positions in token indexes.
func (ps *State) FindToken(tkey token.KeyToken, reg textpos.Region) (textpos.Pos, bool) {
	// prf := profile.Start("FindToken")
	// defer prf.End()
	cp, ok := ps.Src.ValidTokenPos(reg.Start)
	if !ok {
		return cp, false
	}
	tok := tkey.Token
	isCat := tok.Cat() == tok
	isSubCat := tok.SubCat() == tok
	for cp.IsLess(reg.End) {
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
func (ps *State) MatchToken(tkey token.KeyToken, pos textpos.Pos) bool {
	tok := tkey.Token
	isCat := tok.Cat() == tok
	isSubCat := tok.SubCat() == tok
	lx := ps.Src.LexAt(pos)
	tkey.Depth = lx.Token.Depth
	return ps.MatchLex(lx, tkey, isCat, isSubCat, pos)
}

// FindTokenReverse looks *backwards* for token in given region, with same depth as reg.End-1 end
// where the search starts. Returns position where found, false if not.
// Automatically deals with possible confusion with unary operators -- if there are two
// ambiguous operators in a row, automatically gets the first one.  This is mainly / only used for
// binary operator expressions (mathematical binary operators).
// All positions are in token indexes.
func (ps *State) FindTokenReverse(tkey token.KeyToken, reg textpos.Region) (textpos.Pos, bool) {
	// prf := profile.Start("FindTokenReverse")
	// defer prf.End()
	cp, ok := ps.Src.PrevTokenPos(reg.End)
	if !ok {
		return cp, false
	}
	tok := tkey.Token
	isCat := tok.Cat() == tok
	isSubCat := tok.SubCat() == tok
	isAmbigUnary := tok.IsAmbigUnaryOp()
	for reg.Start.IsLess(cp) || cp == reg.Start {
		lx := ps.Src.LexAt(cp)
		if ps.MatchLex(lx, tkey, isCat, isSubCat, cp) {
			if isAmbigUnary { // make sure immed prior is not also!
				pp, ok := ps.Src.PrevTokenPos(cp)
				if ok {
					pt := ps.Src.Token(pp)
					if tok == token.OpMathMul {
						if !pt.Token.IsUnaryOp() {
							return cp, true
						}
					} else {
						if !pt.Token.IsAmbigUnaryOp() {
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

// AddAST adds a child AST node to given parent AST node
func (ps *State) AddAST(parAST *AST, rule string, reg textpos.Region) *AST {
	chAST := NewAST(parAST)
	chAST.SetName(rule)
	chAST.SetTokReg(reg, ps.Src)
	return chAST
}

///////////////////////////////////////////////////////////////////////////
//  Match State, Stack

// MatchState holds state info for rules that matched, recorded at starting position of match
type MatchState struct {

	// rule that either matched or ran here
	Rule *Rule

	// scope for match
	Scope textpos.Region

	// regions of match for each sub-region
	Regs Matches
}

// String is fmt.Stringer
func (rs MatchState) String() string {
	if rs.Rule == nil {
		return ""
	}
	return fmt.Sprintf("%v%v", rs.Rule.Name, rs.Scope)
}

// MatchStack is the stack of rules that matched or ran for each token point
type MatchStack []MatchState

// Add given rule to stack
func (rs *MatchStack) Add(pr *Rule, scope textpos.Region, regs Matches) {
	*rs = append(*rs, MatchState{Rule: pr, Scope: scope, Regs: regs})
}

// Find looks for given rule and scope on the stack
func (rs *MatchStack) Find(pr *Rule, scope textpos.Region) (*MatchState, bool) {
	for i := range *rs {
		r := &(*rs)[i]
		if r.Rule == pr && r.Scope == scope {
			return r, true
		}
	}
	return nil, false
}

// AddMatch adds given rule to rule stack at given scope
func (ps *State) AddMatch(pr *Rule, scope textpos.Region, regs Matches) {
	rs := &ps.Matches[scope.Start.Line][scope.Start.Char]
	rs.Add(pr, scope, regs)
}

// IsMatch looks for rule at given scope in list of matches, if found
// returns match state info
func (ps *State) IsMatch(pr *Rule, scope textpos.Region) (*MatchState, bool) {
	rs := &ps.Matches[scope.Start.Line][scope.Start.Char]
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
				txt += ` "` + string(ps.Src.TokenSrc(textpos.Pos{ln, ch})) + `"`
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
	Scope textpos.Region
	Rule  *Rule
}

// ScopeRuleSet is a map by scope of RuleSets, for non-matching rules
type ScopeRuleSet map[ScopeRule]struct{}

// Add a rule to scope set, with auto-alloc
func (rs ScopeRuleSet) Add(scope textpos.Region, pr *Rule) {
	sr := ScopeRule{scope, pr}
	rs[sr] = struct{}{}
}

// Has checks if scope rule set has given scope, rule
func (rs ScopeRuleSet) Has(scope textpos.Region, pr *Rule) bool {
	sr := ScopeRule{scope, pr}
	_, has := rs[sr]
	return has
}

// AddNonMatch adds given rule to non-matching rule set for this scope
func (ps *State) AddNonMatch(scope textpos.Region, pr *Rule) {
	ps.NonMatches.Add(scope, pr)
}

// IsNonMatch looks for rule in nonmatch list at given scope
func (ps *State) IsNonMatch(scope textpos.Region, pr *Rule) bool {
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
// with the given name in symbols
// also looks in ps.Syms if not found in Scope stack.
func (ps *State) FindNameTopScope(nm string) (*syms.Symbol, bool) {
	sy := ps.Scopes.Top()
	if sy == nil {
		return nil, false
	}
	chs, has := sy.Children[nm]
	return chs, has
}
