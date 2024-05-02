// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
package alignsl performs 16-byte alignment checking of struct fields
and total size modulus checking of struct types to ensure HLSL
(and GSL) compatibility.

Checks that struct sizes are an even multiple of 16 bytes
(4 float32's), fields are 32 bit types: [U]Int32, Float32,
and that fields that are other struct types are aligned
at even 16 byte multiples.
*/
package alignsl

import (
	"errors"
	"fmt"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Context for given package run
type Context struct {
	Sizes   types.Sizes              // from package
	Structs map[*types.Struct]string // structs that have been processed already -- value is name
	Stack   map[*types.Struct]string // structs to process in a second pass -- structs encountered during processing of other structs
	Errs    []string                 // accumulating list of error strings -- empty if all good
}

func NewContext(sz types.Sizes) *Context {
	cx := &Context{Sizes: sz}
	cx.Structs = make(map[*types.Struct]string)
	cx.Stack = make(map[*types.Struct]string)
	return cx
}

func (cx *Context) IsNewStruct(st *types.Struct) bool {
	if _, has := cx.Structs[st]; has {
		return false
	}
	cx.Structs[st] = st.String()
	return true
}

func (cx *Context) AddError(ers string, hasErr bool, stName string) bool {
	if !hasErr {
		cx.Errs = append(cx.Errs, stName)
	}
	cx.Errs = append(cx.Errs, ers)
	return true
}

func TypeName(tp types.Type) string {
	switch x := tp.(type) {
	case *types.Named:
		return x.Obj().Name()
	}
	return tp.String()
}

// CheckStruct is the primary checker -- returns hasErr = true if there
// are any mis-aligned fields or total size of struct is not an
// even multiple of 16 bytes -- adds details to Errs
func CheckStruct(cx *Context, st *types.Struct, stName string) bool {
	if !cx.IsNewStruct(st) {
		return false
	}
	var flds []*types.Var
	nf := st.NumFields()
	if nf == 0 {
		return false
	}
	hasErr := false
	for i := 0; i < nf; i++ {
		fl := st.Field(i)
		flds = append(flds, fl)
		ft := fl.Type()
		ut := ft.Underlying()
		if bt, isBasic := ut.(*types.Basic); isBasic {
			kind := bt.Kind()
			if !(kind == types.Uint32 || kind == types.Int32 || kind == types.Float32 || kind == types.Uint64) {
				hasErr = cx.AddError(fmt.Sprintf("    %s:  basic type != [U]Int32 or Float32: %s", fl.Name(), bt.String()), hasErr, stName)
			}
		} else {
			if sst, is := ut.(*types.Struct); is {
				cx.Stack[sst] = TypeName(ft)
			} else {
				hasErr = cx.AddError(fmt.Sprintf("    %s:  unsupported type: %s", fl.Name(), ft.String()), hasErr, stName)
			}
		}
	}
	offs := cx.Sizes.Offsetsof(flds)
	last := cx.Sizes.Sizeof(flds[nf-1].Type())
	totsz := int(offs[nf-1] + last)
	mod := totsz % 16
	if mod != 0 {
		needs := 4 - (mod / 4)
		hasErr = cx.AddError(fmt.Sprintf("    total size: %d not even multiple of 16 -- needs %d extra 32bit padding fields", totsz, needs), hasErr, stName)
	}

	// check that struct starts at mod 16 byte offset
	for i, fl := range flds {
		ft := fl.Type()
		ut := ft.Underlying()
		if _, is := ut.(*types.Struct); is {
			off := offs[i]
			if off%16 != 0 {

				hasErr = cx.AddError(fmt.Sprintf("    %s:  struct type: %s is not at mod-16 byte offset: %d", fl.Name(), TypeName(ft), off), hasErr, stName)
			}
		}
	}

	return hasErr
}

// CheckPackage is main entry point for checking a package
// returns error string if any errors found.
func CheckPackage(pkg *packages.Package) error {
	cx := NewContext(pkg.TypesSizes)
	sc := pkg.Types.Scope()
	hasErr := CheckScope(cx, sc, 0)
	er := CheckStack(cx)
	if hasErr || er {
		str := `
WARNING: in struct type alignment checking:
    Checks that struct sizes are an even multiple of 16 bytes (4 float32's),
    and fields are 32 bit types: [U]Int32, Float32 or other struct,
    and that fields that are other struct types are aligned at even 16 byte multiples.
    List of errors found follow below, by struct type name:
` + strings.Join(cx.Errs, "\n")
		return errors.New(str)
	}
	return nil
}

func CheckStack(cx *Context) bool {
	hasErr := false
	for {
		if len(cx.Stack) == 0 {
			break
		}
		st := cx.Stack
		cx.Stack = make(map[*types.Struct]string) // new stack
		for st, nm := range st {
			er := CheckStruct(cx, st, nm)
			if er {
				hasErr = true
			}
		}
	}
	return hasErr
}

func CheckScope(cx *Context, sc *types.Scope, level int) bool {
	nms := sc.Names()
	ntyp := 0
	hasErr := false
	for _, nm := range nms {
		ob := sc.Lookup(nm)
		tp := ob.Type()
		if tp == nil {
			continue
		}
		if nt, is := tp.(*types.Named); is {
			ut := nt.Underlying()
			if ut == nil {
				continue
			}
			if st, is := ut.(*types.Struct); is {
				er := CheckStruct(cx, st, nt.Obj().Name())
				if er {
					hasErr = true
				}
				ntyp++
			}
		}
	}
	if ntyp == 0 {
		for i := 0; i < sc.NumChildren(); i++ {
			cs := sc.Child(i)
			er := CheckScope(cx, cs, level+1)
			if er {
				hasErr = true
			}
		}
	}
	return hasErr
}
