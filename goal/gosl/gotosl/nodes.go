// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file is largely copied from the Go source,
// src/go/printer/nodes.go:

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file implements printing of AST nodes; specifically
// expressions, statements, declarations, and files. It uses
// the print functionality implemented in printer.go.

package gotosl

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"math"
	"path"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Formatting issues:
// - better comment formatting for /*-style comments at the end of a line (e.g. a declaration)
//   when the comment spans multiple lines; if such a comment is just two lines, formatting is
//   not idempotent
// - formatting of expression lists
// - should use blank instead of tab to separate one-line function bodies from
//   the function header unless there is a group of consecutive one-liners

// ----------------------------------------------------------------------------
// Common AST nodes.

// Print as many newlines as necessary (but at least min newlines) to get to
// the current line. ws is printed before the first line break. If newSection
// is set, the first line break is printed as formfeed. Returns 0 if no line
// breaks were printed, returns 1 if there was exactly one newline printed,
// and returns a value > 1 if there was a formfeed or more than one newline
// printed.
//
// TODO(gri): linebreak may add too many lines if the next statement at "line"
// is preceded by comments because the computation of n assumes
// the current position before the comment and the target position
// after the comment. Thus, after interspersing such comments, the
// space taken up by them is not considered to reduce the number of
// linebreaks. At the moment there is no easy way to know about
// future (not yet interspersed) comments in this function.
func (p *printer) linebreak(line, min int, ws whiteSpace, newSection bool) (nbreaks int) {
	n := max(nlimit(line-p.pos.Line), min)
	if n > 0 {
		p.print(ws)
		if newSection {
			p.print(formfeed)
			n--
			nbreaks = 2
		}
		nbreaks += n
		for ; n > 0; n-- {
			p.print(newline)
		}
	}
	return
}

// gosl: find any gosl directive in given comments, returns directive(s) and remaining docs
func (p *printer) findDirective(g *ast.CommentGroup) (dirs []string, docs string) {
	if g == nil {
		return
	}
	for _, c := range g.List {
		if strings.HasPrefix(c.Text, "//gosl:") {
			dirs = append(dirs, c.Text[7:])
		} else {
			docs += c.Text + " "
		}
	}
	return
}

// gosl: hasDirective returns whether directive(s) contains string
func hasDirective(dirs []string, dir string) bool {
	for _, d := range dirs {
		if strings.Contains(d, dir) {
			return true
		}
	}
	return false
}

// gosl: directiveAfter returns the directive after given leading text,
// and a bool indicating if the string was found.
func directiveAfter(dirs []string, dir string) (string, bool) {
	for _, d := range dirs {
		if strings.HasPrefix(d, dir) {
			return strings.TrimSpace(strings.TrimPrefix(d, dir)), true
		}
	}
	return "", false
}

// setComment sets g as the next comment if g != nil and if node comments
// are enabled - this mode is used when printing source code fragments such
// as exports only. It assumes that there is no pending comment in p.comments
// and at most one pending comment in the p.comment cache.
func (p *printer) setComment(g *ast.CommentGroup) {
	if g == nil || !p.useNodeComments {
		return
	}
	if p.comments == nil {
		// initialize p.comments lazily
		p.comments = make([]*ast.CommentGroup, 1)
	} else if p.cindex < len(p.comments) {
		// for some reason there are pending comments; this
		// should never happen - handle gracefully and flush
		// all comments up to g, ignore anything after that
		p.flush(p.posFor(g.List[0].Pos()), token.ILLEGAL)
		p.comments = p.comments[0:1]
		// in debug mode, report error
		p.internalError("setComment found pending comments")
	}
	p.comments[0] = g
	p.cindex = 0
	// don't overwrite any pending comment in the p.comment cache
	// (there may be a pending comment when a line comment is
	// immediately followed by a lead comment with no other
	// tokens between)
	if p.commentOffset == infinity {
		p.nextComment() // get comment ready for use
	}
}

type exprListMode uint

const (
	commaTerm exprListMode = 1 << iota // list is optionally terminated by a comma
	noIndent                           // no extra indentation in multi-line lists
)

// If indent is set, a multi-line identifier list is indented after the
// first linebreak encountered.
func (p *printer) identList(list []*ast.Ident, indent bool) {
	// convert into an expression list so we can re-use exprList formatting
	xlist := make([]ast.Expr, len(list))
	for i, x := range list {
		xlist[i] = x
	}
	var mode exprListMode
	if !indent {
		mode = noIndent
	}
	p.exprList(token.NoPos, xlist, 1, mode, token.NoPos, false)
}

const filteredMsg = "contains filtered or unexported fields"

// Print a list of expressions. If the list spans multiple
// source lines, the original line breaks are respected between
// expressions.
//
// TODO(gri) Consider rewriting this to be independent of []ast.Expr
// so that we can use the algorithm for any kind of list
//
//	(e.g., pass list via a channel over which to range).
func (p *printer) exprList(prev0 token.Pos, list []ast.Expr, depth int, mode exprListMode, next0 token.Pos, isIncomplete bool) {
	if len(list) == 0 {
		if isIncomplete {
			prev := p.posFor(prev0)
			next := p.posFor(next0)
			if prev.IsValid() && prev.Line == next.Line {
				p.print("/* " + filteredMsg + " */")
			} else {
				p.print(newline)
				p.print(indent, "// "+filteredMsg, unindent, newline)
			}
		}
		return
	}

	prev := p.posFor(prev0)
	next := p.posFor(next0)
	line := p.lineFor(list[0].Pos())
	endLine := p.lineFor(list[len(list)-1].End())

	if prev.IsValid() && prev.Line == line && line == endLine {
		// all list entries on a single line
		for i, x := range list {
			if i > 0 {
				// use position of expression following the comma as
				// comma position for correct comment placement
				p.setPos(x.Pos())
				p.print(token.COMMA, blank)
			}
			p.expr0(x, depth)
		}
		if isIncomplete {
			p.print(token.COMMA, blank, "/* "+filteredMsg+" */")
		}
		return
	}

	// list entries span multiple lines;
	// use source code positions to guide line breaks

	// Don't add extra indentation if noIndent is set;
	// i.e., pretend that the first line is already indented.
	ws := ignore
	if mode&noIndent == 0 {
		ws = indent
	}

	// The first linebreak is always a formfeed since this section must not
	// depend on any previous formatting.
	prevBreak := -1 // index of last expression that was followed by a linebreak
	if prev.IsValid() && prev.Line < line && p.linebreak(line, 0, ws, true) > 0 {
		ws = ignore
		prevBreak = 0
	}

	// initialize expression/key size: a zero value indicates expr/key doesn't fit on a single line
	size := 0

	// We use the ratio between the geometric mean of the previous key sizes and
	// the current size to determine if there should be a break in the alignment.
	// To compute the geometric mean we accumulate the ln(size) values (lnsum)
	// and the number of sizes included (count).
	lnsum := 0.0
	count := 0

	// print all list elements
	prevLine := prev.Line
	for i, x := range list {
		line = p.lineFor(x.Pos())

		// Determine if the next linebreak, if any, needs to use formfeed:
		// in general, use the entire node size to make the decision; for
		// key:value expressions, use the key size.
		// TODO(gri) for a better result, should probably incorporate both
		//           the key and the node size into the decision process
		useFF := true

		// Determine element size: All bets are off if we don't have
		// position information for the previous and next token (likely
		// generated code - simply ignore the size in this case by setting
		// it to 0).
		prevSize := size
		const infinity = 1e6 // larger than any source line
		size = p.nodeSize(x, infinity)
		pair, isPair := x.(*ast.KeyValueExpr)
		if size <= infinity && prev.IsValid() && next.IsValid() {
			// x fits on a single line
			if isPair {
				size = p.nodeSize(pair.Key, infinity) // size <= infinity
			}
		} else {
			// size too large or we don't have good layout information
			size = 0
		}

		// If the previous line and the current line had single-
		// line-expressions and the key sizes are small or the
		// ratio between the current key and the geometric mean
		// if the previous key sizes does not exceed a threshold,
		// align columns and do not use formfeed.
		if prevSize > 0 && size > 0 {
			const smallSize = 40
			if count == 0 || prevSize <= smallSize && size <= smallSize {
				useFF = false
			} else {
				const r = 2.5                               // threshold
				geomean := math.Exp(lnsum / float64(count)) // count > 0
				ratio := float64(size) / geomean
				useFF = r*ratio <= 1 || r <= ratio
			}
		}

		needsLinebreak := 0 < prevLine && prevLine < line
		if i > 0 {
			// Use position of expression following the comma as
			// comma position for correct comment placement, but
			// only if the expression is on the same line.
			if !needsLinebreak {
				p.setPos(x.Pos())
			}
			p.print(token.COMMA)
			needsBlank := true
			if needsLinebreak {
				// Lines are broken using newlines so comments remain aligned
				// unless useFF is set or there are multiple expressions on
				// the same line in which case formfeed is used.
				nbreaks := p.linebreak(line, 0, ws, useFF || prevBreak+1 < i)
				if nbreaks > 0 {
					ws = ignore
					prevBreak = i
					needsBlank = false // we got a line break instead
				}
				// If there was a new section or more than one new line
				// (which means that the tabwriter will implicitly break
				// the section), reset the geomean variables since we are
				// starting a new group of elements with the next element.
				if nbreaks > 1 {
					lnsum = 0
					count = 0
				}
			}
			if needsBlank {
				p.print(blank)
			}
		}

		if len(list) > 1 && isPair && size > 0 && needsLinebreak {
			// We have a key:value expression that fits onto one line
			// and it's not on the same line as the prior expression:
			// Use a column for the key such that consecutive entries
			// can align if possible.
			// (needsLinebreak is set if we started a new line before)
			p.expr(pair.Key)
			p.setPos(pair.Colon)
			p.print(token.COLON, vtab)
			p.expr(pair.Value)
		} else {
			p.expr0(x, depth)
		}

		if size > 0 {
			lnsum += math.Log(float64(size))
			count++
		}

		prevLine = line
	}

	if mode&commaTerm != 0 && next.IsValid() && p.pos.Line < next.Line {
		// Print a terminating comma if the next token is on a new line.
		p.print(token.COMMA)
		if isIncomplete {
			p.print(newline)
			p.print("// " + filteredMsg)
		}
		if ws == ignore && mode&noIndent == 0 {
			// unindent if we indented
			p.print(unindent)
		}
		p.print(formfeed) // terminating comma needs a line break to look good
		return
	}

	if isIncomplete {
		p.print(token.COMMA, newline)
		p.print("// "+filteredMsg, newline)
	}

	if ws == ignore && mode&noIndent == 0 {
		// unindent if we indented
		p.print(unindent)
	}
}

type paramMode int

const (
	funcParam paramMode = iota
	funcTParam
	typeTParam
)

