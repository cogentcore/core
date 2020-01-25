// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goki/pi/parse"
	"github.com/goki/pi/pi"
	"github.com/goki/pi/syms"
	"github.com/goki/pi/token"
)

// TypeFromAstExpr starts walking the ast expression to find the type.
// It returns the type, any Ast node that remained unprocessed at the end, and bool if found.
func (gl *GoLang) TypeFromAstExpr(fs *pi.FileState, origPkg, pkg *syms.Symbol, tyast *parse.Ast) (*syms.Type, *parse.Ast, bool) {
	pos := tyast.SrcReg.St
	fpath, _ := filepath.Abs(fs.Src.Filename)
	var conts syms.SymMap // containers of given region -- local scoping
	fs.Syms.FindContainsRegion(fpath, pos, token.NameFunction, &conts)
	if TraceTypes && len(conts) == 0 {
		fmt.Printf("no conts for fpath: %v  pos: %v\n", fpath, pos)
	}

	// if TraceTypes {
	// 	tyast.WriteTree(os.Stdout, 0)
	// }

	last := tyast.NextSiblingAst()
	// fmt.Printf("last: %v \n", last.PathUnique())

	tnm := tyast.Nm

	switch {
	case tnm == "FuncCall":
		fun := tyast.NextAst()
		funm := fun.Src
		sym, got := fs.FindNameScoped(funm, conts)
		nxt := tyast.NextSiblingAst() // skip over everything within method in ast
		if got && sym.Type != "" {
			ftnm := sym.Type
			ftyp, _ := gl.FindTypeName(ftnm, fs, pkg)
			if ftyp != nil {
				npars := ftyp.Size[0] // first size is number of params
				nrval := ftyp.Size[1] // second size is number of return values
				if nrval == 0 {
					return nil, nxt, false // no return -- shouldn't happen
				}
				rtyp := ftyp.Els[npars] // first return
				// if TraceTypes {
				// 	fmt.Printf("got return type: %v\n", rtyp)
				// }
				return gl.TypeFromAstType(fs, origPkg, pkg, nxt, last, rtyp.Type)
			} else {
				if TraceTypes {
					fmt.Printf("TExpr: FuncCall: could not find function: %v\n", funm)
				}
			}
			return nil, nxt, false
		} else {
			if TraceTypes {
				fmt.Printf("TExpr: FuncCall: could not find function: %v\n", funm)
			}
			return nil, fun, false
		}
	case tnm == "Selector":
		tnmA := tyast.ChildAst(0)
		if tnmA.Nm != "Name" {
			if TraceTypes {
				fmt.Printf("TExpr: selector start node kid is not a Name: %v, src: %v\n", tnmA.Nm, tnmA.Src)
			}
			return nil, tnmA, false
		}
		snm := tnmA.Src
		sym, got := fs.FindNameScoped(snm, conts)
		if got {
			return gl.TypeFromAstSym(fs, origPkg, pkg, tnmA, last, sym)
			// } else {
			// 	CompleteSyms = &conts
		}
		// maybe it is a package name
		psym, has := gl.PkgSyms(fs, pkg.Children, snm)
		if has {
			if TraceTypes {
				fmt.Printf("TExpr: entering package name: %v\n", snm)
			}
			nxt := tnmA.NextAst()
			if nxt != nil {
				return gl.TypeFromAstExpr(fs, origPkg, psym, nxt)
			}
			if TraceTypes {
				fmt.Printf("TExpr: package alone not useful\n")
			}
			return nil, tnmA, false // package alone not useful
		}
		if TraceTypes {
			fmt.Printf("TExpr: could not find symbol for name: %v\n", snm)
		}
		return nil, tnmA, false
	case strings.HasPrefix(tnm, "Slice"):
		tnmA := tyast.ChildAst(0)
		if tnmA.Nm != "Name" {
			if TraceTypes {
				fmt.Printf("TExpr: slice start node kid is not a Name: %v, src: %v\n", tnmA.Nm, tnmA.Src)
			}
			return nil, tnmA, false
		}
		snm := tnmA.Src
		sym, got := fs.FindNameScoped(snm, conts)
		if got {
			return gl.TypeFromAstSym(fs, origPkg, pkg, tnmA, last, sym)
		}
		if TraceTypes {
			fmt.Printf("TExpr: could not find symbol for slice var name: %v\n", snm)
		}
		return nil, tnmA, false
	case tnm == "Name":
		snm := tyast.Src
		sym, got := fs.FindNameScoped(snm, conts)
		if got {
			return gl.TypeFromAstSym(fs, origPkg, pkg, tyast, last, sym)
		} else {
			if TraceTypes {
				fmt.Printf("TExpr: could not find symbol named: %v\n", snm)
			}
		}
		return nil, tyast, false
	case tnm == "CompositeLit":
		sty, got := gl.SubTypeFromAst(fs, pkg, tyast, 0)
		return sty, nil, got
	case tnm == "AddrExpr":
		ch := tyast.ChildAst(0)
		var sty *syms.Type
		switch ch.Nm {
		case "CompositeLit":
			sty, _ = gl.SubTypeFromAst(fs, pkg, ch, 0)
		case "Name":
			snm := tyast.Src[1:] // after &
			sym, got := fs.FindNameScoped(snm, conts)
			if got {
				sty, _, got = gl.TypeFromAstSym(fs, origPkg, pkg, ch, last, sym)
			} else {
				if TraceTypes {
					fmt.Printf("TExpr: could not find symbol named: %v\n", snm)
				}
			}
		}
		if sty != nil {
			ty := &syms.Type{}
			ty.Kind = syms.Ptr
			ty.Name = "*" + sty.Name
			ty.Els.Add("ptr", sty.Name)
			return ty, nil, true
		}
		if TraceTypes {
			fmt.Printf("TExpr: could not process addr expr:\n")
			tyast.WriteTree(os.Stdout, 0)
		}
		return nil, tyast, false
	case strings.HasSuffix(tnm, "Expr"):
		// note: could figure out actual numerical type, but in practice we don't care
		// for lookup / completion, so ignoring for now.
		return BuiltinTypes["float64"], nil, true
	default:
		if TraceTypes {
			fmt.Printf("TExpr: cannot start with: %v\n", tyast.Nm)
			tyast.WriteTree(os.Stdout, 0)
		}
		return nil, tyast, false
	}
	return nil, tyast, false
}

