// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"fmt"
	"os"
	"strings"

	"github.com/goki/ki/ki"
	"github.com/goki/ki/walki"
	"github.com/goki/pi/parse"
	"github.com/goki/pi/pi"
	"github.com/goki/pi/syms"
	"github.com/goki/pi/token"
)

var TraceTypes = true

// todo: when do we bind kind of type for external refs (qualified names)
// need to import everything (recursively) to get to the bottom of anything
// if we don't, then we have to do it repeatedly live -- would be better to
// do it and cache the results?  but only a problem when we have types
// defined in terms of other types -- maybe not all that relevant?

// FindTypeName finds given type name in pkg and in broader context
// assumed to be called under 	fs.SymsMu.RLock()
//	defer fs.SymsMu.RUnlock()
// returns new package symbol if type name is in a different package
// else returns pkg arg.
func (gl *GoLang) FindTypeName(tynm string, fs *pi.FileState, pkg *syms.Symbol) (*syms.Type, *syms.Symbol) {
	if tynm[0] == '*' {
		tynm = tynm[1:]
	}
	sci := strings.Index(tynm, ".")
	if sci < 0 {
		if btyp, ok := BuiltinTypes[tynm]; ok {
			return btyp, pkg
		}
		if gtyp, ok := pkg.Types[tynm]; ok {
			return gtyp, pkg
		}
		if TraceTypes {
			fmt.Printf("FindTypeName: unqualified type name: %v not found in package: %v\n", tynm, pkg.Name)
		}
		return nil, pkg
	}
	pnm := tynm[:sci]
	if npkg, ok := gl.PkgSyms(fs, pkg.Children, pnm); ok {
		if TraceTypes {
			fmt.Printf("FindTypeName: found package: %v\n", pnm)
		}
		tnm := tynm[sci+1:]
		if gtyp, ok := npkg.Types[tnm]; ok {
			return gtyp, npkg
		}
		if TraceTypes {
			fmt.Printf("FindTypeName: type name: %v not found in package: %v\n", tnm, pnm)
		}
	}
	if TraceTypes {
		fmt.Printf("FindTypeName: type name: %v not found in package: %v\n", tynm, pkg.Name)
	}
	return nil, pkg
}

// QualifyType returns the type name tnm qualified by pkgnm if it is non-empty
// and only if tnm is not a basic type name
func (gl *GoLang) QualifyType(pkgnm, tnm string) string {
	if pkgnm == "" || strings.Index(tnm, ".") > 0 {
		return tnm
	}
	if _, btyp := BuiltinTypes[tnm]; btyp {
		return tnm
	}
	return pkgnm + "." + tnm
}

// ResolveTypes initializes all user-defined types from Ast data
// and then resolves types of symbols.  The pkg must be a single
// package symbol i.e., the children there are all the elements of the
// package and the types are all the global types within the package.
func (gl *GoLang) ResolveTypes(fs *pi.FileState, pkg *syms.Symbol) {
	gl.TypesFromAst(fs, pkg)
	gl.InferSymbolType(pkg, fs, pkg)
}

// TypesFromAst initializes the types from their Ast parse
func (gl *GoLang) TypesFromAst(fs *pi.FileState, pkg *syms.Symbol) {
	InstallBuiltinTypes()

	for _, ty := range pkg.Types {
		if ty.Ast == nil {
			continue // shouldn't be
		}
		tyast, err := ty.Ast.(*parse.Ast).ChildAstTry(1)
		if err != nil {
			continue
		}
		gl.TypeFromAst(fs, pkg, ty, tyast)
	}
}