func (p *printer) parameters(fields *ast.FieldList, mode paramMode) {
	openTok, closeTok := token.LPAREN, token.RPAREN
	if mode != funcParam {
		openTok, closeTok = token.LBRACK, token.RBRACK
	}
	p.setPos(fields.Opening)
	p.print(openTok)
	if len(fields.List) > 0 {
		prevLine := p.lineFor(fields.Opening)
		ws := indent
		for i, par := range fields.List {
			// determine par begin and end line (may be different
			// if there are multiple parameter names for this par
			// or the type is on a separate line)
			parLineBeg := p.lineFor(par.Pos())
			parLineEnd := p.lineFor(par.End())
			// separating "," if needed
			needsLinebreak := 0 < prevLine && prevLine < parLineBeg
			if i > 0 {
				// use position of parameter following the comma as
				// comma position for correct comma placement, but
				// only if the next parameter is on the same line
				if !needsLinebreak {
					p.setPos(par.Pos())
				}
				p.print(token.COMMA)
			}
			// separator if needed (linebreak or blank)
			if needsLinebreak && p.linebreak(parLineBeg, 0, ws, true) > 0 {
				// break line if the opening "(" or previous parameter ended on a different line
				ws = ignore
			} else if i > 0 {
				p.print(blank)
			}
			// parameter names
			if len(par.Names) > 1 {
				nnm := len(par.Names)
				for ni, nm := range par.Names {
					p.print(nm.Name)
					p.print(token.COLON)
					p.print(blank)
					atyp, isPtr := p.ptrType(stripParensAlways(par.Type))
					p.expr(atyp)
					if isPtr {
						p.print(">")
						p.curPtrArgs = append(p.curPtrArgs, par.Names[0])
					}
					if ni < nnm-1 {
						p.print(token.COMMA)
					}
				}
			} else if len(par.Names) > 0 {
				// Very subtle: If we indented before (ws == ignore), identList
				// won't indent again. If we didn't (ws == indent), identList will
				// indent if the identList spans multiple lines, and it will outdent
				// again at the end (and still ws == indent). Thus, a subsequent indent
				// by a linebreak call after a type, or in the next multi-line identList
				// will do the right thing.
				p.identList(par.Names, ws == indent)
				p.print(token.COLON)
				p.print(blank)
				// parameter type -- gosl = type first, replace ptr star with `inout`
				atyp, isPtr := p.ptrType(stripParensAlways(par.Type))
				p.expr(atyp)
				if isPtr {
					p.print(">")
					p.curPtrArgs = append(p.curPtrArgs, par.Names[0])
				}
			} else {
				atyp, isPtr := p.ptrType(stripParensAlways(par.Type))
				p.expr(atyp)
				if isPtr {
					p.print(">")
				}
			}
			prevLine = parLineEnd
		}

		// if the closing ")" is on a separate line from the last parameter,
		// print an additional "," and line break
		if closing := p.lineFor(fields.Closing); 0 < prevLine && prevLine < closing {
			p.print(token.COMMA)
			p.linebreak(closing, 0, ignore, true)
		} else if mode == typeTParam && fields.NumFields() == 1 && combinesWithName(fields.List[0].Type) {
			// A type parameter list [P T] where the name P and the type expression T syntactically
			// combine to another valid (value) expression requires a trailing comma, as in [P *T,]
			// (or an enclosing interface as in [P interface(*T)]), so that the type parameter list
			// is not gotosld as an array length [P*T].
			p.print(token.COMMA)
		}

		// unindent if we indented
		if ws == ignore {
			p.print(unindent)
		}
	}

	p.setPos(fields.Closing)
	p.print(closeTok)
}

type rwArg struct {
	idx    *ast.IndexExpr
	tmpVar string
}

func (p *printer) assignRwArgs(rwargs []rwArg) {
	nrw := len(rwargs)
	if nrw == 0 {
		return
	}
	p.print(token.SEMICOLON, blank, formfeed)
	for i, rw := range rwargs {
		p.expr(rw.idx)
		p.print(token.ASSIGN)
		tv := rw.tmpVar
		if len(tv) > 0 && tv[0] == '&' {
			tv = tv[1:]
		}
		p.print(tv)
		if i < nrw-1 {
			p.print(token.SEMICOLON, blank)
		}
	}
}

// gosl: ensure basic literals are properly cast
func (p *printer) goslFixArgs(args []ast.Expr, params *types.Tuple) ([]ast.Expr, []rwArg) {
	ags := slices.Clone(args)
	mx := min(len(args), params.Len())
	var rwargs []rwArg
	for i := 0; i < mx; i++ {
		ag := args[i]
		pr := params.At(i)
		switch x := ag.(type) {
		case *ast.BasicLit:
			typ := pr.Type()
			tnm := getLocalTypeName(typ)
			nn := normalizedNumber(x)
			nn.Value = tnm + "(" + nn.Value + ")"
			ags[i] = nn
		case *ast.Ident:
			if gvar := p.GoToSL.GetTempVar(x.Name); gvar != nil {
				x.Name = "&" + x.Name
				ags[i] = x
			}
		case *ast.IndexExpr:
			isGlobal, tmpVar, _, _, isReadOnly := p.globalVar(x)
			if isGlobal {
				ags[i] = &ast.Ident{Name: tmpVar}
				if !isReadOnly {
					rwargs = append(rwargs, rwArg{idx: x, tmpVar: tmpVar})
				}
			}
		case *ast.UnaryExpr:
			if idx, ok := x.X.(*ast.IndexExpr); ok {
				isGlobal, tmpVar, _, _, isReadOnly := p.globalVar(idx)
				if isGlobal {
					ags[i] = &ast.Ident{Name: tmpVar}
					if !isReadOnly {
						rwargs = append(rwargs, rwArg{idx: idx, tmpVar: tmpVar})
					}
				}
			}
		}
	}
	return ags, rwargs
}

// gosl: ensure basic literals are properly cast
func (p *printer) matchLiteralType(x ast.Expr, typ *ast.Ident) bool {
	if lit, ok := x.(*ast.BasicLit); ok {
		p.print(typ.Name, token.LPAREN, normalizedNumber(lit), token.RPAREN)
		return true
	}
	return false
}

// gosl: ensure basic literals are properly cast
func (p *printer) matchAssignType(lhs []ast.Expr, rhs []ast.Expr) bool {
	if len(rhs) != 1 || len(lhs) != 1 {
		return false
	}
	val := ""
	lit, ok := rhs[0].(*ast.BasicLit)
	if ok {
		val = normalizedNumber(lit).Value
	} else {
		un, ok := rhs[0].(*ast.UnaryExpr)
		if !ok || un.Op != token.SUB {
			return false
		}
		lit, ok = un.X.(*ast.BasicLit)
		if !ok {
			return false
		}
		val = "-" + normalizedNumber(lit).Value
	}
	var err error
	var typ types.Type
	if id, ok := lhs[0].(*ast.Ident); ok {
		typ = p.getIdType(id)
		if typ == nil {
			return false
		}
	} else if sl, ok := lhs[0].(*ast.SelectorExpr); ok {
		typ, err = p.pathType(sl)
		if err != nil {
			return false
		}
	} else if st, ok := lhs[0].(*ast.StarExpr); ok {
		if id, ok := st.X.(*ast.Ident); ok {
			typ = p.getIdType(id)
			if typ == nil {
				return false
			}
		}
		if err != nil {
			return false
		}
	}
	if typ == nil {
		return false
	}
	tnm := getLocalTypeName(typ)
	if tnm[0] == '*' {
		tnm = tnm[1:]
	}
	p.print(tnm, "(", val, ")")
	return true
}

// gosl: pathType returns the final type for the selector path.
// a.b.c -> sel.X = (a.b) Sel=c -- returns type of c by tracing
// through the path.
func (p *printer) pathType(x *ast.SelectorExpr) (types.Type, error) {
	var paths []*ast.Ident
	cur := x
	for {
		paths = append(paths, cur.Sel)
		if sl, ok := cur.X.(*ast.SelectorExpr); ok { // path is itself a selector
			cur = sl
			continue
		}
		if id, ok := cur.X.(*ast.Ident); ok {
			paths = append(paths, id)
			break
		}
		return nil, fmt.Errorf("gosl pathType: path not a pure selector path")
	}
	np := len(paths)
	idt := p.getIdType(paths[np-1])
	if idt == nil {
		err := fmt.Errorf("gosl pathType ERROR: cannot find type for name: %q", paths[np-1].Name)
		p.userError(err)
		return nil, err
	}
	bt, err := p.getStructType(idt)
	if err != nil {
		return nil, err
	}
	for pi := np - 2; pi >= 0; pi-- {
		pt := paths[pi]
		f := fieldByName(bt, pt.Name)
		if f == nil {
			return nil, fmt.Errorf("gosl pathType: field not found %q in type: %q:", pt, bt.String())
		}
		if pi == 0 {
			return f.Type(), nil
		} else {
			bt, err = p.getStructType(f.Type())
			if err != nil {
				return nil, err
			}
		}
	}
	return nil, fmt.Errorf("gosl pathType: path not a pure selector path")
}

// gosl: check if identifier is a pointer arg
func (p *printer) isPtrArg(id *ast.Ident) bool {
	for _, pt := range p.curPtrArgs {
		if id.Name == pt.Name {
			return true
		}
	}
	return false
}

// gosl: dereference pointer vals
func (p *printer) derefPtrArgs(x ast.Expr, prec, depth int) {
	if id, ok := x.(*ast.Ident); ok {
		if p.isPtrArg(id) {
			p.print(token.LPAREN, token.MUL, id, token.RPAREN)
		} else {
			p.expr1(x, prec, depth)
		}
	} else {
		p.expr1(x, prec, depth)
	}
}

// gosl: mark pointer types, returns true if pointer
func (p *printer) ptrType(x ast.Expr) (ast.Expr, bool) {
	if u, ok := x.(*ast.StarExpr); ok {
		p.print("ptr<function", token.COMMA)
		return u.X, true
	}
	return x, false
}

// gosl: printMethRecv prints the method recv prefix for function. returns true if recv is ptr
func (p *printer) printMethRecv() (isPtr bool, typnm string) {
	if u, ok := p.curMethRecv.Type.(*ast.StarExpr); ok {
		typnm = u.X.(*ast.Ident).Name
		isPtr = true
	} else {
		typnm = p.curMethRecv.Type.(*ast.Ident).Name
	}
	return
}

// combinesWithName reports whether a name followed by the expression x
// syntactically combines to another valid (value) expression. For instance
// using *T for x, "name *T" syntactically appears as the expression x*T.
// On the other hand, using  P|Q or *P|~Q for x, "name P|Q" or name *P|~Q"
// cannot be combined into a valid (value) expression.
func combinesWithName(x ast.Expr) bool {
	switch x := x.(type) {
	case *ast.StarExpr:
		// name *x.X combines to name*x.X if x.X is not a type element
		return !isTypeElem(x.X)
	case *ast.BinaryExpr:
		return combinesWithName(x.X) && !isTypeElem(x.Y)
	case *ast.ParenExpr:
		// name(x) combines but we are making sure at
		// the call site that x is never parenthesized.
		panic("unexpected parenthesized expression")
	}
	return false
}

// isTypeElem reports whether x is a (possibly parenthesized) type element expression.
// The result is false if x could be a type element OR an ordinary (value) expression.
func isTypeElem(x ast.Expr) bool {
	switch x := x.(type) {
	case *ast.ArrayType, *ast.StructType, *ast.FuncType, *ast.InterfaceType, *ast.MapType, *ast.ChanType:
		return true
	case *ast.UnaryExpr:
		return x.Op == token.TILDE
	case *ast.BinaryExpr:
		return isTypeElem(x.X) || isTypeElem(x.Y)
	case *ast.ParenExpr:
		return isTypeElem(x.X)
	}
	return false
}

func (p *printer) signature(sig *ast.FuncType, recv *ast.FieldList) {
	if sig.TypeParams != nil {
		p.parameters(sig.TypeParams, funcTParam)
	}
	if sig.Params != nil {
		if recv != nil {
			flist := &ast.FieldList{}
			*flist = *recv
			flist.List = append(flist.List, sig.Params.List...)
			p.parameters(flist, funcParam)
		} else {
			p.parameters(sig.Params, funcParam)
		}
	} else if recv != nil {
		p.parameters(recv, funcParam)
	} else {
		p.print(token.LPAREN, token.RPAREN)
	}
	res := sig.Results
	n := res.NumFields()
	if n > 0 {
		// res != nil
		if id, ok := res.List[0].Type.(*ast.Ident); ok {
			p.curReturnType = id
		}
		p.print(blank, "->", blank)
		if n == 1 && res.List[0].Names == nil {
			// single anonymous res; no ()'s
			p.expr(stripParensAlways(res.List[0].Type))
			return
		}
		p.parameters(res, funcParam)
	}
}

