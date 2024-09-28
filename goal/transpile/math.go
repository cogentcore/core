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

// TranspileMath does math mode transpiling. fullLine indicates code should be
// full statement(s).
func (st *State) TranspileMath(toks Tokens, code string, fullLine bool) Tokens {
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
	mp := mathParse{state: st, toks: toks, code: code}
	// mp.trace = true

	mods := AllErrors // | Trace

	if fullLine {
		ewords, err := ExecWords(str)
		if len(ewords) > 0 {
			if cmd, ok := datafsCommands[ewords[0]]; ok {
				mp.ewords = ewords
				err := cmd(&mp)
				if err != nil {
					fmt.Println(ewords[0]+":", err.Error())
				}
				return nil
			}
		}

		stmts, err := ParseLine(str, mods)
		if err != nil {
			fmt.Println("line code:", str)
			fmt.Println("parse err:", err)
		}
		if len(stmts) == 0 {
			return toks
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

	// current arg index we are processing
	curArg int
}

// mathParse has the parsing state
type mathParse struct {
	state  *State
	code   string   // code string
	toks   Tokens   // source tokens we are parsing
	ewords []string // exec words
	idx    int      // current index in source tokens -- critical to sync as we "use" source
	out    Tokens   // output tokens we generate
	trace  bool     // trace of parsing -- turn on to see alignment

	// stack of function info -- top of stack reflects the current function
	funcs stack.Stack[*funcInfo]
}

// returns the current argument for current function
func (mp *mathParse) curArg() *tensor.Arg {
	cfun := mp.funcs.Peek()
	if cfun == nil {
		return nil
	}
	if cfun.curArg < len(cfun.Args) {
		return cfun.Args[cfun.curArg]
	}
	return nil
}

func (mp *mathParse) nextArg() {
	cfun := mp.funcs.Peek()
	if cfun == nil || len(cfun.Args) == 0 {
		// fmt.Println("next arg no fun or no args")
		return
	}
	n := len(cfun.Args)
	if cfun.curArg == n-1 {
		carg := cfun.Args[n-1]
		if !carg.IsVariadic {
			fmt.Println("math transpile: args exceed registered function number:", cfun)
		}
		return
	}
	cfun.curArg++
}

func (mp *mathParse) curArgIsTensor() bool {
	carg := mp.curArg()
	if carg == nil {
		return false
	}
	return carg.IsTensor
}

func (mp *mathParse) curArgIsInts() bool {
	carg := mp.curArg()
	if carg == nil {
		return false
	}
	return carg.IsInt && carg.IsVariadic
}

// startFunc is called when starting a new function.
// empty is "dummy" assign case using Inc
func (mp *mathParse) startFunc(name string) *funcInfo {
	fi := &funcInfo{}
	sname := name
	if name == "" {
		sname = "tmath.Inc"
	}
	if tf, err := tensor.FuncByName(sname); err == nil {
		fi.Func = *tf
	} else {
		fi.Name = name // not clear what point is
	}
	mp.funcs.Push(fi)
	if name != "" {
		mp.out.Add(token.IDENT, name)
	}
	return fi
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
		fn := "Inc"
		if x.Tok == token.DEC {
			fn = "Dec"
		}
		mp.startFunc("tmath." + fn)
		mp.out.Add(token.LPAREN)
		mp.expr(x.X)
		mp.addToken(token.RPAREN)

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
		mp.out.Add(token.IDENT, ".Bool1D(0)") // turn bool expr into actual bool
		if x.Body != nil && len(x.Body.List) > 0 {
			mp.addToken(token.LBRACE)
			mp.stmtList(x.Body.List)
			mp.addToken(token.RBRACE)
		} else {
			mp.addToken(token.LBRACE)
		}
		if x.Else != nil {
			mp.addToken(token.ELSE)
			mp.stmt(x.Else)
		}

	case *ast.ForStmt:
		mp.addToken(token.FOR)
		mp.stmt(x.Init)
		if x.Init != nil {
			mp.addToken(token.SEMICOLON)
		}
		mp.expr(x.Cond)
		if x.Cond != nil {
			mp.out.Add(token.IDENT, ".Bool1D(0)") // turn bool expr into actual bool
			mp.addToken(token.SEMICOLON)
		}
		mp.stmt(x.Post)
		if x.Body != nil && len(x.Body.List) > 0 {
			mp.addToken(token.LBRACE)
			mp.stmtList(x.Body.List)
			mp.addToken(token.RBRACE)
		} else {
			mp.addToken(token.LBRACE)
		}

	case *ast.RangeStmt:
		if x.Key == nil || x.Value == nil {
			fmt.Println("for range statement requires both index and value variables")
			return
		}
		ki, _ := x.Key.(*ast.Ident)
		vi, _ := x.Value.(*ast.Ident)
		ei, _ := x.X.(*ast.Ident)
		if ki == nil || vi == nil || ei == nil {
			fmt.Println("for range statement requires all variables (index, value, range) to be variable names, not other expressions")
			return
		}
		knm := ki.Name
		vnm := vi.Name
		enm := ei.Name

		mp.addToken(token.FOR)
		mp.expr(x.Key)
		mp.idx += 2
		mp.addToken(token.DEFINE)
		mp.out.Add(token.IDENT, "0")
		mp.out.Add(token.SEMICOLON)
		mp.out.Add(token.IDENT, knm)
		mp.out.Add(token.IDENT, "<")
		mp.out.Add(token.IDENT, enm)
		mp.out.Add(token.PERIOD)
		mp.out.Add(token.IDENT, "Len")
		mp.idx++
		mp.out.Add(token.LPAREN)
		mp.out.Add(token.RPAREN)
		mp.idx++
		mp.out.Add(token.SEMICOLON)
		mp.idx++
		mp.out.Add(token.IDENT, knm)
		mp.out.Add(token.INC)
		mp.out.Add(token.LBRACE)

		mp.out.Add(token.IDENT, vnm)
		mp.out.Add(token.DEFINE)
		mp.out.Add(token.IDENT, enm)
		mp.out.Add(token.IDENT, ".Float1D")
		mp.out.Add(token.LPAREN)
		mp.out.Add(token.IDENT, knm)
		mp.out.Add(token.RPAREN)

		if x.Body != nil && len(x.Body.List) > 0 {
			mp.stmtList(x.Body.List)
			mp.addToken(token.RBRACE)
		}

		// TODO
		// CaseClause: SwitchStmt:, TypeSwitchStmt:, CommClause:, SelectStmt:
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

	case *ast.StarExpr:
		mp.addToken(token.MUL)
		mp.expr(x.X)

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
		// note: shouldn't happen normally:
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

func (mp *mathParse) argsList(ex []ast.Expr) {
	n := len(ex)
	if n == 0 {
		return
	}
	if n == 1 {
		mp.expr(ex[0])
		return
	}
	for i := range n {
		// cfun := mp.funcs.Peek()
		// if i != cfun.curArg {
		// 	fmt.Println(cfun, "arg should be:", i, "is:", cfun.curArg)
		// }
		mp.expr(ex[i])
		if i < n-1 {
			mp.nextArg()
			mp.addToken(token.COMMA)
		}
	}
}

func (mp *mathParse) exprIsBool(ex ast.Expr) bool {
	switch x := ex.(type) {
	case *ast.BinaryExpr:
		if (x.Op >= token.EQL && x.Op <= token.GTR) || (x.Op >= token.NEQ && x.Op <= token.GEQ) {
			return true
		}
	case *ast.ParenExpr:
		return mp.exprIsBool(x.X)
	}
	return false
}

func (mp *mathParse) exprsAreBool(ex []ast.Expr) bool {
	for _, x := range ex {
		if mp.exprIsBool(x) {
			return true
		}
	}
	return false
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
		if un, ok := ex.Y.(*ast.StarExpr); ok { // ** power operator
			ex.Y = un.X
			fn = "Pow"
		}
	case token.QUO:
		fn = "Div"
	case token.EQL:
		fn = "Equal"
	case token.LSS:
		fn = "Less"
	case token.GTR:
		fn = "Greater"
	case token.NEQ:
		fn = "NotEqual"
	case token.LEQ:
		fn = "LessEqual"
	case token.GEQ:
		fn = "GreaterEqual"
	case token.LOR:
		fn = "Or"
	case token.LAND:
		fn = "And"
	default:
		fmt.Println("binary token:", ex.Op)
	}
	mp.startFunc("tmath." + fn)
	mp.out.Add(token.LPAREN)
	mp.expr(ex.X)
	mp.out.Add(token.COMMA)
	mp.idx++
	if fn == "Pow" {
		mp.idx++
	}
	mp.expr(ex.Y)
	mp.out.Add(token.RPAREN)
	mp.endFunc()
}

func (mp *mathParse) unaryExpr(ex *ast.UnaryExpr) {
	if _, isbl := ex.X.(*ast.BasicLit); isbl {
		mp.addToken(ex.Op)
		mp.expr(ex.X)
		return
	}
	fn := ""
	switch ex.Op {
	case token.NOT:
		fn = "Not"
	case token.SUB:
		fn = "Negate"
	default: // * goes to StarExpr -- not sure what else could happen here?
		mp.addToken(ex.Op)
		mp.expr(ex.X)
		return
	}
	mp.startFunc("tmath." + fn)
	mp.addToken(token.LPAREN)
	mp.expr(ex.X)
	mp.out.Add(token.RPAREN)
	mp.endFunc()
}

func (mp *mathParse) defineStmt(as *ast.AssignStmt) {
	firstStmt := mp.idx == 0
	mp.exprList(as.Lhs)
	mp.addToken(as.Tok)
	mp.startFunc("") // dummy single arg tensor function
	mp.exprList(as.Rhs)
	mp.endFunc()
	if firstStmt && mp.state.MathRecord {
		nvar, ok := as.Lhs[0].(*ast.Ident)
		if ok {
			mp.out.Add(token.SEMICOLON)
			mp.out.Add(token.IDENT, "datafs.Record("+nvar.Name+",`"+nvar.Name+"`)")
		}
	}
}

func (mp *mathParse) assignStmt(as *ast.AssignStmt) {
	if _, ok := as.Lhs[0].(*ast.Ident); ok {
		mp.exprList(as.Lhs)
		mp.addToken(as.Tok)
		mp.startFunc("")
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
	mp.startFunc("tmath." + fn)
	mp.out.Add(token.LPAREN)
	mp.exprList(as.Lhs)
	mp.out.Add(token.COMMA)
	mp.idx++
	mp.exprList(as.Rhs)
	mp.out.Add(token.RPAREN)
	mp.endFunc()
}

func (mp *mathParse) basicLit(lit *ast.BasicLit) {
	if mp.curArgIsTensor() {
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
	mp.startFunc(wrapFun)
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
	fun := "tensor.Reslice"
	if mp.exprsAreBool(iil.Indices) {
		fun = "tensor.Mask"
	}
	mp.startFunc(fun)
	mp.out.Add(token.LPAREN)
	mp.expr(il.X)
	mp.nextArg()
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
	if mp.curArgIsInts() {
		mp.idx++ // opening brace we're not using
		mp.exprList(il.Indices)
		mp.idx++ // closing brace we're not using
		return
	}
	mp.startFunc("tensor.New" + fun + "FromValues")
	mp.out.Add(token.LPAREN)
	mp.out.Add(token.IDENT, "[]"+typ)
	mp.addToken(token.LBRACE)
	mp.exprList(il.Indices)
	mp.addToken(token.RBRACE)
	mp.out.Add(token.ELLIPSIS)
	mp.out.Add(token.RPAREN)
	mp.endFunc()
}

// nofun = do not accept a function version, just a method
var numpyFuncs = map[string]funWrap{
	// "array":   {"tensor.NewFloatFromValues", ""}, // todo: probably not right, maybe don't have?
	"zeros":    {"tensor.NewFloat64", ""},
	"full":     {"tensor.NewFloat64Full", ""},
	"ones":     {"tensor.NewFloat64Ones", ""},
	"rand":     {"tensor.NewFloat64Rand", ""},
	"arange":   {"tensor.NewIntRange", ""},
	"linspace": {"tensor.NewFloat64SpacedLinear", ""},
	"reshape":  {"tensor.Reshape", ""},
	"copy":     {"tensor.Clone", ""},
	"flatten":  {"tensor.Flatten", "nofun"},
	"squeeze":  {"tensor.Squeeze", "nofun"},
}

func (mp *mathParse) callExpr(ex *ast.CallExpr) {
	switch x := ex.Fun.(type) {
	case *ast.Ident:
		if fw, ok := numpyProps[x.Name]; ok && fw.wrap != "nofun" {
			mp.callPropFun(ex, fw)
			return
		}
		mp.callName(ex, x.Name, "")
	case *ast.SelectorExpr:
		fun := x.Sel.Name
		if pkg, ok := x.X.(*ast.Ident); ok {
			if fw, ok := numpyFuncs[fun]; ok {
				mp.callPropSelFun(ex, x.X, fw)
				return
			} else {
				// fmt.Println("call name:", fun, pkg.Name)
				mp.callName(ex, fun, pkg.Name)
			}
		} else {
			if fw, ok := numpyFuncs[fun]; ok {
				mp.callPropSelFun(ex, x.X, fw)
				return
			}
			// todo: dot fun?
			mp.expr(ex)
		}
	default:
		mp.expr(ex.Fun)
	}
	mp.argsList(ex.Args)
	// todo: ellipsis
	mp.addToken(token.RPAREN)
	mp.endFunc()
}

// this calls a "prop" function like ndim(a) on the object.
func (mp *mathParse) callPropFun(cf *ast.CallExpr, fw funWrap) {
	ellip := fw.wrapFunc(mp)
	mp.idx += 2
	mp.exprList(cf.Args) // this is the tensor
	mp.addToken(token.PERIOD)
	mp.out.Add(token.IDENT, fw.fun)
	if ellip {
		mp.out.Add(token.ELLIPSIS)
	}
	mp.out.Add(token.RPAREN)
	mp.endFunc()
}

// this calls global function through selector like: a.reshape()
func (mp *mathParse) callPropSelFun(cf *ast.CallExpr, ex ast.Expr, fw funWrap) {
	mp.startFunc(fw.fun)
	mp.out.Add(token.LPAREN) // use the (
	mp.expr(ex)
	mp.idx += 2
	if len(cf.Args) > 0 {
		mp.nextArg() // did first
		mp.addToken(token.COMMA)
		mp.argsList(cf.Args)
	} else {
		mp.idx++
	}
	mp.addToken(token.RPAREN)
	mp.endFunc()
}

func (mp *mathParse) callName(cf *ast.CallExpr, funName, pkgName string) {
	if fw, ok := numpyFuncs[funName]; ok {
		mp.startFunc(fw.fun)
		mp.addToken(token.LPAREN) // use the (
		mp.idx++                  // paren too
		return
	}
	var err error // validate name
	if pkgName != "" {
		funName = pkgName + "." + funName
		_, err = tensor.FuncByName(funName)
	} else { // non-package qualified names are _only_ in tmath! can be lowercase
		_, err = tensor.FuncByName("tmath." + funName)
		if err != nil {
			funName = strings.ToUpper(funName[:1]) + funName[1:] // first letter uppercased
			_, err = tensor.FuncByName("tmath." + funName)
		}
		if err == nil { // registered, must be in tmath
			funName = "tmath." + funName
		}
	}
	if err != nil { // not a registered tensor function
		// fmt.Println("regular fun", funName)
		mp.startFunc(funName)
		mp.addToken(token.LPAREN) // use the (
		mp.idx += 3
		return
	}
	mp.startFunc(funName)
	mp.idx += 1
	if pkgName != "" {
		mp.idx += 2 // . and selector
	}
	mp.addToken(token.LPAREN)
}

func (mp *mathParse) ident(id *ast.Ident) {
	if id == nil {
		return
	}
	if mp.curArgIsInts() {
		mp.out.Add(token.IDENT, "tensor.AsIntSlice")
		mp.out.Add(token.LPAREN)
		mp.addCur()
		mp.out.Add(token.RPAREN)
		mp.out.Add(token.ELLIPSIS)
	} else {
		mp.addCur()
	}
}