// TypeFromAstSym attemts to get the type from given symbol as part of expression.
// It returns the type, any Ast node that remained unprocessed at the end, and bool if found.
func (gl *GoLang) TypeFromAstSym(fs *pi.FileState, origPkg, pkg *syms.Symbol, tyast, last *parse.Ast, sym *syms.Symbol) (*syms.Type, *parse.Ast, bool) {
	// if TraceTypes {
	// 	fmt.Printf("TExpr: sym named: %v  kind: %v  type: %v\n", sym.Name, sym.Kind, sym.Type)
	// }
	if sym.Type == "" { // hasn't happened yet
		// if TraceTypes {
		// 	fmt.Printf("TExpr: trying to infer type\n")
		// }
		gl.InferSymbolType(sym, fs, pkg, true)
	}
	if sym.Type == TypeErr {
		// if TraceTypes {
		// 	fmt.Printf("TExpr: source symbol has type err: %v  kind: %v\n", sym.Name, sym.Kind)
		// }
		return nil, tyast, false
	}
	if sym.Type == "" { // shouldn't happen
		sym.Type = TypeErr
		if TraceTypes {
			fmt.Printf("TExpr: source symbol has type err (but wasn't marked): %v  kind: %v\n", sym.Name, sym.Kind)
		}
		return nil, tyast, false
	}
	tnm := sym.Type
	return gl.TypeFromAstType(fs, origPkg, pkg, tyast, last, tnm)
}