func identListSize(list []*ast.Ident, maxSize int) (size int) {
	for i, x := range list {
		if i > 0 {
			size += len(", ")
		}
		size += utf8.RuneCountInString(x.Name)
		if size >= maxSize {
			break
		}
	}
	return
}

func (p *printer) isOneLineFieldList(list []*ast.Field) bool {
	if len(list) != 1 {
		return false // allow only one field
	}
	f := list[0]
	if f.Tag != nil || f.Comment != nil {
		return false // don't allow tags or comments
	}
	// only name(s) and type
	const maxSize = 30 // adjust as appropriate, this is an approximate value
	namesSize := identListSize(f.Names, maxSize)
	if namesSize > 0 {
		namesSize = 1 // blank between names and types
	}
	typeSize := p.nodeSize(f.Type, maxSize)
	return namesSize+typeSize <= maxSize
}

func (p *printer) setLineComment(text string) {
	p.setComment(&ast.CommentGroup{List: []*ast.Comment{{Slash: token.NoPos, Text: text}}})
}

func (p *printer) fieldList(fields *ast.FieldList, isStruct, isIncomplete bool) {
	lbrace := fields.Opening
	list := fields.List
	rbrace := fields.Closing
	hasComments := isIncomplete || p.commentBefore(p.posFor(rbrace))
	srcIsOneLine := lbrace.IsValid() && rbrace.IsValid() && p.lineFor(lbrace) == p.lineFor(rbrace)

	if !hasComments && srcIsOneLine {
		// possibly a one-line struct/interface
		if len(list) == 0 {
			// no blank between keyword and {} in this case
			p.setPos(lbrace)
			p.print(token.LBRACE)
			p.setPos(rbrace)
			p.print(token.RBRACE)
			return
		} else if p.isOneLineFieldList(list) {
			// small enough - print on one line
			// (don't use identList and ignore source line breaks)
			p.setPos(lbrace)
			p.print(token.LBRACE, blank)
			f := list[0]
			if isStruct {
				for i, x := range f.Names {
					if i > 0 {
						// no comments so no need for comma position
						p.print(token.COMMA, blank)
					}
					p.expr(x)
				}
				p.print(token.COLON)
				if len(f.Names) > 0 {
					p.print(blank)
				}
				p.expr(f.Type)
			} else { // interface
				if len(f.Names) > 0 {
					name := f.Names[0] // method name
					p.expr(name)
					p.print(token.COLON)
					p.signature(f.Type.(*ast.FuncType), nil) // don't print "func"
				} else {
					// embedded interface
					p.expr(f.Type)
				}
			}
			p.print(blank)
			p.setPos(rbrace)
			p.print(token.RBRACE)
			return
		}
	}
	// hasComments || !srcIsOneLine

	p.print(blank)
	p.setPos(lbrace)
	p.print(token.LBRACE, indent)
	if hasComments || len(list) > 0 {
		p.print(formfeed)
	}

	if isStruct {

		sep := vtab
		if len(list) == 1 {
			sep = blank
		}
		var line int
		for i, f := range list {
			if i > 0 {
				p.linebreak(p.lineFor(f.Pos()), 1, ignore, p.linesFrom(line) > 0)
			}
			extraTabs := 0
			p.setComment(f.Doc)
			p.recordLine(&line)
			if len(f.Names) > 1 {
				nnm := len(f.Names)
				p.setPos(f.Type.Pos())
				for ni, nm := range f.Names {
					p.print(nm.Name)
					p.print(token.COLON)
					p.print(sep)
					p.expr(f.Type)
					if ni < nnm-1 {
						p.print(token.COMMA)
						p.print(formfeed)
					}
				}
				extraTabs = 1
			} else if len(f.Names) > 0 {
				// named fields
				p.identList(f.Names, false)
				p.print(token.COLON)
				p.print(sep)
				p.expr(f.Type)
				extraTabs = 1
			} else {
				// anonymous field
				p.expr(f.Type)
				extraTabs = 2
			}
			p.print(token.COMMA)
			// if f.Tag != nil {
			// 	if len(f.Names) > 0 && sep == vtab {
			// 		p.print(sep)
			// 	}
			// 	p.print(sep)
			// 	p.expr(f.Tag)
			// 	extraTabs = 0
			// }
			if f.Comment != nil {
				for ; extraTabs > 0; extraTabs-- {
					p.print(sep)
				}
				p.setComment(f.Comment)
			}
		}
		if isIncomplete {
			if len(list) > 0 {
				p.print(formfeed)
			}
			p.flush(p.posFor(rbrace), token.RBRACE) // make sure we don't lose the last line comment
			p.setLineComment("// " + filteredMsg)
		}

	} else { // interface

		var line int
		var prev *ast.Ident // previous "type" identifier
		for i, f := range list {
			var name *ast.Ident // first name, or nil
			if len(f.Names) > 0 {
				name = f.Names[0]
			}
			if i > 0 {
				// don't do a line break (min == 0) if we are printing a list of types
				// TODO(gri) this doesn't work quite right if the list of types is
				//           spread across multiple lines
				min := 1
				if prev != nil && name == prev {
					min = 0
				}
				p.linebreak(p.lineFor(f.Pos()), min, ignore, p.linesFrom(line) > 0)
			}
			p.setComment(f.Doc)
			p.recordLine(&line)
			if name != nil {
				// method
				p.expr(name)
				p.signature(f.Type.(*ast.FuncType), nil) // don't print "func"
				prev = nil
			} else {
				// embedded interface
				p.expr(f.Type)
				prev = nil
			}
			p.setComment(f.Comment)
		}
		if isIncomplete {
			if len(list) > 0 {
				p.print(formfeed)
			}
			p.flush(p.posFor(rbrace), token.RBRACE) // make sure we don't lose the last line comment
			p.setLineComment("// contains filtered or unexported methods")
		}

	}
	p.print(unindent, formfeed)
	p.setPos(rbrace)
	p.print(token.RBRACE)
}

// ----------------------------------------------------------------------------
// Expressions

func walkBinary(e *ast.BinaryExpr) (has4, has5 bool, maxProblem int) {
	switch e.Op.Precedence() {
	case 4:
		has4 = true
	case 5:
		has5 = true
	}

	switch l := e.X.(type) {
	case *ast.BinaryExpr:
		if l.Op.Precedence() < e.Op.Precedence() {
			// parens will be inserted.
			// pretend this is an *ast.ParenExpr and do nothing.
			break
		}
		h4, h5, mp := walkBinary(l)
		has4 = has4 || h4
		has5 = has5 || h5
		maxProblem = max(maxProblem, mp)
	}

	switch r := e.Y.(type) {
	case *ast.BinaryExpr:
		if r.Op.Precedence() <= e.Op.Precedence() {
			// parens will be inserted.
			// pretend this is an *ast.ParenExpr and do nothing.
			break
		}
		h4, h5, mp := walkBinary(r)
		has4 = has4 || h4
		has5 = has5 || h5
		maxProblem = max(maxProblem, mp)

	case *ast.StarExpr:
		if e.Op == token.QUO { // `*/`
			maxProblem = 5
		}

	case *ast.UnaryExpr:
		switch e.Op.String() + r.Op.String() {
		case "/*", "&&", "&^":
			maxProblem = 5
		case "++", "--":
			maxProblem = max(maxProblem, 4)
		}
	}
	return
}

func cutoff(e *ast.BinaryExpr, depth int) int {
	has4, has5, maxProblem := walkBinary(e)
	if maxProblem > 0 {
		return maxProblem + 1
	}
	if has4 && has5 {
		if depth == 1 {
			return 5
		}
		return 4
	}
	if depth == 1 {
		return 6
	}
	return 4
}

func diffPrec(expr ast.Expr, prec int) int {
	x, ok := expr.(*ast.BinaryExpr)
	if !ok || prec != x.Op.Precedence() {
		return 1
	}
	return 0
}

func reduceDepth(depth int) int {
	depth--
	if depth < 1 {
		depth = 1
	}
	return depth
}

// Format the binary expression: decide the cutoff and then format.
// Let's call depth == 1 Normal mode, and depth > 1 Compact mode.
// (Algorithm suggestion by Russ Cox.)
//
// The precedences are:
//
//	5             *  /  %  <<  >>  &  &^
//	4             +  -  |  ^
//	3             ==  !=  <  <=  >  >=
//	2             &&
//	1             ||
//
// The only decision is whether there will be spaces around levels 4 and 5.
// There are never spaces at level 6 (unary), and always spaces at levels 3 and below.
//
// To choose the cutoff, look at the whole expression but excluding primary
// expressions (function calls, parenthesized exprs), and apply these rules:
//
//  1. If there is a binary operator with a right side unary operand
//     that would clash without a space, the cutoff must be (in order):
//
//     /*	6
//     &&	6
//     &^	6
//     ++	5
//     --	5
//
//     (Comparison operators always have spaces around them.)
//
//  2. If there is a mix of level 5 and level 4 operators, then the cutoff
//     is 5 (use spaces to distinguish precedence) in Normal mode
//     and 4 (never use spaces) in Compact mode.
//
//  3. If there are no level 4 operators or no level 5 operators, then the
//     cutoff is 6 (always use spaces) in Normal mode
//     and 4 (never use spaces) in Compact mode.
func (p *printer) binaryExpr(x *ast.BinaryExpr, prec1, cutoff, depth int) {
	prec := x.Op.Precedence()
	if prec < prec1 {
		// parenthesis needed
		// Note: The gotoslr inserts an ast.ParenExpr node; thus this case
		//       can only occur if the AST is created in a different way.
		p.print(token.LPAREN)
		p.expr0(x, reduceDepth(depth)) // parentheses undo one level of depth
		p.print(token.RPAREN)
		return
	}

	printBlank := prec < cutoff

	ws := indent
	p.expr1(x.X, prec, depth+diffPrec(x.X, prec))
	if printBlank {
		p.print(blank)
	}
	xline := p.pos.Line // before the operator (it may be on the next line!)
	yline := p.lineFor(x.Y.Pos())
	p.setPos(x.OpPos)
	if x.Op == token.AND_NOT {
		p.print(token.AND, blank, token.TILDE)
	} else {
		p.print(x.Op)
	}
	if xline != yline && xline > 0 && yline > 0 {
		// at least one line break, but respect an extra empty line
		// in the source
		if p.linebreak(yline, 1, ws, true) > 0 {
			ws = ignore
			printBlank = false // no blank after line break
		}
	}
	if printBlank {
		p.print(blank)
	}
	p.expr1(x.Y, prec+1, depth+1)
	if ws == ignore {
		p.print(unindent)
	}
}

func isBinary(expr ast.Expr) bool {
	_, ok := expr.(*ast.BinaryExpr)
	return ok
}

