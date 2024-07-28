// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"fmt"
	"os"
	"strings"

	"cogentcore.org/core/parse"
	"cogentcore.org/core/parse/parser"
	"cogentcore.org/core/parse/syms"
	"cogentcore.org/core/parse/token"
)

// TypeErr indicates is the type name we use to indicate that the type could not be inferred
var TypeErr = "<err>"

// TypeInProcess indicates is the type name we use to indicate that the type
// is currently being processed -- prevents loops
var TypeInProcess = "<in-process>"

// InferSymbolType infers the symbol types for given symbol and all of its children
// funInternal determines whether to include function-internal symbols
// (e.g., variables within function scope -- only for local files).
func (gl *GoLang) InferSymbolType(sy *syms.Symbol, fs *parse.FileState, pkg *syms.Symbol, funInternal bool) {
	if sy.Name == "" {
		sy.Type = TypeErr
		return
	}
	if sy.Name[0] == '_' {
		sy.Type = TypeErr
		return
	}
	if sy.AST != nil {
		ast := sy.AST.(*parser.AST)
		switch {
		case sy.Kind == token.NameField:
			stsc, ok := sy.Scopes[token.NameStruct]
			if ok {
				stty, _ := gl.FindTypeName(stsc, fs, pkg)
				if stty != nil {
					fldel := stty.Els.ByName(sy.Name)
					if fldel != nil {
						sy.Type = fldel.Type
						// fmt.Printf("set field type: %s\n", sy.Label())
					} else {
						if TraceTypes {
							fmt.Printf("InferSymbolType: field named: %v not found in struct type: %v\n", sy.Name, stty.Name)
						}
					}
				} else {
					if TraceTypes {
						fmt.Printf("InferSymbolType: field named: %v struct type: %v not found\n", sy.Name, stsc)
					}
				}
				if sy.Type == "" {
					sy.Type = stsc + "." + sy.Name
				}
			} else {
				if TraceTypes {
					fmt.Printf("InferSymbolType: field named: %v doesn't have NameStruct scope\n", sy.Name)
				}
			}
		case sy.Kind == token.NameVarClass: // method receiver
			stsc, ok := sy.Scopes.SubCat(token.NameType)
			if ok {
				sy.Type = stsc
			}
		case sy.Kind.SubCat() == token.NameVar:
			var astyp *parser.AST
			if ast.HasChildren() {
				if strings.HasPrefix(ast.Name, "ForRange") {
					gl.InferForRangeSymbolType(sy, fs, pkg)
				} else {
					astyp = ast.ChildAST(len(ast.Children) - 1)
					vty, ok := gl.TypeFromAST(fs, pkg, nil, astyp)
					if ok {
						sy.Type = SymTypeNameForPkg(vty, pkg)
						// if TraceTypes {
						// 	fmt.Printf("namevar: %v  type: %v from ast\n", sy.Name, sy.Type)
						// }
					} else {
						sy.Type = TypeErr // actively mark as err so not re-processed
						if TraceTypes {
							fmt.Printf("InferSymbolType: NameVar: %v NOT resolved from ast: %v\n", sy.Name, astyp.Path())
							astyp.WriteTree(os.Stdout, 0)
						}
					}
				}
			} else {
				sy.Type = TypeErr
				if TraceTypes {
					fmt.Printf("InferSymbolType: NameVar: %v has no children\n", sy.Name)
				}
			}
		case sy.Kind == token.NameConstant:
			if !strings.HasPrefix(ast.Name, "ConstSpec") {
				if TraceTypes {
					fmt.Printf("InferSymbolType: NameConstant: %v not a const: %v\n", sy.Name, ast.Name)
				}
				return
			}
			parent := ast.ParentAST()
			if parent != nil && parent.HasChildren() {
				fc := parent.ChildAST(0)
				if fc.HasChildren() {
					ffc := fc.ChildAST(0)
					if ffc.Name == "Name" {
						ffc = ffc.NextAST()
					}
					var vty *syms.Type
					if ffc != nil {
						vty, _ = gl.TypeFromAST(fs, pkg, nil, ffc)
					}
					if vty != nil {
						sy.Type = SymTypeNameForPkg(vty, pkg)
					} else {
						sy.Type = TypeErr
						if TraceTypes {
							fmt.Printf("InferSymbolType: NameConstant: %v NOT resolved from ast: %v\n", sy.Name, ffc.Path())
							ffc.WriteTree(os.Stdout, 1)
						}
					}
				} else {
					sy.Type = TypeErr
				}
			} else {
				sy.Type = TypeErr
			}
		case sy.Kind.SubCat() == token.NameType:
			vty, _ := gl.FindTypeName(sy.Name, fs, pkg)
			if vty != nil {
				sy.Type = SymTypeNameForPkg(vty, pkg)
			} else {
				// if TraceTypes {
				// 	fmt.Printf("InferSymbolType: NameType: %v\n", sy.Name)
				// }
				if ast.HasChildren() {
					astyp := ast.ChildAST(len(ast.Children) - 1)
					if astyp.Name == "FieldTag" {
						// ast.WriteTree(os.Stdout, 1)
						astyp = ast.ChildAST(len(ast.Children) - 2)
					}
					vty, ok := gl.TypeFromAST(fs, pkg, nil, astyp)
					if ok {
						sy.Type = SymTypeNameForPkg(vty, pkg)
						// if TraceTypes {
						// 	fmt.Printf("InferSymbolType: NameType: %v  type: %v from ast\n", sy.Name, sy.Type)
						// }
					} else {
						sy.Type = TypeErr // actively mark as err so not re-processed
						if TraceTypes {
							fmt.Printf("InferSymbolType: NameType: %v NOT resolved from ast: %v\n", sy.Name, astyp.Path())
							ast.WriteTree(os.Stdout, 1)
						}
					}
				} else {
					sy.Type = TypeErr
				}
			}
		case sy.Kind == token.NameFunction:
			ftyp := gl.FuncTypeFromAST(fs, pkg, ast, nil)
			if ftyp != nil {
				ftyp.Name = "func " + sy.Name
				ftyp.Filename = sy.Filename
				ftyp.Region = sy.Region
				sy.Type = ftyp.Name
				pkg.Types.Add(ftyp)
				sy.Detail = "(" + ftyp.ArgString() + ") " + ftyp.ReturnString()
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
			if sy != ss {
				if false && TraceTypes {
					fmt.Printf("InferSymbolType: processing child: %v\n", ss)
				}
				gl.InferSymbolType(ss, fs, pkg, funInternal)
			}
		}
	}
}

// InferForRangeSymbolType infers the type of a ForRange expr
// gets the container type properly
func (gl *GoLang) InferForRangeSymbolType(sy *syms.Symbol, fs *parse.FileState, pkg *syms.Symbol) {
	ast := sy.AST.(*parser.AST)
	if ast.NumChildren() < 2 {
		sy.Type = TypeErr // actively mark as err so not re-processed
		if TraceTypes {
			fmt.Printf("InferSymbolType: ForRange NameVar: %v does not have expected 2+ children\n", sy.Name)
			ast.WriteTree(os.Stdout, 0)
		}
		return
	}
	// vars are in first child, type is in second child, rest of code is on last node
	astyp := ast.ChildAST(1)
	vty, ok := gl.TypeFromAST(fs, pkg, nil, astyp)
	if !ok {
		sy.Type = TypeErr // actively mark as err so not re-processed
		if TraceTypes {
			fmt.Printf("InferSymbolType: NameVar: %v NOT resolved from ForRange ast: %v\n", sy.Name, astyp.Path())
			astyp.WriteTree(os.Stdout, 0)
		}
		return
	}

	varidx := 1 // which variable are we: first or second?
	vast := ast.ChildAST(0)
	if vast.NumChildren() <= 1 {
		varidx = 0
	} else if vast.ChildAST(0).Src == sy.Name {
		varidx = 0
	}
	// vty is the container -- first el should be the type of element
	switch vty.Kind {
	case syms.Map: // need to know if we are the key or el
		if len(vty.Els) > 1 {
			tn := vty.Els[varidx].Type
			if IsQualifiedType(vty.Name) && !IsQualifiedType(tn) {
				pnm, _ := SplitType(vty.Name)
				sy.Type = QualifyType(pnm, tn)
			} else {
				sy.Type = tn
			}
		} else {
			sy.Type = TypeErr
			if TraceTypes {
				fmt.Printf("InferSymbolType: %s has ForRange over Map on type without an el type: %v\n", sy.Name, vty.Name)
			}
		}
	case syms.Array, syms.List:
		if varidx == 0 {
			sy.Type = "int"
		} else if len(vty.Els) > 0 {
			tn := vty.Els[0].Type
			if IsQualifiedType(vty.Name) && !IsQualifiedType(tn) {
				pnm, _ := SplitType(vty.Name)
				sy.Type = QualifyType(pnm, tn)
			} else {
				sy.Type = tn
			}
		} else {
			sy.Type = TypeErr
			if TraceTypes {
				fmt.Printf("InferSymbolType: %s has ForRange over Array, List on type without an el type: %v\n", sy.Name, vty.Name)
			}
		}
	case syms.String:
		if varidx == 0 {
			sy.Type = "int"
		} else {
			sy.Type = "rune"
		}
	default:
		sy.Type = TypeErr
		if TraceTypes {
			fmt.Printf("InferSymbolType: %s has ForRange over non-container type: %v kind: %v\n", sy.Name, vty.Name, vty.Kind)
		}
	}
}

// InferEmptySymbolType ensures that any empty symbol type is resolved during
// processing of other type information -- returns true if was able to resolve
func (gl *GoLang) InferEmptySymbolType(sym *syms.Symbol, fs *parse.FileState, pkg *syms.Symbol) bool {
	if sym.Type == "" { // hasn't happened yet
		// if TraceTypes {
		// 	fmt.Printf("TExpr: trying to infer type\n")
		// }
		sym.Type = TypeInProcess
		gl.InferSymbolType(sym, fs, pkg, true)
	}
	if sym.Type == TypeInProcess {
		if TraceTypes {
			fmt.Printf("TExpr: source symbol is in process -- we have a loop: %v  kind: %v\n", sym.Name, sym.Kind)
		}
		sym.Type = TypeErr
		return false
	}
	if sym.Type == TypeErr {
		if TraceTypes {
			fmt.Printf("TExpr: source symbol has type err: %v  kind: %v\n", sym.Name, sym.Kind)
		}
		return false
	}
	if sym.Type == "" { // shouldn't happen
		sym.Type = TypeErr
		if TraceTypes {
			fmt.Printf("TExpr: source symbol has type err (but wasn't marked): %v  kind: %v\n", sym.Name, sym.Kind)
		}
		return false
	}
	return true
}

func SymTypeNameForPkg(ty *syms.Type, pkg *syms.Symbol) string {
	sc, has := ty.Scopes[token.NamePackage]
	if has && sc != pkg.Name {
		return QualifyType(sc, ty.Name)
	}
	return ty.Name
}
