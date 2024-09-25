// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package transpile

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"cogentcore.org/core/base/stack"
	"cogentcore.org/core/tensor"
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
	// mp.trace = true

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

	if mp.idx != len(toks) {
		fmt.Println(code)
		fmt.Println(mp.out.Code())
		fmt.Printf("parsing error: index: %d != len(toks): %d\n", mp.idx, len(toks))
	}

	return mp.out
}

// funcInfo is info about the function being processed
type funcInfo struct {
	tensor.Func

	//	true if this function takes tensor args
	tensorArgs bool
}

// mathParse has the parsing state
type mathParse struct {
	code  string // code string
	toks  Tokens // source tokens we are parsing
	idx   int    // current index in source tokens -- critical to sync as we "use" source
	out   Tokens // output tokens we generate
	trace bool   // trace of parsing -- turn on to see alignment

	// stack of function info -- top of stack reflects the current function
	funcs stack.Stack[*funcInfo]
}

// startFunc is called when starting a new function -- sets context
func (mp *mathParse) startFunc(name string, tensorArgs bool) *funcInfo {
	fn := &funcInfo{}
	fn.Name = name
	fn.tensorArgs = tensorArgs
	mp.funcs.Push(fn)
	if name != "" {
		mp.out.Add(token.IDENT, name)
	}
	return fn
}

func (mp *mathParse) endFunc() {
	mp.funcs.Pop()
}

// addToken adds output token and increments idx
func (mp *mathParse) addToken(tok token.Token) {
	mp.out.Add(tok)
	if mp.trace {
		ctok := &Token{}
		if mp.idx < len(mp.toks) {
			ctok = mp.toks[mp.idx]
		}
		fmt.Printf("%d\ttok: %s \t replaces: %s\n", mp.idx, tok, ctok)
	}
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
		switch x.Tok {
		case token.DEFINE:
			mp.defineStmt(x)
		default:
			mp.assignStmt(x)
		}

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

	case *ast.UnaryExpr:
		mp.unaryExpr(x)

	case *ast.BinaryExpr:
		mp.binaryExpr(x)

	case *ast.BasicLit:
		mp.basicLit(x)

	case *ast.FuncLit:

	case *ast.ParenExpr:
		mp.addToken(token.LPAREN)
		mp.expr(x.X)
		mp.addToken(token.RPAREN)

	case *ast.SelectorExpr:
		mp.selectorExpr(x)

	case *ast.TypeAssertExpr:

	case *ast.IndexExpr:
		mp.indexExpr(x)

	case *ast.IndexListExpr:
		if x.X == nil { // array literal
			mp.arrayLiteral(x)
		} else {
			mp.indexListExpr(x)
		}

	case *ast.SliceExpr:
		mp.sliceExpr(x)

	case *ast.CallExpr:
		mp.callExpr(x)

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
	mp.startFunc("tensor.CallOut", true) // yes tensor args
	mp.out.Add(token.LPAREN)
	mp.out.Add(token.STRING, `"`+fn+`"`)
	mp.out.Add(token.COMMA)
	mp.expr(ex.X)
	mp.out.Add(token.COMMA)
	mp.idx++
	mp.expr(ex.Y)
	mp.out.Add(token.RPAREN)
	mp.endFunc()
}

func (mp *mathParse) unaryExpr(ex *ast.UnaryExpr) {
	mp.addToken(ex.Op)
	mp.expr(ex.X)
}

func (mp *mathParse) defineStmt(as *ast.AssignStmt) {
	mp.exprList(as.Lhs)
	mp.addToken(as.Tok)
	mp.startFunc("", true) // just to trigger tensor args
	mp.exprList(as.Rhs)
	mp.endFunc()
}

func (mp *mathParse) assignStmt(as *ast.AssignStmt) {
	if _, ok := as.Lhs[0].(*ast.Ident); ok {
		mp.exprList(as.Lhs)
		mp.addToken(as.Tok)
		mp.startFunc("", true) // just to trigger tensor args
		mp.exprList(as.Rhs)
		mp.endFunc()
		return
	}
	fn := ""
	switch as.Tok {
	case token.ASSIGN:
		fn = "Assign"
	case token.ADD_ASSIGN:
		fn = "AddAssign"
	case token.SUB_ASSIGN:
		fn = "SubAssign"
	case token.MUL_ASSIGN:
		fn = "MulAssign"
	case token.QUO_ASSIGN:
		fn = "DivAssign"
	}
	mp.startFunc("tensor.Call", true) // yes tensor args
	mp.out.Add(token.LPAREN)
	mp.out.Add(token.STRING, `"`+fn+`"`)
	mp.out.Add(token.COMMA)
	mp.exprList(as.Lhs)
	mp.out.Add(token.COMMA)
	mp.idx++
	mp.exprList(as.Rhs)
	mp.out.Add(token.RPAREN)
	mp.endFunc()
}