func (p *printer) expr1(expr ast.Expr, prec1, depth int) {
	p.setPos(expr.Pos())

	switch x := expr.(type) {
	case *ast.BadExpr:
		p.print("BadExpr")

	case *ast.Ident:
		if x.Name == "int" {
			p.print("i32")
		} else {
			p.print(x)
		}

	case *ast.BinaryExpr:
		if depth < 1 {
			p.internalError("depth < 1:", depth)
			depth = 1
		}
		p.binaryExpr(x, prec1, cutoff(x, depth), depth)

	case *ast.KeyValueExpr:
		p.expr(x.Key)
		p.setPos(x.Colon)
		p.print(token.COLON, blank)
		p.expr(x.Value)

	case *ast.StarExpr:
		const prec = token.UnaryPrec
		if prec < prec1 {
			// parenthesis needed
			p.print(token.LPAREN)
			p.print(token.MUL)
			p.expr(x.X)
			p.print(token.RPAREN)
		} else {
			// no parenthesis needed
			p.print(token.MUL)
			p.expr(x.X)
		}

	case *ast.UnaryExpr:
		const prec = token.UnaryPrec
		if prec < prec1 {
			// parenthesis needed
			p.print(token.LPAREN)
			p.expr(x)
			p.print(token.RPAREN)
		} else {
			// no parenthesis needed
			p.print(x.Op)
			if x.Op == token.RANGE {
				// TODO(gri) Remove this code if it cannot be reached.
				p.print(blank)
			}
			p.expr1(x.X, prec, depth)
		}

	case *ast.BasicLit:
		if p.PrintConfig.Mode&normalizeNumbers != 0 {
			x = normalizedNumber(x)
		}
		p.print(x)

	case *ast.FuncLit:
		p.setPos(x.Type.Pos())
		p.print(token.FUNC)
		// See the comment in funcDecl about how the header size is computed.
		startCol := p.out.Column - len("func")
		p.signature(x.Type, nil)
		p.funcBody(p.distanceFrom(x.Type.Pos(), startCol), blank, x.Body)

	case *ast.ParenExpr:
		if _, hasParens := x.X.(*ast.ParenExpr); hasParens {
			// don't print parentheses around an already parenthesized expression
			// TODO(gri) consider making this more general and incorporate precedence levels
			p.expr0(x.X, depth)
		} else {
			p.print(token.LPAREN)
			p.expr0(x.X, reduceDepth(depth)) // parentheses undo one level of depth
			p.setPos(x.Rparen)
			p.print(token.RPAREN)
		}

	case *ast.SelectorExpr:
		p.selectorExpr(x, depth)

	case *ast.TypeAssertExpr:
		p.expr1(x.X, token.HighestPrec, depth)
		p.print(token.PERIOD)
		p.setPos(x.Lparen)
		p.print(token.LPAREN)
		if x.Type != nil {
			p.expr(x.Type)
		} else {
			p.print(token.TYPE)
		}
		p.setPos(x.Rparen)
		p.print(token.RPAREN)

	case *ast.IndexExpr:
		// TODO(gri): should treat[] like parentheses and undo one level of depth
		p.expr1(x.X, token.HighestPrec, 1)
		p.setPos(x.Lbrack)
		p.print(token.LBRACK)
		p.expr0(x.Index, depth+1)
		p.setPos(x.Rbrack)
		p.print(token.RBRACK)

	case *ast.IndexListExpr:
		// TODO(gri): as for IndexExpr, should treat [] like parentheses and undo
		// one level of depth
		p.expr1(x.X, token.HighestPrec, 1)
		p.setPos(x.Lbrack)
		p.print(token.LBRACK)
		p.exprList(x.Lbrack, x.Indices, depth+1, commaTerm, x.Rbrack, false)
		p.setPos(x.Rbrack)
		p.print(token.RBRACK)

	case *ast.SliceExpr:
		// TODO(gri): should treat[] like parentheses and undo one level of depth
		p.expr1(x.X, token.HighestPrec, 1)
		p.setPos(x.Lbrack)
		p.print(token.LBRACK)
		indices := []ast.Expr{x.Low, x.High}
		if x.Max != nil {
			indices = append(indices, x.Max)
		}
		// determine if we need extra blanks around ':'
		var needsBlanks bool
		if depth <= 1 {
			var indexCount int
			var hasBinaries bool
			for _, x := range indices {
				if x != nil {
					indexCount++
					if isBinary(x) {
						hasBinaries = true
					}
				}
			}
			if indexCount > 1 && hasBinaries {
				needsBlanks = true
			}
		}
		for i, x := range indices {
			if i > 0 {
				if indices[i-1] != nil && needsBlanks {
					p.print(blank)
				}
				p.print(token.COLON)
				if x != nil && needsBlanks {
					p.print(blank)
				}
			}
			if x != nil {
				p.expr0(x, depth+1)
			}
		}
		p.setPos(x.Rbrack)
		p.print(token.RBRACK)

	case *ast.CallExpr:
		if len(x.Args) > 1 {
			depth++
		}
		fid, isid := x.Fun.(*ast.Ident)
		// Conversions to literal function types or <-chan
		// types require parentheses around the type.
		paren := false
		switch t := x.Fun.(type) {
		case *ast.FuncType:
			paren = true
		case *ast.ChanType:
			paren = t.Dir == ast.RECV
		}
		if paren {
			p.print(token.LPAREN)
		}
		if _, ok := x.Fun.(*ast.SelectorExpr); ok {
			p.methodExpr(x, depth)
			break // handles everything, break out of case
		}
		args := x.Args
		var rwargs []rwArg
		if isid {
			if p.curFunc != nil {
				p.curFunc.Funcs[fid.Name] = p.GoToSL.RecycleFunc(fid.Name)
			}
			if obj, ok := p.pkg.TypesInfo.Uses[fid]; ok {
				if ft, ok := obj.(*types.Func); ok {
					sig := ft.Type().(*types.Signature)
					args, rwargs = p.goslFixArgs(x.Args, sig.Params())
				}
			}
		}
		p.expr1(x.Fun, token.HighestPrec, depth)
		if paren {
			p.print(token.RPAREN)
		}
		p.setPos(x.Lparen)
		p.print(token.LPAREN)
		if x.Ellipsis.IsValid() {
			p.exprList(x.Lparen, args, depth, 0, x.Ellipsis, false)
			p.setPos(x.Ellipsis)
			p.print(token.ELLIPSIS)
			if x.Rparen.IsValid() && p.lineFor(x.Ellipsis) < p.lineFor(x.Rparen) {
				p.print(token.COMMA, formfeed)
			}
		} else {
			p.exprList(x.Lparen, args, depth, commaTerm, x.Rparen, false)
		}
		p.setPos(x.Rparen)
		p.print(token.RPAREN)
		p.assignRwArgs(rwargs)

	case *ast.CompositeLit:
		// composite literal elements that are composite literals themselves may have the type omitted
		if x.Type != nil {
			p.expr1(x.Type, token.HighestPrec, depth)
		}
		p.level++
		p.setPos(x.Lbrace)
		p.print(token.LBRACE)
		p.exprList(x.Lbrace, x.Elts, 1, commaTerm, x.Rbrace, x.Incomplete)
		// do not insert extra line break following a /*-style comment
		// before the closing '}' as it might break the code if there
		// is no trailing ','
		mode := noExtraLinebreak
		// do not insert extra blank following a /*-style comment
		// before the closing '}' unless the literal is empty
		if len(x.Elts) > 0 {
			mode |= noExtraBlank
		}
		// need the initial indent to print lone comments with
		// the proper level of indentation
		p.print(indent, unindent, mode)
		p.setPos(x.Rbrace)
		p.print(token.RBRACE, mode)
		p.level--

	case *ast.Ellipsis:
		p.print(token.ELLIPSIS)
		if x.Elt != nil {
			p.expr(x.Elt)
		}

	case *ast.ArrayType:
		p.print(token.LBRACK)
		if x.Len != nil {
			p.expr(x.Len)
		}
		p.print(token.RBRACK)
		p.expr(x.Elt)

	case *ast.StructType:
		// p.print(token.STRUCT)
		p.fieldList(x.Fields, true, x.Incomplete)

	case *ast.FuncType:
		p.print(token.FUNC)
		p.signature(x, nil)

	case *ast.InterfaceType:
		p.print(token.INTERFACE)
		p.fieldList(x.Methods, false, x.Incomplete)

	case *ast.MapType:
		p.print(token.MAP, token.LBRACK)
		p.expr(x.Key)
		p.print(token.RBRACK)
		p.expr(x.Value)

	case *ast.ChanType:
		switch x.Dir {
		case ast.SEND | ast.RECV:
			p.print(token.CHAN)
		case ast.RECV:
			p.print(token.ARROW, token.CHAN) // x.Arrow and x.Pos() are the same
		case ast.SEND:
			p.print(token.CHAN)
			p.setPos(x.Arrow)
			p.print(token.ARROW)
		}
		p.print(blank)
		p.expr(x.Value)

	default:
		panic("unreachable")
	}
}

// normalizedNumber rewrites base prefixes and exponents
// of numbers to use lower-case letters (0X123 to 0x123 and 1.2E3 to 1.2e3),
// and removes leading 0's from integer imaginary literals (0765i to 765i).
// It leaves hexadecimal digits alone.
//
// normalizedNumber doesn't modify the ast.BasicLit value lit points to.
// If lit is not a number or a number in canonical format already,
// lit is returned as is. Otherwise a new ast.BasicLit is created.
func normalizedNumber(lit *ast.BasicLit) *ast.BasicLit {
	if lit.Kind != token.INT && lit.Kind != token.FLOAT && lit.Kind != token.IMAG {
		return lit // not a number - nothing to do
	}
	if len(lit.Value) < 2 {
		return lit // only one digit (common case) - nothing to do
	}
	// len(lit.Value) >= 2

	// We ignore lit.Kind because for lit.Kind == token.IMAG the literal may be an integer
	// or floating-point value, decimal or not. Instead, just consider the literal pattern.
	x := lit.Value
	switch x[:2] {
	default:
		// 0-prefix octal, decimal int, or float (possibly with 'i' suffix)
		if i := strings.LastIndexByte(x, 'E'); i >= 0 {
			x = x[:i] + "e" + x[i+1:]
			break
		}
		// remove leading 0's from integer (but not floating-point) imaginary literals
		if x[len(x)-1] == 'i' && !strings.ContainsAny(x, ".e") {
			x = strings.TrimLeft(x, "0_")
			if x == "i" {
				x = "0i"
			}
		}
	case "0X":
		x = "0x" + x[2:]
		// possibly a hexadecimal float
		if i := strings.LastIndexByte(x, 'P'); i >= 0 {
			x = x[:i] + "p" + x[i+1:]
		}
	case "0x":
		// possibly a hexadecimal float
		i := strings.LastIndexByte(x, 'P')
		if i == -1 {
			return lit // nothing to do
		}
		x = x[:i] + "p" + x[i+1:]
	case "0O":
		x = "0o" + x[2:]
	case "0o":
		return lit // nothing to do
	case "0B":
		x = "0b" + x[2:]
	case "0b":
		return lit // nothing to do
	}

	return &ast.BasicLit{ValuePos: lit.ValuePos, Kind: lit.Kind, Value: x}
}

// selectorExpr handles an *ast.SelectorExpr node and reports whether x spans
// multiple lines, and thus was indented.
func (p *printer) selectorExpr(x *ast.SelectorExpr, depth int) (wasIndented bool) {
	p.derefPtrArgs(x.X, token.HighestPrec, depth)
	p.print(token.PERIOD)
	if line := p.lineFor(x.Sel.Pos()); p.pos.IsValid() && p.pos.Line < line {
		p.print(indent, newline)
		p.setPos(x.Sel.Pos())
		p.print(x.Sel)
		p.print(unindent)
		return true
	}
	p.setPos(x.Sel.Pos())
	p.print(x.Sel)
	return false
}

