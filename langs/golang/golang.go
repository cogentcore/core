// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/goki/ki/dirs"
	"github.com/goki/pi/complete"
	"github.com/goki/pi/filecat"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/pi"
	"github.com/goki/pi/syms"
	"github.com/goki/pi/token"
)

// GoLang implements the Lang interface for the Go language
type GoLang struct {
	Pr *pi.Parser
}

// TheGoLang is the instance variable providing support for the Go language
var TheGoLang = GoLang{}

func init() {
	pi.StdLangProps[filecat.Go].Lang = &TheGoLang
}

func (gl *GoLang) Parser() *pi.Parser {
	if gl.Pr != nil {
		return gl.Pr
	}
	lp, _ := pi.LangSupport.Props(filecat.Go)
	if lp.Parser == nil {
		pi.LangSupport.OpenStd()
	}
	gl.Pr = lp.Parser
	if gl.Pr == nil {
		return nil
	}
	return gl.Pr
}

func (gl *GoLang) ParseFile(fs *pi.FileState) {
	pr := gl.Parser()
	if pr == nil {
		log.Println("ParseFile: no parser -- must call pi.LangSupport.OpenStd() at startup!")
		return
	}
	// lprf := prof.Start("LexAll")
	pr.LexAll(fs)
	// lprf.End()
	// pprf := prof.Start("ParseAll")
	pr.ParseAll(fs)
	// pprf.End()
	if len(fs.ParseState.Scopes) > 0 { // should be
		path, _ := filepath.Split(fs.Src.Filename)
		pkg := fs.ParseState.Scopes[0]
		fs.Syms[pkg.Name] = pkg // keep around..
		if len(fs.ExtSyms) == 0 {
			go gl.AddPathToExts(fs, path)
		}
		gl.AddImportsToExts(fs, pkg)
		gl.ResolveTypes(fs, pkg)
	}
}

func (gl *GoLang) LexLine(fs *pi.FileState, line int) lex.Line {
	pr := gl.Parser()
	if pr == nil {
		return nil
	}
	return pr.LexLine(fs, line)
}

func (gl *GoLang) ParseLine(fs *pi.FileState, line int) *pi.FileState {
	pr := gl.Parser()
	if pr == nil {
		return nil
	}
	lfs := pr.ParseLine(fs, line) // should highlight same line?
	return lfs
}

func (gl *GoLang) HiLine(fs *pi.FileState, line int) lex.Line {
	pr := gl.Parser()
	if pr == nil {
		return nil
	}
	ll := pr.LexLine(fs, line)
	lfs := pr.ParseLine(fs, line)
	if lfs != nil {
		ll = lfs.Src.Lexs[0]
		cml := fs.Src.Comments[line]
		merge := lex.MergeLines(ll, cml)
		mc := merge.Clone()
		if len(cml) > 0 {
			initDepth := fs.Src.PrevDepth(line)
			pr.PassTwo.NestDepthLine(mc, initDepth)
		}
		return mc
	} else {
		return ll
	}
}

