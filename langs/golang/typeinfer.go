// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"fmt"
	"os"

	"github.com/goki/pi/parse"
	"github.com/goki/pi/pi"
	"github.com/goki/pi/syms"
	"github.com/goki/pi/token"
)

// TypeErr indicates is the type name we use to indicate that the type could not be inferred
var TypeErr = "<err>"

// InferSymbolType infers the symbol types for given symbol and all of its children
// funInternal determines whether to include function-internal symbols
// (e.g., variables within function scope -- only for local files).
func (gl *GoLang) InferSymbolType(sy *syms.Symbol, fs *pi.FileState, pkg *syms.Symbol, funInternal bool) {
	if sy.Ast != nil {
		ast := sy.Ast.(*parse.Ast)
		switch {
		case sy.Kind == token.NameField:
			stsc, ok := sy.Scopes[token.NameStruct]
			if ok {
				stty, _ := gl.FindTypeName(stsc, fs, pkg)
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
			// todo: unclear why NameVarGlobal types not working here
			// if TraceTypes && sy.Kind == token.NameVarGlobal {
			// 	fmt.Printf("processing NVG: %v\n", sy.String())
			// }
			astyp := ast.ChildAst(len(ast.Kids) - 1)
			vty, ok := gl.TypeFromAst(fs, pkg, nil, astyp)
			if ok {
				sy.Type = vty.Name
				// if TraceTypes {
				// 	fmt.Printf("namevar: %v  type: %v from ast\n", sy.Name, sy.Type)
				// }
			} else {
				sy.Type = TypeErr // actively mark as err so not re-processed
				if TraceTypes {
					fmt.Printf("InferSymbolType: NameVar: %v NOT resolved from ast: %v\n", sy.Name, astyp.PathUnique())
					astyp.WriteTree(os.Stdout, 1)
				}
			}
		case sy.Kind.SubCat() == token.NameType:
			vty, _ := gl.FindTypeName(sy.Name, fs, pkg)
			if vty != nil {
				sy.Type = vty.Name
			} else {
				// if TraceTypes {
				// 	fmt.Printf("InferSymbolType: NameType: %v\n", sy.Name)
				// }
				astyp := ast.ChildAst(len(ast.Kids) - 1)
				if astyp.Nm == "FieldTag" {
					// ast.WriteTree(os.Stdout, 1)
					astyp = ast.ChildAst(len(ast.Kids) - 2)
				}
				vty, ok := gl.TypeFromAst(fs, pkg, nil, astyp)
				if ok {
					sy.Type = vty.Name
					// if TraceTypes {
					// 	fmt.Printf("InferSymbolType: NameType: %v  type: %v from ast\n", sy.Name, sy.Type)
					// }
				} else {
					sy.Type = TypeErr // actively mark as err so not re-processed
					if TraceTypes {
						fmt.Printf("InferSymbolType: NameType: %v NOT resolved from ast: %v\n", sy.Name, astyp.PathUnique())
						ast.WriteTree(os.Stdout, 1)
					}
				}
			}
		case sy.Kind == token.NameFunction:
			ftyp := gl.FuncTypeFromAst(fs, pkg, ast, nil)
			if ftyp != nil {
				ftyp.Name = "func " + sy.Name
				sy.Type = ftyp.Name
				pkg.Types.Add(ftyp)
				sy.Detail = ftyp.ArgString() + " " + ftyp.ReturnString()
				// if TraceTypes {
				// 	fmt.Printf("InferSymbolType: added function type: %v  %v\n", ftyp.Name, ftyp.String())
				// }
			}
		}
	}
	if !funInternal && sy.Kind.SubCat() == token.NameFunction {
		sy.Children = nil // nuke!
	} else {
		for _, ss := range sy.Children {
			if ss != sy {
				// if TraceTypes {
				// 	fmt.Printf("InferSymbolType: processing child: %v\n", ss)
				// }
				gl.InferSymbolType(ss, fs, pkg, funInternal)
			}
		}
	}
}