// gosl: methodExpr needs to deal with possible multiple chains of selector exprs
// to determine the actual type and name of the receiver.
// a.b.c() -> sel.X = (a.b) Sel=c
func (p *printer) methodPath(x *ast.SelectorExpr) (recvPath, recvType string, pathType types.Type, err error) {
	var baseRecv *ast.Ident // first receiver in path
	var paths []string
	cur := x
	for {
		paths = append(paths, cur.Sel.Name)
		if sl, ok := cur.X.(*ast.SelectorExpr); ok { // path is itself a selector
			cur = sl
			continue
		}
		if id, ok := cur.X.(*ast.Ident); ok {
			baseRecv = id
			break
		}
		err = fmt.Errorf("gosl methodPath ERROR: path for method call must be simple list of fields, not %#v:", cur.X)
		p.userError(err)
		return
	}
	if p.isPtrArg(baseRecv) {
		recvPath = "&(*" + baseRecv.Name + ")"
	} else {
		recvPath = "&" + baseRecv.Name
	}
	var idt types.Type
	if gvar := p.GoToSL.GetTempVar(baseRecv.Name); gvar != nil {
		var id ast.Ident
		id = *baseRecv
		id.Name = gvar.Var.SLType()
		// fmt.Println("type name:", id.Name)
		obj := p.pkg.Types.Scope().Lookup(id.Name)
		if obj != nil {
			idt = obj.Type()
		}
	} else {
		idt = p.getIdType(baseRecv)
	}
	if idt == nil {
		err = fmt.Errorf("gosl methodPath ERROR: cannot find type for name: %q", baseRecv.Name)
		p.userError(err)
		return
	}
	bt, err := p.getStructType(idt)
	if err != nil {
		fmt.Println(baseRecv)
		return
	}
	curt := bt
	np := len(paths)
	for pi := np - 1; pi >= 0; pi-- {
		pth := paths[pi]
		recvPath += "." + pth
		f := fieldByName(curt, pth)
		if f == nil {
			err = fmt.Errorf("gosl ERROR: field not found %q in type: %q:", pth, curt.String())
			p.userError(err)
			return
		}
		if pi == 0 {
			pathType = f.Type()
			recvType = getLocalTypeName(f.Type())
		} else {
			curt, err = p.getStructType(f.Type())
			if err != nil {
				return
			}
		}
	}
	return
}

func fieldByName(st *types.Struct, name string) *types.Var {
	nf := st.NumFields()
	for i := range nf {
		f := st.Field(i)
		if f.Name() == name {
			return f
		}
	}
	return nil
}

func (p *printer) getIdType(id *ast.Ident) types.Type {
	if obj, ok := p.pkg.TypesInfo.Uses[id]; ok {
		return obj.Type()
	}
	return nil
}

func getLocalTypeName(typ types.Type) string {
	_, nm := path.Split(typ.String())
	return nm
}

func (p *printer) getStructType(typ types.Type) (*types.Struct, error) {
	typ = typ.Underlying()
	if st, ok := typ.(*types.Struct); ok {
		return st, nil
	}
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem().Underlying()
		if st, ok := typ.(*types.Struct); ok {
			return st, nil
		}
	}
	if sl, ok := typ.(*types.Slice); ok {
		typ = sl.Elem().Underlying()
		if st, ok := typ.(*types.Struct); ok {
			return st, nil
		}
	}
	err := fmt.Errorf("gosl ERROR: type is not a struct and it should be: %q %+t", typ.String(), typ)
	p.userError(err)
	return nil, err
}

func (p *printer) getNamedType(typ types.Type) (*types.Named, error) {
	if nmd, ok := typ.(*types.Named); ok {
		return nmd, nil
	}
	typ = typ.Underlying()
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
		if nmd, ok := typ.(*types.Named); ok {
			return nmd, nil
		}
	}
	if sl, ok := typ.(*types.Slice); ok {
		typ = sl.Elem()
		if nmd, ok := typ.(*types.Named); ok {
			return nmd, nil
		}
	}
	err := fmt.Errorf("gosl ERROR: type is not a named type: %q %+t", typ.String(), typ)
	p.userError(err)
	return nil, err
}

// gosl: globalVar looks up whether the id in an IndexExpr is a global gosl variable.
// in which case it returns a temp variable name to use, and the type info.
func (p *printer) globalVar(idx *ast.IndexExpr) (isGlobal bool, tmpVar, typName string, vtyp types.Type, isReadOnly bool) {
	id, ok := idx.X.(*ast.Ident)
	if !ok {
		return
	}
	gvr := p.GoToSL.GlobalVar(id.Name)
	if gvr == nil {
		return
	}
	isGlobal = true
	isReadOnly = gvr.ReadOnly
	tmpVar = strings.ToLower(id.Name)
	vtyp = p.getIdType(id)
	if vtyp == nil {
		err := fmt.Errorf("gosl globalVar ERROR: cannot find type for name: %q", id.Name)
		p.userError(err)
		return
	}
	nmd, err := p.getNamedType(vtyp)
	if err == nil {
		vtyp = nmd
	}
	typName = gvr.SLType()
	p.print("var ", tmpVar, token.ASSIGN)
	p.expr(idx)
	p.print(token.SEMICOLON, blank)
	tmpVar = "&" + tmpVar
	return
}

// gosl: replace GetVar function call with assignment of local var
func (p *printer) getGlobalVar(ae *ast.AssignStmt, gvr *Var) {
	tmpVar := ae.Lhs[0].(*ast.Ident).Name
	cf := ae.Rhs[0].(*ast.CallExpr)
	p.print("var", blank, tmpVar, blank, token.ASSIGN, blank, gvr.Name, token.LBRACK)
	p.expr(cf.Args[0])
	p.print(token.RBRACK, token.SEMICOLON)
	gvars := p.GoToSL.GetVarStack.Peek()
	gvars[tmpVar] = &GetGlobalVar{Var: gvr, TmpVar: tmpVar, IdxExpr: cf.Args[0]}
	p.GoToSL.GetVarStack[len(p.GoToSL.GetVarStack)-1] = gvars
}

// gosl: set non-read-only global vars back from temp var
func (p *printer) setGlobalVars(gvrs map[string]*GetGlobalVar) {
	for _, gvr := range gvrs {
		if gvr.Var.ReadOnly {
			continue
		}
		p.print(formfeed, "\t")
		p.print(gvr.Var.Name, token.LBRACK)
		p.expr(gvr.IdxExpr)
		p.print(token.RBRACK, blank, token.ASSIGN, blank)
		tmpVar := strings.ToLower(gvr.Var.Name)
		p.print(tmpVar)
		p.print(token.SEMICOLON)
	}
}

// gosl: methodIndex processes an index expression as receiver type of method call
func (p *printer) methodIndex(idx *ast.IndexExpr) (recvPath, recvType string, pathType types.Type, isReadOnly bool, err error) {
	id, ok := idx.X.(*ast.Ident)
	if !ok {
		err = fmt.Errorf("gosl methodIndex ERROR: must have a recv variable identifier, not %#v:", idx.X)
		p.userError(err)
		return
	}
	isGlobal, tmpVar, typName, vtyp, isReadOnly := p.globalVar(idx)
	if isGlobal {
		recvPath = tmpVar
		recvType = typName
		pathType = vtyp
	} else {
		_ = id
		// do above
	}
	return
}

func (p *printer) tensorMethod(x *ast.CallExpr, vr *Var, methName string) {
	args := x.Args

	stArg := 0
	if strings.HasPrefix(methName, "Set") {
		stArg = 1
	}
	p.print(vr.Name, token.LBRACK)
	p.print(vr.IndexFunc(), token.LPAREN)
	nd := vr.TensorDims
	for d := range nd {
		p.print(vr.Name, token.LBRACK, strconv.Itoa(d), token.RBRACK, token.COMMA, blank)
	}
	n := len(args)
	for i := stArg; i < n; i++ {
		ag := args[i]
		p.print("u32", token.LPAREN)
		if ce, ok := ag.(*ast.CallExpr); ok { // get rid of int() wrapper from goal n-dim index
			if fn, ok := ce.Fun.(*ast.Ident); ok {
				if fn.Name == "int" {
					ag = ce.Args[0]
				}
			}
		}
		p.expr(ag)
		p.print(token.RPAREN)
		if i < n-1 {
			p.print(token.COMMA)
		}
	}
	p.print(token.RPAREN, token.RBRACK)
	if strings.HasPrefix(methName, "Set") {
		opnm := strings.TrimPrefix(methName, "Set")
		tok := token.ASSIGN
		switch opnm {
		case "Add":
			tok = token.ADD_ASSIGN
		case "Sub":
			tok = token.SUB_ASSIGN
		case "Mul":
			tok = token.MUL_ASSIGN
		case "Div":
			tok = token.QUO_ASSIGN
		}

		p.print(blank, tok, blank)
		p.expr(args[0])
	}
}

func (p *printer) methodExpr(x *ast.CallExpr, depth int) {
	path := x.Fun.(*ast.SelectorExpr) // we know fun is selector
	methName := path.Sel.Name
	recvPath := ""
	recvType := ""
	var err error
	pathIsPackage := false
	var rwargs []rwArg
	var pathType types.Type
	if sl, ok := path.X.(*ast.SelectorExpr); ok { // path is itself a selector
		recvPath, recvType, pathType, err = p.methodPath(sl)
		if err != nil {
			return
		}
	} else if id, ok := path.X.(*ast.Ident); ok {
		gvr := p.GoToSL.GlobalVar(id.Name)
		if gvr != nil && gvr.Tensor {
			p.tensorMethod(x, gvr, methName)
			return
		}
		recvPath = id.Name
		typ := p.getIdType(id)
		if typ != nil {
			recvType = getLocalTypeName(typ)
			if strings.HasPrefix(recvType, "invalid") {
				if gvar := p.GoToSL.GetTempVar(id.Name); gvar != nil {
					recvType = gvar.Var.SLType()
					recvPath = "&" + recvPath
				} else {
					pathIsPackage = true
					recvType = id.Name // is a package path
				}
			} else {
				pathType = typ
				recvPath = recvPath
			}
		} else {
			pathIsPackage = true
			recvType = id.Name // is a package path
		}
	} else if idx, ok := path.X.(*ast.IndexExpr); ok {
		isReadOnly := false
		recvPath, recvType, pathType, isReadOnly, err = p.methodIndex(idx)
		if err != nil {
			return
		}
		if !isReadOnly {
			rwargs = append(rwargs, rwArg{idx: idx, tmpVar: recvPath})
		}
	} else {
		err := fmt.Errorf("gosl methodExpr ERROR: path expression for method call must be simple list of fields, not %#v:", path.X)
		p.userError(err)
		return
	}
	args := x.Args
	if pathType != nil {
		meth, _, _ := types.LookupFieldOrMethod(pathType, true, p.pkg.Types, methName)
		if meth != nil {
			if ft, ok := meth.(*types.Func); ok {
				sig := ft.Type().(*types.Signature)
				var rwa []rwArg
				args, rwa = p.goslFixArgs(x.Args, sig.Params())
				rwargs = append(rwargs, rwa...)
			}
		}
		if len(rwargs) > 0 {
			p.print(formfeed)
		}
	}
	// fmt.Println(pathIsPackage, recvType, methName, recvPath)
	if pathIsPackage {
		if recvType == "atomic" {
			switch methName {
			case "AddInt32":
				p.print("atomicAdd")
			}
		} else {
			p.print(recvType + "." + methName)
		}
		p.setPos(x.Lparen)
		p.print(token.LPAREN)
	} else {
		recvType = strings.TrimPrefix(recvType, "imports.") // no!
		fname := recvType + "_" + methName
		if p.curFunc != nil {
			p.curFunc.Funcs[fname] = p.GoToSL.RecycleFunc(fname)
		}
		p.print(fname)
		p.setPos(x.Lparen)
		p.print(token.LPAREN)
		p.print(recvPath)
		if len(x.Args) > 0 {
			p.print(token.COMMA, blank)
		}
	}
	if x.Ellipsis.IsValid() {
		p.exprList(x.Lparen, args, depth, 0, x.Ellipsis, false)
		p.setPos(x.Ellipsis)
		p.print(token.ELLIPSIS)
		if x.Rparen.IsValid() && p.lineFor(x.Ellipsis) < p.lineFor(x.Rparen) {
			p.print(token.COMMA, formfeed)
		}
	} else {
		p.exprList(x.Lparen, args, depth, commaTerm, x.Rparen, false)
	}
	p.setPos(x.Rparen)
	p.print(token.RPAREN)

	p.assignRwArgs(rwargs) // gosl: assign temp var back to global var
}