func (mp *mathParse) basicLit(lit *ast.BasicLit) {
	cfun := mp.funcs.Peek()
	if cfun != nil && cfun.tensorArgs {
		mp.tensorLit(lit)
		return
	}
	mp.out.Add(lit.Kind, lit.Value)
	if mp.trace {
		fmt.Printf("%d\ttok: %s literal\n", mp.idx, lit.Value)
	}
	mp.idx++
	return
}

func (mp *mathParse) tensorLit(lit *ast.BasicLit) {
	switch lit.Kind {
	case token.INT:
		mp.out.Add(token.IDENT, "tensor.NewIntScalar("+lit.Value+")")
		mp.idx++
	case token.FLOAT:
		mp.out.Add(token.IDENT, "tensor.NewFloat64Scalar("+lit.Value+")")
		mp.idx++
	case token.STRING:
		mp.out.Add(token.IDENT, "tensor.NewStringScalar("+lit.Value+")")
		mp.idx++
	}
}

// funWrap is a function wrapper for simple numpy property / functions
type funWrap struct {
	fun  string // function to call on tensor
	wrap string // code for wrapping function for results of call
}

// nis: NewIntScalar, niv: NewIntFromValues, etc
var numpyProps = map[string]funWrap{
	"ndim":  {"NumDims()", "nis"},
	"len":   {"Len()", "nis"},
	"size":  {"Len()", "nis"},
	"shape": {"Shape().Sizes", "niv"},
}

// tensorFunc outputs the wrapping function and whether it needs ellipsis
func (fw *funWrap) wrapFunc(mp *mathParse) bool {
	ellip := false
	wrapFun := fw.wrap
	switch fw.wrap {
	case "nis":
		wrapFun = "tensor.NewIntScalar"
	case "nfs":
		wrapFun = "tensor.NewFloat64Scalar"
	case "nss":
		wrapFun = "tensor.NewStringScalar"
	case "niv":
		wrapFun = "tensor.NewIntFromValues"
		ellip = true
	case "nfv":
		wrapFun = "tensor.NewFloat64FromValues"
		ellip = true
	case "nsv":
		wrapFun = "tensor.NewStringFromValues"
		ellip = true
	}
	mp.startFunc(wrapFun, false)
	mp.out.Add(token.LPAREN)
	return ellip
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
	ellip := fw.wrapFunc(mp)
	mp.expr(ex.X)
	mp.addToken(token.PERIOD)
	mp.out.Add(token.IDENT, fw.fun)
	mp.idx++
	if ellip {
		mp.out.Add(token.ELLIPSIS)
	}
	mp.out.Add(token.RPAREN)
	mp.endFunc()
}

func (mp *mathParse) indexListExpr(il *ast.IndexListExpr) {
	// fmt.Println("slice expr", se)
}

func (mp *mathParse) indexExpr(il *ast.IndexExpr) {
	if _, ok := il.Index.(*ast.IndexListExpr); ok {
		mp.basicSlicingExpr(il)
	}
}

func (mp *mathParse) basicSlicingExpr(il *ast.IndexExpr) {
	iil := il.Index.(*ast.IndexListExpr)
	mp.startFunc("tensor.Reslice", false)
	mp.out.Add(token.LPAREN)
	mp.expr(il.X)
	mp.addToken(token.COMMA) // use the [ -- can't use ( to preserve X
	mp.exprList(iil.Indices)
	mp.addToken(token.RPAREN) // replaces ]
	mp.endFunc()
}

func (mp *mathParse) sliceExpr(se *ast.SliceExpr) {
	if se.Low == nil && se.High == nil && se.Max == nil {
		mp.out.Add(token.IDENT, "tensor.FullAxis")
		mp.idx++
		return
	}
	mp.out.Add(token.IDENT, "tensor.Slice")
	mp.out.Add(token.LBRACE)
	prev := false
	if se.Low != nil {
		mp.out.Add(token.IDENT, "Start:")
		mp.expr(se.Low)
		prev = true
		if se.High == nil && se.Max == nil {
			mp.idx++
		}
	}
	if se.High != nil {
		if prev {
			mp.out.Add(token.COMMA)
		}
		mp.out.Add(token.IDENT, "Stop:")
		mp.idx++
		mp.expr(se.High)
		prev = true
	}
	if se.Max != nil {
		if prev {
			mp.out.Add(token.COMMA)
		}
		mp.idx++
		if se.Low == nil && se.High == nil {
			mp.idx++
		}
		mp.out.Add(token.IDENT, "Step:")
		mp.expr(se.Max)
	}
	mp.out.Add(token.RBRACE)
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
	// cfun := mp.funcs.Peek()
	// if cfun != nil && !cfun.tensorArgs { // we need int values
	// 	mp.exprList(il.Indices)
	// 	return
	// }
	mp.startFunc("tensor.New"+fun+"FromValues", false)
	mp.out.Add(token.LPAREN)
	mp.out.Add(token.IDENT, "[]"+typ)
	mp.addToken(token.LBRACE)
	mp.exprList(il.Indices)
	mp.addToken(token.RBRACE)
	mp.out.Add(token.ELLIPSIS)
	mp.out.Add(token.RPAREN)
	mp.endFunc()
}

