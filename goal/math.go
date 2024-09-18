// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goal

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

type mathParse struct {
	toks    Tokens // output tokens
	lhs     string
	rhs     string // or the only one
	curToks Tokens // current source tokens we are parsing
	curIdx  int    //  current index in source tokens
	lhsToks Tokens
	rhsToks Tokens
}

func (mp *mathParse) addCur() {
	if len(mp.curToks) > mp.curIdx {
		mp.toks.AddTokens(mp.curToks[mp.curIdx])
		mp.curIdx++
		return
	}
	fmt.Println("out of toks:", mp.curToks)
}

func (gl *Goal) TranspileMath(toks Tokens, ln string) Tokens {
	nt := len(toks)
	// fmt.Println(nt, toks)

	mp := mathParse{}

	// expr can't do statements, so we need to find those ourselves
	assignIdx := -1
	for i, tk := range toks {
		if tk.Tok == token.ASSIGN || tk.Tok == token.DEFINE {
			assignIdx = i
			break
		}
	}
	if assignIdx >= 0 {
		mp.lhsToks = toks[0:assignIdx]
		mp.lhs = ln[toks[0].Pos-1 : toks[assignIdx].Pos-1]
		mp.rhsToks = toks[assignIdx+1 : nt]
		mp.rhs = ln[toks[assignIdx+1].Pos-1 : toks[nt-1].Pos]
		lex, err := parser.ParseExpr(mp.lhs)
		if err != nil {
			fmt.Println("lhs:", mp.lhs)
			fmt.Println("lhs parse err:", err)
		}
		rex, err := parser.ParseExpr(mp.rhs)
		if err != nil {
			fmt.Println("rhs:", mp.rhs)
			fmt.Println("rhs parse err:", err)
			fmt.Printf("%#v\n", rex)
		}
		mp.assignStmt(toks[assignIdx], lex, rex)
	} else {
		mp.rhsToks = toks[0:nt]
		mp.curToks = mp.rhsToks
		mp.rhs = ln[toks[0].Pos-1 : toks[nt-1].Pos]
		ex, err := parser.ParseExpr(mp.rhs)
		if err != nil {
			fmt.Println("expr:", mp.rhs)
			fmt.Println("expr parse err:", err)
		}
		mp.expr(ex)
	}

	return mp.toks
}

func (mp *mathParse) assignStmt(tok *Token, lex, rex ast.Expr) {
	mp.curToks = mp.lhsToks
	mp.expr(lex)
	mp.toks.AddTokens(tok)
	mp.curToks = mp.rhsToks
	mp.curIdx = 0
	mp.expr(rex)
}

func (mp *mathParse) expr(ex ast.Expr) {
	switch x := ex.(type) {
	case *ast.BadExpr:
		fmt.Println("bad!")

	case *ast.Ident:
		// fmt.Println("ident:", x.Name)
		mp.addCur()

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
	mp.toks.Add(token.IDENT, "tensor.CallOut")
	mp.toks.Add(token.LPAREN)
	mp.toks.Add(token.STRING, `"`+fn+`"`)
	mp.toks.Add(token.COMMA)
	mp.expr(ex.X)
	mp.toks.Add(token.COMMA)
	mp.curIdx++
	mp.expr(ex.Y)
	mp.toks.Add(token.RPAREN)
	mp.toks.Add(token.LBRACK)
	mp.toks.Add(token.INT, "0")
	mp.toks.Add(token.RBRACK)
}

func (mp *mathParse) basicLit(lit *ast.BasicLit) {
	switch lit.Kind {
	case token.INT:
		mp.toks.Add(token.IDENT, "tensor.NewIntScalar("+lit.Value+")")
		mp.curIdx++
	case token.FLOAT:
		mp.toks.Add(token.IDENT, "tensor.NewFloatScalar("+lit.Value+")")
		mp.curIdx++
	case token.STRING:
		mp.toks.Add(token.IDENT, "tensor.NewStringScalar("+lit.Value+")")
		mp.curIdx++
	}
}

func (mp *mathParse) selectorExpr(ex *ast.SelectorExpr) {
}

func (mp *mathParse) sliceExpr(se *ast.SliceExpr) {
	fmt.Println("slice expr", se)
}