func (p *printer) expr0(x ast.Expr, depth int) {
	p.expr1(x, token.LowestPrec, depth)
}

func (p *printer) expr(x ast.Expr) {
	const depth = 1
	p.expr1(x, token.LowestPrec, depth)
}

// ----------------------------------------------------------------------------
// Statements

// Print the statement list indented, but without a newline after the last statement.
// Extra line breaks between statements in the source are respected but at most one
// empty line is printed between statements.
func (p *printer) stmtList(list []ast.Stmt, nindent int, nextIsRBrace bool) {
	if nindent > 0 {
		p.print(indent)
	}
	var line int
	i := 0
	for _, s := range list {
		// ignore empty statements (was issue 3466)
		if _, isEmpty := s.(*ast.EmptyStmt); !isEmpty {
			// nindent == 0 only for lists of switch/select case clauses;
			// in those cases each clause is a new section
			if len(p.output) > 0 {
				// only print line break if we are not at the beginning of the output
				// (i.e., we are not printing only a partial program)
				p.linebreak(p.lineFor(s.Pos()), 1, ignore, i == 0 || nindent == 0 || p.linesFrom(line) > 0)
			}
			p.recordLine(&line)
			p.stmt(s, nextIsRBrace && i == len(list)-1, false)
			// labeled statements put labels on a separate line, but here
			// we only care about the start line of the actual statement
			// without label - correct line for each label
			for t := s; ; {
				lt, _ := t.(*ast.LabeledStmt)
				if lt == nil {
					break
				}
				line++
				t = lt.Stmt
			}
			i++
		}
	}
	if nindent > 0 {
		p.print(unindent)
	}
}

// block prints an *ast.BlockStmt; it always spans at least two lines.
func (p *printer) block(b *ast.BlockStmt, nindent int) {
	p.GoToSL.GetVarStack.Push(make(map[string]*GetGlobalVar))
	p.setPos(b.Lbrace)
	p.print(token.LBRACE)
	p.stmtList(b.List, nindent, true)
	getVars := p.GoToSL.GetVarStack.Pop()
	if len(getVars) > 0 { // gosl: set the get vars
		p.setGlobalVars(getVars)
	}
	p.linebreak(p.lineFor(b.Rbrace), 1, ignore, true)
	p.setPos(b.Rbrace)
	p.print(token.RBRACE)
}

func isTypeName(x ast.Expr) bool {
	switch t := x.(type) {
	case *ast.Ident:
		return true
	case *ast.SelectorExpr:
		return isTypeName(t.X)
	}
	return false
}

func stripParens(x ast.Expr) ast.Expr {
	if px, strip := x.(*ast.ParenExpr); strip {
		// parentheses must not be stripped if there are any
		// unparenthesized composite literals starting with
		// a type name
		ast.Inspect(px.X, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.ParenExpr:
				// parentheses protect enclosed composite literals
				return false
			case *ast.CompositeLit:
				if isTypeName(x.Type) {
					strip = false // do not strip parentheses
				}
				return false
			}
			// in all other cases, keep inspecting
			return true
		})
		if strip {
			return stripParens(px.X)
		}
	}
	return x
}

func stripParensAlways(x ast.Expr) ast.Expr {
	if x, ok := x.(*ast.ParenExpr); ok {
		return stripParensAlways(x.X)
	}
	return x
}

func (p *printer) controlClause(isForStmt bool, init ast.Stmt, expr ast.Expr, post ast.Stmt) {
	p.print(blank)
	p.print(token.LPAREN)
	needsBlank := false
	if init == nil && post == nil {
		// no semicolons required
		if expr != nil {
			p.expr(stripParens(expr))
			needsBlank = true
		}
	} else {
		// all semicolons required
		// (they are not separators, print them explicitly)
		if init != nil {
			p.stmt(init, false, false) // false = generate own semi
			p.print(blank)
		} else {
			p.print(token.SEMICOLON, blank)
		}
		if expr != nil {
			p.expr(stripParens(expr))
			needsBlank = true
		}
		if isForStmt {
			p.print(token.SEMICOLON, blank)
			needsBlank = false
			if post != nil {
				p.stmt(post, false, true) // nosemi
				needsBlank = true
			}
		}
	}
	p.print(token.RPAREN)
	if needsBlank {
		p.print(blank)
	}
}

// indentList reports whether an expression list would look better if it
// were indented wholesale (starting with the very first element, rather
// than starting at the first line break).
func (p *printer) indentList(list []ast.Expr) bool {
	// Heuristic: indentList reports whether there are more than one multi-
	// line element in the list, or if there is any element that is not
	// starting on the same line as the previous one ends.
	if len(list) >= 2 {
		var b = p.lineFor(list[0].Pos())
		var e = p.lineFor(list[len(list)-1].End())
		if 0 < b && b < e {
			// list spans multiple lines
			n := 0 // multi-line element count
			line := b
			for _, x := range list {
				xb := p.lineFor(x.Pos())
				xe := p.lineFor(x.End())
				if line < xb {
					// x is not starting on the same
					// line as the previous one ended
					return true
				}
				if xb < xe {
					// x is a multi-line element
					n++
				}
				line = xe
			}
			return n > 1
		}
	}
	return false
}

func (p *printer) stmt(stmt ast.Stmt, nextIsRBrace bool, nosemi bool) {
	p.setPos(stmt.Pos())

	switch s := stmt.(type) {
	case *ast.BadStmt:
		p.print("BadStmt")

	case *ast.DeclStmt:
		p.decl(s.Decl)

	case *ast.EmptyStmt:
		// nothing to do

	case *ast.LabeledStmt:
		// a "correcting" unindent immediately following a line break
		// is applied before the line break if there is no comment
		// between (see writeWhitespace)
		p.print(unindent)
		p.expr(s.Label)
		p.setPos(s.Colon)
		p.print(token.COLON, indent)
		if e, isEmpty := s.Stmt.(*ast.EmptyStmt); isEmpty {
			if !nextIsRBrace {
				p.print(newline)
				p.setPos(e.Pos())
				p.print(token.SEMICOLON)
				break
			}
		} else {
			p.linebreak(p.lineFor(s.Stmt.Pos()), 1, ignore, true)
		}
		p.stmt(s.Stmt, nextIsRBrace, nosemi)

	case *ast.ExprStmt:
		const depth = 1
		p.expr0(s.X, depth)
		if !nosemi {
			p.print(token.SEMICOLON)
		}

	case *ast.SendStmt:
		const depth = 1
		p.expr0(s.Chan, depth)
		p.print(blank)
		p.setPos(s.Arrow)
		p.print(token.ARROW, blank)
		p.expr0(s.Value, depth)

	case *ast.IncDecStmt:
		const depth = 1
		p.expr0(s.X, depth+1)
		p.setPos(s.TokPos)
		p.print(s.Tok)
		if !nosemi {
			p.print(token.SEMICOLON)
		}

	case *ast.AssignStmt:
		var depth = 1
		if len(s.Lhs) > 1 && len(s.Rhs) > 1 {
			depth++
		}
		if s.Tok == token.DEFINE {
			if ce, ok := s.Rhs[0].(*ast.CallExpr); ok {
				if fid, ok := ce.Fun.(*ast.Ident); ok {
					if strings.HasPrefix(fid.Name, "Get") {
						if gvr, ok := p.GoToSL.GetFuncs[fid.Name]; ok {
							p.getGlobalVar(s, gvr) // replace GetVar function call with assignment of local var
							return
						}
					}
				}
			}
			p.print("var", blank) // we don't know if it is var or let..
		}
		p.exprList(s.Pos(), s.Lhs, depth, 0, s.TokPos, false)
		p.print(blank)
		p.setPos(s.TokPos)
		switch s.Tok {
		case token.DEFINE:
			p.print(token.ASSIGN, blank)
		case token.AND_NOT_ASSIGN:
			p.print(token.AND_ASSIGN, blank, "~")
		default:
			p.print(s.Tok, blank)
		}
		if p.matchAssignType(s.Lhs, s.Rhs) {
		} else {
			p.exprList(s.TokPos, s.Rhs, depth, 0, token.NoPos, false)
		}
		if !nosemi {
			p.print(token.SEMICOLON)
		}

	case *ast.GoStmt:
		p.print(token.GO, blank)
		p.expr(s.Call)

	case *ast.DeferStmt:
		p.print(token.DEFER, blank)
		p.expr(s.Call)

	case *ast.ReturnStmt:
		p.print(token.RETURN)
		if s.Results != nil {
			p.print(blank)
			if !p.matchLiteralType(s.Results[0], p.curReturnType) {
				// Use indentList heuristic to make corner cases look
				// better (issue 1207). A more systematic approach would
				// always indent, but this would cause significant
				// reformatting of the code base and not necessarily
				// lead to more nicely formatted code in general.
				if p.indentList(s.Results) {
					p.print(indent)
					// Use NoPos so that a newline never goes before
					// the results (see issue #32854).
					p.exprList(token.NoPos, s.Results, 1, noIndent, token.NoPos, false)
					p.print(unindent)
				} else {
					p.exprList(token.NoPos, s.Results, 1, 0, token.NoPos, false)
				}
			}
		}
		if !nosemi {
			p.print(token.SEMICOLON)
		}

	case *ast.BranchStmt:
		p.print(s.Tok)
		if s.Label != nil {
			p.print(blank)
			p.expr(s.Label)
		}
		p.print(token.SEMICOLON)

	case *ast.BlockStmt:
		p.block(s, 1)

	case *ast.IfStmt:
		p.print(token.IF)
		p.controlClause(false, s.Init, s.Cond, nil)
		p.block(s.Body, 1)
		if s.Else != nil {
			p.print(blank, token.ELSE, blank)
			switch s.Else.(type) {
			case *ast.BlockStmt, *ast.IfStmt:
				p.stmt(s.Else, nextIsRBrace, false)
			default:
				// This can only happen with an incorrectly
				// constructed AST. Permit it but print so
				// that it can be gotosld without errors.
				p.print(token.LBRACE, indent, formfeed)
				p.stmt(s.Else, true, false)
				p.print(unindent, formfeed, token.RBRACE)
			}
		}

	case *ast.CaseClause:
		if s.List != nil {
			p.print(token.CASE, blank)
			p.exprList(s.Pos(), s.List, 1, 0, s.Colon, false)
		} else {
			p.print(token.DEFAULT)
		}
		p.setPos(s.Colon)
		p.print(token.COLON, blank, token.LBRACE) // Go implies new context, C doesn't
		p.stmtList(s.Body, 1, nextIsRBrace)
		p.print(formfeed, token.RBRACE)

	case *ast.SwitchStmt:
		p.print(token.SWITCH)
		p.controlClause(false, s.Init, s.Tag, nil)
		p.block(s.Body, 0)

	case *ast.TypeSwitchStmt:
		p.print(token.SWITCH)
		if s.Init != nil {
			p.print(blank)
			p.stmt(s.Init, false, false)
			p.print(token.SEMICOLON)
		}
		p.print(blank)
		p.stmt(s.Assign, false, false)
		p.print(blank)
		p.block(s.Body, 0)

	case *ast.CommClause:
		if s.Comm != nil {
			p.print(token.CASE, blank)
			p.stmt(s.Comm, false, false)
		} else {
			p.print(token.DEFAULT)
		}
		p.setPos(s.Colon)
		p.print(token.COLON)
		p.stmtList(s.Body, 1, nextIsRBrace)

	case *ast.SelectStmt:
		p.print(token.SELECT, blank)
		body := s.Body
		if len(body.List) == 0 && !p.commentBefore(p.posFor(body.Rbrace)) {
			// print empty select statement w/o comments on one line
			p.setPos(body.Lbrace)
			p.print(token.LBRACE)
			p.setPos(body.Rbrace)
			p.print(token.RBRACE)
		} else {
			p.block(body, 0)
		}

	case *ast.ForStmt:
		p.print(token.FOR)
		p.controlClause(true, s.Init, s.Cond, s.Post)
		p.block(s.Body, 1)

	case *ast.RangeStmt:
		p.print(token.FOR, blank)
		if s.Key != nil {
			p.expr(s.Key)
			if s.Value != nil {
				// use position of value following the comma as
				// comma position for correct comment placement
				p.setPos(s.Value.Pos())
				p.print(token.COMMA, blank)
				p.expr(s.Value)
			}
			p.print(blank)
			p.setPos(s.TokPos)
			p.print(s.Tok, blank)
		}
		p.print(token.RANGE, blank)
		p.expr(stripParens(s.X))
		p.print(blank)
		p.block(s.Body, 1)

	default:
		panic("unreachable")
	}
}