// TypeFromAst initializes the types from their Ast parse -- returns true if successful
// if ty arg is nil, an new type is created, otherwise existing one is filled in.
func (gl *GoLang) TypeFromAst(fs *pi.FileState, pkg *syms.Symbol, ty *syms.Type, tyast *parse.Ast) (*syms.Type, bool) {
	if ty == nil {
		ty = &syms.Type{}
	}
	src := tyast.Src
	switch tyast.Nm {
	case "BasicType":
		if btyp, ok := BuiltinTypes[src]; ok {
			ty.Kind = btyp.Kind
			ty.Els.Add("par", btyp.Name) // parent type
			if ty.Name == "" {
				ty.Name = btyp.Name
			}
		} else {
			if TraceTypes {
				fmt.Printf("basic type: %v not found\n", src)
			}
			return ty, false
		}
	case "TypeNm":
		if btyp, ok := pkg.Types[src]; ok {
			ty.Kind = btyp.Kind
			ty.Els.Add("par", btyp.Name) // parent type
			if ty.Name == "" {
				ty.Name = btyp.Name
			}
		} else {
			if TraceTypes {
				fmt.Printf("unqualified type: %v not found\n", src)
			}
			return ty, false
		}
	case "QualType":
		ttp, _ := gl.FindTypeName(src, fs, pkg)
		if ttp != nil {
			*ty = *ttp
		} else {
			ty.Els.Add("par", src)
		}
		ty.Name = src
		pkg.Types.Add(ty)
	case "PointerType":
		ty.Kind = syms.Ptr
		if sty, ok := gl.SubTypeFromAst(fs, pkg, tyast, 0); ok {
			ty.Els.Add("ptr", sty.Name)
			if ty.Name == "" {
				ty.Name = "*" + sty.Name
				pkg.Types.Add(ty)
			}
		} else {
			return ty, false
		}
	case "MapType":
		ty.Kind = syms.Map
		keyty, kok := gl.SubTypeFromAst(fs, pkg, tyast, 0)
		valty, vok := gl.SubTypeFromAst(fs, pkg, tyast, 1)
		if kok && vok {
			ty.Els.Add("key", keyty.Name)
			ty.Els.Add("val", valty.Name)
			if ty.Name == "" {
				ty.Name = "map[" + keyty.Name + "]" + valty.Name
				pkg.Types.Add(ty)
			}
		} else {
			return ty, false
		}
	case "SliceType":
		ty.Kind = syms.List
		valty, ok := gl.SubTypeFromAst(fs, pkg, tyast, 0)
		if ok {
			ty.Els.Add("val", valty.Name)
			if ty.Name == "" {
				ty.Name = "[]" + valty.Name
				pkg.Types.Add(ty)
			}
		} else {
			return ty, false
		}
	case "ArrayType":
		ty.Kind = syms.Array
		valty, ok := gl.SubTypeFromAst(fs, pkg, tyast, 1)
		if ok {
			ty.Els.Add("val", valty.Name)
			if ty.Name == "" {
				ty.Name = "[]" + valty.Name // todo: get size from child0, set to Size
				pkg.Types.Add(ty)
			}
		} else {
			return ty, false
		}
	case "StructType":
		ty.Kind = syms.Struct
		nfld := len(tyast.Kids)
		ty.Size = []int{nfld}
		tynm := ""
		for i := 0; i < nfld; i++ {
			fld := tyast.Kids[i].(*parse.Ast)
			fsrc := fld.Src
			if ty.Name == "" {
				tynm += fsrc + ";"
			}
			switch fld.Nm {
			case "NamedField":
				if len(fld.Kids) <= 1 {
					ty.Els.Add(fsrc, fsrc) // anon two are same
					continue
				}
				fldty, ok := gl.SubTypeFromAst(fs, pkg, fld, 1)
				if ok {
					nms := gl.NamesFromAst(fs, pkg, fld, 0)
					for _, nm := range nms {
						ty.Els.Add(nm, fldty.Name)
					}
				}
			case "AnonQualField":
				ty.Els.Add(fsrc, fsrc) // anon two are same
				if ty.Name == "" {
					tynm += fsrc + ";"
				}
			}
		}
		if ty.Name == "" {
			ty.Name = "struct{" + tynm + "}"
		}
	case "InterfaceType":
		ty.Kind = syms.Interface
		nmth := len(tyast.Kids)
		ty.Size = []int{nmth}
		for i := 0; i < nmth; i++ {
			fld := tyast.Kids[i].(*parse.Ast)
			fsrc := fld.Src
			switch fld.Nm {
			case "MethSpecAnonLocal":
				fallthrough
			case "MethSpecAnonQual":
				ty.Els.Add(fsrc, fsrc) // anon two are same
			case "MethSpecName":
				if nm, err := fld.ChildAstTry(0); err == nil {
					mty := syms.NewType(ty.Name+":"+nm.Nm, syms.Method)
					pkg.Types.Add(mty) // add interface methods as new types..
					gl.FuncTypeFromAst(fs, pkg, fld, mty)
					ty.Els.Add(nm.Nm, mty.Name)
				}
			}
		}
	case "FuncType":
		ty.Kind = syms.Func
		gl.FuncTypeFromAst(fs, pkg, tyast, ty)
	case "Selector":
		return gl.TypeFromAstExpr(fs, pkg, pkg, tyast)
	default:
		if TraceTypes {
			fmt.Printf("Ast type node: %v not found\n", tyast.Nm)
		}
		return nil, false
	}
	return ty, true // fallthrough is true..
}

