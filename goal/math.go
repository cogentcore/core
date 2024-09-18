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
	mp.toks.AddTokens(mp.curToks[mp.curIdx])
	mp.curIdx++
}

func (gl *Goal) TranspileMath(toks Tokens, ln string) Tokens {
	// fmt.Println("in math")
	nt := len(toks)

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
		fmt.Println("lhs:", mp.lhs)
		mp.rhsToks = toks[assignIdx+1 : nt-1]
		mp.rhs = ln[toks[assignIdx+1].Pos-1 : toks[nt-1].Pos]
		fmt.Println("rhs:", mp.rhs)
		lex, err := parser.ParseExpr(mp.lhs)
		if err != nil {
			fmt.Println("lhs parse err:", err)
		}
		rex, err := parser.ParseExpr(mp.rhs)
		if err != nil {
			fmt.Println("rhs parse err:", err)
		}
		mp.assignStmt(toks[assignIdx], lex, rex)
	} else {
		mp.rhsToks = toks[0 : nt-1]
		mp.curToks = mp.rhsToks
		mp.rhs = ln[toks[0].Pos-1 : toks[nt-1].Pos]
		ex, err := parser.ParseExpr(mp.rhs)
		if err != nil {
			fmt.Println("expr parse err:", err)
		}
		fmt.Println("expr:", mp.rhs)
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

	}
}

func (mp *mathParse) binaryExpr(ex *ast.BinaryExpr) {
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
}
