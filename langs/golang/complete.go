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

// todo: fix completer logic when seed == only item -- should still show it!
// * also need to fix NameVarGlobal in typeinfer
// * transitive nxx1 or fffb stuff not getting pulled in from leabra (pools)
// * var := expr not stopping at right spot for completion -- easy
// * other files in *same package* not getting included in fs. syms -- need to add

var LineParseState *pi.FileState
var FileParseState *pi.FileState
var CompleteSym *syms.Symbol
var CompleteSyms *syms.SymMap

var CompleteTrace = false

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
		lfs.ParseState.Ast.WriteTree(os.Stdout, 0)
		lfs.LexState.Errs.Report(20, "", true, true)
		lfs.ParseState.Errs.Report(20, "", true, true)
	}

	start, last := gl.CompleteAstStart(lfs.ParseState.Ast)
	if CompleteTrace {
		if start == nil {
			fmt.Printf("start = nil\n")
			return
		}
		fmt.Printf("completion start:\n")
		start.WriteTree(os.Stdout, 0)
	}

	pkg := fs.ParseState.Scopes[0]
	// CompleteSym = pkg
	start.SrcReg.St = pos

	if start == last {
		str := start.Src
		if CompleteTrace {
			fmt.Printf("start == last: %v\n", str)
		}

		var conts syms.SymMap // containers of given region -- local scoping
		fs.Syms.FindContainsRegion(pos, token.NameFunction, &conts)
		complete.AddSymsPrefix(conts, "", str, &md)
		var matches syms.SymMap
		pkg.Children.FindNamePrefixScoped(str, &matches)
		complete.AddSyms(matches, "", &md)
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

// CompleteAstStart finds the best starting point in the given current-line Ast
// to start completion process, which walks back down from that starting point
func (gl *GoLang) CompleteAstStart(ast *parse.Ast) (start, last *parse.Ast) {
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
		case cur.Nm == "Name":
			if par != nil && (par.Nm[:4] == "Asgn" || strings.HasSuffix(par.Nm, "Expr")) {
				return cur, last
			}
		case cur.Nm == "ExprStmt":
			if cur.Src != "(" {
				return prv, last
			}
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
