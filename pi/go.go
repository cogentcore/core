// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pi

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/goki/gi/filecat"
	"github.com/goki/ki/dirs"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/parse"
	"github.com/goki/pi/syms"
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
	fs := &FileState{}
	fs.Init()
	gl.Pr.InitAll(fs)
	return gl.Pr
}

func (gl *GoLang) ParseFile(fs *FileState) {
	pr := gl.Parser()
	if pr == nil {
		return
	}
	pr.LexAll(fs)
	pr.ParseAll(fs)
	// todo: manage the symbols!
}

func (gl *GoLang) LexLine(fs *FileState, line int) lex.Line {
	pr := gl.Parser()
	if pr == nil {
		return nil
	}
	// todo: could do some parsing here too!
	return pr.LexLine(fs, line)
}

func (gl *GoLang) ParseLine(fs *FileState, line int) *parse.Ast {
	// todo: writeme
	return nil
}

func (gl *GoLang) CompleteLine(fs *FileState, pos lex.Pos) syms.SymStack {
	// todo: writeme
	return nil
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
	fls := dirs.ExtFileNames(path, []string{".go"})
	if len(fls) == 0 {
		return nil
	}
	lp := StdLangProps[filecat.Go]
	fs := &FileState{}
	pr := lp.Parser
	pr.InitAll(fs)
	var pkgsym *syms.Symbol
	for i := range fls {
		fnm := fls[i]
		fpath := filepath.Join(path, fnm)
		err = fs.OpenFile(fpath)
		if err != nil {
			continue
		}
		fmt.Printf("parsing file: %v\n", fnm)
		stt := time.Now()
		pr.LexAll(fs)
		lxdur := time.Now().Sub(stt)
		pr.ParserInit(fs)
		pr.ParseRun(fs)
		prdur := time.Now().Sub(stt)
		fmt.Printf("\tlex: %v full parse: %v\n", lxdur, prdur-lxdur)
		if len(fs.ParseState.Scopes) > 0 { // should be
			pkg := fs.ParseState.Scopes[0]
			if pkgsym == nil {
				pkgsym = pkg
			} else {
				pkgsym.Children.CopyFrom(pkg.Children)
			}
		}
	}
	if !opts.Nocache {
		syms.SaveSymCache(pkgsym, path)
	}
	return pkgsym
}