// PathNamesInString returns the package, scope (type, fun) and
// element names in given string (line of code) at given position
func (gl *GoLang) PathNamesInString(fs *pi.FileState, str string, pos lex.Pos) (pkg, scope, name string, tok token.KeyToken) {
	pr := gl.Parser()
	if pr == nil {
		return
	}
	lfs := pr.ParseString(str, fs.Src.Filename, fs.Src.Sup)
	if lfs == nil {
		return
	}
	// pkg := fs.Syms.First()

	// lxstr := lfs.Src.LexTagSrc()
	// fmt.Println(lxstr)
	// lfs.Ast.WriteTree(os.Stdout, 0)

	lxs := lfs.Src.Lexs[0]
	sz := len(lxs)
	if lxs[sz-1].Tok.Tok == token.EOS {
		sz--
	}
	if sz <= 0 {
		return
	}
	posidx := -1
	posnm := ""
	for i := sz - 1; i >= 0; i-- {
		lx := lxs[i]
		if lx.St <= pos.Ch && pos.Ch < lx.Ed {
			tok = lx.Tok
			posidx = i
			if tok.Tok.Cat() == token.Name {
				posnm = string(lfs.Src.TokenSrc(lex.Pos{0, i}))
				break
			}
		}
	}
	if posidx < 0 {
		posidx = sz - 1
		for ; posidx >= 0; posidx-- {
			tok = lxs[posidx].Tok
			if tok.Tok.Cat() == token.Name {
				posnm = string(lfs.Src.TokenSrc(lex.Pos{0, posidx}))
				break
			}
		}
	}
	if posnm != "" {
		if _, has := gl.ExtsPkg(fs, posnm); has {
			pkg = posnm
			return // highest level of scoping -- done
		}
		if posidx < sz-1 && lxs[posidx+1].Tok.Tok.SubCat() == token.PunctSep {
			scope = posnm // next element is a sep -- we are not final name
		} else {
			name = posnm
		}
	}
	// now find earlier scopers
	for i := posidx - 1; i >= 0; i-- {
		lx := lxs[i]
		if lx.Tok.Tok.Cat() == token.Name {
			nm := string(lfs.Src.TokenSrc(lex.Pos{0, i}))
			if _, has := gl.ExtsPkg(fs, nm); has {
				pkg = nm
				return
			}
			if scope == "" && name == "" {
				name = nm
			} else if scope == "" && name != "" {
				scope = nm
			}
		} else if lx.Tok.Tok.SubCat() != token.PunctSep {
			break
		}
	}
	return
}

func (gl *GoLang) CompleteLine(fs *pi.FileState, str string, pos lex.Pos) (md complete.MatchData) {
	if str == "" {
		return
	}
	fs.SymsMu.RLock()
	defer fs.SymsMu.RUnlock()
	pkg, scope, name, _ := gl.PathNamesInString(fs, str, pos)
	if pkg == "" && name == "" && scope == "" {
		return
	}
	if pkg != "" {
		return gl.CompleteLinePkg(fs, str, pos, pkg, scope, name)
	}
	if scope != "" {
		md.Seed = scope + "." + name
	} else {
		md.Seed = name
	}

	var conts syms.SymMap // containers of given region -- local scoping
	fs.Syms.FindContainsRegion(pos, token.NameFunction, &conts)

	var matches syms.SymMap
	if scope != "" {
		scsym, got := fs.FindNameScoped(scope, conts)
		if got {
			gotKids := fs.FindChildren(scsym, name, conts, &matches)
			if !gotKids {
				scope = ""
				md.Seed = name
			}
		} else {
			scope = ""
			md.Seed = name
		}
	}

	if scope == "" && name == "" {
		return md
	}

	if len(matches) == 0 { // look just at name if nothing from scope
		nmsym, got := fs.FindNameScoped(name, conts)
		if got {
			fs.FindAnyChildren(nmsym, name, conts, &matches)
		}
		if len(matches) == 0 {
			fs.FindNamePrefixScoped(name, conts, &matches)
		}
	}
	gl.CompleteReturnMatches(matches, scope, &md)
	return
}

// CompleteReturnMatches returns the matched syms in given map as
// completion edit data
func (gl *GoLang) CompleteReturnMatches(matches syms.SymMap, scope string, md *complete.MatchData) {
	if len(matches) == 0 {
		return
	}
	sys := matches.Slice(true) // sorted
	for _, sy := range sys {
		if sy.Name[0] == '_' || sy.Kind == token.NameLibrary { // internal / import
			continue
		}
		nm := sy.Name
		lbl := sy.Name
		if scope != "" {
			lbl = nm + " (." + scope + ")"
			nm = scope + "." + nm
		}
		c := complete.Completion{Text: nm, Label: lbl, Icon: sy.Kind.IconName(), Desc: sy.Detail}
		md.Matches = append(md.Matches, c)
	}
	return
}

