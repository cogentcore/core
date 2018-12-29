// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"github.com/goki/pi/parse"
	"github.com/goki/pi/pi"
	"github.com/goki/pi/syms"
	"github.com/goki/pi/token"
)

// InferSymbolType infers the symbol types for given symbol and all of its children
func (gl *GoLang) InferSymbolType(sy *syms.Symbol, fs *pi.FileState, pkg *syms.Symbol) {
	if sy.Ast != nil {
		ast := sy.Ast.(*parse.Ast)
		switch {
		case sy.Kind == token.NameField:
			stsc, ok := sy.Scopes[token.NameStruct]
			if ok {
				stty := gl.FindTypeName(stsc, fs, pkg)
				if stty != nil {
					fldel := stty.Els.ByName(sy.Name)
					if fldel != nil {
						sy.Type = fldel.Type
					}
				}
				if sy.Type == "" {
					sy.Type = stsc + "." + sy.Name
				}
			}
		case sy.Kind == token.NameVarClass: // method receiver
			stsc, ok := sy.Scopes[token.NameStruct]
			if ok {
				sy.Type = stsc
			}
		case sy.Kind.SubCat() == token.NameVar:
			vty := gl.SubTypeFromAst(fs, pkg, ast, len(ast.Kids)-1) // type always last thing
			if vty != nil {
				sy.Type = vty.Name
			}
		case sy.Kind.SubCat() == token.NameType:
			vty := gl.FindTypeName(sy.Name, fs, pkg)
			if vty != nil {
				sy.Type = vty.Name
			} else {
				sy.Type = sy.Name // should be anyway..
			}
		}
	}
	for _, ss := range sy.Children {
		if ss != sy {
			gl.InferSymbolType(ss, fs, pkg)
		}
	}
}
