// Copyright (c) 2020, The GoKi Authors. All rights reserved.
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

// UnQualifyType returns the type name with any qualifier removed
func (gl *GoLang) UnQualifyType(tnm string) string {
	pi := strings.Index(tnm, ".")
	if pi < 0 {
		return tnm
	}
	return tnm[pi+1:]
}

// FindTypeName finds given type name in pkg and in broader context
// returns new package symbol if type name is in a different package
// else returns pkg arg.
func (gl *GoLang) FindTypeName(tynm string, fs *pi.FileState, pkg *syms.Symbol) (*syms.Type, *syms.Symbol) {
	if tynm == "" {
		return nil, nil
	}
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
		tnm := tynm[sci+1:]
		if gtyp, ok := npkg.Types[tnm]; ok {
			return gtyp, npkg
		}
		if TraceTypes {
			fmt.Printf("FindTypeName: type name: %v not found in package: %v\n", tnm, pnm)
		}
	} else {
		if TraceTypes {
			fmt.Printf("FindTypeName: could not find package: %v\n", pnm)
		}
	}
	if TraceTypes {
		fmt.Printf("FindTypeName: type name: %v not found in package: %v\n", tynm, pkg.Name)
	}
	return nil, pkg
}

// ResolveTypes initializes all user-defined types from Ast data
// and then resolves types of symbols.  The pkg must be a single
// package symbol i.e., the children there are all the elements of the
// package and the types are all the global types within the package.
// funInternal determines whether to include function-internal symbols
// (e.g., variables within function scope -- only for local files).
func (gl *GoLang) ResolveTypes(fs *pi.FileState, pkg *syms.Symbol, funInternal bool) {
	gl.TypesFromAst(fs, pkg)
	gl.InferSymbolType(pkg, fs, pkg, funInternal)
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
		if ty.Name == "" {
			if TraceTypes {
				fmt.Printf("TypesFromAst: Type has no name! %v\n", ty.String())
			}
			continue
		}
		gl.TypeFromAst(fs, pkg, ty, tyast)
		gl.TypeMeths(fs, pkg, ty) // all top-level named types might have methods
	}
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

// TypeToKindMap maps Ast type names to syms.Kind basic categories for how we
// treat them for subsequent processing.  Basically: Primitive or Composite
var TypeToKindMap = map[string]syms.Kinds{
	"BasicType":     syms.Primitive,
	"TypeNm":        syms.Primitive,
	"QualType":      syms.Primitive,
	"PointerType":   syms.Primitive,
	"MapType":       syms.Composite,
	"SliceType":     syms.Composite,
	"ArrayType":     syms.Composite,
	"StructType":    syms.Composite,
	"InterfaceType": syms.Composite,
	"FuncType":      syms.Composite,
}

// AstTypeName returns the effective type name from ast node
// dropping the "Lit" for example.
func (gl *GoLang) AstTypeName(tyast *parse.Ast) string {
	tnm := tyast.Nm
	if strings.HasPrefix(tnm, "Lit") {
		tnm = tnm[3:]
	}
	return tnm
}

// TypeFromAst returns type from Ast parse -- returns true if successful.
// This is used both for initialization of global types via TypesFromAst
// and also for online type processing in the course of tracking down
// other types while crawling the Ast.  In the former case, ty is non-nil
// and the goal is to fill out the type information -- the ty will definitely
// have a name already.  In the latter case, the ty will be nil, but the
// tyast node may have a Src name that will first be looked up to determine
// if a previously-processed type is already available.  The tyast.Name is
// the parser categorization of the type  (BasicType, StructType, etc).
func (gl *GoLang) TypeFromAst(fs *pi.FileState, pkg *syms.Symbol, ty *syms.Type, tyast *parse.Ast) (*syms.Type, bool) {
	tnm := gl.AstTypeName(tyast)
	bkind, ok := TypeToKindMap[tnm]
	if !ok { // must be some kind of expression
		return gl.TypeFromAstExpr(fs, pkg, pkg, tyast)
	}
	switch bkind {
	case syms.Primitive:
		return gl.TypeFromAstPrim(fs, pkg, ty, tyast)
	case syms.Composite:
		return gl.TypeFromAstComp(fs, pkg, ty, tyast)

	}
	return nil, false
}

// TypeFromAstPrim handles primitive (non composite) type processing
func (gl *GoLang) TypeFromAstPrim(fs *pi.FileState, pkg *syms.Symbol, ty *syms.Type, tyast *parse.Ast) (*syms.Type, bool) {
	tnm := gl.AstTypeName(tyast)
	src := tyast.Src
	etyp, _ := gl.FindTypeName(src, fs, pkg)
	if etyp != nil {
		if ty == nil { // if we can find an existing type, and not filling in global, use it
			return etyp, true
		}
	} else {
		if TraceTypes && src != "" {
			fmt.Printf("TypeFromAst: primitive type name: %v not found\n", src)
		}
	}
	switch tnm {
	case "BasicType":
		if etyp != nil {
			ty.Kind = etyp.Kind
			ty.Els.Add("par", etyp.Name) // parent type
			return ty, true
		} else {
			return nil, false
		}
	case "TypeNm", "QualType":
		if etyp != nil && etyp != ty {
			ty.Kind = etyp.Kind
			if ty.Name != etyp.Name {
				ty.Els.Add("par", etyp.Name) // parent type
				if TraceTypes {
					fmt.Printf("TypeFromAst: TypeNm %v defined from parent type: %v\n", ty.Name, etyp.Name)
				}
			}
			return ty, true
		} else {
			return nil, false
		}
	case "PointerType":
		if ty == nil {
			ty = &syms.Type{}
		}
		ty.Kind = syms.Ptr
		if sty, ok := gl.SubTypeFromAst(fs, pkg, tyast, 0); ok {
			ty.Els.Add("ptr", sty.Name)
			if ty.Name == "" {
				ty.Name = "*" + sty.Name
				pkg.Types.Add(ty) // add pointers so we don't have to keep redefining
				if TraceTypes {
					fmt.Printf("TypeFromAst: Adding PointerType %v\n", ty.String())
				}
			}
			return ty, true
		} else {
			return nil, false
		}
	}
	return nil, false
}

