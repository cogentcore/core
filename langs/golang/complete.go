// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"fmt"
	"os"
	"path/filepath"
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

// * completion on new code != saved code -- really needs to do full reparse more frequently
// * val = strings. doesn't work -- needs first letter..
// * second or later vars in multiple assign is not implemented
// * edit needs to be fixed to properly insert completions and retain remaining parts etc

var CompleteTrace = false

// Lookup is the main api called by completion code in giv/complete.go to lookup item
func (gl *GoLang) Lookup(fss *pi.FileStates, str string, pos lex.Pos) (ld complete.Lookup) {
	if str == "" {
		return
	}
	flds := strings.Fields(str)
	// origStr := str
	str = flds[len(flds)-1] // just use last one

	fs := fss.Done()

	fs.SymsMu.RLock()
	defer fs.SymsMu.RUnlock()

	pr := gl.Parser()
	if pr == nil {
		return
	}
	fpath, _ := filepath.Abs(fs.Src.Filename)
	lfs := pr.ParseString(str, fpath, fs.Src.Sup)
	if lfs == nil {
		return
	}

	if CompleteTrace {
		lfs.ParseState.Ast.WriteTree(os.Stdout, 0)
		lfs.LexState.Errs.Report(20, "", true, true)
		lfs.ParseState.Errs.Report(20, "", true, true)
	}

	var scopes syms.SymMap // scope(s) for position, fname
	scope := gl.CompletePosScope(fs, pos, fpath, &scopes)

	start, last := gl.CompleteAstStart(lfs.ParseState.Ast, scope)
	if CompleteTrace {
		if start == nil {
			fmt.Printf("start = nil\n")
			return
		}
		fmt.Printf("\n####################\ncompletion start in scope: %v\n", scope)
		lfs.ParseState.Ast.WriteTree(os.Stdout, 0)
		fmt.Printf("Start tree:\n")
		start.WriteTree(os.Stdout, 0)
	}

	pkg := fs.ParseState.Scopes[0]
	start.SrcReg.St = pos

	if start == last { // single-item
		seed := start.Src
		if seed != "" {
			return gl.LookupString(fs, pkg, seed)
		}
		return gl.LookupString(fs, pkg, str)
	}

	typ, nxt, got := gl.TypeFromAstExpr(fs, pkg, pkg, start)
	lststr := ""
	if nxt != nil {
		lststr = nxt.Src
	}
	if got {
		if lststr != "" {
			for _, mt := range typ.Meths {
				nm := mt.Name
				if !strings.HasPrefix(nm, lststr) {
					continue
				}
				if mt.Filename != "" {
					ld.SetFile(mt.Filename, mt.Region.St.Ln, mt.Region.Ed.Ln)
					return
				}
			}
		}
		// fmt.Printf("got completion type: %v, last str: %v\n", typ.String(), lststr)
		ld.SetFile(typ.Filename, typ.Region.St.Ln, typ.Region.Ed.Ln)
		return
	}
	// see if it starts with a package name..
	snxt := start.NextAst()
	lststr = last.Src
	if snxt != nil && snxt.Src != "" {
		ststr := snxt.Src
		if lststr != "" && lststr != ststr {
			ld = gl.LookupString(fs, pkg, ststr+"."+lststr)
		} else {
			ld = gl.LookupString(fs, pkg, ststr)
		}
	} else {
		ld = gl.LookupString(fs, pkg, lststr)
	}
	if ld.Filename == "" { // didn't work
		ld = gl.LookupString(fs, pkg, str)
	}
	return
}