// CompleteLinePkg returns completion for symbol in given external
// package
func (gl *GoLang) CompleteLinePkg(fs *pi.FileState, str string, pos lex.Pos, pkg, scope, name string) (md complete.MatchData) {
	fs.SymsMu.RLock()
	defer fs.SymsMu.RUnlock()
	psym, _ := gl.ExtsPkg(fs, pkg)

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

func (gl *GoLang) ParseDir(path string, opts pi.LangDirOpts) *syms.Symbol {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		path, err = dirs.GoSrcDir(path)
		if err != nil {
			log.Println(err)
			return nil
		}
	} else if err != nil {
		log.Println(err.Error())
		return nil
	}
	path, _ = filepath.Abs(path)
	// fmt.Printf("Parsing / loading path: %v\n", path)

	fls := dirs.ExtFileNames(path, []string{".go"})
	if len(fls) == 0 {
		return nil
	}

	if !opts.Rebuild {
		csy, cts, err := syms.OpenSymCache(filecat.Go, path)
		if err == nil && csy != nil {
			if !gl.Pr.ModTime.IsZero() && cts.Before(gl.Pr.ModTime) {
				// fmt.Printf("rebuilding %v because parser: %v is newer than cache: %v\n", path, gl.Pr.ModTime, cts)
			} else {
				lstmod := dirs.LatestMod(path, []string{".go"})
				if lstmod.Before(cts) {
					// fmt.Printf("loaded cache for: %v from: %v\n", path, cts)
					return csy
				}
			}
		}
	}

	pr := gl.Parser()
	var pkgsym *syms.Symbol
	var fss []*pi.FileState // file states for each file
	for i := range fls {
		fnm := fls[i]
		if strings.HasSuffix(fnm, "_test.go") {
			continue
		}
		fs := pi.NewFileState() // we use a separate fs for each file, so we have full ast
		fss = append(fss, fs)
		// optional monitoring of parsing
		// fs.ParseState.Trace.On = true
		// fs.ParseState.Trace.Match = true
		// fs.ParseState.Trace.NoMatch = true
		// fs.ParseState.Trace.Run = true
		// fs.ParseState.Trace.RunAct = true
		// fs.ParseState.Trace.StdOut()
		fpath := filepath.Join(path, fnm)
		err = fs.Src.OpenFile(fpath)
		if err != nil {
			continue
		}
		// fmt.Printf("parsing file: %v\n", fnm)
		// stt := time.Now()
		pr.LexAll(fs)
		// lxdur := time.Now().Sub(stt)
		pr.ParseAll(fs)
		// prdur := time.Now().Sub(stt)
		// fmt.Printf("\tlex: %v full parse: %v\n", lxdur, prdur-lxdur)
		if len(fs.ParseState.Scopes) > 0 { // should be
			pkg := fs.ParseState.Scopes[0]
			if pkg.Name == "main" { // todo: not sure about skipping this..
				continue
			}
			gl.DeleteUnexported(pkg)
			if pkgsym == nil {
				pkgsym = pkg
			} else {
				pkgsym.Children.CopyFrom(pkg.Children)
				pkgsym.Types.CopyFrom(pkg.Types)
			}
			// } else {
			// 	fmt.Printf("\tno parse state scopes!\n")
		}
	}
	if pkgsym == nil {
		return nil
	}
	pfs := pi.NewFileState() // master overall package file state
	gl.ResolveTypes(pfs, pkgsym)
	if !opts.Nocache {
		syms.SaveSymCache(pkgsym, filecat.Go, path)
	}
	return pkgsym
}

/////////////////////////////////////////////////////////////////////////////
// Go util funcs

// DeleteUnexported deletes lower-case unexported items from map, and
// children of symbols on map
func (gl *GoLang) DeleteUnexported(sy *syms.Symbol) {
	if sy.Kind.SubCat() != token.NameScope { // only for top-level scopes
		return
	}
	for nm, ss := range sy.Children {
		if ss == sy {
			fmt.Printf("warning: child is self!: %v\n", sy.String())
			continue
		}
		if ss.Kind.SubCat() != token.NameScope { // typically lowercase
			rn, _ := utf8.DecodeRuneInString(nm)
			if nm == "" || unicode.IsLower(rn) {
				delete(sy.Children, nm)
			}
		}
		if ss.HasChildren() {
			gl.DeleteUnexported(ss)
		}
	}
}