// TypeFromAstType walks the ast expression to find the type, starting from current type name.
// It returns the type, any Ast node that remained unprocessed at the end, and bool if found.
func (gl *GoLang) TypeFromAstType(fs *pi.FileState, origPkg, pkg *syms.Symbol, tyast, last *parse.Ast, tnm string) (*syms.Type, *parse.Ast, bool) {
	if tnm[0] == '*' {
		tnm = tnm[1:]
	}
	ttp, npkg := gl.FindTypeName(tnm, fs, pkg)
	if ttp == nil {
		if TraceTypes {
			fmt.Printf("TExpr: error -- couldn't find type name: %v\n", tnm)
		}
		return nil, tyast, false
	}
	pkgnm := ""
	if pi := strings.Index(ttp.Name, "."); pi > 0 {
		pkgnm = ttp.Name[:pi]
	}
	if npkg != origPkg { // need to make a package-qualified copy of type
		if pkgnm == "" {
			pkgnm = npkg.Name
			qtnm := gl.QualifyType(pkgnm, ttp.Name)
			if qtnm != ttp.Name {
				if etyp, ok := pkg.Types[qtnm]; ok {
					ttp = etyp
				} else {
					ntyp := &syms.Type{}
					*ntyp = *ttp
					ntyp.Name = qtnm
					origPkg.Types.Add(ntyp)
					ttp = ntyp
				}
			}
		}
	}
	pkg = npkg // update to new context
	// if TraceTypes {
	// 	fmt.Printf("TExpr: found type: %v  kind: %v\n", ttp.Name, ttp.Kind)
	// }

	if tyast == nil || tyast == last {
		return ttp, tyast, true
	}

	nxt := tyast
	for {
		nxt = nxt.NextAst()
		if nxt == nil || nxt == last {
			// if TraceTypes {
			// 	fmt.Printf("TExpr: returning terminal type\n")
			// }
			return ttp, nxt, true
		}
		brk := false
		switch {
		case nxt.Nm == "Name":
			brk = true
		case strings.HasPrefix(nxt.Nm, "Slice"):
			eltyp := ttp.Els.ByName("val")
			if eltyp != nil {
				elnm := gl.QualifyType(pkgnm, eltyp.Type)
				// if TraceTypes {
				// 	fmt.Printf("TExpr: slice/map el type: %v\n", elnm)
				// }
				return gl.TypeFromAstType(fs, origPkg, pkg, nxt, last, elnm)
			}
			if TraceTypes {
				fmt.Printf("TExpr: slice operator not on slice: %v\n", ttp.Name)
			}
		case nxt.Nm == "FuncCall":
			// ttp is the function type name
			fun := nxt.NextAst()
			if fun == nil || fun == last {
				return ttp, fun, true
			}
			funm := fun.Src
			ftyp, got := ttp.Meths[funm]
			nxt = nxt.NextSiblingAst() // skip over everything within method in ast
			// if TraceTypes && nxt != nil {
			// 	nxt.WriteTree(os.Stdout, 0)
			// }
			if got {
				npars := ftyp.Size[0] // first size is number of params
				nrval := ftyp.Size[1] // second size is number of return values
				if nrval == 0 {
					return nil, nxt, false // no return -- shouldn't happen
				}
				rtyp := ftyp.Els[npars] // first return
				// if TraceTypes {
				// 	fmt.Printf("got return type: %v\n", rtyp)
				// }
				return gl.TypeFromAstType(fs, origPkg, pkg, nxt, last, rtyp.Type)
			} else {
				if TraceTypes {
					fmt.Printf("TExpr: FuncCall: could not find method: %v in type: %v\n", ttp.Name, funm)
				}
				return nil, fun, false
			}
		}
		if brk || nxt == nil || nxt == last {
			break
		}
		if TraceTypes {
			fmt.Printf("TExpr: skipping over %v\n", nxt.Nm)
		}
	}
	if nxt == nil {
		return ttp, nxt, false
	}
	nm := nxt.Src
	stp := ttp.Els.ByName(nm)
	if stp != nil {
		if TraceTypes {
			fmt.Printf("TExpr: found Name: %v in type els\n", nm)
		}
		return gl.TypeFromAstType(fs, origPkg, pkg, nxt, last, stp.Type)
	}
	if TraceTypes {
		fmt.Printf("TExpr: error -- Name: %v not found in type els\n", nm)
		// ttp.WriteDoc(os.Stdout, 0)
	}
	return ttp, nxt, true // robust, needed for completion
}
