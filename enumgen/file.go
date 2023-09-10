// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on http://github.com/dmarkham/enumer and
// golang.org/x/tools/cmd/stringer:

// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enumgen

import (
	"errors"
	"go/ast"
	exact "go/constant"
	"go/token"
	"go/types"
	"html"
	"strings"
)

// GenDecl processes one declaration clause.
// It returns whether the AST inspector should continue,
// and an error if there is one. It should only be
// called in [ast.Inspect].
func (g *Generator) GenDecl(node ast.Node, file *ast.File, typ *Type) ([]Value, bool, error) {
	decl, ok := node.(*ast.GenDecl)
	if !ok || decl.Tok != token.CONST {
		// We only care about const declarations.
		return nil, true, nil
	}
	vals := []Value{}
	// The name of the type of the constants we are declaring.
	// Can change if this is a multi-element declaration.
	typName := ""
	// Loop over the elements of the declaration. Each element is a ValueSpec:
	// a list of names possibly followed by a type, possibly followed by values.
	// If the type and value are both missing, we carry down the type (and value,
	// but the "go/types" package takes care of that).
	for _, spec := range decl.Specs {
		vspec := spec.(*ast.ValueSpec) // Guaranteed to succeed as this is CONST.
		if vspec.Type == nil && len(vspec.Values) > 0 {
			// "X = 1". With no type but a value, the constant is untyped.
			// Skip this vspec and reset the remembered type.
			typName = ""
			continue
		}
		if vspec.Type != nil {
			// "X T". We have a type. Remember it.
			ident, ok := vspec.Type.(*ast.Ident)
			if !ok {
				continue
			}
			typName = ident.Name
		}
		if typName != typ.Name {
			// This is not the type we're looking for.
			continue
		}
		// We now have a list of names (from one line of source code) all being
		// declared with the desired type.
		// Grab their names and actual values and store them in f.values.
		for _, n := range vspec.Names {
			if n.Name == "_" {
				continue
			}
			// This dance lets the type checker find the values for us. It's a
			// bit tricky: look up the object declared by the n, find its
			// types.Const, and extract its value.
			obj, ok := g.Pkg.TypesInfo.Defs[n]
			if !ok {
				return nil, false, errors.New("no value for constant " + n.String())
			}
			info := obj.Type().Underlying().(*types.Basic).Info()
			if info&types.IsInteger == 0 {
				return nil, false, errors.New("can't handle non-integer constant type " + typName)
			}
			value := obj.(*types.Const).Val() // Guaranteed to succeed as this is CONST.
			if value.Kind() != exact.Int {
				return nil, false, errors.New("can't happen: constant is not an integer " + n.String())
			}
			i64, isInt := exact.Int64Val(value)
			u64, isUint := exact.Uint64Val(value)
			if !isInt && !isUint {
				return nil, false, errors.New("internal error: value of " + n.String() + " is not an integer: " + value.String())
			}
			if !isInt {
				u64 = uint64(i64)
			}
			v := Value{
				OriginalName: n.Name,
				Name:         n.Name,
				Desc:         html.EscapeString(strings.Join(strings.Fields(vspec.Doc.Text()), " ")), // need to collapse whitespace and escape
				Value:        u64,
				Signed:       info&types.IsUnsigned == 0,
				Str:          value.String(),
			}
			if c := vspec.Comment; typ.Config.LineComment && c != nil && len(c.List) == 1 {
				v.Name = strings.TrimSpace(c.Text())
			}

			vals = append(vals, v)
		}
	}
	return vals, false, nil
}
