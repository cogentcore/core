// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/goki/ki/walki"
	"github.com/goki/pi/complete"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/parse"
	"github.com/goki/pi/pi"
	"github.com/goki/pi/syms"
	"github.com/goki/pi/token"
)

var LineParseState *pi.FileState
var FileParseState *pi.FileState
var CompleteSym *syms.Symbol
var CompleteSyms *syms.SymMap

var CompleteTrace = true

// todo:
// switch over to using type-based methods
// go up ast to find right point, and skip over the final element if it is a Name
// so you get the type up to the point of the last element, then just look in
// stuff on that.
// note: methods are not going to be found, so need to go back to the symbol once
// we have the type name to get the methods etc.

// CompleteLine is the main api called by completion code in giv/complete.go
func (gl *GoLang) CompleteLine(fs *pi.FileState, str string, pos lex.Pos) (md complete.MatchData) {
	if str == "" {
		return
	}
	fs.SymsMu.RLock()
	defer fs.SymsMu.RUnlock()

	pr := gl.Parser()
	if pr == nil {
		return
	}
	lfs := pr.ParseString(str, fs.Src.Filename, fs.Src.Sup)
	if lfs == nil {
		return
	}

	FileParseState = nil
	LineParseState = nil
	CompleteSym = nil
	CompleteSyms = nil

	// FileParseState = fs
	// LineParseState = lfs
	if CompleteTrace {
		lfs.Ast.WriteTree(os.Stdout, 0)
		lfs.LexState.Errs.Report(20, "", true, true)
		lfs.ParseState.Errs.Report(20, "", true, true)
	}

	var conts syms.SymMap // containers of given region -- local scoping
	fs.Syms.FindContainsRegion(pos, token.NameFunction, &conts)

	nms := gl.WalkUpExpr(lfs.ParseState.Ast)
	if CompleteTrace {
		fmt.Printf("nms: %v\n", nms)
	}

	if len(nms) == 0 {
		return
	}

	fnm := nms[0]
	scsym, got := fs.FindNameScoped(fnm, conts)
	if got {
		return gl.CompleteSym(fs, fs.Syms, scsym, nms[1:])
	}

	if CompleteTrace {
		fmt.Printf("name: %v not found\n", fnm)
		CompleteSyms = &conts
	}

	return
}

// CompleteSym completes to given symbol using following names also.
// psyms is the package syms currently in effect based on prior context.
func (gl *GoLang) CompleteSym(fs *pi.FileState, psyms syms.SymMap, sym *syms.Symbol, nms []string) (md complete.MatchData) {
	nnm := len(nms)
	switch {
	case sym.Type != "":
		return gl.CompleteTypeName(fs, psyms, sym.Type, sym, nms)
	case sym.Kind == token.NameMethod:
		ps := gl.FuncParams(sym)
		if len(ps) > 0 {
			pstr := strings.Join(ps, ", ")
			c := complete.Completion{Text: pstr, Label: pstr, Icon: sym.Kind.IconName(), Desc: pstr}
			md.Matches = append(md.Matches, c)
			md.Seed = ""
		}
		return
	case sym.Kind == token.NamePackage:
		switch nnm {
		case 0:
			complete.AddSyms(sym.Children, "", &md)
			return
		case 1:
			complete.AddSymsPrefix(sym.Children, "", nms[0], &md)
			return
		default:
			ssym, got := fs.FindNameScoped(nms[0], sym.Children)
			if got {
				return gl.CompleteSym(fs, sym.Children, ssym, nms[1:])
			}
		}
		// default: // last restort: try looking up a package name indirectly
		// 	psym, has := gl.PkgSyms(fs, psyms, pnm)
		// if !has {
		// 	return
		// }

	}
	if CompleteTrace {
		CompleteSym = sym
		fmt.Printf("sym: %+v  nms: %v\n", sym, nms)
	}
	return
}