// CompleteLine is the main api called by completion code in giv/complete.go
func (gl *GoLang) CompleteLine(fss *pi.FileStates, str string, pos lex.Pos) (md complete.Matches) {
	if str == "" {
		return
	}
	flds := strings.Fields(str)
	origStr := str
	str = flds[len(flds)-1] // just use last one

	fs := fss.Done()

	fs.SymsMu.RLock()
	defer fs.SymsMu.RUnlock()

	pr := gl.Parser()
	if pr == nil {
		return
	}
	fpath, _ := filepath.Abs(fs.Src.Filename)

	if CompleteTrace {
		fmt.Printf("complete str:  %v  orig: %v\n", str, origStr)
	}
	lfs := pr.ParseString(str, fpath, fs.Src.Sup)
	if lfs == nil {
		return
	}

	if CompleteTrace {
		lfs.ParseState.Ast.WriteTree(os.Stdout, 0)
		lfs.LexState.Errs.Report(20, "", true, true)
		lfs.ParseState.Errs.Report(20, "", true, true)
	}

	var scopes syms.SymMap // scope(s) for position, fname
	scope := gl.CompletePosScope(fs, pos, fpath, &scopes)

	start, last := gl.CompleteAstStart(lfs.ParseState.Ast, scope)
	if CompleteTrace {
		if start == nil {
			fmt.Printf("start = nil\n")
			return
		}
		fmt.Printf("\n####################\ncompletion start in scope: %v\n", scope)
		lfs.ParseState.Ast.WriteTree(os.Stdout, 0)
		fmt.Printf("Start tree:\n")
		start.WriteTree(os.Stdout, 0)
	}

	pkg := fs.ParseState.Scopes[0]
	start.SrcReg.St = pos

	if start == last { // single-item
		seed := start.Src
		if CompleteTrace {
			fmt.Printf("start == last: %v\n", seed)
		}
		md.Seed = seed
		if start.Nm == "TypeNm" {
			gl.CompleteTypeName(fs, pkg, seed, &md)
			return
		}
		if len(scopes) > 0 {
			complete.AddSymsPrefix(scopes, "", seed, &md)
		}
		gl.CompletePkgSyms(fs, pkg, seed, &md)
		gl.CompleteBuiltins(fs, seed, &md)
		return
	}

	typ, nxt, got := gl.TypeFromAstExpr(fs, pkg, pkg, start)
	lststr := ""
	if nxt != nil {
		lststr = nxt.Src
	}
	if got {
		// fmt.Printf("got completion type: %v, last str: %v\n", typ.String(), lststr)
		complete.AddTypeNames(typ, typ.Name, lststr, &md)
	} else {
		// see if it starts with a package name..
		// todo: move this to a function as in lookup
		snxt := start.NextAst()
		if snxt != nil && snxt.Src != "" {
			ststr := snxt.Src
			psym, has := gl.PkgSyms(fs, pkg.Children, ststr)
			if has {
				lststr := last.Src
				if lststr != "" && lststr != ststr {
					var matches syms.SymMap
					psym.Children.FindNamePrefixScoped(lststr, &matches)
					complete.AddSyms(matches, ststr, &md)
					md.Seed = lststr
				} else {
					complete.AddSyms(psym.Children, ststr, &md)
				}
				return
			}
		}
		if CompleteTrace {
			fmt.Printf("completion type not found\n")
		}
	}

	return
}

// CompletePosScope returns the scope for given position in given filename,
// and fills in the scoping symbol(s) in scMap
func (gl *GoLang) CompletePosScope(fs *pi.FileState, pos lex.Pos, fpath string, scopes *syms.SymMap) token.Tokens {
	fs.Syms.FindContainsRegion(fpath, pos, 2, token.None, scopes) // None matches any, 2 extra lines to add for new typing
	if len(*scopes) == 0 {
		return token.None
	}
	if len(*scopes) == 1 {
		for _, sy := range *scopes {
			return sy.Kind
		}
	}
	var last *syms.Symbol
	for _, sy := range *scopes {
		if sy.Kind.SubCat() == token.NameFunction {
			return sy.Kind
		}
		last = sy
	}
	if CompleteTrace {
		fmt.Printf(" > 1 scopes!\n")
		scopes.WriteDoc(os.Stdout, 0)
	}
	return last.Kind
}

// CompletePkgSyms matches all package symbols using seed
func (gl *GoLang) CompletePkgSyms(fs *pi.FileState, pkg *syms.Symbol, seed string, md *complete.Matches) {
	md.Seed = seed
	var matches syms.SymMap
	pkg.Children.FindNamePrefixScoped(seed, &matches)
	complete.AddSyms(matches, "", md)
}

// CompleteTypeName matches builtin and package type names to seed
func (gl *GoLang) CompleteTypeName(fs *pi.FileState, pkg *syms.Symbol, seed string, md *complete.Matches) {
	md.Seed = seed
	for _, tk := range BuiltinTypeKind {
		if strings.HasPrefix(tk.Name, seed) {
			c := complete.Completion{Text: tk.Name, Label: tk.Name, Icon: "type"}
			md.Matches = append(md.Matches, c)
		}
	}
	sfunc := strings.HasPrefix(seed, "func ")
	for _, tk := range pkg.Types {
		if !sfunc && strings.HasPrefix(tk.Name, "func ") {
			continue
		}
		if strings.HasPrefix(tk.Name, seed) {
			c := complete.Completion{Text: tk.Name, Label: tk.Name, Icon: "type"}
			md.Matches = append(md.Matches, c)
		}
	}
}

