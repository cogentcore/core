// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"fmt"

	"github.com/goki/pi/parse"
	"github.com/goki/pi/pi"
	"github.com/goki/pi/syms"
	"github.com/goki/pi/token"
)

// TypeMeths gathers method types from the type symbol's children
func (gl *GoLang) TypeMeths(fs *pi.FileState, pkg *syms.Symbol, ty *syms.Type) {
	tnm := gl.UnQualifyType(ty.Name)
	tsym, got := pkg.Children.FindNameScoped(tnm)
	if !got {
		if TraceTypes {
			fmt.Printf("TypeMeths: error -- did NOT get type sym: %v in pkg: %v\n", tnm, pkg.Name)
		}
		return
	}
	for _, sy := range tsym.Children {
		if sy.Kind.SubCat() != token.NameFunction || sy.Ast == nil {
			continue
		}

		fty := gl.FuncTypeFromAst(fs, pkg, sy.Ast.(*parse.Ast), nil)
		if fty != nil {
			fty.Kind = syms.Method
			fty.Name = sy.Name
			ty.Meths.Add(fty)
			// if TraceTypes {
			// 	fmt.Printf("TypeMeths: Added method: %v\n", fty)
			// }
		} else {
			if TraceTypes {
				fmt.Printf("TypeMeths: method failed: %v\n", sy.Name)
			}
		}
	}
}

// NamesFromAst returns a slice of name(s) from namelist nodes
func (gl *GoLang) NamesFromAst(fs *pi.FileState, pkg *syms.Symbol, ast *parse.Ast, idx int) []string {
	sast, err := ast.ChildAstTry(idx)
	if err != nil {
		if TraceTypes {
			fmt.Println(err)
		}
		return nil
	}
	var sary []string
	if sast.HasChildren() {
		for i := range sast.Kids {
			sary = append(sary, gl.NamesFromAst(fs, pkg, sast, i)...)
		}
	} else {
		sary = append(sary, sast.Src)
	}
	return sary
}

// FuncTypeFromAst initializes a function type from ast -- type can either be anon
// or a named type -- if anon then the name is the full type signature without param names
func (gl *GoLang) FuncTypeFromAst(fs *pi.FileState, pkg *syms.Symbol, ast *parse.Ast, fty *syms.Type) *syms.Type {
	// ast.WriteTree(os.Stdout, 0)

	pars, err := ast.ChildAstTry(0)
	if err != nil {
		if TraceTypes {
			fmt.Println(err)
		}
		return nil
	}
	if fty == nil {
		fty = &syms.Type{}
		fty.Kind = syms.Func
	}
	poff := 0
	isMeth := false
	if pars.Nm == "MethRecvName" && len(ast.Kids) > 2 {
		isMeth = true
		rcv := pars.Kids[0].(*parse.Ast)
		rtyp := pars.Kids[1].(*parse.Ast)
		fty.Els.Add(rcv.Src, rtyp.Src)
		poff = 2
		pars = ast.ChildAst(2)
	} else if pars.Nm == "Name" && len(ast.Kids) > 1 {
		poff = 1
		pars = ast.ChildAst(1)
	}
	npars := len(pars.Kids)
	var sigpars *parse.Ast
	if npars > 0 && (pars.Nm == "SigParams" || pars.Nm == "SigParamsResult") {
		if ps, err := pars.ChildAstTry(0); err == nil {
			sigpars = pars
			pars = ps
			npars = len(pars.Kids)
		} else {
			npars = 0 // not really
		}
	}
	if npars > 0 {
		gl.ParamsFromAst(fs, pkg, pars, fty, "param")
		npars = len(fty.Els) // how many we added -- auto-includes receiver for method
	} else {
		if isMeth {
			npars = 1
		}
	}
	nrvals := 0
	if sigpars != nil && len(sigpars.Kids) >= 2 {
		rvals := sigpars.ChildAst(1)
		gl.RvalsFromAst(fs, pkg, rvals, fty)
		nrvals = len(fty.Els) - npars // how many we added..
	} else if poff < 2 && (len(ast.Kids) >= poff+2) {
		rvals := ast.ChildAst(poff + 1)
		gl.RvalsFromAst(fs, pkg, rvals, fty)
		nrvals = len(fty.Els) - npars // how many we added..
	}
	fty.Size = []int{npars, nrvals}
	return fty
}

// ParamsFromAst sets params as Els for given function type (also for return types)
func (gl *GoLang) ParamsFromAst(fs *pi.FileState, pkg *syms.Symbol, pars *parse.Ast, fty *syms.Type, name string) {
	npars := len(pars.Kids)
	var pnames []string // param names that all share same type
	for i := 0; i < npars; i++ {
		par := pars.Kids[i].(*parse.Ast)
		psz := len(par.Kids)
		if par.Nm == "ParType" && psz == 1 {
			ptypa := par.Kids[0].(*parse.Ast)
			if ptypa.Nm == "TypeNm" { // could be multiple args with same type or a separate type-only arg
				if ptl, _ := gl.FindTypeName(par.Src, fs, pkg); ptl != nil {
					fty.Els.Add(fmt.Sprintf("%s_%v", name, i), par.Src)
					continue
				}
				pnames = append(pnames, par.Src) // add to later type
			} else {
				ptyp, ok := gl.SubTypeFromAst(fs, pkg, par, 0)
				if ok {
					pnsz := len(pnames)
					if pnsz > 0 {
						for _, pn := range pnames {
							fty.Els.Add(pn, ptyp.Name)
						}
					}
					fty.Els.Add(fmt.Sprintf("%s_%v", name, i), ptyp.Name)
					continue
				}
				pnames = nil
			}
		} else if psz == 2 { // ParName
			pnm := par.Kids[0].(*parse.Ast)
			ptyp, ok := gl.SubTypeFromAst(fs, pkg, par, 1)
			if ok {
				pnsz := len(pnames)
				if pnsz > 0 {
					for _, pn := range pnames {
						fty.Els.Add(pn, ptyp.Name)
					}
				}
				fty.Els.Add(pnm.Src, ptyp.Name)
				continue
			}
			pnames = nil
		}
	}
}

// RvalsFromAst sets return value(s) as Els for given function type
func (gl *GoLang) RvalsFromAst(fs *pi.FileState, pkg *syms.Symbol, rvals *parse.Ast, fty *syms.Type) {
	if rvals.Nm == "Block" { // todo: maybe others
		return
	}
	nrvals := len(rvals.Kids)
	if nrvals == 1 { // single rval, unnamed, has type directly..
		rval := rvals.ChildAst(0)
		if rval.Nm != "ParName" {
			nrvals = 1
			rtyp, ok := gl.SubTypeFromAst(fs, pkg, rvals, 0)
			if ok {
				fty.Els.Add("rval", rtyp.Name)
				return
			}
		}
	}
	gl.ParamsFromAst(fs, pkg, rvals, fty, "rval")
}
