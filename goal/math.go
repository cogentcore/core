// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goal

import (
	"fmt"
	"go/ast"
	"go/token"

	"cogentcore.org/core/goal/mparse"
)

type mathParse struct {
	code string // code string
	toks Tokens // source tokens we are parsing
	idx  int    //  current index in source tokens
	out  Tokens // output tokens we generate
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

// TranspileMath does math mode transpiling. fullLine indicates code should be
// full statement(s).
func (gl *Goal) TranspileMath(toks Tokens, code string, fullLine bool) Tokens {
	nt := len(toks)
	if nt == 0 {
		return nil
	}
	// fmt.Println(nt, toks)

	str := code[toks[0].Pos-1 : toks[nt-1].Pos]
	mp := mathParse{toks: toks, code: code}

	if fullLine {
		stmts, err := mparse.ParseLine(str, mparse.AllErrors)
		if err != nil {
			fmt.Println("line code:", str)
			fmt.Println("parse err:", err)
		}
		mp.stmtList(stmts)
	} else {
		ex, err := mparse.ParseExpr(str, mparse.AllErrors)
		if err != nil {
			fmt.Println("expr:", str)
			fmt.Println("parse err:", err)
		}
		mp.expr(ex)
	}

	return mp.out
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
		fmt.Println("index!")

	case *ast.SliceExpr:
		mp.sliceExpr(x)
		// todo: we'll need to work on this!

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

func (mp *mathParse) selectorExpr(ex *ast.SelectorExpr) {
}

func (mp *mathParse) sliceExpr(se *ast.SliceExpr) {
	fmt.Println("slice expr", se)
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