// LookupString attempts to lookup a string, which could be a type name,
// (with package qualifier), could be partial, etc
func (gl *GoLang) LookupString(fs *pi.FileState, pkg *syms.Symbol, str string) (ld complete.Lookup) {
	str = TrimLeftToAlpha(str)
	pnm, tnm := SplitType(str)
	if pnm != "" && tnm != "" {
		psym, has := gl.PkgSyms(fs, pkg.Children, pnm)
		if has {
			tnm = TrimLeftToAlpha(tnm)
			var matches syms.SymMap
			psym.Children.FindNamePrefixScoped(tnm, &matches)
			if len(matches) == 1 {
				var psy *syms.Symbol
				for _, sy := range matches {
					psy = sy
				}
				ld.SetFile(psy.Filename, psy.Region.St.Ln, psy.Region.Ed.Ln)
				return
			}
		}
		if CompleteTrace {
			fmt.Printf("Lookup: package-qualified string not found: %v\n", str)
		}
		return
	}
	// try types to str:
	var tym *syms.Type
	nmatch := 0
	for _, tk := range pkg.Types {
		if strings.HasPrefix(tk.Name, str) {
			tym = tk
			nmatch++
		}
	}
	if nmatch == 1 {
		ld.SetFile(tym.Filename, tym.Region.St.Ln, tym.Region.Ed.Ln)
		return
	}
	// then try any symbol
	var matches syms.SymMap
	pkg.Children.FindNamePrefixScoped(str, &matches)
	if len(matches) == 1 {
		var psy *syms.Symbol
		for _, sy := range matches {
			psy = sy
		}
		ld.SetFile(psy.Filename, psy.Region.St.Ln, psy.Region.Ed.Ln)
		return
	}
	if CompleteTrace {
		fmt.Printf("Lookup: string not found: %v\n", str)
	}
	return
}

// CompleteAstStart finds the best starting point in the given current-line Ast
// to start completion process, which walks back down from that starting point
func (gl *GoLang) CompleteAstStart(ast *parse.Ast, scope token.Tokens) (start, last *parse.Ast) {
	curi := walki.Last(ast)
	if curi == nil {
		return
	}
	cur := curi.(*parse.Ast)
	last = cur
	start = cur
	prv := cur
	for {
		var par *parse.Ast
		if cur.Par != nil {
			par = cur.Par.(*parse.Ast)
		}
		switch {
		case cur.Nm == "TypeNm":
			return cur, last
		case cur.Nm == "File":
			if prv != last && prv.Src == last.Src {
				return last, last // triggers single-item completion
			}
			return prv, last
		case cur.Nm == "Selector":
			if par != nil {
				if par.Nm[:4] == "Asgn" {
					return cur, last
				}
				if strings.HasSuffix(par.Nm, "Expr") {
					return cur, last
				}
			} else {
				flds := strings.Fields(cur.Src)
				cur.Src = flds[len(flds)-1] // skip any spaces
				return cur, last
			}
		case cur.Nm == "Name":
			if cur.Src == "if" { // weird parsing if incomplete
				if prv != last && prv.Src == last.Src {
					return last, last // triggers single-item completion
				}
				return prv, last
			}
			if par != nil {
				if par.Nm[:4] == "Asgn" {
					return prv, last
				}
				if strings.HasSuffix(par.Nm, "Expr") {
					return cur, last
				}
			}
		case cur.Nm == "ExprStmt":
			if scope == token.None {
				return prv, last
			}
			if cur.Src != "(" && cur.Src == prv.Src {
				return prv, last
			}
			if cur.Src != "(" && prv != last {
				return prv, last
			}
		case strings.HasSuffix(cur.Nm, "Stmt"):
			return prv, last
		}
		nxt := cur.PrevAst()
		if nxt == nil {
			return cur, last
		}
		prv = cur
		cur = nxt
	}
	return cur, last
}

// CompleteEdit returns the completion edit data for integrating the selected completion
// into the source
func (gl *GoLang) CompleteEdit(fss *pi.FileStates, text string, cp int, comp complete.Completion, seed string) (ed complete.Edit) {
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

	var nw = comp.Text
	// todo: only do if parens not already there
	//class, ok := comp.Extra["class"]
	//if ok && class == "func" {
	//	new = new + "()"
	//}
	ed.NewText = nw
	ed.ForwardDelete = len(s2)
	return ed
}