// AddPkgToSyms adds given package symbol, with children from package
// to pi.FileState.Syms map -- merges with anything already there
// does NOT add imports -- that is an optional second step.
// Returns true if there was an existing entry for this package.
func (gl *GoLang) AddPkgToSyms(fs *pi.FileState, pkg *syms.Symbol) bool {
	fs.SymsMu.Lock()
	psy, has := fs.Syms[pkg.Name]
	if has {
		psy.Children.CopyFrom(pkg.Children)
		pkg = psy
	} else {
		fs.Syms[pkg.Name] = pkg
	}
	fs.SymsMu.Unlock()
	return has
}

// AddPkgToExts adds given package symbol, with children from package
// to pi.FileState.ExtSyms map -- merges with anything already there
// does NOT add imports -- that is an optional second step.
// Returns true if there was an existing entry for this package.
func (gl *GoLang) AddPkgToExts(fs *pi.FileState, pkg *syms.Symbol) bool {
	fs.SymsMu.Lock()
	psy, has := fs.ExtSyms[pkg.Name]
	if has {
		psy.Children.CopyFrom(pkg.Children)
		pkg = psy
	} else {
		if fs.ExtSyms == nil {
			fs.ExtSyms = make(syms.SymMap)
		}
		fs.ExtSyms[pkg.Name] = pkg
	}
	fs.SymsMu.Unlock()
	return has
}

// ExtsPkg returns the symbol in fs.ExtSyms for given name that
// is a package -- false if not found -- assumed to be called under
// SymsMu lock
func (gl *GoLang) ExtsPkg(fs *pi.FileState, nm string) (*syms.Symbol, bool) {
	sy, has := fs.ExtSyms[nm]
	return sy, has
}

// AddImportsToExts adds imports from given package into pi.FileState.ExtSyms list
// imports are coded as NameLibrary symbols with names = import path
func (gl *GoLang) AddImportsToExts(fs *pi.FileState, pkg *syms.Symbol) {
	fs.SymsMu.RLock()
	var imps syms.SymMap
	pkg.Children.FindKindScoped(token.NameLibrary, &imps)
	fs.SymsMu.RUnlock()
	if len(imps) == 0 {
		return
	}
	for _, im := range imps {
		if im.Name == "C" {
			continue
		}
		go gl.AddImportToExts(fs, im.Name)
	}
}

// AddImportToExts adds given import into pi.FileState.ExtSyms list
// assumed to be called as a separate goroutine
func (gl *GoLang) AddImportToExts(fs *pi.FileState, im string) {
	sz := len(im)
	if sz == 0 {
		return
	}
	pnm := ""
	if im[0] == '"' {
		im = im[1 : sz-1]
		_, pnm = filepath.Split(im)
	} else {
		isp := strings.Index(im, " ")
		return // malformed import but we don't care here
		pnm = im[:isp]
		im = im[isp+2 : sz-1] // assuming quotes around rest..
	}
	psym := gl.ParseDir(im, pi.LangDirOpts{})
	if psym != nil {
		psym.Name = pnm
		gl.AddPkgToExts(fs, psym)
	}
}

// AddPathToSyms adds given path into pi.FileState.Syms list
// assumed to be called as a separate goroutine
func (gl *GoLang) AddPathToSyms(fs *pi.FileState, path string) {
	psym := gl.ParseDir(path, pi.LangDirOpts{})
	if psym != nil {
		gl.AddPkgToSyms(fs, psym)
	}
}

// AddPathToExts adds given path into pi.FileState.ExtSyms list
// assumed to be called as a separate goroutine
func (gl *GoLang) AddPathToExts(fs *pi.FileState, path string) {
	psym := gl.ParseDir(path, pi.LangDirOpts{})
	if psym != nil {
		gl.AddPkgToExts(fs, psym)
	}
}