var numpyFuncs = map[string]funWrap{
	"array":   {"tensor.NewFloat64", ""},
	"zeros":   {"tensor.NewFloat64", ""},
	"full":    {"tensor.NewFloat64Full", ""},
	"ones":    {"tensor.NewFloat64Ones", ""},
	"arange":  {"tensor.NewSliceInts", ""},
	"reshape": {"tensor.Reshape", ""},
	"copy":    {"tensor.Clone", ""},
	"flatten": {"tensor.Flatten", ""},
}

func (mp *mathParse) callExpr(ex *ast.CallExpr) {
	switch x := ex.Fun.(type) {
	case *ast.Ident:
		if fw, ok := numpyProps[x.Name]; ok {
			mp.callPropFun(ex, fw)
			return
		}
		mp.callName(ex, x.Name, "")
	case *ast.SelectorExpr:
		if pkg, ok := x.X.(*ast.Ident); ok {
			fun := x.Sel.Name
			if fw, ok := numpyFuncs[fun]; ok {
				mp.callPropSelFun(ex, pkg.Name, fw)
				return
			} else {
				mp.callName(ex, fun, pkg.Name)
			}
		} else {
			fmt.Printf("call, weird sel: %#v\n", x.X)
		}
	default:
		mp.expr(ex.Fun)
	}
	mp.exprList(ex.Args)
	// todo: ellipsis
	mp.addToken(token.RPAREN)
	mp.endFunc()
}

// this calls a "prop" function like ndim(a) on the object.
func (mp *mathParse) callPropFun(ex *ast.CallExpr, fw funWrap) {
	ellip := fw.wrapFunc(mp)
	mp.idx += 2
	mp.exprList(ex.Args) // this is the tensor
	mp.addToken(token.PERIOD)
	mp.out.Add(token.IDENT, fw.fun)
	if ellip {
		mp.out.Add(token.ELLIPSIS)
	}
	mp.out.Add(token.RPAREN)
	mp.endFunc()
}

// this calls global function through selector like: a.reshape()
func (mp *mathParse) callPropSelFun(ex *ast.CallExpr, obj string, fw funWrap) {
	mp.startFunc(fw.fun, false)
	mp.addToken(token.LPAREN) // use the (
	mp.out.Add(token.IDENT, obj)
	mp.idx += 2
	mp.addToken(token.COMMA)
	mp.exprList(ex.Args)
	mp.addToken(token.RPAREN)
	mp.endFunc()
}

func (mp *mathParse) callName(ex *ast.CallExpr, funName, pkgName string) {
	if fw, ok := numpyFuncs[funName]; ok {
		mp.startFunc(fw.fun, false)
		mp.addToken(token.LPAREN) // use the (
		mp.idx++                  // paren too
		return
	}
	var err error // validate name
	if pkgName != "" {
		funName = pkgName + "." + funName
		_, err = tensor.FuncByName(funName)
	} else {
		_, err = tensor.FuncByName(funName)
		if err != nil {
			funName = strings.ToUpper(funName[:1]) + funName[1:] // first letter uppercased
			_, err = tensor.FuncByName(funName)
		}
	}
	if err != nil { // not a registered tensor function
		mp.startFunc(funName, false)
		mp.addToken(token.LPAREN) // use the (
		mp.idx++
		return
	}
	mp.startFunc("tensor.CallOut", true) // tensors
	mp.addToken(token.LPAREN)
	if pkgName != "" {
		mp.idx += 2 // . and selector
	}
	mp.out.Add(token.IDENT, `"`+funName+`"`)
	mp.addToken(token.COMMA) // use the name -- need more
}

func (mp *mathParse) ident(id *ast.Ident) {
	if id == nil {
		return
	}
	/* TODO: this requires tracking of each arg to determine needed type
	cfun := mp.funcs.Peek()
	if cfun != nil && !cfun.tensorArgs { // we need the numbers from it, usually ints
		mp.out.Add(token.IDENT, "tensor.AsIntSlice")
		mp.out.Add(token.LPAREN)
		mp.addCur()
		mp.out.Add(token.RPAREN)
		mp.out.Add(token.ELLIPSIS)
	} else {
		mp.addCur()
	}
	*/
	mp.addCur()
}
