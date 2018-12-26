// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"fmt"
	"strings"

	"github.com/goki/pi/parse"
	"github.com/goki/pi/pi"
	"github.com/goki/pi/syms"
)

var TraceTypes = false

// todo: when do we bind kind of type for external refs (qualified names)
// need to import everything (recursively) to get to the bottom of anything
// if we don't, then we have to do it repeatedly live -- would be better to
// do it and cache the results?  but only a problem when we have types
// defined in terms of other types -- maybe not all that relevant?

// FindTypeName finds given type name in pkg and in broader context
func (gl *GoLang) FindTypeName(tynm string, fs *pi.FileState, pkg *syms.Symbol) *syms.Type {
	if btyp, ok := BuiltinTypes[tynm]; ok {
		return btyp
	}
	if gtyp, ok := pkg.Types[tynm]; ok {
		return gtyp
	}
	sci := strings.Index(tynm, ".")
	if sci < 0 {
		return nil
	}
	scnm := tynm[:sci]
	if scpkg, ok := fs.Syms[scnm]; ok {
		pnm := tynm[sci+1:]
		if gtyp, ok := scpkg.Types[pnm]; ok {
			return gtyp
		}
	}
	return nil
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
		tyast, ok := ty.Ast.(*parse.Ast).ChildAst(1)
		if !ok {
			continue
		}
		gl.TypeFromAst(fs, pkg, ty, tyast)
	}
}

// SubTypeFromAst returns a subtype from child ast at given index, nil if failed
func (gl *GoLang) SubTypeFromAst(fs *pi.FileState, pkg *syms.Symbol, ast *parse.Ast, idx int) *syms.Type {
	sast, ok := ast.ChildAst(idx)
	if !ok {
		if TraceTypes {
			fmt.Printf("child not found at index: %v in ast node: %v\n", idx, ast.PathUnique())
		}
		return nil
	}
	sty := &syms.Type{}
	if ok = gl.TypeFromAst(fs, pkg, sty, sast); ok {
		return sty
	}
	// will have err msg already
	return nil
}

// TypeFromAst initializes the types from their Ast parse -- returns true if successful
func (gl *GoLang) TypeFromAst(fs *pi.FileState, pkg *syms.Symbol, ty *syms.Type, tyast *parse.Ast) bool {
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
			return false
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
			return false
		}
	case "QualType":
		ty.Els.Add("par", src) // not looking up external at this time -- kind remains unknown
		if ty.Name == "" {
			ty.Name = src
			pkg.Types.Add(ty)
		}
	case "PointerType":
		ty.Kind = syms.Ptr
		if sty := gl.SubTypeFromAst(fs, pkg, tyast, 0); sty != nil {
			ty.Els.Add("ptr", sty.Name)
			if ty.Name == "" {
				ty.Name = "*" + sty.Name
				pkg.Types.Add(ty)
			}
		} else {
			return false
		}
	case "MapType":
		ty.Kind = syms.Map
		keyty := gl.SubTypeFromAst(fs, pkg, tyast, 0)
		valty := gl.SubTypeFromAst(fs, pkg, tyast, 1)
		if keyty != nil && valty != nil {
			ty.Els.Add("key", keyty.Name)
			ty.Els.Add("val", valty.Name)
			if ty.Name == "" {
				ty.Name = "map[" + keyty.Name + "]" + valty.Name
				pkg.Types.Add(ty)
			}
		} else {
			return false
		}
	case "SliceType":
		ty.Kind = syms.List
		valty := gl.SubTypeFromAst(fs, pkg, tyast, 0)
		if valty != nil {
			ty.Els.Add("val", valty.Name)
			if ty.Name == "" {
				ty.Name = "[]" + valty.Name
				pkg.Types.Add(ty)
			}
		} else {
			return false
		}
	case "ArrayType":
		ty.Kind = syms.Array
		valty := gl.SubTypeFromAst(fs, pkg, tyast, 1)
		if valty != nil {
			ty.Els.Add("val", valty.Name)
			if ty.Name == "" {
				ty.Name = "[]" + valty.Name // todo: get size from child0, set to Size
				pkg.Types.Add(ty)
			}
		} else {
			return false
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
				fldty := gl.SubTypeFromAst(fs, pkg, fld, 1)
				if fldty != nil {
					ty.Els.Add(fsrc, fldty.Name)
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
				if nm, ok := fld.ChildAst(0); ok {
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
	default:
		if TraceTypes {
			fmt.Printf("Ast type node: %v not found\n", tyast.Nm)
		}
	}
	return true
}

// FuncTypeFromAst initializes a function type from ast -- type can either be anon
// or a named type -- if anon then the name is the full type signature without param names
func (gl *GoLang) FuncTypeFromAst(fs *pi.FileState, pkg *syms.Symbol, ast *parse.Ast, fty *syms.Type) {
	pars, ok := ast.ChildAst(0)
	if !ok {
		if TraceTypes {
			fmt.Printf("params not found at index: %v in ast node: %v\n", 0, ast.PathUnique())
		}
		return
	}
	poff := 0
	if pars.Nm == "Name" && len(ast.Kids) > 1 {
		poff = 1
		pars = ast.KnownChildAst(1)
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
		rvals := ast.KnownChildAst(poff + 1)
		nrvals := len(rvals.Kids)
		if nrvals == 1 { // single rval, unnamed, has type directly..
			rval := rvals.KnownChildAst(0)
			if rval.Nm != "ParName" {
				nrvals = 1
				rtyp := gl.SubTypeFromAst(fs, pkg, rvals, 0)
				if rtyp != nil {
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
				ptyp := gl.SubTypeFromAst(fs, pkg, par, 0)
				if ptyp != nil {
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
			ptyp := gl.SubTypeFromAst(fs, pkg, par, 1)
			if ptyp != nil {
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
