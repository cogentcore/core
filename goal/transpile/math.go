// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package transpile

import (
	"fmt"
	"go/ast"
	"go/token"
)

func MathParse(toks Tokens, code string, fullLine bool) Tokens {
	nt := len(toks)
	if nt == 0 {
		return nil
	}
	// fmt.Println(nt, toks)

	str := code[toks[0].Pos-1 : toks[nt-1].Pos]
	if toks[nt-1].Str != "" {
		str += toks[nt-1].Str[1:]
	}
	// fmt.Println(str)
	mp := mathParse{toks: toks, code: code}

	mods := AllErrors // | Trace

	if fullLine {
		stmts, err := ParseLine(str, mods)
		if err != nil {
			fmt.Println("line code:", str)
			fmt.Println("parse err:", err)
		}
		mp.stmtList(stmts)
	} else {
		ex, err := ParseExpr(str, mods)
		if err != nil {
			fmt.Println("expr:", str)
			fmt.Println("parse err:", err)
		}
		mp.expr(ex)
	}

	return mp.out
}

// mathParse has the parsing state
type mathParse struct {
	code string // code string
	toks Tokens // source tokens we are parsing
	idx  int    //  current index in source tokens
	out  Tokens // output tokens we generate

	// goLiteral means generate basic literals as standard go literals instead of
	// wrapping them in tensor constructors.  for inner expressions contstructing go
	// objects etc.
	goLiteral bool
}

// addToken adds output token and increments idx
func (mp *mathParse) addToken(tok token.Token) {
	mp.out.Add(tok)
	mp.idx++
}

func (mp *mathParse) addCur() {
	if len(mp.toks) > mp.idx {
		mp.out.AddTokens(mp.toks[mp.idx])
		mp.idx++
		return
	}
	fmt.Println("out of tokens!", mp.idx, mp.toks)
}

func (mp *mathParse) stmtList(sts []ast.Stmt) {
	for _, st := range sts {
		mp.stmt(st)
	}
}

func (mp *mathParse) stmt(st ast.Stmt) {
	if st == nil {
		return
	}
	switch x := st.(type) {
	case *ast.BadStmt:
		fmt.Println("bad stmt!")

	case *ast.DeclStmt:

	case *ast.ExprStmt:
		mp.expr(x.X)

	case *ast.SendStmt:
		mp.expr(x.Chan)
		mp.addToken(token.ARROW)
		mp.expr(x.Value)

	case *ast.IncDecStmt:
		mp.expr(x.X)
		mp.addToken(x.Tok)

	case *ast.AssignStmt:
		mp.exprList(x.Lhs)
		mp.addToken(x.Tok)
		mp.exprList(x.Rhs)

	case *ast.GoStmt:
		mp.addToken(token.GO)
		mp.callExpr(x.Call)

	case *ast.DeferStmt:
		mp.addToken(token.DEFER)
		mp.callExpr(x.Call)

	case *ast.ReturnStmt:
		mp.addToken(token.RETURN)
		mp.exprList(x.Results)

	case *ast.BranchStmt:
		mp.addToken(x.Tok)
		mp.ident(x.Label)

	case *ast.BlockStmt:
		mp.addToken(token.LBRACE)
		mp.stmtList(x.List)
		mp.addToken(token.RBRACE)

	case *ast.IfStmt:
		mp.addToken(token.IF)
		mp.stmt(x.Init)
		if x.Init != nil {
			mp.addToken(token.SEMICOLON)
		}
		mp.expr(x.Cond)
		if x.Body != nil {
			mp.addToken(token.LBRACE)
			mp.stmtList(x.Body.List)
			mp.addToken(token.RBRACE)
		}
		if x.Else != nil {
			mp.addToken(token.ELSE)
			mp.stmt(x.Else)
		}

		// TODO
		// CaseClause: SwitchStmt:, TypeSwitchStmt:, CommClause:, SelectStmt:, ForStmt:, RangeStmt:
	}
}

func (mp *mathParse) expr(ex ast.Expr) {
	if ex == nil {
		return
	}
	switch x := ex.(type) {
	case *ast.BadExpr:
		fmt.Println("bad expr!")

	case *ast.Ident:
		mp.ident(x)

	case *ast.BinaryExpr:
		mp.binaryExpr(x)

	case *ast.BasicLit:
		mp.basicLit(x)

	case *ast.FuncLit:

	case *ast.ParenExpr:

	case *ast.SelectorExpr:
		mp.selectorExpr(x)

	case *ast.TypeAssertExpr:

	case *ast.IndexListExpr:
		if x.X == nil { // array literal
			mp.arrayLiteral(x)
		} else {
			mp.indexListExpr(x)
		}

	case *ast.SliceExpr:
		// mp.sliceExpr(x)
		// todo: probably this is subsumed in indexListExpr

	case *ast.CallExpr:

	case *ast.ArrayType:
		// basically at this point we have a bad expression and
		// need to do our own parsing.
		// it is unclear if perhaps we just need to do that from the start.
		fmt.Println("array type:", x, x.Len)
		fmt.Printf("%#v\n", x.Len)
	}
}

