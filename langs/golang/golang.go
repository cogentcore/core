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

func (gl *GoLang) ParseDir(path string, opts pi.LangDirOpts) *syms.Symbol {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		path, err = dirs.GoSrcDir(path)
		if err != nil {
			// log.Println(err)
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
		if strings.Contains(fnm, "/image/font/gofont/") { // hack to prevent parsing those..
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

// ImportPathPkg returns the package (last dir) and base of import path
// from import path string -- removes any quotes around path first.
func (gl *GoLang) ImportPathPkg(im string) (path, base, pkg string) {
	sz := len(im)
	if sz == 0 {
		return
	}
	path = im
	if im[0] == '"' {
		path = im[1 : sz-1]
	}
	base, pkg = filepath.Split(path)
	return
}

// AddImportToExts adds given import into pi.FileState.ExtSyms list
// assumed to be called as a separate goroutine
func (gl *GoLang) AddImportToExts(fs *pi.FileState, im string) {
	im, _, pkg := gl.ImportPathPkg(im)
	psym := gl.ParseDir(im, pi.LangDirOpts{})
	if psym != nil {
		psym.Name = pkg
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

// FindImportPkg attempts to find an import package based on symbols in
// an existing package.  For indirect loading of packages from other packages
// that we don't direct import.
func (gl *GoLang) FindImportPkg(fs *pi.FileState, psyms syms.SymMap, nm string) (*syms.Symbol, bool) {
	for _, sy := range psyms {
		if sy.Kind != token.NameLibrary {
			continue
		}
		_, _, pkg := gl.ImportPathPkg(sy.Name)
		if pkg == nm {
			return sy, true
		}
	}
	return nil, false
}