// TypeFromAstComp handles composite type processing
func (gl *GoLang) TypeFromAstComp(fs *pi.FileState, pkg *syms.Symbol, ty *syms.Type, tyast *parse.Ast) (*syms.Type, bool) {
	tnm := gl.AstTypeName(tyast)
	newTy := false
	if ty == nil {
		newTy = true
		tn := fs.NextAnonName()
		ty = &syms.Type{Name: tn}
	}
	switch tnm {
	case "MapType":
		ty.Kind = syms.Map
		if newTy {
			ty.Name += "_map"
		}
		keyty, kok := gl.SubTypeFromAst(fs, pkg, tyast, 0)
		valty, vok := gl.SubTypeFromAst(fs, pkg, tyast, 1)
		if kok && vok {
			ty.Els.Add("key", keyty.Name)
			ty.Els.Add("val", valty.Name)
			if ty.Name == "" {
				ty.Name = "map[" + keyty.Name + "]" + valty.Name
			}
		} else {
			return nil, false
		}
	case "SliceType":
		ty.Kind = syms.List
		if newTy {
			ty.Name += "_slice"
		}
		valty, ok := gl.SubTypeFromAst(fs, pkg, tyast, 0)
		if ok {
			ty.Els.Add("val", valty.Name)
			if ty.Name == "" {
				ty.Name = "[]" + valty.Name
			}
		} else {
			return nil, false
		}
	case "ArrayType":
		ty.Kind = syms.Array
		if newTy {
			ty.Name += "_array"
		}
		valty, ok := gl.SubTypeFromAst(fs, pkg, tyast, 1)
		if ok {
			ty.Els.Add("val", valty.Name)
			if ty.Name == "" {
				ty.Name = "[]" + valty.Name // todo: get size from child0, set to Size
			}
		} else {
			return nil, false
		}
	case "StructType":
		ty.Kind = syms.Struct
		if newTy {
			ty.Name += "_struct"
		}
		nfld := len(tyast.Kids)
		ty.Size = []int{nfld}
		tynm := ""
		hasName := ty.Name != ""
		for i := 0; i < nfld; i++ {
			fld := tyast.Kids[i].(*parse.Ast)
			fsrc := fld.Src
			if !hasName {
				tynm += fsrc + ";"
			}
			switch fld.Nm {
			case "NamedField":
				if len(fld.Kids) <= 1 { // anonymous, non-qualified
					ty.Els.Add(fsrc, fsrc)
					atyp, _ := gl.FindTypeName(fsrc, fs, pkg)
					if atyp != nil {
						ty.Els.CopyFrom(atyp.Els)
						ty.Size[0] += len(atyp.Els)
						ty.Meths.CopyFrom(atyp.Meths)
						// if TraceTypes {
						// 	fmt.Printf("Struct Type: %v inheriting from: %v\n", ty.Name, atyp.Name)
						// }
					}
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
				if !hasName {
					tynm += fsrc + ";"
				}
				atyp, _ := gl.FindTypeName(fsrc, fs, pkg)
				if atyp != nil {
					ty.Els.CopyFrom(atyp.Els)
					ty.Size[0] += len(atyp.Els)
					ty.Meths.CopyFrom(atyp.Meths)
					// if TraceTypes {
					// 	fmt.Printf("Struct Type: %v inheriting from: %v\n", ty.Name, atyp.Name)
					// }
				}
			}
		}
		if !hasName {
			ty.Name = "struct{" + tynm + "}"
		}
		// if TraceTypes {
		// 	fmt.Printf("TypeFromAst: New struct type defined: %v\n", ty.Name)
		// 	ty.WriteDoc(os.Stdout, 0)
		// }
	case "InterfaceType":
		ty.Kind = syms.Interface
		if newTy {
			ty.Name += "_interface"
		}
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
					mty := syms.NewType(ty.Name+":"+nm.Src, syms.Method)
					pkg.Types.Add(mty)                    // add interface methods as new types..
					gl.FuncTypeFromAst(fs, pkg, fld, mty) // todo: this is not working -- debug
					ty.Els.Add(nm.Src, mty.Name)
				}
			}
		}
	case "FuncType":
		ty.Kind = syms.Func
		if newTy {
			ty.Name += "_func"
		}
		gl.FuncTypeFromAst(fs, pkg, tyast, ty)
	}
	// if TraceTypes && newTy {
	// 	fmt.Printf("TypeFromAstComp: Created new composite type: %s\n", ty.String())
	// }
	return ty, true // fallthrough is true..
}
