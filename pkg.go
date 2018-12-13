// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pi

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/goki/gi/filecat"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/syms"
)

// FilesInDir returns all the files with given extension(s) in directory (just the file names)
func FilesInDir(path string, exts []string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	files, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return nil
	}
	if len(exts) == 0 {
		return files
	}
	sz := len(files)
	if sz == 0 {
		return nil
	}
	for i := sz - 1; i >= 0; i-- {
		fn := files[i]
		ext := filepath.Ext(fn)
		keep := false
		for _, ex := range exts {
			if strings.EqualFold(ext, ex) {
				keep = true
				break
			}
		}
		if !keep {
			files = append(files[:i], files[i+1:]...)
		}
	}
	sort.StringSlice(files).Sort()
	return files
}

// ParseGoPackage parses all the go files in a given package path
// and optionally saves the symbols in the symbol cache, and also returns
// them.  Path can be an import kind of path or a full path.
func ParseGoPackage(path string, savecache bool) *syms.Symbol {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		path, err = kit.GoSrcDir(path)
		if err != nil {
			log.Println(err)
			return nil
		}
	} else if err != nil {
		log.Println(err.Error())
		return nil
	}
	path, _ = filepath.Abs(path)
	fls := FilesInDir(path, []string{".go"})
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
		// if pkgsym != nil && len(fs.ParseState.ExtScopes) == 0 {
		// 	fs.ParseState.ExtScopes.Add(pkgsym)
		// }
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
	if pkgsym != nil && savecache {
		syms.SaveSymCache(pkgsym, path)
		syms.SaveSymDoc(pkgsym, path)
	}
	return pkgsym
}