// CompleteTypeName completes to given type name using following names from that type.
// psyms is the package syms currently in effect based on prior context.
func (gl *GoLang) CompleteTypeName(fs *pi.FileState, psyms syms.SymMap, typ string, sym *syms.Symbol, nms []string) (md complete.MatchData) {
	if typ[0] == '*' {
		typ = typ[1:]
	}
	tsp := strings.Split(typ, ".")
	pnm := ""
	tnm := ""
	if len(tsp) == 2 {
		pnm = tsp[0]
		tnm = tsp[1]
	} else {
		tnm = tsp[0]
	}
	var tsym *syms.Symbol
	var got bool
	if pnm != "" {
		psym, has := gl.PkgSyms(fs, psyms, pnm)
		if !has {
			CompleteSyms = &psyms
			return
		}
		psyms = psym.Children // update...
	}
	tsym, got = psyms.FindNameScoped(tnm)
	if !got {
		// ok, maybe it was not a type after all
		if CompleteTrace {
			fmt.Printf("type name not found: %v\n", tnm)
			CompleteSym = sym
		}
		return
	}
	if CompleteTrace {
		fmt.Printf("type sym: %v\n", tsym)
		// CompleteSym = tsym
	}

	nnm := len(nms)
	switch nnm {
	case 0:
		complete.AddSyms(tsym.Children, "", &md)
	case 1:
		complete.AddSymsPrefix(tsym.Children, "", nms[0], &md)
	default:
		cnm := nms[0]
		csym, got := tsym.Children.FindNameScoped(cnm)
		if got {
			return gl.CompleteSym(fs, psyms, csym, nms[1:])
		}
	}
	return
}

// CompleteEdit returns the completion edit data for integrating the selected completion
// into the source
func (gl *GoLang) CompleteEdit(fs *pi.FileState, text string, cp int, comp complete.Completion, seed string) (ed complete.EditData) {
	// if the original is ChildByName() and the cursor is between d and B and the comp is Children,
	// then delete the portion after "Child" and return the new comp and the number or runes past
	// the cursor to delete
	s2 := text[cp:]
	if len(s2) > 0 {
		r := rune(s2[0])
		// find the next whitespace or end of text
		if !(unicode.IsSpace(r)) {
			count := len(s2)
			for i, c := range s2 {
				r = rune(c)
				if unicode.IsSpace(r) || r == rune('(') || r == rune('.') || r == rune('[') || r == rune('&') || r == rune('*') {
					s2 = s2[0:i]
					break
				}
				// might be last word
				if i == count-1 {
					break
				}
			}
		}
	}

	var new = comp.Text
	// todo: only do if parens not already there
	//class, ok := comp.Extra["class"]
	//if ok && class == "func" {
	//	new = new + "()"
	//}
	ed.NewText = new
	ed.ForwardDelete = len(s2)
	return ed
}

// WalkUpExpr walks up the AST expression and returns a list of strings that are selectors
// in the expression.
func (gl *GoLang) WalkUpExpr(ast *parse.Ast) []string {
	var nms []string
	curi := walki.Last(ast)
	if curi == nil {
		return nms
	}
	cur := curi.(*parse.Ast)
	for {
		var par *parse.Ast
		if cur.Par != nil {
			par = cur.Par.(*parse.Ast)
		}
		switch {
		case cur.Nm == "Name":
			if par != nil && (par.Nm[:4] == "Asgn" || strings.HasSuffix(par.Nm, "Expr")) {
				break
			}
			nms = append(nms, cur.Src)
		case cur.Nm == "Selector":
			if cur.NumChildren() == 1 {
				nms = append(nms[:len(nms)-1], "", nms[len(nms)-1]) // insert blank
			}
		case cur.Nm == "ExprStmt":
			if len(nms) > 0 {
				break
			}
			switch cur.Src {
			case "(": // pass through so completion processes as a function
				nms = append(nms, cur.Src)
			case "[":
				nms = nil // nothing to do here
				break
			}
			// case cur.Nm == "Slice":
			// nop
		}
		curi = walki.Prev(cur.This())
		if curi == nil {
			break
		}
		cur = curi.(*parse.Ast)
	}
	if nms == nil {
		return nms
	}
	nnm := len(nms)
	rnm := make([]string, nnm)
	for i, nm := range nms {
		rnm[nnm-i-1] = nm
	}
	return rnm
}

/*
	if scope != "" {
		md.Seed = scope + "." + name
	} else {
		md.Seed = name
	}
	var matches syms.SymMap
	if scope != "" {
		scsym, got := psym.Children.FindNameScoped(scope)
		if got {
			gotKids := scsym.FindAnyChildren(name, psym.Children, nil, &matches)
			if !gotKids {
				scope = ""
				md.Seed = name
			}
		} else {
			scope = ""
			md.Seed = name
		}
	}
	if len(matches) == 0 { // look just at name if nothing from scope
		nmsym, got := psym.Children.FindNameScoped(name)
		if got {
			nmsym.FindAnyChildren("", psym.Children, nil, &matches)
		}
		if len(matches) == 0 {
			psym.Children.FindNamePrefixScoped(name, &matches)
		}
	}
	md.Seed = pkg + "." + md.Seed
	effscp := pkg
	if scope != "" {
		effscp += "." + scope
	}
	gl.CompleteReturnMatches(matches, effscp, &md)
*/
