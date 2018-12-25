// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"fmt"

	"github.com/goki/pi/parse"
	"github.com/goki/pi/syms"
)

var TraceTypes = true

// todo: FindType -- lookup across all imports -- need 2 args -- one for global
// including all packages and the one just for this file -- that is the typical
// case
//
// todo: need to detect loops here or else!
//
// also, when do we bind kind of type for external refs (qualified names)
// need to import everything (recursively) to get to the bottom of anything
// if we don't, then we have to do it repeatedly live -- would be better to
// do it and cache the results?  but only a problem when we have types
// defined in terms of other types -- maybe not all that relevant?

// ResolveTypes initializes all user-defined types from Ast data
// and then resolves types of symbols
func (gl *GoLang) ResolveTypes(pkgsym *syms.Symbol) {
	gl.TypesFromAst(pkgsym)
}

// TypesFromAst initializes the types from their Ast parse
func (gl *GoLang) TypesFromAst(pkgsym *syms.Symbol) {
	InstallBuiltinTypes()

	for _, ty := range pkgsym.Types {
		if ty.Ast == nil {
			continue // shouldn't be
		}
		tyasti, ok := ty.Ast.Child(1)
		if !ok {
			continue
		}
		tyast := tyasti.(*parse.Ast)
		gl.TypeFromAst(pkgsym, ty, tyast)
	}
}

// SubTypeFromAst returns a subtype from child ast at given index, nil if failed
func (gl *GoLang) SubTypeFromAst(pkgsym *syms.Symbol, tyast *parse.Ast, idx int) *syms.Type {
	sasti, ok := tyast.Child(idx)
	if !ok {
		if TraceTypes {
			fmt.Printf("child not found at index: %v in ast node: %v\n", idx, tyast.PathUnique())
		}
		return nil
	}
	sast := sasti.(*parse.Ast)
	sty := &syms.Type{}
	if ok = gl.TypeFromAst(pkgsym, sty, sast); ok {
		return sty
	}
	// will have err msg already
	return nil
}

// TypeFromAst initializes the types from their Ast parse -- returns true if successful
func (gl *GoLang) TypeFromAst(pkgsym *syms.Symbol, ty *syms.Type, tyast *parse.Ast) bool {
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
		if btyp, ok := pkgsym.Types[src]; ok {
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
			pkgsym.Types.Add(ty)
		}
	case "PointerType":
		ty.Kind = syms.Ptr
		if sty := gl.SubTypeFromAst(pkgsym, tyast, 0); sty != nil {
			ty.Els.Add("ptr", sty.Name)
			if ty.Name == "" {
				ty.Name = "*" + sty.Name
				pkgsym.Types.Add(ty)
			}
		} else {
			return false
		}
	case "MapType":
		ty.Kind = syms.Map
		keyty := gl.SubTypeFromAst(pkgsym, tyast, 0)
		valty := gl.SubTypeFromAst(pkgsym, tyast, 1)
		if keyty != nil && valty != nil {
			ty.Els.Add("key", keyty.Name)
			ty.Els.Add("val", valty.Name)
			if ty.Name == "" {
				ty.Name = "map[" + keyty.Name + "]" + valty.Name
				pkgsym.Types.Add(ty)
			}
		} else {
			return false
		}
	case "SliceType":
		ty.Kind = syms.List
		valty := gl.SubTypeFromAst(pkgsym, tyast, 0)
		if valty != nil {
			ty.Els.Add("val", valty.Name)
			if ty.Name == "" {
				ty.Name = "[]" + valty.Name
				pkgsym.Types.Add(ty)
			}
		} else {
			return false
		}
	case "ArrayType":
		ty.Kind = syms.Array
		valty := gl.SubTypeFromAst(pkgsym, tyast, 1)
		if valty != nil {
			ty.Els.Add("val", valty.Name)
			if ty.Name == "" {
				ty.Name = "[]" + valty.Name // todo: get size from child0
				pkgsym.Types.Add(ty)
			}
		} else {
			return false
		}
	case "StructType":
		ty.Kind = syms.Struct
		// todo: loop!
	case "InterfaceType":
		ty.Kind = syms.Interface
		// todo: loop!
	default:
		if TraceTypes {
			fmt.Printf("Ast type node: %v not found\n", tyast.Nm)
		}
	}
	return true
}