// ----------------------------------------------------------------------------
// Declarations

// The keepTypeColumn function determines if the type column of a series of
// consecutive const or var declarations must be kept, or if initialization
// values (V) can be placed in the type column (T) instead. The i'th entry
// in the result slice is true if the type column in spec[i] must be kept.
//
// For example, the declaration:
//
//		const (
//			foobar int = 42 // comment
//			x          = 7  // comment
//			foo
//	             bar = 991
//		)
//
// leads to the type/values matrix below. A run of value columns (V) can
// be moved into the type column if there is no type for any of the values
// in that column (we only move entire columns so that they align properly).
//
//		matrix        formatted     result
//	                   matrix
//		T  V    ->    T  V     ->   true      there is a T and so the type
//		-  V          -  V          true      column must be kept
//		-  -          -  -          false
//		-  V          V  -          false     V is moved into T column
func keepTypeColumn(specs []ast.Spec) []bool {
	m := make([]bool, len(specs))

	populate := func(i, j int, keepType bool) {
		if keepType {
			for ; i < j; i++ {
				m[i] = true
			}
		}
	}

	i0 := -1 // if i0 >= 0 we are in a run and i0 is the start of the run
	var keepType bool
	for i, s := range specs {
		t := s.(*ast.ValueSpec)
		if t.Values != nil {
			if i0 < 0 {
				// start of a run of ValueSpecs with non-nil Values
				i0 = i
				keepType = false
			}
		} else {
			if i0 >= 0 {
				// end of a run
				populate(i0, i, keepType)
				i0 = -1
			}
		}
		if t.Type != nil {
			keepType = true
		}
	}
	if i0 >= 0 {
		// end of a run
		populate(i0, len(specs), keepType)
	}

	return m
}

func (p *printer) valueSpec(s *ast.ValueSpec, keepType bool, tok token.Token, firstSpec *ast.ValueSpec, isIota bool, idx int) {
	p.setComment(s.Doc)

	// gosl: key to use Pos() as first arg to trigger emitting of comments!
	switch tok {
	case token.CONST:
		p.setPos(s.Pos())
		p.print(tok, blank)
	case token.TYPE:
		p.setPos(s.Pos())
		p.print("alias", blank)
	}
	p.print(vtab)

	extraTabs := 3
	p.identList(s.Names, false) // always present
	if isIota {
		if s.Type != nil {
			p.print(token.COLON, blank)
			p.expr(s.Type)
		} else if firstSpec.Type != nil {
			p.print(token.COLON, blank)
			p.expr(firstSpec.Type)
		}
		p.print(vtab, token.ASSIGN, blank)
		p.print(fmt.Sprintf("%d", idx))
	} else if s.Type != nil || keepType {
		p.print(token.COLON, blank)
		p.expr(s.Type)
		extraTabs--
	} else if tok == token.CONST && firstSpec.Type != nil {
		p.expr(firstSpec.Type)
		extraTabs--
	}

	if !(isIota && s == firstSpec) && s.Values != nil {
		p.print(vtab, token.ASSIGN, blank)
		p.exprList(token.NoPos, s.Values, 1, 0, token.NoPos, false)
		extraTabs--
	}
	p.print(token.SEMICOLON)
	if s.Comment != nil {
		for ; extraTabs > 0; extraTabs-- {
			p.print(vtab)
		}
		p.setComment(s.Comment)
	}
}

func sanitizeImportPath(lit *ast.BasicLit) *ast.BasicLit {
	// Note: An unmodified AST generated by go/gotoslr will already
	// contain a backward- or double-quoted path string that does
	// not contain any invalid characters, and most of the work
	// here is not needed. However, a modified or generated AST
	// may possibly contain non-canonical paths. Do the work in
	// all cases since it's not too hard and not speed-critical.

	// if we don't have a proper string, be conservative and return whatever we have
	if lit.Kind != token.STRING {
		return lit
	}
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return lit
	}

	// if the string is an invalid path, return whatever we have
	//
	// spec: "Implementation restriction: A compiler may restrict
	// ImportPaths to non-empty strings using only characters belonging
	// to Unicode's L, M, N, P, and S general categories (the Graphic
	// characters without spaces) and may also exclude the characters
	// !"#$%&'()*,:;<=>?[\]^`{|} and the Unicode replacement character
	// U+FFFD."
	if s == "" {
		return lit
	}
	const illegalChars = `!"#$%&'()*,:;<=>?[\]^{|}` + "`\uFFFD"
	for _, r := range s {
		if !unicode.IsGraphic(r) || unicode.IsSpace(r) || strings.ContainsRune(illegalChars, r) {
			return lit
		}
	}

	// otherwise, return the double-quoted path
	s = strconv.Quote(s)
	if s == lit.Value {
		return lit // nothing wrong with lit
	}
	return &ast.BasicLit{ValuePos: lit.ValuePos, Kind: token.STRING, Value: s}
}

// The parameter n is the number of specs in the group. If doIndent is set,
// multi-line identifier lists in the spec are indented when the first
// linebreak is encountered.
func (p *printer) spec(spec ast.Spec, n int, doIndent bool, tok token.Token) {
	switch s := spec.(type) {
	case *ast.ImportSpec:
		p.setComment(s.Doc)
		if s.Name != nil {
			p.expr(s.Name)
			p.print(blank)
		}
		p.expr(sanitizeImportPath(s.Path))
		p.setComment(s.Comment)
		p.setPos(s.EndPos)

	case *ast.ValueSpec:
		if n != 1 {
			p.internalError("expected n = 1; got", n)
		}
		p.setComment(s.Doc)

		if len(s.Names) > 1 {
			nnm := len(s.Names)
			for ni, nm := range s.Names {
				p.print(tok, blank)
				p.print(nm.Name)
				if s.Type != nil {
					p.print(token.COLON, blank)
					p.expr(s.Type)
				}
				if s.Values != nil {
					p.print(blank, token.ASSIGN, blank)
					p.exprList(token.NoPos, s.Values, 1, 0, token.NoPos, false)
				}
				p.print(token.SEMICOLON)
				if ni < nnm-1 {
					p.print(formfeed)
				}
			}
		} else {
			p.print(tok, blank)
			p.identList(s.Names, doIndent) // always present
			if s.Type != nil {
				p.print(token.COLON, blank)
				p.expr(s.Type)
			}
			if s.Values != nil {
				p.print(blank, token.ASSIGN, blank)
				p.exprList(token.NoPos, s.Values, 1, 0, token.NoPos, false)
			}
			p.print(token.SEMICOLON)
			p.setComment(s.Comment)
		}

	case *ast.TypeSpec:
		p.setComment(s.Doc)
		st, isStruct := s.Type.(*ast.StructType)
		if isStruct {
			p.setPos(st.Pos())
			p.print(token.STRUCT, blank)
		} else {
			p.print("alias", blank)
		}
		p.expr(s.Name)
		if !isStruct {
			p.print(blank, token.ASSIGN, blank)
		}
		if s.TypeParams != nil {
			p.parameters(s.TypeParams, typeTParam)
		}
		// if n == 1 {
		// 	p.print(blank)
		// } else {
		// 	p.print(vtab)
		// }
		if s.Assign.IsValid() {
			p.print(token.ASSIGN, blank)
		}
		p.expr(s.Type)
		if !isStruct {
			p.print(token.SEMICOLON)
		}
		p.setComment(s.Comment)

	default:
		panic("unreachable")
	}
}

// gosl: process system global vars
func (p *printer) systemVars(d *ast.GenDecl, sysname string) {
	if !p.GoToSL.GetFuncGraph {
		return
	}
	sy := p.GoToSL.System(sysname)
	var gp *Group
	var err error
	for _, s := range d.Specs {
		vs := s.(*ast.ValueSpec)
		dirs, docs := p.findDirective(vs.Doc)
		readOnly := false
		if hasDirective(dirs, "read-only") {
			readOnly = true
		}
		if gpnm, ok := directiveAfter(dirs, "group"); ok {
			if gpnm == "" {
				gp = &Group{Name: fmt.Sprintf("Group_%d", len(sy.Groups)), Doc: docs}
				sy.Groups = append(sy.Groups, gp)
			} else {
				gps := strings.Fields(gpnm)
				gp = &Group{Doc: docs}
				if gps[0] == "-uniform" {
					gp.Uniform = true
					if len(gps) > 1 {
						gp.Name = gps[1]
					}
				} else {
					gp.Name = gps[0]
				}
				sy.Groups = append(sy.Groups, gp)
			}
		}
		if gp == nil {
			gp = &Group{Name: fmt.Sprintf("Group_%d", len(sy.Groups)), Doc: docs}
			sy.Groups = append(sy.Groups, gp)
		}
		if len(vs.Names) != 1 {
			err = fmt.Errorf("gosl: system %q: vars must have only 1 variable per line", sysname)
			p.userError(err)
		}
		nm := vs.Names[0].Name
		typ := ""
		if sl, ok := vs.Type.(*ast.ArrayType); ok {
			id, ok := sl.Elt.(*ast.Ident)
			if !ok {
				err = fmt.Errorf("gosl: system %q: Var type not recognized: %#v", sysname, sl.Elt)
				p.userError(err)
				continue
			}
			typ = "[]" + id.Name
		} else {
			sel, ok := vs.Type.(*ast.SelectorExpr)
			if !ok {
				st, ok := vs.Type.(*ast.StarExpr)
				if !ok {
					err = fmt.Errorf("gosl: system %q: Var types must be []slices or tensor.Float32,  tensor.Uint32", sysname)
					p.userError(err)

					continue
				}
				sel, ok = st.X.(*ast.SelectorExpr)
				if !ok {
					err = fmt.Errorf("gosl: system %q: Var types must be []slices or tensor.Float32,  tensor.Uint32", sysname)
					p.userError(err)
					continue
				}
			}
			sid, ok := sel.X.(*ast.Ident)
			if !ok {
				err = fmt.Errorf("gosl: system %q: Var type selector is not recognized: %#v", sysname, sel.X)
				p.userError(err)
				continue
			}
			typ = sid.Name + "." + sel.Sel.Name
		}
		vr := &Var{Name: nm, Type: typ, ReadOnly: readOnly}
		if strings.HasPrefix(typ, "tensor.") {
			vr.Tensor = true
			dstr, ok := directiveAfter(dirs, "dims")
			if !ok {
				err = fmt.Errorf("gosl: system %q: variable %q tensor vars require //gosl:dims <n> to specify number of dimensions", sysname, nm)
				p.userError(err)
				continue
			}
			dims, err := strconv.Atoi(dstr)
			if !ok {
				err = fmt.Errorf("gosl: system %q: variable %q tensor dims parse error: %s", sysname, nm, err.Error())
				p.userError(err)
			}
			vr.SetTensorKind()
			vr.TensorDims = dims
		}
		gp.Vars = append(gp.Vars, vr)
		if p.GoToSL.Config.Debug {
			fmt.Println("\tAdded var:", nm, typ, "to group:", gp.Name)
		}
	}
	p.GoToSL.VarsAdded()
}