// SubTypeFromAst returns a subtype from child ast at given index, nil if failed
func (gl *GoLang) SubTypeFromAst(fs *pi.FileState, pkg *syms.Symbol, ast *parse.Ast, idx int) (*syms.Type, bool) {
	sast, err := ast.ChildAstTry(idx)
	if err != nil {
		if TraceTypes {
			fmt.Println(err)
		}
		return nil, false
	}
	return gl.TypeFromAst(fs, pkg, nil, sast)
}

// TypeFromAstExpr starts walking the ast expression to find the type
func (gl *GoLang) TypeFromAstExpr(fs *pi.FileState, origPkg, pkg *syms.Symbol, tyast *parse.Ast) (*syms.Type, bool) {
	pos := tyast.SrcReg.St
	var conts syms.SymMap // containers of given region -- local scoping
	fs.Syms.FindContainsRegion(pos, token.NameFunction, &conts)

	tyast.WriteTree(os.Stdout, 0)
	last := walki.NextSibling(tyast)
	// fmt.Printf("last: %v \n", last.PathUnique())

	switch tyast.Nm {
	case "Selector":
		if !tyast.HasChildren() {
			if TraceTypes {
				fmt.Printf("TExpr: selector start node has no kids: %v\n", tyast.Nm)
			}
			return nil, false
		}
		tnmA := tyast.Kids[0].(*parse.Ast)
		if tnmA.Nm != "Name" {
			if TraceTypes {
				fmt.Printf("TExpr: selector start node kid is not a Name: %v, src: %v\n", tnmA.Nm, tnmA.Src)
			}
			return nil, false
		}
		snm := tnmA.Src
		sym, got := fs.FindNameScoped(snm, conts)
		if got {
			return gl.TypeFromAstSym(fs, origPkg, pkg, tnmA, last, sym)
		}
		// maybe it is a package name
		psym, has := gl.PkgSyms(fs, pkg.Children, snm)
		if has {
			if TraceTypes {
				fmt.Printf("TExpr: entering package name: %v\n", snm)
			}
			nxt := walki.Next(tnmA)
			if nxt != nil {
				return gl.TypeFromAstExpr(fs, origPkg, psym, nxt.(*parse.Ast))
			}
			if TraceTypes {
				fmt.Printf("TExpr: package alone not useful\n")
			}
			return nil, false // package alone not useful
		}
		if TraceTypes {
			fmt.Printf("TExpr: could not find symbol for name: %v  with no current type, bailing\n", snm)
		}
		return nil, false
	case "Name":
		// from package, will have name here which should be type name
		// todo: add prefix to type name!
	default:
		if TraceTypes {
			fmt.Printf("TExpr: cannot start with: %v\n", tyast.Nm)
		}
		return nil, false
	}
	return nil, false
}

// TypeFromAstSym attemts to get the type from given symbol as part of expression
func (gl *GoLang) TypeFromAstSym(fs *pi.FileState, origPkg, pkg *syms.Symbol, tyast *parse.Ast, last ki.Ki, sym *syms.Symbol) (*syms.Type, bool) {
	if TraceTypes {
		fmt.Printf("TExpr: sym named: %v  kind: %v  type: %v\n", sym.Name, sym.Kind, sym.Type)
	}
	if sym.Type == "" { // hasn't happened yet
		if TraceTypes {
			fmt.Printf("TExpr: trying to infer type\n")
		}
		gl.InferSymbolType(sym, fs, pkg)
	}
	if sym.Type == TypeErr {
		if TraceTypes {
			fmt.Printf("TExpr: source symbol has type err: %v  kind: %v\n", sym.Name, sym.Kind)
			return nil, false
		}
	}
	tnm := sym.Type
	return gl.TypeFromAstType(fs, origPkg, pkg, tyast, last, tnm)
}

