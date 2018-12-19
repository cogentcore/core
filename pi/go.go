// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pi

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/goki/gi/complete"
	"github.com/goki/gi/filecat"
	"github.com/goki/ki/dirs"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/syms"
	"github.com/goki/pi/token"
)

// GoLang implements the Lang interface for the Go language
type GoLang struct {
	Pr *Parser
}

// TheGoLang is the instance variable providing support for the Go language
var TheGoLang = GoLang{}

func (gl *GoLang) Parser() *Parser {
	if gl.Pr != nil {
		return gl.Pr
	}
	lp, _ := LangSupport.Props(filecat.Go)
	if lp.Parser == nil {
		LangSupport.OpenStd()
	}
	gl.Pr = lp.Parser
	if gl.Pr == nil {
		return nil
	}
	gl.Pr.InitAll()
	return gl.Pr
}

func (gl *GoLang) ParseFile(fs *FileState) {
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
		// gl.DeleteUnexported(pkg.Children) // for local access we keep unexported!
		if !gl.AddPkgToSyms(fs, pkg) { // first time, no existing
			go gl.AddPathToSyms(fs, path)
		}
		gl.AddImportsToSyms(fs, pkg)
	}
}

func (gl *GoLang) LexLine(fs *FileState, line int) lex.Line {
	pr := gl.Parser()
	if pr == nil {
		return nil
	}
	ll := pr.LexLine(fs, line)
	lfs := pr.ParseLine(fs, line)
	if lfs != nil {
		return lfs.Src.Lexs[0]
	} else {
		return ll
	}
}

func (gl *GoLang) ParseLine(fs *FileState, line int) *FileState {
	pr := gl.Parser()
	if pr == nil {
		return nil
	}
	lfs := pr.ParseLine(fs, line) // should highlight same line?
	return lfs
}

func (gl *GoLang) CompleteLine(fs *FileState, str string, pos lex.Pos) (md complete.MatchData) {
	if str == "" {
		return
	}
	pr := gl.Parser()
	if pr == nil {
		return
	}
	lfs := pr.ParseString(str, fs.Src.Filename, fs.Src.Sup)
	if lfs == nil {
		return
	}

	// lxstr := lfs.Src.LexTagSrc()
	// fmt.Println(lxstr)
	// lfs.Ast.WriteTree(os.Stdout, 0)

	// first pass: just use lexical tokens even though we have the full Ast..
	lxs := lfs.Src.Lexs[0]
	sz := len(lxs)
	// look for scope.name
	name := ""
	scope := ""
	if lxs[sz-1].Tok.Tok == token.EOS {
		sz--
	}
	gotSep := false
	for i := sz - 1; i >= 0; i-- {
		lx := lxs[i]
		if lx.Tok.Tok.Cat() == token.Name {
			nm := string(lfs.Src.TokenSrc(lex.Pos{0, i}))
			if gotSep {
				scope = nm
				break
			} else {
				name = nm
			}
		} else if lx.Tok.Tok.SubCat() == token.PunctSep {
			gotSep = true
		} else {
			break
		}
	}
	if name == "" && scope == "" {
		return
	}

	if scope != "" {
		md.Seed = scope + "." + name
	} else {
		md.Seed = name
	}
	// fmt.Printf("seed: %v\n", md.Seed)

	fs.SymsMu.RLock() // syms access needs to be locked -- could be updated..
	var matches syms.SymMap
	if scope != "" {
		scsym, got := fs.Syms.FindNameScoped(scope)
		if got {
			if name == "" {
				matches = scsym.Children
			} else {
				scsym.Children.FindNamePrefix(name, &matches)
			}
		} else {
			scope = ""
			md.Seed = name
		}
	}
	if len(matches) == 0 {
		fs.Syms.FindNamePrefix(name, &matches)
	}
	fs.SymsMu.RUnlock()
	if len(matches) == 0 {
		return
	}

	sys := matches.Slice(true) // sorted
	for _, sy := range sys {
		if sy.Name[0] == '_' || sy.Kind == token.NameLibrary { // internal / import
			continue
		}
		nm := sy.Name
		if scope != "" {
			nm = scope + "." + nm
		}
		c := complete.Completion{Text: nm, Icon: sy.Kind.IconName(), Desc: sy.Detail}
		md.Matches = append(md.Matches, c)
	}
	return
}