func (mp *mathParse) exprList(ex []ast.Expr) {
	n := len(ex)
	if n == 0 {
		return
	}
	if n == 1 {
		mp.expr(ex[0])
		return
	}
	for i := range n {
		mp.expr(ex[i])
		if i < n-1 {
			mp.addToken(token.COMMA)
		}
	}
}

func (mp *mathParse) binaryExpr(ex *ast.BinaryExpr) {
	fn := ""
	switch ex.Op {
	case token.ADD:
		fn = "Add"
	case token.SUB:
		fn = "Sub"
	case token.MUL:
		fn = "Mul"
	case token.QUO:
		fn = "Div"
	}
	mp.out.Add(token.IDENT, "tensor.CallOut")
	mp.out.Add(token.LPAREN)
	mp.out.Add(token.STRING, `"`+fn+`"`)
	mp.out.Add(token.COMMA)
	mp.expr(ex.X)
	mp.out.Add(token.COMMA)
	mp.idx++
	mp.expr(ex.Y)
	mp.out.Add(token.RPAREN)
}

func (mp *mathParse) basicLit(lit *ast.BasicLit) {
	if mp.goLiteral {
		mp.out.Add(lit.Kind, lit.Value)
		return
	}
	switch lit.Kind {
	case token.INT:
		mp.out.Add(token.IDENT, "tensor.NewIntScalar("+lit.Value+")")
		mp.idx++
	case token.FLOAT:
		mp.out.Add(token.IDENT, "tensor.NewFloatScalar("+lit.Value+")")
		mp.idx++
	case token.STRING:
		mp.out.Add(token.IDENT, "tensor.NewStringScalar("+lit.Value+")")
		mp.idx++
	}
}

type funWrap struct {
	fun  string
	wrap string
}

// nis: NewIntScalar
var numpyProps = map[string]funWrap{
	"ndim":  {"NumDims()", "nis"},
	"len":   {"Len()", "nis"},
	"size":  {"Len()", "nis"},
	"shape": {"Shape().Sizes", "nifs"},
}

func (mp *mathParse) selectorExpr(ex *ast.SelectorExpr) {
	fw, ok := numpyProps[ex.Sel.Name]
	if !ok {
		mp.expr(ex.X)
		mp.addToken(token.PERIOD)
		mp.out.Add(token.IDENT, ex.Sel.Name)
		mp.idx++
		return
	}
	elip := false
	switch fw.wrap {
	case "nis":
		mp.out.Add(token.IDENT, "tensor.NewIntScalar")
	case "nfs":
		mp.out.Add(token.IDENT, "tensor.NewFloat64Scalar")
	case "nss":
		mp.out.Add(token.IDENT, "tensor.NewStringScalar")
	case "nifs":
		mp.out.Add(token.IDENT, "tensor.NewIntFromSlice")
		elip = true
	case "nffs":
		mp.out.Add(token.IDENT, "tensor.NewFloat64FromSlice")
		elip = true
	case "nsfs":
		mp.out.Add(token.IDENT, "tensor.NewStringFromSlice")
		elip = true
	}
	mp.out.Add(token.LPAREN)
	mp.expr(ex.X)
	mp.addToken(token.PERIOD)
	mp.out.Add(token.IDENT, fw.fun)
	mp.idx++
	if elip {
		mp.out.Add(token.ELLIPSIS)
	}
	mp.out.Add(token.RPAREN)
}

func (mp *mathParse) indexListExpr(il *ast.IndexListExpr) {
	// fmt.Println("slice expr", se)
}

func (mp *mathParse) arrayLiteral(il *ast.IndexListExpr) {
	kind := inferKindExprList(il.Indices)
	if kind == token.ILLEGAL {
		kind = token.FLOAT // default
	}
	// todo: look for sub-arrays etc.
	typ := "float64"
	fun := "Float"
	switch kind {
	case token.FLOAT:
	case token.INT:
		typ = "int"
		fun = "Int"
	case token.STRING:
		typ = "string"
		fun = "String"
	}
	mp.out.Add(token.IDENT, "tensor.New"+fun+"FromSlice")
	mp.out.Add(token.LPAREN)
	mp.out.Add(token.IDENT, "[]"+typ)
	mp.addToken(token.LBRACE)
	mp.goLiteral = true
	mp.exprList(il.Indices)
	mp.goLiteral = false
	mp.addToken(token.RBRACE)
	mp.addToken(token.ELLIPSIS)
	mp.out.Add(token.RPAREN, ")")
}

func (mp *mathParse) callExpr(ex *ast.CallExpr) {
}

func (mp *mathParse) ident(id *ast.Ident) {
	if id == nil {
		return
	}
	// fmt.Println("ident:", x.Name)
	mp.addCur()
}