// TypeFromAstType walks the ast expression to find the type, starting from current type name
func (gl *GoLang) TypeFromAstType(fs *pi.FileState, origPkg, pkg *syms.Symbol, tyast *parse.Ast, last ki.Ki, tnm string) (*syms.Type, bool) {
	if tnm[0] == '*' {
		tnm = tnm[1:]
	}
	ttp, npkg := gl.FindTypeName(tnm, fs, pkg)
	if ttp == nil {
		if TraceTypes {
			fmt.Printf("TExpr: error -- couldn't find type name: %v\n", tnm)
		}
		return nil, false
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
	if TraceTypes {
		fmt.Printf("TExpr: found type: %v  kind: %v\n", ttp.Name, ttp.Kind)
	}
	nxt := tyast
	for {
		nxti := walki.Next(nxt)
		if nxti == nil || nxti == last {
			if TraceTypes {
				fmt.Printf("TExpr: returning terminal type\n")
			}
			return ttp, true
		}
		nxt = nxti.(*parse.Ast)
		brk := false
		switch {
		case nxt.Nm == "Name":
			brk = true
		case strings.HasPrefix(nxt.Nm, "Slice"):
			eltyp := ttp.Els.ByName("val")
			if eltyp != nil {
				elnm := gl.QualifyType(pkgnm, eltyp.Type)
				if TraceTypes {
					fmt.Printf("TExpr: slice/map el type: %v\n", elnm)
				}
				return gl.TypeFromAstType(fs, origPkg, pkg, nxt, last, elnm)
			}
			if TraceTypes {
				fmt.Printf("TExpr: slice operator not on slice: %v\n", ttp.Name)
			}
		case nxt.Nm == "FuncCall":
			tnm := ttp.Name
			if pi := strings.Index(tnm, "."); pi > 0 {
				tnm = tnm[pi+1:]
			}
			tsym, got := pkg.Children.FindNameScoped(tnm)
			if got {
				fmt.Printf("got type sym: %v\n", tsym)
				nxti := walki.Next(nxt)
				if nxti == nil || nxti == last {
					return ttp, true
				}
				fun := nxti.(*parse.Ast)
				funm := fun.Src
				fmt.Printf("looking for function symbol: %v\n", funm)
				fsym, got := tsym.Children.FindNameScoped(funm)
				if got {
					fmt.Printf("got function sym: %v\n", fsym)
					// todo: get return value
					// multiple return values are going to be a pita.
				} else {
					fmt.Printf("did NOT get function sym: %v in pkg: %v\n", funm, pkg.Name)
				}
			} else {
				fmt.Printf("did NOT get type sym: %v in pkg: %v\n", tnm, pkg.Name)
			}
		}
		if brk {
			break
		}
		if TraceTypes {
			fmt.Printf("TExpr: skipping over %v\n", nxt.Nm)
		}
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
		ttp.WriteDoc(os.Stdout, 0)
	}
	return nil, false
}

/////////////////////////////////////////////////////////////////////////////////
//  Function / Params

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
func (gl *GoLang) FuncTypeFromAst(fs *pi.FileState, pkg *syms.Symbol, ast *parse.Ast, fty *syms.Type) {
	pars, err := ast.ChildAstTry(0)
	if err != nil {
		if TraceTypes {
			fmt.Println(err)
		}
		return
	}
	poff := 0
	if pars.Nm == "Name" && len(ast.Kids) > 1 {
		poff = 1
		pars = ast.ChildAst(1)
	}
	npars := len(pars.Kids)
	if npars > 0 && pars.Nm == "SigParams" {
		npars = 0 // not really
	}
	if npars > 0 {
		gl.ParamsFromAst(fs, pkg, pars, fty)
		npars = len(fty.Els) // how many we added..
	}
	nrvals := 0
	if len(ast.Kids) >= poff+2 {
		rvals := ast.ChildAst(poff + 1)
		nrvals := len(rvals.Kids)
		if nrvals == 1 { // single rval, unnamed, has type directly..
			rval := rvals.ChildAst(0)
			if rval.Nm != "ParName" {
				nrvals = 1
				rtyp, ok := gl.SubTypeFromAst(fs, pkg, rvals, 0)
				if ok {
					fty.Els.Add("rval", rtyp.Name)
					goto finalize
				}
			}
		}
		gl.ParamsFromAst(fs, pkg, rvals, fty)
		nrvals = len(fty.Els) - npars // how many we added..
	}
finalize:
	fty.Size = []int{npars, nrvals}
}

// ParamsFromAst sets params as Els for given function type (also for return types)
func (gl *GoLang) ParamsFromAst(fs *pi.FileState, pkg *syms.Symbol, pars *parse.Ast, fty *syms.Type) {
	npars := len(pars.Kids)
	var pnames []string // param names that all share same type
	for i := 0; i < npars; i++ {
		par := pars.Kids[i].(*parse.Ast)
		psz := len(par.Kids)
		if par.Nm == "ParType" && psz == 1 {
			ptypa := par.Kids[0].(*parse.Ast)
			if ptypa.Nm == "TypeNm" { // could be multiple args with same type or a separate type-only arg
				if _, istyp := pkg.Types[par.Src]; istyp {
					fty.Els.Add(fmt.Sprintf("par_%v", i), par.Src)
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
					fty.Els.Add(fmt.Sprintf("par_%v", i), ptyp.Name)
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