func (gl *GoLang) ParseDir(path string, opts LangDirOpts) *syms.Symbol {
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
		csy, cts, err := syms.OpenSymCache(path)
		if err == nil && csy != nil {
			lstmod := dirs.LatestMod(path, []string{".go"})
			if lstmod.Before(cts) {
				// fmt.Printf("loaded cache for: %v from: %v\n", path, cts)
				return csy
			}
		}
	}

	fs := &FileState{}
	// optional monitoring of parsing
	// fs.ParseState.Trace.On = true
	// fs.ParseState.Trace.Match = true
	// fs.ParseState.Trace.NoMatch = true
	// fs.ParseState.Trace.Run = true
	// fs.ParseState.Trace.StdOut()
	pr := gl.Parser()
	var pkgsym *syms.Symbol
	for i := range fls {
		fnm := fls[i]
		if strings.HasSuffix(fnm, "_test.go") {
			continue
		}
		fpath := filepath.Join(path, fnm)
		err = fs.OpenFile(fpath)
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
			if pkg.Name == "main" {
				continue
			}
			gl.DeleteUnexported(pkg.Children)
			if pkgsym == nil {
				pkgsym = pkg
			} else {
				pkgsym.Children.CopyFrom(pkg.Children)
			}
			// } else {
			// 	fmt.Printf("\tno parse state scopes!\n")
		}
	}
	if pkgsym != nil && !opts.Nocache {
		syms.SaveSymCache(pkgsym, path)
	}
	return pkgsym
}

/////////////////////////////////////////////////////////////////////////////
// Go util funcs

// DeleteUnexported deletes lower-case unexported items from map, and
// children of symbols on map
func (gl *GoLang) DeleteUnexported(sm syms.SymMap) {
	for nm, sy := range sm {
		if sy.Kind.SubCat() == token.NameScope { // typically lowercase
			continue
		}
		rn, _ := utf8.DecodeRuneInString(nm)
		if nm == "" || unicode.IsLower(rn) {
			delete(sm, nm)
		}
		if sy.HasChildren() {
			gl.DeleteUnexported(sy.Children)
		}
	}
}

// AddPkgToSyms adds given package symbol, with children from package
// to FileState.Syms list -- merges with anything already there
// does NOT add imports -- that is an optional second step.
// Returns true if there was an existing entry for this package.
func (gl *GoLang) AddPkgToSyms(fs *FileState, pkg *syms.Symbol) bool {
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

// AddImportsToSyms adds imports from given package into FileState.Syms list
// imports are coded as NameLibrary symbols with names = import path
func (gl *GoLang) AddImportsToSyms(fs *FileState, pkg *syms.Symbol) {
	fs.SymsMu.RLock()
	imps := pkg.Children.FindKindScoped(token.NameLibrary)
	fs.SymsMu.RUnlock()
	if len(imps) == 0 {
		return
	}
	for _, im := range imps {
		go gl.AddImportToSyms(fs, im.Name) // todo: should be "go"
	}
}

// AddImportToSyms adds given import into FileState.Syms list
// assumed to be called as a separate goroutine
func (gl *GoLang) AddImportToSyms(fs *FileState, im string) {
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
	psym := gl.ParseDir(im, LangDirOpts{})
	if psym != nil {
		psym.Name = pnm
		gl.AddPkgToSyms(fs, psym)
	}
}

// AddPathToSyms adds given path into FileState.Syms list
// assumed to be called as a separate goroutine
func (gl *GoLang) AddPathToSyms(fs *FileState, path string) {
	psym := gl.ParseDir(path, LangDirOpts{})
	if psym != nil {
		gl.AddPkgToSyms(fs, psym)
	}
}