func (p *printer) genDecl(d *ast.GenDecl) {
	p.setComment(d.Doc)
	// note: critical to print here to trigger comment generation in right place
	p.setPos(d.Pos())
	if d.Tok == token.IMPORT {
		return
	}
	// p.print(d.Pos(), d.Tok, blank)
	p.print(ignore) // don't print

	if d.Lparen.IsValid() || len(d.Specs) != 1 {
		// group of parenthesized declarations
		// p.setPos(d.Lparen)
		// p.print(token.LPAREN)
		if n := len(d.Specs); n > 0 {
			// p.print(indent, formfeed)
			if n > 1 && (d.Tok == token.CONST || d.Tok == token.VAR) {
				// two or more grouped const/var declarations:
				if d.Tok == token.VAR {
					dirs, _ := p.findDirective(d.Doc)
					if sysname, ok := directiveAfter(dirs, "vars"); ok {
						p.systemVars(d, sysname)
						return
					}
				}
				// determine if the type column must be kept
				keepType := keepTypeColumn(d.Specs)
				firstSpec := d.Specs[0].(*ast.ValueSpec)
				isIota := false
				if d.Tok == token.CONST {
					if id, isId := firstSpec.Values[0].(*ast.Ident); isId {
						if id.Name == "iota" {
							isIota = true
						}
					}
				}
				var line int
				for i, s := range d.Specs {
					vs := s.(*ast.ValueSpec)
					if i > 0 {
						p.linebreak(p.lineFor(s.Pos()), 1, ignore, p.linesFrom(line) > 0)
					}
					p.recordLine(&line)
					p.valueSpec(vs, keepType[i], d.Tok, firstSpec, isIota, i)
				}
			} else {
				var line int
				for i, s := range d.Specs {
					if i > 0 {
						p.linebreak(p.lineFor(s.Pos()), 1, ignore, p.linesFrom(line) > 0)
					}
					p.recordLine(&line)
					p.spec(s, n, false, d.Tok)
				}
			}
			// p.print(unindent, formfeed)
		}
		// p.setPos(d.Rparen)
		// p.print(token.RPAREN)
	} else if len(d.Specs) > 0 {
		// single declaration
		p.spec(d.Specs[0], 1, true, d.Tok)
	}
}

// sizeCounter is an io.Writer which counts the number of bytes written,
// as well as whether a newline character was seen.
type sizeCounter struct {
	hasNewline bool
	size       int
}

func (c *sizeCounter) Write(p []byte) (int, error) {
	if !c.hasNewline {
		for _, b := range p {
			if b == '\n' || b == '\f' {
				c.hasNewline = true
				break
			}
		}
	}
	c.size += len(p)
	return len(p), nil
}

// nodeSize determines the size of n in chars after formatting.
// The result is <= maxSize if the node fits on one line with at
// most maxSize chars and the formatted output doesn't contain
// any control chars. Otherwise, the result is > maxSize.
func (p *printer) nodeSize(n ast.Node, maxSize int) (size int) {
	// nodeSize invokes the printer, which may invoke nodeSize
	// recursively. For deep composite literal nests, this can
	// lead to an exponential algorithm. Remember previous
	// results to prune the recursion (was issue 1628).
	if size, found := p.nodeSizes[n]; found {
		return size
	}

	size = maxSize + 1 // assume n doesn't fit
	p.nodeSizes[n] = size

	// nodeSize computation must be independent of particular
	// style so that we always get the same decision; print
	// in RawFormat
	cfg := PrintConfig{Mode: RawFormat}
	var counter sizeCounter
	if err := cfg.fprint(&counter, p.pkg, n, p.nodeSizes); err != nil {
		return
	}
	if counter.size <= maxSize && !counter.hasNewline {
		// n fits in a single line
		size = counter.size
		p.nodeSizes[n] = size
	}
	return
}

// numLines returns the number of lines spanned by node n in the original source.
func (p *printer) numLines(n ast.Node) int {
	if from := n.Pos(); from.IsValid() {
		if to := n.End(); to.IsValid() {
			return p.lineFor(to) - p.lineFor(from) + 1
		}
	}
	return infinity
}

// bodySize is like nodeSize but it is specialized for *ast.BlockStmt's.
func (p *printer) bodySize(b *ast.BlockStmt, maxSize int) int {
	pos1 := b.Pos()
	pos2 := b.Rbrace
	if pos1.IsValid() && pos2.IsValid() && p.lineFor(pos1) != p.lineFor(pos2) {
		// opening and closing brace are on different lines - don't make it a one-liner
		return maxSize + 1
	}
	if len(b.List) > 5 {
		// too many statements - don't make it a one-liner
		return maxSize + 1
	}
	// otherwise, estimate body size
	bodySize := p.commentSizeBefore(p.posFor(pos2))
	for i, s := range b.List {
		if bodySize > maxSize {
			break // no need to continue
		}
		if i > 0 {
			bodySize += 2 // space for a semicolon and blank
		}
		bodySize += p.nodeSize(s, maxSize)
	}
	return bodySize
}

// funcBody prints a function body following a function header of given headerSize.
// If the header's and block's size are "small enough" and the block is "simple enough",
// the block is printed on the current line, without line breaks, spaced from the header
// by sep. Otherwise the block's opening "{" is printed on the current line, followed by
// lines for the block's statements and its closing "}".
func (p *printer) funcBody(headerSize int, sep whiteSpace, b *ast.BlockStmt) {
	if b == nil {
		return
	}

	// save/restore composite literal nesting level
	defer func(level int) {
		p.level = level
	}(p.level)
	p.level = 0

	const maxSize = 100
	if headerSize+p.bodySize(b, maxSize) <= maxSize {
		p.print(sep)
		p.setPos(b.Lbrace)
		p.print(token.LBRACE)
		if len(b.List) > 0 {
			p.print(blank)
			for i, s := range b.List {
				if i > 0 {
					p.print(token.SEMICOLON, blank)
				}
				p.stmt(s, i == len(b.List)-1, false)
			}
			p.print(blank)
		}
		p.print(noExtraLinebreak)
		p.setPos(b.Rbrace)
		p.print(token.RBRACE, noExtraLinebreak)
		return
	}

	if sep != ignore {
		p.print(blank) // always use blank
	}
	p.block(b, 1)
}

// distanceFrom returns the column difference between p.out (the current output
// position) and startOutCol. If the start position is on a different line from
// the current position (or either is unknown), the result is infinity.
func (p *printer) distanceFrom(startPos token.Pos, startOutCol int) int {
	if startPos.IsValid() && p.pos.IsValid() && p.posFor(startPos).Line == p.pos.Line {
		return p.out.Column - startOutCol
	}
	return infinity
}

func (p *printer) methRecvType(typ ast.Expr) string {
	switch x := typ.(type) {
	case *ast.StarExpr:
		return p.methRecvType(x.X)
	case *ast.Ident:
		return x.Name
	default:
		return fmt.Sprintf("recv type unknown: %+T", x)
	}
	return ""
}

func (p *printer) funcDecl(d *ast.FuncDecl) {
	fname := ""
	if d.Recv != nil {
		for ex := range p.ExcludeFunctions {
			if d.Name.Name == ex {
				return
			}
		}
		if d.Recv.List[0].Names != nil {
			p.curMethRecv = d.Recv.List[0]
			isptr, typnm := p.printMethRecv()
			if isptr {
				p.curPtrArgs = []*ast.Ident{p.curMethRecv.Names[0]}
			}
			fname = typnm + "_" + d.Name.Name
			// fmt.Printf("cur func recv: %v\n", p.curMethRecv)
		}
		// p.parameters(d.Recv, funcParam) // method: print receiver
		// p.print(blank)
	} else {
		fname = d.Name.Name
	}
	if p.GoToSL.GetFuncGraph {
		p.curFunc = p.GoToSL.RecycleFunc(fname)
	} else {
		_, ok := p.GoToSL.KernelFuncs[fname]
		if !ok {
			return
		}
	}
	p.setComment(d.Doc)
	p.setPos(d.Pos())
	// We have to save startCol only after emitting FUNC; otherwise it can be on a
	// different line (all whitespace preceding the FUNC is emitted only when the
	// FUNC is emitted).
	startCol := p.out.Column - len("func ")
	p.print("fn", blank, fname)
	p.signature(d.Type, d.Recv)
	p.funcBody(p.distanceFrom(d.Pos(), startCol), vtab, d.Body)
	p.curPtrArgs = nil
	p.curMethRecv = nil
	if p.GoToSL.GetFuncGraph {
		p.GoToSL.FuncGraph[fname] = p.curFunc
		p.curFunc = nil
	}
}

func (p *printer) decl(decl ast.Decl) {
	switch d := decl.(type) {
	case *ast.BadDecl:
		p.setPos(d.Pos())
		p.print("BadDecl")
	case *ast.GenDecl:
		p.genDecl(d)
	case *ast.FuncDecl:
		p.funcDecl(d)
	default:
		panic("unreachable")
	}
}

// ----------------------------------------------------------------------------
// Files

func declToken(decl ast.Decl) (tok token.Token) {
	tok = token.ILLEGAL
	switch d := decl.(type) {
	case *ast.GenDecl:
		tok = d.Tok
	case *ast.FuncDecl:
		tok = token.FUNC
	}
	return
}

func (p *printer) declList(list []ast.Decl) {
	tok := token.ILLEGAL
	for _, d := range list {
		prev := tok
		tok = declToken(d)
		// If the declaration token changed (e.g., from CONST to TYPE)
		// or the next declaration has documentation associated with it,
		// print an empty line between top-level declarations.
		// (because p.linebreak is called with the position of d, which
		// is past any documentation, the minimum requirement is satisfied
		// even w/o the extra getDoc(d) nil-check - leave it in case the
		// linebreak logic improves - there's already a TODO).
		if len(p.output) > 0 {
			// only print line break if we are not at the beginning of the output
			// (i.e., we are not printing only a partial program)
			min := 1
			if prev != tok || getDoc(d) != nil {
				min = 2
			}
			// start a new section if the next declaration is a function
			// that spans multiple lines (see also issue #19544)
			p.linebreak(p.lineFor(d.Pos()), min, ignore, tok == token.FUNC && p.numLines(d) > 1)
		}
		p.decl(d)
	}
}

func (p *printer) file(src *ast.File) {
	p.setComment(src.Doc)
	p.setPos(src.Pos())
	p.print(token.PACKAGE, blank)
	p.expr(src.Name)
	p.declList(src.Decls)
	p.print(newline)
}
