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

	// nonMatches are the rules that did not match at each point, in 1-to-1
	// correspondence with the Src.Lex tokens like Matches, for the scopes
	// that start there.
	nonMatches [][]nonMatchSet

	// nonMatched has the positions in nonMatches that have rules,
	// which are cleared by [State.ResetNonMatches].
	nonMatched []textpos.Pos

	// matchBuf and nonMatchBuf are the buffers for Matches and nonMatches,
	// which are reused for the next parse, typically of the same file after
	// an edit, along with the slices for each position (see [State.AllocRules]).
	matchBuf    []MatchStack
	nonMatchBuf []nonMatchSet

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
	ntot := 0
	for ln := 0; ln < nlines; ln++ {
		ntot += len(ps.Src.Lexs[ln])
	}
	// We make one allocation for all of the lines, and reuse the buffers
	// of the previous parse when they are big enough, truncating the slices
	// of each position so that their capacity is also reused.
	if cap(ps.matchBuf) < ntot {
		ps.matchBuf = make([]MatchStack, ntot, ntot+ntot/4)
		ps.nonMatchBuf = make([]nonMatchSet, ntot, ntot+ntot/4)
	} else {
		ps.matchBuf = ps.matchBuf[:ntot]
		ps.nonMatchBuf = ps.nonMatchBuf[:ntot]
		for i := range ntot {
			ps.matchBuf[i] = ps.matchBuf[i][:0]
			nm := &ps.nonMatchBuf[i]
			nm.rules = nm.rules[:0]
			nm.endLines = nm.endLines[:0]
			nm.endChars = nm.endChars[:0]
		}
	}
	if cap(ps.Matches) < nlines {
		ps.Matches = make([][]MatchStack, nlines, nlines+nlines/4)
		ps.nonMatches = make([][]nonMatchSet, nlines, nlines+nlines/4)
	} else {
		ps.Matches = ps.Matches[:nlines]
		ps.nonMatches = ps.nonMatches[:nlines]
	}
	ps.nonMatched = ps.nonMatched[:0]
	off := 0
	for ln := 0; ln < nlines; ln++ {
		sz := len(ps.Src.Lexs[ln])
		ps.Matches[ln] = ps.matchBuf[off : off+sz : off+sz]
		ps.nonMatches[ln] = ps.nonMatchBuf[off : off+sz : off+sz]
		off += sz
	}
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
	s := *rs
	for i := range s {
		// We only take the address of a match, which is costly in JavaScript.
		if s[i].Rule == pr && s[i].Scope == scope {
			return &s[i], true
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
//  NonMatch

// nonMatchSet has the rules that did not match for the scopes that start at
// its position in [State.nonMatches], with the line and character of the end
// of each scope. These are in separate slices of basic types, which are
// faster to search than a slice of structs, especially in JavaScript.
type nonMatchSet struct {
	rules    []*Rule
	endLines []int
	endChars []int
}

// nonMatchesAt returns the non-matching rules at the start of the given
// scope, or nil if it is outside of the source.
func (ps *State) nonMatchesAt(scope textpos.Region) *nonMatchSet {
	ln, ch := scope.Start.Line, scope.Start.Char
	if ln < 0 || ln >= len(ps.nonMatches) || ch < 0 || ch >= len(ps.nonMatches[ln]) {
		return nil
	}
	return &ps.nonMatches[ln][ch]
}

// AddNonMatch adds given rule to non-matching rule set for this scope
func (ps *State) AddNonMatch(scope textpos.Region, pr *Rule) {
	nm := ps.nonMatchesAt(scope)
	if nm == nil {
		return
	}
	if len(nm.rules) == 0 {
		ps.nonMatched = append(ps.nonMatched, scope.Start)
	}
	nm.rules = append(nm.rules, pr)
	nm.endLines = append(nm.endLines, scope.End.Line)
	nm.endChars = append(nm.endChars, scope.End.Char)
}

// IsNonMatch looks for rule in nonmatch list at given scope
func (ps *State) IsNonMatch(scope textpos.Region, pr *Rule) bool {
	nm := ps.nonMatchesAt(scope)
	if nm == nil {
		return false
	}
	eln, ech := scope.End.Line, scope.End.Char
	endLines, endChars := nm.endLines, nm.endChars
	for i, r := range nm.rules {
		if r == pr && endLines[i] == eln && endChars[i] == ech {
			return true
		}
	}
	return false
}

// ResetNonMatches resets the non-matching rules -- do after every EOS
func (ps *State) ResetNonMatches() {
	for _, p := range ps.nonMatched {
		nm := &ps.nonMatches[p.Line][p.Char]
		nm.rules = nm.rules[:0]
		nm.endLines = nm.endLines[:0]
		nm.endChars = nm.endChars[:0]
	}
	ps.nonMatched = ps.nonMatched[:0]
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
